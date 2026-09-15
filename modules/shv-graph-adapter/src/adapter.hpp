#pragma once
#include "interface.generated.hpp"
#include "symphony/graph/adapter.hpp"
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::graph {
inline constexpr const char* engine_id="symphony-shv-graph-adapter";
Json descriptor();
Json handle(const knowledge::engine::Request& request);
}
