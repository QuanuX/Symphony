#include "knowledge.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"

#include <functional>
#include <iostream>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

namespace engine = symphony::knowledge::engine;
namespace scv = symphony::knowledge::scv;
using engine::Json;
namespace {
void require(bool condition, const std::string& message) {
    if (!condition) throw std::runtime_error(message);
}
template<class F> void require_error(F&& function, const std::string& code) {
    try { function(); }
    catch (const engine::Error& error) {
        require(error.code() == code, "expected " + code + ", got " + error.code() + ": " + error.what());
        return;
    }
    throw std::runtime_error("expected error: " + code);
}
engine::Request request(const std::string& operation, Json payload) {
    return {"req-knowledge-test", "corr-knowledge-test", operation, "symphony-scv",
        engine::unix_time_ms() + 60000, std::move(payload)};
}
Json call(const std::string& operation, Json payload, const std::string& domain = "scv") {
    return scv::handle_knowledge(request(operation, std::move(payload)), domain);
}
Json source(const std::string& id = "vendor-docs", const std::string& provider = "cf", const std::string& family = "scev") {
    return scv::sealed(Json{{"protocol", "symphony.scv.source.v1"}, {"source_id", id},
        {"provider_id", provider}, {"family_id", family}, {"publisher", "Fixture publisher"},
        {"authority_role", "documentation"}, {"scope", "fixture product documentation"},
        {"locators", Json::array({Json{{"locator_id", "main"}, {"uri", "https://fixture.invalid/docs"},
            {"role", "primary"}, {"format", "markdown"}, {"selector", ""}}})},
        {"continuity_evidence", Json::array()}, {"generation", 1}, {"predecessor_digest", nullptr}});
}
Json capture(const std::string& body = "# Product\nLimit is 10.\n[Guide](https://fixture.invalid/guide)\n",
             const std::string& id = "vendor-docs", const std::string& observed = "2026-09-10T00:00:00Z",
             const std::string& media = "text/markdown", const std::string& completeness = "complete") {
    return scv::handle_source(request("capture_import", Json{{"source", source(id)}, {"locator_id", "main"},
        {"resolved_uri", "https://fixture.invalid/docs"}, {"redirects", Json::array()}, {"observed_at", observed},
        {"upstream_revision", nullptr}, {"media_type", media}, {"body", body}, {"completeness", completeness},
        {"issues", completeness == "complete" ? Json::array() : Json::array({"fixture interrupted acquisition"})}}), "scv");
}
Json selected_policy(Json age = nullptr) {
    return {{"policy_id", "fixture-policy"}, {"max_age_seconds", age}, {"partial_capture", "exclude"},
        {"allowed_statement_kinds", Json::array({"documented_fact", "requirement", "recommendation", "observation",
            "user_assertion", "inference", "hypothesis"})}};
}
Json claim(const std::string& id, const Json& cap = nullptr, const std::string& quote = "Limit is 10.") {
    return {{"claim_id", id}, {"subject", id}, {"predicate", "maximum"},
        {"value", {{"type", "integer"}, {"value", 10}, {"unit", "requests"}}},
        {"scope", {{"plan", "fixture"}}}, {"statement_kind", "documented_fact"},
        {"evidence", cap.is_null() ? Json::array() : Json::array({Json{{"capture_digest", cap.at("digest")}, {"quote", quote}}})},
        {"dependencies", Json::array()}, {"valid_from", nullptr}, {"valid_until", nullptr}};
}
Json knowledge(Json caps, Json assertions, Json selection = selected_policy(), const std::string& version = "fixture-interpreter-1") {
    return call("knowledge_interpret", {{"captures", std::move(caps)}, {"claims", std::move(assertions)},
        {"interpreter_version", version}, {"selection_policy", std::move(selection)}});
}
Json graph(Json caps, Json assertions, Json selection = selected_policy()) {
    return call("graph_build", {{"knowledge", Json::array({knowledge(std::move(caps), std::move(assertions), std::move(selection))})}});
}
Json evaluation(const Json& value, const std::string& time = "2026-09-10T00:00:00Z") {
    return call("graph_evaluate", {{"graph", value}, {"query_time", time}});
}
const Json& finding(const Json& result, const std::string& id) {
    for (const auto& value : result.at("findings")) if (value.at("claim").at("claim_id") == id) return value;
    throw std::runtime_error("missing finding: " + id);
}
Json dependency(const std::string& id, const std::string& role = "support") { return {{"claim_id", id}, {"role", role}}; }
bool contains_text(const Json& values, const std::string& text) {
    for (const auto& value : values) if (value.is_string() && value.get<std::string>().find(text) != std::string::npos) return true;
    return false;
}

void test_source_knowledge_producer() {
    const auto cap = capture();
    const auto input_claim = claim("limit", cap);
    const auto result = knowledge(Json::array({cap}), Json::array({input_claim}));
    require(result.at("protocol") == "symphony.scv.knowledge.v1", "knowledge protocol");
    require(result.at("captures").at(0) == cap, "capture bytes and authority preserved");
    require(result.at("claims").at(0).at("statement_kind") == "documented_fact", "statement kind preserved");
    require(result.at("native_nodes").size() >= 3, "source, heading and link extracted");
    require(result == scv::sealed(result), "producer self digest");
    const auto assessed = evaluation(call("graph_build", {{"knowledge", Json::array({result})}}));
    require(finding(assessed, "limit").at("status") == "supported", "anchored claim eligible");
    require(finding(assessed, "limit").at("semantic_validation").get<std::string>().find("caller_proposed") != std::string::npos,
        "eligibility does not relabel proposed semantics as truth");
}
void test_graph_consumer_rejects_tampered_capture() {
    const auto cap = capture();
    auto value = graph(Json::array({cap}), Json::array({claim("limit", cap)}));
    value["captures"][0]["body"] = "Tampered body";
    value = scv::sealed(value);
    require_error([&] { static_cast<void>(evaluation(value)); }, "scv.capture_size");
}
void test_deterministic_projection() {
    const auto a = capture(), b = capture("Second source.", "second");
    const auto ca = claim("a", a), cb = claim("b", b, "Second source.");
    require(graph(Json::array({a, b}), Json::array({ca, cb})) == graph(Json::array({b, a}), Json::array({cb, ca})),
        "input enumeration order cannot change semantic digest");
}
void test_false_quote_rejected() {
    const auto cap = capture();
    require_error([&] { static_cast<void>(knowledge(Json::array({cap}), Json::array({claim("bad", cap, "The source never said this.")}))); }, "knowledge.invalid");
}
void test_typed_conflict_and_dependency_propagation() {
    const auto cap = capture();
    auto a = claim("a", cap), b = claim("b", cap), child = claim("child");
    b["subject"] = "a"; b["value"]["value"] = 20;
    child["dependencies"] = Json::array({dependency("a", "requires")});
    const auto result = evaluation(graph(Json::array({cap}), Json::array({a, b, child})));
    require(result.at("conflicts").at(0).at("type") == "incompatible_values", "typed incompatible values");
    require(finding(result, "a").at("status") == "disputed" && finding(result, "b").at("status") == "disputed", "both statements retained");
    require(finding(result, "child").at("status") == "unsupported", "disputed prerequisite propagates");
}
void test_scope_distinctions_do_not_conflict() {
    const auto cap = capture(); auto a = claim("a", cap), b = claim("b", cap);
    b["subject"] = "a"; b["value"]["value"] = 20; b["scope"]["plan"] = "different-plan";
    const auto result = evaluation(graph(Json::array({cap}), Json::array({a, b})));
    require(result.at("conflicts").empty(), "different plan claims are not contradictions");
}
void test_recommendations_are_not_requirements() {
    const auto cap = capture(); auto a = claim("a", cap), b = claim("b", cap);
    a["statement_kind"] = "recommendation"; b["subject"] = "a"; b["value"]["value"] = 20;
    const auto result = evaluation(graph(Json::array({cap}), Json::array({a, b})));
    require(result.at("conflicts").at(0).at("type") == "statement_kind_distinction", "kind distinction survives");
    require(contains_text(finding(result, "a").at("reasons"), "not_a_requirement"), "recommendation qualification");
}
void test_exact_decimal_and_unit_distinctions() {
    const auto cap = capture(); auto a = claim("a", cap), b = claim("b", cap);
    a["value"] = {{"type", "decimal"}, {"value", "0.1"}, {"unit", "USD"}};
    b["subject"] = "a"; b["value"] = {{"type", "decimal"}, {"value", "0.1"}, {"unit", "EUR"}};
    const auto result = evaluation(graph(Json::array({cap}), Json::array({a, b})));
    require(result.at("conflicts").at(0).at("type") == "unit_mismatch", "no implicit currency conversion");
    a["value"]["value"] = "0.10";
    require_error([&] { static_cast<void>(knowledge(Json::array({cap}), Json::array({a}))); }, "knowledge.invalid");
    a["value"] = {{"type", "integer"}, {"value", 1.25}, {"unit", nullptr}};
    require_error([&] { static_cast<void>(knowledge(Json::array({cap}), Json::array({a}))); }, "knowledge.invalid");
    a["value"]["value"] = 18446744073709551615ULL;
    require_error([&] { static_cast<void>(knowledge(Json::array({cap}), Json::array({a}))); }, "knowledge.invalid");
}
void test_freshness_rechecked_without_graph_change() {
    const auto cap = capture(); const auto value = graph(Json::array({cap}), Json::array({claim("a", cap)}), selected_policy(60));
    const auto current = evaluation(value, "2026-09-10T00:01:00Z"), expired = evaluation(value, "2026-09-10T00:01:01Z");
    require(finding(current, "a").at("status") == "supported", "age at inclusive freshness limit admitted");
    require(finding(expired, "a").at("status") == "unsupported", "age beyond limit expires");
    require(current.at("graph_digest") == expired.at("graph_digest") && current.at("digest") != expired.at("digest"), "time is evaluation input independent of graph revision");
}
void test_effective_time_and_future_capture() {
    const auto cap = capture(); auto a = claim("a", cap);
    a["valid_until"] = "2026-09-10T00:01:00Z";
    auto value = graph(Json::array({cap}), Json::array({a}));
    require(finding(evaluation(value, "2026-09-10T00:01:00Z"), "a").at("status") == "stale", "exclusive validity end");
    require(finding(evaluation(value, "2026-09-09T23:59:59Z"), "a").at("status") == "unsupported", "cannot know future capture");
}
void test_alternative_support_survives_expired_premise() {
    const auto cap = capture(); auto a = claim("a", cap), b = claim("b", cap), c = claim("c", cap), x = claim("x");
    b["valid_until"] = "2026-09-10T00:01:00Z";
    x["dependencies"] = Json::array({dependency("a"), dependency("b")});
    x["alternative_supports"] = Json::array({Json{{"support_id", "from-c"}, {"evidence", Json::array()}, {"dependencies", Json::array({dependency("c")})}}});
    const auto result = evaluation(graph(Json::array({cap}), Json::array({a, b, c, x})), "2026-09-10T00:01:01Z");
    const auto& fx = finding(result, "x");
    require(fx.at("status") == "supported", "independent alternative survives");
    require(fx.at("support_sets").at(0).at("status") == "unsupported" && fx.at("support_sets").at(1).at("status") == "supported", "each support set reassessed separately");
}
void test_cycles_cannot_manufacture_support() {
    auto a = claim("a"), b = claim("b");
    a["dependencies"] = Json::array({dependency("b")}); b["dependencies"] = Json::array({dependency("a")});
    const auto result = evaluation(graph(Json::array(), Json::array({a, b})));
    require(finding(result, "a").at("status") == "unsupported" && finding(result, "b").at("status") == "unsupported", "ungrounded cycle stays unsupported");
    require(finding(result, "a").at("evidence_roots").empty() && finding(result, "a").at("support_cycle_present") == true, "cycle visible without roots");
}
void test_grounded_cycle_does_not_inflate_evidence() {
    const auto cap = capture(); auto a = claim("a"), b = claim("b");
    a["dependencies"] = Json::array({dependency("b")}); b["dependencies"] = Json::array({dependency("a")});
    a["alternative_supports"] = Json::array({Json{{"support_id", "ground"}, {"evidence", claim("unused", cap).at("evidence")}, {"dependencies", Json::array()}}});
    const auto result = evaluation(graph(Json::array({cap}), Json::array({a, b})));
    require(finding(result, "b").at("status") == "supported", "cycle can inherit existing grounded path");
    require(finding(result, "a").at("evidence_roots").size() == 1, "cycle adds no new root");
}
void test_mirrors_do_not_count_as_independent_corroboration() {
    const auto a = capture("Identical copied assertion.", "origin"), b = capture("Identical copied assertion.", "mirror");
    auto c = claim("c", a, "Identical copied assertion.");
    c["evidence"].push_back({{"capture_digest", b.at("digest")}, {"quote", "Identical copied assertion."}});
    const auto result = evaluation(graph(Json::array({a, b}), Json::array({c})));
    require(finding(result, "c").at("evidence_roots").size() == 1, "same bytes one evidence root");
    require(result.at("evidence_origins").at(0).at("source_ids").size() == 2, "source identities remain distinct");
    require(result.at("evidence_origins").at(0).at("independence") == "not_asserted", "no independence inferred from source count");
}
void test_scope_addition_invalidates_bounded_absence() {
    const auto original = capture("No selected service listed.");
    const auto newer = capture("New selected service.", "vendor-docs", "2026-09-10T00:01:00Z");
    auto absence = claim("absence", original, "No selected service listed.");
    absence["scope_dependencies"] = Json::array({Json{{"source_id", "vendor-docs"}, {"capture_digests", Json::array({original.at("digest")})}}});
    auto dependent = claim("dependent"); dependent["dependencies"] = Json::array({dependency("absence", "scope")});
    const auto unrelated = claim("unrelated", original, "No selected service listed.");
    const auto before = graph(Json::array({original}), Json::array({absence, dependent, unrelated}));
    const auto after = graph(Json::array({original, newer}), Json::array({absence, dependent, unrelated}));
    const auto result = evaluation(after, "2026-09-10T00:01:00Z");
    require(contains_text(finding(result, "absence").at("reasons"), "scope_capture_selection_changed"), "scope addition invalidates old absence");
    require(finding(result, "dependent").at("status") == "unsupported", "scope reassessment propagates");
    const auto diff = call("graph_diff", {{"before", before}, {"after", after}, {"query_time", "2026-09-10T00:01:00Z"}});
    require(contains_text(diff.at("affected_claim_ids"), "absence") && contains_text(diff.at("affected_claim_ids"), "dependent"), "scope impacts included in diff");
    require(!contains_text(diff.at("affected_claim_ids"), "unrelated"), "unchanged independent claim is outside declared impact");
}
void test_interpreter_correction_preserves_source_revision() {
    const auto cap = capture(); auto a = claim("a", cap), corrected = a;
    corrected["predicate"] = "default";
    const auto before = graph(Json::array({cap}), Json::array({a}));
    const auto revised = knowledge(Json::array({cap}), Json::array({corrected}), selected_policy(), "fixture-interpreter-2");
    const auto after = call("graph_build", {{"knowledge", Json::array({revised})}});
    const auto result = call("graph_diff", {{"before", before}, {"after", after}, {"query_time", "2026-09-10T00:00:00Z"}});
    require(result.at("changes").at("captures").empty(), "interpreter correction does not fabricate upstream revision");
    require(contains_text(result.at("changes").at("changed_claim_ids"), "a"), "changed interpretation identified");
    require(before.at("claims").at(0).at("predicate") == "maximum", "old interpretation remains immutable");
}
void test_partial_capture_policy_is_explicit() {
    const auto cap = capture("Limit is 10.", "partial-doc", "2026-09-10T00:00:00Z", "text/plain", "partial");
    const auto excluded = evaluation(graph(Json::array({cap}), Json::array({claim("a", cap)})));
    auto selected = selected_policy(); selected["partial_capture"] = "include_qualified";
    const auto included = evaluation(graph(Json::array({cap}), Json::array({claim("a", cap)}), selected));
    require(finding(excluded, "a").at("status") == "unsupported", "partial excluded by policy");
    require(finding(included, "a").at("status") == "conditional", "partial included with qualification");
}
void test_hypothesis_and_unverified_inference_remain_conditional() {
    auto a = claim("a"), b = claim("b"); a["statement_kind"] = "hypothesis";
    b["statement_kind"] = "inference"; b["dependencies"] = Json::array({dependency("a")});
    const auto result = evaluation(graph(Json::array(), Json::array({a, b})));
    require(finding(result, "a").at("status") == "conditional" && finding(result, "b").at("status") == "conditional", "assumptions never become verified facts");
    require(finding(result, "b").at("evidence_roots").empty(), "hypothesis has no manufactured evidence");
}
void test_openapi_structure_preserves_native_version_meaning() {
    const auto cap = capture(R"({"openapi":"3.1.0","info":{"title":"Fixture","version":"2024-01-01"},"paths":{"/thing":{"get":{"operationId":"getThing"}}}})", "api", "2026-09-10T00:00:00Z", "application/json");
    const auto result = knowledge(Json::array({cap}), Json::array());
    bool api = false, operation = false;
    for (const auto& node : result.at("native_nodes")) {
        if (node.at("kind") == "openapi_document") {
            api = node.at("attributes").at("description_language_version") == "3.1.0" && node.at("attributes").at("info_version") == "2024-01-01";
        }
        if (node.at("kind") == "api_operation") operation = node.at("pointer") == "/paths/~1thing/get";
    }
    require(api && operation, "native operation and distinct version fields extracted");
    require(result.at("claims").empty(), "structure is not automatic live service fact");
}
void test_unknown_and_duplicate_json_are_retained_as_limited() {
    const auto cap = capture(R"({"openapi":"99.0.0","paths":{}})", "unknown", "2026-09-10T00:00:00Z", "application/json");
    const auto dup = capture(R"({"openapi":"3.1.0","openapi":"3.2.0"})", "duplicate", "2026-09-10T00:00:00Z", "application/json");
    const auto result = knowledge(Json::array({cap, dup}), Json::array());
    require(contains_text(result.at("limitations"), "unsupported JSON dialect"), "unknown dialect explicit");
    require(contains_text(result.at("limitations"), "invalid_or_unsupported_native_json"), "ambiguous duplicate fields not interpreted");
    require(result.at("captures").size() == 2, "original bytes retained");
}
void test_missing_dependency_is_unresolved() {
    auto a = claim("a"); a["dependencies"] = Json::array({dependency("missing")});
    const auto result = evaluation(graph(Json::array(), Json::array({a})));
    require(contains_text(finding(result, "a").at("support_sets").at(0).at("reasons"), "missing_dependency"), "missing premise is not false or true");
}
void test_native_projection_consumer_rejects_fabricated_node() {
    const auto cap = capture(); auto value = graph(Json::array({cap}), Json::array());
    value["native_nodes"][0]["attributes"]["invented"] = "unsupported";
    value = scv::sealed(value);
    require_error([&] { static_cast<void>(evaluation(value)); }, "knowledge.invalid");
}
void test_composition_policy_collision_and_domain_rejection() {
    const auto cap = capture(); const auto one = knowledge(Json::array({cap}), Json::array({claim("a", cap)}));
    const auto other = knowledge(Json::array({cap}), Json::array(), selected_policy(3600));
    require_error([&] { static_cast<void>(call("graph_build", {{"knowledge", Json::array({one, other})}})); }, "knowledge.invalid");
    require_error([&] { static_cast<void>(call("graph_build", {{"knowledge", Json::array({one})}}, "schv")); }, "scv.domain");
    auto conflicting = claim("a", cap); conflicting["value"]["value"] = 11;
    const auto revision = knowledge(Json::array({cap}), Json::array({conflicting}));
    require_error([&] { static_cast<void>(call("graph_build", {{"knowledge", Json::array({one, revision})}})); }, "knowledge.invalid");
}
void test_query_absence_and_dependency_explanation() {
    const auto cap = capture(); auto a = claim("a", cap), b = claim("b"); b["dependencies"] = Json::array({dependency("a")});
    const auto value = graph(Json::array({cap}), Json::array({a, b}));
    const auto query_result = call("graph_query", {{"graph", value}, {"query_time", "2026-09-10T00:00:00Z"}, {"subject", "not-found"}});
    require(query_result.at("findings").empty() && query_result.contains("absence"), "bounded absence qualified");
    const auto explanation = call("graph_explain", {{"graph", value}, {"query_time", "2026-09-10T00:00:00Z"}, {"claim_id", "b"}});
    require(explanation.at("dependency_closure").size() == 2 && explanation.at("findings").size() == 2, "explanation traverses exact premises");
}
void test_limits_unknown_fields_and_deadline() {
    auto a = claim("a"); a["silently_adopt"] = true;
    require_error([&] { static_cast<void>(knowledge(Json::array(), Json::array({a}))); }, "knowledge.invalid");
    Json oversized = Json::array(); for (int i = 0; i < 129; ++i) oversized.push_back(claim("claim-" + std::to_string(i)));
    require_error([&] { static_cast<void>(knowledge(Json::array(), oversized)); }, "knowledge.invalid");
    auto req = request("knowledge_interpret", Json::object()); req.deadline_unix_ms = 0;
    require_error([&] { static_cast<void>(scv::handle_knowledge(req, "scv")); }, "request.deadline_exceeded");
}
} // namespace

