#include <iostream>
#include <map>
#include <source_location>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/snv/local_observer.hpp>
#include <thread>
namespace {
namespace observer = symphony::snv::local_observer;
namespace engine = symphony::knowledge::engine;
using observer::Json;
using observer::ReadStatus;
using observer::Source;
int checks = 0;
int source_checks = 0;
void check(bool b, const char *message,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!b)
    throw std::runtime_error(std::string(message) + " at line " +
                             std::to_string(loc.line()));
}
template <class F>
void rejects(F f, const char *code = "invalid_input",
             std::source_location loc = std::source_location::current()) {
  bool failed = false;
  try {
    f();
  } catch (const engine::Error &e) {
    failed = e.code() == code;
  }
  check(failed, "expected exact rejection", loc);
}
struct Fixture final : observer::SourceReader {
  observer::Platform target{true, "fixture-architecture", "fixture-kernel"};
  std::map<Source, std::vector<observer::ReadResult>> scripts{
      {Source::boot,
       {{ReadStatus::ok, "00000000-1111-2222-3333-444444444444\n"}}},
      {Source::cpu_present, {{ReadStatus::ok, "0-7\n"}}},
      {Source::cpu_online, {{ReadStatus::ok, "0-3\n"}}},
      {Source::meminfo,
       {{ReadStatus::ok, "MemTotal:       1024 kB\nMemAvailable:    512 "
                         "kB\nIgnored: secret-hostname-and-serial\n"}}}};
  std::vector<Source> calls;
  std::map<Source, std::size_t> indices;
  bool expire = false;
  observer::Platform platform() const override { return target; }
  observer::ReadResult read(Source source, std::size_t limit,
                            std::int64_t) override {
    calls.push_back(source);
    ++source_checks;
    if (limit != (source == Source::boot      ? 64
                  : source == Source::meminfo ? 16384
                                              : 4096))
      throw std::runtime_error("fixed source byte bound");
    if (expire)
      std::this_thread::sleep_for(std::chrono::milliseconds(10));
    const auto &script = scripts.at(source);
    return script.at(std::min(indices[source]++, script.size() - 1));
  }
};
Json request() {
  return {{"protocol", "symphony.snv.local-observe-input.v1"},
          {"observation_id", "fixture-observation"},
          {"node_ref", nullptr},
          {"profile", "linux-proc-sysfs-v1"},
          {"fields", Json::array({"cpu_present", "cpu_online", "memory_total",
                                  "memory_available"})}};
}
Json run(Fixture &f, const Json &p = request()) {
  return observer::collect(p, f, engine::unix_time_ms() + 1000);
}
const Json &field(const Json &r, const char *name) {
  for (const auto &v : r.at("fields"))
    if (v.at("field") == name)
      return v;
  throw std::runtime_error("field absent");
}
} // namespace
int main() {
  try {
    Fixture f;
    const auto p = request(), r = run(f, p);
    check(r.at("status") == "complete" &&
              r.at("capture").at("boot_consistency") == "stable",
          "selected profile complete with stable checked boot");
    check(field(r, "cpu_present").at("value").at("value") == "8" &&
              field(r, "cpu_online").at("value").at("value") == "4",
          "present logical indices remain separate from online");
    check(field(r, "memory_total").at("value").at("value") == "1048576" &&
              field(r, "memory_available").at("value").at("value") == "524288",
          "reported kB converted exactly to decimal bytes");
    check(r.at("physical_inventory_complete") == false &&
              r.at("allocation_verified") == false &&
              r.at("capture").at("atomic_inventory") == false,
          "no invented inventory/allocation/atomic capture");
    check(r.at("source_digest") == engine::tagged_sha256(p.dump()) &&
              r.at("subject_ids").empty(),
          "original request binding and no invented Node ID");
    check(r.at("acquisition_route") == "sdk_supplied_reader",
          "injected reader evidence stays explicitly attributed");
    check(r.dump().find("00000000-1111") == std::string::npos &&
              r.dump().find("secret-hostname") == std::string::npos,
          "boot and unrelated sensitive content omitted");
    auto supplied = p;
    supplied["node_ref"] = "supplied-node";
    Fixture association;
    const auto associated = run(association, supplied);
    check(associated.at("subject_ids") == Json::array({"supplied-node"}) &&
              associated.at("node_association") == "caller_supplied_unverified",
          "Node reference remains supplied association only");
    auto one = p;
    one["fields"] = Json::array({"memory_total"});
    Fixture selected;
    const auto limited = run(selected, one);
    check(limited.at("fields").size() == 1 &&
              selected.calls == std::vector<Source>{Source::boot,
                                                    Source::meminfo,
                                                    Source::boot},
          "no unrequested CPU source read or scan");
    Fixture absent;
    absent.target.available = false;
    check(run(absent, p).at("status") == "unavailable" && absent.calls.empty(),
          "unavailable platform performs no source reads");
    Fixture reboot;
    reboot.scripts[Source::boot].push_back(
        {ReadStatus::ok, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\n"});
    check(run(reboot).at("status") == "changed_during_capture",
          "reboot during capture is explicit");
    Fixture hotplug;
    hotplug.scripts[Source::cpu_online].push_back({ReadStatus::ok, "0-1\n"});
    const auto changed = run(hotplug);
    check(changed.at("status") == "changed_during_capture" &&
              field(changed, "cpu_online").at("value").at("value") == "4" &&
              field(changed, "cpu_online").at("consistency") == "changed",
          "CPU change preserves first observation without simultaneous claim");
    Fixture unreadable;
    unreadable.scripts[Source::cpu_present] = {
        {ReadStatus::permission_denied, {}}};
    check(run(unreadable).at("status") == "partial",
          "unreadable field is partial");
    Fixture last_denied;
    last_denied.scripts[Source::cpu_online].push_back(
        {ReadStatus::permission_denied, {}});
    const auto unverifiable = run(last_denied);
    check(field(unverifiable, "cpu_online").at("verification_status") ==
                  "permission_denied" &&
              unverifiable.at("status") == "partial",
          "repeat-read permission failure remains explicit");
    Fixture boot_denied;
    boot_denied.scripts[Source::boot] = {{ReadStatus::permission_denied, {}}};
    check(run(boot_denied).at("capture").at("boot_before_status") ==
              "permission_denied",
          "boot access failure stays explicit without raw marker");
    Fixture malformed_boot;
    malformed_boot.scripts[Source::boot] = {
        {ReadStatus::ok, "private-but-invalid-marker"}};
    check(run(malformed_boot).at("capture").at("boot_consistency") ==
              "unavailable",
          "malformed boot marker never fabricates consistency");
    for (const auto status :
         {ReadStatus::unsupported_source, ReadStatus::too_large,
          ReadStatus::io_error, ReadStatus::unavailable}) {
      Fixture failure;
      failure.scripts[Source::meminfo] = {{status, {}}};
      check(field(run(failure), "memory_total").at("value").is_null(),
            "unavailable/unsafe source does not become zero capacity");
    }
    Fixture oversized;
    oversized.scripts[Source::cpu_present] = {
        {ReadStatus::ok, std::string(4097, '1')}};
    check(field(run(oversized), "cpu_present").at("status") == "too_large",
          "SDK cannot evade byte profile");
    for (const auto *mask :
         {"1,1", "2-1", "0-2,2-4", "1048576", "01", "1,", "1-1", "-1"}) {
      Fixture invalid;
      invalid.scripts[Source::cpu_present] = {{ReadStatus::ok, mask}};
      check(field(run(invalid), "cpu_present").at("status") == "malformed",
            "invalid logical CPU map rejected");
    }
    for (const auto *mem :
         {"MemTotal: 18446744073709551615 kB\n", "MemTotal: 1 MB\n",
          "MemTotal: 1 kB\nMemTotal: 2 kB\n", "MemTotal: 01 kB\n"}) {
      Fixture invalid;
      invalid.scripts[Source::meminfo] = {{ReadStatus::ok, mem}};
      check(field(run(invalid), "memory_total").at("status") == "malformed",
            "overflow/unit/duplicate/noncanonical quantities remain malformed");
    }
    Fixture missing;
    missing.scripts[Source::meminfo] = {{ReadStatus::ok, "MemTotal: 1 kB\n"}};
    check(field(run(missing), "memory_available").at("status") == "unavailable",
          "missing estimated availability is explicit");
    for (const auto *key : {"root", "command", "endpoint"}) {
      auto invalid = p;
      invalid[key] = "unrequested";
      Fixture reader;
      rejects([&] { run(reader, invalid); });
      check(reader.calls.empty(), "arbitrary input rejected before collection");
    }
    auto invalid = p;
    invalid["fields"] = Json::array();
    Fixture empty;
    rejects([&] { run(empty, invalid); });
    invalid = p;
    invalid["fields"] = Json::array({"cpu_online", "cpu_online"});
    Fixture duplicate;
    rejects([&] { run(duplicate, invalid); });
    invalid = p;
    invalid["fields"] = Json::array({"serial"});
    Fixture sensitive;
    rejects([&] { run(sensitive, invalid); });
    invalid = p;
    invalid["profile"] = "anything-else";
    Fixture unsupported;
    rejects([&] { run(unsupported, invalid); });
    rejects(
        [&] { observer::handle("shell", p, engine::unix_time_ms() + 1000); },
        "unsupported_operation");
    Fixture expired;
    rejects([&] { observer::collect(p, expired, engine::unix_time_ms() - 1); },
            "deadline_exceeded");
    check(expired.calls.empty(), "expired request reads nothing");
    Fixture expire_during;
    expire_during.expire = true;
    rejects(
        [&] {
          observer::collect(p, expire_during, engine::unix_time_ms() + 5);
        },
        "deadline_exceeded");
    const auto descriptor = observer::descriptor();
    check(descriptor.at("engine_id") == observer::engine_id &&
              descriptor.at("vector_id").is_null() &&
              descriptor.at("limits").at("records") == 4,
          "collector descriptor distinct and bounded");
    auto unsigned_descriptor = descriptor;
    unsigned_descriptor.erase("descriptor_digest");
    check(descriptor.at("descriptor_digest") ==
              engine::tagged_sha256(unsigned_descriptor.dump()),
          "descriptor binds exact compiled shape");
#ifndef __linux__
    const auto native =
        observer::handle("observe", p, engine::unix_time_ms() + 1000);
    check(native.at("status") == "unavailable" &&
              native.at("acquisition_route") == "native_fixed_sources",
          "portable native executable does not claim Linux observation");
#endif
    std::cout << "Observer " << checks << " scenario checks and "
              << source_checks
              << " source-bound checks passed (synthetic/native-platform "
                 "availability; not Linux target conformance)\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
