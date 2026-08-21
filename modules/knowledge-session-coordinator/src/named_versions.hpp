#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::session {

[[nodiscard]] engine::Json named_version_capabilities();
[[nodiscard]] engine::Json handle_named_version(const engine::Request& request);

}
