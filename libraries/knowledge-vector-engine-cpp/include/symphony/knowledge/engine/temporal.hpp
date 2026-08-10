#pragma once

#include <string_view>

namespace symphony::knowledge::engine {

// Validates the canonical STSC proleptic-Gregorian civil-date profile.
[[nodiscard]] bool is_civil_date(std::string_view value) noexcept;

// Validates the canonical STSC whole-second UTC profile.
[[nodiscard]] bool is_utc_seconds(std::string_view value) noexcept;

// Validates the canonical STSC exact-nine-digit nanosecond UTC profile.
[[nodiscard]] bool is_utc_nanoseconds(std::string_view value) noexcept;

} // namespace symphony::knowledge::engine
