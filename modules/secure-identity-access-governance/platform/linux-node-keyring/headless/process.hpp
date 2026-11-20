#pragma once

#include <algorithm>
#include <array>
#include <cerrno>
#include <charconv>
#include <chrono>
#include <csignal>
#include <cstddef>
#include <filesystem>
#include <fcntl.h>
#include <pwd.h>
#include <string>
#include <string_view>
#include <sys/prctl.h>
#include <sys/resource.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <thread>
#include <unistd.h>
#include <vector>

namespace node_keyring {
namespace fs = std::filesystem;
using Clock = std::chrono::steady_clock;
using namespace std::chrono_literals;

struct Refusal {};
inline void require(bool ok) { if (!ok) throw Refusal{}; }

class Descriptor {
    int fd_;
public:
    explicit Descriptor(int fd) : fd_(fd) { require(fd_ >= 0); }
    Descriptor(const Descriptor&) = delete;
    Descriptor(Descriptor&& other) noexcept : fd_(other.fd_) { other.fd_ = -1; }
    Descriptor& operator=(const Descriptor&) = delete;
    void reset() { if (fd_ >= 0) close(fd_); fd_ = -1; }
    ~Descriptor() { reset(); }
    int get() const { return fd_; }
    struct stat metadata() const {
        struct stat s{};
        require(fstat(fd_, &s) == 0);
        return s;
    }
};

inline uid_t parse_uid(std::string_view text) {
    uid_t uid{};
    auto [end, ec] = std::from_chars(text.data(), text.data() + text.size(), uid);
    require(ec == std::errc{} && end == text.data() + text.size() && uid != 0 &&
            uid != static_cast<uid_t>(-1) && text == std::to_string(uid));
    return uid;
}

inline void protect_process() {
    const rlimit limit{};
    require(setrlimit(RLIMIT_CORE, &limit) == 0 && prctl(PR_SET_DUMPABLE, 0) == 0);
    umask(0077);
}

inline fs::path account_home(uid_t uid) {
    const auto* account = getpwuid(uid);
    require(account && account->pw_dir);
    fs::path home(account->pw_dir);
    require(home.is_absolute() && home != "/" && fs::canonical(home) == home);
    struct stat s{};
    require(lstat(home.c_str(), &s) == 0 && S_ISDIR(s.st_mode) && s.st_uid == uid);
    return home;
}

inline void write_all(int fd, std::string_view bytes) {
    while (!bytes.empty()) {
        const ssize_t n = write(fd, bytes.data(), bytes.size());
        if (n < 0 && errno == EINTR) continue;
        require(n > 0);
        bytes.remove_prefix(static_cast<size_t>(n));
    }
}

// These helpers are single-threaded. Prepare argv/env before fork; children
// inherit only the three requested streams and an explicit environment.
class Child {
    pid_t pid_;
    int status_{};
    static std::vector<char*> pointers(std::vector<std::string>& values) {
        std::vector<char*> v;
        for (auto& value : values) v.push_back(value.data());
        v.push_back(nullptr);
        return v;
    }
public:
    Child(std::vector<std::string> args, std::vector<std::string> env, int input, int output) {
        auto argv = pointers(args);
        auto envp = pointers(env);
        Descriptor devnull(open("/dev/null", O_RDWR | O_CLOEXEC));
        const pid_t parent = getpid();
        pid_ = fork();
        require(pid_ >= 0);
        if (pid_ == 0) {
            if (prctl(PR_SET_PDEATHSIG, SIGTERM) < 0 || getppid() != parent ||
                dup2(input < 0 ? devnull.get() : input, STDIN_FILENO) < 0 ||
                dup2(output < 0 ? devnull.get() : output, STDOUT_FILENO) < 0 ||
                dup2(devnull.get(), STDERR_FILENO) < 0 || close_range(3, ~0U, 0) < 0) _exit(126);
            execve(argv.front(), argv.data(), envp.data());
            _exit(127);
        }
    }
    Child(const Child&) = delete;
    Child& operator=(const Child&) = delete;
    bool running() {
        if (pid_ <= 0) return false;
        pid_t found;
        do { found = waitpid(pid_, &status_, WNOHANG); } while (found < 0 && errno == EINTR);
        if (found == pid_ || (found < 0 && errno == ECHILD)) pid_ = -1;
        require(found >= 0);
        return found == 0;
    }
    bool wait_until(Clock::time_point deadline) {
        while (running()) {
            if (Clock::now() >= deadline) return false;
            std::this_thread::sleep_for(20ms);
        }
        return WIFEXITED(status_) && WEXITSTATUS(status_) == 0;
    }
    ~Child() {
        if (pid_ <= 0) return;
        // The PID remains ours until reaped, even if exec or startup failed.
        kill(pid_, SIGTERM);
        const auto deadline = Clock::now() + 3s;
        while (Clock::now() < deadline) {
            const auto found = waitpid(pid_, &status_, WNOHANG);
            if (found == pid_ || (found < 0 && errno == ECHILD)) return;
            std::this_thread::sleep_for(20ms);
        }
        kill(pid_, SIGKILL);
        while (waitpid(pid_, &status_, 0) < 0 && errno == EINTR) {}
    }
};

class Pipe {
    std::array<int, 2> fds_{-1, -1};
public:
    Pipe() { require(pipe2(fds_.data(), O_CLOEXEC) == 0); }
    Pipe(const Pipe&) = delete;
    Pipe& operator=(const Pipe&) = delete;
    int read_end() const { return fds_[0]; }
    int write_end() const { return fds_[1]; }
    void finish_writing() { if (fds_[1] >= 0) { close(fds_[1]); fds_[1] = -1; } }
    ~Pipe() { close(fds_[0]); finish_writing(); }
};
}
