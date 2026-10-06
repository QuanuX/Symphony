#include "../src/detail.hpp"
#include <atomic>
#include <iostream>
#include <source_location>
#include <symphony/sbv/sdk.hpp>
#include <thread>
namespace s = symphony::sbv;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool b, std::source_location l = std::source_location::current()) {
  ++checks;
  if (!b)
    throw std::runtime_error("sdk assertion " + std::to_string(l.line()));
}
int main() try {
  check(symphony_sbv_sdk_abi_v1() == 1);
  check(std::string(symphony_sbv_sdk_version_v1()) == s::version);
  char *buffer = nullptr;
  std::size_t size = 0;
  check(symphony_sbv_sdk_process_v1(nullptr, 1, &buffer, &size) == 64 &&
        buffer == nullptr && size == 0);
  check(symphony_sbv_sdk_process_v1(nullptr, 0, nullptr, &size) == 64);
  check(symphony_sbv_sdk_process_v1(nullptr, 0, &buffer, nullptr) == 64);
  symphony_sbv_sdk_release_v1(nullptr);
  J input{{"protocol", e::process_protocol_v1},
          {"request_id", "sdk-test"},
          {"correlation_id", "sdk-test"},
          {"operation", "capabilities"},
          {"target_engine", "symphony-sbv"},
          {"deadline_unix_ms", e::unix_time_ms() + 30000},
          {"payload", {{"protocol", "symphony.sbv.capabilities-input.v1"}}}};
  auto response = s::sdk::process(input.dump());
  check(response.status == 0);
  auto parsed = J::parse(response.json);
  check(parsed.at("result") == s::dispatch("capabilities", input.at("payload"),
                                           e::unix_time_ms() + 30000));
  auto digest = parsed.at("response_digest");
  parsed.erase("response_digest");
  check(digest == e::tagged_sha256(parsed.dump()));
  auto q = input;
  q["operation"] = "descriptor";
  q["payload"] = J::object();
  auto descriptor = s::sdk::process(q.dump());
  check(descriptor.status == 0 &&
        J::parse(descriptor.json).at("result") == s::descriptor());
  q["payload"] = {{"bad", true}};
  check(s::sdk::process(q.dump()).status != 0);
  for (unsigned i = 0; i < 6; ++i) {
    q = input;
    if (i == 0)
      q["target_engine"] = "wrong";
    if (i == 1)
      q["deadline_unix_ms"] = 0;
    if (i == 2)
      q["deadline_unix_ms"] = e::unix_time_ms() + 400000;
    if (i == 3)
      q["payload"]["unknown"] = "x";
    if (i == 4)
      q["operation"] = "not-an-operation";
    if (i == 5)
      q["payload"]["protocol"] = "wrong";
    auto bad = s::sdk::process(q.dump());
    check(bad.status != 0);
    check(J::parse(bad.json).at("outcome") == "error");
  }
  for (const auto &raw :
       {std::string{}, std::string("{}"), std::string("{\"x\":1,\"x\":2}"),
        std::string("\xff"), std::string("{}\0trailing", 11)}) {
    auto bad = s::sdk::process(raw);
    check(bad.status != 0);
    check(J::parse(bad.json).at("outcome") == "error");
  }
  check(symphony_sbv_sdk_process_v1("x", (1U << 20) + 1, &buffer, &size) != 0);
  check(buffer && size > 0 && buffer[size] == '\0');
  symphony_sbv_sdk_release_v1(buffer);
  // Borrowed input is unchanged; concurrent callers own separate responses.
  const auto raw = input.dump();
  std::atomic<unsigned> good{0};
  std::vector<std::jthread> threads;
  for (unsigned n = 0; n < 8; ++n)
    threads.emplace_back([&] {
      for (unsigned j = 0; j < 16; ++j) {
        auto r = s::sdk::process(raw);
        if (r.status == 0 &&
            J::parse(r.json).at("result").at("engine_version") == s::version)
          ++good;
      }
    });
  threads.clear();
  check(good == 128);
  check(raw == input.dump());
  std::cout << checks
            << " SDK ABI assertions plus 128 concurrent calls passed\n";
  return 0;
} catch (const std::exception &e) {
  std::cerr << e.what() << '\n';
  return 1;
}
