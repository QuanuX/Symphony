#pragma once

#include <symphony/knowledge/engine/json.hpp>
#include <cstdint>
#include <functional>
#include <map>
#include <optional>
#include <string>
#include <string_view>

namespace symphony::sbv::detail {

struct ExactIdInsert {
  bool inserted;
  std::uint64_t ordinal;
};

// Exact byte keys; financial ID admission remains the domain caller's job.
// Single-caller memory backend. No hidden aggregate cap and no disk spill.
// Ordinals are first-successful-insertion order and remain stable on duplicate.
class ExactIdIndex {
public:
  explicit ExactIdIndex(
      std::optional<std::uint64_t> max_tracked_bytes = {},
      std::function<void()> checkpoint = {});
  ExactIdIndex(const ExactIdIndex &) = delete;
  ExactIdIndex &operator=(const ExactIdIndex &) = delete;
  ExactIdInsert insert(std::string_view);
  std::optional<std::uint64_t> find(std::string_view) const;
  std::uint64_t size() const noexcept;
  std::uint64_t retained_key_bytes() const noexcept;
  std::uint64_t tracked_bytes() const noexcept;
  knowledge::engine::Json evidence() const;

private:
  std::map<std::string, std::uint64_t, std::less<>> entries_;
  std::optional<std::uint64_t> max_tracked_bytes_;
  std::function<void()> checkpoint_;
  std::uint64_t count_ = 0, key_bytes_ = 0, tracked_bytes_ = 0;
};
} // namespace symphony::sbv::detail
