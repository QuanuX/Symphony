#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include <atomic>
#include <chrono>
#include <cstring>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/sbv/sbv.hpp>
#include <symphony/sqav/databento/dbn.hpp>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>
#include <thread>
#include <unistd.h>
namespace e = symphony::knowledge::engine;
namespace s = symphony::sbv;
using J = s::Json;
unsigned checks = 0;
void check(bool ok,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("test failure line " + std::to_string(loc.line()));
}
template <class F> void rejects(F f) {
  bool rejected = false;
  try {
    f();
  } catch (...) {
    rejected = true;
  }
  check(rejected);
}
J call(const std::string &op, const J &p) {
  return s::dispatch(op, p, e::no_deadline);
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
void save(const std::string &path, const std::string &bytes) {
  std::ofstream f(path, std::ios::binary);
  f << bytes;
}
J rat(const char *n, const char *d) {
  return {{"numerator", n}, {"denominator", d}};
}
int main() try {
  char temp[] = "/private/tmp/sbv-contract-XXXXXX";
  check(::mkdtemp(temp) != nullptr);
  const std::string root = temp;
  struct Cleanup {
    std::string root;
    ~Cleanup() { std::filesystem::remove_all(root); }
  } cleanup{root};
  auto caps = call("capabilities",
                   {{"protocol", "symphony.sbv.capabilities-input.v1"}});
  check(caps.at("cpu").at("available") == true);
  check(caps.at("cuda").at("available") == false);
  check(caps.at("live").at("available") == false);
  auto original = public_fixture::fixture();
  std::vector<unsigned char> bytes(original.begin(), original.begin() + 360);
  auto put = [&](std::size_t offset, std::uint64_t n, unsigned length) {
    for (unsigned i = 0; i < length; ++i)
      bytes[offset + i] = static_cast<unsigned char>(n >> (8 * i));
  };
  for (unsigned i = 0; i < 4; ++i) {
    const auto off = bytes.size();
    bytes.insert(bytes.end(), original.begin() + 360, original.begin() + 416);
    put(off + 8, 1609160400000000000ULL + i * 100, 8);
    put(off + 40, 1609160400000000000ULL + i * 100, 8);
    put(off + 24, (100 + i) * 1000000000ULL, 8);
    bytes[off + 38] = 'T';
  }
  const std::string source(reinterpret_cast<const char *>(bytes.data()),
                           bytes.size());
  save(root + "/sample.dbn", source);
  J p{{"protocol", "symphony.sbv.run-input.v1"},
      {"source_path", root + "/sample.dbn"},
      {"source_sha256", e::sha256_hex(source)},
      {"dataset", "GLBX.MDP3"},
      {"output_path", root + "/serial.json"},
      {"criteria",
       {{"rule", "spaced_trades"},
        {"spacing_ns", "100"},
        {"min_trade_size", "1"},
        {"direction", "any"},
        {"max_signals", "4"}}},
      {"replay",
       {{"before_ns", "100"}, {"after_ns", "100"}, {"retain_events", true}}},
      {"execution",
       {{"model", "touch_observation"},
        {"horizon_ns", "100"},
        {"price_offsets_nanos", J::array({"1000000000", "-1000000000"})},
        {"probability_numerator", "0"},
        {"probability_denominator", "1"}}},
      {"studies", J::array({"signal_summary", "forward_markout"})},
      {"workers", "1"},
      {"extensions",
       {{"user/key~name",
         J::array({"9007199254740993", true, "\x1b[31m", J::object()})}}}};
  const auto baseline = call("run", p);
  const auto scientific = read(root + "/serial.json");
  J load{{"protocol", "symphony.sbv.dataset-load-input.v1"},
         {"directory", root},
         {"instance_id", "00000000000000000000000000000001"},
         {"source_path", p["source_path"]},
         {"source_sha256", p["source_sha256"]},
         {"dataset", p["dataset"]},
         {"memory_budget_bytes", "1048576"},
         {"residency", "pageable"},
         {"max_concurrent_jobs", "4"},
         {"worker_budget", "4"},
         {"idle_timeout_ms", "60000"}};
  J ref{{"protocol", "symphony.sbv.dataset-inspect-input.v1"},
        {"directory", root},
        {"instance_id", load["instance_id"]}};
  const auto loaded = call("dataset_load", load);
  check(loaded["source_reads"] == "1" && loaded["decode_passes"] == "1" &&
        loaded["events"] == "4");
  rejects([&] { call("dataset_load", load); });
  auto q = p;
  q.erase("output_path");
  J job{{"protocol", "symphony.sbv.dataset-execute-input.v1"},
        {"directory", root},
        {"instance_id", load["instance_id"]},
        {"operation", "run"},
        {"request", q},
        {"output_path", root + "/resident.json"}};
  // Frozen bytes remain usable even after the original source disappears.
  std::filesystem::remove(root + "/sample.dbn");
  call("dataset_execute", job);
  auto result = read(root + "/resident.json");
  s::validate_result(result);
  for (const auto *section :
       {"signals", "execution", "studies", "replay", "summary", "provenance",
        "choices", "diagnostics"})
    check(result["sections"][section] == scientific["sections"][section]);
  check(result["sections"]["resources"]["data"]["dataset_feed"]
              ["source_reads_this_job"] == "0");
  check(result["sections"]["resources"]["data"]["dataset_feed"]
              ["decodes_this_job"] == "0");
  check(call("dataset_inspect", ref)["completed_jobs"] == "1");
  rejects([&] { call("dataset_execute", job); }); // immutable output
  auto mismatch = job;
  mismatch["request"]["source_sha256"] = std::string(64, '0');
  mismatch["output_path"] = root + "/wrong.json";
  rejects([&] { call("dataset_execute", mismatch); });
  auto cap = job;
  cap["request"]["workers"] = "5";
  cap["output_path"] = root + "/cap.json";
  rejects([&] { call("dataset_execute", cap); });
  std::vector<std::jthread> callers;
  std::atomic<unsigned> good{0};
  for (unsigned i = 0; i < 4; ++i)
    callers.emplace_back([&, i] {
      auto j = job;
      j["output_path"] = root + "/parallel-" + std::to_string(i) + ".json";
      try {
        call("dataset_execute", j);
        ++good;
      } catch (...) {
      }
    });
  callers.clear();
  check(good == 4);
  auto status = call("dataset_inspect", ref);
  check(status["completed_jobs"] == "5" && status["source_reads"] == "1" &&
        status["decode_passes"] == "1");
  auto release = ref;
  release["protocol"] = "symphony.sbv.dataset-release-input.v1";
  check(call("dataset_release", release)["state"] == "released");
  rejects([&] { call("dataset_inspect", ref); });
  rejects([&] { call("dataset_execute", job); });
  save(root + "/sample.dbn", source);
  rejects([&] { call("dataset_load", load); }); // incarnation never reused
  load["instance_id"] = "00000000000000000000000000000002";
  load["memory_budget_bytes"] = "1";
  rejects([&] { call("dataset_load", load); });
  load["instance_id"] = "00000000000000000000000000000003";
  load["memory_budget_bytes"] = "1048576";
  load["source_sha256"] = std::string(64, '0');
  rejects([&] { call("dataset_load", load); });
  load["source_sha256"] = p["source_sha256"];
  load["instance_id"] = "00000000000000000000000000000004";
  load["idle_timeout_ms"] = "1000";
  call("dataset_load", load);
  ref["instance_id"] = load["instance_id"];
  std::this_thread::sleep_for(std::chrono::milliseconds(1300));
  rejects([&] { call("dataset_inspect", ref); });
  load["instance_id"] = "00000000000000000000000000000005";
  load["residency"] = "locked";
  load["idle_timeout_ms"] = "60000";
  // Locked mode is either actually admitted by the OS or explicitly refused.
  try {
    auto locked = call("dataset_load", load);
    check(locked["residency"] == "locked");
    release["instance_id"] = load["instance_id"];
    call("dataset_release", release);
  } catch (const e::Error &x) {
    check(std::string(x.what()).find("locked RAM unavailable") !=
          std::string::npos);
  }
  load["instance_id"] = "00000000000000000000000000000006";
  ::chmod(root.c_str(), 0755);
  rejects([&] { call("dataset_load", load); });
  ::chmod(root.c_str(), 0700);
  // Release must terminate even if the authenticated client has closed
  // its receiving direction and cannot receive the acknowledgement.
  load["instance_id"] = "00000000000000000000000000000007";
  load["residency"] = "pageable";
  load["idle_timeout_ms"] = "0";
  call("dataset_load", load);
  release["instance_id"] = load["instance_id"];
  const auto socket_path =
      root + "/" + load["instance_id"].get<std::string>() + ".sock";
  int fd = ::socket(AF_UNIX, SOCK_STREAM, 0);
  check(fd >= 0);
  sockaddr_un address{};
  address.sun_family = AF_UNIX;
  std::strcpy(address.sun_path, socket_path.c_str());
  check(::connect(fd, reinterpret_cast<sockaddr *>(&address),
                  sizeof(address)) == 0);
  // Wait for the host's authenticated readiness frame before abandoning reads.
  timeval timeout{2, 0};
  check(::setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout)) ==
        0);
  char header[4];
  check(::recv(fd, header, 4, MSG_WAITALL) == 4);
  std::size_t length = 0;
  for (auto c : header)
    length = (length << 8) | static_cast<unsigned char>(c);
  check(length > 0 && length < 1024);
  std::string hello(length, '\0');
  check(::recv(fd, hello.data(), hello.size(), MSG_WAITALL) ==
        static_cast<ssize_t>(hello.size()));
  check(J::parse(hello).at("instance_id") == load["instance_id"]);
  J wire{{"protocol", "symphony.sbv.resident-wire.v2"},
         {"engine_version", s::version},
         {"operation", "dataset_release"},
         {"input", release},
         {"deadline_ms", nullptr}};
  const auto message = wire.dump();
  std::string frame(4, '\0');
  for (unsigned i = 0; i < 4; ++i)
    frame[i] = static_cast<char>(message.size() >> (24 - 8 * i));
  frame += message;
  // Exceed the former two-second handshake cap before submitting the request.
  std::this_thread::sleep_for(std::chrono::milliseconds(2100));
  check(::shutdown(fd, SHUT_RD) == 0);
  check(::send(fd, frame.data(), frame.size(), 0) ==
        static_cast<ssize_t>(frame.size()));
  for (unsigned i = 0; i < 100 && std::filesystem::exists(socket_path); ++i)
    std::this_thread::sleep_for(std::chrono::milliseconds(10));
  ::close(fd);
  // Best-effort explicit cleanup if the assertion below fails.
  const bool released = !std::filesystem::exists(socket_path);
  if (!released)
    call("dataset_release", release);
  check(released);
  // Cross all former source-byte, event-count and metadata ceilings together.
  namespace db = symphony::sqav::databento;
  constexpr std::size_t extra_symbols = 15000, count = 1200000;
  std::vector<unsigned char> large(original.begin(), original.begin() + 116);
  for (std::size_t i = 0; i <= extra_symbols; ++i)
    large.insert(large.end(), original.begin() + 116, original.begin() + 187);
  large.insert(large.end(), original.begin() + 187, original.begin() + 360);
  auto write_le = [&](std::size_t offset, std::uint64_t value, unsigned width) {
    for (unsigned i = 0; i < width; ++i)
      large[offset + i] = static_cast<unsigned char>(value >> (8 * i));
  };
  const auto metadata_size = large.size();
  write_le(4, metadata_size - 8, 4);
  write_le(112, extra_symbols + 1, 4);
  large.reserve(metadata_size + count * 56);
  for (std::size_t i = 0; i < count; ++i) {
    const auto offset = large.size();
    large.insert(large.end(), bytes.begin() + 360, bytes.begin() + 416);
    write_le(offset + 8, 1609160400000000000ULL + i * 100, 8);
    write_le(offset + 40, 1609160400000000000ULL + i * 100, 8);
  }
  check(large.size() > (64U << 20) && metadata_size > (1U << 20));
  db::FileView view;
  check(db::FileView::inspect_dataset(large, {}, view) == db::Status::ok);
  check(view.metadata().record_count == count);
  const auto prior_data = view.original().data();
  check(db::FileView::inspect(large, {64U << 20, 1U << 20, 1U << 20}, view) ==
        db::Status::limit);
  check(view.original().data() == prior_data);
  for (auto l : {db::DatasetLimits{large.size() - 1, {}, {}},
                 db::DatasetLimits{{}, metadata_size - 1, {}},
                 db::DatasetLimits{{}, {}, count - 1}}) {
    check(db::FileView::inspect_dataset(large, l, view) == db::Status::limit);
    check(view.original().data() == prior_data);
  }
  check(db::FileView::inspect_dataset(large, {0, {}, {}}, view) ==
        db::Status::invalid_argument);
  check(db::FileView::inspect_dataset(large,
                                      {UINT64_MAX, UINT64_MAX, UINT64_MAX},
                                      view) == db::Status::ok);
  const auto large_path = root + "/large.dbn";
  {
    std::ofstream f(large_path, std::ios::binary);
    f.write(reinterpret_cast<const char *>(large.data()), large.size());
  }
  p["source_path"] = large_path;
  p["source_sha256"] = e::sha256_hex(std::span<const unsigned char>(large));
  p["criteria"]["max_signals"] = "2";
  p["output_path"] = root + "/large-file.json";
  large.clear();
  large.shrink_to_fit();
  call("run", p); // omitted controls mean no user ceiling
  const auto file_large = read(p["output_path"]);
  check(file_large["sections"]["resources"]["data"]["dataset_feed"]
                  ["memory_budget_bytes"]
                      .is_null());
  J unlimited{{"max_source_bytes", nullptr},
              {"max_source_events", nullptr},
              {"max_metadata_bytes", nullptr}};
  load["source_path"] = p["source_path"];
  load["source_sha256"] = p["source_sha256"];
  load["memory_budget_bytes"] = nullptr;
  load["dataset_limits"] = unlimited;
  load["instance_id"] = "00000000000000000000000000000008";
  load["idle_timeout_ms"] = "60000";
  load["idle_timeout_ms"] = "18446744073709551615";
  const auto big =
      s::dispatch("dataset_load", load, e::unix_time_ms() + 7LL * 86400000);
  check(big["events"] == std::to_string(count) &&
        big["memory_budget_bytes"].is_null() &&
        big["dataset_limits"] == unlimited);
  job["instance_id"] = load["instance_id"];
  job["request"] = p;
  job["request"].erase("output_path");
  job["request"]["memory_budget_bytes"] = "1099511627776";
  job["request"]["dataset_limits"] = unlimited;
  for (auto key :
       {"max_source_bytes", "max_source_events", "max_metadata_bytes"})
    job["request"]["dataset_limits"][key] = "1099511627776";
  job["output_path"] = root + "/large-resident.json";
  call("dataset_execute", job);
  const auto resident_large = read(job["output_path"]);
  for (auto key :
       {"signals", "execution", "studies", "replay", "summary", "provenance"})
    check(resident_large["sections"][key] == file_large["sections"][key]);
  check(resident_large["sections"]["choices"]["data"]["memory_budget_bytes"] ==
        "1099511627776");
  job["output_path"] = root + "/large-rejected.json";
  for (auto key :
       {"max_source_bytes", "max_source_events", "max_metadata_bytes"}) {
    job["request"]["dataset_limits"] = unlimited;
    job["request"]["dataset_limits"][key] = "1";
    rejects([&] { call("dataset_execute", job); });
  }
  job["request"]["dataset_limits"] = unlimited;
  job["request"]["memory_budget_bytes"] = "1";
  rejects([&] { call("dataset_execute", job); });
  p["output_path"] = root + "/file-rejected.json";
  p["dataset_limits"] = unlimited;
  for (auto key :
       {"max_source_bytes", "max_source_events", "max_metadata_bytes"}) {
    p["dataset_limits"] = unlimited;
    p["dataset_limits"][key] = "1";
    rejects([&] { call("run", p); });
  }
  p["dataset_limits"] = unlimited;
  for (J invalid : {J("0"), J("01"), J("18446744073709551616"), J(100)}) {
    p["memory_budget_bytes"] = invalid;
    rejects([&] { call("run", p); });
  }
  release["instance_id"] = load["instance_id"];
  check(call("dataset_release", release)["state"] == "released");
  check(caps["limits"]["max_source_events"].is_null() &&
        caps["limits"]["max_source_bytes"].is_null() &&
        caps["limits"]["dataset_limit_authority"] == "user");
  std::cout << checks << " resident dataset checks passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
