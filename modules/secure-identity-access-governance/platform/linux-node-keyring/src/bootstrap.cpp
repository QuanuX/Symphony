#include <symphony/ssiag/node_keyring_bootstrap.hpp>
#include <symphony/ssiag/linux_secret_service.hpp>

#include <array>
#include <cerrno>
#include <charconv>
#include <condition_variable>
#include <fcntl.h>
#include <memory>
#include <mutex>
#include <optional>
#include <string_view>
#include <sys/stat.h>
#include <sys/un.h>
#include <thread>
#include <unistd.h>
#include <utility>

namespace symphony::ssiag::node_keyring {
namespace {
namespace keyring = linux_secret_service;
struct File {
    int fd;
    ~File() { if (fd >= 0) close(fd); }
    File(const File&) = delete;
    File& operator=(const File&) = delete;
    explicit File(int value) : fd(value) {}
};
struct Binding {
    std::string bus_socket;
    keyring::Item item;
    unsigned maximum_bytes = 0;
};

bool owned(int fd, bool directory) {
    struct stat st{};
    return fd >= 0 && fstat(fd, &st) == 0 && st.st_uid == 0 && (st.st_mode & 0022) == 0 &&
           (directory ? S_ISDIR(st.st_mode) : (S_ISREG(st.st_mode) && st.st_nlink == 1));
}

std::optional<std::string> read_binding(std::string_view path) {
    if (path.empty() || path.front() != '/' || path.size() > 4096 || path.find('\0') != path.npos)
        return std::nullopt;
    path.remove_prefix(1);
    File directory(open("/", O_PATH | O_DIRECTORY | O_CLOEXEC));
    while (!path.empty()) {
        if (!owned(directory.fd, true)) return std::nullopt;
        const auto slash = path.find('/');
        const std::string part(path.substr(0, slash));
        if (part.empty() || part == "." || part == "..") return std::nullopt;
        if (slash != path.npos) {
            const int next = openat(directory.fd, part.c_str(), O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
            close(directory.fd);
            directory.fd = next;
            path.remove_prefix(slash + 1);
            continue;
        }
        struct stat candidate{};
        if (fstatat(directory.fd, part.c_str(), &candidate, AT_SYMLINK_NOFOLLOW) != 0 ||
            !S_ISREG(candidate.st_mode)) return std::nullopt;
        File file(openat(directory.fd, part.c_str(), O_RDONLY | O_NOCTTY | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC));
        if (!owned(file.fd, false)) return std::nullopt;
        constexpr std::size_t limit = 2048;
        std::array<char, limit + 1> bytes{};
        std::size_t used = 0;
        while (used < bytes.size()) {
            const auto n = read(file.fd, bytes.data() + used, bytes.size() - used);
            if (n == 0) return std::string(bytes.data(), used);
            if (n < 0) {
                if (errno == EINTR) continue;
                return std::nullopt;
            }
            used += static_cast<std::size_t>(n);
        }
        return std::nullopt;
    }
    return std::nullopt;
}

std::optional<std::string_view> field(std::string_view& data, std::string_view key) {
    const auto end = data.find('\n');
    if (end == data.npos || !data.starts_with(key) || end <= key.size()) return std::nullopt;
    const auto value = data.substr(key.size(), end - key.size());
    data.remove_prefix(end + 1);
    return value;
}
bool number(std::string_view text, unsigned& value) {
    auto [end, error] = std::from_chars(text.data(), text.data() + text.size(), value);
    return error == std::errc{} && end == text.data() + text.size() && text.front() != '0';
}
std::optional<Binding> parse(std::string_view text) {
    constexpr std::string_view header = "SYMPHONY-NODE-KEYRING-1\n";
    if (!text.starts_with(header) || text.find('\0') != text.npos) return std::nullopt;
    text.remove_prefix(header.size());
    const auto uid = field(text, "uid=");
    const auto bus = field(text, "bus_socket=");
    const auto collection = field(text, "collection=");
    const auto tops = field(text, "tops=");
    const auto reference = field(text, "reference=");
    const auto generation = field(text, "generation=");
    const auto ceiling = field(text, "maximum_bytes=");
    unsigned selected_uid{}, maximum{};
    if (!uid || !number(*uid, selected_uid) || selected_uid != getuid() ||
        !bus || bus->front() != '/' || bus->size() >= sizeof(sockaddr_un::sun_path) ||
        bus->find_first_of("\r\t") != bus->npos ||
        !collection || !tops || !reference || !generation || !ceiling ||
        !number(*ceiling, maximum) || maximum > keyring::maximum_size || !text.empty())
        return std::nullopt;
    // The provider validates the canonical UUIDs and exact collection path before I/O.
    return Binding{std::string(*bus),
        keyring::Item{{std::string(*collection), std::string(*tops), std::string(*reference)},
                      std::string(*generation)}, maximum};
}

BootstrapStatus provider_status(keyring::Status s) {
    switch (s) {
    case keyring::Status::invalid: return BootstrapStatus::invalid;
    case keyring::Status::unavailable: return BootstrapStatus::unavailable;
    case keyring::Status::locked: return BootstrapStatus::locked;
    case keyring::Status::not_found: return BootstrapStatus::not_found;
    case keyring::Status::ambiguous: return BootstrapStatus::ambiguous;
    case keyring::Status::cancelled: return BootstrapStatus::cancelled;
    case keyring::Status::ok:
    case keyring::Status::indeterminate: return BootstrapStatus::indeterminate;
    }
    return BootstrapStatus::indeterminate;
}
} // namespace

SessionBootstrap::SessionBootstrap(std::string binding_path)
    : binding_path_(std::move(binding_path)), pid_(getpid()) {}

BootstrapStatus SessionBootstrap::authenticate(Clock::time_point deadline, std::stop_token stop,
                                               const Authenticate& connect) {
    if (used_.test_and_set()) return BootstrapStatus::already_attempted;
    if (pid_ != getpid() || getuid() == 0 || getuid() != geteuid() || getgid() != getegid())
        return BootstrapStatus::binding_refused;
    const auto now = Clock::now();
    if (stop.stop_requested()) return BootstrapStatus::cancelled;
    if (deadline <= now) return BootstrapStatus::expired;
    if (!connect || deadline > now + std::chrono::seconds(30)) return BootstrapStatus::invalid;
    bool entered = false;
    try {
        const auto data = read_binding(binding_path_);
        if (!data) return BootstrapStatus::binding_refused;
        const auto selected = parse(*data);
        if (!selected) return BootstrapStatus::binding_refused;
        if (stop.stop_requested()) return BootstrapStatus::cancelled;
        if (Clock::now() >= deadline) return BootstrapStatus::expired;

        std::unique_ptr<GCancellable, decltype(&g_object_unref)> cancel(g_cancellable_new(), g_object_unref);
        std::stop_source active;
        std::stop_callback cancel_io(active.get_token(), [&] { g_cancellable_cancel(cancel.get()); });
        std::stop_callback cancel_caller(stop, [&] { active.request_stop(); });
        std::mutex mutex;
        std::condition_variable_any changed;
        std::jthread timer([&](std::stop_token done) {
            std::unique_lock lock(mutex);
            changed.wait_until(lock, done, deadline, [] { return false; });
            if (!done.stop_requested()) active.request_stop();
        });

        Handshake outcome = Handshake::indeterminate;
        const auto result = keyring::use_on_bus(selected->bus_socket, selected->item, selected->maximum_bytes,
            cancel.get(), [&](keyring::Bytes bytes) {
                if (active.stop_requested() || Clock::now() >= deadline) return;
                entered = true;
                outcome = connect(bytes, deadline, active.get_token());
            });
        if (entered) {
            // An external session may already exist after cancellation or timeout.
            // Never describe a late or throwing handshake as safe to retry.
            if (result != keyring::Status::ok || active.stop_requested() || Clock::now() >= deadline)
                return BootstrapStatus::indeterminate;
            if (outcome == Handshake::established) return BootstrapStatus::established;
            if (outcome == Handshake::refused) return BootstrapStatus::refused;
            return BootstrapStatus::indeterminate;
        }
        if (Clock::now() >= deadline) return BootstrapStatus::expired;
        if (active.stop_requested()) return BootstrapStatus::cancelled;
        return provider_status(result);
    } catch (...) {
        return entered ? BootstrapStatus::indeterminate : BootstrapStatus::unavailable;
    }
}
} // namespace symphony::ssiag::node_keyring
