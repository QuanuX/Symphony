#include "provider.hpp"
#include <algorithm>
#include <array>
#include <cerrno>
#include <cstdlib>
#include <cstring>
#include <dlfcn.h>
#include <fcntl.h>
#include <limits>
#include <mutex>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/stat.h>
#include <thread>
#include <unistd.h>
#include <utility>
#include <vector>
namespace symphony::sbv::detail {
namespace {
namespace db = sqav::databento;
struct FD {
  int n = -1;
  explicit FD(int fd = -1) : n(fd) {}
  ~FD() {
    if (n >= 0)
      ::close(n);
  }
  FD(const FD &) = delete;
  FD &operator=(const FD &) = delete;
  FD(FD &&o) noexcept : n(std::exchange(o.n, -1)) {}
  FD &operator=(FD &&o) noexcept {
    if (n >= 0)
      ::close(n);
    n = std::exchange(o.n, -1);
    return *this;
  }
};
// Streaming SHA-256 reused from the SQFV frame codec; fixed stack storage.
constexpr std::array<std::uint32_t, 64> kRoundConstants = {
    0x428a2f98U, 0x71374491U, 0xb5c0fbcfU, 0xe9b5dba5U, 0x3956c25bU,
    0x59f111f1U, 0x923f82a4U, 0xab1c5ed5U, 0xd807aa98U, 0x12835b01U,
    0x243185beU, 0x550c7dc3U, 0x72be5d74U, 0x80deb1feU, 0x9bdc06a7U,
    0xc19bf174U, 0xe49b69c1U, 0xefbe4786U, 0x0fc19dc6U, 0x240ca1ccU,
    0x2de92c6fU, 0x4a7484aaU, 0x5cb0a9dcU, 0x76f988daU, 0x983e5152U,
    0xa831c66dU, 0xb00327c8U, 0xbf597fc7U, 0xc6e00bf3U, 0xd5a79147U,
    0x06ca6351U, 0x14292967U, 0x27b70a85U, 0x2e1b2138U, 0x4d2c6dfcU,
    0x53380d13U, 0x650a7354U, 0x766a0abbU, 0x81c2c92eU, 0x92722c85U,
    0xa2bfe8a1U, 0xa81a664bU, 0xc24b8b70U, 0xc76c51a3U, 0xd192e819U,
    0xd6990624U, 0xf40e3585U, 0x106aa070U, 0x19a4c116U, 0x1e376c08U,
    0x2748774cU, 0x34b0bcb5U, 0x391c0cb3U, 0x4ed8aa4aU, 0x5b9cca4fU,
    0x682e6ff3U, 0x748f82eeU, 0x78a5636fU, 0x84c87814U, 0x8cc70208U,
    0x90befffaU, 0xa4506cebU, 0xbef9a3f7U, 0xc67178f2U,
};

constexpr std::uint32_t rotate_right(std::uint32_t value,
                                     unsigned count) noexcept {
  return (value >> count) | (value << (32U - count));
}

class Sha256 final {
public:
  bool update(const std::uint8_t *input, std::size_t size) noexcept {
    if (size != 0 && input == nullptr)
      return false;
    constexpr std::uint64_t kMaxBytes =
        std::numeric_limits<std::uint64_t>::max() / 8U;
    if (size > kMaxBytes - bytes_seen_)
      return false;
    bytes_seen_ += size;
    while (size != 0) {
      const std::size_t take = std::min(size, block_.size() - used_);
      std::memcpy(block_.data() + used_, input, take);
      used_ += take;
      input += take;
      size -= take;
      if (used_ == block_.size()) {
        transform();
        used_ = 0;
      }
    }
    return true;
  }

