#include "symphony/knowledge/engine/protocol.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"

#include <algorithm>
#include <chrono>
#include <set>
#include <sstream>
#include <string_view>
#include <vector>

namespace symphony::knowledge::engine {
namespace {

bool is_safe_token(std::string_view value, std::size_t max_bytes) {
    if (value.empty() || value.size() > max_bytes) {
        return false;
    }
    return std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        const bool ascii_alphanumeric =
            (character >= 'a' && character <= 'z') ||
            (character >= 'A' && character <= 'Z') ||
            (character >= '0' && character <= '9');
        return ascii_alphanumeric || character == '.' || character == '_' ||
               character == ':' || character == '-';
    });
}

std::string safe_message(std::string_view value) {
    std::string result;
    result.reserve(std::min<std::size_t>(value.size(), 512U));
    for (const unsigned char character : value) {
        if (result.size() == 512U) {
            break;
        }
        if (character >= 0x20U && character != 0x7fU) {
            result.push_back(static_cast<char>(character));
        }
    }
    return result.empty() ? "request rejected" : result;
}

std::string require_string(const Json& object, const char* field, std::size_t max_bytes) {
    const auto& value = object.at(field);
    if (!value.is_string()) {
        throw Error("request.invalid_field", std::string(field) + " must be a string", 2);
    }
    const auto text = value.get<std::string>();
    if (!is_safe_token(text, max_bytes)) {
        throw Error("request.invalid_field", std::string(field) + " has invalid token syntax", 2);
    }
    return text;
}

}

std::int64_t unix_time_ms() {
    const auto now = std::chrono::system_clock::now().time_since_epoch();
    return std::chrono::duration_cast<std::chrono::milliseconds>(now).count();
}

std::string read_bounded(std::istream& input, std::size_t max_bytes) {
    std::string contents;
    contents.reserve(std::min<std::size_t>(max_bytes, 65536U));
    char buffer[16384];
    while (input.good()) {
        input.read(buffer, static_cast<std::streamsize>(sizeof(buffer)));
        const auto count = input.gcount();
        if (count > 0) {
            if (contents.size() + static_cast<std::size_t>(count) > max_bytes) {
                throw Error("input.too_large", "request exceeds byte limit", 2);
            }
            contents.append(buffer, static_cast<std::size_t>(count));
        }
    }
    if (input.bad()) {
        throw Error("input.read_failed", "request stream could not be read", 2);
    }
    if (contents.empty()) {
        throw Error("input.empty", "request is empty", 2);
    }
    return contents;
}

Json parse_bounded_json(const std::string& input, std::size_t max_bytes) {
    if (input.empty()) {
        throw Error("input.empty", "request is empty", 2);
    }
    if (input.size() > max_bytes) {
        throw Error("input.too_large", "request exceeds byte limit", 2);
    }

    std::vector<std::set<std::string>> object_keys;
    std::size_t value_count = 0;
    const auto callback = [&](int depth, Json::parse_event_t event, Json& parsed) {
        if (depth < 0 || static_cast<std::size_t>(depth) > Limits::max_json_depth) {
            throw Error("json.depth_exceeded", "JSON nesting depth limit exceeded", 2);
        }
        if (event == Json::parse_event_t::object_start ||
            event == Json::parse_event_t::array_start ||
            event == Json::parse_event_t::key ||
            event == Json::parse_event_t::value) {
            ++value_count;
            if (value_count > Limits::max_json_values) {
                throw Error("json.value_count_exceeded", "JSON value-count limit exceeded", 2);
            }
        }
        if (event == Json::parse_event_t::object_start) {
            object_keys.emplace_back();
        } else if (event == Json::parse_event_t::object_end) {
            if (object_keys.empty()) {
                throw Error("json.invalid", "JSON object stack is invalid", 2);
            }
            object_keys.pop_back();
        } else if (event == Json::parse_event_t::key) {
            const auto key = parsed.get<std::string>();
            if (key.size() > Limits::max_string_bytes) {
                throw Error("json.string_too_large", "JSON key exceeds byte limit", 2);
            }
            if (object_keys.empty()) {
                throw Error("json.invalid", "JSON key is outside an object", 2);
            }
            auto& keys = object_keys.back();
            if (!keys.insert(key).second) {
                throw Error("json.duplicate_key", "JSON object contains a duplicate key", 2);
            }
        } else if (event == Json::parse_event_t::value) {
            if (parsed.is_string() && parsed.get_ref<const std::string&>().size() > Limits::max_string_bytes) {
                throw Error("json.string_too_large", "JSON string exceeds byte limit", 2);
            }
            if (parsed.is_number_float()) {
                throw Error("json.float_prohibited", "floating-point JSON values are prohibited", 2);
            }
            if (parsed.is_number_unsigned() && parsed.get<std::uint64_t>() > 9007199254740991ULL) {
                throw Error("json.integer_out_of_range", "JSON integer exceeds the interoperable range", 2);
            }
            if (parsed.is_number_integer() && !parsed.is_number_unsigned()) {
                const auto integer = parsed.get<std::int64_t>();
                if (integer < -9007199254740991LL || integer > 9007199254740991LL) {
                    throw Error("json.integer_out_of_range", "JSON integer exceeds the interoperable range", 2);
                }
            }
        }
        return true;
    };

    try {
        return Json::parse(input, callback, true, false);
    } catch (const Error&) {
        throw;
    } catch (const nlohmann::json::exception&) {
        throw Error("json.invalid", "request is not valid bounded UTF-8 JSON", 2);
    }
}

