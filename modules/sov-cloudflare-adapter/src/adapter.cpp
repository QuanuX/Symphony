#include "adapter.hpp"
#include "execution.hpp"
#include "connector.hpp"
#include "owner_data.hpp"
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <algorithm>
#include <chrono>
#include <cerrno>
#include <fcntl.h>
#include <fstream>
#include <poll.h>
#include <regex>
#include <set>
#include <signal.h>
#include <stdexcept>
#include <sys/file.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <unistd.h>

namespace cloudflare {
namespace {
struct Failure : std::runtime_error { using std::runtime_error::runtime_error; };
void require(bool ok, const char* code = "invalid_control") { if (!ok) throw Failure(code); }
std::string str(const Json& j) { require(j.is_string()); return j.get<std::string>(); }
constexpr const char* state_protocol = "symphony.sov.cloudflare-state.v1";
constexpr const char* plan_protocol = "symphony.sov.cloudflare-plan.v1";
constexpr const char* bundle_protocol = "symphony.sov.cloudflare-bundle.v1";

struct FD {
    int fd;
    explicit FD(int f) : fd(f) { require(fd >= 0, "local_failure"); }
    ~FD() { if (fd >= 0) ::close(fd); }
    FD(const FD&) = delete;
    FD& operator=(const FD&) = delete;
};
void sync_dir(const fs::path& path) { FD f(::open(path.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW)); require(::fsync(f.fd) == 0, "local_failure"); }
void directories(const fs::path& path) {
    require(path.is_absolute() && path.lexically_normal() == path);
    fs::path current;
    for (const auto& part : path) {
        current /= part;
        struct stat s{};
        require(::lstat(current.c_str(), &s) == 0 && S_ISDIR(s.st_mode), "integrity_failure");
        // A root-owned sticky temp ancestor is permitted; the actual store is private.
        require((s.st_mode & 0022) == 0 || (s.st_uid == 0 && (s.st_mode & S_ISVTX)), "integrity_failure");
    }
}
std::string read_file(const fs::path& path, std::size_t limit) {
    directories(path.parent_path());
    FD f(::open(path.c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC));
    struct stat before{}, after{};
    require(::fstat(f.fd, &before) == 0 && S_ISREG(before.st_mode) && before.st_nlink == 1 && before.st_size >= 0 && static_cast<std::uint64_t>(before.st_size) <= limit, "integrity_failure");
    std::string bytes(static_cast<std::size_t>(before.st_size), '\0');
    std::size_t offset = 0;
    while (offset < bytes.size()) {
        auto n = ::read(f.fd, bytes.data() + offset, bytes.size() - offset);
        if (n < 0 && errno == EINTR) continue;
        require(n > 0, "integrity_failure"); offset += static_cast<std::size_t>(n);
    }
    char extra;
    require(::read(f.fd, &extra, 1) == 0 && ::fstat(f.fd, &after) == 0 && before.st_size == after.st_size && before.st_mtime == after.st_mtime, "integrity_failure");
    return bytes;
}
void write_file(const fs::path& path, const std::string& bytes, mode_t mode = 0600) {
    directories(path.parent_path());
    FD f(::open(path.c_str(), O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, mode));
    std::size_t offset = 0;
    while (offset < bytes.size()) {
        auto n = ::write(f.fd, bytes.data() + offset, bytes.size() - offset);
        if (n < 0 && errno == EINTR) continue;
        require(n > 0, "local_failure"); offset += static_cast<std::size_t>(n);
    }
    require(::fsync(f.fd) == 0, "local_failure");
}
void atomic(const fs::path& path, const Json& value) {
    auto tmp = path; tmp += ".tmp-" + std::to_string(::getpid());
    write_file(tmp, value.dump() + "\n");
    require(::rename(tmp.c_str(), path.c_str()) == 0, "local_failure");
    sync_dir(path.parent_path());
}
bool relative(const std::string& p) {
    if (p.empty() || p.size() > 4096 || fs::path(p).is_absolute() || fs::path(p).lexically_normal().string() != p) return false;
    for (const auto& part : fs::path(p)) if (part == "." || part == ".." || part.empty()) return false;
    return std::all_of(p.begin(), p.end(), [](unsigned char c) { return c >= 32 && c < 127 && c != '\\'; });
}
void make_private(const fs::path& p) {
    if (::mkdir(p.c_str(), 0700) != 0) require(errno == EEXIST, "local_failure");
    struct stat s{};
    require(::lstat(p.c_str(), &s) == 0 && S_ISDIR(s.st_mode) && s.st_uid == ::geteuid() && (s.st_mode & 0077) == 0, "integrity_failure");
}
Json initial() { return Json{{"protocol", state_protocol}, {"generation", "0"}, {"selected", nullptr}, {"installations", Json::object()}}; }
Json journal(const fs::path& path, const std::string& expected) {
    auto j = parse(read_file(path, frame_limit));
    require(j.is_object() && j.size() == 4 && j.contains("protocol") && j.contains("plan_digest") && j.contains("stage") && j.contains("effect_state"), "integrity_failure");
    require(j.at("protocol") == "symphony.sov.cloudflare-attempt.v1" && j.at("plan_digest") == expected, "integrity_failure");
    const std::set<std::string> stages{"prepared", "admitted", "dispatch_started", "observed", "settled"};
    const std::set<std::string> effects{"not_applied", "applied_verified", "unresolved"};
    require(stages.contains(str(j.at("stage"))) && effects.contains(str(j.at("effect_state"))), "integrity_failure");
    return j;
}
struct Store {
    fs::path path;
    FD lock;
    explicit Store(const Json& p, bool create = false) : path(str(p)), lock(open_store(path, create)) {}
    static int open_store(const fs::path& path, bool create) {
        require(::geteuid() != 0 && ::getuid() == ::geteuid(), "unavailable_contract");
        directories(path.parent_path());
        if (create) {
            make_private(path);
            for (const auto* name : {"plans", "attempts", "packages", "staging", "trash"}) make_private(path / name);
        } else {
            directories(path);
            struct stat root{};
            require(::lstat(path.c_str(), &root) == 0 && root.st_uid == ::geteuid() && (root.st_mode & 0077) == 0, "integrity_failure");
        }
        auto fd = ::open((path / "lock").c_str(), O_RDWR | (create ? O_CREAT : 0) | O_NOFOLLOW | O_CLOEXEC, 0600);
        require(fd >= 0, "local_failure");
        struct stat s{};
        if (::fstat(fd, &s) != 0 || !S_ISREG(s.st_mode) || s.st_nlink != 1 || s.st_uid != ::geteuid() || (s.st_mode & 0077)) { ::close(fd); throw Failure("integrity_failure"); }
        if (::flock(fd, LOCK_EX | LOCK_NB) != 0) { ::close(fd); throw Failure("state_conflict"); }
        return fd;
    }
    Json state() const {
        if (!fs::exists(path / "state.json")) return initial();
        auto j = parse(read_file(path / "state.json", frame_limit));
        require(j.is_object() && j.size() == 4 && j.at("protocol") == state_protocol && j.at("installations").is_object(), "integrity_failure");
        require(std::regex_match(str(j.at("generation")), std::regex("0|[1-9][0-9]{0,18}")), "integrity_failure");
        require(j.at("selected").is_null() || j.at("installations").contains(str(j.at("selected"))), "integrity_failure");
        return j;
    }
    fs::path record(const char* kind, const std::string& d) const {
        require(std::regex_match(d, std::regex("sha256:[0-9a-f]{64}")));
        return path / kind / (d.substr(7) + ".json");
    }
    fs::path package(const std::string& d) const {
        require(std::regex_match(d, std::regex("sha256:[0-9a-f]{64}")));
        return path / "packages" / d.substr(7);
    }
};
Json bundle(const fs::path& path, const std::string& expected) {
    directories(path);
    auto manifest = parse(read_file(path / "bundle.json", frame_limit));
    validate(manifest, owner().at("bundle_schema"));
    require(digest(manifest, bundle_protocol) == expected, "binding_mismatch");
#ifdef __APPLE__
    require(manifest.at("os") == "macos", "unavailable_contract");
#else
    require(manifest.at("os") == "linux", "unavailable_contract");
#endif
#if defined(__aarch64__) || defined(__arm64__)
    require(manifest.at("architecture") == "arm64", "unavailable_contract");
#else
    require(manifest.at("architecture") == "amd64", "unavailable_contract");
#endif
    std::set<std::string> seen;
    for (const auto& f : manifest.at("files")) {
        auto p = str(f.at("path")); require(relative(p) && p != "bundle.json" && seen.insert(p).second, "integrity_failure");
        auto content = read_file(path / p, 512ULL * 1024 * 1024);
        require(content.size() == f.at("size").get<std::uint64_t>() && symphony::knowledge::engine::tagged_sha256(content) == str(f.at("digest")), "integrity_failure");
        struct stat s{}; require(::lstat((path / p).c_str(), &s) == 0 && (s.st_mode & 0022) == 0, "integrity_failure");
        require(((s.st_mode & 0111) != 0) == f.at("executable").get<bool>(), "integrity_failure");
    }
    std::set<std::string> actual;
    for (const auto& entry : fs::recursive_directory_iterator(path)) {
        auto status = entry.symlink_status();
        require(fs::is_directory(status) || fs::is_regular_file(status), "integrity_failure");
        if (fs::is_regular_file(status) && entry.path() != path / "bundle.json") actual.insert(entry.path().lexically_relative(path).generic_string());
    }
    require(seen == actual && seen.contains("node/bin/node") && seen.contains("cf/node_modules/cf/bin/cf") && seen.contains("cf/package-lock.json") && seen.contains("node/LICENSE"), "integrity_failure");
    auto package = parse(read_file(path / "cf/node_modules/cf/package.json", frame_limit));
    require(package.at("version") == manifest.at("cf_version"), "integrity_failure");
    return manifest;
}
Json installed(Store& store, const std::string& id) {
    auto state = store.state();
    require(state.at("installations").contains(id) && !state.at("installations").at(id).at("removed").get<bool>(), "state_conflict");
    return bundle(store.package(id), id);
}
void copy_bundle(const fs::path& source, const fs::path& target, const Json& manifest) {
    make_private(target);
    for (const auto& f : manifest.at("files")) {
        auto rel = fs::path(str(f.at("path")));
        auto dir = target;
        for (const auto& part : rel.parent_path()) { dir /= part; make_private(dir); }
        auto content = read_file(source / rel, 512ULL * 1024 * 1024);
        require(symphony::knowledge::engine::tagged_sha256(content) == str(f.at("digest")), "integrity_failure");
        write_file(target / rel, content, f.at("executable").get<bool>() ? 0700 : 0600);
    }
    write_file(target / "bundle.json", manifest.dump());
    // fsync directories bottom-up so publication follows durable contents.
    std::vector<fs::path> dirs{target};
    for (const auto& e : fs::recursive_directory_iterator(target)) if (e.is_directory()) dirs.push_back(e.path());
    std::sort(dirs.rbegin(), dirs.rend()); for (const auto& d : dirs) sync_dir(d);
}
const Json& descriptor(const Json& declaration, const std::string& op) {
    for (const auto& d : declaration.at("operations")) if (d.at("operation") == op) return d;
    throw Failure("unavailable_contract");
}
Json plan(Store& store, const Json& input, const Json& request) {
    auto before = store.state();
    auto before_digest = digest(before, state_protocol);
    require(input.at("expected_state").is_null() ? before == initial() : input.at("expected_state") == before_digest, "state_conflict");
    auto action = str(input.at("action"));
    bool imports = action == "install" || action == "update";
    Json manifest = nullptr;
    std::string id;
    if (imports) {
        require(input.at("installation_id").is_null() && !input.at("bundle_path").is_null() && !input.at("bundle_digest").is_null());
        id = str(input.at("bundle_digest")); manifest = bundle(str(input.at("bundle_path")), id);
        require(!before.at("installations").contains(id), "state_conflict");
    } else {
        require(input.at("bundle_path").is_null() && input.at("bundle_digest").is_null());
        id = str(input.at("installation_id")); manifest = installed(store, id);
    }
    auto after = before;
    auto generation = std::stoull(str(before.at("generation")));
    require(generation < 999999999999999999ULL, "state_conflict");
    after["generation"] = std::to_string(generation + 1);
    if (imports) after["installations"][id] = Json{{"removed", false}, {"cf_version", manifest.at("cf_version")}, {"node_version", manifest.at("node_version")}};
    if (action == "select" || action == "update" || action == "rollback") after["selected"] = id;
    if (action == "remove") { require(before.at("selected") != id, "state_conflict"); after["installations"][id]["removed"] = true; }
    Json p{{"protocol", plan_protocol}, {"action", action}, {"input", input}, {"uid", std::to_string(::geteuid())}, {"descriptor_ref", request.at("descriptor_ref")}, {"installation_ref", request.at("installation_ref")}, {"before", before}, {"after", after}, {"toolchain_id", id}, {"expires_unix_ms", std::to_string(symphony::knowledge::engine::unix_time_ms() + 3600000)}, {"provider_effects", "none"}};
    auto d = digest(p, plan_protocol); auto path = store.record("plans", d);
    if (!fs::exists(path)) { write_file(path, p.dump()); sync_dir(path.parent_path()); }
    return Json{{"plan_digest", d}, {"plan", p}, {"state_digest", before_digest}};
}
Json load_plan(Store& store, const std::string& d, const Json& request) {
    auto p = parse(read_file(store.record("plans", d), frame_limit));
    require(digest(p, plan_protocol) == d && p.at("protocol") == plan_protocol && p.at("uid") == std::to_string(::geteuid()) && p.at("input").at("store") == store.path.string() && p.at("descriptor_ref") == request.at("descriptor_ref") && p.at("installation_ref") == request.at("installation_ref"), "binding_mismatch");
    return p;
}
void no_unsettled(Store& store) {
    resource_barrier(store.path);
    connector_barrier(store.path);
    for (const auto& e : fs::directory_iterator(store.path / "attempts")) {
        if (e.path().extension() != ".json") continue; // Unpublished atomic-write temporaries are not journals.
        auto j = journal(e.path(), "sha256:" + e.path().stem().string());
        require(j.at("stage") == "settled", "unresolved_attempt");
    }
}
Json lifecycle(Store& store, const Json& request, bool& started) {
    const auto& input = request.at("input"); auto op = str(request.at("operation"));
    if (op == "installation.list") { auto s = store.state(); return Json{{"state", s}, {"state_digest", digest(s, state_protocol)}, {"completeness", "complete_local_store"}}; }
    if (op == "installation.inspect") return Json{{"toolchain_id", input.at("installation_id")}, {"bundle", installed(store, str(input.at("installation_id")))}, {"integrity", "verified"}};
    if (op == "installation.plan") return plan(store, input, request);
    auto d = str(input.at("plan_digest")); auto p = load_plan(store, d, request);
    auto journal_path = store.record("attempts", d);
    Json journal = fs::exists(journal_path) ? cloudflare::journal(journal_path, d) : Json(nullptr);
    if (op == "installation.status") return Json{{"plan_digest", d}, {"attempt", journal}, {"current_state_digest", digest(store.state(), state_protocol)}};
    require(request.at("plan_ref") == d, "binding_mismatch");
    if (op == "installation.reconcile") {
        require(!journal.is_null(), "state_conflict");
        auto state = store.state();
        started = true; // Reconciliation may reclaim owned scratch or settle intent.
        if (state == p.at("after")) {
            auto id = str(p.at("toolchain_id"));
            if (p.at("action") != "remove") static_cast<void>(installed(store, id));
            else {
                require(!fs::exists(store.package(id)), "integrity_failure");
                auto trash = store.path / "trash" / d.substr(7);
                if (fs::exists(trash)) { static_cast<void>(bundle(trash, id)); fs::remove_all(trash); sync_dir(store.path / "trash"); }
            }
            journal["stage"] = "settled"; journal["effect_state"] = "applied_verified";
            atomic(journal_path, journal);
        } else if (journal.at("stage") == "prepared" || journal.at("stage") == "admitted") {
            require(state == p.at("before"), "state_conflict");
            journal["stage"] = "settled"; journal["effect_state"] = "not_applied"; atomic(journal_path, journal);
        } else if (state == p.at("before") && journal.at("stage") == "dispatch_started") {
            // Local-only recovery never reruns a binary or repeats installation.
            // With the old state still authoritative, reclaim an unselected copy
            // or restore the exactly retired package, then settle no net effect.
            auto action = str(p.at("action")); auto id = str(p.at("toolchain_id"));
            if (action == "install" || action == "update") {
                require(!state.at("installations").contains(id), "unresolved_attempt");
                if (fs::exists(store.package(id))) { static_cast<void>(bundle(store.package(id), id)); fs::remove_all(store.package(id)); sync_dir(store.path / "packages"); }
                auto stage = store.path / "staging" / d.substr(7);
                if (fs::exists(stage)) {
                    directories(stage); struct stat st{};
                    require(::lstat(stage.c_str(), &st) == 0 && st.st_uid == ::geteuid() && (st.st_mode & 0077) == 0, "integrity_failure");
                    fs::remove_all(stage); sync_dir(store.path / "staging");
                }
            } else if (action == "remove") {
                auto trash = store.path / "trash" / d.substr(7);
                if (fs::exists(trash)) {
                    require(!fs::exists(store.package(id)), "unresolved_attempt"); static_cast<void>(bundle(trash, id));
                    require(::rename(trash.c_str(), store.package(id).c_str()) == 0, "local_failure"); sync_dir(store.path / "packages"); sync_dir(store.path / "trash");
                }
                static_cast<void>(installed(store, id));
            }
            journal["stage"] = "settled"; journal["effect_state"] = "not_applied"; atomic(journal_path, journal);
        } else require(journal.at("stage") == "settled" && (state == p.at("before") || state == p.at("after")), "unresolved_attempt");
        return Json{{"plan_digest", d}, {"attempt", journal}};
    }
    require(journal.is_null(), "state_conflict"); no_unsettled(store);
    require(std::stoll(str(p.at("expires_unix_ms"))) > symphony::knowledge::engine::unix_time_ms(), "state_conflict");
    require(store.state() == p.at("before"), "state_conflict");
    auto action = str(p.at("action")); auto id = str(p.at("toolchain_id"));
    Json manifest = nullptr;
    if (action == "install" || action == "update") manifest = bundle(str(p.at("input").at("bundle_path")), id);
    else static_cast<void>(installed(store, id));
    journal = Json{{"protocol", "symphony.sov.cloudflare-attempt.v1"}, {"plan_digest", d}, {"stage", "prepared"}, {"effect_state", "not_applied"}};
    atomic(journal_path, journal); journal["stage"] = "admitted"; atomic(journal_path, journal);
    journal["stage"] = "dispatch_started"; journal["effect_state"] = "unresolved"; atomic(journal_path, journal); started = true;
    if (action == "install" || action == "update") {
        auto stage = store.path / "staging" / d.substr(7);
        require(!fs::exists(stage) && !fs::exists(store.package(id)), "state_conflict");
        copy_bundle(str(p.at("input").at("bundle_path")), stage, manifest);
        static_cast<void>(bundle(stage, id));
        require(::rename(stage.c_str(), store.package(id).c_str()) == 0, "local_failure");
        sync_dir(store.path / "packages"); sync_dir(store.path / "staging");
    } else if (action == "remove") {
        auto trash = store.path / "trash" / d.substr(7);
        require(!fs::exists(trash) && ::rename(store.package(id).c_str(), trash.c_str()) == 0, "local_failure");
        sync_dir(store.path / "packages"); sync_dir(store.path / "trash");
    }
    atomic(store.path / "state.json", p.at("after"));
    require(store.state() == p.at("after"), "integrity_failure");
    journal["stage"] = "observed"; journal["effect_state"] = "applied_verified"; atomic(journal_path, journal);
    // Removal reclaims only this attempt's exact retired package, never provider resources.
    if (action == "remove") { fs::remove_all(store.path / "trash" / d.substr(7)); sync_dir(store.path / "trash"); }
    journal["stage"] = "settled"; atomic(journal_path, journal);
    return Json{{"plan_digest", d}, {"attempt", journal}, {"state", p.at("after")}};
}

Json vendor(Store& store, const std::string& id, const std::vector<std::string>& args, bool& started) {
    auto qualified = owner().at("qualified_offline_toolchains");
    require(std::find(qualified.begin(), qualified.end(), Json(id)) != qualified.end(), "unavailable_contract");
    auto manifest = installed(store, id); auto package = store.package(id);
    auto scratch = store.path / "staging" / ("probe-" + std::to_string(::getpid()));
    require(!fs::exists(scratch), "state_conflict"); make_private(scratch);
    struct Cleanup { fs::path p; ~Cleanup() { std::error_code ec; fs::remove_all(p, ec); } } cleanup{scratch};
    std::vector<std::string> argv{(package / "node/bin/node").string(), "--permission", "--allow-fs-read=" + package.string(), "--allow-fs-read=" + scratch.string(), (package / "cf/node_modules/cf/bin/cf").string()};
    argv.insert(argv.end(), args.begin(), args.end());
    // The selected Node denies network, writes, subprocesses and addons. cf's
    // audited delegation sentinel prevents searching ambient project packages.
    std::vector<std::string> environment{"PATH=/usr/bin:/bin", "HOME=" + scratch.string(), "XDG_CONFIG_HOME=" + scratch.string(), "XDG_CACHE_HOME=" + scratch.string(), "TMPDIR=" + scratch.string(), "CI=1", "CF_SEND_TELEMETRY=false", "DO_NOT_TRACK=1", "CF_DELEGATION=1", "NO_COLOR=1", "LANG=C", "LC_ALL=C"};
    std::vector<char*> av, ev; for (auto& a : argv) av.push_back(a.data()); av.push_back(nullptr); for (auto& e : environment) ev.push_back(e.data()); ev.push_back(nullptr);
    int pipefd[2]; require(::pipe(pipefd) == 0, "local_failure"); FD reader(pipefd[0]), writer(pipefd[1]);
    auto child = ::fork(); require(child >= 0, "local_failure");
    if (child == 0) {
        // Inherit qxctl's cancellation group: cancelling the adapter kills Node too.
        ::dup2(writer.fd, STDOUT_FILENO); int nullfd = ::open("/dev/null", O_RDWR);
        ::dup2(nullfd, STDERR_FILENO); ::dup2(nullfd, STDIN_FILENO);
        ::close(reader.fd); ::close(writer.fd); ::close(nullfd);
        if (::chdir(scratch.c_str()) != 0) _exit(126);
        ::execve(av[0], av.data(), ev.data()); _exit(127);
    }
    started = true;
    ::close(writer.fd); writer.fd = -1;
    require(::fcntl(reader.fd, F_SETFL, O_NONBLOCK) == 0, "local_failure");
    auto deadline = std::chrono::steady_clock::now() + std::chrono::seconds(20);
    std::string output; bool eof = false; int status = 0; bool done = false;
    while (!eof || !done) {
        if (output.size() >= frame_limit / 2 || std::chrono::steady_clock::now() >= deadline) { ::kill(child, SIGKILL); ::waitpid(child, &status, 0); throw Failure("child_failure"); }
        pollfd pfd{reader.fd, POLLIN, 0}; ::poll(&pfd, 1, 10);
        char buf[16384]; auto n = ::read(reader.fd, buf, sizeof(buf));
        if (n > 0) output.append(buf, static_cast<std::size_t>(n)); else if (n == 0) eof = true;
        if (!done) { auto r = ::waitpid(child, &status, WNOHANG); if (r == child) done = true; }
    }
    require(WIFEXITED(status) && WEXITSTATUS(status) == 0, "child_failure");
    // Verify the full dependency closure again after the read-only child exits.
    static_cast<void>(installed(store, id));
    auto value = parse(output);
    if (args.at(0) == "cli") {
        require(value.is_array() && value.size() <= 5, "child_failure");
        for (const auto& row : value) require(row.is_object() && row.size() == 2 && row.at("command").is_string() && row.at("summary").is_string(), "child_failure");
    } else require(value.is_object() && value.contains("operationId") && value.contains("httpMethod") && value.contains("path"), "child_failure");
    return Json{{"vendor_result", value}, {"toolchain_id", id}, {"completeness", args.at(0) == "cli" ? "bounded_search_not_inventory" : "generated_schema_may_be_incomplete"}, {"provider_dispatch", false}};
}
Json context_state(Store& store) {
    auto path=store.path/"context.json";
    auto state=fs::exists(path)?parse(read_file(path,frame_limit)):Json{{"generation","0"},{"context",nullptr}};
    require(state.is_object()&&state.size()==2&&state.contains("context")&&state.contains("generation"));
    require(std::regex_match(str(state.at("generation")),std::regex("0|[1-9][0-9]{0,18}")));
    if(!state.at("context").is_null())validate(state.at("context"),descriptor(owner(),"context.plan").at("input_schema").at("properties").at("context"));
    return state;
}
Json context_admin(Store& store,const Json& request,bool& started) {
    const auto& in=request.at("input");auto op=str(request.at("operation"));auto state=context_state(store);
    if(op=="context.inspect"||op=="context.plan") {
        no_unsettled(store);auto c=in.at("context");require(c.at("principal_uid")==std::to_string(::geteuid()),"binding_mismatch");
        static_cast<void>(installed(store,str(c.at("toolchain_id"))));auto cd=digest(c,"symphony.sov.cloudflare-context.v1");
        require(in.at("expected_context_ref").is_null()||in.at("expected_context_ref")==cd,"binding_mismatch");
        Json p=nullptr,pd=nullptr;
        if(op=="context.plan") {
            auto generation=std::stoull(str(state.at("generation")));require(generation<999999999999999999ULL);
            p=Json{{"protocol","symphony.sov.cloudflare-context-plan.v1"},{"uid",std::to_string(::geteuid())},{"request_id",request.at("request_id")},{"descriptor_ref",request.at("descriptor_ref")},{"installation_ref",request.at("installation_ref")},{"store",store.path.string()},{"before",state},{"after",Json{{"generation",std::to_string(generation+1)},{"context",c}}},{"expires_unix_ms",std::to_string(symphony::ssiag::native::now()+3600000)}};
            pd=digest(p,"symphony.sov.cloudflare-context-plan.v1");auto path=store.record("plans",str(pd));if(!fs::exists(path))write_file(path,p.dump());sync_dir(path.parent_path());
        }
        return Json{{"context",c},{"context_ref",cd},{"selected",state},{"plan",p},{"plan_digest",pd},{"authority_readiness","nonroot_local_owner"},{"provider_dispatch",false},{"persistence",op=="context.plan"?"persisted_plan":"inspection"}};
    }
    auto d=str(in.at("plan_digest"));require(request.at("plan_ref")==d,"binding_mismatch");auto p=parse(read_file(store.record("plans",d),frame_limit));
    validate(p,descriptor(owner(),"context.plan").at("data_schema").at("properties").at("plan"));
    require(digest(p,"symphony.sov.cloudflare-context-plan.v1")==d&&p.at("uid")==std::to_string(::geteuid())&&p.at("store")==store.path.string()&&p.at("descriptor_ref")==request.at("descriptor_ref")&&p.at("installation_ref")==request.at("installation_ref"),"binding_mismatch");
    auto path=store.record("attempts",d);auto j=fs::exists(path)?journal(path,d):Json(nullptr);
    if(op=="context.apply") {
        no_unsettled(store);require(j.is_null()&&state==p.at("before"),"state_conflict");require(std::stoll(str(p.at("expires_unix_ms")))>symphony::ssiag::native::now(),"state_conflict");
        static_cast<void>(installed(store,str(p.at("after").at("context").at("toolchain_id"))));
        j=Json{{"protocol","symphony.sov.cloudflare-attempt.v1"},{"plan_digest",d},{"stage","prepared"},{"effect_state","not_applied"}};atomic(path,j);j["stage"]="dispatch_started";j["effect_state"]="unresolved";atomic(path,j);started=true;
        atomic(store.path/"context.json",p.at("after"));state=context_state(store);require(state==p.at("after"),"state_conflict");j["stage"]="settled";j["effect_state"]="applied_verified";atomic(path,j);
    } else if(op=="context.reconcile") {
        require(!j.is_null(),"state_conflict");require(state==p.at("before")||state==p.at("after"),"unresolved_attempt");started=true;j["stage"]="settled";j["effect_state"]=state==p.at("after")?"applied_verified":"not_applied";atomic(path,j);
    }
    return Json{{"plan_digest",d},{"attempt",j},{"selected",state},{"provider_dispatch",false}};
}

}

Json parse(const std::string& text) { return symphony::knowledge::engine::parse_bounded_json(text, frame_limit, 500000); }
std::string digest(const Json& value, const std::string& domain) { return symphony::knowledge::engine::tagged_sha256(domain + '\0' + value.dump()); }
std::string file_digest(const fs::path& path) { return symphony::knowledge::engine::tagged_sha256(read_file(path, 512ULL * 1024 * 1024)); }
Json owner() { static const auto o = parse(reinterpret_cast<const char*>(owner_json)); return o; }
void validate(const Json& value, const Json& schema) {
    if (schema.contains("oneOf")) { int matches = 0; for (const auto& s : schema.at("oneOf")) { try { validate(value, s); ++matches; } catch (...) {} } require(matches == 1); return; }
    if (schema.contains("const")) require(value == schema.at("const"));
    if (schema.contains("enum")) require(std::find(schema.at("enum").begin(), schema.at("enum").end(), value) != schema.at("enum").end());
    if (!schema.contains("type")) return;
    auto t = str(schema.at("type"));
    if (t == "null") require(value.is_null());
    else if (t == "boolean") require(value.is_boolean());
    else if (t == "string") {
        auto v = str(value); require(v.size() <= schema.value("maxLength", 65536U));
        if (schema.contains("pattern")) require(std::regex_match(v, std::regex(str(schema.at("pattern")))));
    } else if (t == "integer") {
        require(value.is_number_integer() && !value.is_number_float());
        if (schema.contains("minimum")) require(value >= schema.at("minimum"));
        if (schema.contains("maximum")) require(value <= schema.at("maximum"));
    } else if (t == "array") {
        require(value.is_array() && value.size() >= schema.value("minItems", 0U) && value.size() <= schema.value("maxItems", 500000U));
        if (schema.contains("items")) for (const auto& v : value) validate(v, schema.at("items"));
    } else if (t == "object") {
        require(value.is_object());
        if (schema.contains("properties")) {
            for (const auto& k : schema.at("required")) require(value.contains(str(k)));
            for (const auto& [key, v] : value.items()) { require(schema.at("properties").contains(key)); validate(v, schema.at("properties").at(key)); }
        }
    } else throw Failure("invalid_control");
}

Json process(const Json& request, const fs::path& executable) {
    auto declaration = owner(); validate(request, declaration.at("request_schema"));
    Json result{{"protocol", "symphony.sov.cloudflare-result.v1"}, {"request_id", request.at("request_id")}, {"correlation_id", request.at("correlation_id")}, {"operation", request.at("operation")}, {"request_ref", digest(request, "symphony.sov.cloudflare-request.v1")}, {"descriptor_ref", request.at("descriptor_ref")}, {"installation_ref", request.at("installation_ref")}, {"context_ref", request.at("context_ref")}, {"plan_ref", request.at("plan_ref")}, {"dispatch_state", "not_dispatched"}, {"outcome", "completed"}, {"effect_state", "not_applicable"}, {"error_code", nullptr}, {"authority_ref", "os-uid:" + std::to_string(::geteuid())}, {"audit_ref", nullptr}, {"evidence", Json::array()}, {"artifacts", Json::array()}, {"next_action", nullptr}, {"data", nullptr}};
    bool started = false;
    const auto operation_name=str(request.at("operation"));
    const bool mutation=operation_name.ends_with(".apply")||operation_name.ends_with(".reconcile");
    try {
        require(request.at("descriptor_ref") == digest(declaration, "symphony.sov.cloudflare-owner.v1") && request.at("installation_ref").at("executable_digest") == file_digest(executable), "binding_mismatch");
        auto op = str(request.at("operation")); const auto& d = descriptor(declaration, op); const auto& input = request.at("input");
        validate(input, d.at("input_schema")); require(d.at("implemented").get<bool>(), "unavailable_contract");
        if (!op.starts_with("resource.") || op == "resource.plan") require(request.at("context_ref").is_null(), "binding_mismatch");
        if (!op.ends_with(".apply") && !op.ends_with(".reconcile") && !op.ends_with(".status") && (!op.starts_with("resource.") || op == "resource.plan")) require(request.at("plan_ref").is_null(), "binding_mismatch");
        if(op.starts_with("connector.")){
            Store store(input.at("store"),true);result["data"]=connector_admin(request,executable,result,started);
        } else if (op.starts_with("installation.")) {
            Store store(input.at("store"), op == "installation.plan"); result["data"] = lifecycle(store, request, started);
            if (op == "installation.plan") result["next_action"] = "inspect_plan";
            if (op == "installation.apply" || op == "installation.reconcile") result["effect_state"] = result.at("data").at("attempt").at("effect_state");
        } else if (op == "resource.plan") {
            Store store(input.at("store")); no_unsettled(store);
            const auto context = input.at("context");
            require(context.at("principal_uid") == std::to_string(::geteuid()), "binding_mismatch");
            require(input.at("expected_context_ref") == digest(context, "symphony.sov.cloudflare-context.v1"), "binding_mismatch");
            const auto state = store.state();
            require(input.at("expected_state") == digest(state, state_protocol), "state_conflict");
            require(state.at("selected") == context.at("toolchain_id"), "binding_mismatch");
            static_cast<void>(installed(store, str(context.at("toolchain_id"))));
            Json binding = nullptr;
            if (!input.at("execution").is_null()) {
                auto qualified = declaration.at("qualified_resource_toolchains");
                require(std::find(qualified.begin(), qualified.end(), context.at("toolchain_id")) != qualified.end(), "unavailable_contract");
                binding = execution_binding(input.at("execution"));
                require(binding.at("config").at("consumer_digest") == request.at("installation_ref").at("executable_digest") && binding.at("config").at("toolchain_digest") == context.at("toolchain_id"), "binding_mismatch");
                require(!context.at("credential_ref").is_null(), "unavailable_contract");
            }
            const auto preview = resource_preview(input);
            const auto now = std::chrono::duration_cast<std::chrono::milliseconds>(std::chrono::system_clock::now().time_since_epoch()).count();
            Json p{{"protocol", "symphony.sov.cloudflare-resource-plan.v1"}, {"request_id", request.at("request_id")}, {"uid", std::to_string(::geteuid())}, {"descriptor_ref", request.at("descriptor_ref")}, {"installation_ref", request.at("installation_ref")}, {"input", input}, {"preview", preview}, {"execution_binding", binding}, {"expires_unix_ms", std::to_string(now + 3600000)}, {"provider_dispatch", false}, {"execution_available", !binding.is_null()}, {"authority_readiness", binding.is_null() ? "unavailable_contract" : "requires_ssiag_admission"}};
            const auto d = digest(p, "symphony.sov.cloudflare-resource-plan.v1");
            const auto record = store.record("plans", d);
            if (fs::exists(record)) require(parse(read_file(record, frame_limit)) == p, "binding_mismatch");
            else { write_file(record, p.dump()+"\n"); sync_dir(store.path / "plans"); }
            result["data"] = Json{{"plan", p}, {"plan_digest", d}, {"provider_dispatch", false}, {"authority_readiness", p.at("authority_readiness")}, {"execution_available", p.at("execution_available")}};
            result["next_action"] = "inspect_plan";
        } else if (op.starts_with("resource.")) {
            Store store(input.at("store")); auto d = str(input.at("plan_digest"));
            require(request.at("plan_ref") == d && request.at("context_ref") == input.at("context_ref"), "binding_mismatch");
            auto p = parse(read_file(store.record("plans", d), frame_limit));
            validate(p, descriptor(declaration, "resource.plan").at("data_schema").at("properties").at("plan"));
            require(digest(p, "symphony.sov.cloudflare-resource-plan.v1") == d && p.at("uid") == std::to_string(::geteuid()) && p.at("input").at("store") == input.at("store"), "binding_mismatch");
            require(input.at("context_ref") == digest(p.at("input").at("context"), "symphony.sov.cloudflare-context.v1"), "binding_mismatch");
            if (op != "resource.status") {
                if(op!="resource.reconcile")no_unsettled(store);
                require(p.at("descriptor_ref") == request.at("descriptor_ref") && p.at("installation_ref") == request.at("installation_ref"), "binding_mismatch");
                require(store.state().at("selected") == p.at("input").at("context").at("toolchain_id") && p.at("input").at("expected_state") == digest(store.state(), state_protocol), "state_conflict");
                static_cast<void>(installed(store, str(p.at("input").at("context").at("toolchain_id"))));
                require(p.at("execution_binding") == execution_binding(p.at("input").at("execution")), "binding_mismatch");
            }
            result["data"] = execute_resource(request, p, store.package(str(p.at("input").at("context").at("toolchain_id"))), result, started);
            if(op!="resource.status")static_cast<void>(installed(store,str(p.at("input").at("context").at("toolchain_id"))));
        } else if (op.starts_with("context.")) {
            Store store(input.at("store")); result["data"]=context_admin(store,request,started);
            if(mutation) result["effect_state"]=result.at("data").at("attempt").at("effect_state");
        } else if (op == "capability.list") {
            if (input.at("query").is_null()) {
                require(input.at("store").is_null() && input.at("toolchain_id").is_null());
                result["data"] = Json{{"owner", declaration}, {"completeness", "complete_adapter_catalogue"}};
            } else { Store store(input.at("store")); result["data"] = vendor(store, str(input.at("toolchain_id")), {"cli", "search", str(input.at("query"))}, started); }
        } else {
            auto selected = str(input.at("operation"));
            if (op == "capability.schema" && input.at("kind") == "vendor") {
                const Json* candidate = nullptr;
                for (const auto& c : declaration.at("vendor_candidates")) if (c.at("operation") == selected && c.at("vendor_available") == true) candidate = &c;
                require(candidate != nullptr, "unavailable_contract"); Store store(input.at("store"));
                result["data"] = vendor(store, str(input.at("toolchain_id")), candidate->at("argv").get<std::vector<std::string>>(), started);
            } else {
                const auto& item = descriptor(declaration, selected);
                if (op == "capability.describe") result["data"] = Json{{"descriptor", item}};
                else if (op == "capability.template") result["data"] = Json{{"input", item.at("template")}};
                else {
                    require(input.at("store").is_null() && input.at("toolchain_id").is_null()); auto kind = str(input.at("kind"));
                    auto schema = kind == "input" ? item.at("input_schema") : declaration.at(kind + "_schema");
                    if (kind == "request" || kind == "result") {
                        schema["properties"]["operation"] = Json{{"const", selected}};
                        if (kind == "request") schema["properties"]["input"] = item.at("input_schema");
                        else schema["properties"]["data"] = item.at("implemented").get<bool>() ? Json{{"oneOf", Json::array({Json{{"type", "null"}}, item.at("data_schema")})}} : Json{{"type", "null"}};
                    }
                    result["data"] = Json{{"schema", schema}};
                }
            }
        }
        if (started) result["dispatch_state"] = "response_received";
    } catch (const Failure& e) {
        if (std::string(e.what()) == "unresolved_attempt") started = true;
        result["outcome"] = started ? (mutation ? "unknown" : "failed") : "rejected"; result["effect_state"] = started && mutation ? "unresolved" : "not_applied";
        result["error_code"] = e.what(); result["dispatch_state"] = started ? "dispatch_started" : "not_dispatched";
        result["next_action"] = started || std::string(e.what()) == "unresolved_attempt" ? "reconcile" : "new_plan";
    } catch (...) {
        result["outcome"] = started ? (mutation ? "unknown" : "failed") : "rejected"; result["effect_state"] = started && mutation ? "unresolved" : "not_applied";
        result["error_code"] = "invalid_control"; result["dispatch_state"] = started ? "dispatch_started" : "not_dispatched"; result["next_action"] = started ? "reconcile" : "new_plan";
    }
    validate(result, declaration.at("result_schema"));
    if (result.at("outcome") == "completed") validate(result.at("data"), descriptor(declaration, str(request.at("operation"))).at("data_schema"));
    return result;
}
}
