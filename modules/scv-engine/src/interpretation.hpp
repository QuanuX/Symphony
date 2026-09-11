#pragma once
#include "symphony/knowledge/engine/protocol.hpp"
#include <string>
namespace symphony::knowledge::scv {
// Pure, bounded profile extraction and caller-selected connection checks.
// Retained interpretation/evaluation wrappers are replayed before consumption.
[[nodiscard]] engine::Json handle_interpretation(const engine::Request& request, const std::string& domain);
}
