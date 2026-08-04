#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::session {

[[nodiscard]] engine::Json authority_session_capabilities();
[[nodiscard]] engine::Json handle_authority_session(const engine::Request& request);

}
