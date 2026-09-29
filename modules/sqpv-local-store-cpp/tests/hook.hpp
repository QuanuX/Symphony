#pragma once
namespace symphony::sqpv::testing {
// Private instrumented test build only; absent from the installed archive.
void crash_at(int checkpoint) noexcept;
void fail_at(int checkpoint) noexcept;
void pause_at(int checkpoint, int notify_fd, int resume_fd) noexcept;
} // namespace symphony::sqpv::testing
