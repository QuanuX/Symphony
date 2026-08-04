#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::session {

[[nodiscard]] engine::Json lifecycle_capabilities();
[[nodiscard]] engine::Json handle_lifecycle_plan(const engine::Request& request);

}
