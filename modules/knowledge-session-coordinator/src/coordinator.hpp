#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::session {

inline constexpr const char* module_id = "knowledge-session-coordinator";
inline constexpr const char* engine_id = "symphony-knowledge-session";
inline constexpr const char* engine_version = "0.1.0-dev";

[[nodiscard]] engine::Json descriptor();
[[nodiscard]] engine::Json handle_request(const engine::Request& request);

}
