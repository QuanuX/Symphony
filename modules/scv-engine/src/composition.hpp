#pragma once
#include "symphony/knowledge/engine/protocol.hpp"
#include <string>
namespace symphony::knowledge::scv {
// Pure finite recipe exploration, exact reassessment, precise obligation inventory
// and reference-only follow-up. Caller declarations and matched interface IDs
// never establish deployment readiness or causation by a submitted reference.
[[nodiscard]] engine::Json handle_composition(const engine::Request& request, const std::string& domain);
}
