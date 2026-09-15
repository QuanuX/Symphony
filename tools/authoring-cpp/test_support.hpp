#pragma once
#include "authoring.hpp"
namespace symphony::authoring::test {
inline int checks = 0;
inline void equal(const Json &actual, const Json &expected,
                  const std::string &label) {
  ++checks;
  require(same(actual, expected), label);
}
inline void check(bool ok, const std::string &label) {
  ++checks;
  require(ok, label);
}
template <class Function>
void rejects(Function function, const std::string &label,
             const std::string &message = "") {
  bool rejected = false;
  try {
    function();
  } catch (const std::exception &error) {
    rejected = true;
    require(message.empty() ||
                std::string(error.what()).find(message) != std::string::npos,
            label + ": wrong diagnostic: " + error.what());
  }
  check(rejected, label + ": accepted invalid input");
}
inline void copy(const fs::path &root, const fs::path &target,
                 const std::string &path) {
  write_bytes(target / path, read_bytes(root / path));
}
inline void descriptor(const std::string &executable, Json expected,
                       const Json &version) {
  require(!executable.empty(),
          "--engine is required for independent descriptor parity");
  expected["engine_version"] = version;
  expected.erase("descriptor_digest");
  expected["descriptor_digest"] = digest(expected);
  const auto result = run({executable, "--descriptor"});
  check(result.status == 0, "native descriptor failed: " + result.error);
  test::equal(Json::parse(result.output), expected,
              "frozen native descriptor differs");
}
inline void done() {
  std::cout << "PASS " << checks << " native authoring assertions\n";
}
} // namespace symphony::authoring::test