int main() {
    const std::vector<std::pair<std::string, std::function<void()>>> tests = {
        {"source knowledge producer", test_source_knowledge_producer},
        {"capture consumer integrity", test_graph_consumer_rejects_tampered_capture},
        {"deterministic projection", test_deterministic_projection},
        {"false quote", test_false_quote_rejected},
        {"typed conflict propagation", test_typed_conflict_and_dependency_propagation},
        {"scope distinction", test_scope_distinctions_do_not_conflict},
        {"recommendation kind", test_recommendations_are_not_requirements},
        {"exact decimal units", test_exact_decimal_and_unit_distinctions},
        {"query-time freshness", test_freshness_rechecked_without_graph_change},
        {"effective and observation time", test_effective_time_and_future_capture},
        {"alternative support", test_alternative_support_survives_expired_premise},
        {"ungrounded cycle", test_cycles_cannot_manufacture_support},
        {"grounded cycle", test_grounded_cycle_does_not_inflate_evidence},
        {"mirror origins", test_mirrors_do_not_count_as_independent_corroboration},
        {"scope addition", test_scope_addition_invalidates_bounded_absence},
        {"interpreter correction", test_interpreter_correction_preserves_source_revision},
        {"partial policy", test_partial_capture_policy_is_explicit},
        {"conditional inference", test_hypothesis_and_unverified_inference_remain_conditional},
        {"OpenAPI structure", test_openapi_structure_preserves_native_version_meaning},
        {"unknown native dialect", test_unknown_and_duplicate_json_are_retained_as_limited},
        {"missing premise", test_missing_dependency_is_unresolved},
        {"native graph integrity", test_native_projection_consumer_rejects_fabricated_node},
        {"composition and domain", test_composition_policy_collision_and_domain_rejection},
        {"query and explanation", test_query_absence_and_dependency_explanation},
        {"bounds and deadline", test_limits_unknown_fields_and_deadline}};
    std::size_t passed = 0;
    for (const auto& [name, function] : tests) {
        try { function(); ++passed; }
        catch (const std::exception& error) { std::cerr << "FAIL " << name << ": " << error.what() << '\n'; return 1; }
    }
    std::cout << passed << " source-knowledge tests passed\n";
    return 0;
}
