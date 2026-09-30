#include <algorithm>
#include <array>
#include <charconv>
#include <chrono>
#include <limits>
#include <map>
#include <set>
#include <sstream>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/snv/local_observer.hpp>
#ifdef __linux__
#include <cerrno>
#include <fcntl.h>
#include <linux/magic.h>
#include <sys/stat.h>
#include <sys/statfs.h>
#include <sys/utsname.h>
#include <unistd.h>
#endif

namespace symphony::snv::local_observer {
namespace engine = symphony::knowledge::engine;
namespace {
[[noreturn]] void fail(const char *message) {
  throw engine::Error("invalid_input", message, 2);
}
void require(bool b, const char *message) {
  if (!b)
    fail(message);
}
void deadline(std::int64_t d) {
  if (engine::unix_time_ms() >= d)
    throw engine::Error("deadline_exceeded", "observer deadline exceeded", 3);
}
std::string text(const Json &j, std::size_t bound = 256) {
  require(j.is_string(), "string required");
  const auto s = j.get<std::string>();
  require(!s.empty() && s.size() <= bound, "string byte bound exceeded");
  for (const unsigned char c : s)
    require(c >= 32 && c != 127, "control character rejected");
  // Direct SDK calls have the same strict UTF8 admission as finite requests.
  try {
    (void)j.dump();
  } catch (const nlohmann::json::exception &) {
    fail("invalid UTF8 string");
  }
  return s;
}
std::string_view trim(std::string_view s) {
  while (!s.empty() && (s.front() == ' ' || s.front() == '\t' ||
                        s.front() == '\n' || s.front() == '\r'))
    s.remove_prefix(1);
  while (!s.empty() && (s.back() == ' ' || s.back() == '\t' ||
                        s.back() == '\n' || s.back() == '\r'))
    s.remove_suffix(1);
  return s;
}
bool unsigned_decimal(std::string_view s, std::uint64_t &value) {
  if (s.empty() || (s.size() > 1 && s.front() == '0'))
    return false;
  const auto [end, ec] = std::from_chars(s.data(), s.data() + s.size(), value);
  return ec == std::errc{} && end == s.data() + s.size();
}
std::string status(ReadStatus s) {
  switch (s) {
  case ReadStatus::ok:
    return "observed";
  case ReadStatus::unavailable:
    return "unavailable";
  case ReadStatus::permission_denied:
    return "permission_denied";
  case ReadStatus::unsupported_source:
    return "unsupported_source";
  case ReadStatus::too_large:
    return "too_large";
  case ReadStatus::io_error:
    return "io_error";
  }
  return "io_error";
}
bool boot_valid(std::string_view s) {
  s = trim(s);
  if (s.size() != 36)
    return false;
  for (std::size_t i = 0; i < s.size(); ++i) {
    if (i == 8 || i == 13 || i == 18 || i == 23) {
      if (s[i] != '-')
        return false;
    } else if (!((s[i] >= '0' && s[i] <= '9') || (s[i] >= 'a' && s[i] <= 'f')))
      return false;
  }
  return true;
}
Json cpu_value(std::string_view raw) {
  const auto s = trim(raw);
  if (s.empty())
    return {{"logical_cpu_indices", ""}, {"value", "0"}, {"unit", "count"}};
  std::uint64_t count = 0, previous = 0;
  bool first = true;
  std::size_t cursor = 0, segments = 0;
  while (cursor < s.size()) {
    if (++segments > 2048)
      return nullptr;
    const auto comma = s.find(',', cursor);
    const auto item =
        s.substr(cursor, comma == s.npos ? s.size() - cursor : comma - cursor);
    const auto hyphen = item.find('-');
    std::uint64_t lo = 0, hi = 0;
    if (hyphen == item.npos) {
      if (!unsigned_decimal(item, lo))
        return nullptr;
      hi = lo;
    } else if (!unsigned_decimal(item.substr(0, hyphen), lo) ||
               !unsigned_decimal(item.substr(hyphen + 1), hi) || hi <= lo)
      return nullptr;
    if (hi > 1048575 || (!first && lo <= previous))
      return nullptr;
    count += hi - lo + 1;
    previous = hi;
    first = false;
    if (comma == s.npos)
      break;
    cursor = comma + 1;
    if (cursor == s.size())
      return nullptr;
  }
  return {{"logical_cpu_indices", std::string(s)},
          {"value", std::to_string(count)},
          {"unit", "count"}};
}
std::map<std::string, Json> memory_values(std::string_view raw) {
  std::map<std::string, Json> result;
  std::set<std::string> seen;
  std::istringstream lines{std::string(raw)};
  std::string line;
  while (std::getline(lines, line)) {
    const auto colon = line.find(':');
    if (colon == line.npos)
      continue;
    const auto label = line.substr(0, colon);
    if (label != "MemTotal" && label != "MemAvailable")
      continue;
    if (!seen.insert(label).second) {
      result[label] = nullptr;
      continue;
    }
    auto tail = trim(std::string_view(line).substr(colon + 1));
    const auto split = tail.find_first_of(" \t");
    std::uint64_t kb = 0;
    if (split == tail.npos || !unsigned_decimal(tail.substr(0, split), kb) ||
        trim(tail.substr(split)) != "kB" ||
        kb > std::numeric_limits<std::uint64_t>::max() / 1024)
      result[label] = nullptr;
    else
      result[label] = {{"value", std::to_string(kb * 1024)}, {"unit", "bytes"}};
  }
  return result;
}

class NativeReader final : public SourceReader {
public:
  Platform platform() const override {
#ifdef __linux__
    struct utsname u{};
    if (::uname(&u) != 0)
      return {false, "", ""};
    return {true, u.machine, u.release};
#else
    return {false, "", ""};
#endif
  }
  ReadResult read(Source source, std::size_t limit, std::int64_t d) override {
    deadline(d);
#ifdef __linux__
    const bool proc = source == Source::boot || source == Source::meminfo;
    const char *path = source == Source::boot ? "proc/sys/kernel/random/boot_id"
                       : source == Source::meminfo ? "proc/meminfo"
                       : source == Source::cpu_present
                           ? "sys/devices/system/cpu/present"
                           : "sys/devices/system/cpu/online";
    struct OwnedFd {
      int value;
      ~OwnedFd() {
        if (value >= 0)
          ::close(value);
      }
      void replace(int next) {
        if (value >= 0)
          ::close(value);
        value = next;
      }
    } current{::open("/", O_RDONLY | O_CLOEXEC | O_DIRECTORY | O_NOFOLLOW)};
    if (current.value < 0)
      return {ReadStatus::io_error, {}};
    auto error_status = [](int e) {
      return e == EACCES || e == EPERM    ? ReadStatus::permission_denied
             : e == ENOENT                ? ReadStatus::unavailable
             : e == ELOOP || e == ENOTDIR ? ReadStatus::unsupported_source
                                          : ReadStatus::io_error;
    };
    std::string remaining(path);
    while (true) {
      deadline(d);
      const auto slash = remaining.find('/');
      const auto name = remaining.substr(0, slash);
      const bool last = slash == remaining.npos;
      const int flags = O_RDONLY | O_CLOEXEC | O_NOFOLLOW | O_NONBLOCK |
                        (last ? 0 : O_DIRECTORY);
      const int next = ::openat(current.value, name.c_str(), flags);
      const int saved = errno;
      if (next < 0)
        return {error_status(saved), {}};
      current.replace(next);
      if (last)
        break;
      remaining.erase(0, slash + 1);
    }
    struct stat info{};
    struct statfs fs{};
    if (::fstat(current.value, &info) != 0 ||
        ::fstatfs(current.value, &fs) != 0)
      return {ReadStatus::io_error, {}};
    if (!S_ISREG(info.st_mode) ||
        fs.f_type != (proc ? PROC_SUPER_MAGIC : SYSFS_MAGIC))
      return {ReadStatus::unsupported_source, {}};
    std::string bytes;
    std::array<char, 4096> buffer{};
    while (true) {
      deadline(d);
      const auto n = ::read(current.value, buffer.data(),
                            std::min(buffer.size(), limit + 1 - bytes.size()));
      if (n < 0) {
        const int saved = errno;
        if (saved == EINTR)
          continue;
        return {error_status(saved), {}};
      }
      if (n == 0)
        break;
      bytes.append(buffer.data(), static_cast<std::size_t>(n));
      if (bytes.size() > limit)
        return {ReadStatus::too_large, {}};
    }
    deadline(d);
    return {ReadStatus::ok, std::move(bytes)};
#else
    (void)source;
    (void)limit;
    return {ReadStatus::unavailable, {}};
#endif
  }
};
} // namespace

std::vector<engine::OperationSpec> operations() {
  return {{"engop:symphony:snv-local-observer.observe",
           "observe",
           "implemented",
           false,
           true,
           {"ssfv:symphony:snv-local-observer"},
           {"inspect"},
           "qxctl_required",
           "symphony.snv.local-observe-input.v1",
           "symphony.snv.local-observe.v1",
           "read_only",
           "non_idempotent",
           false,
           "none",
           "",
           "supported",
           "freezing"}};
}
Json descriptor() {
  const auto ops = operations();
  engine::validate_operation_specs(ops);
  Json j{{"protocol", engine::descriptor_protocol_v2},
         {"format_version", 2},
         {"module_id", "snv-local-observer"},
         {"engine_id", engine_id},
         {"vector_id", nullptr},
         {"engine_version", version},
         {"process_protocols", Json::array({engine::process_protocol_v1})},
         {"contract_versions",
          Json::array({"modules/snv-local-observer/SPEC.md@v1",
                       "symphony.snv.local-observe-input.v1",
                       "symphony.snv.local-observe.v1"})},
         {"operations", engine::administration_operation_descriptors(ops)},
         {"embedded_dependencies", Json::array()},
         {"limits",
          {{"request_bytes", engine::Limits::max_request_bytes},
           {"response_bytes", engine::Limits::max_response_bytes},
           {"json_depth", engine::Limits::max_json_depth},
           {"json_values", max_json_values},
           {"records", 4},
           {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms}}},
         {"supported_scopes", Json::array({"user"})},
         {"language", "C++26"},
         {"thermal_path", "freezing"},
         {"canonical_apply_enabled", false},
         {"session_mutation_enabled", false},
         {"network_listener", false}};
  j["descriptor_digest"] = engine::tagged_sha256(j.dump());
  return j;
}
namespace {
Json collect_with_route(const Json &p, SourceReader &reader, std::int64_t d,
                        std::string_view acquisition_route) {
  deadline(d);
  require(p.is_object() && p.size() == 5 && p.contains("protocol") &&
              p.contains("observation_id") && p.contains("node_ref") &&
              p.contains("profile") && p.contains("fields"),
          "exact observer request fields required");
  require(p.at("protocol") == "symphony.snv.local-observe-input.v1" &&
              p.at("profile") == "linux-proc-sysfs-v1",
          "unsupported observer protocol/profile");
  (void)text(p.at("observation_id"));
  if (!p.at("node_ref").is_null())
    (void)text(p.at("node_ref"));
  require(p.at("fields").is_array() && !p.at("fields").empty() &&
              p.at("fields").size() <= 4,
          "one to four selected fields required");
  std::set<std::string> selected;
  for (const auto &field : p.at("fields")) {
    const auto f = text(field);
    require(f == "cpu_present" || f == "cpu_online" || f == "memory_total" ||
                f == "memory_available",
            "unsupported observer field");
    require(selected.insert(f).second, "duplicate observer field");
  }
  const auto started = engine::unix_time_ms();
  const auto steady_start = std::chrono::steady_clock::now();
  const auto effective = std::min(d, started + 5000);
  auto guard = [&] {
    deadline(effective);
    if (std::chrono::steady_clock::now() - steady_start >=
        std::chrono::seconds(5))
      throw engine::Error("deadline_exceeded", "observer interval exceeded", 3);
  };
  const auto platform = reader.platform();
  guard();
  if (platform.available) {
    (void)text(Json(platform.architecture));
    (void)text(Json(platform.kernel_release));
  }
  auto read = [&](Source source, std::size_t limit) {
    guard();
    auto r = platform.available ? reader.read(source, limit, effective)
                                : ReadResult{ReadStatus::unavailable, {}};
    guard();
    if (r.bytes.size() > limit)
      return ReadResult{ReadStatus::too_large, {}};
    return r;
  };
  const auto before = read(Source::boot, 64);
  std::map<std::string, ReadResult> first, last;
  for (const auto &field : selected)
    if (field == "cpu_present" || field == "cpu_online")
      first[field] = read(field == "cpu_present" ? Source::cpu_present
                                                 : Source::cpu_online,
                          4096);
  ReadResult mem{ReadStatus::unavailable, {}};
  if (selected.contains("memory_total") ||
      selected.contains("memory_available"))
    mem = read(Source::meminfo, 16384);
  for (const auto &field : selected)
    if (field == "cpu_present" || field == "cpu_online")
      last[field] = read(field == "cpu_present" ? Source::cpu_present
                                                : Source::cpu_online,
                         4096);
  const auto after = read(Source::boot, 64);
  const auto boot_before =
      before.status == ReadStatus::ok && !boot_valid(before.bytes)
          ? "malformed"
          : status(before.status);
  const auto boot_after =
      after.status == ReadStatus::ok && !boot_valid(after.bytes)
          ? "malformed"
          : status(after.status);
  const bool boot_ok = before.status == ReadStatus::ok &&
                       after.status == ReadStatus::ok &&
                       boot_valid(before.bytes) && boot_valid(after.bytes);
  const auto boot_state = !boot_ok ? "unavailable"
                          : trim(before.bytes) == trim(after.bytes) ? "stable"
                                                                    : "changed";
  bool changed = std::string_view(boot_state) == "changed",
       incomplete = !boot_ok;
  Json fields = Json::array();
  const auto memory = memory_values(mem.bytes);
  std::size_t observed = 0;
  for (const auto &field : selected) {
    std::string state, source, consistency = "not_checked";
    Json verification = nullptr;
    Json value = nullptr;
    if (field == "cpu_present" || field == "cpu_online") {
      const auto &a = first.at(field), &b = last.at(field);
      source = field == "cpu_present" ? "/sys/devices/system/cpu/present"
                                      : "/sys/devices/system/cpu/online";
      state = status(a.status);
      if (a.status == ReadStatus::ok) {
        value = cpu_value(a.bytes);
        if (value.is_null())
          state = "malformed";
      }
      const auto later =
          b.status == ReadStatus::ok ? cpu_value(b.bytes) : Json(nullptr);
      verification = b.status == ReadStatus::ok && later.is_null()
                         ? "malformed"
                         : status(b.status);
      if (state == "observed" && !later.is_null()) {
        consistency = value == later ? "stable" : "changed";
        changed |= consistency == "changed";
      } else
        consistency = "unavailable";
      if (consistency == "unavailable")
        incomplete = true;
    } else {
      source = "/proc/meminfo";
      state = status(mem.status);
      const auto label = field == "memory_total" ? "MemTotal" : "MemAvailable";
      if (mem.status == ReadStatus::ok) {
        const auto found = memory.find(label);
        if (found == memory.end())
          state = "unavailable";
        else if (found->second.is_null())
          state = "malformed";
        else
          value = found->second;
      }
    }
    if (state == "observed")
      ++observed;
    else
      incomplete = true;
    fields.push_back({{"field", field},
                      {"source_path", source},
                      {"status", state},
                      {"value", value},
                      {"consistency", consistency},
                      {"verification_status", verification}});
  }
  guard();
  const auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
                           std::chrono::steady_clock::now() - steady_start)
                           .count();
  return {
      {"protocol", "symphony.snv.local-observe.v1"},
      {"owner", "local-observer"},
      {"owner_version", version},
      {"source_digest", engine::tagged_sha256(p.dump())},
      {"observation_id", p.at("observation_id")},
      {"node_ref", p.at("node_ref")},
      {"subject_ids", p.at("node_ref").is_null()
                          ? Json::array()
                          : Json::array({p.at("node_ref")})},
      {"profile", p.at("profile")},
      {"method", {{"id", "linux-proc-sysfs"}, {"version", "1"}}},
      {"platform",
       {{"os", platform.available ? Json("linux") : Json(nullptr)},
        {"architecture",
         platform.available ? Json(platform.architecture) : Json(nullptr)},
        {"kernel_release",
         platform.available ? Json(platform.kernel_release) : Json(nullptr)}}},
      {"visibility_scope", "invoked_process_kernel_proc_sysfs_exposure"},
      {"acquisition_route", acquisition_route},
      {"node_association", "caller_supplied_unverified"},
      {"physical_inventory_complete", false},
      {"allocation_verified", false},
      {"capture",
       {{"started_unix_ms", started},
        {"finished_unix_ms", engine::unix_time_ms()},
        {"elapsed_ms", elapsed},
        {"boot_consistency", boot_state},
        {"boot_before_status", boot_before},
        {"boot_after_status", boot_after},
        {"atomic_inventory", false}}},
      {"status", changed ? "changed_during_capture"
                 : !platform.available || observed == 0 ? "unavailable"
                 : incomplete                           ? "partial"
                                                        : "complete"},
      {"fields", fields},
      {"limitations",
       Json::array({"logical_cpu_indices_not_physical_cores",
                    "kernel_memory_view_not_cgroup_allocation",
                    "stable_checks_cover_only_checked_interval_fields",
                    "memory_available_is_kernel_estimate",
                    "no_physical_identity_or_full_inventory_attestation"})}};
}
} // namespace
Json collect(const Json &p, SourceReader &reader, std::int64_t d) {
  return collect_with_route(p, reader, d, "sdk_supplied_reader");
}
Json handle(std::string_view op, const Json &p, std::int64_t d) {
  if (op != "observe")
    throw engine::Error("unsupported_operation",
                        "unsupported observer operation", 2);
  NativeReader reader;
  return collect_with_route(p, reader, d, "native_fixed_sources");
}
} // namespace symphony::snv::local_observer
