#include "adapter.hpp"
#include <symphony/knowledge/engine/digest.hpp>
#include <algorithm>
#include <cerrno>
#include <fcntl.h>
#include <set>
#include <stdexcept>
#include <sys/stat.h>
#include <unistd.h>
#include <vector>

namespace cloudflare {
namespace {
void check(bool ok) { if (!ok) throw std::runtime_error("invalid project selection"); }
struct File {
    int fd;
    explicit File(int value) : fd(value) { check(fd >= 0); }
    ~File() { ::close(fd); }
    File(const File&) = delete;
    File& operator=(const File&) = delete;
};
// Resolve every component relative to an already opened directory, refusing
// symlinks and special files. No filesystem traversal discovers project inputs.
int directory(int parent, const std::string& name) {
    return ::openat(parent, name.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
}
void relative_path(const std::string& text) {
    fs::path p(text);
    check(!text.empty() && text.size() <= 4096 && !p.is_absolute() && p.lexically_normal().generic_string() == text);
    check(std::all_of(text.begin(), text.end(), [](unsigned char c) { return c >= 32 && c < 127 && c != '\\'; }));
    for (const auto& component : p) {
        auto s = component.string();
        check(!s.empty() && s != "." && s != "..");
        // Credentials, VCS state, build output and dependency trees belong to
        // separate admitted inputs, never this source-only observation.
        check(s != ".git" && s != "node_modules" && s != ".cloudflare" && s != ".npmrc" &&
              !s.starts_with(".env") && !s.starts_with(".dev.vars"));
    }
}
bool unchanged(const struct stat& a, const struct stat& b) {
    if (a.st_dev != b.st_dev || a.st_ino != b.st_ino || a.st_mode != b.st_mode ||
        a.st_nlink != b.st_nlink || a.st_size != b.st_size) return false;
#ifdef __APPLE__
    return a.st_mtimespec.tv_sec == b.st_mtimespec.tv_sec && a.st_mtimespec.tv_nsec == b.st_mtimespec.tv_nsec &&
           a.st_ctimespec.tv_sec == b.st_ctimespec.tv_sec && a.st_ctimespec.tv_nsec == b.st_ctimespec.tv_nsec;
#else
    return a.st_mtim.tv_sec == b.st_mtim.tv_sec && a.st_mtim.tv_nsec == b.st_mtim.tv_nsec &&
           a.st_ctim.tv_sec == b.st_ctim.tv_sec && a.st_ctim.tv_nsec == b.st_ctim.tv_nsec;
#endif
}
Json observe_file(int root, const std::string& path, std::uint64_t& total) {
    File parent(::dup(root));
    for (const auto& part : fs::path(path).parent_path()) {
        File next(directory(parent.fd, part.string()));
        check(::dup2(next.fd, parent.fd) >= 0);
    }
    File file(::openat(parent.fd, fs::path(path).filename().c_str(), O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC));
    struct stat before{}, after{};
    check(::fstat(file.fd, &before) == 0 && S_ISREG(before.st_mode) && before.st_nlink == 1 &&
          before.st_size >= 0 && before.st_size <= 16 * 1024 * 1024);
    total += static_cast<std::uint64_t>(before.st_size);
    check(total <= 64 * 1024 * 1024);
    std::string bytes(static_cast<std::size_t>(before.st_size), '\0');
    std::size_t offset = 0;
    while (offset < bytes.size()) {
        auto n = ::read(file.fd, bytes.data() + offset, bytes.size() - offset);
        if (n < 0 && errno == EINTR) continue;
        check(n > 0); offset += static_cast<std::size_t>(n);
    }
    char extra;
    check(::read(file.fd, &extra, 1) == 0 && ::fstat(file.fd, &after) == 0 && unchanged(before, after));
    // Also refuse replacement of the selected directory entry while reading.
    struct stat named{};
    check(::fstatat(parent.fd, fs::path(path).filename().c_str(), &named, AT_SYMLINK_NOFOLLOW) == 0 && unchanged(after, named));
    return Json{{"path", path}, {"digest", symphony::knowledge::engine::tagged_sha256(bytes)},
                {"size", bytes.size()}, {"executable", (before.st_mode & 0111) != 0}};
}
}
Json project_inspect(const Json& input) {
    check(::geteuid() != 0 && ::getuid() == ::geteuid());
    const auto root = fs::path(input.at("root").get<std::string>());
    check(root.is_absolute() && root.lexically_normal().generic_string() == input.at("root").get<std::string>() && root != root.root_path());
    File directory_fd(::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC));
    for (const auto& component : root.relative_path()) {
        check(component != "." && component != ".." && !component.empty());
        File next(directory(directory_fd.fd, component.string()));
        check(::dup2(next.fd, directory_fd.fd) >= 0);
    }
    auto files = input.at("files").get<std::vector<std::string>>();
    check(!files.empty() && files.size() <= 4096);
    std::sort(files.begin(), files.end());
    check(std::adjacent_find(files.begin(), files.end()) == files.end());
    for (const auto& file : files) relative_path(file);
    auto config = input.at("configuration").get<std::string>();
    auto kind = input.at("configuration_kind").get<std::string>();
    check(std::binary_search(files.begin(), files.end(), config));
    check((kind == "cf" && config == "cloudflare.config.ts") ||
          (kind == "wrangler" && (config == "wrangler.json" || config == "wrangler.jsonc" || config == "wrangler.toml")));
    Json entries = Json::array(); std::uint64_t total = 0;
    for (const auto& path : files) entries.push_back(observe_file(directory_fd.fd, path, total));
    Json manifest{{"protocol", "symphony.sov.cloudflare-project-manifest.v1"}, {"root", root.string()},
                  {"configuration", config}, {"configuration_kind", kind}, {"files", entries}, {"total_bytes", total}};
    return Json{{"manifest", manifest}, {"snapshot_digest", digest(manifest, "symphony.sov.cloudflare-project-manifest.v1")},
                {"completeness", "selected_files_only"}, {"content_retained", false}, {"configuration_evaluated", false},
                {"migration_required", kind == "wrangler"}, {"provider_dispatch", false}, {"execution_available", false}};
}
}
