#pragma once
#include <symphony/snv/common.hpp>
namespace symphony::snv::sniv {
inline constexpr char engine_id[] = "symphony-sniv";
inline constexpr char version[] = "0.1.0-dev";
[[nodiscard]] std::vector<Operation> operations();
[[nodiscard]] Json handle(std::string_view operation, const Json &payload,
                          std::int64_t deadline_unix_ms);
} // namespace symphony::snv::sniv
