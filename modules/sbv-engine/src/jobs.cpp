#include "experiment_result.hpp"
#include <chrono>
#include <fcntl.h>
#include <limits>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/file.h>
#include <sys/stat.h>
#include <thread>
#include <unistd.h>

namespace symphony::sbv::detail {
namespace {
struct File {
  int fd;
  explicit File(int value) : fd(value) {
    need(fd >= 0, "job file unavailable");
  }
  ~File() { ::close(fd); }
  File(const File &) = delete;
};
void regular(int fd) {
  struct stat st{};
  need(::fstat(fd, &st) == 0 && S_ISREG(st.st_mode) && st.st_nlink == 1 &&
           st.st_uid == ::geteuid() && (st.st_mode & 077) == 0,
       "job journal requires private caller-owned regular files");
}
bool lock(int fd) {
  if (::flock(fd, LOCK_EX | LOCK_NB) == 0)
    return true;
  need(errno == EWOULDBLOCK || errno == EAGAIN, "job lock unavailable");
  return false;
}
void acquire(int fd, std::int64_t end) {
  while (!lock(fd)) {
    deadline(end);
    std::this_thread::sleep_for(std::chrono::milliseconds(2));
  }
}
struct Journal {
  std::string path;
  File dir;
  File control;
  static int open_dir(const std::string &path) {
    need(path.size() > 1 && path[0] == '/' && path.size() <= 3500 &&
             e::is_safe_relative_path(path.substr(1)),
         "absolute job directory without traversal required");
    File root(::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC));
    for (const auto &part : std::filesystem::path(path).relative_path()) {
      int next = ::openat(root.fd, part.c_str(),
                          O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
      need(next >= 0, "job directory must exist without symlinks");
      ::close(root.fd);
      root.fd = next;
    }
    struct stat st{};
    need(::fstat(root.fd, &st) == 0 && st.st_uid == ::geteuid() &&
             (st.st_mode & 077) == 0,
         "job directory must be private and caller-owned");
    return ::fcntl(root.fd, F_DUPFD_CLOEXEC, 0);
  }
  explicit Journal(std::string selected, bool create)
      : path(std::move(selected)), dir(open_dir(path)),
        control(::openat(
            dir.fd, "control.lock",
            O_RDWR | (create ? O_CREAT : 0) | O_NOFOLLOW | O_CLOEXEC, 0600)) {
    regular(control.fd);
  }
  bool exists(const std::string &name) const {
    struct stat st{};
    if (::fstatat(dir.fd, name.c_str(), &st, AT_SYMLINK_NOFOLLOW) == 0) {
      need(S_ISREG(st.st_mode) && st.st_nlink == 1 &&
               st.st_uid == ::geteuid() && (st.st_mode & 077) == 0,
           "invalid job journal file");
      return true;
    }
    need(errno == ENOENT, "job journal status unavailable");
    return false;
  }
  Json load(const std::string &name) const {
    File file(
        ::openat(dir.fd, name.c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC));
    regular(file.fd);
    std::string bytes;
    char buffer[65536];
    for (;;) {
      const auto n = ::read(file.fd, buffer, sizeof(buffer));
      if (n < 0 && errno == EINTR)
        continue;
      need(n >= 0, "job journal read failed");
      if (n == 0)
        break;
      need(bytes.size() + n <= artifact_bytes, "job journal framing exceeded");
      bytes.append(buffer, n);
    }
    auto value = e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
    validate_result(value);
    return value;
  }
  void write(const std::string &name, const Json &value, bool replace = false) {
    validate_result(value);
    const auto bytes = value.dump();
    need(bytes.size() <= artifact_bytes, "job journal framing exceeded");
    if (!replace && exists(name)) {
      need(load(name) == value, "immutable job journal differs");
      return;
    }
    static std::atomic_uint64_t serial{};
    const auto temp =
        ".job-" + dec(::getpid()) + "-" +
        dec(std::chrono::steady_clock::now().time_since_epoch().count()) + "-" +
        dec(serial++) + ".tmp";
    File file(::openat(dir.fd, temp.c_str(),
                       O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC,
                       0600));
    try {
      for (std::size_t offset = 0; offset < bytes.size();) {
        auto n = ::write(file.fd, bytes.data() + offset, bytes.size() - offset);
        if (n < 0 && errno == EINTR)
          continue;
        need(n > 0, "job journal write failed");
        offset += n;
      }
      need(::fsync(file.fd) == 0, "job journal sync failed");
      if (replace) {
        (void)exists(name);
        need(::renameat(dir.fd, temp.c_str(), dir.fd, name.c_str()) == 0,
             "job journal replacement failed");
      } else {
        need(::linkat(dir.fd, temp.c_str(), dir.fd, name.c_str(), 0) == 0,
             "job journal publication failed");
        need(::unlinkat(dir.fd, temp.c_str(), 0) == 0,
             "job journal staging cleanup failed");
      }
      need(::fsync(dir.fd) == 0, "job journal durability uncertain; reconcile");
    } catch (...) {
      ::unlinkat(dir.fd, temp.c_str(), 0);
      throw;
    }
  }
};
Json artifact(const char *origin, Json data) {
  auto value = base(origin);
  value["sections"]["job"] = section(std::move(data));
  return seal_result(std::move(value));
}
const Json &data(const Json &value) {
  return value.at("sections").at("job").at("data");
}
Json reference(const Journal &j, const std::string &name, const Json &value) {
  return {{"path", j.path + "/" + name},
          {"expected_sha256", value.at("content_sha256")},
          {"pointer", ""}};
}
Json identity(const Json &plan) {
  return {{"engine_version", version},
          {"job_id", data(plan).at("job_id")},
          {"plan_sha256", plan.at("content_sha256")}};
}
Json load_bound(const Journal &j, const char *name, const char *origin,
                const Json &plan) {
  auto value = j.load(name);
  need(value.at("origin") == origin, "job journal kind mismatch");
  for (const auto &[key, v] : identity(plan).items())
    need(data(value).at(key) == v, "job journal identity mismatch");
  return value;
}
void save_state(Journal &j, const Json &state) {
  const auto sealed = artifact("job_state", state);
  j.write("state-" + str(sealed.at("content_sha256")) + ".json", sealed);
  j.write("state.json", sealed, true);
}
void advance(Journal &j, Json &state) {
  auto generation = u64(state.at("generation"));
  need(generation != std::numeric_limits<std::uint64_t>::max(),
       "job generation exhausted");
  state["generation"] = dec(generation + 1);
  save_state(j, state);
}
bool terminal(const Json &state) {
  return state.at("phase") != "queued" && state.at("phase") != "running";
}
bool supported(const std::string &op) {
  static const std::set<std::string> operations{"run",
                                                "generate_census",
                                                "evaluate",
                                                "compose",
                                                "compose_joint",
                                                "economics",
                                                "book",
                                                "liquidity",
                                                "queue_scenarios",
                                                "account_simulate",
                                                "allocation_economics",
                                                "analyze",
                                                "compare",
                                                "resample",
                                                "split",
                                                "fit",
                                                "predict",
                                                "research_history",
                                                "compose_economics",
                                                "result_select"};
  return operations.contains(op);
}
void validate_plan(const Json &p, const Journal &j) {
  const auto id = str(p.at("job_id"));
  need(!id.empty() && id.size() <= 80 &&
           std::all_of(id.begin(), id.end(),
                       [](char c) {
                         return (c >= 'a' && c <= 'z') ||
                                (c >= 'A' && c <= 'Z') ||
                                (c >= '0' && c <= '9') || c == '_' || c == '-';
                       }),
       "job id must be 1..80 ASCII letters, digits, underscores or hyphens");
  const auto op = str(p.at("operation"));
  need(supported(op), "unsupported durable local calculation operation");
  const auto &child = p.at("request");
  need(child.is_object() &&
           child.at("protocol") ==
               trial::protocol(op).substr(0, trial::protocol(op).size() - 3) +
                   "-input.v1",
       "job child protocol mismatch");
  need(child.contains("output") != child.contains("output_path"),
       "job requires one explicit child output format");
  auto outside = [&](const Json &value) {
    const auto path = str(value);
    need(path.size() > 1 && path[0] == '/' &&
             e::is_safe_relative_path(path.substr(1)) && path != j.path &&
             !path.starts_with(j.path + "/") && !j.path.starts_with(path + "/"),
         "child output must be disjoint from the job journal");
  };
  if (child.contains("output")) {
    outside(child.at("output").at("bundle_path"));
    outside(child.at("output").at("workspace_path"));
  } else {
    outside(child.at("output_path"));
  }
  trial::read_options(p.at("read_options"));
  if (!p.at("deadline_unix_ms").is_null())
    need(i64(p.at("deadline_unix_ms")) > 0,
         "positive absolute job deadline or null required");
  need(p.at("extensions").is_object(), "job extensions required");
}
Json snapshot(Journal &j, const Json &plan, const std::string &op,
              bool executing) {
  auto state_file = load_bound(j, "state.json", "job_state", plan);
  const auto &state = data(state_file);
  Json outcome = nullptr, receipt = nullptr, error = nullptr;
  if (j.exists("outcome.json")) {
    auto saved = load_bound(j, "outcome.json", "job_outcome", plan);
    outcome = reference(j, "outcome.json", saved);
    receipt = data(saved).at("result_receipt");
    error = data(saved).at("error_code");
  }
  Json claim = nullptr, cancellation = nullptr;
  if (j.exists("claim.json"))
    claim = reference(j, "claim.json",
                      load_bound(j, "claim.json", "job_claim", plan));
  if (j.exists("cancel.json"))
    cancellation = reference(j, "cancel.json",
                             load_bound(j, "cancel.json", "job_cancel", plan));
  auto result = identity(plan);
  result.update(
      {{"protocol", trial::protocol(op)},
       {"directory", j.path},
       {"generation", state.at("generation")},
       {"phase", state.at("phase")},
       {"cancellation_requested", j.exists("cancel.json")},
       {"cancellation_observed", state.at("cancellation_observed")},
       {"execution_lock_held", executing},
       {"admitted", j.exists("claim.json")},
       {"automatic_retry", false},
       {"result_receipt", receipt},
       {"error_code", error},
       {"plan_reference", reference(j, "plan.json", plan)},
       {"state_reference",
        reference(j, "state-" + str(state_file.at("content_sha256")) + ".json",
                  state_file)},
       {"outcome_reference", outcome},
       {"claim_reference", claim},
       {"cancellation_reference", cancellation}});
  return result;
}
void reconcile(Journal &j, const Json &plan, Json &state, std::int64_t end) {
  if (j.exists("outcome.json")) {
    const auto outcome = load_bound(j, "outcome.json", "job_outcome", plan);
    const auto &o = data(outcome);
    if (!o.at("result_receipt").is_null() && o.at("phase") != "unknown")
      (void)trial::verify(o.at("result_receipt"), data(plan).at("request"),
                          str(data(plan).at("operation")),
                          data(plan).at("read_options"), end);
    if (state.at("phase") != o.at("phase") ||
        state.at("cancellation_observed") != o.at("cancellation_observed")) {
      state["phase"] = o.at("phase");
      state["cancellation_observed"] = o.at("cancellation_observed");
      advance(j, state);
    }
  } else if (j.exists("claim.json") && state.at("phase") != "unknown") {
    (void)load_bound(j, "claim.json", "job_claim", plan);
    state["phase"] = "unknown";
    advance(j, state);
  } else if (!j.exists("claim.json") && j.exists("cancel.json") &&
             state.at("phase") != "cancelled") {
    state["phase"] = "cancelled";
    state["cancellation_observed"] = true;
    advance(j, state);
  }
}
} // namespace

Json job(const std::string &op, const Json &p, std::int64_t end) {
  if (op == "job_create")
    keys(p, {"protocol", "directory", "job_id", "operation", "request",
             "deadline_unix_ms", "read_options", "extensions"});
  else if (op == "job_cancel")
    keys(p, {"protocol", "directory", "job_id", "expected_plan_sha256",
             "expected_generation", "request_id", "reason"});
  else if (op == "job_reconcile")
    keys(p, {"protocol", "directory", "job_id", "expected_plan_sha256",
             "expected_generation"});
  else
    keys(p, {"protocol", "directory", "job_id", "expected_plan_sha256"});
  Journal j(str(p.at("directory")), op == "job_create");
  // Execution is always acquired before control; cancellation only needs
  // control.
  File execution(::openat(j.dir.fd, "execution.lock",
                          O_RDWR | (op == "job_create" ? O_CREAT : 0) |
                              O_NOFOLLOW | O_CLOEXEC,
                          0600));
  regular(execution.fd);
  const bool available = lock(execution.fd);
  acquire(j.control.fd, end);
  Json plan;
  if (op == "job_create") {
    validate_plan(p, j);
    auto selection = p;
    selection["engine_version"] = version;
    plan = artifact("job_plan", selection);
    j.write("plan.json", plan);
    if (!j.exists("state.json")) {
      need(!j.exists("claim.json") && !j.exists("outcome.json") &&
               !j.exists("cancel.json"),
           "missing job state with existing admission evidence; manual "
           "recovery required");
      auto state = identity(plan);
      state.update({{"generation", "0"},
                    {"phase", "queued"},
                    {"cancellation_observed", false}});
      save_state(j, state);
    }
  } else {
    plan = j.load("plan.json");
    need(plan.at("origin") == "job_plan" &&
             data(plan).at("directory") == j.path &&
             data(plan).at("engine_version") == version &&
             data(plan).at("job_id") == p.at("job_id") &&
             plan.at("content_sha256") == p.at("expected_plan_sha256"),
         "job plan, engine version or caller identity mismatch");
  }
  auto state = data(load_bound(j, "state.json", "job_state", plan));
  if (op == "job_cancel") {
    need(!str(p.at("request_id")).empty(), "cancellation request id required");
    (void)str(p.at("reason"));
    auto cancel = identity(plan);
    cancel.update(
        {{"request_id", p.at("request_id")}, {"reason", p.at("reason")}});
    if (j.exists("cancel.json")) {
      need(data(load_bound(j, "cancel.json", "job_cancel", plan)) == cancel,
           "existing cancellation request differs");
    } else {
      need(u64(p.at("expected_generation")) == u64(state.at("generation")),
           "stale job generation");
      if (!terminal(state)) {
        j.write("cancel.json", artifact("job_cancel", cancel));
        if (!j.exists("claim.json")) {
          state["phase"] = "cancelled";
          state["cancellation_observed"] = true;
        }
        advance(j, state);
      }
    }
  } else if (op == "job_reconcile") {
    need(u64(p.at("expected_generation")) == u64(state.at("generation")),
         "stale job generation");
    if (available)
      reconcile(j, plan, state, end);
  } else if (op == "job_execute" && available && !terminal(state) &&
             !j.exists("claim.json")) {
    if (j.exists("cancel.json")) {
      reconcile(j, plan, state, end);
    } else {
      auto selected_end = end;
      if (!data(plan).at("deadline_unix_ms").is_null()) {
        const auto chosen = i64(data(plan).at("deadline_unix_ms"));
        selected_end = end == e::no_deadline ? chosen : std::min(end, chosen);
      }
      // No deadline/cancellation may interrupt authoritative publication after
      // admission.
      deadline(end);
      j.write("claim.json", artifact("job_claim", identity(plan)));
      state["phase"] = "running";
      advance(j, state);
      need(::flock(j.control.fd, LOCK_UN) == 0, "job control unlock failed");
      Json outcome = identity(plan);
      outcome.update({{"phase", "failed"},
                      {"cancellation_observed", false},
                      {"result_receipt", nullptr},
                      {"error_code", nullptr}});
      const auto inherited = caller_cancellation;
      std::atomic_bool cancelled{inherited && inherited->load()},
          monitor_failed{false};
      {
        std::jthread monitor([&](std::stop_token stop) {
          while (!stop.stop_requested()) {
            try {
              bool requested = inherited && inherited->load();
              if (lock(j.control.fd)) {
                if (j.exists("cancel.json")) {
                  (void)load_bound(j, "cancel.json", "job_cancel", plan);
                  requested = true;
                }
                need(::flock(j.control.fd, LOCK_UN) == 0,
                     "job cancellation monitor unlock failed");
              }
              if (requested) {
                cancelled.store(true);
                return;
              }
            } catch (...) {
              ::flock(j.control.fd, LOCK_UN);
              monitor_failed.store(true);
              cancelled.store(true);
              return;
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(5));
          }
        });
        CancellationScope scope(&cancelled);
        try {
          const auto &child = data(plan).at("request");
          need(!trial::appeared(child),
               "job output already exists before dispatch");
          auto receipt =
              dispatch(str(data(plan).at("operation")), child, selected_end);
          outcome["result_receipt"] = receipt;
          if (receipt.at("status") == "recovery_required") {
            trial::recovery(receipt, child, str(data(plan).at("operation")));
            outcome["phase"] = "unknown";
          } else {
            // Completion wins a cancellation arriving after successful
            // publication.
            CancellationScope completed(nullptr);
            (void)trial::verify(receipt, child, str(data(plan).at("operation")),
                                data(plan).at("read_options"), e::no_deadline);
            outcome["phase"] = receipt.at("status") == "partial"
                                   ? "completed_partial"
                                   : "completed";
          }
        } catch (const e::Error &error) {
          outcome["error_code"] = error.code();
          outcome["cancellation_observed"] =
              error.code() == "sbv.cancelled" && !monitor_failed.load();
          outcome["phase"] =
              error.code() == "sbv.cancelled" ? "cancelled" : "failed";
          if (monitor_failed.load())
            outcome["error_code"] = "sbv.job_monitor_failed";
          if (trial::appeared(data(plan).at("request")))
            outcome["phase"] = "unknown";
        } catch (const std::exception &) {
          outcome["error_code"] = "sbv.job_calculation_failed";
          if (trial::appeared(data(plan).at("request")))
            outcome["phase"] = "unknown";
        }
      }
      CancellationScope publishing(nullptr);
      acquire(j.control.fd, e::no_deadline);
      j.write("outcome.json", artifact("job_outcome", outcome));
      state = data(load_bound(j, "state.json", "job_state", plan));
      state["phase"] = outcome.at("phase");
      state["cancellation_observed"] = outcome.at("cancellation_observed");
      advance(j, state);
    }
  }
  if (available)
    need(::flock(execution.fd, LOCK_UN) == 0, "job execution unlock failed");
  return snapshot(j, plan, op, !available);
}
} // namespace symphony::sbv::detail
