#pragma once
#include "interface.generated.hpp"
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::knowledge::shv_pdf {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
inline constexpr auto engine_id = "symphony-shv-pdf";
Json descriptor();
Json project_graph(const Json &);
Json parse_table(const std::string &);
Json handle_request(const engine::Request &);
} // namespace symphony::knowledge::shv_pdf
