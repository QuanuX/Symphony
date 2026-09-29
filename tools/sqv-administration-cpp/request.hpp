#pragma once
#include "common.hpp"
#include "interface.generated.hpp"
namespace SQV_NAMESPACE {
using Json = symphony::knowledge::engine::Json;
inline constexpr auto engine_id = SQV_ENGINE;
inline constexpr auto module_id = SQV_MODULE;
inline constexpr auto vector_id = SQV_VECTOR;
inline constexpr std::size_t payload_bytes = 262144;
inline constexpr std::size_t process_bytes = 524288;
Json administer(std::string_view operation, const Json &,
                std::int64_t deadline_ms);
Json descriptor();
Json handle_request(const symphony::knowledge::engine::Request &);
} // namespace SQV_NAMESPACE
