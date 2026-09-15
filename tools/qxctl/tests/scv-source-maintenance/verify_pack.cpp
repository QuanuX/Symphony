#include "maintenance_support.hpp"
using namespace scv_maintenance;
constexpr const char *AUTHOR =
    "Symphony increment 12 independently authored source expectation";

// The expectation file's byte digest is included in package provenance. Keep
// the original authored member order so the native runner reproduces the same
// sealed package, as well as the same independently authored expected values.
nlohmann::ordered_json ordered_expectations(const J &claims) {
  using Ordered = nlohmann::ordered_json;
  auto result = Ordered::array();
  for (const auto &claim : claims) {
    auto scope = Ordered::object();
    for (const auto *key : {"provider", "service", "operating-system",
                            "storage", "component", "engine"}) {
      if (claim.at("scope").contains(key))
        scope[key] = claim.at("scope").at(key);
    }
    NT_REQUIRE(J(scope) == claim.at("scope"));
    result.push_back(
        Ordered{{"claim_id", claim.at("claim_id")},
                {"subject", claim.at("subject")},
                {"predicate", claim.at("predicate")},
                {"scope", scope},
                {"statement_kind", claim.at("statement_kind")},
                {"dependencies", claim.at("dependencies")},
                {"value", Ordered{{"type", claim.at("value").at("type")},
                                  {"value", claim.at("value").at("value")},
                                  {"unit", claim.at("value").at("unit")}}}});
  }
  NT_REQUIRE(J(result) == claims);
  return result;
}

