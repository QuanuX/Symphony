#pragma once
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::scv::duckdb_connector {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
inline constexpr const char* engine_id = "symphony-scv-graph-duckdb-connector";
inline constexpr const char* version = "0.1.0-dev";
Json descriptor();
Json handle(const engine::Request& request);
}
