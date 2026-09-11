#pragma once
#include "symphony/knowledge/engine/protocol.hpp"
#include <string>
namespace symphony::knowledge::scv {
[[nodiscard]] engine::Json handle_pack(const engine::Request&, const std::string& domain);
// Recompute a retained wrapper, including selected conformance cases, before
// making its ordinary knowledge available to another native operation.
[[nodiscard]] engine::Json replay_pack(const engine::Json&, const std::string& domain, const engine::Request&);
}
