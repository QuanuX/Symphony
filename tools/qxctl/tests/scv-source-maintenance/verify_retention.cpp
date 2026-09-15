#include "maintenance_support.hpp"
using namespace scv_maintenance;
constexpr int MAX_AGE = 86400;
constexpr const char *MEMBER = "limits";
constexpr const char *CLAIM = "maintenance-host-local-storage";
constexpr const char *QUOTE =
    "The host instances running App Platform containers do not provide "
    "persistent data storage.";
inline J capture_fields(const J &value) {
  return pick(value, {"source", "locator_id", "resolved_uri", "redirects",
                      "observed_at", "upstream_revision", "media_type", "body",
                      "completeness", "issues"});
}
inline J desired(const J &source) {
  return without(source,
                 {"protocol", "generation", "predecessor_digest", "digest"});
}
inline J source_tuple(const J &capture) {
  auto source = capture["source"];
  return J::array({source["family_id"], source["provider_id"],
                   source["source_id"], capture["locator_id"]});
}
struct RetentionCampaign {
  nt::Arguments args;
  fs::path out;
  std::string tops;
  J commands = array(), assertions = array(), summary;
  explicit RetentionCampaign(const nt::Arguments &value)
      : args(value), out(fs::absolute(args.require("out"))), tops(uuid()) {
    NT_REQUIRE(!fs::exists(out));
    private_dir(out);
    out = fs::canonical(out);
    summary = {
        {"status", "running"},
        {"started_at", utc()},
        {"native_version", VERSION},
        {"domain", args.get("domain", "scv")},
        {"tops_id", tops},
        {"network_requests", 0},
        {"synthetic_failure", "Authored fixture only; no live request or "
                              "provider outage is asserted."},
        {"recovery_scope", "Completed-job exact replay; no interrupted-write "
                           "recovery is claimed."},
        {"qxctl", fs::canonical(args.require("qxctl")).string()},
        {"qxctl_sha256", sha(nt::read(args.require("qxctl")))},
        {"prefix", fs::canonical(args.require("prefix")).string()},
        {"script_sha256", sha(nt::read(__FILE__))},
        {"old_capture_file_sha256", sha(nt::read(args.require("old-capture")))},
        {"fresh_capture_file_sha256",
         sha(nt::read(args.require("fresh-capture")))}};
    save();
  }
  void save() {
    summary["commands"] = commands.size();
    summary["assertions"] = assertions.size();
    nt::write_json(out / "COMMANDS.json", commands);
    nt::write_json(out / "ASSERTIONS.json", assertions);
    nt::write_json(out / "SUMMARY.json", summary);
  }
  void check(const std::string &label, bool condition) {
    assertions.push_back({{"assertion", label}, {"passed", condition}});
    save();
    nt::require(condition, label);
  }
  J call(const std::string &name, const std::vector<std::string> &command,
         const J &value = nullptr, bool corpus = false,
         const J &error = nullptr) {
    auto input = out / (name + "-input.json");
    if (!value.is_null())
      nt::write(input, encoded(value));
    std::vector<std::string> argv = {
        fs::canonical(args.require("qxctl")).string(), "scv"};
    argv.insert(argv.end(), command.begin(), command.end());
    argv.insert(argv.end(),
                {"--domain", args.get("domain", "scv"), "--version", VERSION,
                 "--prefix", fs::canonical(args.require("prefix")).string(),
                 "--json"});
    if (!value.is_null())
      argv.insert(argv.end(), {"--input", input.string()});
    if (corpus)
      argv.insert(argv.end(), {"--corpus-root", (out / "corpus-store").string(),
                               "--tops-id", tops});
    auto start = std::chrono::steady_clock::now();
    auto result = nt::run(argv, "", 180, {}, out);
    auto stdout_path = out / (name + "-stdout.json"),
         stderr_path = out / (name + "-stderr.txt");
    nt::write(stdout_path, result.stdout_text);
    nt::write(stderr_path, result.stderr_text);
    commands.push_back(
        {{"name", name},
         {"argv", argv},
         {"cwd", out.string()},
         {"exit_code", result.returncode},
         {"expected_error", error},
         {"elapsed_seconds", elapsed(start)},
         {"input", value.is_null() ? J(nullptr) : J(input.filename().string())},
         {"stdout", stdout_path.filename().string()},
         {"stderr", stderr_path.filename().string()},
         {"input_file_digest",
          value.is_null() ? J(nullptr) : J(sha(nt::read(input)))},
         {"stdout_file_digest", sha(result.stdout_text)},
         {"stderr_file_digest", sha(result.stderr_text)},
         {"input_bytes", value.is_null() ? 0 : fs::file_size(input)},
         {"stdout_bytes", result.stdout_text.size()}});
    save();
    auto parsed = read_unique(stdout_path);
    if (error.is_null())
      check(name + ": successful owner/adapter execution",
            result.returncode == 0);
    else
      check(name + ": published rejection",
            result.returncode != 0 &&
                parsed.value("protocol", std::string()) ==
                    "symphony.qxctl.error.v1" &&
                parsed.value("error", object()).value("code", J(nullptr)) ==
                    error);
    return parsed;
  }
  J imported(const std::string &name, const J &capture,
             const J &previous = nullptr, const std::string &corpus_id = "",
             const J &error = nullptr) {
    return call(name, {"corpus", "import"},
                {{"operation_id", name},
                 {"corpus_id", corpus_id.empty() ? name : corpus_id},
                 {"previous_snapshot_digest", previous},
                 {"members",
                  J::array({{{"member_id", MEMBER}, {"capture", capture}}})}},
                true, error);
  }
  J query(const std::string &name, const J &snapshot,
          const std::string &selection, const std::string &query_time) {
    return call(name, {"corpus", "query"},
                {{"snapshot_digest", snapshot},
                 {"member_ids", J::array({MEMBER})},
                 {"selection", selection},
                 {"query_time", query_time},
                 {"max_age_seconds", MAX_AGE}},
                true);
  }
  void member(const std::string &name, const J &result, const J &capture,
              const std::string &freshness, bool source_matches,
              const J &reasons) {
    check(name + ": exactly selected member",
          result["members"].size() == 1 &&
              result["members"][0]["member_id"] == MEMBER);
    auto member = result["members"][0], selected = member["selected"];
    check(name + ": preserved capture/source/time",
          selected["capture_digest"] == capture["digest"] &&
              selected["source_digest"] == capture["source"]["digest"] &&
              selected["observed_at"] == capture["observed_at"]);
    check(name + ": independent disposition and freshness",
          member["status"] == capture["completeness"] &&
              member["freshness"] == freshness &&
              member["source_revision_matches_latest"] == source_matches);
    check(name + ": exact reasons", member["reasons"] == reasons);
  }
  J graph(const std::string &name, const J &capture,
          bool include_claim = true) {
    J authored = array();
    if (include_claim) {
      check(name + ": independently selected literal anchor exists",
            capture["body"].get<std::string>().find(QUOTE) !=
                std::string::npos);
      authored = J::array(
          {{{"claim_id", CLAIM},
            {"subject", "do.app-platform.container"},
            {"predicate", "local-storage.persistent"},
            {"value",
             {{"type", "boolean"}, {"value", false}, {"unit", nullptr}}},
            {"scope",
             {{"provider", "do"},
              {"service", "app-platform"},
              {"storage", "host-local-filesystem"}}},
            {"statement_kind", "documented_fact"},
            {"evidence", J::array({{{"capture_digest", capture["digest"]},
                                    {"quote", QUOTE}}})},
            {"dependencies", array()},
            {"valid_from", nullptr},
            {"valid_until", nullptr}}});
    }
    auto knowledge =
        call(name + "-interpret", {"interpret"},
             {{"captures", J::array({capture})},
              {"claims", authored},
              {"interpreter_version", "source-maintenance-acceptance-v1"},
              {"selection_policy",
               {{"policy_id", "source-maintenance-one-day"},
                {"max_age_seconds", MAX_AGE},
                {"partial_capture", "exclude"},
                {"allowed_statement_kinds", J::array({"documented_fact"})}}}});
    return call(name + "-graph", {"graph"},
                {{"knowledge", J::array({knowledge})}});
  }
  void run() {
    auto old = read_unique(args.require("old-capture")),
         fresh = read_unique(args.require("fresh-capture"));
    auto time_now = utc();
    check("source1/source2 exact selected chain",
          source_tuple(old) == source_tuple(fresh) &&
              source_tuple(fresh) ==
                  J::array({"schv", "do", "do-app-platform-limits", "docs"}) &&
              old["source"]["generation"] == 1 &&
              fresh["source"]["generation"] == 2 &&
              fresh["source"]["predecessor_digest"] == old["source"]["digest"]);
    bool complete = true;
    for (const auto &c : J::array({old, fresh}))
      complete = complete && c["completeness"] == "complete" &&
                 instant(c["observed_at"]) <= instant(time_now);
    check("both supplied captures are complete and nonfuture", complete);
    check("historical capture is actually expired at campaign time",
          instant(time_now) >
              instant(old["observed_at"]) + std::chrono::seconds(MAX_AGE));
    call("owner-inspect", {"inspect"});
    for (const auto &[name, capture] : std::vector<std::pair<std::string, J>>{
             {"old", old}, {"fresh", fresh}}) {
      auto replay = call(name + "-capture-replay", {"capture-import"},
                         capture_fields(capture));
      check(name + ": native offline import reproduces exact capture",
            replay == capture);
    }
    auto failure = call(
        "synthetic-failure-import", {"capture-import"},
        {{"source", fresh["source"]},
         {"locator_id", fresh["locator_id"]},
         {"resolved_uri", fresh["resolved_uri"]},
         {"redirects", array()},
         {"observed_at", utc()},
         {"upstream_revision", nullptr},
         {"media_type", fresh["media_type"]},
         {"body", ""},
         {"completeness", "failed"},
         {"issues",
          J::array({"Synthetic offline acceptance failure; no live request was "
                    "made and no provider outage is asserted."})}});
    check("synthetic failure preserves source2 and has no body",
          failure["source"] == fresh["source"] &&
              failure["completeness"] == "failed" &&
              failure["byte_size"] == 0 && failure["body"] == "");
    auto failed_time = failure["observed_at"].get<std::string>(),
         boundary = stamp(instant(fresh["observed_at"]) +
                          std::chrono::seconds(MAX_AGE)),
         expired = stamp(instant(fresh["observed_at"]) +
                         std::chrono::seconds(MAX_AGE + 1));
    summary["query_times"] = {
        {"historical_expiry_actual_time", failed_time},
        {"fresh_inclusive_boundary_simulation", boundary},
        {"fresh_first_expired_second_simulation", expired}};
    summary["identities"] = {{"source_tuple", source_tuple(fresh)},
                             {"old_source", old["source"]["digest"]},
                             {"fresh_source", fresh["source"]["digest"]},
                             {"old_capture", old["digest"]},
                             {"fresh_capture", fresh["digest"]},
                             {"synthetic_failure", failure["digest"]}};
    J previous_reasons =
          J::array({"latest_attempt_failed", "selected_previous_complete"}),
      snapshots = object();
    for (const auto &[name, capture] : std::vector<std::pair<std::string, J>>{
             {"historical", old}, {"fresh", fresh}}) {
      auto first = imported(name + "-base", capture, nullptr, name),
           second = imported(name + "-failed", failure,
                             first["snapshot_digest"], name);
      check(name + ": completed recovery returns exact original result",
            call(name + "-recover", {"corpus", "recover"},
                 {{"operation_id", name + "-failed"}}, true) == second);
      auto inspected =
          call(name + "-inspect", {"corpus", "inspect"},
               {{"snapshot_digest", second["snapshot_digest"]}}, true);
      check(name + ": retained snapshot reconstructs exactly",
            inspected == second["snapshot"]);
      check(name + ": stable member tuple and previous complete",
            inspected["generation"] == 2 &&
                inspected["parent_digest"] == first["snapshot_digest"] &&
                inspected["members"].size() == 1 &&
                inspected["members"][0]["member_id"] == MEMBER &&
                inspected["members"][0]["last_complete"] ==
                    first["snapshot"]["members"][0]["last_complete"]);
      check(name + ": actual snapshot clock did not rewrite capture age",
            instant(failure["observed_at"]) <=
                    instant(inspected["snapshot_time"]) &&
                instant(inspected["snapshot_time"]) <= instant(utc()));
      auto latest = query(name + "-latest-failed", second["snapshot_digest"],
                          "latest_attempt", failed_time);
      member(name + " latest", latest, failure, "current", true,
             J::array({"latest_attempt_failed"}));
      auto when = name == "historical" ? failed_time : boundary;
      auto selected = query(name + "-last-complete", second["snapshot_digest"],
                            "last_complete", when),
           reasons = previous_reasons;
      if (name == "historical") {
        reasons.push_back("source_revision_differs_from_latest_attempt");
        reasons.push_back("maximum_age_exceeded");
      }
      member(name + " retained", selected, capture,
             name == "historical" ? "expired" : "current", name == "fresh",
             reasons);
      auto exported = call(name + "-export", {"corpus", "export"},
                           {{"snapshot_digest", second["snapshot_digest"]},
                            {"member_ids", J::array({MEMBER})},
                            {"selection", "last_complete"},
                            {"query_time", when},
                            {"max_age_seconds", MAX_AGE}},
                           true);
      check(name + ": exported old evidence was not relabeled",
            exported["query"] == selected &&
                exported["captures"] == J::array({capture}));
      auto difference =
          call(name + "-diff", {"corpus", "diff"},
               {{"before_snapshot_digest", first["snapshot_digest"]},
                {"after_snapshot_digest", second["snapshot_digest"]}},
               true);
      check(name + ": independently expected refresh axes",
            difference["added_member_ids"] == array() &&
                difference["removed_member_ids"] == array() &&
                difference["affected_member_ids"] == J::array({MEMBER}) &&
                difference["changes"] ==
                    J::array({{{"member_id", MEMBER},
                               {"body_changed", true},
                               {"source_changed", name == "historical"},
                               {"observation_changed", true},
                               {"coverage_changed", true},
                               {"last_complete_changed", false}}}));
      snapshots[name] = second;
    }
    auto later = query("fresh-expired", snapshots["fresh"]["snapshot_digest"],
                       "last_complete", expired),
         expired_reasons = previous_reasons;
    expired_reasons.push_back("maximum_age_exceeded");
    member("fresh later", later, fresh, "expired", true, expired_reasons);
    auto selected_graph = graph("fresh-evidence", fresh);
    for (const auto &c : std::vector<std::vector<std::string>>{
             {"inclusive", boundary, "supported"},
             {"expired", expired, "unsupported"}}) {
      auto finding = call("evidence-" + c[0], {"query"},
                          {{"graph", selected_graph},
                           {"query_time", c[1]},
                           {"claim_ids", J::array({CLAIM})}});
      check(c[0] + ": same graph evidence ages without changed caller claim",
            finding["graph_digest"] == selected_graph["digest"] &&
                finding["findings"].size() == 1 &&
                finding["findings"][0]["status"] == c[2] &&
                finding["findings"][0]["claim"]["value"] ==
                    J({{"type", "boolean"},
                       {"unit", nullptr},
                       {"value", false}}));
    }
    auto empty_graph = graph("failed-evidence", failure, false),
         empty = call("failed-evidence-query", {"query"},
                      {{"graph", empty_graph},
                       {"query_time", failed_time},
                       {"claim_ids", J::array({CLAIM})}});
    check("failed current acquisition supplies no positive assertion",
          empty["findings"] == array());
    for (const auto field : {"source_id", "provider_id"}) {
      auto replacement = desired(fresh["source"]);
      replacement[field] =
          replacement[field].get<std::string>() + "-replacement";
      auto rejected =
          call(std::string("reject-") + field, {"source-plan"},
               {{"operation_id", std::string("reject-") + field},
                {"current", fresh["source"]},
                {"desired", replacement},
                {"reason", "Synthetic negative: identity replacement cannot "
                           "reuse an existing source chain."}},
               false, "engine_rejected");
      check(std::string(field) + ": chain identity rejection",
            rejected["error"]["engine_code"] == "scv.identity_change");
    }
    auto relocated = desired(fresh["source"]);
    bool found = false;
    for (auto &locator : relocated["locators"])
      if (locator["locator_id"] == fresh["locator_id"]) {
        locator["locator_id"] =
            locator["locator_id"].get<std::string>() + "-other";
        found = true;
        break;
      }
    NT_REQUIRE(found);
    auto plan =
        call("other-locator-plan", {"source-plan"},
             {{"operation_id", "other-locator"},
              {"current", fresh["source"]},
              {"desired", relocated},
              {"reason", "Synthetic negative: a different locator ID is a "
                         "different corpus tuple, even at the same URI."}});
    auto alternate = capture_fields(failure);
    alternate["source"] = plan["source"];
    alternate["locator_id"] = fresh["locator_id"].get<std::string>() + "-other";
    alternate = call("other-locator-capture", {"capture-import"}, alternate);
    imported("reject-locator-rebinding", alternate,
             snapshots["fresh"]["snapshot_digest"], "fresh", "command_failed");
    auto unchanged = call(
        "post-rejection-inspect", {"corpus", "inspect"},
        {{"snapshot_digest", snapshots["fresh"]["snapshot_digest"]}}, true);
    check("rejected member rebinding preserves the prior immutable snapshot",
          unchanged == snapshots["fresh"]["snapshot"]);
    if (args.has("authority-result")) {
      auto raw = nt::read(args.require("authority-result"));
      nt::write(out / "supplied-authority-result.json", raw);
      auto authority = read_unique(args.require("authority-result"));
      check("supplied protected-state report selects the exact fresh source",
            authority.value("protocol", std::string()) ==
                    "symphony.qxctl.scv-source-result.v1" &&
                authority.value("state_digest", J(nullptr)) ==
                    fresh["source"]["digest"] &&
                authority.value("source_id", J(nullptr)) ==
                    fresh["source"]["source_id"]);
      summary["authority_comparison"] = {
          {"file_sha256", sha(raw)},
          {"state_digest", authority["state_digest"]},
          {"scope", "Compared supplied report only; this offline runner does "
                    "not replay SSIAG authority or query protected state."}};
    } else
      summary["authority_comparison"] = nullptr;
    std::size_t successes = 0, rejections = 0;
    for (const auto &c : commands) {
      successes += c["exit_code"] == 0;
      rejections += !c["expected_error"].is_null();
    }
    summary.update(J{{"status", "pass"},
                     {"completed_at", utc()},
                     {"successes", successes},
                     {"expected_rejections", rejections}});
    save();
  }
};
int main(int argc, char **argv) {
  return nt::test_main([&] {
    nt::Arguments args(argc, argv);
    auto domain = args.get("domain", "scv");
    nt::require(domain == "scv" || domain == "schv",
                "use scv or schv so replacement-provider tests reach the "
                "identity boundary");
    RetentionCampaign campaign(args);
    try {
      campaign.run();
    } catch (const std::exception &e) {
      campaign.summary.update(
          J{{"status", "fail"}, {"completed_at", utc()}, {"error", e.what()}});
      campaign.save();
      throw;
    }
    std::cout << campaign.summary.dump(2) << "\n";
  });
}
