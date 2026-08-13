#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <cstddef>

namespace symphony::knowledge::ssfv {

inline constexpr const char* module_id = "ssfv-engine";
inline constexpr const char* engine_id = "symphony-ssfv";
inline constexpr const char* vector_id = "ssfv";
inline constexpr const char* engine_version = "0.1.0-dev";
inline constexpr std::size_t max_process_json_values = 65536U;

[[nodiscard]] engine::Json descriptor();
[[nodiscard]] engine::Json descriptor_v2();
[[nodiscard]] engine::Json handle_request(const engine::Request& request);

}
