#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::sclv {

namespace engine = symphony::knowledge::engine;

inline constexpr const char* module_id = "sclv-engine";
inline constexpr const char* engine_id = "symphony-sclv";
inline constexpr const char* engine_version = "0.1.0-dev";
inline constexpr const char* vector_id = "sclv";

[[nodiscard]] engine::Json descriptor();
[[nodiscard]] engine::Json handle_request(const engine::Request& request);

}
