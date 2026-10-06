#include "detail.hpp"
#include <algorithm>
#include <atomic>
#include <cerrno>
#include <fcntl.h>
#include <filesystem>
#include <mutex>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/file.h>
#include <sys/stat.h>
#include <thread>
#include <unistd.h>
namespace symphony::sbv::detail {
namespace {
struct Handle {
  int fd = -1;
  explicit Handle(int n) : fd(n) {}
  ~Handle() {
    if (fd >= 0)
      ::close(fd);
  }
  Handle(const Handle &) = delete;
};
Handle directory(const std::string &path) {
  need(path.size() > 1 && path[0] == '/' && path.size() <= 3500 &&
           e::is_safe_relative_path(path.substr(1)),
       "bounded absolute experiment directory required");
  int current = ::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC);
  need(current >= 0, "directory root unavailable");
  for (const auto &part : std::filesystem::path(path).relative_path()) {
    int next = ::openat(current, part.c_str(),
                        O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    ::close(current);
    need(next >= 0, "experiment directory must exist without symlinks");
    current = next;
  }
  return Handle(current);
}
bool exists(int dir, const std::string &name) {
  struct stat st{};
  if (::fstatat(dir, name.c_str(), &st, AT_SYMLINK_NOFOLLOW) == 0) {
    need(S_ISREG(st.st_mode) && st.st_nlink == 1 && st.st_uid == ::geteuid(),
         "journal artifact must be a private-owner regular file without "
         "hardlinks");
    return true;
  }
  need(errno == ENOENT, "journal artifact status unavailable");
  return false;
}
Json load(const std::string &path, std::int64_t end) {
  auto r = e::parse_bounded_json(read_file(path, end), artifact_bytes,
                                 artifact_values);
  validate_result(r);
  return r;
}
Json save(const std::string &path, Json result, std::int64_t end) {
  result = seal_result(std::move(result));
  validate_result(result);
  create_file(path, result.dump(), end);
  return result;
}
bool id_ok(const std::string &id) {
  return !id.empty() && id.size() <= 80 &&
         std::all_of(id.begin(), id.end(), [](char c) {
           return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
                  (c >= '0' && c <= '9') || c == '_' || c == '-';
         });
}
} // namespace
Json experiment(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "output_path", "directory", "experiment_id", "trials",
           "workers", "on_failure", "extensions"});
  need(p.at("extensions").is_object(), "experiment extensions required");
  const auto root = str(p.at("directory")), id = str(p.at("experiment_id"));
  need(id_ok(id), "bounded experiment id required");
  const auto workers = u64(p.at("workers"));
  need(workers >= 1 && workers <= 16, "experiment workers 1..16");
  need(p.at("on_failure") == "continue" || p.at("on_failure") == "stop",
       "experiment failure policy required");
  need(p.at("on_failure") != "stop" || workers == 1,
       "ordered stop-on-failure requires one worker");
  const auto &trials = p.at("trials");
  need(trials.is_array() && trials.size() <= 64, "experiment trial bound 64");
  std::set<std::string> names;
  std::vector<Json> requests;
  std::uint64_t inner_max = 1;
  const std::set<std::string> operations{"run",
                                         "evaluate",
                                         "compose",
                                         "compose_joint",
                                         "economics",
                                         "book",
                                         "liquidity",
                                         "allocation_economics",
                                         "analyze",
                                         "compare",
                                         "resample",
                                         "split",
                                         "dataset_execute"};
  for (const auto &t : trials) {
    keys(t, {"id", "state", "operation", "request", "reason", "parameters",
             "lineage"});
    auto name = str(t.at("id"));
    need(id_ok(name) && names.insert(name).second,
         "unique safe trial id required");
    need(t.at("state") == "ready" || t.at("state") == "pruned",
         "trial must be ready or caller-pruned");
    need(str(t.at("reason")).size() <= 4096 && t.at("parameters").is_object() &&
             t.at("lineage").is_object(),
         "trial metadata required");
    auto request = t.at("request");
    if (t.at("state") == "ready") {
      need(operations.contains(str(t.at("operation"))) && request.is_object() &&
               !request.contains("output_path"),
           "trial operation must be a native SBV producer and request must "
           "omit output_path");
      request["output_path"] = root + "/" + name + ".result.json";
      const auto &worker_request = t.at("operation") == "dataset_execute"
                                       ? request.at("request")
                                       : request;
      if (worker_request.contains("workers")) {
        const auto inner = u64(worker_request.at("workers"));
        need(inner >= 1 && inner <= 64, "child worker bound");
        inner_max = std::max(inner_max, inner);
      }
    } else
      need(t.at("operation").is_null() && request.is_null() &&
               !str(t.at("reason")).empty(),
           "pruned trial requires reason and null operation/request");
    requests.push_back(request);
  }
  need(workers * inner_max <= 64,
       "selected outer and inner worker product exceeds 64; choose explicit "
       "resource allocation");
  auto choices = p;
  choices.erase("output_path");
  auto plan = base("immutable local experiment plan");
  plan["sections"]["choices"] = section(choices);
  plan["sections"]["provenance"] = section(
      {{"engine_version", version},
       {"scope", "private local SBV artifacts; no remote or SOV job claim"}});
  plan = seal_result(plan);
  validate_result(plan);
  const auto plan_hash = plan.at("content_sha256");
  const auto plan_path = root + "/experiment-plan.json";
  const auto output = str(p.at("output_path"));
  need(output != plan_path && output != root + "/experiment.lock",
       "summary cannot replace journal");
  for (const auto &name : names)
    for (const auto *suffix : {".claim.json", ".receipt.json", ".result.json"})
      need(output != root + "/" + name + suffix,
           "summary collides with trial artifact");
  require_new_file(output);
  auto dir = directory(root);
  Handle lock(::openat(dir.fd, "experiment.lock",
                       O_RDWR | O_CREAT | O_NOFOLLOW | O_CLOEXEC, 0600));
  need(lock.fd >= 0, "experiment lock unavailable");
  struct stat st{};
  need(::fstat(lock.fd, &st) == 0 && S_ISREG(st.st_mode) && st.st_nlink == 1 &&
           st.st_uid == ::geteuid() && (st.st_mode & 0077) == 0,
       "private experiment lock required");
  need(::flock(lock.fd, LOCK_EX | LOCK_NB) == 0,
       "experiment already active or lock unavailable");
  if (exists(dir.fd, "experiment-plan.json"))
    need(load(plan_path, end) == plan,
         "existing experiment identity/plan/version differs; select a separate "
         "directory");
  else {
    for (const auto &name : names)
      for (const auto *suffix :
           {".claim.json", ".receipt.json", ".result.json"})
        need(!exists(dir.fd, name + suffix),
             "unowned trial artifacts exist before plan");
    save(plan_path, plan, end);
  }
  std::vector<Json> rows(trials.size());
  std::atomic<std::size_t> cursor{0}, executed{0}, reused{0};
  std::atomic<bool> stop{false};
  std::exception_ptr failure;
  std::mutex errors;
  const auto actual = std::min<std::uint64_t>(workers, trials.size());
  auto work = [&] {
    try {
      for (;;) {
        if (stop)
          break;
        auto i = cursor.fetch_add(1);
        if (i >= trials.size())
          break;
        deadline(end);
        const auto &t = trials[i];
        auto name = str(t.at("id"));
        Json row{{"id", name},        {"state", "not_started"},
                 {"reason", ""},      {"request_sha256", nullptr},
                 {"result", nullptr}, {"journal", nullptr}};
        if (t.at("state") == "pruned") {
          row["state"] = "pruned";
          row["reason"] = t.at("reason");
          rows[i] = row;
          continue;
        }
        const auto &request = requests[i];
        const auto request_hash = e::sha256_hex(request.dump());
        row["request_sha256"] = request_hash;
        auto claim = base("local experiment trial claim");
        claim["sections"]["search"] = section({{"plan_sha256", plan_hash},
                                               {"trial_id", name},
                                               {"request_sha256", request_hash},
                                               {"state", "claimed"}});
        claim = seal_result(claim);
        const auto claim_path = root + "/" + name + ".claim.json",
                   receipt_path = root + "/" + name + ".receipt.json";
        row["journal"] = {{"claim", claim_path}, {"receipt", receipt_path}};
        const bool claimed = exists(dir.fd, name + ".claim.json");
        if (claimed)
          need(load(claim_path, end) == claim, "trial claim identity mismatch");
        else
          need(!exists(dir.fd, name + ".receipt.json") &&
                   !exists(dir.fd, name + ".result.json"),
               "trial result/receipt has no claim");
        if (claimed && exists(dir.fd, name + ".receipt.json")) {
          auto stored = load(receipt_path, end);
          const auto &record = stored.at("sections").at("search").at("data");
          keys(record, {"plan_sha256", "trial_id", "request_sha256", "state",
                        "failure_code", "result"});
          need(record.at("plan_sha256") == plan_hash &&
                   record.at("trial_id") == name &&
                   record.at("request_sha256") == request_hash,
               "trial receipt identity mismatch");
          const auto state = str(record.at("state"));
          need(state == "completed" || state == "failed",
               "trial receipt state invalid");
          if (state == "completed") {
            const auto &receipt = record.at("result");
            need(receipt.at("path") == request.at("output_path"),
                 "trial result path mismatch");
            auto bytes = read_file(str(receipt.at("path")), end);
            auto result =
                e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
            validate_result(result);
            need(result.at("content_sha256") == receipt.at("content_sha256") &&
                     e::sha256_hex(bytes) == str(receipt.at("file_sha256")) &&
                     dec(bytes.size()) == str(receipt.at("bytes")),
                 "completed trial result changed");
            row["result"] = receipt;
          } else {
            need(record.at("result").is_null() &&
                     !exists(dir.fd, name + ".result.json"),
                 "failed trial has unexpected result");
          }
          row["state"] = state;
          row["reason"] = record.at("failure_code");
          ++reused;
        } else if (claimed) {
          row["state"] = "ambiguous";
          row["reason"] = "prior claim has no committed receipt; inspect "
                          "artifacts; no automatic retry or result adoption";
        } else {
          save(claim_path, claim, end);
          ++executed;
          Json record{{"plan_sha256", plan_hash},
                      {"trial_id", name},
                      {"request_sha256", request_hash},
                      {"state", "completed"},
                      {"failure_code", ""},
                      {"result", nullptr}};
          try {
            record["result"] =
                symphony::sbv::dispatch(str(t.at("operation")), request, end);
          } catch (const e::Error &error) {
            record["state"] = "failed";
            record["failure_code"] = error.code();
          } catch (const std::exception &) {
            record["state"] = "failed";
            record["failure_code"] = "sbv.trial_exception";
          }
          // A failed process can still have published its result before
          // reporting an uncertain filesystem outcome. Preserve the claim
          // without a receipt.
          if (record.at("state") == "failed" &&
              exists(dir.fd, name + ".result.json")) {
            row["state"] = "ambiguous";
            row["reason"] =
                "child failed after result appeared; no automatic adoption";
          } else {
            auto receipt = base("local experiment trial completion");
            receipt["sections"]["search"] = section(record);
            save(receipt_path, receipt, end);
            row["state"] = record.at("state");
            row["reason"] = record.at("failure_code");
            row["result"] = record.at("result");
          }
        }
        if (p.at("on_failure") == "stop" && row.at("state") != "completed")
          stop = true;
        rows[i] = std::move(row);
      }
    } catch (...) {
      std::lock_guard guard(errors);
      if (!failure)
        failure = std::current_exception();
      stop = true;
    }
  };
  std::vector<std::jthread> pool;
  try {
    for (std::uint64_t i = 0; i < actual; ++i)
      pool.emplace_back(work);
  } catch (...) {
    stop = true;
    throw;
  }
  pool.clear();
  if (failure)
    std::rethrow_exception(failure);
  deadline(end);
  Json ledger = Json::array();
  std::map<std::string, std::size_t> counts{{"completed", 0},
                                            {"failed", 0},
                                            {"pruned", 0},
                                            {"ambiguous", 0},
                                            {"not_started", 0}};
  for (std::size_t i = 0; i < rows.size(); ++i) {
    if (rows[i].is_null())
      rows[i] = {{"id", trials[i].at("id")},
                 {"state", "not_started"},
                 {"reason", "ordered stop-on-failure prevented claim"},
                 {"request_sha256", nullptr},
                 {"result", nullptr},
                 {"journal", nullptr}};
    ++counts.at(str(rows[i].at("state")));
    ledger.push_back(rows[i]);
  }
  auto result = base("durable private local SBV experiment observation");
  if (counts["failed"] || counts["ambiguous"] || counts["not_started"])
    result["status"] = "partial";
  auto &s = result["sections"];
  Json summary = Json::object();
  for (const auto &[state, n] : counts)
    summary[state] = dec(n);
  s["summary"] = section(summary);
  s["search"] = section({{"experiment_id", id},
                         {"plan_sha256", plan_hash},
                         {"plan_path", plan_path},
                         {"trials", ledger},
                         {"executed_this_invocation", dec(executed.load())},
                         {"reused_receipts", dec(reused.load())}});
  s["choices"] = section(choices);
  s["resources"] =
      section({{"backend", "cpu"},
               {"requested_workers", dec(workers)},
               {"actual_outer_workers", dec(actual)},
               {"maximum_child_workers", dec(inner_max)},
               {"selected_worker_product", dec(workers * inner_max)}});
  s["provenance"] =
      section({{"engine_version", version},
               {"scope", "local private filesystem artifacts; no daemon, "
                         "remote execution or SOV authority"},
               {"provider_requests", "0"},
               {"additional_spend_usd", "0"}});
  s["diagnostics"] = section(Json::array(
      {"A process-scoped exclusive lock serializes invocations for one "
       "experiment directory; process death releases the lock.",
       "Plan, per-trial claim, per-trial completion and child result are "
       "immutable and separately queryable through qxctl.",
       "A claim without a committed receipt remains ambiguous and is never "
       "automatically retried or adopted; use a separately identified "
       "experiment after review.",
       "Reconciliation verifies exact plan/version, request hash, source "
       "result content and file bytes before reusing completed trials.",
       "This is bounded local native trial execution; optimizer proposals, "
       "holdout history and pruning decisions remain user supplied."}));
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "experiment", end);
}
} // namespace symphony::sbv::detail
