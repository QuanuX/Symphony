#pragma once
#include "detail.hpp"
#include <atomic>
#include <optional>
#include <thread>
#include <vector>

namespace symphony::sbv::detail::resource {
inline std::size_t effective(std::uint64_t requested, std::size_t tasks) {
  return static_cast<std::size_t>(std::min<std::uint64_t>(requested, tasks));
}
inline std::optional<std::uint64_t> limit(const Json &j) {
  if (j.is_null())
    return std::nullopt;
  auto n = u64(j);
  need(n > 0, "positive user resource limit or null required");
  return n;
}
inline std::uint64_t product(std::uint64_t a, std::uint64_t b) {
  need(a == 0 || b <= UINT64_MAX / a,
       "requested resource product exceeds uint64 representation");
  return a * b;
}
inline bool admits(std::optional<std::uint64_t> budget, std::uint64_t active,
                   std::uint64_t requested) {
  return requested <= UINT64_MAX - active &&
         (!budget || (active <= *budget && requested <= *budget - active));
}
// A launch failure stops existing workers before joining them. No partial
// result is returned as successful. The launcher seam permits failure testing
// without exhausting the caller's host.
template <class Worker, class Launcher>
void launch(std::size_t count, std::atomic<bool> &stop, Worker worker,
            Launcher launcher) {
  std::vector<std::jthread> threads;
  try {
    threads.reserve(count);
    for (std::size_t i = 0; i < count && !stop; ++i)
      threads.push_back(launcher([&, i] { worker(i); }));
  } catch (...) {
    stop = true;
    threads.clear();
    throw;
  }
  threads.clear();
}
template <class Worker>
void launch(std::size_t count, std::atomic<bool> &stop, Worker worker) {
  launch(count, stop, worker,
         [](auto job) { return std::jthread(std::move(job)); });
}
} // namespace symphony::sbv::detail::resource
