#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::session {

[[nodiscard]] engine::Json lifecycle_journal_capabilities();
[[nodiscard]] engine::Json handle_lifecycle_journal(const engine::Request& request);

}
