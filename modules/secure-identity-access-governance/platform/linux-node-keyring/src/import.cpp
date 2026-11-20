#include <symphony/ssiag/linux_secret_service.hpp>

#include <charconv>
#include <cerrno>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <memory>
#include <span>
#include <string>
#include <string_view>
#include <utility>

#include <fcntl.h>
#include <signal.h>
#include <sys/mman.h>
#include <sys/prctl.h>
#include <sys/resource.h>
#include <sys/stat.h>
#include <unistd.h>

namespace {
namespace backend = symphony::ssiag::linux_secret_service;
constexpr unsigned deadline_seconds = 30;

// The OpenSSH forced command supplies the binding. The remote stream cannot
// select a collection, node principal, executable, resource, or active version.
bool uuid(const std::string& value) {
    return value.size() == 36 && g_uuid_string_is_valid(value.c_str()) &&
           value.find_first_of("ABCDEF") == std::string::npos;
}

bool valid_collection(const std::string& name) {
    constexpr std::string_view prefix = "/org/freedesktop/secrets/collection/";
    return name.size() > prefix.size() && name.size() <= 512 && name.starts_with(prefix) &&
           name.find('/', prefix.size()) == std::string::npos && g_variant_is_object_path(name.c_str());
}

class Descriptor {
public:
    explicit Descriptor(int fd = -1) : value(fd) {}
    Descriptor(const Descriptor&) = delete;
    Descriptor& operator=(const Descriptor&) = delete;
    Descriptor(Descriptor&& other) noexcept : value(std::exchange(other.value, -1)) {}
    ~Descriptor() { if (value >= 0) close(value); }
    void reset(int next) {
        if (value >= 0) close(value);
        value = next;
    }
    int get() const { return value; }
    explicit operator bool() const { return value >= 0; }
private:
    int value;
};

// Pin each directory using openat. A sticky system temporary directory may
// contain a private staging root, as in the disposable integration tests.
Descriptor open_state(std::string_view path) {
    if (path.empty() || path.front() != '/' || path.back() == '/') return Descriptor{};
    Descriptor current(open("/", O_DIRECTORY | O_NOFOLLOW | O_RDONLY | O_CLOEXEC));
    path.remove_prefix(1);
    while (current && !path.empty()) {
        const auto slash = path.find('/');
        const auto part = path.substr(0, slash);
        if (part.empty() || part == "." || part == "..") return Descriptor{};
        current.reset(openat(current.get(), std::string(part).c_str(),
                             O_DIRECTORY | O_NOFOLLOW | O_RDONLY | O_CLOEXEC));
        struct stat st{};
        if (!current || fstat(current.get(), &st) < 0) return Descriptor{};
        if (slash == path.npos)
            return st.st_uid == getuid() && (st.st_mode & 07777) == 0700 ? std::move(current) : Descriptor{};
        if ((st.st_uid != 0 && st.st_uid != getuid()) ||
            ((st.st_mode & 0022) && !(st.st_uid == 0 && (st.st_mode & S_ISVTX)))) return Descriptor{};
        path.remove_prefix(slash + 1);
    }
    return Descriptor{};
}

bool secure_process(const char* bus_path) {
    struct stat input{}, bus{};
    const char* command = std::getenv("SSH_ORIGINAL_COMMAND");
    if (!command || std::strcmp(command, "symphony-keyring-import-v1") != 0 ||
        getuid() == 0 || geteuid() != getuid() || getegid() != getgid() ||
        fstat(STDIN_FILENO, &input) < 0 || !(S_ISFIFO(input.st_mode) || S_ISSOCK(input.st_mode)) ||
        bus_path[0] != '/' || std::strlen(bus_path) >= 108 || std::strpbrk(bus_path, "\n\r") ||
        lstat(bus_path, &bus) < 0 || !S_ISSOCK(bus.st_mode) || bus.st_uid != getuid())
        return false;
    const rlimit core{0, 0};
    if (setrlimit(RLIMIT_CORE, &core) < 0 || prctl(PR_SET_DUMPABLE, 0) < 0 || clearenv() < 0) return false;
    std::unique_ptr<gchar, decltype(&g_free)> escaped(g_dbus_address_escape_value(bus_path), g_free);
    if (!escaped || setenv("DBUS_SESSION_BUS_ADDRESS", ("unix:path=" + std::string(escaped.get())).c_str(), 1) < 0)
        return false;
    umask(0077);
    return true;
}

class ProtectedBuffer {
public:
    ProtectedBuffer() {
        auto* allocation = mmap(nullptr, backend::maximum_size, PROT_READ | PROT_WRITE,
                                MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
        if (allocation == MAP_FAILED) return;
        if (mlock(allocation, backend::maximum_size) < 0 ||
            madvise(allocation, backend::maximum_size, MADV_DONTDUMP) < 0) {
            munmap(allocation, backend::maximum_size);
            return;
        }
        data = static_cast<std::byte*>(allocation);
    }
    ProtectedBuffer(const ProtectedBuffer&) = delete;
    ProtectedBuffer& operator=(const ProtectedBuffer&) = delete;
    ~ProtectedBuffer() {
        if (data) {
            explicit_bzero(data, backend::maximum_size);
            munmap(data, backend::maximum_size);
        }
    }
    std::byte* data{};
};

ssize_t read_some(int fd, void* data, std::size_t size) {
    ssize_t n;
    do { n = read(fd, data, size); } while (n < 0 && errno == EINTR);
    return n;
}

struct Request {
    std::string id;
    std::size_t count{};
    ProtectedBuffer secret;
};

bool read_request(Request& r, const backend::Scope& slot) {
    if (!r.secret.data) return false;
    std::string header;
    char next{};
    while (true) {
        if (read_some(STDIN_FILENO, &next, 1) != 1) return false;
        if (next == '\n') break;
        if (header.size() >= 121 || next < ' ' || next > '~') return false;
        header.push_back(next);
    }
    if (!header.starts_with("SNKI1 ") || header.size() < 43) return false;
    r.id = header.substr(6, 36);
    const std::string prefix = "SNKI1 " + r.id + " " + slot.tops_id + " " + slot.reference + " ";
    if (!uuid(r.id) || !header.starts_with(prefix) || header.size() <= prefix.size() ||
        header[prefix.size()] == '0') return false;
    const char* end = header.data() + header.size();
    const auto parsed = std::from_chars(header.data() + prefix.size(), end, r.count);
    if (parsed.ec != std::errc{} || parsed.ptr != end || r.count == 0 || r.count > backend::maximum_size)
        return false;
    std::size_t done = 0;
    while (done < r.count) {
        const auto n = read_some(STDIN_FILENO, r.secret.data + done, r.count - done);
        if (n <= 0) return false;
        done += static_cast<std::size_t>(n);
    }
    std::byte trailer{};
    const bool complete = read_some(STDIN_FILENO, &trailer, 1) == 0;
    explicit_bzero(&trailer, sizeof trailer);
    return complete;
}

bool write_all(int fd, std::string_view text) {
    while (!text.empty()) {
        const auto n = write(fd, text.data(), text.size());
        if (n < 0 && errno == EINTR) continue;
        if (n <= 0) return false;
        text.remove_prefix(static_cast<std::size_t>(n));
    }
    return true;
}

int report(std::string text, int status) {
    text.push_back('\n');
    return write_all(STDOUT_FILENO, text) ? status : 1;
}

// Only nonsecret metadata reaches this journal. mkdir/fsync reserves the
// request before any keyring call. A crash at any later stage leaves an
// existing directory: a replay cannot issue a second native mutation.
int duplicate(int request_dir, const std::string& scope, const std::string& id) {
    Descriptor receipt(openat(request_dir, "result", O_RDONLY | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC));
    struct stat st{};
    if (!receipt || fsync(request_dir) < 0 || fstat(receipt.get(), &st) < 0 || !S_ISREG(st.st_mode) ||
        (st.st_mode & 07777) != 0600 || st.st_uid != getuid() || st.st_nlink != 1 ||
        st.st_size <= 0 || st.st_size > 1024) return report("UNCERTAIN " + id, 2);
    std::string body(static_cast<std::size_t>(st.st_size), '\0');
    std::size_t done = 0;
    while (done < body.size()) {
        const auto n = read_some(receipt.get(), body.data() + done, body.size() - done);
        if (n <= 0) return report("UNCERTAIN " + id, 2);
        done += static_cast<std::size_t>(n);
    }
    if (!body.starts_with(scope)) return report("REFUSED", 1);
    const auto result = body.substr(scope.size());
    if (result.starts_with("STORED ") && uuid(result.substr(7)))
        return report("ALREADY_STORED " + id + " " + result.substr(7), 3);
    if (result == "REFUSED") return report("ALREADY_REFUSED " + id, 3);
    if (result.starts_with("UNCERTAIN ") && uuid(result.substr(10)))
        return report("UNCERTAIN " + id + " " + result.substr(10), 2);
    return report("UNCERTAIN " + id, 2);
}

bool save_result(int dir, const std::string& metadata) {
    Descriptor out(openat(dir, "result.tmp", O_CREAT | O_EXCL | O_NOFOLLOW | O_WRONLY | O_CLOEXEC, 0600));
    return out && write_all(out.get(), metadata) && fsync(out.get()) == 0 &&
           renameat(dir, "result.tmp", dir, "result") == 0 && fsync(dir) == 0;
}

int run(int argc, char** argv) {
    if (argc != 6 || !secure_process(argv[5])) return report("REFUSED", 1);
    const backend::Scope slot{argv[1], argv[2], argv[3]};
    if (!valid_collection(slot.collection_path) || !uuid(slot.tops_id) || !uuid(slot.reference))
        return report("REFUSED", 1);
    Descriptor state = open_state(argv[4]);
    if (!state) return report("REFUSED", 1);
    Request request;
    if (!read_request(request, slot)) return report("REFUSED", 1);
    const std::string scope = request.id + "\n" + std::to_string(getuid()) + "\n" + argv[5] + "\n" +
                              slot.tops_id + "\n" + slot.reference + "\n" + slot.collection_path + "\n";
    const bool fresh = mkdirat(state.get(), request.id.c_str(), 0700) == 0;
    if (!fresh && errno != EEXIST) return report("REFUSED", 1);
    Descriptor dir(openat(state.get(), request.id.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    struct stat st{};
    if (!dir || fstat(dir.get(), &st) < 0 || st.st_uid != getuid() || (st.st_mode & 07777) != 0700)
        return report("REFUSED", 1);
    if (!fresh) return duplicate(dir.get(), scope, request.id);
    if (fsync(state.get()) < 0) return report("UNCERTAIN " + request.id, 2);
    std::unique_ptr<GCancellable, decltype(&g_object_unref)> cancel(g_cancellable_new(), g_object_unref);
    const auto result = backend::provision(slot, {request.secret.data, request.count}, cancel.get());
    std::string record = "REFUSED";
    int code = 1;
    if (result.status == backend::Status::ok || result.status == backend::Status::indeterminate) {
        if (!uuid(result.item.generation)) return report("UNCERTAIN " + request.id, 2);
        code = result.status == backend::Status::ok ? 0 : 2;
        record = std::string(code == 0 ? "STORED " : "UNCERTAIN ") + result.item.generation;
    }
    if (!save_result(dir.get(), scope + record)) return report("UNCERTAIN " + request.id, 2);
    if (code == 1) return report("REFUSED " + request.id, code);
    return report(std::string(code == 0 ? "STORED " : "UNCERTAIN ") + request.id + " " + result.item.generation, code);
}
}

int main(int argc, char** argv) {
    // SIGALRM's default action terminates even if a provider blocks. A timeout
    // after reservation requires reconciliation, exactly like a lost SSH reply.
    struct sigaction terminate{};
    terminate.sa_handler = SIG_DFL;
    sigemptyset(&terminate.sa_mask);
    sigset_t alarm_signal;
    sigemptyset(&alarm_signal);
    sigaddset(&alarm_signal, SIGALRM);
    if (sigaction(SIGALRM, &terminate, nullptr) < 0 || sigprocmask(SIG_UNBLOCK, &alarm_signal, nullptr) < 0)
        return report("REFUSED", 1);
    alarm(deadline_seconds);
    try { return run(argc, argv); } catch (...) { return report("UNCERTAIN", 2); }
}
