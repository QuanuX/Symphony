#include "control_common.hpp"

#include <algorithm>
#include <array>
#include <cerrno>
#include <cstdio>
#include <cstring>
#include <limits>
#include <optional>
#include <string>
#include <sys/file.h>
#include <sys/random.h>
#include <sys/stat.h>
#include <sys/xattr.h>

namespace {
using namespace node_keyring_cli;
constexpr std::size_t limit = 2048;
constexpr int safe = O_NOFOLLOW | O_CLOEXEC | O_NONBLOCK;

std::optional<std::string> open_parent(File& parent, std::string_view path) {
    if (!path.starts_with('/') || path.size() > 4096) return {};
    path.remove_prefix(1);
    parent.reset(open("/", O_DIRECTORY | O_RDONLY | safe));
    while (parent && !path.empty()) {
        struct stat st{};
        if (fstat(parent.get(), &st) < 0 || st.st_uid != 0 || (st.st_mode & 0022)) return {};
        const auto slash = path.find('/');
        const std::string part(path.substr(0, slash));
        if (part.empty() || part == "." || part == "..") return {};
        if (slash == path.npos) return part;
        parent.reset(openat(parent.get(), part.c_str(), O_DIRECTORY | O_RDONLY | safe));
        path.remove_prefix(slash + 1);
    }
    return {};
}

std::optional<std::string_view> take_field(std::string_view& s, std::string_view name) {
    if (!s.starts_with(name)) return {};
    s.remove_prefix(name.size());
    const auto end = s.find('\n');
    if (end == s.npos || end == 0) return {};
    auto field = s.substr(0, end);
    s.remove_prefix(end + 1);
    return field;
}
std::optional<std::string_view> generation(std::string_view s) {
    constexpr std::string_view header = "SYMPHONY-NODE-KEYRING-1\n";
    constexpr std::string_view prefix = "/org/freedesktop/secrets/collection/";
    if (!s.starts_with(header) || s.find('\0') != s.npos) return {};
    s.remove_prefix(header.size());
    auto uid = take_field(s, "uid="), bus = take_field(s, "bus_socket="), collection = take_field(s, "collection=");
    auto tops = take_field(s, "tops="), ref = take_field(s, "reference="), gen = take_field(s, "generation=");
    auto ceiling = take_field(s, "maximum_bytes=");
    unsigned number;
    if (!s.empty() || !uid || uid->front() == '0' || !decimal(*uid, number) ||
        number == std::numeric_limits<unsigned>::max() || !bus || !bus->starts_with('/') ||
        bus->size() < 2 || bus->size() > 107 || bus->find_first_of("\t\r") != bus->npos ||
        !collection || !collection->starts_with(prefix) || collection->size() <= prefix.size() || collection->size() > 512 ||
        !tops || !uuid(*tops) || !ref || !uuid(*ref) || !gen || !uuid(*gen) ||
        !ceiling || ceiling->front() == '0' || !decimal(*ceiling, number) || number > 4096) return {};
    for (char c : collection->substr(prefix.size())) {
        if (!((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_')) return {};
    }
    return gen;
}

// File ACLs, capabilities and content-integrity xattrs cannot be silently discarded on rename.
// Encode the presence of an SELinux label separately from its raw value.
std::optional<std::string> attributes(int fd) {
    std::array<char, 512> names{};
    const auto n = flistxattr(fd, names.data(), names.size());
    if (n < 0) return {};
    if (n == 0) return std::string{};
    constexpr char label[] = "security.selinux";
    if (static_cast<std::size_t>(n) != sizeof label || memcmp(names.data(), label, sizeof label) != 0) return {};
    const auto size = fgetxattr(fd, label, nullptr, 0);
    if (size < 0 || size > 65536) return {};
    std::string result(1 + static_cast<std::size_t>(size), '\0');
    if (fgetxattr(fd, label, result.data() + 1, size) != size) return {};
    return result;
}

std::string temp_name() {
    std::array<unsigned char, 16> random{};
    if (getrandom(random.data(), random.size(), 0) != static_cast<ssize_t>(random.size())) return {};
    std::string result = ".symphony-keyring-";
    constexpr char hex[] = "0123456789abcdef";
    for (auto b : random) { result += hex[b >> 4]; result += hex[b & 15]; }
    return result + ".tmp";
}

enum class Status { active, already, stale, uncertain, refused };

Status activate(std::string_view path, std::string_view expected, std::string_view replacement) {
    if (getuid() != 0 || geteuid() != 0 || !uuid(expected) || !uuid(replacement)) return Status::refused;
    File parent;
    const auto name = open_parent(parent, path);
    if (!name || flock(parent.get(), LOCK_EX) < 0) return Status::refused;
    // Serialize on the stable parent. A lock on the old file would lose protection at rename.
    File original(openat(parent.get(), name->c_str(), O_RDONLY | safe));
    struct stat before{};
    if (!original || fstat(original.get(), &before) < 0 || !S_ISREG(before.st_mode) || before.st_uid != 0 ||
        before.st_nlink != 1 || (before.st_mode & 07022) || before.st_size <= 0 ||
        before.st_size > static_cast<off_t>(limit)) return Status::refused;
    std::array<char, limit + 1> raw{};
    if (pread(original.get(), raw.data(), raw.size(), 0) != before.st_size) return Status::refused;
    auto current = generation({raw.data(), static_cast<std::size_t>(before.st_size)});
    if (!current) return Status::refused;
    if (*current == replacement) {
        return fsync(original.get()) == 0 && fsync(parent.get()) == 0 ? Status::already : Status::uncertain;
    }
    if (*current != expected) return Status::stale;
    const auto original_attrs = attributes(original.get());
    if (!original_attrs) return Status::refused;
    const auto temp = temp_name();
    if (temp.empty()) return Status::refused;
    File staged(openat(parent.get(), temp.c_str(), O_CREAT | O_EXCL | O_RDWR | safe, 0600));
    if (!staged) return Status::refused;
    struct RemoveTemp {
        int parent;
        const char* name;
        ~RemoveTemp() { if (name) unlinkat(parent, name, 0); }
    } cleanup{parent.get(), temp.c_str()};
    if (fchown(staged.get(), before.st_uid, before.st_gid) != 0 || fchmod(staged.get(), before.st_mode & 07777) != 0 ||
        attributes(staged.get()) != original_attrs) return Status::refused;
    const auto offset = current->data() - raw.data();
    std::copy(replacement.begin(), replacement.end(), raw.begin() + offset);
    if (write(staged.get(), raw.data(), before.st_size) != before.st_size || fsync(staged.get()) < 0)
        return Status::refused;
    struct stat latest{};
    if (fstatat(parent.get(), name->c_str(), &latest, AT_SYMLINK_NOFOLLOW) < 0) return Status::refused;
    if (before.st_dev != latest.st_dev || before.st_ino != latest.st_ino ||
        before.st_ctim.tv_sec != latest.st_ctim.tv_sec || before.st_ctim.tv_nsec != latest.st_ctim.tv_nsec)
        return Status::stale;
    if (renameat(parent.get(), temp.c_str(), parent.get(), name->c_str()) < 0) return Status::refused;
    cleanup.name = nullptr;
    return fsync(parent.get()) == 0 ? Status::active : Status::uncertain;
}
}

int main(int argc, char** argv) {
    if (argc != 4) {
        fputs("usage: symphony-node-keyring-activate ABSOLUTE_BINDING EXPECTED_GENERATION NEW_GENERATION\n", stderr);
        return 1;
    }
    try {
        switch (activate(argv[1], argv[2], argv[3])) {
        case Status::active: printf("ACTIVE %s\n", argv[3]); return 0;
        case Status::already: printf("ALREADY_ACTIVE %s\n", argv[3]); return 0;
        case Status::stale: puts("STALE"); return 3;
        case Status::uncertain: puts("UNCERTAIN"); return 2;
        case Status::refused: break;
        }
    } catch (...) {}
    puts("REFUSED");
    return 1;
}
