#pragma once
#include "interface.generated.hpp"
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::knowledge::shv_publication {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
inline constexpr auto engine_id = "symphony-shv-publication";
Json descriptor();
Json handle_request(const engine::Request &);
} // namespace symphony::knowledge::shv_publication
