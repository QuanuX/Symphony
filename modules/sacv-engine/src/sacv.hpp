#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::sacv {

inline constexpr const char* module_id = "sacv-engine";
inline constexpr const char* engine_id = "symphony-sacv";
inline constexpr const char* vector_id = "sacv";
inline constexpr const char* engine_version = "0.1.0-dev";

[[nodiscard]] engine::Json descriptor();
[[nodiscard]] engine::Json handle_request(const engine::Request& request);

}
