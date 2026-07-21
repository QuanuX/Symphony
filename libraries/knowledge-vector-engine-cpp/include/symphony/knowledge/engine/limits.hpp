#pragma once

#include <cstddef>
#include <cstdint>

namespace symphony::knowledge::engine {

struct Limits final {
    static constexpr std::size_t max_request_bytes = 1U << 20;
    static constexpr std::size_t max_response_bytes = 4U << 20;
    static constexpr std::size_t max_json_depth = 64;
    static constexpr std::size_t max_json_values = 16384;
    static constexpr std::size_t max_string_bytes = 65536;
    static constexpr std::size_t max_token_bytes = 128;
    static constexpr std::size_t max_operation_bytes = 64;
    static constexpr std::size_t max_path_bytes = 4096;
    static constexpr std::size_t max_snapshot_files = 1024;
    static constexpr std::size_t max_snapshot_file_bytes = 4U << 20;
    static constexpr std::int64_t max_deadline_ahead_ms = 300000;
};

}
