#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::skvi {

inline constexpr const char* module_id = "skvi-engine";
inline constexpr const char* engine_id = "symphony-skvi";
inline constexpr const char* vector_id = "skvi";
inline constexpr const char* engine_version = "0.2.0-dev";
inline constexpr std::size_t max_json_values = 65536;

[[nodiscard]] engine::Json descriptor();
[[nodiscard]] engine::Json handle_request(const engine::Request& request);

}
