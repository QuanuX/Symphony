#pragma once

#include <cstdint>
#include <filesystem>
#include <limits>
#include <string>
#include <vector>

namespace symphony::knowledge::engine {

struct FileDigest final {
    std::string path;
    std::uint64_t size;
    std::string digest;
};

struct Snapshot final {
    std::vector<FileDigest> files;
    std::string digest;
};

[[nodiscard]] bool is_safe_relative_path(const std::string& path);
[[nodiscard]] std::string read_regular_file_no_follow(
    const std::filesystem::path& root,
    const std::string& relative_path,
    std::size_t max_bytes,
    std::int64_t deadline_unix_ms = std::numeric_limits<std::int64_t>::max());
[[nodiscard]] Snapshot snapshot_files(
    const std::filesystem::path& root,
    const std::vector<std::string>& relative_paths,
    std::int64_t deadline_unix_ms = std::numeric_limits<std::int64_t>::max());

}
