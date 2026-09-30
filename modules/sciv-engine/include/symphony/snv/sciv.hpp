#pragma once

#include <symphony/snv/common.hpp>

#include <cstdint>
#include <string_view>
#include <vector>

namespace symphony::snv::sciv {

inline constexpr char engine_id[] = "symphony-sciv";
inline constexpr char version[] = "0.1.0-dev";

[[nodiscard]] std::vector<symphony::snv::Operation> operations();
[[nodiscard]] Json handle(std::string_view operation, const Json& payload,
                          std::int64_t deadline_unix_ms);

}