struct PackRun {
  nt::Arguments args;
  fs::path out, old_package, fresh_capture, qxctl, prefix;
  J commands = array(), originals;
  explicit PackRun(const nt::Arguments &value)
      : args(value), out(fs::absolute(args.require("out"))),
        old_package(fs::canonical(args.require("old-package"))),
        fresh_capture(fs::canonical(args.require("fresh-capture"))),
        qxctl(fs::canonical(args.require("qxctl"))),
        prefix(fs::canonical(args.require("prefix"))) {
    nt::require(!fs::exists(out),
                "--out must name a new immutable evidence directory");
    fs::create_directories(out);
    out = fs::canonical(out);
    originals = snapshot();
    nt::write_json(out / "ORIGINALS_BEFORE.json", originals);
  }
  J snapshot() const {
    J files = array();
    for (const auto &path : std::vector<fs::path>{
             old_package / "pack.json", old_package / "prepare-input.json",
             fresh_capture})
      files.push_back({{"path", path.string()},
                       {"bytes", fs::file_size(path)},
                       {"sha256", nt::digest(nt::read(path))}});
    return files;
  }
  J call(const std::string &name, const std::string &route, const J &input,
         bool expected_error = false) {
    auto inp = out / (name + "-input.json");
    nt::write_json(inp, input);
    nt::require(fs::file_size(inp) < 1048576,
                "Do not expand the process input bound");
    std::vector<std::string> argv = {qxctl.string(), "scv"};
    auto words = split(route);
    argv.insert(argv.end(), words.begin(), words.end());
    argv.insert(argv.end(),
                {"--domain", "schv-do", "--version", VERSION, "--prefix",
                 prefix.string(), "--input", inp.string(), "--json"});
    auto started = std::chrono::steady_clock::now();
    auto r = nt::run(argv, "", 40);
    auto stdout_path = out / (name + "-stdout.json"),
         stderr_path = out / (name + "-stderr.txt");
    nt::write(stdout_path, r.stdout_text);
    nt::write(stderr_path, r.stderr_text);
    commands.push_back({{"name", name},
                        {"argv", argv},
                        {"exit_code", r.returncode},
                        {"elapsed_seconds", elapsed(started)},
                        {"expected_error", expected_error},
                        {"input_file", inp.filename().string()},
                        {"stdout_file", stdout_path.filename().string()},
                        {"stderr_file", stderr_path.filename().string()},
                        {"input_bytes", fs::file_size(inp)},
                        {"stdout_bytes", r.stdout_text.size()},
                        {"input_sha256", nt::digest(nt::read(inp))},
                        {"stdout_sha256", nt::digest(r.stdout_text)},
                        {"stderr_sha256", nt::digest(r.stderr_text)}});
    nt::write_json(out / "COMMANDS.json", commands);
    if (expected_error) {
      nt::require(r.returncode != 0,
                  "Fixture substitution must not be accepted");
      auto value = J::parse(r.stdout_text);
      NT_REQUIRE(value["protocol"] == "symphony.qxctl.error.v1");
      NT_REQUIRE(value["error"]["code"] == "engine_rejected");
      NT_REQUIRE(value["error"]["engine_code"] == "pack.invalid");
      return value;
    }
    nt::require(r.returncode == 0, name + ": " + r.stdout_text.substr(0, 2000) +
                                       r.stderr_text.substr(0, 2000));
    return J::parse(r.stdout_text);
  }
  std::pair<J, J> fixture(const std::string &id, const J &captures,
                          const J &policy, const J &expectations,
                          const std::string &rationale) {
    J bindings = array(), extractions = array(), claims = array();
    NT_REQUIRE(captures.size() == expectations.size());
    for (std::size_t i = 0; i < captures.size(); ++i) {
      const auto &cap = captures[i];
      auto profile_id =
          "coverage-" + cap["source"]["source_id"].get<std::string>();
      bindings.push_back(
          {{"profile_id", profile_id}, {"capture_digest", cap["digest"]}});
      for (const auto &c : expectations[i]) {
        claims.push_back(c);
        extractions.push_back({{"profile_id", profile_id},
                               {"capture_digest", cap["digest"]},
                               {"rule_id", c["claim_id"]},
                               {"claim_id", c["claim_id"]},
                               {"status", "matched"},
                               {"reasons", array()}});
      }
    }
    J data = {{"fixture_id", id},
              {"captures", captures},
              {"bindings", bindings},
              {"selection_policy", policy}},
      manifest = {{"fixture_id", id},
                  {"label", id},
                  {"authored_by", AUTHOR},
                  {"rationale", rationale},
                  {"input_digest", nullptr},
                  {"expected_claims", claims},
                  {"expected_extractions", extractions}};
    return {manifest, data};
  }
  void execute() {
    auto old_prepare = nt::read_json(old_package / "prepare-input.json"),
         old_pack = nt::read_json(old_package / "pack.json");
    J captures = object();
    for (const auto &f : old_prepare["fixtures"])
      for (const auto &cap : f["captures"]) {
        auto id = cap["source"]["source_id"].get<std::string>();
        nt::require(!captures.contains(id) || captures[id] == cap,
                    "Ambiguous old source capture");
        captures[id] = cap;
      }
    auto old_capture = captures.at("do-app-platform-limits"),
         fresh = nt::read_json(fresh_capture),
         labs = captures.at("do-app-platform-designer");
    auto find_fixture = [&](const std::string &id) {
      for (const auto &f : old_prepare["fixtures"])
        if (f["fixture_id"] == id)
          return f;
      throw std::runtime_error("Missing fixture " + id);
    };
    auto all_fixture = find_fixture("do-all-retained-source-semantics"),
         policy = all_fixture["selection_policy"];
    NT_REQUIRE(fresh["source"]["generation"] == 2 &&
               fresh["source"]["predecessor_digest"] ==
                   old_capture["source"]["digest"]);
    NT_REQUIRE(fresh["body"] == old_capture["body"] &&
               fresh["body_digest"] == old_capture["body_digest"]);
    NT_REQUIRE(fresh["completeness"] == "complete" &&
               fresh["byte_size"] == 16774);
    NT_REQUIRE(
        fresh["source"]["locators"][0]["uri"].get<std::string>().ends_with(
            "/limits/index.html.md"));
    nlohmann::ordered_json oracle = {
        {"authored_by", AUTHOR},
        {"method", "Manually authored typed meanings after reading "
                   "exact source bodies; independent reviewer agrees. "
                   "No native/profile values used as an oracle."},
        {"limits_capture_digest", fresh["digest"]},
        {"limits_expected_claims", ordered_expectations(limits_expected())},
        {"labs_capture_digest", labs["digest"]},
        {"labs_expected_claims", ordered_expectations(labs_expected())},
        {"qualifications",
         J::array({"Only App Platform Linux image architecture; not "
                   "all DigitalOcean compute.",
                   "Host-local filesystem is ephemeral; Spaces/managed "
                   "databases are separate persistence options.",
                   "Development PostgreSQL-only is separate from "
                   "managed Valkey availability.",
                   "Historical Labs table says Valkey development DB "
                   "yes; opposing source facts remain separate.",
                   "One-day policy preserved; fixture pass does not "
                   "freshen historical Labs evidence or evaluate a "
                   "graph at current time."})}};
    nt::write(out / "SEMANTIC_EXPECTATIONS.json", oracle.dump(2) + "\n");
    struct FixtureCase {
      std::string name;
      J captures, expected;
      std::string rationale;
    };
    const std::vector<FixtureCase> cases = {
        {"do-all-retained-source-semantics", J::array({labs, fresh}),
         J::array({labs_expected(), limits_expected()}),
         "New limits declaration/capture with unchanged historical Labs "
         "evidence; contradictory development-Valkey facts remain explicit."},
        {"do-product-limits-only", J::array({fresh}),
         J::array({limits_expected()}),
         "Source generation 2 with unchanged three-rule mapping; host-local "
         "persistence, Linux architecture and development/managed database "
         "distinction stay scoped."},
        {"do-labs-guide-only", J::array({labs}), J::array({labs_expected()}),
         "Unchanged exact historical Labs statement; a fixture pass does not "
         "renew its observation or reconcile it with product limits."}};
    J manifests = array(), detached = array();
    for (const auto &c : cases) {
      auto [manifest, data] =
          fixture(c.name, c.captures, policy, c.expected, c.rationale);
      manifests.push_back(manifest);
      detached.push_back(data);
      nt::write_json(out / "package/fixtures" / (c.name + ".json"), data);
    }
    auto draft = old_prepare["pack"];
    draft["pack_version"] = "2026-09-13.1";
    draft["authored_by"] =
        "Symphony increment 12 source-maintenance package author";
    draft["provenance"] = J::array(
        {"Derived separately from historical pack " +
             old_pack["digest"].get<std::string>() + "; originals preserved.",
         "One fresh candidate capture of the already-established official "
         "Markdown destination; body unchanged. Protected source adoption is "
         "separate.",
         "Limits source generation 2; Labs declaration and historical capture "
         "unchanged; no provider/API upgrade.",
         "Exact sealed profile version 1 objects unchanged. Fixture inputs "
         "match their selected source declarations.",
         "Independent expectation file " +
             nt::digest(nt::read(out / "SEMANTIC_EXPECTATIONS.json")),
         "Authored provenance and conformance do not authenticate publishers, "
         "refresh historical evidence or prove runtime compatibility."});
    auto desired = without(fresh["source"], {"protocol", "digest", "generation",
                                             "predecessor_digest"});
    bool found = false;
    for (auto &source : draft["provider"]["sources"])
      if (source["source_id"] == fresh["source"]["source_id"]) {
        source = desired;
        found = true;
      }
    NT_REQUIRE(found);
    draft["fixtures"] = manifests;
    NT_REQUIRE(draft["profiles"] == old_pack["profiles"]);
    J prepare = {{"pack", draft}, {"fixtures", detached}};
    nt::write_json(out / "package/prepare-input.json", prepare);
    auto pack = call("prepare", "provider pack prepare", prepare);
    nt::write_json(out / "package/pack.json", pack);
    auto selection = without(detached[0], {"fixture_id"});
    selection["fixtures"] = detached;
    nt::write_json(out / "package/evaluate-selection.json", selection);
    auto evaluation_input = nt::updated(selection, {{"pack", pack}});
    nt::write_json(out / "package/evaluate-input.json", evaluation_input);
    auto evaluation =
        call("evaluate", "provider pack evaluate", evaluation_input);
    nt::write_json(out / "package/evaluation.json", evaluation);
    NT_REQUIRE((evaluation["conformance"] ==
                J{{"passed", 3}, {"failed", 0}, {"not_run", 0}}));
    auto expected = labs_expected();
    for (const auto &c : limits_expected())
      expected.push_back(c);
    NT_REQUIRE(projected(evaluation["knowledge"]["claims"]) ==
               projected(expected));
    for (const auto &e : evaluation["extractions"])
      NT_REQUIRE(e["status"] == "matched" && e["reasons"] == array());
    auto old_fixture = find_fixture("do-product-limits-only");
    auto substituted =
        nt::updated(without(selection, {"fixtures"}),
                    {{"pack", pack}, {"fixtures", J::array({old_fixture})}});
    call("reject-old-fixture-identity", "provider pack evaluate", substituted,
         true);
    auto negative = draft;
    negative["pack_version"] = "2026-09-13.1-negative-old-source";
    auto [bad_manifest, bad_fixture] =
        fixture("do-old-generation1-fixture-diagnostic",
                J::array({old_capture}), policy, J::array({limits_expected()}),
                "Deliberately mismatched generation-1 desired source: expected "
                "facts remain unchanged; native conformance must fail with "
                "source_declaration_differs.");
    negative["fixtures"] = J::array({bad_manifest});
    auto negative_pack =
        call("diagnostic-prepare", "provider pack prepare",
             {{"pack", negative}, {"fixtures", J::array({bad_fixture})}});
    auto negative_input = nt::updated(
        without(selection, {"fixtures"}),
        {{"pack", negative_pack}, {"fixtures", J::array({bad_fixture})}});
    auto failed =
        call("diagnostic-evaluate", "provider pack evaluate", negative_input);
    NT_REQUIRE((failed["conformance"] ==
                J{{"passed", 0}, {"failed", 1}, {"not_run", 0}}));
    auto result = failed["fixture_results"][0];
    NT_REQUIRE(result["actual_claims"] == array() &&
               result["actual_extractions"].size() == 3);
    for (const auto &e : result["actual_extractions"])
      NT_REQUIRE(e["status"] == "unresolved" &&
                 e["reasons"] == J::array({"source_declaration_differs"}));
    NT_REQUIRE((result["difference_axes"] ==
                J{{"claims", true}, {"extractions", true}}));
    for (const auto &profile : old_pack["profiles"])
      nt::write_json(out / "package/profiles" /
                         (profile["profile_id"].get<std::string>() + ".json"),
                     profile);
    nt::write_json(out / "package/captures/do-app-platform-limits.json", fresh);
    nt::write_json(out / "package/captures/do-app-platform-designer.json",
                   labs);
    nt::require(snapshot() == originals, "Original evidence changed");
    nt::write_json(out / "ORIGINALS_AFTER.json", snapshot());
    J profiles = array();
    for (const auto &p : pack["profiles"])
      profiles.push_back(p["digest"]);
    std::uintmax_t max_input = 0, max_output = 0;
    for (const auto &c : commands) {
      max_input = std::max(max_input, c["input_bytes"].get<std::uintmax_t>());
      max_output =
          std::max(max_output, c["stdout_bytes"].get<std::uintmax_t>());
    }
    J summary = {{"status", "passed"},
                 {"command_count", commands.size()},
                 {"successful_commands", 4},
                 {"expected_command_rejections", 1},
                 {"cli", qxctl.string()},
                 {"cli_sha256", nt::digest(nt::read(qxctl))},
                 {"prefix", prefix.string()},
                 {"domain", "schv-do"},
                 {"engine_version", VERSION},
                 {"pack_id", pack["pack_id"]},
                 {"pack_version", pack["pack_version"]},
                 {"pack_digest", pack["digest"]},
                 {"evaluation_digest", evaluation["digest"]},
                 {"conformance", evaluation["conformance"]},
                 {"production_matched_rules", evaluation["extractions"].size()},
                 {"limits_matched_rules", 3},
                 {"profile_digests", profiles},
                 {"source_generation", 2},
                 {"source_digest", fresh["source"]["digest"]},
                 {"capture_digest", fresh["digest"]},
                 {"body_digest", fresh["body_digest"]},
                 {"body_bytes", fresh["byte_size"]},
                 {"old_fixture_identity_rejected", true},
                 {"diagnostic_conformance", failed["conformance"]},
                 {"diagnostic_generation1_unresolved_rules", 3},
                 {"diagnostic_result_digest", failed["digest"]},
                 {"original_files_preserved", originals.size()},
                 {"network_acquisitions", 0},
                 {"maximum_input_file_bytes", max_input},
                 {"maximum_output_file_bytes", max_output},
                 {"scope", "Mapping/conformance only. No source-head adoption "
                           "or graph freshness/runtime compatibility claim."}};
    nt::write_json(out / "SUMMARY.json", summary);
    std::cout << summary.dump(2) << "\n";
  }
};
int main(int argc, char **argv) {
  return nt::test_main([&] {
    PackRun run(nt::Arguments(argc, argv));
    run.execute();
  });
}
