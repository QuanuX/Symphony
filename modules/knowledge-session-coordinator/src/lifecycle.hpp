#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <cstdint>
#include <string>

namespace symphony::knowledge::session {

[[nodiscard]] engine::Json lifecycle_capabilities();
[[nodiscard]] engine::Json handle_lifecycle_plan(const engine::Request& request);
[[nodiscard]] engine::Json build_lifecycle_plan(
    const engine::Json& payload,
    std::int64_t deadline_unix_ms);
[[nodiscard]] std::string lifecycle_stable_inventory_digest(
    const engine::Json& observation,
    std::int64_t deadline_unix_ms);

}
