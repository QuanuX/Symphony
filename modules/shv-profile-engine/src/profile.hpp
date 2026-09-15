#pragma once
#include "interface.generated.hpp"
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::knowledge::shv_profile {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
inline constexpr auto engine_id = "symphony-shv-profile";
Json descriptor();
Json extraction_diagnose(const engine::Request &);
Json references_analyze(const Json &);
void validate_mappings(const Json &, bool);
void validate_sources(const Json &);
Json handle_request(const engine::Request &);
} // namespace symphony::knowledge::shv_profile
