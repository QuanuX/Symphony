#pragma once

#include <charconv>
#include <string_view>
#include <utility>
#include <fcntl.h>
#include <unistd.h>

namespace node_keyring_cli {
class File {
public:
    explicit File(int fd = -1) : fd_(fd) {}
    File(const File&) = delete;
    File& operator=(const File&) = delete;
    File(File&& other) noexcept : fd_(std::exchange(other.fd_, -1)) {}
    ~File() { reset(); }
    int get() const { return fd_; }
    explicit operator bool() const { return fd_ >= 0; }
    void reset(int next = -1) {
        if (fd_ >= 0) close(fd_);
        fd_ = next;
    }
private:
    int fd_;
};

inline bool uuid(std::string_view s) {
    if (s.size() != 36) return false;
    for (std::size_t i = 0; i < s.size(); ++i) {
        const char c = s[i];
        if (i == 8 || i == 13 || i == 18 || i == 23) {
            if (c != '-') return false;
        } else if (!(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f')) return false;
    }
    return true;
}

inline bool decimal(std::string_view s, unsigned& n) {
    if (s.empty()) return false;
    const auto [p, e] = std::from_chars(s.data(), s.data() + s.size(), n);
    return e == std::errc{} && p == s.data() + s.size();
}
}
