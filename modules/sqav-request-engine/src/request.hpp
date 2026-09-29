#pragma once
#include "interface.generated.hpp"
#include "symphony/knowledge/engine/protocol.hpp"
namespace symphony::sqav::request {
using Json = engine::Json;
inline constexpr auto engine_id = "symphony-sqav-request";
inline constexpr auto input_protocol = "symphony.sqav.request-validation-input.v1";
inline constexpr auto result_protocol = "symphony.sqav.request-validation.v1";
inline constexpr std::size_t payload_bytes = 32768;
inline constexpr std::size_t process_bytes = 65536;
Json validate(const Json &);
Json descriptor();
Json handle_request(const engine::Request &);
}
