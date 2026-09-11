#pragma once
#include "symphony/knowledge/engine/protocol.hpp"
#include <string>
namespace symphony::knowledge::scv {
// Pure finite Cartesian recipe exploration and exact retained-result reassessment.
// Caller declarations and matched interface IDs never establish deployment readiness.
[[nodiscard]] engine::Json handle_composition(const engine::Request& request, const std::string& domain);
}
