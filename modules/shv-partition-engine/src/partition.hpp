#pragma once
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::knowledge::shv_partition {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
inline constexpr auto version = "0.2.0-dev";
inline constexpr auto engine_id = "symphony-shv-partition";
Json descriptor();
Json handle_request(const engine::Request &);
} // namespace symphony::knowledge::shv_partition
