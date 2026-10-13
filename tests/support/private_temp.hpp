#pragma once

#include <cstdlib>
#include <filesystem>
#include <stdexcept>
#include <string>

namespace symphony::testing {
// Test fixtures only: resolve the caller's chosen scratch root once into an
// absolute path so macOS /tmp aliases do not enter no-follow storage APIs.
// A short default also accommodates local Unix-domain socket path limits.
inline std::string temporary_root() {
  const auto *selected = std::getenv("SYMPHONY_TEST_TMPDIR");
  return std::filesystem::canonical(selected && *selected ? selected : "/tmp")
      .string();
}
inline std::string temporary_pattern(const std::string &leaf) {
  if (leaf.empty() || leaf.find('/') != std::string::npos ||
      leaf.find('\0') != std::string::npos || !leaf.ends_with("XXXXXX"))
    throw std::invalid_argument("invalid temporary fixture pattern");
  return (std::filesystem::path(temporary_root()) / leaf).string();
}
} // namespace symphony::testing
