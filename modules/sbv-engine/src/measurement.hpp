#pragma once
// Private benchmark instrumentation. No observer is installed by native
// process, SDK or resident dispatch; ordinary requests never sample a clock
// here.
#include <array>
#include <chrono>
#include <cstdint>
namespace symphony::sbv::detail::measurement {
enum class Phase : std::size_t { read, decode, compute, publication, count };
struct Record {
  std::uint64_t wall_ns = 0, calls = 0;
};
struct Recorder {
  std::array<Record, static_cast<std::size_t>(Phase::count)> phases{};
};
class Scope;
inline thread_local Recorder *recorder = nullptr;
inline thread_local Scope *current = nullptr;
class Scope {
  using Clock = std::chrono::steady_clock;
  Recorder *target_;
  Scope *parent_;
  Phase phase_;
  Clock::time_point start_;
  std::uint64_t children_ = 0;

public:
  explicit Scope(Phase phase)
      : target_(recorder), parent_(current), phase_(phase) {
    if (target_) {
      start_ = Clock::now();
      current = this;
    }
  }
  Scope(const Scope &) = delete;
  Scope &operator=(const Scope &) = delete;
  ~Scope() { finish(); }
  void finish() noexcept {
    if (!target_)
      return;
    const auto elapsed = static_cast<std::uint64_t>(
        std::chrono::duration_cast<std::chrono::nanoseconds>(Clock::now() -
                                                             start_)
            .count());
    auto &entry = target_->phases[static_cast<std::size_t>(phase_)];
    entry.wall_ns += elapsed - children_;
    ++entry.calls;
    if (parent_)
      parent_->children_ += elapsed;
    current = parent_;
    target_ = nullptr;
  }
};
// Caller owns recorder lifetime; bindings are stack-scoped and per-thread.
class Binding {
  Recorder *previous_;

public:
  explicit Binding(Recorder &value) : previous_(recorder) { recorder = &value; }
  Binding(const Binding &) = delete;
  Binding &operator=(const Binding &) = delete;
  ~Binding() { recorder = previous_; }
};
} // namespace symphony::sbv::detail::measurement