  void finish(std::uint8_t output[32]) noexcept {
    const std::uint64_t bit_count = bytes_seen_ * 8U;
    block_[used_++] = 0x80U;
    if (used_ > 56U) {
      std::fill(block_.begin() + used_, block_.end(), 0);
      transform();
      used_ = 0;
    }
    std::fill(block_.begin() + used_, block_.begin() + 56U, 0);
    for (std::size_t index = 0; index < 8U; ++index) {
      block_[56U + index] =
          static_cast<std::uint8_t>(bit_count >> ((7U - index) * 8U));
    }
    transform();
    for (std::size_t word = 0; word < state_.size(); ++word) {
      for (std::size_t byte = 0; byte < 4U; ++byte) {
        output[word * 4U + byte] =
            static_cast<std::uint8_t>(state_[word] >> ((3U - byte) * 8U));
      }
    }
  }

private:
  void transform() noexcept {
    std::array<std::uint32_t, 64> schedule{};
    for (std::size_t index = 0; index < 16U; ++index) {
      const std::size_t base = index * 4U;
      schedule[index] = (static_cast<std::uint32_t>(block_[base]) << 24U) |
                        (static_cast<std::uint32_t>(block_[base + 1U]) << 16U) |
                        (static_cast<std::uint32_t>(block_[base + 2U]) << 8U) |
                        static_cast<std::uint32_t>(block_[base + 3U]);
    }
    for (std::size_t index = 16U; index < schedule.size(); ++index) {
      const auto s0 = rotate_right(schedule[index - 15U], 7U) ^
                      rotate_right(schedule[index - 15U], 18U) ^
                      (schedule[index - 15U] >> 3U);
      const auto s1 = rotate_right(schedule[index - 2U], 17U) ^
                      rotate_right(schedule[index - 2U], 19U) ^
                      (schedule[index - 2U] >> 10U);
      schedule[index] = schedule[index - 16U] + s0 + schedule[index - 7U] + s1;
    }

    auto a = state_[0];
    auto b = state_[1];
    auto c = state_[2];
    auto d = state_[3];
    auto e = state_[4];
    auto f = state_[5];
    auto g = state_[6];
    auto h = state_[7];
    for (std::size_t index = 0; index < schedule.size(); ++index) {
      const auto s1 =
          rotate_right(e, 6U) ^ rotate_right(e, 11U) ^ rotate_right(e, 25U);
      const auto choose = (e & f) ^ ((~e) & g);
      const auto first =
          h + s1 + choose + kRoundConstants[index] + schedule[index];
      const auto s0 =
          rotate_right(a, 2U) ^ rotate_right(a, 13U) ^ rotate_right(a, 22U);
      const auto majority = (a & b) ^ (a & c) ^ (b & c);
      const auto second = s0 + majority;
      h = g;
      g = f;
      f = e;
      e = d + first;
      d = c;
      c = b;
      b = a;
      a = first + second;
    }
    state_[0] += a;
    state_[1] += b;
    state_[2] += c;
    state_[3] += d;
    state_[4] += e;
    state_[5] += f;
    state_[6] += g;
    state_[7] += h;
  }

