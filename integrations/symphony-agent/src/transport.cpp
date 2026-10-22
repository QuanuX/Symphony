#include "symphony/transport.hpp"

#if !defined(__APPLE__) && !defined(__linux__)
#error "Symphony local transport currently supports macOS and Linux only"
#endif

#include <algorithm>
#include <array>
#include <cerrno>
#include <chrono>
#include <climits>
#include <csignal>
#include <cstring>
#include <fcntl.h>
#include <mutex>
#include <poll.h>
#include <stdexcept>
#include <sys/random.h>
#include <sys/resource.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/un.h>
#include <sys/wait.h>
#include <unistd.h>
#if defined(__linux__)
#include <sys/syscall.h>
#else
#include <spawn.h>
#endif

namespace symphony {
namespace {

class Fd {
public:
    explicit Fd(int value = -1) : value_(value) {}
    ~Fd() { reset(); }
    Fd(const Fd&) = delete;
    Fd& operator=(const Fd&) = delete;
    Fd(Fd&& other) noexcept : value_(other.release()) {}
    Fd& operator=(Fd&& other) noexcept {
        if (this != &other) reset(other.release());
        return *this;
    }
    int get() const { return value_; }
    int release() { const int result = value_; value_ = -1; return result; }
    void reset(int value = -1) {
        // Retrying close after EINTR can close an unrelated reused descriptor.
        if (value_ >= 0) ::close(value_);
        value_ = value;
    }
private:
    int value_;
};

[[noreturn]] void fail(const char* operation) {
    // Do not include argv, environment values, or child output in exceptions.
    throw std::runtime_error(std::string(operation) + " failed (errno " +
                             std::to_string(errno) + ")");
}

void cloexec(int fd) {
    const int flags = ::fcntl(fd, F_GETFD);
    if (flags < 0 || ::fcntl(fd, F_SETFD, flags | FD_CLOEXEC) < 0) fail("descriptor protection");
}

void nonblocking(int fd) {
    const int flags = ::fcntl(fd, F_GETFL);
    if (flags < 0 || ::fcntl(fd, F_SETFL, flags | O_NONBLOCK) < 0) fail("nonblocking descriptor");
}

std::array<Fd, 2> make_pipe() {
    int descriptors[2];
    if (::pipe(descriptors) != 0) fail("pipe creation");
    std::array<Fd, 2> result{Fd(descriptors[0]), Fd(descriptors[1])};
    cloexec(result[0].get());
    cloexec(result[1].get());
    return result;
}

bool has_nul(const std::string& value) { return value.find('\0') != std::string::npos; }

void absolute(const std::string& value) {
    if (value.empty() || value.front() != '/' || has_nul(value))
        throw std::invalid_argument("an absolute path without NUL bytes is required");
}

void kill_group(pid_t pid) noexcept {
    // A child is retained as waitable until cleanup, preventing PID reuse.
    ::kill(-pid, SIGKILL);
    ::kill(pid, SIGKILL);
}

void terminate_group(pid_t pid) noexcept {
    // qxctl can own subprocesses in independent process groups. Give its
    // signal handler a bounded chance to reconcile/stop those owners before
    // forcing local cleanup. This does not assert cancellation or rollback.
    if (::kill(-pid, SIGTERM) != 0) ::kill(pid, SIGTERM);
    const auto end = std::chrono::steady_clock::now() + std::chrono::milliseconds(500);
    while (std::chrono::steady_clock::now() < end) {
        siginfo_t info{};
        const int rc = ::waitid(P_PID, static_cast<id_t>(pid), &info, WEXITED | WNOHANG | WNOWAIT);
        if (rc == 0 && info.si_pid == pid) break;
        if (rc < 0 && errno != EINTR) break;
        ::poll(nullptr, 0, 5);
    }
    kill_group(pid);
}

void reap(pid_t pid, int& status) noexcept {
    while (::waitpid(pid, &status, 0) < 0 && errno == EINTR) {}
}

struct ChildGuard {
    pid_t pid;
    bool done = false;
    ~ChildGuard() {
        if (!done) {
            terminate_group(pid);
            int ignored = 0;
            reap(pid, ignored);
        }
    }
};

#if defined(__linux__)
void close_child_descriptors(unsigned long limit) noexcept {
#if defined(SYS_close_range)
    if (::syscall(SYS_close_range, 3U, UINT_MAX, 0U) == 0) return;
#endif
    for (unsigned long fd = 3; fd < limit; ++fd) ::close(static_cast<int>(fd));
}
#else
pid_t spawn_mac(const std::string& executable, const std::string& cwd,
                char* const* argv, char* const* environment,
                int input, int output, int error) {
    struct Actions {
        posix_spawn_file_actions_t value;
        Actions() {
            if (const int e = ::posix_spawn_file_actions_init(&value)) { errno = e; fail("spawn actions initialization"); }
        }
        ~Actions() { ::posix_spawn_file_actions_destroy(&value); }
    } actions;
    struct Attributes {
        posix_spawnattr_t value;
        Attributes() {
            if (const int e = ::posix_spawnattr_init(&value)) { errno = e; fail("spawn attributes initialization"); }
        }
        ~Attributes() { ::posix_spawnattr_destroy(&value); }
    } attributes;
    auto check = [](int error_code) {
        if (error_code) { errno = error_code; fail("process spawn setup"); }
    };
    check(::posix_spawnattr_setflags(&attributes.value, POSIX_SPAWN_CLOEXEC_DEFAULT | POSIX_SPAWN_SETPGROUP));
    check(::posix_spawnattr_setpgroup(&attributes.value, 0));
    check(::posix_spawn_file_actions_adddup2(&actions.value, input, STDIN_FILENO));
    check(::posix_spawn_file_actions_adddup2(&actions.value, output, STDOUT_FILENO));
    check(::posix_spawn_file_actions_adddup2(&actions.value, error, STDERR_FILENO));
    // The _np spelling supports macOS 10.15 onward; the non-_np alias is new in
    // macOS 26. Keep deployment compatibility while compiling with its SDK.
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
    check(::posix_spawn_file_actions_addchdir_np(&actions.value, cwd.c_str()));
#pragma clang diagnostic pop
    pid_t pid = -1;
    const int rc = ::posix_spawn(&pid, executable.c_str(), &actions.value, &attributes.value, argv, environment);
    if (rc) { errno = rc; fail("process creation"); }
    return pid;
}
#endif

struct SocketPath {
    Fd directory;
    std::string name;
    sockaddr_un address{};
};

SocketPath checked_path(const std::string& path) {
    absolute(path);
    if (path.size() >= sizeof(sockaddr_un::sun_path))
        throw std::invalid_argument("local socket path exceeds the platform limit");
    const auto slash = path.rfind('/');
    const std::string directory = slash == 0 ? "/" : path.substr(0, slash);
    const std::string name = path.substr(slash + 1);
    if (name.empty() || name == "." || name == "..")
        throw std::invalid_argument("local socket filename is invalid");
    Fd dir(::open(directory.c_str(), O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW));
    if (dir.get() < 0) fail("private socket directory open");
    struct stat st{};
    if (::fstat(dir.get(), &st) != 0) fail("private socket directory inspection");
    if (!S_ISDIR(st.st_mode) || st.st_uid != ::geteuid() || (st.st_mode & 07777) != 0700)
        throw std::runtime_error("local socket parent must be owned by this effective UID with mode 0700");
    SocketPath result{std::move(dir), name, {}};
    result.address.sun_family = AF_UNIX;
    std::memcpy(result.address.sun_path, path.c_str(), path.size() + 1);
#if defined(__APPLE__)
    result.address.sun_len = static_cast<unsigned char>(sizeof(sockaddr_un));
#endif
    return result;
}

void check_peer(int fd) {
#if defined(__APPLE__)
    uid_t uid;
    gid_t gid;
    if (::getpeereid(fd, &uid, &gid) != 0) fail("local peer identity");
    if (uid != ::geteuid()) throw std::runtime_error("local peer UID does not match");
#else
    struct ucred credentials{};
    socklen_t size = sizeof(credentials);
    if (::getsockopt(fd, SOL_SOCKET, SO_PEERCRED, &credentials, &size) != 0 || size != sizeof(credentials))
        fail("local peer identity");
    if (credentials.uid != ::geteuid()) throw std::runtime_error("local peer UID does not match");
#endif
}

struct BoundSocket {
    Fd directory;
    std::string name;
    dev_t device;
    ino_t inode;
};
std::mutex sockets_mutex;
std::map<std::string, BoundSocket> sockets;

void unlink_matching(const BoundSocket& socket) noexcept {
    struct stat current{};
    if (::fstatat(socket.directory.get(), socket.name.c_str(), &current, AT_SYMLINK_NOFOLLOW) == 0 &&
        S_ISSOCK(current.st_mode) && current.st_dev == socket.device && current.st_ino == socket.inode)
        ::unlinkat(socket.directory.get(), socket.name.c_str(), 0);
}

} // namespace

ProcessResult run_process(const std::string& executable,
                          const std::vector<std::string>& args,
                          const std::string& cwd,
                          const std::map<std::string, std::string>& env,
                          int timeout_ms, std::size_t max_output_bytes) {
    absolute(executable);
    absolute(cwd);
    if (timeout_ms <= 0 || max_output_bytes == 0)
        throw std::invalid_argument("process limits must be positive");
    std::vector<std::string> argv_storage{executable};
    argv_storage.insert(argv_storage.end(), args.begin(), args.end());
    for (const auto& arg : argv_storage)
        if (has_nul(arg)) throw std::invalid_argument("process argument contains a NUL byte");
    std::vector<std::string> env_storage;
    for (const auto& [key, value] : env) {
        if (key.empty() || key.find('=') != std::string::npos || has_nul(key) || has_nul(value))
            throw std::invalid_argument("process environment entry is invalid");
        env_storage.push_back(key + "=" + value);
    }
    std::vector<char*> argv;
    for (auto& item : argv_storage) argv.push_back(item.data());
    argv.push_back(nullptr);
    std::vector<char*> environment;
    for (auto& item : env_storage) environment.push_back(item.data());
    environment.push_back(nullptr);

    auto stdout_pipe = make_pipe();
    auto stderr_pipe = make_pipe();
    Fd null_input(::open("/dev/null", O_RDONLY | O_CLOEXEC));
    if (null_input.get() < 0) fail("null input open");
    nonblocking(stdout_pipe[0].get());
    nonblocking(stderr_pipe[0].get());
#if defined(__linux__)
    struct rlimit fd_limit{};
    if (::getrlimit(RLIMIT_NOFILE, &fd_limit) != 0) fail("descriptor limit query");
    const unsigned long close_limit = fd_limit.rlim_max == RLIM_INFINITY ?
        static_cast<unsigned long>(::sysconf(_SC_OPEN_MAX)) :
        static_cast<unsigned long>(std::min<rlim_t>(fd_limit.rlim_max, INT_MAX));
#endif
    const auto deadline = std::chrono::steady_clock::now() + std::chrono::milliseconds(timeout_ms);

#if defined(__APPLE__)
    const pid_t pid = spawn_mac(executable, cwd, argv.data(), environment.data(), null_input.get(),
                               stdout_pipe[1].get(), stderr_pipe[1].get());
#else
    const pid_t pid = ::fork();
    if (pid < 0) fail("process creation");
    if (pid == 0) {
        // Only async-signal-safe operations after fork; storage is prepared above.
        if (::setpgid(0, 0) != 0 || ::chdir(cwd.c_str()) != 0 ||
            ::dup2(null_input.get(), STDIN_FILENO) < 0 ||
            ::dup2(stdout_pipe[1].get(), STDOUT_FILENO) < 0 ||
            ::dup2(stderr_pipe[1].get(), STDERR_FILENO) < 0) ::_exit(126);
        close_child_descriptors(close_limit);
        ::execve(executable.c_str(), argv.data(), environment.data());
        ::_exit(127);
    }
#endif
    ChildGuard guard{pid};
    // The child also performs this before exec, covering either scheduling order.
    if (::setpgid(pid, pid) != 0 && errno != EACCES && errno != ESRCH) fail("process group setup");
    stdout_pipe[1].reset();
    stderr_pipe[1].reset();
    null_input.reset();

    ProcessResult result;
    bool exited = false;
    bool stopped = false;
    std::array<char, 8192> buffer{};
    while (!exited || stdout_pipe[0].get() >= 0 || stderr_pipe[0].get() >= 0) {
        if (!exited) {
            siginfo_t info{};
            int rc;
            do { rc = ::waitid(P_PID, static_cast<id_t>(pid), &info, WEXITED | WNOHANG | WNOWAIT); }
            while (rc < 0 && errno == EINTR);
            if (rc < 0) fail("process state query");
            exited = info.si_pid == pid;
        }
        if (std::chrono::steady_clock::now() >= deadline && !stopped &&
            (!exited || stdout_pipe[0].get() >= 0 || stderr_pipe[0].get() >= 0)) {
            result.timed_out = true;
            stopped = true;
            terminate_group(pid);
        }

        std::array<pollfd, 2> polls{{{stdout_pipe[0].get(), POLLIN, 0}, {stderr_pipe[0].get(), POLLIN, 0}}};
        int ready;
        do { ready = ::poll(polls.data(), polls.size(), stopped ? 0 : 10); }
        while (ready < 0 && errno == EINTR);
        if (ready < 0) fail("process output polling");
        for (std::size_t index = 0; index < polls.size(); ++index) {
            Fd& fd = index == 0 ? stdout_pipe[0] : stderr_pipe[0];
            std::string& output = index == 0 ? result.out : result.err;
            if (fd.get() < 0 || (!polls[index].revents && !stopped)) continue;
            // Bound each drain pass so a continuous writer cannot starve timeout.
            for (int reads = 0; reads < 16; ++reads) {
                const ssize_t count = ::read(fd.get(), buffer.data(), buffer.size());
                if (count == 0) { fd.reset(); break; }
                if (count < 0) {
                    if (errno == EINTR) continue;
                    if (errno == EAGAIN || errno == EWOULDBLOCK) break;
                    fail("process output read");
                }
                const auto available = max_output_bytes - result.out.size() - result.err.size();
                const auto keep = std::min<std::size_t>(available, static_cast<std::size_t>(count));
                output.append(buffer.data(), keep);
                if (keep < static_cast<std::size_t>(count)) {
                    result.output_limited = true;
                    stopped = true;
                    terminate_group(pid);
                    break;
                }
            }
        }
        if (stopped) {
            // A deliberately detached descendant may retain a pipe. The caller
            // still receives a bounded result; no unbounded post-kill drain.
            stdout_pipe[0].reset();
            stderr_pipe[0].reset();
            break;
        }
    }
    // A child must not leave background descendants after the request ends.
    kill_group(pid);
    int status = 0;
    reap(pid, status);
    guard.done = true;
    if (WIFEXITED(status)) result.exit_code = WEXITSTATUS(status);
    else if (WIFSIGNALED(status)) result.exit_code = 128 + WTERMSIG(status);
    return result;
}

std::string random_token() {
    std::array<unsigned char, 32> bytes{};
    int result;
    do { result = ::getentropy(bytes.data(), bytes.size()); } while (result != 0 && errno == EINTR);
    if (result != 0) fail("OS randomness");
    constexpr char alphabet[] = "0123456789abcdef";
    std::string token;
    token.reserve(bytes.size() * 2);
    for (const auto byte : bytes) {
        token += alphabet[byte >> 4];
        token += alphabet[byte & 15];
    }
    return token;
}

int listen_local_socket(const std::string& path) {
    auto location = checked_path(path);
    struct stat existing{};
    if (::fstatat(location.directory.get(), location.name.c_str(), &existing, AT_SYMLINK_NOFOLLOW) == 0)
        throw std::runtime_error("local socket path already exists; no stale-path replacement is performed");
    if (errno != ENOENT) fail("local socket path inspection");
    Fd fd(::socket(AF_UNIX, SOCK_STREAM, 0));
    if (fd.get() < 0) fail("local socket creation");
    cloexec(fd.get());
    if (::bind(fd.get(), reinterpret_cast<const sockaddr*>(&location.address), sizeof(location.address)) != 0)
        fail("local socket bind");
    struct stat created{};
    if (::fstatat(location.directory.get(), location.name.c_str(), &created, AT_SYMLINK_NOFOLLOW) != 0)
        fail("bound socket inspection");
    BoundSocket bound{std::move(location.directory), location.name, created.st_dev, created.st_ino};
    try {
        if (!S_ISSOCK(created.st_mode) || created.st_uid != ::geteuid())
            throw std::runtime_error("bound socket identity mismatch");
        if (::fchmodat(bound.directory.get(), bound.name.c_str(), 0600, 0) != 0) fail("local socket permission setting");
        if (::listen(fd.get(), 16) != 0) fail("local socket listen");
        std::lock_guard<std::mutex> lock(sockets_mutex);
        if (sockets.contains(path)) throw std::runtime_error("a local socket is already tracked at this path");
        sockets.emplace(path, std::move(bound));
    } catch (...) {
        unlink_matching(bound);
        throw;
    }
    return fd.release();
}

int connect_local_socket(const std::string& path) {
    auto location = checked_path(path);
    struct stat st{};
    if (::fstatat(location.directory.get(), location.name.c_str(), &st, AT_SYMLINK_NOFOLLOW) != 0)
        fail("local socket inspection");
    if (!S_ISSOCK(st.st_mode) || st.st_uid != ::geteuid() || (st.st_mode & 07777) != 0600)
        throw std::runtime_error("local socket must be owned by this effective UID with mode 0600");
    Fd fd(::socket(AF_UNIX, SOCK_STREAM, 0));
    if (fd.get() < 0) fail("local socket creation");
    cloexec(fd.get());
    int rc;
    do { rc = ::connect(fd.get(), reinterpret_cast<const sockaddr*>(&location.address), sizeof(location.address)); }
    while (rc < 0 && errno == EINTR);
    if (rc < 0) fail("local socket connection");
    check_peer(fd.get());
    return fd.release();
}

int accept_local_socket(int listening_fd) {
    int accepted;
    do { accepted = ::accept(listening_fd, nullptr, nullptr); } while (accepted < 0 && errno == EINTR);
    if (accepted < 0) fail("local socket accept");
    Fd fd(accepted);
    cloexec(fd.get());
    check_peer(fd.get());
    return fd.release();
}

void remove_local_socket(const std::string& path) {
    std::lock_guard<std::mutex> lock(sockets_mutex);
    const auto found = sockets.find(path);
    if (found == sockets.end()) return;
    unlink_matching(found->second);
    sockets.erase(found);
}

} // namespace symphony
