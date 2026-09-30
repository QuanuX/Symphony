#pragma once
#include <symphony/snv/common.hpp>
#include <cstdint>
#include <string_view>
#include <vector>

namespace symphony::snv::scnv {
inline constexpr char engine_id[] = "symphony-scnv";
inline constexpr char version[] = "0.1.0-dev";
std::vector<symphony::snv::Operation> operations();
Json handle(std::string_view operation, const Json& payload,
            std::int64_t deadline_unix_ms);
}
