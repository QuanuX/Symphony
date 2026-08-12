#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::session {

[[nodiscard]] engine::Json ssfv_maintenance_capabilities();
[[nodiscard]] engine::Json handle_ssfv_maintenance(const engine::Request& request);

}
