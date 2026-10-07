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
    need(u64(p.at("max_concurrent_jobs")) >= 1 &&
             u64(p.at("max_concurrent_jobs")) <= 16,
         "resident concurrent job bound 1..16");
    need(u64(p.at("worker_budget")) >= 1 && u64(p.at("worker_budget")) <= 64,
         "resident worker budget 1..64");
    (void)u64(p.at("idle_timeout_ms"));
  } else if (op == "dataset_execute") {
    keys(p, {"protocol", "directory", "instance_id", "operation", "request",
             "output_path"});
    need(p.at("operation") == "run" || p.at("operation") == "evaluate" ||
             p.at("operation") == "book" ||
             p.at("operation") == "generate_census",
         "resident execution supports run, evaluate, book and generate_census");
    const auto &r = p.at("request");
    auto slug = str(p.at("operation"));
    std::replace(slug.begin(), slug.end(), '_', '-');
    need(r.is_object() && !r.contains("output_path") &&
             r.at("protocol") == "symphony.sbv." + slug + "-input.v1",
         "resident child request must omit output_path and match operation");
  } else {
    need(op == "dataset_inspect" || op == "dataset_release",
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
    const auto max_jobs = u64(p.at("max_concurrent_jobs")),
               budget = u64(p.at("worker_budget")),
               idle = u64(p.at("idle_timeout_ms"));
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
          {"max_concurrent_jobs", dec(max_jobs)},
          {"worker_budget", dec(budget)},
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
    struct Job {
      std::shared_ptr<std::atomic<bool>> done;
      std::jthread thread;
    };
    std::vector<Job> jobs;
    jobs.reserve(max_jobs);
    for (;;) {
      std::erase_if(jobs, [](const Job &j) { return j.done->load(); });
      {
        std::lock_guard lock(mutex);
        if (idle && active == 0 &&
            static_cast<std::uint64_t>(
                std::max<std::int64_t>(0, e::unix_time_ms() - last)) >= idle)
          break;
      }
      pollfd poller{listener.n, POLLIN, 0};
      int n = ::poll(&poller, 1, 100);
      if (n <= 0) {
        if (n < 0 && errno != EINTR)
          break;
        continue;
      }
      FD client(::accept(listener.n, nullptr, nullptr));
      if (client.n < 0)
        continue;
      auto end = e::no_deadline;
      try {
        configure(client.n);
        peer(client.n);
        send(client.n,
             {{"ready", true},
              {"engine_version", version},
              {"instance_id", p.at("instance_id")}},
             end);
        auto q = receive(client.n, end);
        validate_wire(q);
        end = wire_deadline(q);
        const auto op = str(q.at("operation"));
        const auto input = q.at("input");
        control_shape(op, input);
        need(op != "dataset_load" &&
                 input.at("directory") == p.at("directory") &&
                 input.at("instance_id") == p.at("instance_id"),
             "resident instance mismatch");
        std::unique_lock lock(mutex);
        last = e::unix_time_ms();
        if (op == "dataset_inspect") {
          send(client.n, success(status(op, "ready")), end);
          continue;
        }
        if (op == "dataset_release") {
          need(active == 0,
               "resident dataset busy; release refused while jobs are active");
          // Stop admission and free the payload before acknowledging release.
          auto result = status(op, "released");
          ::close(listener.n);
          listener.n = -1;
          data.reset();
          endpoint.remove();
          // A disconnected peer fails immediately; acknowledgement follows the
          // user deadline.
          try {
            send(client.n, success(result), end);
          } catch (...) {
          }
          break;
        }
        const auto inner_op = str(input.at("operation"));
        auto request = input.at("request");
        request["output_path"] = input.at("output_path");
        data->bind(request);
        const auto inner = (inner_op == "book" || inner_op == "generate_census")
                               ? 1
                               : u64(request.at("workers"));
        need(inner >= 1 && inner <= 64 && jobs.size() < max_jobs &&
                 active < max_jobs && inner <= budget - workers,
             "resident job/worker capacity unavailable; select fewer workers "
             "or retry explicitly");
        auto done = std::make_shared<std::atomic<bool>>(false);
        ++active;
        workers += inner;
        try {
          jobs.push_back(
              {done, std::jthread([&, fd = std::move(client), request, inner_op,
                                   inner, end, done]() {
                 Json response;
                 bool ok = false;
                 try {
                   auto receipt =
                       inner_op == "run" ? run(request, end, data.get())
                       : inner_op == "evaluate"
                           ? evaluate(request, end, data.get())
                       : inner_op == "generate_census"
                           ? generate_census(request, end, data.get())
                           : book(request, end, data.get());
                   receipt["protocol"] = "symphony.sbv.dataset-execute.v1";
                   response = success(receipt);
                   ok = true;
                 } catch (const e::Error &x) {
                   response = failure(x.code(), x.what());
                 } catch (...) {
                   response =
                       failure("sbv.failure", "resident job failed; inspect "
                                              "destination before retrying");
                 }
                 {
                   std::lock_guard guard(mutex);
                   --active;
                   workers -= inner;
                   (ok ? completed : failed)++;
                   last = e::unix_time_ms();
                 }
                 try {
                   send(fd.n, response, end);
                 } catch (...) {
                 }
                 done->store(true);
               })});
        } catch (...) {
          --active;
          workers -= inner;
          throw;
        }
      } catch (const e::Error &x) {
        try {
          send(client.n, failure(x.code(), x.what()), end);
        } catch (...) {
        }
      } catch (...) {
        try {
          send(client.n,
               failure("sbv.failure", "resident request failed; inspect "
                                      "destination before retrying"),
               end);
        } catch (...) {
        }
      }
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
