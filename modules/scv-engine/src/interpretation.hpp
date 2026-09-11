#pragma once
#include "symphony/knowledge/engine/protocol.hpp"
#include <string>
namespace symphony::knowledge::scv {
// Pure, bounded authored-profile preparation, extraction and caller-selected checks.
// Retained interpretation/evaluation wrappers are replayed before consumption.
[[nodiscard]] engine::Json handle_interpretation(const engine::Request& request, const std::string& domain);
// Internal composition helper. graph_assessment must have been built/replayed
// inside this native request; this is not an operation accepting external findings.
[[nodiscard]] engine::Json evaluate_connections(const engine::Json& connections, const engine::Json& graph_assessment);
}
