#pragma once
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::shv::duckdb_connector {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
inline constexpr const char *engine_id = "symphony-shv-graph-duckdb-connector";
inline constexpr const char *version = "0.1.0-dev";
Json descriptor();
Json handle(const engine::Request &request);
} // namespace symphony::shv::duckdb_connector
