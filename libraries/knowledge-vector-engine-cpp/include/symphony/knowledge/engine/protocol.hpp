#pragma once

#include "symphony/knowledge/engine/json.hpp"

#include <cstdint>
#include <istream>
#include <string>

namespace symphony::knowledge::engine {

inline constexpr const char* process_protocol_v1 = "symphony.knowledge.engine-process.v1";
inline constexpr const char* descriptor_protocol_v1 = "symphony.knowledge.engine-descriptor.v1";
inline constexpr const char* install_receipt_protocol_v1 = "symphony.knowledge.install-receipt.v1";

struct Request final {
    std::string request_id;
    std::string correlation_id;
    std::string operation;
    std::string target_engine;
    std::int64_t deadline_unix_ms;
    Json payload;
};

[[nodiscard]] std::int64_t unix_time_ms();
[[nodiscard]] std::string read_bounded(std::istream& input, std::size_t max_bytes);
[[nodiscard]] Json parse_bounded_json(const std::string& input, std::size_t max_bytes);
[[nodiscard]] Request parse_request(
    const std::string& input,
    const std::string& expected_engine,
    std::int64_t now_unix_ms);
[[nodiscard]] Json success_response(
    const Request& request,
    const std::string& engine_id,
    const std::string& engine_version,
    Json result);
[[nodiscard]] Json error_response(
    const std::string& request_id,
    const std::string& correlation_id,
    const std::string& operation,
    const std::string& engine_id,
    const std::string& engine_version,
    const std::string& code,
    const std::string& message);
[[nodiscard]] std::string serialize_response(Json response);

}
