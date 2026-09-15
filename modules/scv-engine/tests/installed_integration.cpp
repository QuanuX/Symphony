#include "native_test.hpp"
#include <algorithm>
#include <chrono>
#include <iostream>
#include <map>

namespace nt = native_test;
using J = nt::Json;
namespace fs = std::filesystem;

// Native replacement of the independently installed process campaign. Each
// named case retains its original inputs, rejection assertions and
// exact-version admission. Synthetic fixtures never assert provider capability.
struct InstalledProcessTests {
  nt::TempDir temporary{"scv-installed-"};
  fs::path prefix, binary;
  std::string module, engine_id, domain, version, family, provider;
  J receipt;
  int skipped = 0;

  explicit InstalledProcessTests(const nt::Arguments &args)
      : prefix(fs::canonical(temporary.path)), domain(args.require("domain")),
        version(args.require("version")) {
    auto installed = nt::run({"cmake", "--install", args.require("build"),
                              "--prefix", prefix.string()});
    nt::require(installed.returncode == 0,
                installed.stdout_text + installed.stderr_text);
    module = domain + "-engine";
    engine_id = "symphony-" + domain;
    binary = prefix / "libexec/symphony" / module / version / engine_id;
    receipt = nt::read_json(prefix / "share/symphony/receipts" / module /
                            version / "install-receipt.json");
    if (domain.starts_with("schv-")) {
      family = "schv";
      provider = domain.substr(5);
    } else if (domain == "scev" || domain == "scev-cf") {
      family = "scev";
      provider = "cf";
    } else {
      family = "schv";
      provider = "fixture";
    }
  }
  static J object() { return J::object(); }
  static J array() { return J::array(); }
  static std::int64_t now_ms() {
    return std::chrono::duration_cast<std::chrono::milliseconds>(
               std::chrono::system_clock::now().time_since_epoch())
        .count();
  }
  J request(const std::string &operation, const J &payload,
            const J &overrides = object()) const {
    J value = {{"protocol", "symphony.knowledge.engine-process.v1"},
               {"request_id", "fixture-request"},
               {"correlation_id", "fixture-correlation"},
               {"operation", operation},
               {"target_engine", engine_id},
               {"deadline_unix_ms", now_ms() + 5000},
               {"payload", payload}};
    value.update(overrides);
    return value;
  }
  std::pair<nt::ProcessResult, J> invoke(const std::string &operation,
                                         const J &payload,
                                         const J &overrides = object()) {
    auto result =
        nt::run({binary.string()},
                request(operation, payload, overrides).dump(), 6, {}, {}, true);
    auto response = J::parse(result.stdout_text);
    nt::require(std::count(result.stdout_text.begin(), result.stdout_text.end(),
                           '\n') == 1,
                "one stdout line required");
    nt::require(response == nt::seal(response, "response_digest"),
                "response seal mismatch");
    return {result, response};
  }
  J owner(const std::string &operation, const J &payload) {
    auto [p, r] = invoke(operation, payload);
    nt::require(p.returncode == 0, r.dump());
    return r.at("result");
  }
  J desired() const {
    return {{"source_id", "fixture-docs"},
            {"provider_id", provider},
            {"family_id", family},
            {"publisher", "Synthetic publisher"},
            {"authority_role", "user_declared"},
            {"scope", "Synthetic documentation"},
            {"locators", J::array({{{"locator_id", "docs"},
                                    {"uri", "https://example.invalid/docs.md"},
                                    {"role", "preferred"},
                                    {"format", "markdown"},
                                    {"selector", "fixture-a"}}})},
            {"continuity_evidence", J::array({"fixture"})}};
  }
  J initial() {
    return owner("source_plan", {{"operation_id", "onboard"},
                                 {"current", nullptr},
                                 {"desired", desired()},
                                 {"reason", "Fixture onboarding"}})
        .at("source");
  }
  bool since(int minor, const std::string &reason) {
    static const std::map<std::string, int> versions = {
        {"0.1.0-dev", 1}, {"0.2.0-dev", 2},  {"0.3.0-dev", 3}, {"0.4.0-dev", 4},
        {"0.5.0-dev", 5}, {"0.6.0-dev", 6},  {"0.7.0-dev", 7}, {"0.8.0-dev", 8},
        {"0.9.0-dev", 9}, {"0.10.0-dev", 10}};
    if (versions.at(version) >= minor)
      return true;
    ++skipped;
    std::cout << "SKIP " << reason << "\n";
    return false;
  }
  void test_receipt_owned_standalone_process() {
    NT_REQUIRE(receipt.at("component_id") == module);
    NT_REQUIRE(receipt.at("engine_id") == engine_id);
    NT_REQUIRE(receipt.at("version") == version);
    auto [p, response] = invoke("inspect", object());
    NT_REQUIRE(p.returncode == 0);
    const auto &r = response.at("result");
    NT_REQUIRE(response.at("engine_id") == engine_id);
    NT_REQUIRE(response.at("request_id") == "fixture-request");
    NT_REQUIRE(r.at("engine_version") == version);
    NT_REQUIRE(r.at("canonical_apply_enabled") == false);
    const std::map<std::string, std::size_t> counts = {
        {"0.1.0-dev", 13}, {"0.2.0-dev", 17}, {"0.3.0-dev", 20},
        {"0.4.0-dev", 21}, {"0.5.0-dev", 22}, {"0.6.0-dev", 26},
        {"0.7.0-dev", 26}, {"0.8.0-dev", 28}, {"0.9.0-dev", 30},
        {"0.10.0-dev", 30}};
    NT_REQUIRE(r.at("operations").size() == counts.at(version));
  }
  void verify_owned(const std::string &path, const std::string &bytes) {
    auto found =
        std::find_if(receipt.at("files").begin(), receipt.at("files").end(),
                     [&](const J &v) { return v.at("path") == path; });
    NT_REQUIRE(found != receipt.at("files").end());
    NT_REQUIRE(found->at("digest") == nt::digest(bytes));
  }
  void test_profile_preparation_and_packaged_discovery_evidence() {
    if (!since(4, "profile preparation is an exact .4 addition"))
      return;
    J draft = {
        {"protocol", "symphony.scv.interpretation-profile.v1"},
        {"profile_id", "fixture-profile"},
        {"profile_version", "test-1"},
        {"provider_id", provider},
        {"source_id", "fixture-docs"},
        {"locator_id", "docs"},
        {"media_types", J::array({"text/markdown"})},
        {"authored_by", "test fixture"},
        {"rationale",
         "Empty authored mapping tests structure, asserts no provider facts"},
        {"rules", array()}};
    NT_REQUIRE(owner("profile_prepare", {{"profile", draft}}) ==
               nt::seal(draft));
    auto [p, r] = invoke("profile_prepare", {{"profile", nt::seal(draft)}});
    NT_REQUIRE(p.returncode != 0);
    auto path = "share/symphony/schemas/" + module + "/" + version +
                "/schema-catalog.json";
    auto raw = nt::read(prefix / path);
    verify_owned(path, raw);
    auto catalog = J::parse(raw);
    NT_REQUIRE(catalog.at("engine_version") == version);
    NT_REQUIRE(std::any_of(catalog.at("entries").begin(),
                           catalog.at("entries").end(), [](const J &v) {
                             return v.at("protocol") ==
                                    "symphony.scv.profile-prepare-input.v1";
                           }));
  }
  std::pair<J, J> portable_pack() {
    auto wanted = desired();
    wanted["locators"][0]["format"] = "json";
    auto source =
        owner("source_plan",
              {{"operation_id", "pack-source"},
               {"current", nullptr},
               {"desired", wanted},
               {"reason", "Synthetic independently authored package fixture"}})
            .at("source");
    auto captured =
        owner("capture_import",
              {{"source", source},
               {"locator_id", "docs"},
               {"resolved_uri", wanted["locators"][0]["uri"]},
               {"redirects", array()},
               {"observed_at", "2026-09-10T12:00:00Z"},
               {"upstream_revision", nullptr},
               {"media_type", "application/json"},
               {"body", "{\"plan\":\"fixture\",\"default\":3,\"maximum\":10}"},
               {"completeness", "complete"},
               {"issues", array()}});
    J expected = {
        {"claim_id", "maximum"},
        {"subject", "fixture-service"},
        {"predicate", "maximum"},
        {"scope", {{"plan", "fixture"}}},
        {"statement_kind", "documented_fact"},
        {"dependencies", array()},
        {"value", {{"type", "integer"}, {"value", 10}, {"unit", "units"}}}};
    auto rule = expected;
    rule.erase("value");
    rule.update(J{{"rule_id", "max-rule"},
                  {"context", J::array({{{"pointer", "/plan"},
                                         {"value",
                                          {{"type", "string"},
                                           {"value", "fixture"},
                                           {"unit", nullptr}}}}})},
                  {"extractor",
                   {{"kind", "json_pointer"},
                    {"pointer", "/maximum"},
                    {"type", "integer"},
                    {"unit", "units"}}}});
    J profile = {{"protocol", "symphony.scv.structured-profile.v1"},
                 {"profile_id", "max-profile"},
                 {"profile_version", "fixture-1"},
                 {"provider_id", provider},
                 {"source_id", "fixture-docs"},
                 {"locator_id", "docs"},
                 {"media_types", J::array({"application/json"})},
                 {"authored_by", "Mapping fixture author"},
                 {"rationale", "Map maximum separately from default"},
                 {"rules", J::array({rule})}};
    J policy = {{"policy_id", "pack-fixture"},
                {"max_age_seconds", 60},
                {"partial_capture", "exclude"},
                {"allowed_statement_kinds", J::array({"documented_fact"})}};
    J fixture = {
        {"fixture_id", "maximum-not-default"},
        {"captures", J::array({captured})},
        {"bindings", J::array({{{"profile_id", "max-profile"},
                                {"capture_digest", captured["digest"]}}})},
        {"selection_policy", policy}};
    J draft = {
        {"protocol", "symphony.scv.provider-pack.v1"},
        {"pack_id", "installed-fixture"},
        {"pack_version", "fixture-1"},
        {"authored_by", "Independent package author"},
        {"provenance",
         J::array({"Synthetic conformance case, not provider capability"})},
        {"provider",
         {{"provider_id", provider},
          {"family_id", family},
          {"display_name", "Fixture"},
          {"sources", J::array({wanted})}}},
        {"profiles", array()},
        {"structured_profiles", J::array({profile})},
        {"fixtures",
         J::array(
             {{{"fixture_id", fixture["fixture_id"]},
               {"label", "Maximum is 10, default is 3"},
               {"authored_by", "Independent expectation author"},
               {"rationale",
                "Expected value authored before extraction; synthetic example"},
               {"input_digest", nullptr},
               {"expected_claims", J::array({expected})},
               {"expected_extractions",
                J::array({{{"profile_id", "max-profile"},
                           {"capture_digest", captured["digest"]},
                           {"rule_id", "max-rule"},
                           {"claim_id", "maximum"},
                           {"status", "matched"},
                           {"reasons", array()}}})}}})}};
    auto pack = owner("provider_pack_prepare",
                      {{"pack", draft}, {"fixtures", J::array({fixture})}});
    J input = {{"pack", pack},
               {"captures", J::array({captured})},
               {"bindings", fixture["bindings"]},
               {"selection_policy", policy},
               {"fixtures", J::array({fixture})}};
    return {input, owner("provider_pack_evaluate", input)};
  }
  void test_provider_pack_process_conformance_and_interface_receipt() {
    if (!since(6, "portable provider package is an exact .6 addition"))
      return;
    auto [input, result] = portable_pack();
    NT_REQUIRE((result.at("conformance") ==
                J{{"passed", 1}, {"failed", 0}, {"not_run", 0}}));
    NT_REQUIRE(result["knowledge"]["claims"][0]["value"]["value"] == 10);
    input["fixtures"] = array();
    NT_REQUIRE(
        owner("provider_pack_evaluate", input)["conformance"]["not_run"] == 1);
    auto path = "share/symphony/contracts/" + module + "/" + version +
                "/OWNER-INTERFACE.json";
    auto raw = nt::read(prefix / path);
    verify_owned(path, raw);
    const std::map<std::string, std::size_t> counts = {{"0.6.0-dev", 26},
                                                       {"0.7.0-dev", 26},
                                                       {"0.8.0-dev", 28},
                                                       {"0.9.0-dev", 30},
                                                       {"0.10.0-dev", 30}};
    NT_REQUIRE(J::parse(raw).at("operations").size() == counts.at(version));
  }
  void test_composition_process_preserves_caller_choices_and_change_evidence() {
    if (!since(6, "bounded composition is an exact .6 addition"))
      return;
    auto [unused, pack] = portable_pack();
    J how = {{"kind", "adapter"},
             {"reference", "fixture.adapter.v1"},
             {"description", "Verify the caller-selected adapter"}};
    J recipe = {
        {"recipe_id", "first"},
        {"provider_id", provider},
        {"bindings", J::array({{{"requirement_id", "capacity"},
                                {"claim",
                                 {{"claim_id", "maximum"},
                                  {"subject", "fixture-service"},
                                  {"scope", {{"plan", "fixture"}}}}}}})},
        {"prerequisites", array()},
        {"requires_interfaces", array()},
        {"supplies_interfaces", array()},
        {"guarantee_changes", array()},
        {"implementation",
         {{"status", "unimplemented"}, {"reference", nullptr}}},
        {"resolution", how}};
    J requirement = {
        {"requirement_id", "capacity"},
        {"importance", "required"},
        {"operator", "gte"},
        {"right",
         {{"kind", "literal"},
          {"value", {{"type", "integer"}, {"value", 8}, {"unit", "units"}}}}},
        {"resolution", how}};
    J input = {
        {"interpretations", array()},
        {"additional_knowledge", array()},
        {"provider_packs", J::array({pack})},
        {"query_time", "2026-09-10T12:00:01Z"},
        {"requirements", J::array({requirement})},
        {"slots", J::array({{{"slot_id", "compute"},
                             {"allowed_provider_ids", J::array({provider})},
                             {"recipes", J::array({recipe})}}})},
        {"allowed_guarantee_changes", array()},
        {"counterfactuals", array()},
        {"bounds", {{"max_candidates", 2}}}};
    auto before = owner("composition_explore", input);
    auto candidate = before["scenarios"][0]["candidates"][0];
    NT_REQUIRE(candidate["status"] == "satisfied");
    NT_REQUIRE(candidate["implementation_status"] == "unimplemented");
    NT_REQUIRE(!candidate["obligations"].empty());
    input["query_time"] = "2026-09-10T12:02:00Z";
    auto after = owner("composition_explore", input);
    NT_REQUIRE(after["scenarios"][0]["candidates"][0]["status"] ==
               "unresolved");
    auto changed =
        owner("composition_reassess", {{"before", before}, {"after", after}});
    NT_REQUIRE(changed["change_axes"]["query_time"] == true);
    NT_REQUIRE(changed["change_axes"]["evidence"] == false);
    if (version == "0.8.0-dev" || version == "0.9.0-dev" ||
        version == "0.10.0-dev") {
      auto inventory =
          owner("composition_obligations", {{"composition", after}});
      auto submissions = array();
      for (const auto &item : inventory["obligations"])
        submissions.push_back(
            {{"submission_id", item["kind"]},
             {"obligation_id", item["obligation_id"]},
             {"provenance",
              {{"kind", "observation"},
               {"producer", "Synthetic installed-process fixture"},
               {"reference", "fixture:unverified-report"},
               {"content_digest", nullptr},
               {"recorded_at", "2026-09-10T12:02:01Z"},
               {"description",
                "No report content or runtime verification supplied"}}}});
      auto followup = owner(
          "composition_followup",
          {{"before", after}, {"after", before}, {"submissions", submissions}});
      J states = object();
      for (const auto &entry : followup["entries"])
        states[entry["kind"].get<std::string>()] = entry;
      NT_REQUIRE(states["check"]["outcome"] == "criterion_satisfied");
      NT_REQUIRE(states["check"]["causation"] == "not_established");
      NT_REQUIRE(states["implementation"]["outcome"] ==
                 "requires_separate_verification");
      NT_REQUIRE(states["implementation"]["current_status"].is_null());
      NT_REQUIRE(followup["change_axes"]["query_time"] == true);
      NT_REQUIRE(followup["change_axes"]["evidence"] == false);
    }
    after["search"]["eligible_combinations"] = 99;
    after = nt::seal(after);
    auto [p, r] =
        invoke("composition_reassess", {{"before", before}, {"after", after}});
    NT_REQUIRE(p.returncode != 0);
    input["slots"][0]["allowed_provider_ids"] =
        J::array({"unselected-provider"});
    NT_REQUIRE(owner("composition_explore", input)["search"]["stop_reason"] ==
               "no_eligible_recipe");
  }
  void test_source_transition_expected_state_and_digest() {
    auto current = initial(), wanted = desired();
    wanted["locators"][0]["uri"] = "https://example.invalid/moved.md";
    auto plan = owner("source_plan", {{"operation_id", "move"},
                                      {"current", current},
                                      {"desired", wanted},
                                      {"reason", "Fixture continuity"}});
    auto state = owner("source_apply", {{"plan", plan}, {"current", current}})
                     .at("state");
    NT_REQUIRE(state["source_id"] == current["source_id"]);
    NT_REQUIRE(state["predecessor_digest"] == current["digest"]);
    auto [p, r] = invoke("source_apply", {{"plan", plan}, {"current", state}});
    NT_REQUIRE(p.returncode != 0);
    NT_REQUIRE(r["error"]["code"] == "scv.stale_state");
  }
  J capture(const J &source, const std::string &body,
            const std::string &observed,
            const std::string &completeness = "complete") {
    return owner("capture_import",
                 {{"source", source},
                  {"locator_id", "docs"},
                  {"resolved_uri", source["locators"][0]["uri"]},
                  {"redirects", array()},
                  {"observed_at", observed},
                  {"upstream_revision", nullptr},
                  {"media_type", "text/markdown"},
                  {"body", body},
                  {"completeness", completeness},
                  {"issues", completeness == "complete"
                                 ? array()
                                 : J::array({"fixture_fetch_failed"})}});
  }
  void test_capture_provenance_and_graph_process() {
    std::string body =
        "# Fixture\nSee [interface](https://example.invalid/interface).\n";
    auto captured = capture(initial(), body, "2026-09-10T12:00:00Z");
    NT_REQUIRE(captured["body_digest"] == nt::digest(body));
    J policy = {{"policy_id", "fixture-policy"},
                {"max_age_seconds", nullptr},
                {"partial_capture", "exclude"},
                {"allowed_statement_kinds",
                 J::array({"documented_fact", "requirement", "recommendation",
                           "observation", "user_assertion", "inference",
                           "hypothesis"})}};
    auto knowledge =
        owner("knowledge_interpret", {{"captures", J::array({captured})},
                                      {"claims", array()},
                                      {"interpreter_version", "fixture-v1"},
                                      {"selection_policy", policy}});
    auto graph = owner("graph_build", {{"knowledge", J::array({knowledge})}});
    NT_REQUIRE(
        owner("graph_query",
              {{"graph", graph},
               {"query_time", "2026-09-10T12:00:01Z"}})["graph_digest"] ==
        graph["digest"]);
  }
  void test_protocol_rejection_is_machine_readable() {
    for (const auto &override :
         J::array({{{"target_engine", "symphony-foreign"}},
                   {{"deadline_unix_ms", 0}},
                   {{"protocol", "unsupported.v9"}}})) {
      auto [p, r] = invoke("inspect", object(), override);
      NT_REQUIRE(p.returncode != 0);
      NT_REQUIRE(r["outcome"] == "error");
    }
  }
  void test_provider_interpretation_connection_reassessment() {
    auto source = initial();
    auto captured = capture(source, "Fixture service. Port: 7844 UDP.",
                            "2026-09-10T12:00:00Z");
    J scope = {{"service", "fixture-v1"}};
    J profile = nt::seal(
        {{"protocol", "symphony.scv.interpretation-profile.v1"},
         {"profile_id", "fixture-profile"},
         {"profile_version", "1"},
         {"provider_id", provider},
         {"source_id", source["source_id"]},
         {"locator_id", "docs"},
         {"media_types", J::array({"text/markdown"})},
         {"authored_by", "synthetic test"},
         {"rationale", "Local bounded extraction test"},
         {"rules", J::array({{{"rule_id", "port"},
                              {"claim_id", "service-port"},
                              {"subject", "edge"},
                              {"predicate", "requires-egress-port"},
                              {"scope", scope},
                              {"statement_kind", "requirement"},
                              {"dependencies", array()},
                              {"context", J::array({"Fixture service."})},
                              {"extractor",
                               {{"kind", "delimited"},
                                {"prefix", "Port: "},
                                {"suffix", " UDP."},
                                {"type", "integer"},
                                {"unit", "port"}}}}})}});
    J policy = {{"policy_id", "fixture"},
                {"max_age_seconds", 60},
                {"allowed_statement_kinds", J::array({"requirement"})},
                {"partial_capture", "exclude"}};
    auto interpret = [&](const J &cap) {
      J input = {{"captures", J::array({cap})},
                 {"profiles", J::array({profile})},
                 {"bindings", J::array({{{"profile_digest", profile["digest"]},
                                         {"capture_digest", cap["digest"]}}})},
                 {"selection_policy", policy}};
      auto r = owner("provider_interpret", input);
      NT_REQUIRE(r == owner("provider_interpret", input));
      return r;
    };
    auto evaluate = [&](const J &interpreted) {
      return owner(
          "connection_evaluate",
          {{"interpretations", J::array({interpreted})},
           {"additional_knowledge", array()},
           {"query_time", "2026-09-10T12:00:01Z"},
           {"connections",
            J::array({{{"connection_id", "edge-node"},
                       {"from_subject", "edge"},
                       {"to_subject", "node"},
                       {"checks", J::array({{{"check_id", "port"},
                                             {"importance", "required"},
                                             {"left",
                                              {{"claim_id", "service-port"},
                                               {"subject", "edge"},
                                               {"scope", scope}}},
                                             {"operator", "eq"},
                                             {"right",
                                              {{"kind", "literal"},
                                               {"value",
                                                {{"type", "integer"},
                                                 {"value", 7844},
                                                 {"unit", "port"}}}}}}})}}})}});
    };
    auto before = evaluate(interpret(captured));
    NT_REQUIRE(before["connections"][0]["status"] == "satisfied");
    auto after = evaluate(interpret(capture(
        source, "Fixture service. Port: 7845 UDP.", "2026-09-10T12:00:00Z")));
    NT_REQUIRE(after["connections"][0]["status"] == "contradicted");
    auto reassessed =
        owner("connection_reassess", {{"before", before}, {"after", after}});
    NT_REQUIRE(reassessed["checks"][0]["changed"] == true);
    NT_REQUIRE(reassessed["checks"][0]["affected"] == true);
    NT_REQUIRE(reassessed["change_axes"]["captures"] == true);
    auto forged = before;
    forged["connections"][0]["status"] = "contradicted";
    forged = nt::seal(forged);
    auto [p, r] =
        invoke("connection_reassess", {{"before", forged}, {"after", after}});
    NT_REQUIRE(p.returncode != 0);
    NT_REQUIRE(r["outcome"] == "error");
    auto missing =
        interpret(capture(source, "Port: 7844 UDP.", "2026-09-10T12:00:00Z"));
    NT_REQUIRE(missing["extractions"][0]["status"] == "unresolved");
    NT_REQUIRE(evaluate(missing)["connections"][0]["status"] == "unresolved");
  }
  void test_provider_coverage_process_binding_and_declared_gaps() {
    if (version != "0.5.0-dev") {
      ++skipped;
      std::cout << "SKIP provider coverage is an exact .5 addition\n";
      return;
    }
    auto source = initial();
    auto captured =
        capture(source, "Fixture limit: 10 units.", "2026-09-10T12:00:00Z");
    auto profile = owner(
        "profile_prepare",
        {{"profile",
          {{"protocol", "symphony.scv.interpretation-profile.v1"},
           {"profile_id", "coverage-profile"},
           {"profile_version", "fixture-1"},
           {"provider_id", provider},
           {"source_id", source["source_id"]},
           {"locator_id", "docs"},
           {"media_types", J::array({"text/markdown"})},
           {"authored_by", "Synthetic fixture"},
           {"rationale",
            "A bounded literal extraction fixture, not live provider evidence"},
           {"rules", J::array({{{"rule_id", "limit"},
                                {"claim_id", "fixture-limit"},
                                {"subject", "fixture-service"},
                                {"predicate", "maximum"},
                                {"scope", object()},
                                {"statement_kind", "documented_fact"},
                                {"dependencies", array()},
                                {"context", array()},
                                {"extractor",
                                 {{"kind", "delimited"},
                                  {"prefix", "Fixture limit: "},
                                  {"suffix", " units."},
                                  {"type", "integer"},
                                  {"unit", "units"}}}}})}}}});
    auto interpreted = owner(
        "provider_interpret",
        {{"captures", J::array({captured})},
         {"profiles", J::array({profile})},
         {"bindings", J::array({{{"profile_digest", profile["digest"]},
                                 {"capture_digest", captured["digest"]}}})},
         {"selection_policy",
          {{"policy_id", "coverage-policy"},
           {"max_age_seconds", nullptr},
           {"allowed_statement_kinds", J::array({"documented_fact"})},
           {"partial_capture", "exclude"}}}});
    auto missing = desired();
    missing["source_id"] = "not-acquired";
    auto declaration = owner("provider_onboard",
                             {{"provider_id", provider},
                              {"family_id", family},
                              {"display_name", "Fixture provider"},
                              {"sources", J::array({desired(), missing})}});
    auto index = owner("capture_index", {{"capture", captured}});
    auto corpus =
        owner("corpus_build",
              {{"corpus_id", "coverage-corpus"},
               {"previous", nullptr},
               {"snapshot_time", "2026-09-10T12:00:01Z"},
               {"attempts",
                J::array({{{"member_id", "docs"}, {"capture", index}}})}});
    J input = {{"provider", declaration},
               {"corpus_query",
                {{"corpus", corpus},
                 {"member_ids", array()},
                 {"selection", "latest_attempt"},
                 {"query_time", "2026-09-10T12:00:01Z"},
                 {"max_age_seconds", 60}}},
               {"interpretations", J::array({interpreted})}};
    auto result = owner("provider_coverage", input);
    NT_REQUIRE(result["protocol"] == "symphony.scv.provider-coverage.v1");
    NT_REQUIRE(result["domain"] == domain);
    NT_REQUIRE(result["input"] == input);
    NT_REQUIRE(result["summary"]["declared_locators"] == 2);
    NT_REQUIRE(result["summary"]["unselected_declared_locators"] == 1);
    NT_REQUIRE(result["summary"]["matched_rule_attempts"] == 1);
    NT_REQUIRE(result["summary"]["replayed_selected_captures"] == 1);
    auto row = std::find_if(
        result["sources"].begin(), result["sources"].end(),
        [&](const J &v) { return v["source_id"] == source["source_id"]; });
    NT_REQUIRE(row != result["sources"].end());
    NT_REQUIRE((*row)["declaration_match"] == "matches");
    auto metadata = input;
    metadata["interpretations"] = array();
    NT_REQUIRE(owner("provider_coverage",
                     metadata)["summary"]["replayed_selected_captures"] == 0);
    auto forged = interpreted;
    forged["extractions"][0]["status"] = "unresolved";
    input["interpretations"] = J::array({nt::seal(forged)});
    auto [p, r] = invoke("provider_coverage", input);
    NT_REQUIRE(p.returncode != 0);
  }
  void test_corpus_refresh_retains_exact_historical_capture() {
    auto source = initial();
    auto captured =
        capture(source, "# Retained evidence\n", "2026-09-10T12:00:00Z");
    auto first_index = owner("capture_index", {{"capture", captured}});
    auto first = owner("corpus_build",
                       {{"corpus_id", "fixture"},
                        {"previous", nullptr},
                        {"snapshot_time", "2026-09-10T12:00:01Z"},
                        {"attempts", J::array({{{"member_id", "docs"},
                                                {"capture", first_index}}})}});
    auto wanted = desired();
    wanted["locators"][0]["uri"] = "https://example.invalid/moved.md";
    auto moved = owner("source_plan", {{"operation_id", "move"},
                                       {"current", source},
                                       {"desired", wanted},
                                       {"reason", "fixture"}})["source"];
    auto failed = capture(moved, "", "2026-09-10T12:01:00Z", "failed");
    auto failed_index = owner("capture_index", {{"capture", failed}});
    auto refreshed = owner(
        "corpus_build",
        {{"corpus_id", "fixture"},
         {"previous", first},
         {"snapshot_time", "2026-09-10T12:01:01Z"},
         {"attempts",
          J::array({{{"member_id", "docs"}, {"capture", failed_index}}})}});
    auto member = refreshed["members"][0];
    NT_REQUIRE(member["latest_attempt"] == failed_index);
    NT_REQUIRE(member["last_complete"] == first_index);
    auto result = owner("corpus_query",
                        {{"corpus", refreshed},
                         {"member_ids", J::array({"docs"})},
                         {"selection", "last_complete"},
                         {"max_age_seconds", 60},
                         {"query_time", "2026-09-10T13:00:00Z"}})["members"][0];
    NT_REQUIRE(result["selected"]["capture_digest"] == captured["digest"]);
    NT_REQUIRE(result["freshness"] == "expired");
    NT_REQUIRE(result["source_revision_matches_latest"] == false);
    NT_REQUIRE(refreshed["generation"] == 2);
    NT_REQUIRE(refreshed["parent_digest"] == first["digest"]);
  }
  void test_corpus_selection_removal_is_explicit() {
    auto captured = capture(initial(), "# Fixture\n", "2026-09-10T12:00:00Z");
    auto index = owner("capture_index", {{"capture", captured}});
    auto before =
        owner("corpus_build",
              {{"corpus_id", "fixture"},
               {"previous", nullptr},
               {"snapshot_time", "2026-09-10T12:00:01Z"},
               {"attempts",
                J::array({{{"member_id", "docs"}, {"capture", index}}})}});
    auto after =
        owner("corpus_build", {{"corpus_id", "fixture"},
                               {"previous", before},
                               {"snapshot_time", "2026-09-10T12:00:02Z"},
                               {"attempts", array()}});
    auto changed = owner("corpus_diff", {{"before", before}, {"after", after}});
    NT_REQUIRE(changed["removed_member_ids"] == J::array({"docs"}));
    NT_REQUIRE(changed["added_member_ids"] == array());
    NT_REQUIRE(after["coverage"]["requested_members"] == 0);
    auto [p, r] =
        invoke("corpus_query", {{"corpus", after},
                                {"member_ids", J::array({"docs"})},
                                {"selection", "latest_attempt"},
                                {"max_age_seconds", nullptr},
                                {"query_time", "2026-09-10T13:00:00Z"}});
    NT_REQUIRE(p.returncode != 0);
  }
  void test_duplicate_and_unknown_fields_rejected() {
    auto input = request("inspect", object()).dump();
    input.pop_back();
    input += ",\"payload\":{}}";
    auto p = nt::run({binary.string()}, input, 6, {}, {}, true);
    NT_REQUIRE(p.returncode != 0);
    NT_REQUIRE(J::parse(p.stdout_text)["outcome"] == "error");
    auto [rejected, r] =
        invoke("source_status",
               {{"source", nullptr}, {"fabricated_permission", true}});
    NT_REQUIRE(rejected.returncode != 0);
    NT_REQUIRE(r["error"]["code"] == "scv.fields");
  }
};

