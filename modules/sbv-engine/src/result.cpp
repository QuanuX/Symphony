#include "detail.hpp"
#include "interface.generated.hpp"
#include <cerrno>
#include <fcntl.h>
#include <filesystem>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/knowledge/engine/path.hpp>
#include <symphony/sbv/models.hpp>
#include <sys/stat.h>
#include <unistd.h>
#include <utility>
namespace symphony::sbv {
namespace d = detail;
namespace e = knowledge::engine;
namespace detail {
struct FD {
  int n;
  explicit FD(int fd) : n(fd) {}
  ~FD() {
    if (n >= 0)
      ::close(n);
  }
  FD(const FD &) = delete;
};
std::string relative(const std::string &path) {
  need(path.size() > 1 && path[0] == '/' && path.size() <= 4096,
       "absolute local path required");
  auto rel = path.substr(1);
  need(e::is_safe_relative_path(rel), "unsafe local path");
  return rel;
}
std::string read_file(const std::string &path, std::int64_t end) {
  deadline(end);
  return e::read_regular_file_no_follow("/", relative(path), artifact_bytes,
                                        end);
}
void require_new_file(const std::string &path) {
  relative(path);
  const auto parent = std::filesystem::path(path).parent_path();
  FD dir(::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC));
  need(dir.n >= 0, "output root unavailable");
  for (const auto &part : parent.relative_path()) {
    const auto next = ::openat(dir.n, part.c_str(),
                               O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    need(next >= 0, "output parent must exist without symlinks");
    ::close(dir.n);
    dir.n = next;
  }
  struct stat st{};
  const auto leaf = std::filesystem::path(path).filename().string();
  need(::fstatat(dir.n, leaf.c_str(), &st, AT_SYMLINK_NOFOLLOW) != 0 &&
           errno == ENOENT,
       "output must not exist before experiment execution");
}
void create_file(const std::string &path, const std::string &bytes,
                 std::int64_t end) {
  need(bytes.size() <= artifact_bytes, "artifact exceeds 128 MiB bound");
  relative(path);
  deadline(end);
  auto parent = std::filesystem::path(path).parent_path();
  FD dir(::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC));
  need(dir.n >= 0, "output root unavailable");
  for (const auto &part : parent.relative_path()) {
    const auto next = ::openat(dir.n, part.c_str(),
                               O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    need(next >= 0, "output parent must exist without symlinks");
    ::close(dir.n);
    dir.n = next;
  }
  const auto leaf = std::filesystem::path(path).filename().string();
  // Exclusive staging and linkat publication give no-replace semantics. Readers
  // see either no result or a fully written result, never a partial JSON file.
  const auto temp = ".sbv-" + dec(::getpid()) + "-" +
                    e::sha256_hex(leaf + "\n" + bytes).substr(0, 24) + ".tmp";
  FD file(::openat(dir.n, temp.c_str(),
                   O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600));
  need(file.n >= 0, "exclusive artifact staging failed");
  bool published = false;
  try {
    for (std::size_t pos = 0; pos < bytes.size();) {
      deadline(end);
      auto n = ::write(file.n, bytes.data() + pos,
                       std::min<std::size_t>(65536, bytes.size() - pos));
      if (n < 0 && errno == EINTR)
        continue;
      need(n > 0, "artifact write failed");
      pos += static_cast<std::size_t>(n);
    }
    need(::fsync(file.n) == 0, "artifact synchronization failed");
    deadline(end);
    need(::linkat(dir.n, temp.c_str(), dir.n, leaf.c_str(), 0) == 0,
         "artifact publication refused; destination may already exist");
    published = true;
    ::unlinkat(dir.n, temp.c_str(), 0);
    if (::fsync(dir.n) != 0)
      throw e::Error("sbv.outcome_uncertain",
                     "artifact published; verify destination before retry", 5);
  } catch (...) {
    ::unlinkat(dir.n, temp.c_str(), 0);
    if (published)
      throw e::Error("sbv.outcome_uncertain",
                     "artifact published; verify destination before retry", 5);
    throw;
  }
}
void validate_value(const Json &v, std::size_t depth, std::size_t &count) {
  need(++count <= artifact_values && depth <= 64, "artifact structure bound");
  need(!v.is_number(), "result numbers must be exact strings");
  if (v.is_string())
    need(v.get_ref<const std::string &>().size() <= 65536,
         "result string bound");
  if (v.is_object()) {
    count += v.size();
    need(count <= artifact_values, "artifact key count bound");
    for (auto it = v.begin(); it != v.end(); ++it)
      need(it.key().size() <= 65536, "artifact key size bound");
  }
  if (v.is_structured())
    for (const auto &c : v)
      validate_value(c, depth + 1, count);
}
Json persist(Json result, const Json &request, const std::string &op,
             std::int64_t end) {
  result = seal_result(std::move(result));
  validate_result(result);
  const auto bytes = result.dump();
  const auto path = str(request.at("output_path"));
  create_file(path, bytes, end);
  return {{"protocol", "symphony.sbv." + op + ".v1"},
          {"path", path},
          {"content_sha256", result.at("content_sha256")},
          {"file_sha256", e::sha256_hex(bytes)},
          {"bytes", dec(bytes.size())},
          {"status", result.at("status")},
          {"summary", result.at("sections").at("summary")}};
}
std::string token(const std::string &key) {
  std::string s;
  for (char c : key)
    s += c == '~' ? "~0" : c == '/' ? "~1" : std::string(1, c);
  return s;
}
} // namespace detail
Json seal_result(Json r) {
  r.erase("content_sha256");
  r["content_sha256"] = e::sha256_hex(r.dump());
  return r;
}
void validate_result(const Json &r) {
  d::keys(r, {"protocol", "origin", "status", "sections", "content_sha256"});
  d::need(r.at("protocol") == result_protocol && r.at("origin").is_string(),
          "unsupported result contract");
  d::need(r.at("status") == "completed" || r.at("status") == "partial",
          "invalid result status");
  const auto &s = r.at("sections");
  d::need(s.is_object(), "sections object required");
  for (auto name : {"summary", "signals", "execution", "distributions",
                    "studies", "replay", "comparisons", "search", "resources",
                    "diagnostics", "choices", "provenance"})
    d::need(s.contains(name), "required result section missing");
  for (const auto &section : s) {
    d::keys(section, {"status", "reason", "data"});
    const auto status = d::str(section.at("status"));
    d::need(status == "available" || status == "partial" ||
                status == "not_selected" || status == "unavailable",
            "invalid section status");
    d::need(
        section.at("reason").is_string() &&
            (status == "available" || !d::str(section.at("reason")).empty()),
        "section reason required");
  }
  std::size_t n = 0;
  d::validate_value(r, 0, n);
  auto body = r;
  body.erase("content_sha256");
  d::need(d::str(r.at("content_sha256")) == e::sha256_hex(body.dump()),
          "result digest mismatch");
}
Json descriptor() {
  auto ops = interface::interface_operations();
  e::validate_operation_specs(ops);
  Json r{{"protocol", e::descriptor_protocol_v2},
         {"format_version", 2},
         {"module_id", "sbv-engine"},
         {"engine_id", "symphony-sbv"},
         {"vector_id", "sbv"},
         {"engine_version", version},
         {"process_protocols", Json::array({e::process_protocol_v1})},
         {"contract_versions",
          Json::array({"modules/sbv-engine/SPEC.md@v1", result_protocol})},
         {"operations", e::administration_operation_descriptors(ops)},
         {"limits",
          {{"request_bytes", e::Limits::max_request_bytes},
           {"response_bytes", e::Limits::max_response_bytes},
           {"json_depth", e::Limits::max_json_depth},
           {"json_values", e::Limits::max_json_values},
           {"path_bytes", e::Limits::max_path_bytes},
           {"snapshot_files", 1},
           {"snapshot_file_bytes", d::artifact_bytes},
           {"deadline_ahead_ms", e::Limits::max_deadline_ahead_ms}}},
         {"supported_scopes", Json::array({"user"})},
         {"language", "C++26"},
         {"thermal_path", "freezing"},
         {"canonical_apply_enabled", false},
         {"session_mutation_enabled", false},
         {"network_listener", false}};
  for (const auto &op : ops) {
    r["contract_versions"].push_back(*op.input_protocol);
    r["contract_versions"].push_back(*op.output_protocol);
  }
  r["descriptor_digest"] = e::tagged_sha256(r.dump());
  return r;
}
Json dispatch(const std::string &op, const Json &p, std::int64_t end) {
  d::deadline(end);
  d::need(p.is_object() && p.contains("protocol"), "input protocol required");
  auto slug = op;
  std::replace(slug.begin(), slug.end(), '_', '-');
  d::need(d::str(p.at("protocol")) == "symphony.sbv." + slug + "-input.v1",
          "input protocol mismatch");
  if (op == "capabilities") {
    d::keys(p, {"protocol"});
    return {
        {"protocol", "symphony.sbv.capabilities.v1"},
        {"engine_version", version},
        {"sdk",
         {{"abi", "symphony.sbv.sdk.v1"},
          {"shared_library", true},
          {"bindings", Json::array({"c", "cpp"})},
          {"request_protocol", e::process_protocol_v1}}},
        {"cpu",
         {{"available", true},
          {"max_workers", "64"},
          {"deterministic_output_order", true}}},
        {"cuda",
         {{"available", false},
          {"contract", "explicit device, owner, stream, completion, dtype, "
                       "byte strides and shape; no automatic fallback"}}},
        {"tensor",
         {{"available", false},
          {"contract", "TensorView lifetime contract; no connected runtime"}}},
        {"live",
         {{"available", false},
          {"plan", "capture -> ordered availability stream -> causal signal "
                   "observer -> pending horizon -> sealed result; reconnect "
                   "and gaps require explicit policy"}}},
        {"studies",
         Json::array({"signal_summary", "forward_markout", "model_summary",
                      "path_excursion", "weighted_return_sum", "return_moments",
                      "return_quantiles", "liquidity_summary", "fill_quality",
                      "allocation_costs", "activation_moments",
                      "series_summary", "series_moments", "series_quantiles",
                      "equity_drawdown", "return_ratios",
                      "bootstrap_mean_distribution",
                      "bootstrap_mean_quantiles"})},
        {"execution_models",
         Json::array({"none", "touch_observation", "user_probability",
                      "observed_trade_levels", "external_outcomes",
                      "displayed_depth_sweep"})},
        {"economic_transforms",
         Json::array({"linear_price_pnl", "filled_quantity_markout"})},
        {"limits",
         {{"max_source_events", "200000"},
          {"max_signals", "4096"},
          {"max_artifact_bytes", d::dec(d::artifact_bytes)},
          {"max_composed_paths", "65536"}}}};
  }
  if (op == "run")
    return d::run(p, end);
  if (op == "compose")
    return d::compose(p, end);
  if (op == "compose_joint")
    return d::compose_joint(p, end);
  if (op == "split")
    return d::split(p, end);
  if (op == "resample")
    return d::resample(p, end);
  if (op == "experiment")
    return d::experiment(p, end);
  if (op == "analyze")
    return d::analyze(p, end);
  if (op == "compare")
    return d::compare(p, end);
  if (op == "result_select")
    return d::result_select(p, end);
  if (op == "backend_plan")
    return d::backend_plan(p, end);
  if (op == "live_plan")
    return d::live_plan(p, end);
  if (op == "allocation_economics")
    return d::allocation_economics(p, end);
  if (op == "liquidity")
    return d::liquidity(p, end);
  if (op == "book")
    return d::book(p, end);
  if (op == "economics")
    return d::economics(p, end);
  if (op == "evaluate")
    return d::evaluate(p, end);
  if (op == "catalogue") {
    d::keys(p, {"protocol"});
    return model_catalogue();
  }
  d::need(op == "result_inspect" || op == "result_query",
          "unsupported SBV operation");
  if (op == "result_inspect")
    d::keys(p, {"protocol", "path", "expected_sha256"});
  else
    d::keys(p, {"protocol", "path", "expected_sha256", "pointer", "limit",
                "cursor"});
  const auto bytes = d::read_file(d::str(p.at("path")), end);
  auto r = e::parse_bounded_json(bytes, d::artifact_bytes, d::artifact_values);
  validate_result(r);
  d::deadline(end);
  d::need((op == "result_inspect" && p.at("expected_sha256") == "") ||
              r.at("content_sha256") == p.at("expected_sha256"),
          "snapshot mismatch");
  if (op == "result_inspect") {
    Json sections = Json::array();
    for (auto it = r.at("sections").begin(); it != r.at("sections").end(); ++it)
      sections.push_back({{"name", it.key()},
                          {"pointer", "/sections/" + d::token(it.key())},
                          {"status", it.value().at("status")}});
    return {{"protocol", "symphony.sbv.result-inspect.v1"},
            {"path", p.at("path")},
            {"content_sha256", r.at("content_sha256")},
            {"file_sha256", e::sha256_hex(bytes)},
            {"bytes", d::dec(bytes.size())},
            {"sections", sections},
            {"status", r.at("status")},
            {"expected_digest_verified", p.at("expected_sha256") != ""}};
  }
  const auto pointer = d::str(p.at("pointer"));
  const auto limit = d::u64(p.at("limit"));
  d::need(limit >= 1 && limit <= 256, "page limit must be 1..256");
  const Json *selected = nullptr;
  try {
    selected = &r.at(Json::json_pointer(pointer));
  } catch (...) {
    throw e::Error("sbv.pointer", "result pointer not found", 2);
  }
  const auto qhash = e::sha256_hex(Json{
      {"snapshot", r.at("content_sha256")},
      {"pointer", pointer},
      {"limit", p.at("limit")}}.dump());
  const auto cursor = d::str(p.at("cursor"));
  std::uint64_t offset = 0;
  if (!cursor.empty()) {
    d::need(cursor.starts_with(qhash + ":"), "cursor snapshot/query mismatch");
    offset = d::u64(cursor.substr(qhash.size() + 1));
  }
  const auto total = selected->is_structured() ? selected->size() : 1;
  d::need(offset <= total, "cursor offset outside result");
  Json rows = Json::array();
  std::size_t index = 0, used = 0;
  bool full = false;
  auto add = [&](const Json &v, const std::string &path) {
    if (index++ < offset || rows.size() >= limit || full)
      return;
    Json row{{"pointer", path},
             {"type", v.type_name()},
             {"value", v.is_object()  ? Json::object()
                       : v.is_array() ? Json::array()
                                      : v},
             {"children", d::dec(v.is_structured() ? v.size() : 0)}};
    const auto size = row.dump().size();
    if (used + size > 512000) {
      full = true;
      return;
    }
    used += size;
    rows.push_back(std::move(row));
  };
  if (selected->is_object())
    for (auto it = selected->begin(); it != selected->end(); ++it)
      add(it.value(), pointer + "/" + d::token(it.key()));
  else if (selected->is_array())
    for (std::size_t i = 0; i < selected->size(); ++i)
      add((*selected)[i], pointer + "/" + d::dec(i));
  else
    add(*selected, pointer);
  const auto next = offset + rows.size();
  d::need(next > offset || offset == total, "page cannot progress");
  return {{"protocol", "symphony.sbv.result-query.v1"},
          {"content_sha256", r.at("content_sha256")},
          {"pointer", pointer},
          {"query_sha256", qhash},
          {"offset", d::dec(offset)},
          {"total", d::dec(total)},
          {"complete", next == total},
          {"next_cursor", next == total ? "" : qhash + ":" + d::dec(next)},
          {"nodes", rows}};
}
} // namespace symphony::sbv
