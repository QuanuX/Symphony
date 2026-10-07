#ifdef __APPLE__
#define SBV_TEST_TMP "/private/tmp"
#else
#define SBV_TEST_TMP "/tmp"
#endif
#include "exact_id_index.hpp"
#include <iostream>
#include <stdexcept>
#include <string>

namespace s = symphony::sbv::detail;
using Json = symphony::knowledge::engine::Json;
std::uint64_t checks = 0;
void require(bool yes, const char *message) {
  ++checks;
  if (!yes) throw std::runtime_error(message);
}
template<class Function> void rejects(Function &&f, const char *message) {
  bool rejected = false;
  try { f(); } catch (const std::exception &) { rejected = true; }
  require(rejected, message);
}
int main() {
  try {
    s::ExactIdIndex ids;
    require(ids.size() == 0 && ids.tracked_bytes() == 0, "empty identity state");
    require(!ids.find("absent"), "absent lookup");
    require(ids.insert("z").ordinal == 0 && ids.insert("a").ordinal == 1,
            "insertion order distinct from map sort order");
    const auto repeated = ids.insert("z");
    require(!repeated.inserted && repeated.ordinal == 0 && ids.size() == 2,
            "duplicate returns original ordinal");
    require(ids.find("a") == 1 && ids.find("z") == 0, "exact lookup order");
    require(ids.retained_key_bytes() == 2 && ids.tracked_bytes() == 18,
            "explicit accounting formula");
    const std::string null_key("a\0b", 3);
    require(ids.insert(null_key).inserted && ids.find(null_key) == 2 &&
                ids.find("a") == 1, "exact bytes do not truncate NUL");
    require(ids.insert("é").inserted && ids.insert("é").inserted,
            "no implicit Unicode normalization");
    require(ids.insert("").inserted, "domain caller controls empty ID admission");
    require(ids.evidence().at("max_tracked_bytes").is_null() &&
                ids.evidence().at("spill") == "not_supported",
            "no default cap or claimed spill");
    s::ExactIdIndex bounded(19);
    require(bounded.insert("abc").inserted && bounded.tracked_bytes() == 11,
            "budget accounts ordinal and key");
    require(bounded.insert("").inserted && bounded.tracked_bytes() == 19,
            "budget exact boundary");
    rejects([&] { bounded.insert("new"); }, "selected budget refuses new key");
    require(bounded.size() == 2 && bounded.tracked_bytes() == 19 &&
                !bounded.find("new"), "budget refusal has no partial mutation");
    require(!bounded.insert("abc").inserted && bounded.find("abc") == 0,
            "duplicate remains available at budget");
    s::ExactIdIndex zero(0);
    rejects([&] { zero.insert(""); }, "zero caller budget is meaningful");
    require(zero.size() == 0, "zero budget unchanged");
    bool cancel = false;
    s::ExactIdIndex cancellable({}, [&] {
      if (cancel) throw std::runtime_error("caller cancellation");
    });
    cancellable.insert("first"); cancel = true;
    rejects([&] { cancellable.insert("second"); }, "insertion cancellation");
    rejects([&] { (void)cancellable.find("first"); }, "lookup cancellation");
    cancel = false;
    require(cancellable.size() == 1 && cancellable.find("first") == 0,
            "cancellation has no mutation");
    s::ExactIdIndex large;
    std::uint64_t expected_keys = 0;
    constexpr std::uint64_t total = 65537;
    for (std::uint64_t i = 0; i < total; ++i) {
      const auto key = "signal-" + std::to_string(i);
      const auto added = large.insert(key);
      if (!added.inserted || added.ordinal != i)
        throw std::runtime_error("large insertion order mismatch");
      expected_keys += key.size();
    }
    require(large.size() == total && large.retained_key_bytes() == expected_keys &&
                large.tracked_bytes() == expected_keys + 8 * total,
            "large exact accounting");
    require(!large.insert("signal-0").inserted &&
                !large.insert("signal-65536").inserted &&
                large.find("signal-32768") == 32768 &&
                large.find("signal-65536") == 65536,
            "distant duplicates and middle/final lookups");
    std::cout << Json{{"status", "passed"}, {"checks", std::to_string(checks)},
                      {"large", large.evidence()}}.dump() << '\n';
    return 0;
  } catch (const std::exception &error) {
    std::cerr << error.what() << '\n'; return 1;
  }
}
