#include "symphony/knowledge/engine/path.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"

#include <algorithm>
#include <cerrno>
#include <chrono>
#include <cstring>
#include <fcntl.h>
#include <set>
#include <sstream>
#include <sys/stat.h>
#include <unistd.h>
#include <utility>

namespace symphony::knowledge::engine {
namespace {

class FileDescriptor final {
public:
    explicit FileDescriptor(int value = -1) : value_(value) {}
    ~FileDescriptor() {
        if (value_ >= 0) {
            ::close(value_);
        }
    }
    FileDescriptor(const FileDescriptor&) = delete;
    FileDescriptor& operator=(const FileDescriptor&) = delete;
    FileDescriptor(FileDescriptor&& other) noexcept : value_(std::exchange(other.value_, -1)) {}
    FileDescriptor& operator=(FileDescriptor&& other) noexcept {
        if (this != &other) {
            if (value_ >= 0) {
                ::close(value_);
            }
            value_ = std::exchange(other.value_, -1);
        }
        return *this;
    }
    [[nodiscard]] int get() const noexcept { return value_; }

private:
    int value_;
};

[[noreturn]] void throw_path_error(const std::string& code, const std::string& path) {
    const int saved_errno = errno;
    throw Error(code, path + ": " + std::strerror(saved_errno), 5);
}

std::vector<std::string> path_components(const std::string& path) {
    std::vector<std::string> components;
    std::size_t begin = 0;
    while (begin < path.size()) {
        const auto end = path.find('/', begin);
        components.push_back(path.substr(begin, end == std::string::npos ? std::string::npos : end - begin));
        if (end == std::string::npos) {
            break;
        }
        begin = end + 1U;
    }
    return components;
}

FileDescriptor open_root_no_follow(const std::filesystem::path& root) {
    if (!root.is_absolute()) {
        throw Error("path.root_not_absolute", "snapshot root must be an absolute internal path", 5);
    }
    const int slash_fd = ::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    if (slash_fd < 0) {
        throw_path_error("path.root_unreadable", "/");
    }
    FileDescriptor current(slash_fd);
    for (const auto& component_value : root.relative_path()) {
        const auto component = component_value.string();
        if (component.empty() || component == ".") {
            continue;
        }
        if (component == "..") {
            throw Error("path.root_unsafe", "snapshot root contains traversal", 5);
        }
        const int next = ::openat(
            current.get(), component.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0) {
            throw_path_error("path.root_unsafe", root.string());
        }
        current = FileDescriptor(next);
    }
    return current;
}

FileDescriptor open_regular_no_follow(const std::filesystem::path& root, const std::string& relative_path) {
    FileDescriptor current = open_root_no_follow(root);
    const auto components = path_components(relative_path);
    for (std::size_t index = 0; index + 1U < components.size(); ++index) {
        const int next = ::openat(
            current.get(),
            components[index].c_str(),
            O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
        if (next < 0) {
            throw_path_error("path.component_unsafe", relative_path);
        }
        current = FileDescriptor(next);
    }
    const int file_fd = ::openat(
        current.get(), components.back().c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC);
    if (file_fd < 0) {
        throw_path_error("path.file_unreadable", relative_path);
    }
    FileDescriptor file(file_fd);
    struct stat status {};
    if (::fstat(file.get(), &status) != 0) {
        throw_path_error("path.file_stat_failed", relative_path);
    }
    if (!S_ISREG(status.st_mode)) {
        throw Error("path.not_regular", relative_path + ": expected a regular file", 5);
    }
    return file;
}

std::int64_t current_unix_time_ms() {
    const auto now = std::chrono::system_clock::now().time_since_epoch();
    return std::chrono::duration_cast<std::chrono::milliseconds>(now).count();
}

}

bool is_safe_relative_path(const std::string& path) {
    if (path.empty() || path.size() > Limits::max_path_bytes || path.front() == '/' || path.back() == '/') {
        return false;
    }
    if (path.find('\\') != std::string::npos || path.find('\0') != std::string::npos) {
        return false;
    }
    for (const unsigned char character : path) {
        if (character < 0x20U || character == 0x7fU) {
            return false;
        }
    }
    const auto components = path_components(path);
    return !components.empty() && std::all_of(components.begin(), components.end(), [](const std::string& component) {
        return !component.empty() && component != "." && component != "..";
    });
}

std::string read_regular_file_no_follow(
    const std::filesystem::path& root,
    const std::string& relative_path,
    std::size_t max_bytes,
    std::int64_t deadline_unix_ms) {
    if (!is_safe_relative_path(relative_path)) {
        throw Error("path.unsafe", relative_path + ": unsafe relative path", 5);
    }
    auto file = open_regular_no_follow(root, relative_path);
    std::string contents;
    contents.reserve(std::min<std::size_t>(max_bytes, 65536U));
    char buffer[16384];
    for (;;) {
        if (current_unix_time_ms() >= deadline_unix_ms) {
            throw Error("request.deadline_expired", "request deadline expired during file read", 3);
        }
        const auto count = ::read(file.get(), buffer, sizeof(buffer));
        if (count < 0) {
            if (errno == EINTR) {
                continue;
            }
            throw_path_error("path.file_read_failed", relative_path);
        }
        if (count == 0) {
            break;
        }
        if (contents.size() + static_cast<std::size_t>(count) > max_bytes) {
            throw Error("path.file_too_large", relative_path + ": file exceeds byte limit", 5);
        }
        contents.append(buffer, static_cast<std::size_t>(count));
    }
    if (current_unix_time_ms() >= deadline_unix_ms) {
        throw Error("request.deadline_expired", "request deadline expired during file read", 3);
    }
    return contents;
}

Snapshot snapshot_files(
    const std::filesystem::path& root,
    const std::vector<std::string>& relative_paths,
    std::int64_t deadline_unix_ms) {
    if (relative_paths.size() > Limits::max_snapshot_files) {
        throw Error("snapshot.too_many_files", "snapshot file-count limit exceeded", 5);
    }
    std::set<std::string> unique_paths;
    for (const auto& path : relative_paths) {
        if (!is_safe_relative_path(path)) {
            throw Error("path.unsafe", path + ": unsafe relative path", 5);
        }
        if (!unique_paths.insert(path).second) {
            throw Error("snapshot.duplicate_path", path + ": duplicate snapshot path", 5);
        }
    }

    Snapshot snapshot;
    std::ostringstream canonical;
    for (const auto& path : unique_paths) {
        if (current_unix_time_ms() >= deadline_unix_ms) {
            throw Error("request.deadline_expired", "request deadline expired during snapshot", 3);
        }
        const auto contents = read_regular_file_no_follow(
            root, path, Limits::max_snapshot_file_bytes, deadline_unix_ms);
        const auto digest = tagged_sha256(contents);
        const auto size = static_cast<std::uint64_t>(contents.size());
        snapshot.files.push_back(FileDigest{path, size, digest});
        canonical << path.size() << ':' << path << ':' << size << ':' << digest << '\n';
    }
    if (current_unix_time_ms() >= deadline_unix_ms) {
        throw Error("request.deadline_expired", "request deadline expired during snapshot", 3);
    }
    snapshot.digest = tagged_sha256(canonical.str());
    return snapshot;
}

}
