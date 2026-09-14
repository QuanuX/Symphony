#include "commit_barrier.hpp"
#include <cerrno>
#include <charconv>
#include <csignal>
#include <cstdlib>
#include <cstring>
#include <string>
#include <sys/stat.h>
#include <unistd.h>

namespace symphony::shv::duckdb_connector::test_support {
void commit_barrier(const char* point) {
    const auto selected = std::getenv("SYMPHONY_SHV_TEST_BARRIER");
    if (selected == nullptr || std::strcmp(selected, point) != 0) return;
    const auto descriptor = std::getenv("SYMPHONY_SHV_TEST_BARRIER_FD");
    if (descriptor == nullptr) ::_exit(90);
    int fd = -1;
    const auto end = descriptor + std::strlen(descriptor);
    const auto parsed = std::from_chars(descriptor, end, fd);
    struct stat st{};
    if (parsed.ec != std::errc{} || parsed.ptr != end || fd < 3 ||
        ::fstat(fd, &st) != 0 || !S_ISFIFO(st.st_mode)) ::_exit(91);
    const auto marker = std::string(point) + "\n";
    std::size_t sent = 0;
    while (sent < marker.size()) {
        const auto size = ::write(fd, marker.data() + sent, marker.size() - sent);
        if (size < 0 && errno == EINTR) continue;
        if (size <= 0) ::_exit(92);
        sent += static_cast<std::size_t>(size);
    }
    // No return into the transaction if the test parent unexpectedly resumes us.
    if (::raise(SIGSTOP) != 0) ::_exit(93);
    ::_exit(94);
}
}
