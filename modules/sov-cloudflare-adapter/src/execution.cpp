#include "execution.hpp"
#include <set>
namespace cloudflare {
namespace n = symphony::ssiag::native;
namespace {
const char *attempt_protocol = "symphony.sov.cloudflare-resource-attempt.v1";
const char *observation_protocol = "symphony.sov.cloudflare-observation.v1";
Json profile(const std::string &op) {
  for (const auto &p : owner().at("resource_profiles"))
    if (p.at("operation") == op)
      return p;
  n::need(false);
  return nullptr;
}
void journal_valid(const Json &j, const std::string &d) {
  n::fields(j, {"protocol", "plan_digest", "stage", "outcome", "effect_state",
                "resource_id", "credential_ref", "authority_ref", "audit_ref",
                "observation_ref"});
  n::need(j.at("protocol") == attempt_protocol && j.at("plan_digest") == d);
  n::need(std::set<std::string>{"prepared", "admitted", "dispatch_started",
                                "observed", "settled"}
              .contains(j.at("stage")));
  n::need(std::set<std::string>{"completed", "rejected", "aborted", "failed",
                                "unknown"}
              .contains(j.at("outcome")));
  n::need(std::set<std::string>{"not_applied", "not_applicable",
                                "applied_verified", "unresolved"}
              .contains(j.at("effect_state")));
  for (auto k : {"observation_ref", "audit_ref"})
    n::need(j.at(k).is_null() || n::tagged(j.at(k)));
  if (!j.at("resource_id").is_null())
    n::need(std::regex_match(j.at("resource_id").get<std::string>(),
                             std::regex("[a-f0-9-]{32,36}")));
  if (!j.at("credential_ref").is_null())
    n::need(
        std::regex_match(j.at("credential_ref").get<std::string>(),
                         std::regex("ssiag-native:[a-z0-9][a-z0-9_-]{0,63}")));
}
Json projection(const Json &row, const std::string &op,
                const std::string &secret) {
  n::need(row.is_object());
  Json out = Json::object();
  if (op == "tunnels.config.get") {
    n::need(row.contains("config"));
    auto p = profile("tunnels.config.update");
    Json parameters = p.at("template");
    parameters["body"] = Json{{"config", row.at("config")}};
    // Admit only the supported ingress fields; unknown provider fields stay
    // private.
    Json ingress = Json::array();
    for (const auto &r : row.at("config").at("ingress"))
      ingress.push_back(Json{{"hostname", r.value("hostname", Json(nullptr))},
                             {"service", r.at("service")}});
    out["config"] = Json{{"ingress", ingress}};
    parameters["body"] = out;
    validate(parameters, p.at("parameters_schema"));
  } else {
    n::need(row.contains("id") && row.at("id").is_string());
    auto rid = row.at("id").get<std::string>();
    n::need(std::regex_match(rid,
                             std::regex(op.starts_with("dns.")
                                            ? "[a-f0-9]{32}"
                                            : "[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]"
                                              "{4}-[a-f0-9]{4}-[a-f0-9]{12}")));
    out["id"] = rid;
    auto create = op.substr(0, op.rfind('.')) + ".create";
    const auto p = profile(create);
    const auto &schema = p.at("parameters_schema").at("properties").at("body");
    for (const auto &[key, rule] : schema.at("properties").items())
      if (row.contains(key)) {
        validate(row.at(key), rule);
        out[key] = row.at(key);
      }
  }
  n::need(out.dump().find(secret) == std::string::npos);
  return out;
}
Json scope(const Json &input) {
  return Json{{"operation", input.at("resource_operation")},
              {"account_id", input.at("context").at("account_id")},
              {"zone_id", input.at("context").at("zone_id")},
              {"resource_id", input.at("parameters").at("resource_id")},
              {"toolchain_id", input.at("context").at("toolchain_id")}};
}
Json read_input(const Json &in, const std::string &op, const Json &id) {
  auto p = profile(op);
  auto read = in;
  read["resource_operation"] = op;
  read["parameters"] = p.at("template");
  read["parameters"]["resource_id"] = id;
  return read;
}
std::string get_operation(const std::string &op) {
  if (op == "tunnels.config.update")
    return "tunnels.config.get";
  return op.substr(0, op.rfind('.')) + ".get";
}
std::string list_operation(const std::string &op) {
  return op.substr(0, op.rfind('.')) + ".list";
}
bool subset(const Json &expected, const Json &actual) {
  if (expected.is_object()) {
    if (!actual.is_object())
      return false;
    for (const auto &[k, v] : expected.items())
      if (!actual.contains(k) || !subset(v, actual.at(k)))
        return false;
    return true;
  }
  return expected == actual;
}
Json observe(const Json &in, const HTTPRunner &run, const std::string &token,
             const fs::path &package, const fs::path &scratch,
             std::int64_t deadline, std::string *sensitive = nullptr) {
  auto query = in;
  auto op = in.at("resource_operation").get<std::string>();
  const bool listing = op.ends_with(".list");
  Json rows = Json::array();
  bool complete = !listing;
  int first =
          listing ? in.at("parameters").at("page").at("page").get<int>() : 1,
      last = first, next = first;
  std::set<std::string> seen;
  Json total = nullptr;
  const int maximum =
      listing ? in.at("execution").at("max_pages").get<int>() : 1;
  for (int count = 0; count < maximum; ++count) {
    n::need(n::now() < deadline);
    if (listing)
      query["parameters"]["page"]["page"] = next;
    auto r = run(resource_preview(query), token, package, scratch, deadline);
    n::need(r.attempted && r.status >= 200 && r.status < 300 &&
            r.body.is_object() && r.body.value("success", false) &&
            r.body.contains("result"));
    auto data = r.body.at("result");
    if (!listing) {
      if (sensitive)
        *sensitive = data.dump();
      rows = data.is_null() ? Json(nullptr) : projection(data, op, token);
      break;
    }
    n::need(data.is_array());
    auto per = in.at("parameters").at("page").at("per_page").get<int>();
    n::need(data.size() <= std::size_t(per));
    for (const auto &row : data) {
      auto safe = projection(row, op, token);
      n::need(seen.insert(safe.at("id").get<std::string>()).second);
      rows.push_back(safe);
    }
    last = next;
    auto info = r.body.value("result_info", Json::object());
    n::need(info.is_object());
    if (info.contains("page"))
      n::need(info.at("page") == next);
    if (info.contains("per_page"))
      n::need(info.at("per_page") == per);
    if (info.contains("count"))
      n::need(info.at("count") == data.size());
    if (info.contains("total_count")) {
      n::need(info.at("total_count").is_number_unsigned() ||
              info.at("total_count").is_number_integer());
      n::need(info.at("total_count") >= 0);
      if (!total.is_null())
        n::need(total == info.at("total_count"));
      total = info.at("total_count");
    }
    bool end = data.size() < std::size_t(per);
    if (info.contains("total_pages")) {
      n::need(info.at("total_pages").is_number_integer() &&
              info.at("total_pages") >= 0);
      end = next >= info.at("total_pages").get<int>();
    }
    n::need(next < 2147483647);
    ++next;
    if (end) {
      complete = first == 1;
      if (complete && !total.is_null())
        n::need(total == rows.size());
      break;
    }
  }
  if (listing)
    std::sort(rows.begin(), rows.end(), [](const Json &a, const Json &b) {
      return a.at("id") < b.at("id");
    });
  Json snapshot{{"scope", scope(in)}, {"data", rows}};
  Json record{
      {"protocol", observation_protocol},
      {"observed_unix_ms", std::to_string(n::now())},
      {"snapshot", snapshot},
      {"state_digest",
       digest(snapshot, "symphony.sov.cloudflare-resource-state.v1")},
      {"coverage",
       {{"first_page", first},
        {"last_page", last},
        {"complete", complete},
        {"next_page", listing && !complete ? Json(next) : Json(nullptr)}}}};
  n::need(record.dump().size() < 4 * 1024 * 1024);
  return record;
}
std::string retain(const fs::path &store, const Json &record) {
  auto d = digest(record, observation_protocol);
  auto path = store / "observations" / (d.substr(7) + ".json");
  if (!fs::exists(path))
    n::put(path, record.dump());
  return d;
}
} // namespace
void resource_barrier(const fs::path &root) {
  if (!fs::exists(root / "resource-attempts"))
    return;
  n::private_dir(root / "resource-attempts");
  for (const auto &f : fs::directory_iterator(root / "resource-attempts")) {
    if (f.path().extension() != ".json")
      continue; // Unpublished atomic-write temporaries are not admitted
                // journals.
    auto d = "sha256:" + f.path().stem().string();
    n::need(n::tagged(d));
    auto j = parse(n::read(f.path(), 65536, true));
    journal_valid(j, d);
    n::need(j.at("stage") == "settled" && j.at("effect_state") != "unresolved");
  }
}
Json execute_resource(const Json &request, const Json &plan,
                      const fs::path &package, Json &result, bool &started,
                      RuntimeHooks *hooks) {
  auto in = plan.at("input");
  const auto &e = in.at("execution");
  n::need(!e.is_null());
  const auto op = in.at("resource_operation").get<std::string>();
  const bool mutation = profile(op).at("method") != "GET";
  const auto command = request.at("operation").get<std::string>();
  const auto d = request.at("input").at("plan_digest").get<std::string>();
  auto root = fs::path(in.at("store").get<std::string>());
  for (auto sub : {"resource-attempts", "observations"})
    n::private_dir(root / sub, true);
  auto path = root / "resource-attempts" / (d.substr(7) + ".json");
  Json journal = nullptr;
  if (fs::exists(path)) {
    journal = parse(n::read(path, 65536, true));
    journal_valid(journal, d);
  }
  auto data = [&](const Json &observation) {
    return Json{{"plan_digest", d},
                {"attempt", journal},
                {"observation", observation},
                {"credential_ref", journal.is_null()
                                       ? Json(nullptr)
                                       : journal.at("credential_ref")},
                {"precondition_mode", mutation ? "preflight_only" : "none"},
                {"automatic_retry", false}};
  };
  if (command == "resource.status")
    return data(nullptr);
  const bool recovery = command == "resource.reconcile";
  n::need(recovery ? !journal.is_null() : journal.is_null());
  if (recovery && journal.at("stage") == "settled" &&
      journal.at("effect_state") != "unresolved")
    return data(nullptr);
  if (!recovery) {
    if (op == "tunnels.token.get" || op == "tunnels.get" ||
        op == "tunnels.create")
      n::need(!e.at("result_ref").is_null());
    n::need(std::stoll(plan.at("expires_unix_ms").get<std::string>()) >
            n::now());
    resource_barrier(root);
    n::need(
        (mutation && command == "resource.apply") ||
        (!mutation && command == (op.ends_with(".list") ? "resource.list"
                                                        : "resource.inspect")));
    journal = Json{{"protocol", attempt_protocol},
                   {"plan_digest", d},
                   {"stage", "prepared"},
                   {"outcome", "rejected"},
                   {"effect_state", "not_applied"},
                   {"resource_id", in.at("parameters").at("resource_id")},
                   {"credential_ref", nullptr},
                   {"authority_ref", nullptr},
                   {"audit_ref", nullptr},
                   {"observation_ref", nullptr}};
    n::atomic(path, journal);
  }
  const bool prior_dispatched =
      recovery && (journal.at("stage") == "dispatch_started" ||
                   journal.at("stage") == "observed" ||
                   journal.at("effect_state") == "unresolved");
  if (recovery && !prior_dispatched) {
    journal["stage"] = "settled";
    journal["outcome"] = "completed";
    journal["effect_state"] = "not_applied";
    n::atomic(path, journal);
    return data(nullptr);
  }
  // A lost sensitive import acknowledgment is not settled by repeating the
  // endpoint or by discarding its required result channel during recovery.
  if (recovery && prior_dispatched && journal.at("credential_ref").is_null() &&
      (op == "tunnels.token.get" || op == "tunnels.get" ||
       op == "tunnels.create")) {
    result["outcome"] = "unknown";
    result["effect_state"] = "unresolved";
    result["error_code"] = "unresolved_attempt";
    result["next_action"] = "inspect_native_import";
    return data(nullptr);
  }
  bool mutated = prior_dispatched;
  Json observed = nullptr;
  std::unique_ptr<CredentialSession> credentials;
  std::string imported;
  try {
    auto binding = execution_binding(e);
    n::need(binding.at("config").at("consumer_digest") ==
            request.at("installation_ref").at("executable_digest"));
    auto deadline = n::now() + 55000;
    // Reconciliation is an independently authorized observation, never the
    // original mutation.
    auto authority_op =
        recovery ? (journal.at("resource_id").is_null() ? list_operation(op)
                                                        : get_operation(op))
                 : op;
    Json intent{{"protocol", "symphony.ssiag.cloudflare-use.v1"},
                {"request_id", n::uuid()},
                {"tops_id", binding.at("config").at("tops_id")},
                {"plan_digest", d},
                {"adapter_digest",
                 request.at("installation_ref").at("executable_digest")},
                {"toolchain_digest", in.at("context").at("toolchain_id")},
                {"credential_ref", in.at("context").at("credential_ref")},
                {"result_ref", recovery ? Json(nullptr) : e.at("result_ref")},
                {"operation", authority_op},
                {"account_id", in.at("context").at("account_id")},
                {"zone_id", in.at("context").at("zone_id")},
                {"resource_id", journal.at("resource_id")},
                {"deadline_unix_ms", std::to_string(deadline)}};
    credentials = std::make_unique<CredentialSession>(e, intent);
    deadline = std::min(
        deadline,
        std::stoll(
            credentials->grant.at("deadline_unix_ms").get<std::string>()));
    journal["authority_ref"] =
        "ssiag:" + credentials->grant.at("policy_digest").get<std::string>();
    journal["audit_ref"] =
        credentials->grant.at("receipt").at("commit").at("event_digest");
    if (!recovery)
      journal["stage"] = "admitted";
    n::atomic(path, journal);
    HTTPRunner run = hooks && hooks->http ? hooks->http : run_http;
    const auto &token = credentials->secret.bytes;
    auto read = [&](const Json &query) {
      started = true;
      return observe(query, run, token, package, root / "staging", deadline,
                     (!mutation && !recovery && op == "tunnels.get") ? &imported
                                                                     : nullptr);
    };
    if (!mutation) {
      journal["stage"] = "dispatch_started";
      journal["effect_state"] = "unresolved";
      n::atomic(path, journal);
      if (op == "tunnels.token.get") {
        n::need(!e.at("result_ref").is_null() && !recovery);
        started = true;
        auto r = run(resource_preview(in), token, package, root / "staging",
                     deadline);
        n::need(r.attempted && r.status == 200 &&
                r.body.value("success", false) &&
                r.body.at("result").is_string());
        imported = r.body.at("result").get<std::string>();
        n::need(!imported.empty() && imported.size() <= 16384);
      } else {
        observed = read(in);
        if (op == "tunnels.get") {
          n::need(!e.at("result_ref")
                       .is_null()); /* Raw sensitive fields travel only to the
                                       native credential owner. */
        }
      }
      journal["outcome"] = "completed";
      journal["effect_state"] = "not_applicable";
    } else {
      auto method = profile(op).at("method").get<std::string>();
      const bool create = method == "POST";
      if (!recovery) {
        n::need(!in.at("expected_observation").is_null());
        auto ref = in.at("expected_observation").get<std::string>();
        auto prior =
            parse(n::read(root / "observations" / (ref.substr(7) + ".json"),
                          4 * 1024 * 1024, true));
        n::need(digest(prior, observation_protocol) == ref &&
                prior.at("coverage").at("complete") == true);
        auto query = read_input(
            in, create ? list_operation(op) : get_operation(op),
            create ? Json(nullptr) : in.at("parameters").at("resource_id"));
        auto fresh = read(query);
        n::need(fresh.at("coverage").at("complete") == true &&
                prior.at("snapshot").at("scope") ==
                    fresh.at("snapshot").at("scope") &&
                prior.at("state_digest") == fresh.at("state_digest"));
        journal["stage"] = "dispatch_started";
        journal["outcome"] = "unknown";
        journal["effect_state"] = "unresolved";
        n::atomic(path, journal);
        mutated = true;
        started = true;
        auto response = run(resource_preview(in), token, package,
                            root / "staging", deadline);
        if (!response.attempted) {
          n::need(response.clean_exit);
          mutated = false;
          journal["outcome"] = "aborted";
          journal["effect_state"] = "not_applied";
        } else {
          n::need(response.status >= 200 && response.status < 300 &&
                  (response.status == 204 ||
                   (response.body.is_object() &&
                    response.body.value("success", false))));
          if (create) {
            if (op == "tunnels.create") {
              n::need(!e.at("result_ref").is_null());
              imported = response.body.at("result").dump();
            }
            auto safe = projection(response.body.at("result"), op, token);
            journal["resource_id"] = safe.at("id");
            n::atomic(path, journal);
          }
        }
      }
      if (mutated) {
        n::need(!journal.at("resource_id").is_null());
        if (method == "DELETE") {
          auto query = read_input(in, list_operation(op), nullptr);
          observed = read(query);
          n::need(observed.at("coverage").at("complete") == true);
          for (const auto &row : observed.at("snapshot").at("data"))
            n::need(row.at("id") != journal.at("resource_id"));
        } else {
          auto query =
              read_input(in, get_operation(op), journal.at("resource_id"));
          observed = read(query);
          n::need(subset(in.at("parameters").at("body"),
                         observed.at("snapshot").at("data")));
        }
        journal["outcome"] = "completed";
        journal["effect_state"] = "applied_verified";
      }
    }
    if (!observed.is_null())
      journal["observation_ref"] = retain(root, observed);
    journal["stage"] = "observed";
    n::atomic(path, journal);
    auto finished = credentials->finish(
        journal.at("outcome") == "completed" ? "completed" : "failed",
        imported.empty() ? nullptr : &imported);
    n::wipe(imported);
    journal["credential_ref"] = finished.at("result_ref");
    journal["audit_ref"] =
        finished.at("completion").at("receipt").at("commit").at("event_digest");
    journal["stage"] = "settled";
    n::atomic(path, journal);
  } catch (...) {
    n::wipe(imported);
    journal["outcome"] = mutated ? "unknown" : "failed";
    journal["effect_state"] = mutated ? "unresolved" : "not_applied";
    journal["stage"] = "settled";
    try {
      if (credentials)
        credentials->finish(mutated ? "unknown" : "failed");
    } catch (...) {
      if (started) {
        journal["outcome"] = "unknown";
        journal["effect_state"] = "unresolved";
      }
    }
    n::atomic(path, journal);
    result["error_code"] =
        mutated ? "unresolved_attempt" : "unavailable_contract";
    result["next_action"] = "reconcile";
  }
  result["outcome"] = journal.at("outcome");
  result["effect_state"] = journal.at("effect_state");
  result["authority_ref"] = journal.at("authority_ref");
  result["audit_ref"] = journal.at("audit_ref");
  if (result.at("outcome") != "completed" && result.at("error_code").is_null())
    result["error_code"] = "child_failure";
  return data(observed);
}
} // namespace cloudflare
