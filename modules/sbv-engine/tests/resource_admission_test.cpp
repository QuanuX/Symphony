#include <private_temp.hpp>
#include "../src/resource_admission.hpp"
#include <filesystem>
#include <iostream>
#include <source_location>
#include <symphony/sbv/interop.hpp>
#include <unistd.h>
namespace r = symphony::sbv::detail::resource;
namespace i = symphony::sbv::interop;
using J = symphony::sbv::Json;
unsigned checks{};
void check(bool b, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!b)
    throw std::runtime_error("resource check " + std::to_string(at.line()));
}
template <class F> void rejects(F f) {
  bool failed = false;
  try {
    f();
  } catch (...) {
    failed = true;
  }
  check(failed);
}
int main() try {
  check(!r::limit(nullptr));
  check(r::limit("18446744073709551615") == UINT64_MAX);
  for (auto bad : {J("0"), J("-1"), J("18446744073709551616"), J(1), J(true)})
    rejects([&] { r::limit(bad); });
  check(r::admits({}, 64, 65));
  check(r::admits(65, 0, 65));
  check(!r::admits(65, 1, 65));
  check(!r::admits(1, 2, 1));
  check(!r::admits({}, UINT64_MAX, 1));
  check(r::effective(UINT64_MAX, 2) == 2);
  check(r::product(65, 17) == 1105);
  check(r::product(UINT64_MAX, 1) == UINT64_MAX);
  rejects([&] { r::product(UINT64_MAX, 2); });
  i::BackendRequest old{"cpu", "", "reject", "independent", 65, 1, 65, 1};
  check(i::plan_backend(old).workers == 65);
  i::BackendRequestV2 request{"cpu",      "", "reject", "independent",
                              UINT64_MAX, 1,  {},       {}};
  check(i::plan_backend_v2(request).workers == UINT64_MAX);
  request.scheduling = "serial_state";
  check(i::plan_backend_v2(request).status == "unavailable");
  request.limit_policy = "reduce";
  check(i::plan_backend_v2(request).workers == 1);
  request.declared_worker_limit = 0;
  rejects([&] { i::plan_backend_v2(request); });
  // Real 65-thread barrier proves simultaneous admission without host
  // exhaustion.
  std::atomic<bool> stop{false};
  std::atomic<unsigned> entered{}, completed{};
  r::launch(65, stop, [&](std::size_t) {
    ++entered;
    while (entered < 65 && !stop)
      std::this_thread::yield();
    ++completed;
  });
  check(entered == 65 && completed == 65 && !stop);
  entered = 0;
  completed = 0;
  unsigned attempted{};
  rejects([&] {
    r::launch(
        65, stop,
        [&](std::size_t) {
          ++entered;
          while (!stop)
            std::this_thread::yield();
          ++completed;
        },
        [&](auto work) {
          if (++attempted == 4)
            throw std::runtime_error("injected launch failure");
          return std::jthread(std::move(work));
        });
  });
  check(stop && attempted == 4 && entered == 3 && completed == 3);
  auto directory = symphony::testing::temporary_pattern("sbv-resources-XXXXXX");
  check(::mkdtemp(directory.data()) != nullptr);
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{directory};
  J trials = J::array();
  for (unsigned n = 0; n < 65; ++n)
    trials.push_back({{"id", "trial-" + std::to_string(n)},
                      {"state", "pruned"},
                      {"operation", nullptr},
                      {"request", nullptr},
                      {"reason", "caller selection"},
                      {"parameters", J::object()},
                      {"lineage", J::object()}});
  J experiment{{"protocol", "symphony.sbv.experiment-input.v1"},
               {"output_path", std::string(directory) + "/observation.json"},
               {"directory", directory},
               {"experiment_id", "more-than-64"},
               {"trials", trials},
               {"workers", "65"},
               {"worker_budget", nullptr},
               {"on_failure", "continue"},
               {"extensions", J::object()}};
  auto receipt = symphony::sbv::dispatch(
      "experiment", experiment, symphony::knowledge::engine::no_deadline);
  check(receipt.at("summary").at("data").at("pruned") == "65");
  std::cout << "resource admission checks " << checks << '\n';
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
