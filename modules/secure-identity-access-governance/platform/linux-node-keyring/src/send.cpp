#include "control_common.hpp"

#include <algorithm>
#include <array>
#include <cerrno>
#include <chrono>
#include <cstdio>
#include <cstring>
#include <getopt.h>
#include <poll.h>
#include <signal.h>
#include <string>
#include <vector>
#include <sys/resource.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/wait.h>

namespace {
using namespace node_keyring_cli;
using Clock = std::chrono::steady_clock;
constexpr std::size_t maximum = 4096;
constexpr const char* usage =
    "usage: symphony-node-keyring-send --host NODE --user SERVICE_USER --known-hosts PIN_FILE "
    "--identity KEY_FILE --request UUID --tops UUID --reference UUID --length BYTES [--port PORT]\n";

bool word(char c) {
    return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9');
}
bool trusted_file(std::string_view path) {
    if (!path.starts_with('/') || path.size() > 4096) return false;
    path.remove_prefix(1);
    File directory(open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC));
    while (directory && !path.empty()) {
        auto slash = path.find('/');
        const std::string part(path.substr(0, slash));
        if (part.empty() || part == "." || part == ".." || !std::all_of(part.begin(), part.end(),
                [](char c) { return word(c) || c == '-' || c == '_' || c == '.'; })) return false;
        struct stat st{};
        if (fstatat(directory.get(), part.c_str(), &st, AT_SYMLINK_NOFOLLOW) != 0 ||
            (st.st_uid != 0 && st.st_uid != geteuid()) || (st.st_mode & 0022)) return false;
        if (slash == path.npos) return S_ISREG(st.st_mode);
        if (!S_ISDIR(st.st_mode)) return false;
        directory.reset(openat(directory.get(), part.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
        path.remove_prefix(slash + 1);
    }
    return false;
}

struct Options {
    std::string host, user, pins, identity, request, tops, reference;
    unsigned length = 0, port = 22;
};

bool options(int argc, char** argv, Options& o) {
    static const option flags[] = {
        {"host", required_argument, nullptr, 'h'}, {"user", required_argument, nullptr, 'u'},
        {"known-hosts", required_argument, nullptr, 'k'}, {"identity", required_argument, nullptr, 'i'},
        {"request", required_argument, nullptr, 'r'}, {"tops", required_argument, nullptr, 't'},
        {"reference", required_argument, nullptr, 'f'}, {"length", required_argument, nullptr, 'l'},
        {"port", required_argument, nullptr, 'p'}, {nullptr, 0, nullptr, 0}};
    opterr = 0;
    int c;
    while ((c = getopt_long(argc, argv, "", flags, nullptr)) != -1) {
        switch (c) {
        case 'h': o.host = optarg; break;
        case 'u': o.user = optarg; break;
        case 'k': o.pins = optarg; break;
        case 'i': o.identity = optarg; break;
        case 'r': o.request = optarg; break;
        case 't': o.tops = optarg; break;
        case 'f': o.reference = optarg; break;
        case 'l': if (!decimal(optarg, o.length)) return false; break;
        case 'p': if (!decimal(optarg, o.port)) return false; break;
        default: return false;
        }
    }
    auto lower = [](char ch) { return ch >= 'a' && ch <= 'z'; };
    return optind == argc && uuid(o.request) && uuid(o.tops) && uuid(o.reference) &&
           o.length > 0 && o.length <= maximum && o.port > 0 && o.port <= 65535 &&
           !o.host.empty() && o.host.size() <= 253 && word(o.host[0]) &&
           std::all_of(o.host.begin(), o.host.end(), [](char v) { return word(v) || v == '.' || v == ':' || v == '-'; }) &&
           !o.user.empty() && o.user.size() <= 32 && (lower(o.user[0]) || o.user[0] == '_') &&
           std::all_of(o.user.begin(), o.user.end(), [&](char v) { return lower(v) || (v >= '0' && v <= '9') || v == '_' || v == '-'; }) &&
           trusted_file(o.pins) && trusted_file(o.identity);
}

// One bounded buffer, including the validated header, plus one byte to detect an overlong input.
struct Frame {
    std::array<char, 128 + maximum + 1> bytes{};
    std::size_t size = 0;
    ~Frame() {
        volatile char* p = bytes.data();
        for (std::size_t i = 0; i < bytes.size(); ++i) p[i] = 0;
    }
};

bool ready(int fd, short events, Clock::time_point until, short& returned) {
    for (;;) {
        const auto remaining = std::chrono::ceil<std::chrono::milliseconds>(until - Clock::now()).count();
        if (remaining <= 0) return false;
        pollfd p{fd, events, 0};
        int r = poll(&p, 1, static_cast<int>(remaining));
        if (r < 0 && errno == EINTR) continue;
        returned = p.revents;
        return r == 1 && !(p.revents & POLLNVAL);
    }
}

bool read_source(const Options& o, Frame& frame) {
    struct stat st{};
    if (fstat(STDIN_FILENO, &st) != 0 || !(S_ISFIFO(st.st_mode) || S_ISSOCK(st.st_mode))) return false;
    const std::string header = "SNKI1 " + o.request + " " + o.tops + " " + o.reference + " " + std::to_string(o.length) + "\n";
    if (header.size() > 128) return false;
    memcpy(frame.bytes.data(), header.data(), header.size());
    frame.size = header.size();
    const auto limit = header.size() + o.length;
    const auto until = Clock::now() + std::chrono::seconds(15);
    while (frame.size <= limit) {
        short events{};
        if (!ready(STDIN_FILENO, POLLIN, until, events)) return false;
        const auto n = read(STDIN_FILENO, frame.bytes.data() + frame.size, limit + 1 - frame.size);
        if (n < 0 && (errno == EINTR || errno == EAGAIN)) continue;
        if (n <= 0) return n == 0 && frame.size == limit;
        frame.size += static_cast<std::size_t>(n);
    }
    return false;
}

struct Child {
    pid_t pid;
    ~Child() {
        if (pid > 0) {
            kill(pid, SIGKILL);
            int ignored;
            while (waitpid(pid, &ignored, 0) < 0 && errno == EINTR) {}
        }
    }
    int wait(Clock::time_point until) {
        int status = 0;
        while (Clock::now() < until) {
            auto observed = waitpid(pid, &status, WNOHANG);
            if (observed == pid) {
                pid = -1;
                return WIFEXITED(status) ? WEXITSTATUS(status) : -1;
            }
            if (observed < 0 && errno != EINTR) return -1;
            poll(nullptr, 0, 10);
        }
        return -1;
    }
};

std::vector<std::string> arguments(const Options& o) {
    return {"/usr/bin/ssh", "-F", "/dev/null", "-T", "-a", "-x", "-oBatchMode=yes",
            "-oStrictHostKeyChecking=yes", "-oGlobalKnownHostsFile=/dev/null", "-oUserKnownHostsFile=" + o.pins,
            "-oUpdateHostKeys=no", "-oIdentitiesOnly=yes", "-oIdentityAgent=none", "-oPreferredAuthentications=publickey",
            "-oClearAllForwardings=yes", "-oControlMaster=no", "-oControlPath=none", "-oConnectTimeout=8",
            "-oConnectionAttempts=1", "-i", o.identity, "-p", std::to_string(o.port), "-l", o.user,
            "--", o.host, "symphony-keyring-import-v1"};
}

int exchange(const Options& o, const Frame& frame, std::string& reply) {
    auto args = arguments(o);
    std::vector<char*> argv;
    for (auto& arg : args) argv.push_back(arg.data());
    argv.push_back(nullptr);
    int pair[2];
    if (socketpair(AF_UNIX, SOCK_STREAM, 0, pair) < 0) return -1;
    File local(pair[0]), remote(pair[1]), quiet(open("/dev/null", O_WRONLY | O_CLOEXEC));
    if (!quiet || fcntl(local.get(), F_SETFL, O_NONBLOCK) < 0) return -1;
    const auto fd_limit = sysconf(_SC_OPEN_MAX);
    if (fd_limit < 0) return -1;
    Child child{fork()};
    if (child.pid < 0) return -1;
    if (child.pid == 0) {
        if (dup2(remote.get(), STDIN_FILENO) < 0 || dup2(remote.get(), STDOUT_FILENO) < 0 ||
            dup2(quiet.get(), STDERR_FILENO) < 0) _exit(127);
#ifdef __linux__
        if (close_range(3, ~0u, 0) != 0)
#endif
            for (long fd = 3; fd < fd_limit; ++fd) close(static_cast<int>(fd));
        char locale[] = "LC_ALL=C", path[] = "PATH=/usr/bin:/bin";
        char* env[] = {locale, path, nullptr};
        signal(SIGPIPE, SIG_DFL);
        execve(argv[0], argv.data(), env);
        _exit(127);
    }
    remote.reset();
    const auto until = Clock::now() + std::chrono::seconds(30);
    std::size_t sent = 0;
    for (;;) {
        short events{};
        if (!ready(local.get(), static_cast<short>(POLLIN | (sent == frame.size ? 0 : POLLOUT)), until, events)) return -1;
        if ((events & POLLOUT) && sent < frame.size) {
            auto n = write(local.get(), frame.bytes.data() + sent, frame.size - sent);
            if (n < 0 && (errno == EINTR || errno == EAGAIN)) continue;
            if (n <= 0) return -1;
            sent += static_cast<std::size_t>(n);
            if (sent == frame.size && shutdown(local.get(), SHUT_WR) < 0) return -1;
        }
        if (events & (POLLIN | POLLHUP | POLLERR)) {
            std::array<char, 161> buf{};
            auto n = read(local.get(), buf.data(), buf.size() - reply.size());
            if (n < 0 && (errno == EINTR || errno == EAGAIN)) continue;
            if (n < 0) return -1;
            if (n == 0) break;
            reply.append(buf.data(), static_cast<std::size_t>(n));
            if (reply.size() > 160) return -1;
        }
    }
    return sent == frame.size ? child.wait(until) : -1;
}

bool receipt(int status, const std::string& id, std::string_view reply) {
    if (!reply.ends_with('\n')) return false;
    reply.remove_suffix(1);
    const auto space = reply.find(' ');
    if (space == reply.npos) return false;
    auto tag = reply.substr(0, space);
    reply.remove_prefix(space + 1);
    if (!reply.starts_with(id)) return false;
    reply.remove_prefix(id.size());
    const bool generation = reply.starts_with(' ') && uuid(reply.substr(1));
    return (status == 0 && tag == "STORED" && generation) ||
           (status == 3 && tag == "ALREADY_STORED" && generation) ||
           (status == 1 && tag == "REFUSED" && reply.empty()) ||
           (status == 3 && tag == "ALREADY_REFUSED" && reply.empty()) ||
           (status == 2 && tag == "UNCERTAIN" && (generation || reply.empty()));
}
}

int main(int argc, char** argv) {
    if (argc == 2 && (std::string_view(argv[1]) == "--help" || std::string_view(argv[1]) == "-h")) {
        fputs(usage, stdout);
        return 0;
    }
    Options o;
    if (!options(argc, argv, o)) { fputs(usage, stderr); return 2; }
    const rlimit core{0, 0};
    if (setrlimit(RLIMIT_CORE, &core) < 0 || signal(SIGPIPE, SIG_IGN) == SIG_ERR) return 1;
    Frame frame;
    if (!read_source(o, frame)) { fputs("Source refused before transfer\n", stderr); return 1; }
    std::string reply;
    const auto status = exchange(o, frame, reply);
    if (!receipt(status, o.request, reply)) {
        fprintf(stderr, "UNCONFIRMED %s\n", o.request.c_str());
        return 2;
    }
    fputs(reply.c_str(), stdout);
    return status;
}