Request parse_request(
    const std::string& input,
    const std::string& expected_engine,
    std::int64_t now_unix_ms) {
    const auto document = parse_bounded_json(input, Limits::max_request_bytes);
    if (!document.is_object()) {
        throw Error("request.not_object", "request must be a JSON object", 2);
    }
    static const std::set<std::string> fields = {
        "protocol", "request_id", "correlation_id", "operation",
        "target_engine", "deadline_unix_ms", "payload",
    };
    if (document.size() != fields.size()) {
        throw Error("request.field_set", "request field set is incomplete or contains unknown fields", 2);
    }
    for (const auto& [key, value] : document.items()) {
        static_cast<void>(value);
        if (!fields.contains(key)) {
            throw Error("request.unknown_field", "request contains an unknown field", 2);
        }
    }

    if (!document.at("protocol").is_string() ||
        document.at("protocol").get<std::string>() != process_protocol_v1) {
        throw Error("protocol.unsupported", "unsupported process protocol", 3);
    }

    Request request;
    request.request_id = require_string(document, "request_id", Limits::max_token_bytes);
    request.correlation_id = require_string(document, "correlation_id", Limits::max_token_bytes);
    request.operation = require_string(document, "operation", Limits::max_operation_bytes);
    request.target_engine = require_string(document, "target_engine", Limits::max_token_bytes);
    if (request.target_engine != expected_engine) {
        throw Error("engine.target_mismatch", "request targets a different engine", 3);
    }

    const auto& deadline = document.at("deadline_unix_ms");
    if ((!deadline.is_number_integer() && !deadline.is_number_unsigned()) ||
        (deadline.is_number_unsigned() && deadline.get<std::uint64_t>() > 9007199254740991ULL)) {
        throw Error("request.invalid_deadline", "deadline_unix_ms must be a safe integer", 2);
    }
    try {
        request.deadline_unix_ms = deadline.get<std::int64_t>();
    } catch (const nlohmann::json::exception&) {
        throw Error("request.invalid_deadline", "deadline_unix_ms is outside the supported range", 2);
    }
    if (request.deadline_unix_ms <= now_unix_ms) {
        throw Error("request.deadline_expired", "request deadline has expired", 3);
    }
    if (request.deadline_unix_ms - now_unix_ms > Limits::max_deadline_ahead_ms) {
        throw Error("request.deadline_too_far", "request deadline exceeds the allowed window", 3);
    }

    request.payload = document.at("payload");
    if (!request.payload.is_object()) {
        throw Error("request.invalid_payload", "payload must be a JSON object", 2);
    }
    return request;
}

Json success_response(
    const Request& request,
    const std::string& engine_id,
    const std::string& engine_version,
    Json result) {
    return Json{
        {"protocol", process_protocol_v1},
        {"request_id", request.request_id},
        {"correlation_id", request.correlation_id},
        {"operation", request.operation},
        {"engine_id", engine_id},
        {"engine_version", engine_version},
        {"outcome", "ok"},
        {"result", std::move(result)},
        {"error", nullptr},
    };
}

Json error_response(
    const std::string& request_id,
    const std::string& correlation_id,
    const std::string& operation,
    const std::string& engine_id,
    const std::string& engine_version,
    const std::string& code,
    const std::string& message) {
    return Json{
        {"protocol", process_protocol_v1},
        {"request_id", is_safe_token(request_id, Limits::max_token_bytes) ? request_id : "unavailable"},
        {"correlation_id", is_safe_token(correlation_id, Limits::max_token_bytes) ? correlation_id : "unavailable"},
        {"operation", is_safe_token(operation, Limits::max_operation_bytes) ? operation : "unavailable"},
        {"engine_id", engine_id},
        {"engine_version", engine_version},
        {"outcome", "error"},
        {"result", nullptr},
        {"error", Json{{"code", code}, {"message", safe_message(message)}}},
    };
}

std::string serialize_response(Json response) {
    if (!response.is_object() || response.contains("response_digest")) {
        throw Error("response.invalid", "response must be an object without a precomputed digest", 5);
    }
    std::string canonical;
    try {
        canonical = response.dump(-1, ' ', false, nlohmann::json::error_handler_t::strict);
        try {
            static_cast<void>(parse_bounded_json(canonical, Limits::max_response_bytes));
        } catch (const Error&) {
            throw Error("response.invalid", "response violates bounded JSON requirements", 5);
        }
        response["response_digest"] = tagged_sha256(canonical);
        canonical = response.dump(-1, ' ', false, nlohmann::json::error_handler_t::strict);
    } catch (const nlohmann::json::exception&) {
        throw Error("response.invalid_utf8", "response contains invalid UTF-8", 5);
    }
    if (canonical.size() + 1U > Limits::max_response_bytes) {
        throw Error("response.too_large", "response exceeds byte limit", 5);
    }
    return canonical + '\n';
}

}
