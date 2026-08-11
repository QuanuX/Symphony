#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <string>

namespace symphony::maestro {
namespace engine = symphony::knowledge::engine;

inline constexpr const char* engine_id = "symphony-maestro";
inline constexpr const char* engine_version = "0.1.0-dev";

[[nodiscard]] engine::Json descriptor(const std::string& receptor_id);
[[nodiscard]] engine::Json handle_request(const engine::Request& request);

} // namespace symphony::maestro
