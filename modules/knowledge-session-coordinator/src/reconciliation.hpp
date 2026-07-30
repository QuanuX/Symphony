#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::session {

[[nodiscard]] engine::Json reconciliation_capabilities();
[[nodiscard]] engine::Json handle_reconciliation(const engine::Request& request);

}
