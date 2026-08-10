#include "symphony/knowledge/engine/temporal.hpp"

#include <array>
#include <cstddef>

namespace symphony::knowledge::engine {
namespace {

bool digits(std::string_view value, std::size_t begin, std::size_t count) noexcept {
    if (begin > value.size() || count > value.size() - begin) return false;
    for (std::size_t index = begin; index < begin + count; ++index) {
        if (value[index] < '0' || value[index] > '9') return false;
    }
    return true;
}

unsigned decimal(std::string_view value, std::size_t begin, std::size_t count) noexcept {
    unsigned result = 0U;
    for (std::size_t index = begin; index < begin + count; ++index) {
        result = result * 10U + static_cast<unsigned>(value[index] - '0');
    }
    return result;
}

bool valid_date_fields(std::string_view value) noexcept {
    const auto year = decimal(value, 0U, 4U);
    const auto month = decimal(value, 5U, 2U);
    const auto day = decimal(value, 8U, 2U);
    if (year == 0U || month == 0U || month > 12U || day == 0U) return false;

    constexpr std::array<unsigned, 12U> days_per_month{
        31U, 28U, 31U, 30U, 31U, 30U, 31U, 31U, 30U, 31U, 30U, 31U,
    };
    auto maximum_day = days_per_month[month - 1U];
    const bool leap_year = year % 4U == 0U && (year % 100U != 0U || year % 400U == 0U);
    if (month == 2U && leap_year) maximum_day = 29U;
    return day <= maximum_day;
}

bool valid_time_fields(std::string_view value) noexcept {
    return decimal(value, 11U, 2U) <= 23U &&
           decimal(value, 14U, 2U) <= 59U &&
           decimal(value, 17U, 2U) <= 59U;
}

bool timestamp_prefix(std::string_view value) noexcept {
    return value.size() >= 19U && value[4] == '-' && value[7] == '-' && value[10] == 'T' &&
           value[13] == ':' && value[16] == ':' && digits(value, 0U, 4U) &&
           digits(value, 5U, 2U) && digits(value, 8U, 2U) && digits(value, 11U, 2U) &&
           digits(value, 14U, 2U) && digits(value, 17U, 2U) && valid_date_fields(value) &&
           valid_time_fields(value);
}

} // namespace

bool is_civil_date(std::string_view value) noexcept {
    return value.size() == 10U && value[4] == '-' && value[7] == '-' && digits(value, 0U, 4U) &&
           digits(value, 5U, 2U) && digits(value, 8U, 2U) && valid_date_fields(value);
}

bool is_utc_seconds(std::string_view value) noexcept {
    return value.size() == 20U && value[19] == 'Z' && timestamp_prefix(value);
}

bool is_utc_nanoseconds(std::string_view value) noexcept {
    return value.size() == 30U && value[19] == '.' && digits(value, 20U, 9U) && value[29] == 'Z' &&
           timestamp_prefix(value);
}

} // namespace symphony::knowledge::engine
