#pragma once
#include "symphony/graph/adapter.hpp"
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::graph {
inline constexpr const char* engine_id="symphony-shv-graph-adapter";
inline constexpr const char* version="0.1.0-dev";
Json descriptor();
Json handle(const knowledge::engine::Request& request);
}
