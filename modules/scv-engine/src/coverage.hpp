#pragma once

#include "symphony/knowledge/engine/protocol.hpp"
#include <string>

namespace symphony::knowledge::scv {
// Explicit declared-source coverage, selected corpus metadata and independently
// replayed profile extraction attempts. No source/head selection or provider action.
[[nodiscard]] engine::Json handle_coverage(const engine::Request& request, const std::string& domain);
}