int main(int argc, char **argv) {
  try {
    nt::Arguments args(argc, argv);
    InstalledProcessTests suite(args);
    const std::pair<const char *, void (InstalledProcessTests::*)()> cases[] = {
#define SCV_CASE(name) {#name, &InstalledProcessTests::name}
        SCV_CASE(test_receipt_owned_standalone_process),
        SCV_CASE(test_profile_preparation_and_packaged_discovery_evidence),
        SCV_CASE(test_provider_pack_process_conformance_and_interface_receipt),
        SCV_CASE(
            test_composition_process_preserves_caller_choices_and_change_evidence),
        SCV_CASE(test_source_transition_expected_state_and_digest),
        SCV_CASE(test_capture_provenance_and_graph_process),
        SCV_CASE(test_protocol_rejection_is_machine_readable),
        SCV_CASE(test_provider_interpretation_connection_reassessment),
        SCV_CASE(test_provider_coverage_process_binding_and_declared_gaps),
        SCV_CASE(test_corpus_refresh_retains_exact_historical_capture),
        SCV_CASE(test_corpus_selection_removal_is_explicit),
        SCV_CASE(test_duplicate_and_unknown_fields_rejected)
#undef SCV_CASE
    };
    for (const auto &[name, call] : cases) {
      (suite.*call)();
      std::cout << name << " complete\n";
    }
    std::cout << J{{"tests", std::size(cases)},
                   {"skipped", suite.skipped},
                   {"status", "passed"}}
                     .dump()
              << "\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << "\n";
    return 1;
  }
}
