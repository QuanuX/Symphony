#pragma once

#include "symphony/knowledge/engine/protocol.hpp"

#include <string>

namespace symphony::knowledge::scv {
// Immutable evidence operations; none select a mutable source, corpus or graph head.
// Indexes must additionally be regenerated against retained captures by the store.
// A supplied predecessor proves immediate continuity; complete retained chains are
// checked by the adapter. Historical source revisions remain explicit inputs.
void validate_capture_index(const engine::Json& index, const std::string& domain);
void validate_corpus(const engine::Json& corpus, const std::string& domain);
[[nodiscard]] engine::Json handle_corpus(const engine::Request& request, const std::string& domain);
}
