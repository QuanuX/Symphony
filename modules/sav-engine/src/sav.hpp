#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::sav {

inline constexpr const char* module_id = "sav-engine";
inline constexpr const char* engine_id = "symphony-sav";
inline constexpr const char* vector_id = "sav";
inline constexpr const char* engine_version = "0.1.0-dev";

[[nodiscard]] engine::Json descriptor();
[[nodiscard]] engine::Json handle_request(const engine::Request& request);

}
