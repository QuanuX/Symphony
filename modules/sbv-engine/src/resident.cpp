#include "resource_admission.hpp"
#include "dataset.hpp"
#include <algorithm>
#include <array>
#include <atomic>
#include <cerrno>
#include <cstring>
#include <dlfcn.h>
#include <fcntl.h>
#include <filesystem>
#include <mutex>
#include <map>
#include <poll.h>
#include <spawn.h>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>
#include <sys/wait.h>
#include <thread>
#include <unistd.h>
extern char **environ;
namespace symphony::sbv::detail {
namespace {
struct FD {
  int n = -1;
  explicit FD(int v = -1) : n(v) {}
  FD(const FD &) = delete;
  FD(FD &&o) noexcept : n(o.n) { o.n = -1; }
  ~FD() {
    if (n >= 0)
      ::close(n);
  }
};
constexpr auto wire_limit = e::Limits::max_request_bytes;
void io_fail() {
  throw e::Error("sbv.resident_transport",
                 "resident connection failed or timed out; outcome may be "
                 "uncertain; inspect output before retrying",
                 4);
}
void ready(int fd, short events, std::int64_t end) {
  for (;;) {
    deadline(end);
    const auto left = end - e::unix_time_ms();
    if (left <= 0)
      io_fail();
    pollfd p{fd, events, 0};
    int n = ::poll(&p, 1, static_cast<int>(std::min<std::int64_t>(left, 1000)));
    if (n > 0) {
      if (p.revents & events)
        return;
      io_fail();
    }
    if (n < 0 && errno != EINTR)
      io_fail();
  }
}
void configure(int fd) {
  need(fd >= 0 && ::fcntl(fd, F_SETFD, FD_CLOEXEC) == 0 &&
           ::fcntl(fd, F_SETFL, O_NONBLOCK) == 0,
       "resident descriptor setup failed");
#ifdef SO_NOSIGPIPE
  int one = 1;
  need(::setsockopt(fd, SOL_SOCKET, SO_NOSIGPIPE, &one, sizeof(one)) == 0,
       "resident socket setup failed");
#endif
}
void transfer(int fd, char *p, std::size_t size, bool write, std::int64_t end) {
  while (size) {
    ready(fd, write ? POLLOUT : POLLIN, end);
    constexpr int send_flags =
#ifdef MSG_NOSIGNAL
        MSG_NOSIGNAL;
#else
        0;
#endif
    const auto n =
        write ? ::send(fd, p, size, send_flags) : ::recv(fd, p, size, 0);
    if (n < 0 && (errno == EINTR || errno == EAGAIN || errno == EWOULDBLOCK))
      continue;
    if (n <= 0)
      io_fail();
    p += n;
    size -= static_cast<std::size_t>(n);
  }
}
void send(int fd, const Json &j, std::int64_t end) {
  auto bytes = j.dump();
  need(bytes.size() <= wire_limit, "resident message too large");
  std::array<char, 4> h{};
  for (unsigned i = 0; i < 4; ++i)
    h[i] = static_cast<char>(bytes.size() >> (24 - 8 * i));
  transfer(fd, h.data(), h.size(), true, end);
  transfer(fd, bytes.data(), bytes.size(), true, end);
}
// A stop acknowledgement must never wait for the peer to read. A lost or
// partial receipt is an uncertain transport outcome, never permission to retry.
void final_send(int fd, const Json &j) noexcept {
  try {
    const auto bytes = j.dump();
    std::string frame(4, '\0');
    for (unsigned i = 0; i < 4; ++i)
      frame[i] = static_cast<char>(bytes.size() >> (24 - 8 * i));
    frame += bytes;
    constexpr int flags =
#ifdef MSG_NOSIGNAL
        MSG_NOSIGNAL;
#else
        0;
#endif
    (void)::send(fd, frame.data(), frame.size(), flags);
  } catch (...) {
  }
}
std::size_t control_nodes(const Json &j) {
  std::size_t count = 1;
  if (j.is_structured()) {
    if (j.is_object()) count += j.size();
    for (const auto &child : j) count += control_nodes(child);
  }
  return count;
}
bool job_identifier(const Json &j) {
  if (!j.is_string()) return false;
  const auto &s = j.get_ref<const std::string &>();
  return !s.empty() && s.size() <= 128 &&
         std::all_of(s.begin(), s.end(), [](unsigned char c) {
           return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
                  (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.';
         });
}
Json receive(int fd, std::int64_t end) {
  std::array<char, 4> h{};
  transfer(fd, h.data(), h.size(), false, end);
  std::size_t size = 0;
  for (auto c : h)
    size = (size << 8) | static_cast<unsigned char>(c);
  need(size > 0 && size <= wire_limit, "resident frame bound exceeded");
  std::string bytes(size, '\0');
  transfer(fd, bytes.data(), size, false, end);
  return e::parse_bounded_json(bytes, wire_limit, 1000000);
}
FD directory(const std::string &path) {
  need(path.starts_with('/') && e::is_safe_relative_path(path.substr(1)),
       "absolute no-follow resident directory required");
  FD current(::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC));
  need(current.n >= 0, "root directory unavailable");
  for (const auto &part : std::filesystem::path(path).relative_path()) {
    int next = ::openat(current.n, part.c_str(),
                        O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    need(next >= 0, "resident directory must exist without symlinks");
    ::close(current.n);
    current.n = next;
  }
  struct stat st{};
  need(::fstat(current.n, &st) == 0 && st.st_uid == ::geteuid() &&
           (st.st_mode & 0777) == 0700,
       "resident directory must be owned by caller with mode 0700");
  return current;
}
sockaddr_un address(const Json &p) {
  auto root = str(p.at("directory"));
  (void)directory(root);
  const auto id = str(p.at("instance_id"));
  need(id.size() == 32 && std::all_of(id.begin(), id.end(),
                                      [](char c) {
                                        return (c >= 'a' && c <= 'f') ||
                                               (c >= '0' && c <= '9');
                                      }),
       "32 lowercase hex instance id required");
  // Instance-scoped endpoint: a stale reference cannot name a later load.
  auto path = root + "/" + id + ".sock";
  sockaddr_un a{};
  a.sun_family = AF_UNIX;
  need(path.size() < sizeof(a.sun_path),
       "resident socket path too long; choose a short private directory");
  std::copy(path.begin(), path.end(), a.sun_path);
  return a;
}
void peer(int fd) {
#ifdef __APPLE__
  uid_t uid;
  gid_t gid;
  need(::getpeereid(fd, &uid, &gid) == 0 && uid == ::geteuid(),
       "resident peer owner mismatch");
#else
  ucred cred{};
  socklen_t size = sizeof(cred);
  need(::getsockopt(fd, SOL_SOCKET, SO_PEERCRED, &cred, &size) == 0 &&
           cred.uid == ::geteuid(),
       "resident peer owner mismatch");
#endif
}
std::int64_t wire_deadline(const Json &w) {
  if (w.at("deadline_ms").is_null())
    return e::no_deadline;
  const auto value = i64(w.at("deadline_ms"));
  need(value > 0 && value != e::no_deadline,
       "invalid finite resident deadline");
  return value;
}
Json wire(const std::string &op, const Json &p, std::int64_t end) {
  return {
      {"protocol", "symphony.sbv.resident-wire.v2"},
      {"engine_version", version},
      {"operation", op},
      {"input", p},
      {"deadline_ms", end == e::no_deadline ? Json(nullptr) : Json(dec(end))}};
}
void validate_wire(const Json &w) {
  keys(w, {"protocol", "engine_version", "operation", "input", "deadline_ms"});
  need(w.at("protocol") == "symphony.sbv.resident-wire.v2" &&
           w.at("engine_version") == version,
       "resident protocol/version mismatch");
  const auto end = wire_deadline(w);
  deadline(end);
}
Json unwrap(const Json &j) {
  need(j.is_object() && j.value("engine_version", "") == version,
       "resident response version mismatch");
  if (j.contains("error"))
    throw e::Error(str(j.at("error").at("code")),
                   str(j.at("error").at("message")), 4);
  keys(j, {"engine_version", "result"});
  return j.at("result");
}
Json success(Json j) {
  return {{"engine_version", version}, {"result", std::move(j)}};
}
Json failure(const std::string &code, const std::string &message) {
  return {{"engine_version", version},
          {"error", {{"code", code}, {"message", message}}}};
}
Json transact(const Json &w) {
  const auto &p = w.at("input");
  const auto end = wire_deadline(w);
  auto a = address(p);
  struct stat st{};
  need(::lstat(a.sun_path, &st) == 0 && S_ISSOCK(st.st_mode) &&
           st.st_uid == ::geteuid() && (st.st_mode & 0777) == 0600,
       "resident endpoint unavailable or unsafe; no file fallback");
  FD fd(::socket(AF_UNIX, SOCK_STREAM, 0));
  configure(fd.n);
  if (::connect(fd.n, reinterpret_cast<sockaddr *>(&a), sizeof(a)) != 0) {
    if (errno != EINPROGRESS)
      io_fail();
    ready(fd.n, POLLOUT, end);
    int err = 0;
    socklen_t size = sizeof(err);
    if (::getsockopt(fd.n, SOL_SOCKET, SO_ERROR, &err, &size) != 0 || err)
      io_fail();
  }
  peer(fd.n);
  const auto hello = receive(fd.n, end);
  keys(hello, {"ready", "engine_version", "instance_id"});
  need(hello.at("ready") == true && hello.at("engine_version") == version &&
           hello.at("instance_id") == p.at("instance_id"),
       "resident ready identity mismatch");
  send(fd.n, w, end);
  return unwrap(receive(fd.n, end));
}
void host_marker() {}
std::string host_path() {
  Dl_info info{};
  need(::dladdr(reinterpret_cast<void *>(&host_marker), &info) != 0 &&
           info.dli_fname,
       "resident host location unavailable");
  const auto module = std::filesystem::absolute(info.dli_fname);
  auto candidate = module.parent_path() / "symphony-sbv";
  if (!std::filesystem::exists(candidate) &&
      module.parent_path().filename() == version &&
      module.parent_path().parent_path().filename() == "sbv-engine") {
    auto prefix = module.parent_path()
                      .parent_path()
                      .parent_path()
                      .parent_path()
                      .parent_path();
    candidate =
        prefix / "libexec/symphony/sbv-engine" / version / "symphony-sbv";
  }
  struct stat st{};
  need(::lstat(candidate.c_str(), &st) == 0 && S_ISREG(st.st_mode) &&
           (st.st_mode & 0111),
       "exact companion symphony-sbv executable required");
  return candidate.string();
}
Json launch(const Json &w) {
  const auto end = wire_deadline(w);
  int pair[2];
  need(::socketpair(AF_UNIX, SOCK_STREAM, 0, pair) == 0,
       "resident startup channel unavailable");
  FD parent(pair[0]), child(pair[1]);
  configure(parent.n);
  configure(child.n);
  auto exe = host_path();
  posix_spawn_file_actions_t actions;
  need(::posix_spawn_file_actions_init(&actions) == 0,
       "spawn actions unavailable");
  struct Actions {
    posix_spawn_file_actions_t *p;
    ~Actions() { ::posix_spawn_file_actions_destroy(p); }
  } cleanup{&actions};
  need(::posix_spawn_file_actions_adddup2(&actions, child.n, STDIN_FILENO) ==
               0 &&
           ::posix_spawn_file_actions_adddup2(&actions, child.n,
                                              STDOUT_FILENO) == 0 &&
           ::posix_spawn_file_actions_addopen(&actions, STDERR_FILENO,
                                              "/dev/null", O_WRONLY, 0) == 0,
       "spawn descriptor actions unavailable");
  char mode[] = "--resident-worker";
  char *args[] = {exe.data(), mode, nullptr};
  pid_t pid{};
  need(::posix_spawn(&pid, exe.c_str(), &actions, nullptr, args, environ) == 0,
       "resident host launch failed");
  ::close(child.n);
  child.n = -1;
  // Reap the short-lived launcher. The resident host reparents itself after
  // exec, so the SDK owns no background reaper thread or unload-time callback.
  for (;;) {
    int status = 0;
    const auto reaped = ::waitpid(pid, &status, WNOHANG);
    if (reaped == pid) {
      need(WIFEXITED(status) && WEXITSTATUS(status) == 0,
           "resident launcher failed");
      break;
    }
    if (reaped < 0 && errno == ECHILD)
      break; // caller may own SIGCHLD handling
    need(reaped >= 0 || errno == EINTR, "resident launcher wait failed");
    deadline(end);
    ::poll(nullptr, 0, 1);
  }
  send(parent.n, w, end);
  auto result = unwrap(receive(parent.n, end));
  send(parent.n, {{"ready", true}}, end);
  return result;
}
void output_shape(const Json &p) {
  const auto path = [](const Json &v) {
    const auto s = str(v);
    need(s.size() > 1 && s.size() <= 4096 && s.front() == '/' &&
             s.back() != '/' && e::is_safe_relative_path(s.substr(1)),
         "absolute safe output path required");
    return s;
  };
  if (p.contains("output_path")) { (void)path(p.at("output_path")); return; }
  const auto &o = p.at("output");
  keys_optional(o, {"kind", "bundle_path", "workspace_path", "write_options"}, {"scratch_read_options"});
  need(o.at("kind") == "partitioned", "partitioned output required");
  const auto bundle = path(o.at("bundle_path")), work = path(o.at("workspace_path"));
  need(bundle != work && !bundle.starts_with(work + "/") && !work.starts_with(bundle + "/"),
       "workspace and output bundle must be disjoint");
  const auto &w = o.at("write_options");
  keys(w, {"page_bytes", "index_fanout"});
  need(u64(w.at("page_bytes")) >= 512 && u64(w.at("index_fanout")) >= 2,
       "invalid output page layout");
  if (o.contains("scratch_read_options")) {
    const auto &r = o.at("scratch_read_options");
    keys(r, {"max_page_bytes", "cache_bytes"});
    need(r.at("max_page_bytes").is_null() || u64(r.at("max_page_bytes")) > 0,
         "positive scratch page limit or null required");
    (void)u64(r.at("cache_bytes"));
  }
}
void control_shape(const std::string &op, const Json &p) {
  if (op == "dataset_load") {
    keys_optional(p,
                  {"protocol", "directory", "instance_id",
                   "memory_budget_bytes", "residency", "max_concurrent_jobs",
                   "worker_budget", "idle_timeout_ms"},
                  {"dataset_limits", "source_path", "source_sha256", "dataset",
                   "retained_source"});
    (void)dataset_source_selection(p);
    need(p.at("residency") == "pageable" || p.at("residency") == "locked",
         "residency must be pageable or locked");
    need(p.at("memory_budget_bytes").is_null() ||
             u64(p.at("memory_budget_bytes")) > 0,
         "positive load buffer budget or null required");
    (void)resource::limit(p.at("max_concurrent_jobs"));
    (void)resource::limit(p.at("worker_budget"));
    (void)u64(p.at("idle_timeout_ms"));
  } else if (op == "dataset_execute") {
    keys_optional(
        p, {"protocol", "directory", "instance_id", "operation", "request"},
        {"output_path", "output", "job_id"});
    need(!p.contains("job_id") || job_identifier(p.at("job_id")),
         "job_id must contain 1..128 ASCII letters, digits, dot, dash or underscore");
    need(p.contains("output_path") != p.contains("output"),
         "resident request selects exactly one output form");
    output_shape(p);
    need(p.at("operation") == "run" || p.at("operation") == "evaluate" ||
             p.at("operation") == "book" ||
             p.at("operation") == "generate_census",
         "resident execution supports run, evaluate, book and generate_census");
    const auto &r = p.at("request");
    auto slug = str(p.at("operation"));
    std::replace(slug.begin(), slug.end(), '_', '-');
    need(r.is_object() && !r.contains("output_path") && !r.contains("output") &&
             r.at("protocol") == "symphony.sbv." + slug + "-input.v1",
         "resident child request must omit both output forms and match "
         "operation");
  } else if (op == "dataset_configure") {
    keys(p, {"protocol", "directory", "instance_id", "expected_generation",
             "max_concurrent_jobs", "worker_budget", "idle_timeout_ms",
             "on_active_over_limit"});
    (void)u64(p.at("expected_generation"));
    (void)resource::limit(p.at("max_concurrent_jobs"));
    (void)resource::limit(p.at("worker_budget"));
    (void)u64(p.at("idle_timeout_ms"));
    need(p.at("on_active_over_limit") == "reject" ||
             p.at("on_active_over_limit") == "drain", "select reject or drain");
  } else if (op == "dataset_jobs") {
    keys(p, {"protocol", "directory", "instance_id", "job_id", "after_sequence", "limit"});
    need(p.at("job_id").is_null() || job_identifier(p.at("job_id")), "invalid job_id");
    (void)u64(p.at("after_sequence"));
    need(u64(p.at("limit")) > 0, "positive job page limit required");
    need(p.at("job_id").is_null() ||
             (p.at("after_sequence") == "0" && p.at("limit") == "1"),
         "single job lookup requires after_sequence 0 and limit 1");
  } else if (op == "dataset_cancel") {
    keys(p, {"protocol", "directory", "instance_id", "job_id"});
    need(job_identifier(p.at("job_id")), "invalid job_id");
  } else {
    need(op == "dataset_inspect" || op == "dataset_release" || op == "dataset_stop",
         "unknown dataset operation");
    keys(p, {"protocol", "directory", "instance_id"});
  }
  auto slug = op;
  std::replace(slug.begin(), slug.end(), '_', '-');
  need(p.at("protocol") == "symphony.sbv." + slug + "-input.v1",
       "dataset input protocol mismatch");
  (void)address(p);
}
} // namespace
Json dataset_control(const std::string &op, const Json &p, std::int64_t end) {
  control_shape(op, p);
  auto w = wire(op, p, end);
  validate_wire(w);
  return op == "dataset_load" ? launch(w) : transact(w);
}
int resident_worker() {
  // This entrypoint is a separate bounded local wire protocol. It does not
  // change the one-request/one-response engine-process.v2 contract.
  // Fork only in this freshly exec'd single-threaded companion, never in the
  // embedding SDK process (which may have unrelated threads and locks).
  const auto child = ::fork();
  if (child < 0)
    return 4;
  if (child > 0)
    return 0;
  ::setsid();
  for (int fd = 3, max = ::getdtablesize(); fd < max; ++fd)
    ::close(fd);
  struct Endpoint {
    std::string path;
    dev_t dev{};
    ino_t ino{};
    void remove() noexcept {
      struct stat st{};
      if (!path.empty() && ::lstat(path.c_str(), &st) == 0 &&
          st.st_dev == dev && st.st_ino == ino && ::unlink(path.c_str()) == 0)
        path.clear();
    }
    ~Endpoint() { remove(); }
  } endpoint;
  auto startup_end = e::no_deadline;
  try {
    configure(STDIN_FILENO);
    auto w = receive(STDIN_FILENO, startup_end);
    validate_wire(w);
    need(w.at("operation") == "dataset_load",
         "resident startup operation mismatch");
    const auto p = w.at("input");
    control_shape("dataset_load", p);
    startup_end = wire_deadline(w);
    auto a = address(p);
    // An incarnation is never reused, including after a failed load or crash.
    auto dir = directory(str(p.at("directory")));
    const auto claim_name = str(p.at("instance_id")) + ".claim";
    FD claim(::openat(dir.n, claim_name.c_str(),
                      O_CREAT | O_EXCL | O_NOFOLLOW | O_WRONLY | O_CLOEXEC,
                      0600));
    need(claim.n >= 0,
         "resident instance id already claimed; choose a fresh id");
    FD listener(::socket(AF_UNIX, SOCK_STREAM, 0));
    configure(listener.n);
    if (::bind(listener.n, reinterpret_cast<sockaddr *>(&a), sizeof(a)) != 0) {
      const int error = errno;
      need(false,
           error == EADDRINUSE
               ? "resident endpoint already exists; choose a fresh instance id"
               : ("resident endpoint bind failed: " +
                  std::string(std::strerror(error)) + " (errno " +
                  std::to_string(error) + ")")
                     .c_str());
    }
    struct stat st{};
    need(::lstat(a.sun_path, &st) == 0,
         "resident endpoint identity unavailable");
    endpoint.path = a.sun_path;
    endpoint.dev = st.st_dev;
    endpoint.ino = st.st_ino;
    need(::chmod(a.sun_path, 0600) == 0 && ::listen(listener.n, 16) == 0,
         "resident endpoint activation failed");
    auto data = load_dataset(p, startup_end, p.at("residency") == "locked");
    data->resident_identity = {{"directory", p.at("directory")},
                               {"instance_id", p.at("instance_id")},
                               {"engine_version", version}};
    auto max_jobs = resource::limit(p.at("max_concurrent_jobs")),
               budget = resource::limit(p.at("worker_budget"));
    auto idle = u64(p.at("idle_timeout_ms"));
    std::uint64_t generation = 0;
    std::mutex mutex;
    std::uint64_t active = 0, workers = 0, completed = 0, failed = 0;
    auto last = e::unix_time_ms();
    auto status = [&](const std::string &op, const std::string &state) {
      auto slug = op;
      std::replace(slug.begin(), slug.end(), '_', '-');
      Json result{
          {"protocol", "symphony.sbv." + slug + ".v1"},
          {"engine_version", version},
          {"state", state},
          {"directory", p.at("directory")},
          {"instance_id", p.at("instance_id")},
          {"source_path", data->path},
          {"source_sha256", data->sha256},
          {"dataset", data->dataset_name},
          {"events", dec(data->events.size())},
          {"decoded_bytes", dec(data->events.size() * sizeof(data->events[0]))},
          {"load_buffer_bytes", dec(data->load_buffer_bytes)},
          {"memory_budget_bytes", p.at("memory_budget_bytes")},
          {"dataset_limits", data->limits_evidence()},
          {"residency", p.at("residency")},
          {"source_reads", "1"},
          {"decode_passes", "1"},
          {"active_jobs", dec(active)},
          {"active_workers", dec(workers)},
          {"completed_jobs", dec(completed)},
          {"failed_jobs", dec(failed)},
          {"max_concurrent_jobs", max_jobs ? Json(dec(*max_jobs)) : Json(nullptr)},
          {"worker_budget", budget ? Json(dec(*budget)) : Json(nullptr)},
          {"configuration_generation", dec(generation)},
          {"idle_timeout_ms", dec(idle)}};
      if (!data->source_delivery.is_null()) {
        result["source_delivery"] = data->source_delivery;
        result["load_buffer_scope"] = data->load_buffer_scope;
      }
      return result;
    };
    send(STDIN_FILENO, success(status("dataset_load", "ready")), startup_end);
    need(receive(STDIN_FILENO, startup_end) == Json{{"ready", true}},
         "resident startup acknowledgement missing");
    ::close(STDIN_FILENO);
    ::close(STDOUT_FILENO);
    struct Record {
      std::atomic_bool cancelled{false};
      Json view;
    };
    std::map<std::string, std::shared_ptr<Record>> by_id;
    std::map<std::uint64_t, std::shared_ptr<Record>> records;
    std::uint64_t sequence = 0;
    std::atomic_bool stopping{false};
    auto identity = [&](const std::string &op) {
      auto slug = op;
      std::replace(slug.begin(), slug.end(), '_', '-');
      return Json{{"protocol", "symphony.sbv." + slug + ".v1"},
                  {"engine_version", version}, {"directory", p.at("directory")},
                  {"instance_id", p.at("instance_id")}};
    };
    // All record views and host accounting are protected by mutex. Network I/O
    // and computation never hold it; incomplete clients cannot monopolize control.
    auto handle = [&](FD client) {
      CancellationScope connection_cancel(&stopping);
      auto end = e::no_deadline;
      try {
        configure(client.n);
        peer(client.n);
        send(client.n, {{"ready", true}, {"engine_version", version},
                       {"instance_id", p.at("instance_id")}}, end);
        auto q = receive(client.n, end);
        validate_wire(q);
        end = wire_deadline(q);
        deadline(end);
        const auto op = str(q.at("operation"));
        const auto input = q.at("input");
        control_shape(op, input);
        need(op != "dataset_load" && input.at("directory") == p.at("directory") &&
                 input.at("instance_id") == p.at("instance_id"), "resident instance mismatch");
        std::unique_lock lock(mutex);
        need(!stopping.load(), "resident host is stopping");
        last = e::unix_time_ms();
        if (op == "dataset_inspect" || op == "dataset_configure") {
          if (op == "dataset_configure") {
            need(u64(input.at("expected_generation")) == generation && generation < UINT64_MAX,
                 "resident configuration generation mismatch or exhausted");
            const auto next_jobs = resource::limit(input.at("max_concurrent_jobs"));
            const auto next_budget = resource::limit(input.at("worker_budget"));
            need(input.at("on_active_over_limit") == "drain" ||
                     ((!next_jobs || active <= *next_jobs) && (!next_budget || workers <= *next_budget)),
                 "active jobs exceed selected policy; select drain or retry after completion");
            max_jobs = next_jobs;
            budget = next_budget;
            idle = u64(input.at("idle_timeout_ms"));
            ++generation;
          }
          auto result = status(op, "ready");
          lock.unlock();
          send(client.n, success(result), end);
          return;
        }
        if (op == "dataset_jobs" || op == "dataset_cancel") {
          auto result = identity(op);
          if (op == "dataset_cancel") {
            const auto it = by_id.find(str(input.at("job_id")));
            need(it != by_id.end(), "unknown resident job_id");
            if (it->second->view.at("state") == "running") {
              it->second->cancelled.store(true);
              it->second->view["cancellation_requested"] = true;
            }
            result["job"] = it->second->view;
          } else {
            Json page = Json::array();
            auto after = u64(input.at("after_sequence"));
            bool more = false;
            if (!input.at("job_id").is_null()) {
              const auto it = by_id.find(str(input.at("job_id")));
              need(it != by_id.end(), "unknown resident job_id");
              page.push_back(it->second->view);
              after = u64(it->second->view.at("sequence"));
            } else {
              std::size_t page_bytes = 0, page_nodes = 128;
              for (auto it = records.upper_bound(after); it != records.end(); ++it) {
                const auto bytes = it->second->view.dump().size();
                const auto nodes = control_nodes(it->second->view);
                if (page.size() >= u64(input.at("limit")) ||
                    (!page.empty() && (page_bytes + bytes > wire_limit / 2 ||
                                      page_nodes + nodes > e::Limits::max_json_values - 256))) {
                  more = true;
                  break;
                }
                page.push_back(it->second->view);
                page_bytes += bytes;
                page_nodes += nodes;
                after = it->first;
              }
            }
            result["jobs"] = std::move(page);
            result["next_after_sequence"] = dec(after);
            result["highest_sequence"] = dec(sequence);
            result["more"] = more;
          }
          lock.unlock();
          send(client.n, success(result), end);
          return;
        }
        if (op == "dataset_release" || op == "dataset_stop") {
          need(op == "dataset_stop" || active == 0,
               "resident dataset busy; release refused while jobs are active");
          Json result;
          if (op == "dataset_stop") {
            result = identity(op);
            result.update({{"state", "termination_requested"}, {"active_jobs", dec(active)},
                           {"active_workers", dec(workers)}, {"results_may_be_incomplete", true},
                           {"automatic_retry", false}, {"rollback_performed", false}});
          } else {
            result = status(op, "released");
            data.reset();
          }
          stopping.store(true);
          endpoint.remove();
          lock.unlock();
          final_send(client.n, success(result));
          if (op == "dataset_stop") ::_exit(0); // explicit whole-host termination
          return;
        }
        const auto inner_op = str(input.at("operation"));
        auto request = input.at("request");
        const auto output_field = input.contains("output") ? "output" : "output_path";
        request[output_field] = input.at(output_field);
        const auto parent_request_sha = e::sha256_hex(input.dump());
        const auto id = input.value("job_id", parent_request_sha);
        need(!by_id.contains(id), "resident job_id already admitted; inspect it before selecting a new id");
        data->bind(request);
        const auto inner = (inner_op == "book" || inner_op == "generate_census")
                               ? 1 : u64(request.at("workers"));
        need(inner >= 1 && resource::admits(max_jobs, active, 1) &&
                 resource::admits(budget, workers, inner) && sequence < UINT64_MAX &&
                 completed <= UINT64_MAX - failed && active < UINT64_MAX - (completed + failed),
             "resident job/worker capacity unavailable; select fewer workers or retry explicitly");
        auto record = std::make_shared<Record>();
        record->view = {{"job_id", id}, {"sequence", dec(sequence + 1)},
                        {"request_sha256", parent_request_sha}, {"operation", inner_op},
                        {"state", "running"}, {"requested_workers", dec(inner)},
                        {"cancellation_requested", false}, {"error_code", ""},
                        {"output", Json{{output_field, input.at(output_field)}}}};
        need(record->view.dump().size() < wire_limit / 2 &&
                 control_nodes(record->view) < e::Limits::max_json_values / 2,
             "resident job control reference exceeds frame budget");
        records.emplace(sequence + 1, record);
        try { by_id.emplace(id, record); }
        catch (...) { records.erase(sequence + 1); throw; }
        ++sequence;
        ++active;
        workers += inner;
        lock.unlock();
        Json response;
        std::string state = "failed", error_code;
        try {
          CancellationScope job_cancel(&record->cancelled);
          deadline(end);
          auto receipt = inner_op == "run" ? run(request, end, data.get())
              : inner_op == "evaluate" ? evaluate(request, end, data.get())
              : inner_op == "generate_census" ? generate_census(request, end, data.get())
              : book(request, end, data.get());
          if (receipt.value("status", "") == "recovery_required") {
            receipt = {{"protocol", "symphony.sbv.dataset-execute.v1"},
                       {"status", "recovery_required"}, {"code", "sbv.dataset_execution_incomplete"},
                       {"request_sha256", parent_request_sha}, {"automatic_retry", false},
                       {"rollback_performed", false}, {"operation", inner_op},
                       {"child_recovery", std::move(receipt)}};
            state = "recovery_required";
            error_code = "sbv.dataset_execution_incomplete";
          } else {
            receipt["protocol"] = "symphony.sbv.dataset-execute.v1";
            state = "completed";
          }
          response = success(receipt);
        } catch (const e::Error &x) {
          error_code = x.code();
          if (error_code == "sbv.cancelled") state = "cancelled";
          response = failure(x.code(), x.what());
        } catch (...) {
          error_code = "sbv.failure";
          response = failure(error_code, "resident job failed; inspect destination before retrying");
        }
        {
          std::lock_guard guard(mutex);
          record->view["state"] = state;
          record->view["error_code"] = error_code;
          --active;
          workers -= inner;
          (state == "completed" ? completed : failed)++;
          last = e::unix_time_ms();
        }
        send(client.n, response, end);
      } catch (const e::Error &x) {
        try { send(client.n, failure(x.code(), x.what()), end); } catch (...) {}
      } catch (...) {
        try { send(client.n, failure("sbv.failure", "resident request failed; inspect destination before retrying"), end); }
        catch (...) {}
      }
    };
    struct Connection {
      std::shared_ptr<std::atomic_bool> done;
      std::jthread thread;
    };
    std::vector<Connection> connections;
    // Declared after connections: cancel stalled framing and cooperative work
    // before any joining destructor, including exceptional exits.
    struct Shutdown {
      std::atomic_bool &stopping;
      std::mutex &mutex;
      decltype(records) &jobs;
      ~Shutdown() {
        stopping.store(true);
        std::lock_guard lock(mutex);
        for (const auto &[_, job] : jobs) job->cancelled.store(true);
      }
    } shutdown{stopping, mutex, records};
    while (!stopping.load()) {
      std::erase_if(connections, [](const Connection &c) { return c.done->load(); });
      {
        std::lock_guard lock(mutex);
        if (idle && active == 0 && static_cast<std::uint64_t>(
              std::max<std::int64_t>(0, e::unix_time_ms() - last)) >= idle)
          break;
      }
      pollfd poller{listener.n, POLLIN, 0};
      const int n = ::poll(&poller, 1, 100);
      if (n <= 0) {
        if (n < 0 && errno != EINTR) break;
        continue;
      }
      FD client(::accept(listener.n, nullptr, nullptr));
      if (client.n < 0) continue;
      // Allocate bookkeeping before the thread starts. Failure closes just this
      // connection, leaving the resident host and accepted work available.
      try {
        auto done = std::make_shared<std::atomic_bool>(false);
        connections.emplace_back();
        connections.back().done = done;
        try {
          connections.back().thread = std::jthread([&, fd = std::move(client), done]() mutable {
            handle(std::move(fd));
            done->store(true);
          });
        } catch (...) { connections.pop_back(); }
      } catch (...) {}
    }
    return 0;
  } catch (const e::Error &x) {
    try {
      send(STDIN_FILENO, failure(x.code(), x.what()), startup_end);
    } catch (...) {
    }
  } catch (...) {
    try {
      send(STDIN_FILENO, failure("sbv.failure", "resident preload failed"),
           startup_end);
    } catch (...) {
    }
  }
  return 4;
}
} // namespace symphony::sbv::detail
