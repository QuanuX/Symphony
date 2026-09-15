#include "maintenance_support.hpp"
using namespace scv_maintenance;
struct SemanticsRun {
  nt::Arguments args;
  fs::path out;
  J commands = array();
  explicit SemanticsRun(const nt::Arguments &value)
      : args(value), out(fs::absolute(args.require("out"))) {
    NT_REQUIRE(!fs::exists(out));
    private_dir(out);
    out = fs::canonical(out);
  }
  J call(const std::string &name, const std::string &route, const J &value) {
    auto input = out / (name + "-input.json");
    nt::write_json(input, value);
    std::vector<std::string> argv = {
        fs::canonical(args.require("qxctl")).string(), "scv"};
    auto parts = split(route);
    argv.insert(argv.end(), parts.begin(), parts.end());
    argv.insert(argv.end(),
                {"--domain", "scv", "--version", VERSION, "--prefix",
                 fs::canonical(args.require("prefix")).string(), "--input",
                 input.string(), "--json"});
    auto run = nt::run(argv, "", 120, {}, out);
    nt::write(out / (name + "-stdout.json"), run.stdout_text);
    nt::write(out / (name + "-stderr.txt"), run.stderr_text);
    commands.push_back({{"name", name},
                        {"argv", argv},
                        {"exit_code", run.returncode},
                        {"input_file_digest", nt::digest(nt::read(input))},
                        {"stdout_file_digest", nt::digest(run.stdout_text)},
                        {"stderr_file_digest", nt::digest(run.stderr_text)}});
    nt::write_json(out / "COMMANDS.json", commands);
    nt::require(run.returncode == 0, name + ": " +
                                         run.stdout_text.substr(0, 2000) +
                                         run.stderr_text.substr(0, 2000));
    return J::parse(run.stdout_text);
  }
  void run() {
    J captures = {
        {"old", nt::read_json(args.require("old-capture"))},
        {"fresh", nt::read_json(args.require("fresh-capture"))},
        {"insufficient", nt::read_json(args.require("insufficient-capture"))}};
    auto profile = nt::read_json(args.require("profile"));
    NT_REQUIRE(profile["profile_id"] == "coverage-do-app-platform-limits");
    NT_REQUIRE(profile["profile_version"] == "1");
    for (const auto &c : captures)
      NT_REQUIRE(c["completeness"] == "complete");
    NT_REQUIRE(captures["old"]["body"] == captures["fresh"]["body"]);
    NT_REQUIRE(captures["old"]["source"]["digest"] !=
               captures["fresh"]["source"]["digest"]);
    NT_REQUIRE(captures["insufficient"]["source"]["digest"] !=
               captures["fresh"]["source"]["digest"]);
    J policy = {{"policy_id", "maintained-provider-reference-policy-v1"},
                {"max_age_seconds", 86400},
                {"allowed_statement_kinds",
                 J::array(
                     {"documented_fact", "requirement", "recommendation"})},
                {"partial_capture", "exclude"}},
      interpretations = object();
    for (const auto label : {"old", "fresh", "insufficient"}) {
      auto captured = captures[label];
      auto value = call(
          std::string(label) + "-interpret", "provider interpret",
          {{"captures", J::array({captured})},
           {"profiles", J::array({profile})},
           {"bindings", J::array({{{"profile_digest", profile["digest"]},
                                   {"capture_digest", captured["digest"]}}})},
           {"selection_policy", policy}});
      interpretations[label] = value;
      J semantics = object();
      for (const auto &c : value["knowledge"]["claims"])
        semantics[c["claim_id"].get<std::string>()] = pick(
            c, {"claim_id", "subject", "predicate", "scope", "statement_kind",
                "value", "dependencies", "valid_from", "valid_until"});
      if (std::string(label) == "insufficient") {
        NT_REQUIRE(semantics == object());
        NT_REQUIRE(value["extractions"].size() == 3);
        for (const auto &e : value["extractions"])
          NT_REQUIRE(e["status"] == "unresolved");
      } else {
        NT_REQUIRE(semantics == expected_claims());
        for (const auto &e : value["extractions"])
          NT_REQUIRE(e["status"] == "matched");
      }
    }
    auto latest = instant(captures["old"]["observed_at"]);
    for (const auto &c : captures)
      latest = std::max(latest, instant(c["observed_at"]));
    auto query = latest + std::chrono::seconds(1);
    auto query_time = stamp(query);
    NT_REQUIRE(query - instant(captures["old"]["observed_at"]) >
               std::chrono::seconds(86400));
    NT_REQUIRE(query >= instant(captures["fresh"]["observed_at"]));
    NT_REQUIRE(query - instant(captures["fresh"]["observed_at"]) <=
               std::chrono::seconds(86400));
    J scope = {{"provider", "do"},
               {"service", "app-platform"},
               {"storage", "host-local-filesystem"}},
      resolution = {
          {"kind", "caller_decision"},
          {"reference", "example:host-local-storage-requirement"},
          {"description",
           "Caller selects a disposable host-local filesystem for this "
           "hypothetical recipe; this is not a platform preference."}};
    J requirement = {
        {"requirement_id", "host-local-persistence"},
        {"importance", "required"},
        {"operator", "eq"},
        {"right",
         {{"kind", "literal"},
          {"value",
           {{"type", "boolean"}, {"value", false}, {"unit", nullptr}}}}},
        {"resolution", resolution}};
    auto persistent = requirement;
    persistent["right"]["value"]["value"] = true;
    J base = {
        {"interpretations", array()},
        {"additional_knowledge", array()},
        {"provider_packs", array()},
        {"query_time", query_time},
        {"requirements", J::array({requirement})},
        {"slots",
         J::array(
             {{{"slot_id", "caller-selected-app-platform"},
               {"allowed_provider_ids", J::array({"do"})},
               {"recipes",
                J::array(
                    {{{"recipe_id", "hypothetical-host-local-workload"},
                      {"provider_id", "do"},
                      {"bindings",
                       J::array(
                           {{{"requirement_id", "host-local-persistence"},
                             {"claim",
                              {{"claim_id", "do-app-platform-local-storage"},
                               {"subject", "do.app-platform.container"},
                               {"scope", scope}}}}})},
                      {"prerequisites", array()},
                      {"requires_interfaces", array()},
                      {"supplies_interfaces", array()},
                      {"guarantee_changes", array()},
                      {"implementation",
                       {{"status", "unimplemented"}, {"reference", nullptr}}},
                      {"resolution",
                       {{"kind", "adapter"},
                        {"reference",
                         "example:unimplemented-host-local-workload"},
                        {"description",
                         "Authored recipe only; no runtime, account, network, "
                         "deployment or payload verification."}}}}})}}})},
        {"allowed_guarantee_changes", array()},
        {"counterfactuals",
         J::array({{{"counterfactual_id", "caller-requires-persistence"},
                    {"requirements", J::array({persistent})}}})},
        {"bounds", {{"max_candidates", 1}}}};
    J compositions = object(), statuses = object();
    for (const auto label : {"old", "fresh", "insufficient"}) {
      auto result =
          call(std::string(label) + "-explore", "composition explore",
               nt::updated(base, {{"interpretations",
                                   J::array({interpretations[label]})}}));
      compositions[label] = result;
      statuses[label] = object();
      for (const auto &s : result["scenarios"]) {
        auto candidate = s["candidates"][0];
        statuses[label][s["scenario_id"].get<std::string>()] =
            candidate["status"];
        NT_REQUIRE(candidate["implementation_status"] == "unimplemented");
        NT_REQUIRE(std::any_of(
            candidate["obligations"].begin(), candidate["obligations"].end(),
            [](const J &o) { return o["kind"] == "implementation_gap"; }));
      }
      J expected = std::string(label) == "fresh"
                       ? J{{"baseline", "satisfied"},
                           {"caller-requires-persistence", "contradicted"}}
                       : J{{"baseline", "unresolved"},
                           {"caller-requires-persistence", "unresolved"}};
      NT_REQUIRE(statuses[label] == expected);
    }
    J reassessments = object();
    for (const auto &triple : std::vector<std::vector<std::string>>{
             {"refresh", "old", "fresh"},
             {"insufficient", "fresh", "insufficient"}}) {
      auto result = call(triple[0] + "-reassess", "composition reassess",
                         {{"before", compositions[triple[1]]},
                          {"after", compositions[triple[2]]}});
      reassessments[triple[0]] = result;
      NT_REQUIRE((result["change_axes"] == J{{"evidence", true},
                                             {"policy", false},
                                             {"requirements", false},
                                             {"query_time", false},
                                             {"recipes", false},
                                             {"selections", false},
                                             {"bounds", false}}));
    }
    auto expiry_time = stamp(instant(captures["fresh"]["observed_at"]) +
                             std::chrono::seconds(86401));
    auto expired =
        call("expired-explore", "composition explore",
             nt::updated(base, {{"query_time", expiry_time},
                                {"interpretations",
                                 J::array({interpretations["fresh"]})}}));
    for (const auto &s : expired["scenarios"])
      NT_REQUIRE(s["candidates"][0]["status"] == "unresolved");
    auto expiry = call("expiry-reassess", "composition reassess",
                       {{"before", compositions["fresh"]}, {"after", expired}});
    NT_REQUIRE((expiry["change_axes"] == J{{"evidence", false},
                                           {"policy", false},
                                           {"requirements", false},
                                           {"query_time", true},
                                           {"recipes", false},
                                           {"selections", false},
                                           {"bounds", false}}));
    J inputs = object();
    for (const auto key :
         {"old-capture", "fresh-capture", "insufficient-capture", "profile"}) {
      auto path = fs::canonical(args.require(key));
      inputs[path.string()] = nt::digest(nt::read(path));
    }
    J summary = {
        {"format", "local.scv.source-maintenance-semantics.v1"},
        {"passed", true},
        {"native_version", VERSION},
        {"qxctl_file_digest", nt::digest(nt::read(args.require("qxctl")))},
        {"input_files", inputs},
        {"capture_digests", by_digest(captures)},
        {"profile_digest", profile["digest"]},
        {"independent_expected_values", expected_values()},
        {"independent_expected_claims", expected_claims()},
        {"query_time", query_time},
        {"expiry_query_time", expiry_time},
        {"selection_policy", policy},
        {"statuses", statuses},
        {"composition_digests", by_digest(compositions)},
        {"reassessment_digests", by_digest(reassessments)},
        {"expiry_reassessment_digest", expiry["digest"]},
        {"commands", commands.size()},
        {"scope", "Retained-document and hypothetical caller-recipe acceptance "
                  "only; no publisher authentication, protected selection, "
                  "runtime proof or provider-wide exclusion."}};
    nt::write_json(out / "SUMMARY.json", summary);
    std::cout << J{{"passed", true},
                   {"commands", commands.size()},
                   {"statuses", statuses},
                   {"out", out.string()}}
                     .dump(2)
              << "\n";
  }
};
int main(int argc, char **argv) {
  return nt::test_main([&] {
    SemanticsRun run(nt::Arguments(argc, argv));
    run.run();
  });
}
