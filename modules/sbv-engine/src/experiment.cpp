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
  std::map<std::string, std::size_t> indices;
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
                                         "fit",
                                         "predict",
                                         "dataset_execute",
                                         "research_history",
                                         "compose_economics",
                                         "generate_census"};
  for (const auto &t : trials) {
    keys_optional(t,
                  {"id", "state", "operation", "request", "reason",
                   "parameters", "lineage"},
                  {"depends_on", "bindings"});
    auto name = str(t.at("id"));
    need(id_ok(name) && names.insert(name).second,
         "unique safe trial id required");
    need(t.at("state") == "ready" || t.at("state") == "pruned",
         "trial must be ready or caller-pruned");
    need(str(t.at("reason")).size() <= 4096 && t.at("parameters").is_object() &&
             t.at("lineage").is_object(),
         "trial metadata required");
    indices.emplace(name, requests.size());
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
  // Admit the entire graph and destination slots before creating the journal.
  // Dependencies can point forward in the caller's list; results retain that
  // list order.
  std::vector<std::vector<std::size_t>> dependencies(trials.size());
  std::vector<std::size_t> levels(trials.size(), trials.size());
  for (std::size_t i = 0; i < trials.size(); ++i) {
    const auto &t = trials[i];
    const auto deps = t.value("depends_on", Json::array());
    const auto bindings = t.value("bindings", Json::array());
    need(deps.is_array() && bindings.is_array(),
         "dependency and binding arrays required");
    std::set<std::string> unique;
    for (const auto &dep : deps) {
      const auto id = str(dep);
      need(indices.contains(id) && indices.at(id) != i &&
               unique.insert(id).second,
           "dependency must name a unique other trial");
      dependencies[i].push_back(indices.at(id));
    }
    need(t.at("state") != "pruned" || (deps.empty() && bindings.empty()),
         "pruned trials have no dependencies or bindings");
    auto slots = requests[i];
    for (const auto &b : bindings) {
      keys(b, {"target_pointer", "trial_id", "result_pointer"});
      need(unique.contains(str(b.at("trial_id"))),
           "binding trial must be an explicit dependency");
      const auto target = str(b.at("target_pointer")),
                 source = str(b.at("result_pointer"));
      need(!target.empty() && target[0] == '/' &&
               (source.empty() || source[0] == '/'),
           "binding pointers must be RFC6901 pointers; request root cannot be "
           "replaced");
      try {
        Json::json_pointer destination(target), selection(source);
        (void)selection;
        need(slots.at(destination).is_null(),
             "binding target must be a unique existing null slot");
        slots.at(destination) = Json::object();
      } catch (const Json::exception &) {
        need(false, "invalid binding pointer or missing destination slot");
      }
    }
  }
  std::vector<std::vector<std::size_t>> waves;
  std::size_t remaining = trials.size();
  while (remaining) {
    std::vector<std::size_t> wave;
    for (std::size_t i = 0; i < trials.size(); ++i)
      if (levels[i] == trials.size() &&
          std::all_of(dependencies[i].begin(), dependencies[i].end(),
                      [&](auto dep) { return levels[dep] < waves.size(); }))
        wave.push_back(i);
    need(!wave.empty(), "experiment dependency cycle");
    for (auto i : wave)
      levels[i] = waves.size();
    remaining -= wave.size();
    waves.push_back(std::move(wave));
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
  std::size_t actual = 0;
  for (const auto &wave : waves)
    actual = std::max(actual, std::min<std::size_t>(workers, wave.size()));
  const std::vector<std::size_t> *active_wave = nullptr;
  auto work = [&] {
    try {
      for (;;) {
        if (stop)
          break;
        auto offset = cursor.fetch_add(1);
        if (offset >= active_wave->size())
          break;
        const auto i = active_wave->at(offset);
        deadline(end);
        const auto &t = trials[i];
        auto name = str(t.at("id"));
        Json row{{"id", name},
                 {"state", "not_started"},
                 {"reason", ""},
                 {"request_sha256", nullptr},
                 {"result", nullptr},
                 {"journal", nullptr},
                 {"dependencies", Json::array()},
                 {"bindings", Json::array()}};
        if (t.at("state") == "pruned") {
          row["state"] = "pruned";
          row["reason"] = t.at("reason");
          rows[i] = row;
          continue;
        }
        auto request = requests[i];
        bool blocked = false;
        for (auto dep : dependencies[i]) {
          const auto &parent = rows[dep];
          need(!parent.is_null(), "dependency scheduler invariant");
          row["dependencies"].push_back({{"trial_id", parent.at("id")},
                                         {"state", parent.at("state")},
                                         {"result", parent.at("result")}});
          blocked |= parent.at("state") != "completed";
        }
        if (!blocked) {
          // Recheck referenced immutable bytes before binding their content
          // identity.
          std::map<std::string, Json> sources;
          for (const auto &b : t.value("bindings", Json::array())) {
            const auto parent_id = str(b.at("trial_id"));
            const auto &receipt = rows[indices.at(parent_id)].at("result");
            if (!sources.contains(parent_id)) {
              const auto bytes = read_file(str(receipt.at("path")), end);
              auto result =
                  e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
              validate_result(result);
              need(result.at("content_sha256") ==
                           receipt.at("content_sha256") &&
                       e::sha256_hex(bytes) == str(receipt.at("file_sha256")) &&
                       dec(bytes.size()) == str(receipt.at("bytes")),
                   "dependency result changed before binding");
              sources.emplace(parent_id, std::move(result));
            }
            const Json::json_pointer selection(str(b.at("result_pointer")));
            bool selected = false;
            try {
              selected = sources.at(parent_id).contains(selection);
            } catch (
                const Json::exception &) { /* invalid index for this result */
            }
            if (!selected) {
              blocked = true;
              row["reason"] = "dependency result pointer is absent";
              break;
            }
            Json reference{{"path", receipt.at("path")},
                           {"expected_sha256", receipt.at("content_sha256")},
                           {"pointer", b.at("result_pointer")}};
            request.at(Json::json_pointer(str(b.at("target_pointer")))) =
                reference;
            row["bindings"].push_back(
                {{"target_pointer", b.at("target_pointer")},
                 {"trial_id", parent_id},
                 {"reference", reference}});
          }
        }
        if (blocked) {
          // Never leave or reuse a claimed dependent when its required evidence
          // is absent.
          for (const auto *suffix :
               {".claim.json", ".receipt.json", ".result.json"})
            need(!exists(dir.fd, name + suffix),
                 "blocked trial has unexpected journal artifacts");
          row["state"] = "blocked";
          if (row.at("reason") == "")
            row["reason"] = "one or more dependencies did not complete";
          if (p.at("on_failure") == "stop")
            stop = true;
          rows[i] = std::move(row);
          continue;
        }
        const auto request_hash = e::sha256_hex(request.dump());
        row["request_sha256"] = request_hash;
        auto claim = base("local experiment trial claim");
        claim["sections"]["search"] =
            section({{"plan_sha256", plan_hash},
                     {"trial_id", name},
                     {"request_sha256", request_hash},
                     {"state", "claimed"},
                     {"resolved_request", request},
                     {"dependencies", row.at("dependencies")},
                     {"bindings", row.at("bindings")}});
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
  for (const auto &wave : waves) {
    if (stop)
      break;
    active_wave = &wave;
    cursor = 0;
    std::vector<std::jthread> pool;
    try {
      for (std::size_t i = 0; i < std::min<std::size_t>(workers, wave.size());
           ++i)
        pool.emplace_back(work);
    } catch (...) {
      stop = true;
      throw;
    }
    // Joining a complete wave publishes all parent rows before a child can read
    // them.
    pool.clear();
    if (failure)
      std::rethrow_exception(failure);
  }
  deadline(end);
  Json ledger = Json::array();
  std::map<std::string, std::size_t> counts{
      {"completed", 0}, {"failed", 0},      {"pruned", 0},
      {"ambiguous", 0}, {"not_started", 0}, {"blocked", 0}};
  for (std::size_t i = 0; i < rows.size(); ++i) {
    if (rows[i].is_null())
      rows[i] = {{"id", trials[i].at("id")},
                 {"state", "not_started"},
                 {"reason", "ordered stop-on-failure prevented claim"},
                 {"request_sha256", nullptr},
                 {"result", nullptr},
                 {"journal", nullptr},
                 {"dependencies", Json::array()},
                 {"bindings", Json::array()}};
    ++counts.at(str(rows[i].at("state")));
    ledger.push_back(rows[i]);
  }
  auto result = base("durable private local SBV experiment observation");
  if (counts["failed"] || counts["ambiguous"] || counts["not_started"] ||
      counts["blocked"])
    result["status"] = "partial";
  auto &s = result["sections"];
  Json summary = Json::object();
  for (const auto &[state, n] : counts)
    summary[state] = dec(n);
  s["summary"] = section(summary);
  Json planned_waves = Json::array();
  for (const auto &wave : waves) {
    Json ids = Json::array();
    for (auto i : wave)
      ids.push_back(trials[i].at("id"));
    planned_waves.push_back(std::move(ids));
  }
  s["search"] = section({{"experiment_id", id},
                         {"plan_sha256", plan_hash},
                         {"plan_path", plan_path},
                         {"trials", ledger},
                         {"executed_this_invocation", dec(executed.load())},
                         {"reused_receipts", dec(reused.load())},
                         {"schedule", "stable_topological_waves_v1"},
                         {"wave_count", dec(waves.size())},
                         {"planned_waves", planned_waves}});
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
       "Explicit dependencies execute in stable topological waves; dependent "
       "claims bind resolved requests and immutable parent receipts. "
       "Noncompleted "
       "parents and absent selections block dependents without claiming them.",
       "Completed means producer execution committed; child result status and "
       "per-study availability remain visible and are not a quality gate.",
       "This is bounded local native trial execution; optimizer proposals, "
       "holdout history and pruning decisions remain user supplied."}));
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "experiment", end);
}
} // namespace symphony::sbv::detail