  std::array<std::uint32_t, 8> state_{0x6a09e667U, 0xbb67ae85U, 0x3c6ef372U,
                                      0xa54ff53aU, 0x510e527fU, 0x9b05688cU,
                                      0x1f83d9abU, 0x5be0cd19U};
  std::array<std::uint8_t, 64> block_{};
  std::size_t used_ = 0;
  std::uint64_t bytes_seen_ = 0;
};
bool digest(const Json &v) {
  const auto s = str(v);
  return s.size() == 64 && std::all_of(s.begin(), s.end(), [](char c) {
           return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
         });
}
void label(const Json &v) { need(!str(v).empty(), "provider label required"); }
FD open_source(const std::string &path) {
  need(path.starts_with('/') && e::is_safe_relative_path(path.substr(1)),
       "provider requires absolute no-follow source path");
  FD dir(::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
  need(dir.n >= 0, "provider root open failed");
  std::size_t start = 1;
  while (true) {
    auto slash = path.find('/', start);
    const auto name =
        path.substr(start, slash == std::string::npos ? slash : slash - start);
    if (slash == std::string::npos) {
      FD file(::openat(dir.n, name.c_str(),
                       O_RDONLY | O_NONBLOCK | O_CLOEXEC | O_NOFOLLOW));
      struct stat st{};
      need(file.n >= 0 && ::fstat(file.n, &st) == 0 && S_ISREG(st.st_mode) &&
               st.st_size >= 0,
           "provider source must be a readable regular no-follow file");
      return file;
    }
    dir = FD(::openat(dir.n, name.c_str(),
                      O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
    need(dir.n >= 0, "provider source directory unavailable");
    start = slash + 1;
  }
}
struct Hashed {
  std::string sha256;
  std::uint64_t bytes;
};
Hashed transfer(FD &source, int destination, std::int64_t end) {
  Sha256 hash;
  std::array<std::uint8_t, 65536> buffer{};
  std::uint64_t total = 0;
  for (;;) {
    deadline(end);
    const auto count = ::read(source.n, buffer.data(), buffer.size());
    if (count < 0 && errno == EINTR)
      continue;
    need(count >= 0, "provider source read failed");
    if (count == 0)
      break;
    const auto size = static_cast<std::size_t>(count);
    need(size <= UINT64_MAX - total && hash.update(buffer.data(), size),
         "provider source exceeds digest length representation");
    total += size;
    for (std::size_t offset = 0; destination >= 0 && offset < size;) {
      deadline(end);
      const auto n =
          ::write(destination, buffer.data() + offset, size - offset);
      if (n < 0 && errno == EINTR)
        continue;
      need(n > 0, "provider private stage write failed");
      offset += static_cast<std::size_t>(n);
    }
  }
  std::array<std::uint8_t, 32> out{};
  hash.finish(out.data());
  constexpr char hex[] = "0123456789abcdef";
  std::string text;
  for (auto byte : out) {
    text += hex[byte >> 4];
    text += hex[byte & 15];
  }
  return {text, total};
}
struct Scope {
  std::int64_t end;
  const std::atomic_bool *cancelled;
  std::thread::id thread = std::this_thread::get_id();
  bool active = true;
  static std::uint32_t cancel(void *p) noexcept {
    if (!p)
      return SBV_CALL_CANCEL_REQUESTED;
    const auto &s = *static_cast<Scope *>(p);
    if (std::this_thread::get_id() != s.thread || !s.active)
      return SBV_CALL_CANCEL_REQUESTED;
    if (s.cancelled && s.cancelled->load())
      return SBV_CALL_CANCEL_REQUESTED;
    if (s.end != e::no_deadline && e::unix_time_ms() >= s.end)
      return SBV_CALL_DEADLINE_EXCEEDED;
    return SBV_CALL_RUNNING;
  }
  sbv_call_context_v1 context() {
    return {
        sizeof(sbv_call_context_v1),     1,    end == e::no_deadline ? 0U : 1U,
        end == e::no_deadline ? 0 : end, this, cancel};
  }
};
sbv_mbo_event_v1 event_value(const db::Mbo &m, std::uint64_t ordinal) {
  return {sizeof(sbv_mbo_event_v1),
          1,
          ordinal,
          m.publisher_id,
          m.instrument_id,
          m.ts_event,
          m.order_id,
          m.price,
          m.size,
          m.flags,
          m.channel_id,
          m.action,
          m.side,
          m.ts_recv,
          m.ts_in_delta,
          m.sequence,
          m.ts_out ? 1U : 0U,
          m.ts_out.value_or(0)};
}
struct Reader {
  Scope *scope;
  std::span<const db::Mbo> events;
  std::size_t first, end;
  static std::int32_t read(void *p, std::uint64_t ordinal,
                           sbv_mbo_event_v1 *out) noexcept {
    if (!p || !out)
      return SBV_PROVIDER_INVALID_INPUT;
    const auto &r = *static_cast<Reader *>(p);
    if (std::this_thread::get_id() != r.scope->thread || !r.scope->active ||
        out->struct_size < sizeof(*out) || out->abi_version != 1 ||
        ordinal < r.first || ordinal >= r.end || ordinal >= r.events.size())
      return SBV_PROVIDER_INVALID_INPUT;
    if (Scope::cancel(r.scope) != SBV_CALL_RUNNING)
      return SBV_PROVIDER_CANCELLED;
    *out = event_value(r.events[static_cast<std::size_t>(ordinal)], ordinal);
    return SBV_PROVIDER_OK;
  }
  sbv_event_reader_v1 view() {
    return {sizeof(sbv_event_reader_v1), 1, first, end, this, read};
  }
};
struct Buffer {
  sbv_owned_bytes_v1 value{
      sizeof(sbv_owned_bytes_v1), 1, nullptr, 0, nullptr, nullptr};
  ~Buffer() {
    if (value.release)
      value.release(value.owner, value.data, value.size);
  }
  std::string copy() const {
    const auto &v = value;
    need(v.struct_size == sizeof(v) && v.abi_version == 1,
         "provider buffer ABI mismatch");
    if (v.size == 0) {
      need(!v.data && !v.owner && !v.release,
           "provider empty buffer ownership mismatch");
      return {};
    }
    need(v.data && v.release && v.size <= artifact_bytes &&
             v.size <= std::numeric_limits<std::size_t>::max(),
         "provider candidate buffer violates existing result representation");
    return {reinterpret_cast<const char *>(v.data),
            static_cast<std::size_t>(v.size)};
  }
};
sbv_bytes_v1 bytes(const std::string &s) {
  return {sizeof(sbv_bytes_v1), 1,
          reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}
template <class F>
ProviderReply invoke(Scope &scope, F &&fn, std::string *raw = nullptr) {
  deadline(scope.end);
  if (scope.cancelled && scope.cancelled->load())
    return {SBV_PROVIDER_CANCELLED,
            nullptr,
            {{"code", "host_cancelled"},
             {"message", "caller requested cancellation"},
             {"details", Json::object()}}};
  Buffer output, error;
  const auto context = scope.context();
  std::int32_t status;
  try {
    status = fn(context, output.value, error.value);
  } catch (...) {
    scope.active = false;
    throw;
  }
  scope.active = false;
  const auto value = output.copy(), err = error.copy();
  need(status >= SBV_PROVIDER_OK && status <= SBV_PROVIDER_FAILED,
       "unknown provider status");
  ProviderReply reply{status, nullptr, nullptr};
  if (status == SBV_PROVIDER_OK) {
    need(err.empty(), "successful provider callback returned an error");
    if (!value.empty())
      reply.value =
          e::parse_bounded_json(value, artifact_bytes, artifact_values);
    if (raw)
      *raw = value;
  } else {
    need(value.empty() && !err.empty(),
         "failed provider callback output/error mismatch");
    reply.error = e::parse_bounded_json(err, artifact_bytes, artifact_values);
    keys(reply.error, {"code", "message", "details"});
    label(reply.error.at("code"));
    label(reply.error.at("message"));
    need(reply.error.at("details").is_object(),
         "provider error details must be an object");
  }
  deadline(scope.end);
  if (scope.cancelled && scope.cancelled->load() && status == SBV_PROVIDER_OK)
    return {SBV_PROVIDER_CANCELLED,
            nullptr,
            {{"code", "host_cancelled"},
             {"message", "caller requested cancellation"},
             {"details", Json::object()}}};
  return reply;
}
bool list(const Json &v, std::initializer_list<const char *> choices,
          const std::string &selected = {}) {
  need(v.is_array(), "provider descriptor list required");
  std::set<std::string> seen;
  for (const auto &x : v) {
    const auto value = str(x);
    bool known = false;
    for (const auto *choice : choices)
      known |= value == choice;
    need(known && seen.insert(value).second,
         "unknown or duplicate provider descriptor choice");
  }
  return selected.empty() || seen.contains(selected);
}
void portable_configuration(const Json &value) {
  std::vector<const Json *> pending{&value};
  while (!pending.empty()) {
    const auto &node = *pending.back();
    pending.pop_back();
    need(!node.is_number(),
         "provider parameters/extensions require exact numeric strings");
    if (node.is_structured())
      for (const auto &child : node)
        pending.push_back(&child);
  }
}
void admit_selection(const Json &p) {
  keys(p, {"protocol", "library", "id", "version", "role", "input_profile",
           "concurrency", "parameters", "dependencies", "extensions"});
  need(p.at("protocol") == "symphony.sbv.native-provider-selection.v1",
       "provider selection protocol mismatch");
  label(p.at("id"));
  label(p.at("version"));
  need(p.at("role") == "strategy" || p.at("role") == "model",
       "unknown provider role");
  need(p.at("input_profile") == "symphony.sbv.provider-databento-mbo-event.v1",
       "unsupported provider input profile");
  const auto mode = str(p.at("concurrency"));
  need(mode == "serialized_instance" || mode == "per_worker_instances" ||
           mode == "shared_reentrant_instance",
       "unknown provider concurrency");
  need(p.at("role") != "strategy" || mode == "serialized_instance",
       "strategy instance requires ordered serialized calls");
  need(p.at("parameters").is_object() && p.at("extensions").is_object(),
       "provider parameters/extensions must be objects");
  portable_configuration(p.at("parameters"));
  portable_configuration(p.at("extensions"));
  keys(p.at("library"), {"path", "expected_sha256"});
  const auto path = str(p.at("library").at("path"));
  need(path.starts_with('/') && e::is_safe_relative_path(path.substr(1)),
       "provider library requires absolute no-follow path");
  need(digest(p.at("library").at("expected_sha256")),
       "provider library digest required");
  const auto &dep = p.at("dependencies");
  keys(dep, {"capture", "description", "artifacts"});
  label(dep.at("description"));
  need((dep.at("capture") == "declared_artifacts" ||
        dep.at("capture") == "uncaptured") &&
           dep.at("artifacts").is_array(),
       "provider dependency disclosure mismatch");
  need(dep.at("capture") != "uncaptured" || dep.at("artifacts").empty(),
       "uncaptured dependencies cannot claim checked artifacts");
  std::set<std::string> paths;
  for (const auto &ref : dep.at("artifacts")) {
    keys(ref, {"path", "expected_sha256"});
    const auto ref_path = str(ref.at("path"));
    need(ref_path.starts_with('/') &&
             e::is_safe_relative_path(ref_path.substr(1)) &&
             digest(ref.at("expected_sha256")) && paths.insert(ref_path).second,
         "unique dependency absolute path and digest required");
  }
}
void admit_descriptor(const Json &d, const Json &p,
                      const sbv_provider_api_v1 &api) {
  keys(d,
       {"protocol", "id", "version", "roles", "input_profiles", "config_schema",
        "strategy_concurrency", "model_concurrency", "reproducibility",
        "cancellation", "dependency_disclosure", "extensions"});
  need(d.at("protocol") == "symphony.sbv.native-provider-descriptor.v1" &&
           d.at("id") == p.at("id") && d.at("version") == p.at("version"),
       "provider descriptor identity mismatch");
  need(list(d.at("roles"), {"strategy", "model"}, str(p.at("role"))) &&
           !d.at("roles").empty(),
       "provider role not declared");
  need(list(d.at("input_profiles"),
            {"symphony.sbv.provider-databento-mbo-event.v1"},
            str(p.at("input_profile"))),
       "provider input profile not declared");
  list(d.at("strategy_concurrency"), {"serialized_instance"});
  list(d.at("model_concurrency"),
       {"serialized_instance", "per_worker_instances",
        "shared_reentrant_instance"});
  std::uint64_t roles = 0;
  for (const auto &r : d.at("roles"))
    roles |= r == "strategy" ? SBV_PROVIDER_STRATEGY : SBV_PROVIDER_MODEL;
  need(roles == api.role_bits &&
           (bool(roles & SBV_PROVIDER_STRATEGY) ==
            !d.at("strategy_concurrency").empty()) &&
           (bool(roles & SBV_PROVIDER_MODEL) ==
            !d.at("model_concurrency").empty()),
       "provider role/table/concurrency mismatch");
  const auto &modes = d.at(p.at("role") == "strategy" ? "strategy_concurrency"
                                                      : "model_concurrency");
  need(std::find(modes.begin(), modes.end(), p.at("concurrency")) !=
           modes.end(),
       "selected concurrency not declared");
  need(d.at("config_schema").is_object() && d.at("extensions").is_object(),
       "provider descriptor schema/extensions required");
  need(d.at("reproducibility") == "deterministic_declared" ||
           d.at("reproducibility") == "nondeterministic" ||
           d.at("reproducibility") == "uncaptured",
       "provider reproducibility declaration required");
  need(d.at("cancellation") == "cooperative_polling" ||
           d.at("cancellation") == "between_callbacks_only",
       "provider cancellation declaration required");
  label(d.at("dependency_disclosure"));
}
Json evidence_value(const Json &p, const Json &descriptor,
                    const std::string &raw, std::uint64_t library_bytes,
                    const Json &dependencies) {
  return {
      {"protocol", "symphony.sbv.native-provider-evidence.v1"},
      {"abi_version", "1"},
      {"library",
       {{"path", p.at("library").at("path")},
        {"expected_sha256", p.at("library").at("expected_sha256")},
        {"bytes", dec(library_bytes)}}},
      {"selection_sha256", e::sha256_hex(p.dump())},
      {"descriptor_sha256", e::sha256_hex(raw)},
      {"descriptor_json", raw},
      {"id", p.at("id")},
      {"version", p.at("version")},
      {"role", p.at("role")},
      {"concurrency", p.at("concurrency")},
      {"reproducibility", descriptor.at("reproducibility")},
      {"cancellation", descriptor.at("cancellation")},
      {"dependencies", dependencies},
      {"configuration_validation", "provider_owned; config schema retained "
                                   "without generic host validation"},
      {"loading",
       {{"profile", "private_verified_copy"},
        {"symbol", "symphony_sbv_provider_api_v1"},
        {"visibility", "local"},
        {"dependency_resolution",
         "loader-native; dependencies are not staged or relocated; "
         "loader-relative paths resolve against private stage; listed "
         "dependency bytes checked separately without proving loaded identity"},
        {"trust", "caller-selected trusted native code; hashes establish byte "
                  "identity, not authentication"},
        {"isolation", false}}}};
}
} // namespace
void validate_native_provider_evidence(const Json &p, const Json &evidence) {
  admit_selection(p);
  keys(evidence, {"protocol", "abi_version", "library", "selection_sha256",
                  "descriptor_sha256", "descriptor_json", "id", "version",
                  "role", "concurrency", "reproducibility", "cancellation",
                  "dependencies", "configuration_validation", "loading"});
  const auto raw = str(evidence.at("descriptor_json"));
  const auto descriptor =
      e::parse_bounded_json(raw, artifact_bytes, artifact_values);
  // Retained evidence attests the admitted descriptor, not current loader
  // availability. The runtime table was checked when originally loaded.
  sbv_provider_api_v1 declared_api{};
  list(descriptor.at("roles"), {"strategy", "model"});
  for (const auto &role : descriptor.at("roles"))
    declared_api.role_bits |=
        role == "strategy" ? SBV_PROVIDER_STRATEGY : SBV_PROVIDER_MODEL;
  admit_descriptor(descriptor, p, declared_api);
  keys(evidence.at("library"), {"path", "expected_sha256", "bytes"});
  const auto library_bytes = u64(evidence.at("library").at("bytes"));
  need(library_bytes > 0, "retained provider library cannot be empty");
  const auto &captured = evidence.at("dependencies");
  keys(captured, {"capture", "description", "artifacts"});
  need(captured.at("artifacts").is_array() &&
           captured.at("artifacts").size() ==
               p.at("dependencies").at("artifacts").size(),
       "retained provider dependency count mismatch");
  auto dependencies = p.at("dependencies");
  for (std::size_t i = 0; i < captured.at("artifacts").size(); ++i) {
    const auto &ref = captured.at("artifacts").at(i);
    keys(ref, {"path", "expected_sha256", "bytes"});
    dependencies["artifacts"][i]["bytes"] = dec(u64(ref.at("bytes")));
  }
  need(evidence ==
           evidence_value(p, descriptor, raw, library_bytes, dependencies),
       "retained provider evidence differs from selected descriptor or "
       "dependencies");
}
struct NativeProviderState {
  Json selection, descriptor, evidence;
  std::string root, staged;
  void *library = nullptr;
  sbv_provider_api_v1 api{};
  ~NativeProviderState() {
    if (library)
      ::dlclose(library);
    if (!staged.empty())
      ::unlink(staged.c_str());
    if (!root.empty())
      ::rmdir(root.c_str());
  }
};
NativeProvider::NativeProvider(const Json &p, std::int64_t end)
    : state_(std::make_shared<NativeProviderState>()) {
  deadline(end);
  admit_selection(p);
  state_->selection = p;
  Json dependencies = p.at("dependencies");
  dependencies["artifacts"] = Json::array();
  std::set<std::string> paths;
  for (const auto &ref : p.at("dependencies").at("artifacts")) {
    keys(ref, {"path", "expected_sha256"});
    need(digest(ref.at("expected_sha256")) &&
             paths.insert(str(ref.at("path"))).second,
         "unique dependency path/digest required");
    auto file = open_source(str(ref.at("path")));
    auto h = transfer(file, -1, end);
    need(h.sha256 == str(ref.at("expected_sha256")),
         "provider dependency digest mismatch");
    dependencies["artifacts"].push_back({{"path", ref.at("path")},
                                         {"expected_sha256", h.sha256},
                                         {"bytes", dec(h.bytes)}});
  }
  auto file = open_source(str(p.at("library").at("path")));
#if defined(__APPLE__)
  char name[] = "/private/tmp/sbv-provider-XXXXXX";
#else
  char name[] = "/tmp/sbv-provider-XXXXXX";
#endif
  need(::mkdtemp(name) != nullptr, "provider private stage unavailable");
  state_->root = name;
  state_->staged = state_->root + "/provider.dylib";
  Hashed h;
  {
    FD destination(::open(state_->staged.c_str(),
                          O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC,
                          0600));
    need(destination.n >= 0, "provider private stage creation failed");
    h = transfer(file, destination.n, end);
    need(h.sha256 == str(p.at("library").at("expected_sha256")) && h.bytes > 0,
         "provider library digest mismatch");
    need(::fchmod(destination.n, 0500) == 0,
         "provider stage permissions failed");
  }
  auto check = open_source(state_->staged);
  auto checked = transfer(check, -1, end);
  need(checked.sha256 == h.sha256 && checked.bytes == h.bytes,
       "provider private stage readback mismatch");
  deadline(end);
  state_->library = ::dlopen(state_->staged.c_str(), RTLD_NOW | RTLD_LOCAL);
  if (!state_->library) {
    const auto *loader_error = ::dlerror();
    throw e::Error("sbv.contract",
                   "provider load failed; dependency paths are not relocated "
                   "into the private stage: " +
                       std::string(loader_error ? loader_error
                                                : "loader error unavailable"),
                   2);
  }
  const auto getter = reinterpret_cast<sbv_provider_get_api_v1>(
      ::dlsym(state_->library, "symphony_sbv_provider_api_v1"));
  need(getter != nullptr, "provider ABI entry point unavailable");
  auto &api = state_->api;
  api.struct_size = sizeof(api);
  api.abi_version = 1;
  need(getter(&api) == SBV_PROVIDER_OK && api.struct_size == sizeof(api) &&
           api.abi_version == 1 && api.required_capability_bits == 0 &&
           api.role_bits != 0 &&
           (api.role_bits & ~(SBV_PROVIDER_STRATEGY | SBV_PROVIDER_MODEL)) ==
               0 &&
           api.descriptor,
       "provider ABI/table admission failed");
  const bool strategy = api.role_bits & SBV_PROVIDER_STRATEGY,
             model = api.role_bits & SBV_PROVIDER_MODEL;
  need(strategy ? api.strategy_create && api.strategy_event &&
                      api.strategy_finish && api.strategy_destroy
                : !api.strategy_create && !api.strategy_event &&
                      !api.strategy_finish && !api.strategy_destroy,
       "provider strategy callback table mismatch");
  need(model ? api.model_create && api.model_evaluate && api.model_destroy
             : !api.model_create && !api.model_evaluate && !api.model_destroy,
       "provider model callback table mismatch");
  Scope scope{end, nullptr};
  std::string raw;
  const auto reply = invoke(
      scope,
      [&](const auto &ctx, auto &out, auto &err) {
        return api.descriptor(&ctx, &out, &err);
      },
      &raw);
  if (reply.status != SBV_PROVIDER_OK)
    throw e::Error("sbv.contract",
                   "provider descriptor unavailable: " + reply.error.dump(), 2);
  need(reply.value.is_object(), "provider descriptor must be an object");
  admit_descriptor(reply.value, p, api);
  state_->descriptor = reply.value;
  state_->evidence = evidence_value(p, reply.value, raw, h.bytes, dependencies);
}
const Json &NativeProvider::descriptor() const { return state_->descriptor; }
const Json &NativeProvider::evidence() const { return state_->evidence; }
struct NativeProviderInstance::State {
  std::shared_ptr<NativeProviderState> provider;
  void *instance = nullptr;
  std::mutex mutex;
  std::size_t next = 0;
  bool finished = false;
  ~State() {
    if (!instance)
      return;
    if (provider->selection.at("role") == "strategy")
      provider->api.strategy_destroy(instance);
    else
      provider->api.model_destroy(instance);
  }
  std::unique_lock<std::mutex> lock() {
    std::unique_lock l(mutex, std::defer_lock);
    if (provider->selection.at("concurrency") != "shared_reentrant_instance")
      l.lock();
    return l;
  }
};
NativeProviderInstance::NativeProviderInstance(std::unique_ptr<State> state)
    : state_(std::move(state)) {}
NativeProviderInstance::~NativeProviderInstance() = default;
std::unique_ptr<NativeProviderInstance>
NativeProvider::create(std::int64_t end,
                       const std::atomic_bool *cancelled) const {
  auto state = std::make_unique<NativeProviderInstance::State>();
  state->provider = state_;
  const auto config = state_->selection.at("parameters").dump();
  auto input = bytes(config);
  Scope scope{end, cancelled};
  auto reply = invoke(scope, [&](const auto &ctx, auto &, auto &err) {
    auto fn = state_->selection.at("role") == "strategy"
                  ? state_->api.strategy_create
                  : state_->api.model_create;
    return fn(&ctx, &input, &state->instance, &err);
  });
  if (reply.status != SBV_PROVIDER_OK)
    throw e::Error("sbv.contract",
                   "provider instance creation failed: " + reply.error.dump(),
                   2);
  need(state->instance != nullptr,
       "successful provider creation returned a null instance");
  return std::unique_ptr<NativeProviderInstance>(
      new NativeProviderInstance(std::move(state)));
}
ProviderReply NativeProviderInstance::event(std::span<const db::Mbo> events,
                                            std::size_t ordinal,
                                            std::int64_t end,
                                            const std::atomic_bool *cancelled) {
  auto lock = state_->lock();
  need(state_->provider->selection.at("role") == "strategy" &&
           !state_->finished && ordinal == state_->next &&
           ordinal < events.size(),
       "provider strategy event order/lifecycle mismatch");
  Scope scope{end, cancelled};
  Reader reader{&scope, events, 0, ordinal + 1};
  const auto current = event_value(events[ordinal], ordinal);
  const auto view = reader.view();
  const sbv_strategy_event_v1 input{sizeof(sbv_strategy_event_v1), 1, &current,
                                    &view};
  auto reply = invoke(scope, [&](const auto &ctx, auto &out, auto &err) {
    return state_->provider->api.strategy_event(state_->instance, &ctx, &input,
                                                &out, &err);
  });
  if (reply.status == SBV_PROVIDER_OK)
    ++state_->next;
  return reply;
}
ProviderReply
NativeProviderInstance::finish(std::int64_t end,
                               const std::atomic_bool *cancelled) {
  auto lock = state_->lock();
  need(state_->provider->selection.at("role") == "strategy" &&
           !state_->finished,
       "provider strategy finish lifecycle mismatch");
  Scope scope{end, cancelled};
  auto reply = invoke(scope, [&](const auto &ctx, auto &out, auto &err) {
    return state_->provider->api.strategy_finish(state_->instance, &ctx, &out,
                                                 &err);
  });
  state_->finished = true;
  return reply;
}
ProviderReply
NativeProviderInstance::evaluate(std::span<const db::Mbo> events,
                                 std::size_t ordinal, std::size_t followup_end,
                                 const Json &signal, const Json &coverage,
                                 std::uint64_t horizon, std::int64_t end,
                                 const std::atomic_bool *cancelled) {
  auto lock = state_->lock();
  need(state_->provider->selection.at("role") == "model" &&
           ordinal < events.size() && followup_end >= ordinal + 1 &&
           followup_end <= events.size(),
       "provider model source range mismatch");
  Scope scope{end, cancelled};
  Reader causal{&scope, events, 0, ordinal + 1},
      forward{&scope, events, ordinal + 1, followup_end};
  const auto c = causal.view(), f = forward.view();
  const auto signal_json = signal.dump(), coverage_json = coverage.dump();
  const sbv_model_frame_v1 input{
      sizeof(sbv_model_frame_v1), 1, bytes(signal_json), &c, &f, horizon,
      bytes(coverage_json)};
  return invoke(scope, [&](const auto &ctx, auto &out, auto &err) {
    return state_->provider->api.model_evaluate(state_->instance, &ctx, &input,
                                                &out, &err);
  });
}
Json provider_inspect(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "provider", "extensions"});
  need(p.at("protocol") == "symphony.sbv.provider-inspect-input.v1" &&
           p.at("extensions").is_object(),
       "provider inspection input mismatch");
  NativeProvider provider(p.at("provider"), end);
  return {{"protocol", "symphony.sbv.provider-inspect.v1"},
          {"engine_version", version},
          {"provider", provider.evidence()},
          {"extensions", p.at("extensions")}};
}
} // namespace symphony::sbv::detail
