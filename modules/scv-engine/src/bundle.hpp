#pragma once
#include "symphony/knowledge/engine/protocol.hpp"

namespace symphony::knowledge::scv {
struct DecodedBundle {
    engine::Json value;
    engine::Json metrics;
};
// Closed transport codec only. These helpers never fetch references or establish
// evidence ownership/semantics. The operation below delegates semantic replay to
// the unchanged composition owner after complete bounded reconstruction.
[[nodiscard]] engine::Json encode_bundle(const engine::Json& value, const engine::Request& request);
[[nodiscard]] DecodedBundle decode_bundle(const engine::Json& bundle, const engine::Request& request);
[[nodiscard]] engine::Json handle_bundle(const engine::Request& request, const std::string& domain);
}
