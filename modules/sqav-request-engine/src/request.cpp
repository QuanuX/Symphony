#include "request.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <symphony/sqav/databento/historical.hpp>
#include <symphony/sqav/databento/reference.hpp>
#include <symphony/sqav/fred.hpp>
#include <charconv>
#include <limits>
#include <set>

namespace symphony::sqav::request {
namespace {
[[noreturn]] void invalid() {
  throw engine::Error("sqav.request.invalid", "invalid bounded SQAV request", 2);
}
void fields(const Json &j, std::initializer_list<const char *> names) {
  if (!j.is_object() || j.size() != names.size()) invalid();
  for (const auto *name : names) if (!j.contains(name)) invalid();
}
std::string text(const Json &j, const char *key, std::size_t max = 128) {
  const auto &v = j.at(key);
  if (!v.is_string() || v.get_ref<const std::string &>().size() > max) invalid();
  return v.get<std::string>();
}
template<class T> T number(const Json &j, const char *key) {
  const auto &v = j.at(key);
  if (!v.is_number_unsigned() && !(v.is_number_integer() && v.get<std::int64_t>() >= 0)) invalid();
  const auto n = v.get<std::uint64_t>();
  if (n > std::numeric_limits<T>::max()) invalid();
  return static_cast<T>(n);
}
std::uint64_t nanoseconds(const Json &j, const char *key) {
  const auto s = text(j, key, 20);
  if (s.empty() || (s.size() > 1 && s.front() == '0')) invalid();
  for (const auto c : s) if (c < '0' || c > '9') invalid();
  std::uint64_t out = 0;
  const auto [end, error] = std::from_chars(s.data(), s.data()+s.size(), out);
  if (error != std::errc{} || end != s.data()+s.size()) invalid();
  return out;
}
std::vector<std::string> symbols(const Json &j) {
  const auto &v = j.at("symbols");
  if (!v.is_array() || v.size() > 128) invalid();
  std::vector<std::string> out;
  for (const auto &item : v) {
    if (!item.is_string() || item.get_ref<const std::string &>().size() > 128) invalid();
    out.push_back(item.get<std::string>());
  }
  return out;
}
template<class Status> void accepted(Status s) {
  if (s == Status::ok) return;
  if (s == Status::no_memory || s == Status::internal_error)
    throw engine::Error("sqav.request.unavailable", "native validation unavailable", 5);
  if (s == Status::limit)
    throw engine::Error("sqav.request.limit", "native request limits refused", 2);
  throw engine::Error("sqav.request.rejected", "native request validation refused", 2);
}
Json seal(Json j, const char *key) {
  j[key] = engine::tagged_sha256(j.dump());
  return j;
}
}
Json validate(const Json &p) {
  if (p.dump().size() > payload_bytes) invalid();
  fields(p, {"protocol", "adapter", "selection", "limits"});
  if (p.at("protocol") != input_protocol) invalid();
  const auto adapter = text(p, "adapter");
  const auto &s = p.at("selection");
  const auto &l = p.at("limits");
  std::string module, release, reference;
  if (adapter == "fred") {
    fields(s, {"operation", "series", "observation_start", "observation_end", "realtime_start", "realtime_end", "limit", "offset"});
    if (!l.is_null()) invalid();
    const auto operation = text(s, "operation");
    fred::Selection selected;
    if (operation == "observations") selected.operation = fred::Operation::observations;
    else if (operation == "vintage_dates") selected.operation = fred::Operation::vintage_dates;
    else invalid();
    selected.series = text(s, "series");
    selected.observation_start = text(s, "observation_start", 10);
    selected.observation_end = text(s, "observation_end", 10);
    selected.realtime_start = text(s, "realtime_start", 10);
    selected.realtime_end = text(s, "realtime_end", 10);
    selected.limit = number<std::uint32_t>(s, "limit");
    selected.offset = number<std::uint32_t>(s, "offset");
    fred::Plan plan;
    accepted(fred::Plan::create(selected, plan));
    reference = plan.reference(); module = "sqav-fred-cpp"; release = "0.1.0-dev";
  } else if (adapter == "databento_historical") {
    fields(s, {"dataset", "symbols", "start_ns", "end_ns", "record_limit"});
    fields(l, {"max_file_bytes", "max_metadata_bytes", "max_records", "max_window_ns", "max_symbols", "max_attempts", "max_retry_after_seconds"});
    databento::HistoricalSelection selected;
    selected.dataset = text(s, "dataset", 15); selected.symbols = symbols(s);
    selected.start = nanoseconds(s, "start_ns"); selected.end = nanoseconds(s, "end_ns");
    selected.record_limit = number<std::uint64_t>(s, "record_limit");
    databento::HistoricalLimits limits;
    limits.dbn = {number<std::uint64_t>(l,"max_file_bytes"), number<std::uint32_t>(l,"max_metadata_bytes"), number<std::uint64_t>(l,"max_records")};
    limits.max_window_ns = number<std::uint64_t>(l, "max_window_ns");
    limits.max_symbols = number<std::uint16_t>(l, "max_symbols");
    limits.max_attempts = number<std::uint8_t>(l, "max_attempts");
    limits.max_retry_after_seconds = number<std::uint32_t>(l, "max_retry_after_seconds");
    databento::HistoricalPlan plan;
    accepted(databento::HistoricalPlan::create(selected, limits, plan));
    reference = plan.reference(); module = "sqav-databento-dbn-cpp"; release = "0.5.0-dev";
  } else if (adapter == "databento_reference") {
    fields(s, {"operation", "symbols", "start", "end"});
    if (!l.is_null()) invalid();
    const auto operation = text(s, "operation");
    databento::reference::Selection selected;
    using Op = databento::reference::Operation;
    if (operation == "corporate_actions") selected.operation = Op::corporate_actions;
    else if (operation == "adjustment_factors") selected.operation = Op::adjustment_factors;
    else if (operation == "security_master_range") selected.operation = Op::security_master_range;
    else if (operation == "security_master_last") selected.operation = Op::security_master_last;
    else invalid();
    selected.symbols = symbols(s); selected.start = text(s, "start", 10); selected.end = text(s, "end", 10);
    databento::reference::Plan plan;
    accepted(databento::reference::Plan::create(selected, plan));
    reference = plan.reference(); module = "sqav-databento-reference-cpp"; release = "0.1.0-dev";
  } else invalid();
  return seal(Json{{"protocol", result_protocol}, {"adapter", adapter}, {"adapter_id", module},
                   {"adapter_version", release}, {"request_digest", engine::tagged_sha256(p.dump())},
                   {"plan_reference", reference}, {"validation_scope", "native_request_only"},
                   {"provider_observation", "not_performed"}}, "result_digest");
}
Json descriptor() {
  const auto ops = interface_operations();
  engine::validate_operation_specs(ops);
  return seal(Json{{"protocol", engine::descriptor_protocol_v2}, {"format_version", 2},
    {"module_id", "sqav-request-engine"}, {"engine_id", engine_id}, {"vector_id", "sqav"}, {"engine_version", version},
    {"process_protocols", Json::array({engine::process_protocol_v1})},
    {"contract_versions", Json::array({"modules/sqav-request-engine/SPEC.md@v1", input_protocol, result_protocol})},
    {"operations", engine::administration_operation_descriptors(ops)},
    {"limits", {{"request_bytes", process_bytes}, {"response_bytes", process_bytes}, {"json_depth", engine::Limits::max_json_depth},
      {"json_values", engine::Limits::max_json_values}, {"path_bytes", engine::Limits::max_path_bytes},
      {"snapshot_files", 1}, {"snapshot_file_bytes", 1}, {"deadline_ahead_ms", 5000}}},
    {"supported_scopes", Json::array({"user"})}, {"language", "C++26"}, {"thermal_path", "freezing"},
    {"canonical_apply_enabled", false}, {"session_mutation_enabled", false}, {"network_listener", false}}, "descriptor_digest");
}
Json handle_request(const engine::Request &r) {
  if (r.operation != "request_validate")
    throw engine::Error("operation.unsupported", "unsupported SQAV operation", 3);
  if (r.deadline_unix_ms > engine::unix_time_ms() + 5000)
    throw engine::Error("request.invalid_deadline", "SQAV deadline exceeds five seconds", 2);
  auto result = validate(r.payload);
  if (engine::unix_time_ms() >= r.deadline_unix_ms)
    throw engine::Error("request.deadline_expired", "SQAV deadline expired", 3);
  return result;
}
}
