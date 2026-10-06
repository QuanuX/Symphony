// Build this file in a separate project against the installed header target.
#include <iostream>
#include <symphony/sbv/interop.hpp>
namespace i = symphony::sbv::interop;
struct HostCompletion final : i::Completion {
  i::CompletionState s = i::CompletionState::ready;
  i::CompletionState state() const noexcept override { return s; }
  i::CompletionState wait_until(std::uint64_t) override { return s; }
  bool request_cancel() noexcept override {
    s = i::CompletionState::cancelled;
    return true;
  }
};
int main() {
  auto data = std::make_shared<std::vector<float>>(6, 2.5f);
  i::TensorLease lease{
      {"f32", "cpu", "standalone-consumer", "host", 24, 0, {2, 3}, {12, 4}},
      data,
      data->data(),
      std::make_shared<HostCompletion>()};
  i::require(lease.ready_host_base() == data->data(),
             "ownership bridge mismatch");
  auto admitted = i::admit_buffer(lease.descriptor);
  i::require(admitted.required_bytes == 24, "extent mismatch");
  auto p = i::plan_backend(
      {"cuda", "cpu", "reduce", "independent", 8, 4096, 4, 8192});
  i::require(p.status == "planned" && p.backend == "cpu" && p.workers == 4 &&
                 p.fallback_used,
             "explicit fallback mismatch");
  i::require(lease.completion->request_cancel(), "cancellation unavailable");
  bool failed = false;
  try {
    lease.ready_host_base();
  } catch (const std::invalid_argument &) {
    failed = true;
  }
  i::require(failed, "cancelled lease exposed");
  std::cout << i::contract_version << " installed consumer passed\n";
}
