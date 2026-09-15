#pragma once
#include "interface.generated.hpp"
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::shv::duckdb_connector {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
inline constexpr const char *engine_id = "symphony-shv-graph-duckdb-connector";
Json descriptor();
Json handle(const engine::Request &request);
} // namespace symphony::shv::duckdb_connector
