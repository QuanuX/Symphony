#include "interpretation.hpp"
#include "knowledge.hpp"
#include "scv.hpp"
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
const std::string time0 = "2026-09-10T00:00:00Z";
void require(bool condition, const std::string& message) { if (!condition) throw std::runtime_error(message); }
template<class F> void rejects(F&& function) {
    try { function(); } catch (const engine::Error&) { return; }
    throw std::runtime_error("expected bounded owner rejection");
}
engine::Request request(const std::string& operation, Json payload) {
    return {"req-interpret-test", "corr-interpret-test", operation, "symphony-scv", engine::unix_time_ms() + 60000, std::move(payload)};
}
Json call(const std::string& op, Json payload, const std::string& domain = "scv") { return scv::handle_interpretation(request(op, std::move(payload)), domain); }
Json policy(Json age = nullptr, const std::string& partial = "exclude") {
    return {{"policy_id", "fixture-policy"}, {"max_age_seconds", age}, {"partial_capture", partial},
        {"allowed_statement_kinds", Json::array({"documented_fact", "requirement", "recommendation", "observation", "user_assertion", "inference", "hypothesis"})}};
}
Json source(const std::string& id = "docs", const std::string& provider = "cf", const std::string& family = "scev") {
    return scv::sealed(Json{{"protocol", "symphony.scv.source.v1"}, {"source_id", id}, {"provider_id", provider}, {"family_id", family},
        {"publisher", "Fixture publisher"}, {"authority_role", "documentation"}, {"scope", "fixture scope"},
        {"locators", Json::array({Json{{"locator_id", "main"}, {"uri", "https://fixture.invalid/" + id}, {"role", "primary"}, {"format", "markdown"}, {"selector", "v1"}}})},
        {"continuity_evidence", Json::array()}, {"generation", 1}, {"predecessor_digest", nullptr}});
}
Json capture(const std::string& body = "Scope: fixture\nLimit: 10 units\n", const std::string& id = "docs", const std::string& disposition = "complete", const std::string& media = "text/markdown", const std::string& provider = "cf", const std::string& family = "scev") {
    return scv::handle_source(request("capture_import", {{"source", source(id, provider, family)}, {"locator_id", "main"},
        {"resolved_uri", "https://fixture.invalid/" + id}, {"redirects", Json::array()}, {"observed_at", time0}, {"upstream_revision", nullptr},
        {"media_type", media}, {"body", body}, {"completeness", disposition},
        {"issues", disposition == "complete" ? Json::array() : Json::array({"fixture acquisition gap"})}}), "scv");
}
Json value(Json number = 8, const std::string& type = "integer", Json unit = "units") { return {{"type", type}, {"value", number}, {"unit", unit}}; }
Json rule(const std::string& id = "limit", const std::string& kind = "documented_fact", const std::string& type = "integer") {
    return {{"rule_id", "rule-" + id}, {"claim_id", id}, {"subject", "fixture-service"}, {"predicate", "maximum"},
        {"scope", {{"plan", "fixture"}}}, {"statement_kind", kind}, {"dependencies", Json::array()}, {"context", Json::array({"Scope: fixture"})},
        {"extractor", {{"kind", "delimited"}, {"prefix", "Limit: "}, {"suffix", " units"}, {"type", type}, {"unit", "units"}}}};
}
Json profile(Json rules = Json::array({rule()}), const std::string& id = "docs", const std::string& provider = "cf") {
    return scv::sealed(Json{{"protocol", "symphony.scv.interpretation-profile.v1"}, {"profile_id", "profile-" + id}, {"profile_version", "fixture-1"},
        {"provider_id", provider}, {"source_id", id}, {"locator_id", "main"}, {"media_types", Json::array({"text/markdown"})},
        {"authored_by", "Fixture author"}, {"rationale", "Explicit fixture mapping; not publisher endorsement"}, {"rules", rules}});
}
Json binding(const Json& cap, const Json& prof) { return {{"profile_digest", prof.at("digest")}, {"capture_digest", cap.at("digest")}}; }
Json interpret(Json cap = capture(), Json prof = profile(), Json selected = policy(), const std::string& domain = "scv") {
    return call("provider_interpret", {{"captures", Json::array({cap})}, {"profiles", Json::array({prof})}, {"bindings", Json::array({binding(cap, prof)})}, {"selection_policy", selected}}, domain);
}
Json check(const std::string& id = "capacity", const std::string& claim = "limit", Json expected = value(), const std::string& op = "gte", const std::string& importance = "required") {
    return {{"check_id", id}, {"importance", importance}, {"left", {{"claim_id", claim}, {"subject", "fixture-service"}, {"scope", {{"plan", "fixture"}}}}},
        {"operator", op}, {"right", {{"kind", "literal"}, {"value", expected}}}};
}
Json connection(Json checks = Json::array({check()}), const std::string& id = "edge-to-user-node") {
    return {{"connection_id", id}, {"from_subject", "caller-edge"}, {"to_subject", "caller-research-node"}, {"checks", checks}};
}
Json evaluate(Json wrappers = Json::array({interpret()}), Json connections = Json::array({connection()}), Json extras = Json::array(), const std::string& time = time0, const std::string& domain = "scv") {
    return call("connection_evaluate", {{"interpretations", wrappers}, {"additional_knowledge", extras}, {"query_time", time}, {"connections", connections}}, domain);
}
Json reassess(const Json& before, const Json& after) { return call("connection_reassess", {{"before", before}, {"after", after}}); }
const Json& outcome(const Json& evaluation, const std::string& id = "capacity") {
    for (const auto& c : evaluation.at("connections")) for (const auto& item : c.at("checks")) if (item.at("specification").at("check_id") == id) return item;
    throw std::runtime_error("missing check");
}
const Json& impact(const Json& reassessment, const std::string& id = "capacity") {
    for (const auto& c : reassessment.at("checks")) if (c.at("check_id") == id) return c;
    throw std::runtime_error("missing impact");
}
Json extra_claim(const std::string& id = "assumed", const std::string& kind = "user_assertion") {
    return {{"claim_id", id}, {"subject", "fixture-service"}, {"predicate", "maximum"}, {"scope", {{"plan", "fixture"}}},
        {"statement_kind", kind}, {"value", value(10)}, {"evidence", Json::array()}, {"dependencies", Json::array()}};
}
Json knowledge(Json claims, Json selected = policy()) {
    return scv::handle_knowledge(request("knowledge_interpret", {{"captures", Json::array()}, {"claims", claims}, {"selection_policy", selected}, {"interpreter_version", "independent-fixture"}}), "scv");
}

void test_provider_interpretation_producer() {
    const auto result = interpret();
    require(result.at("protocol") == "symphony.scv.provider-interpretation.v1" && result == scv::sealed(result), "sealed interpretation protocol");
    require(result.at("extractions")[0].at("status") == "matched", "numeric rule matched");
    const auto claim = result.at("knowledge").at("claims")[0];
    require(claim.at("value") == value(10) && claim.at("evidence").size() == 2, "exact numeric token and contextual evidence retained");
    require(claim.at("statement_kind") == "documented_fact", "statement kind retained");
    const auto assessed = evaluate(Json::array({result}));
    require(outcome(assessed).at("status") == "satisfied" && outcome(assessed).at("comparison") == true, "documented check matches");
}
void test_profile_preparation_preserves_authored_mapping() {
    const auto expected = profile(); auto draft = expected; draft.erase("digest");
    const auto prepared = call("profile_prepare", {{"profile", draft}});
    require(prepared == expected, "preparation preserves exact author mapping and computes native seal");
    require(interpret(capture(), prepared).at("extractions")[0].at("status") == "matched", "prepared artifact feeds existing interpretation");
    auto wrong = draft; wrong["rules"][0]["statement_kind"] = "publisher_verified";
    rejects([&] { static_cast<void>(call("profile_prepare", {{"profile", wrong}})); });
    wrong = draft; wrong["rules"][0]["scope"] = Json::array();
    rejects([&] { static_cast<void>(call("profile_prepare", {{"profile", wrong}})); });
    rejects([&] { static_cast<void>(call("profile_prepare", {{"profile", expected}})); });
    rejects([&] { static_cast<void>(call("profile_prepare", {{"profile", draft}}, "schv-gcp")); });
    require(call("profile_prepare", {{"profile", draft}}, "scev-cf") == expected, "matching leaf prepares without invented capture");
}
void test_profile_enumeration_is_deterministic() {
    const auto a = capture(), b = capture("Scope: fixture\nLimit: 20 units\n", "other");
    const auto p = profile(), q = profile(Json::array({rule("other-limit")}), "other");
    const auto make = [&](bool reverse) { return call("provider_interpret", {{"captures", reverse ? Json::array({b, a}) : Json::array({a, b})},
        {"profiles", reverse ? Json::array({q, p}) : Json::array({p, q})}, {"bindings", reverse ? Json::array({binding(b, q), binding(a, p)}) : Json::array({binding(a, p), binding(b, q)})}, {"selection_policy", policy()}}); };
    require(make(false) == make(true), "outer enumeration cannot change interpretation");
}
void test_profile_identity_is_unique_within_each_selection() {
    const auto a = capture(), b = capture("Scope: fixture\nLimit: 20 units\n", "other");
    const auto p = profile(); auto q = profile(Json::array({rule("other-limit")}), "other");
    q["profile_id"] = p.at("profile_id"); q = scv::sealed(q);
    rejects([&] { static_cast<void>(call("provider_interpret", {{"captures", Json::array({a, b})},
        {"profiles", Json::array({p, q})}, {"bindings", Json::array({binding(a, p), binding(b, q)})}, {"selection_policy", policy()}})); });
    const auto first = interpret(a, p), second = interpret(b, q);
    rejects([&] { static_cast<void>(evaluate(Json::array({first, second}))); });
    require(outcome(evaluate(Json::array({first}))).at("status") == "satisfied", "profile identity does not restrict independent evaluations");
    require(outcome(evaluate(Json::array({second}), Json::array({connection(Json::array({check("capacity", "other-limit")}))}))).at("status") == "satisfied", "separate explicit version remains usable");
}
void test_context_missing_and_ambiguity_remain_unresolved() {
    for (const auto& body : {"Limit: 10 units\n", "Scope: fixture\nScope: fixture\nLimit: 10 units\n"}) {
        const auto result = interpret(capture(body));
        require(result.at("knowledge").at("claims").empty() && result.at("extractions")[0].at("status") == "unresolved", "context cannot disappear or duplicate silently");
    }
}
void test_value_missing_and_ambiguity_remain_unresolved() {
    for (const auto& body : {"Scope: fixture\nUnavailable\n", "Scope: fixture\nLimit: 10 units\nLimit: 20 units\n"}) {
        const auto result = interpret(capture(body));
        require(result.at("knowledge").at("claims").empty(), "missing/ambiguous value produces no claim");
    }
}
void test_invalid_numeric_tokens_are_not_reinterpreted() {
    for (const auto& token : {"01", "+1", "-0", "1e2", "9007199254740992", " 10"})
        require(interpret(capture(std::string("Scope: fixture\nLimit: ") + token + " units\n")).at("knowledge").at("claims").empty(), "invalid integer remains unresolved");
    const auto decimal_profile = profile(Json::array({rule("limit", "documented_fact", "decimal")}));
    require(interpret(capture("Scope: fixture\nLimit: 0.10 units\n"), decimal_profile).at("knowledge").at("claims").empty(), "noncanonical decimal is not silently rounded");
}
void test_partial_failed_and_media_qualification() {
    const auto partial = capture("Scope: fixture\nLimit: 10 units\n", "docs", "partial");
    require(interpret(partial).at("knowledge").at("claims").empty(), "excluded partial creates no claim");
    require(interpret(partial, profile(), policy(nullptr, "include_qualified")).at("knowledge").at("claims").size() == 1, "explicit partial policy permits qualified extraction");
    require(interpret(capture("Scope: fixture\nLimit: 10 units\n", "docs", "failed")).at("knowledge").at("claims").empty(), "failed capture never produces claims");
    require(interpret(capture("Scope: fixture\nLimit: 10 units\n", "docs", "complete", "text/html")).at("extractions")[0].at("reasons")[0] == "unsupported_media_type", "unselected representation unresolved");
}
void test_malformed_profiles_fail_even_without_a_match() {
    auto p = profile(); p["rules"][0]["statement_kind"] = "verified_truth"; p = scv::sealed(p);
    rejects([&] { static_cast<void>(interpret(capture("no matching content"), p)); });
    p = profile(); p["rules"][0]["extractor"]["script"] = "ignored()"; p = scv::sealed(p);
    rejects([&] { static_cast<void>(interpret(capture(), p)); });
    p = profile(); p["rules"][0]["dependencies"] = Json::array({Json{{"claim_id", "x"}, {"role", "invented"}}}); p = scv::sealed(p);
    rejects([&] { static_cast<void>(interpret(capture("no matching content"), p)); });
}
void test_exact_bindings_and_domain_scope() {
    const auto cap = capture(), p = profile();
    rejects([&] { static_cast<void>(call("provider_interpret", {{"captures", Json::array({cap})}, {"profiles", Json::array({p})}, {"bindings", Json::array()}, {"selection_policy", policy()}})); });
    rejects([&] { static_cast<void>(interpret(cap, profile(Json::array({rule()}), "other"))); });
    rejects([&] { static_cast<void>(interpret(cap, p, policy(), "schv-gcp")); });
    require(outcome(evaluate(Json::array({interpret(cap, p, policy(), "scev-cf")}))).at("status") == "satisfied", "parent consumes exactly replayed provider interpretation");
}
void test_connection_consumer_rejects_resealed_interpretation() {
    auto wrapped = interpret(); wrapped["extractions"][0]["status"] = "unresolved"; wrapped = scv::sealed(wrapped);
    rejects([&] { static_cast<void>(evaluate(Json::array({wrapped}))); });
    wrapped = interpret(); wrapped["knowledge"]["claims"][0]["value"]["value"] = 99;
    wrapped["knowledge"] = scv::sealed(wrapped.at("knowledge")); wrapped = scv::sealed(wrapped);
    rejects([&] { static_cast<void>(evaluate(Json::array({wrapped}))); });
}
void test_decimal_order_is_exact_and_signed() {
    const auto p = profile(Json::array({rule("limit", "documented_fact", "decimal")}));
    const auto assessed = [&](const std::string& lhs, const std::string& rhs, const std::string& op) {
        return outcome(evaluate(Json::array({interpret(capture("Scope: fixture\nLimit: " + lhs + " units\n"), p)}),
            Json::array({connection(Json::array({check("capacity", "limit", value(rhs, "decimal"), op)}))}))).at("comparison");
    };
    require(assessed("1000000000000000000000000000000.0000000000000000000000001", "1000000000000000000000000000000", "gte") == true, "precision beyond machine float retained");
    require(assessed("-10.01", "-10.001", "lte") == true && assessed("-0.1", "0", "gte") == false, "signed decimal magnitude correct");
    require(assessed("0.001", "0.01", "gte") == false, "fraction padding compares exact place values");
}
void test_integer_safe_boundaries() {
    const auto cap = capture("Scope: fixture\nLimit: -9007199254740991 units\n");
    const auto result = evaluate(Json::array({interpret(cap)}), Json::array({connection(Json::array({check("capacity", "limit", value(-9007199254740991LL), "eq")}))}));
    require(outcome(result).at("status") == "satisfied", "minimum interoperable integer exactly compared");
}
void test_literal_and_string_tokens_preserve_native_values() {
    auto r = rule(); r["extractor"] = {{"kind", "literal"}, {"quote", "Supports HTTPS"}, {"value", value("HTTPS", "string", nullptr)}};
    const auto literal = interpret(capture("Scope: fixture\nSupports HTTPS\n"), profile(Json::array({r})));
    require(outcome(evaluate(Json::array({literal}), Json::array({connection(Json::array({check("capacity", "limit", value("HTTPS", "string", nullptr), "eq")}))}))).at("status") == "satisfied", "literal mapping preserved");
    r = rule("limit", "documented_fact", "string"); r["extractor"]["unit"] = nullptr;
    const auto native = interpret(capture("Scope: fixture\nLimit: v2026-09-preview units\n"), profile(Json::array({r})));
    require(native.at("knowledge").at("claims")[0].at("value").at("value") == "v2026-09-preview", "vendor token unchanged");
}
void test_type_unit_and_order_mismatches_are_unresolved() {
    for (const auto& expected : {value("10", "decimal"), value(10, "integer", "bytes")})
        require(outcome(evaluate(Json::array({interpret()}), Json::array({connection(Json::array({check("capacity", "limit", expected)}))}))).at("status") == "unresolved", "no type or unit promotion");
    const auto p = profile(Json::array({rule("limit", "documented_fact", "string")}));
    require(outcome(evaluate(Json::array({interpret(capture(), p)}), Json::array({connection(Json::array({check("capacity", "limit", value("8", "string"))}))}))).at("comparison").is_null(), "string ordering not silently numeric");
}
void test_exact_subject_and_complete_scope() {
    for (const bool subject : {false, true}) {
        auto c = check(); if (subject) c["left"]["subject"] = "different-service"; else c["left"]["scope"] = Json::object();
        require(outcome(evaluate(Json::array({interpret()}), Json::array({connection(Json::array({c}))}))).at("status") == "unresolved", "scope/subject cannot be broadened");
    }
}
void test_claim_pair_comparison_is_explicit() {
    const auto a = interpret(), b = interpret(capture("Scope: fixture\nLimit: 8 units\n", "other"), profile(Json::array({rule("other-limit")}), "other"));
    auto c = check(); c["right"] = {{"kind", "claim"}, {"claim_id", "other-limit"}, {"subject", "fixture-service"}, {"scope", {{"plan", "fixture"}}}};
    // Different predicate prevents the two fixture capacities from being one disputed question.
    auto p = profile(Json::array({rule("other-limit")}), "other"); p["rules"][0]["predicate"] = "required-minimum"; p = scv::sealed(p);
    const auto target = interpret(capture("Scope: fixture\nLimit: 8 units\n", "other"), p);
    const auto result = evaluate(Json::array({a, target}), Json::array({connection(Json::array({c}))}));
    require(outcome(result).at("status") == "satisfied" && outcome(result).at("claim_ids").size() == 2, "both explicitly selected operands retained");
    static_cast<void>(b);
}
void test_recommendation_default_and_assumption_stay_qualified() {
    auto p = profile(Json::array({rule("default", "recommendation")})); p["rules"][0]["predicate"] = "default"; p = scv::sealed(p);
    const auto wrapped = interpret(capture(), p);
    const auto c = check("capacity", "default", value(20));
    const auto result = evaluate(Json::array({wrapped}), Json::array({connection(Json::array({c}))}));
    require(outcome(result).at("status") == "conditional" && outcome(result).at("comparison") == false, "recommendation mismatch remains conditional");
    require(outcome(evaluate(Json::array({wrapped}))).at("status") == "unresolved", "default does not manufacture a missing maximum claim");
    const auto assumed = knowledge(Json::array({extra_claim()}));
    require(outcome(evaluate(Json::array(), Json::array({connection(Json::array({check("capacity", "assumed")}))}), Json::array({assumed}))).at("status") == "conditional", "independent user assumptions remain usable and conditional");
}
void test_optional_checks_do_not_choose_user_topology() {
    const auto yes = check(), optional = check("optional", "limit", value(99), "gte", "optional"), missing = check("missing", "unknown");
    const auto result = evaluate(Json::array({interpret()}), Json::array({connection(Json::array({optional, yes}))}));
    require(result.at("connections")[0].at("status") == "satisfied" && outcome(result, "optional").at("status") == "contradicted", "optional contradiction stays visible without overriding required success");
    require(evaluate(Json::array({interpret()}), Json::array({connection(Json::array({optional}))})).at("connections")[0].at("status") == "unresolved", "no required checks cannot prove connection");
    auto contradiction = optional; contradiction["importance"] = "required";
    require(evaluate(Json::array({interpret()}), Json::array({connection(Json::array({missing, contradiction}))})).at("connections")[0].at("status") == "contradicted", "required aggregation precedence explicit");
}
void test_stale_future_and_disputed_claims_do_not_pass() {
    const auto wrapped = interpret(capture(), profile(), policy(10));
    for (const auto& time : {"2026-09-10T00:00:11Z", "2026-09-09T23:59:59Z"})
        require(outcome(evaluate(Json::array({wrapped}), Json::array({connection()}), Json::array(), time)).at("comparison").is_null(), "ineligible evidence cannot be compared as current");
    const auto other = interpret(capture("Scope: fixture\nLimit: 20 units\n", "other"), profile(Json::array({rule("other-limit")}), "other"));
    require(outcome(evaluate(Json::array({interpret(), other}))).at("status") == "unresolved", "existing typed conflict support rules preserved");
}
void test_source_change_reassesses_only_affected_evidence() {
    auto stable_rule = rule("stable"); stable_rule["predicate"] = "separate-maximum";
    const auto stable = interpret(capture("Scope: fixture\nLimit: 30 units\n", "stable"), profile(Json::array({stable_rule}), "stable"));
    const auto checks = Json::array({connection(Json::array({check(), check("stable-check", "stable")}))});
    const auto before = evaluate(Json::array({interpret(), stable}), checks);
    const auto after = evaluate(Json::array({interpret(capture("Scope: fixture\nLimit: 5 units\n")), stable}), checks);
    const auto changed = reassess(before, after);
    require(changed.at("change_axes").at("captures") == true && changed.at("change_axes").at("profiles") == false, "source refresh distinct from profile revision");
    require(impact(changed).at("after_status") == "contradicted" && impact(changed).at("affected") == true, "changed numeric requirement reassessed");
    require(impact(changed, "stable-check").at("changed") == false && impact(changed, "stable-check").at("affected") == false, "independent unchanged evidence not affected");
}
void test_profile_requirement_policy_and_time_axes() {
    const auto before = evaluate();
    auto p = profile(); p["rationale"] = "Revised mapping rationale"; p = scv::sealed(p);
    const auto profile_change = reassess(before, evaluate(Json::array({interpret(capture(), p)})));
    require(profile_change.at("change_axes").at("profiles") == true && profile_change.at("change_axes").at("captures") == false, "profile-only revision does not relabel capture selection");
    const auto changed_requirement = evaluate(Json::array({interpret()}), Json::array({connection(Json::array({check("capacity", "limit", value(11))}))}));
    require(reassess(before, changed_requirement).at("change_axes").at("requirements") == true, "caller threshold change explicit");
    require(reassess(before, evaluate(Json::array({interpret(capture(), profile(), policy(5))}))).at("change_axes").at("selection_policy") == true, "evidence policy change explicit");
    const auto timed = interpret(capture(), profile(), policy(10));
    const auto time_change = reassess(evaluate(Json::array({timed})), evaluate(Json::array({timed}), Json::array({connection()}), Json::array(), "2026-09-10T00:00:11Z"));
    require(time_change.at("change_axes").at("query_time") == true && time_change.at("change_axes").at("captures") == false && impact(time_change).at("affected") == true, "time-only expiry changes eligibility without evidence refresh");
    require(reassess(before, evaluate(Json::array({interpret()}), Json::array({connection()}), Json::array({knowledge(Json::array({extra_claim("unrelated")}))}))).at("change_axes").at("additional_knowledge") == true, "independent knowledge selection explicit");
}
void test_reassessment_consumer_rejects_resealed_evaluation() {
    const auto before = evaluate(); auto forged = before;
    forged["connections"][0]["checks"][0]["status"] = "contradicted"; forged = scv::sealed(forged);
    rejects([&] { static_cast<void>(reassess(before, forged)); });
    forged = before; forged["graph_evaluation"]["findings"][0]["status"] = "conditional";
    forged["graph_evaluation"] = scv::sealed(forged.at("graph_evaluation")); forged = scv::sealed(forged);
    rejects([&] { static_cast<void>(reassess(before, forged)); });
    const auto same = reassess(before, before);
    require(impact(same).at("changed") == false && impact(same).at("affected") == false, "unchanged exact replay stable");
}
void test_reassessment_union_and_dependency_closure() {
    const auto before = evaluate(), after = evaluate(Json::array({interpret()}), Json::array({connection(Json::array({check("replacement")}))}));
    const auto result = reassess(before, after);
    require(impact(result).at("after_status").is_null() && impact(result, "replacement").at("before_status").is_null(), "union identifies added and removed checks");
    auto premise_rule = rule(); premise_rule["extractor"] = {{"kind", "literal"}, {"quote", "Scope: fixture"}, {"value", value(10)}};
    auto p = profile(Json::array({premise_rule})); const auto old = interpret(capture(), p);
    p["rules"][0]["extractor"]["value"] = value(20); p = scv::sealed(p); const auto corrected = interpret(capture(), p);
    auto child = extra_claim("derived", "inference"); child["predicate"] = "derived-capacity";
    child["dependencies"] = Json::array({Json{{"claim_id", "limit"}, {"role", "requires"}}});
    const auto extra = knowledge(Json::array({child})); const auto c = Json::array({connection(Json::array({check("capacity", "derived")}))});
    const auto impacted = reassess(evaluate(Json::array({old}), c, Json::array({extra})), evaluate(Json::array({corrected}), c, Json::array({extra})));
    require(impact(impacted).at("changed") == false && impact(impacted).at("affected") == true, "dependency changed even when direct conditional check remains identical");
}
void test_failed_unused_capture_still_changes_capture_axis() {
    const auto a = interpret(capture("", "docs", "failed")), b = interpret(capture("new diagnostic", "docs", "failed"));
    const auto change = reassess(evaluate(Json::array({a})), evaluate(Json::array({b})));
    require(change.at("change_axes").at("captures") == true && change.at("change_axes").at("profiles") == false, "failed no-claim evidence still participates in capture axis");
}
void test_bounds_deadline_and_real_request() {
    Json many = Json::array(); for (int i = 0; i < 33; ++i) many.push_back(connection(Json::array(), "connection-" + std::to_string(i)));
    rejects([&] { static_cast<void>(evaluate(Json::array({interpret()}), many)); });
    auto p = profile(); p["rules"].push_back(p.at("rules")[0]); p = scv::sealed(p); rejects([&] { static_cast<void>(interpret(capture(), p)); });
    auto expired = request("provider_interpret", Json::object()); expired.deadline_unix_ms = 0;
    rejects([&] { static_cast<void>(scv::handle_interpretation(expired, "scv")); });
    const auto wrapped = interpret();
    const auto input = evaluate(Json::array({wrapped})).at("input");
    const auto now = engine::unix_time_ms();
    const Json envelope = {{"protocol", engine::process_protocol_v1}, {"request_id", "request-native"}, {"correlation_id", "correlation-native"},
        {"operation", "connection_evaluate"}, {"target_engine", "symphony-scv"}, {"deadline_unix_ms", now + 60000}, {"payload", input}};
    const auto parsed = engine::parse_request(envelope.dump(), "symphony-scv", now);
    const auto result = scv::handle_interpretation(parsed, "scv");
    require(result.at("input") == input && outcome(result).at("specification") == check(), "real protocol retains exact caller input/check specification");
}
}
int main() {
    const std::vector<std::pair<std::string, std::function<void()>>> tests = {
        {"provider_interpretation_producer", test_provider_interpretation_producer},
        {"profile_preparation_preserves_authored_mapping", test_profile_preparation_preserves_authored_mapping},
        {"profile_enumeration_is_deterministic", test_profile_enumeration_is_deterministic},
        {"profile_identity_is_unique_within_each_selection", test_profile_identity_is_unique_within_each_selection},
        {"context_missing_and_ambiguity_remain_unresolved", test_context_missing_and_ambiguity_remain_unresolved},
        {"value_missing_and_ambiguity_remain_unresolved", test_value_missing_and_ambiguity_remain_unresolved},
        {"invalid_numeric_tokens_are_not_reinterpreted", test_invalid_numeric_tokens_are_not_reinterpreted},
        {"partial_failed_and_media_qualification", test_partial_failed_and_media_qualification},
        {"malformed_profiles_fail_even_without_a_match", test_malformed_profiles_fail_even_without_a_match},
        {"exact_bindings_and_domain_scope", test_exact_bindings_and_domain_scope},
        {"connection_consumer_rejects_resealed_interpretation", test_connection_consumer_rejects_resealed_interpretation},
        {"decimal_order_is_exact_and_signed", test_decimal_order_is_exact_and_signed},
        {"integer_safe_boundaries", test_integer_safe_boundaries},
        {"literal_and_string_tokens_preserve_native_values", test_literal_and_string_tokens_preserve_native_values},
        {"type_unit_and_order_mismatches_are_unresolved", test_type_unit_and_order_mismatches_are_unresolved},
        {"exact_subject_and_complete_scope", test_exact_subject_and_complete_scope},
        {"claim_pair_comparison_is_explicit", test_claim_pair_comparison_is_explicit},
        {"recommendation_default_and_assumption_stay_qualified", test_recommendation_default_and_assumption_stay_qualified},
        {"optional_checks_do_not_choose_user_topology", test_optional_checks_do_not_choose_user_topology},
        {"stale_future_and_disputed_claims_do_not_pass", test_stale_future_and_disputed_claims_do_not_pass},
        {"source_change_reassesses_only_affected_evidence", test_source_change_reassesses_only_affected_evidence},
        {"profile_requirement_policy_and_time_axes", test_profile_requirement_policy_and_time_axes},
        {"reassessment_consumer_rejects_resealed_evaluation", test_reassessment_consumer_rejects_resealed_evaluation},
        {"reassessment_union_and_dependency_closure", test_reassessment_union_and_dependency_closure},
        {"failed_unused_capture_still_changes_capture_axis", test_failed_unused_capture_still_changes_capture_axis},
        {"bounds_deadline_and_real_request", test_bounds_deadline_and_real_request},
    };
    try { for (const auto& [name, test] : tests) { test(); std::cout << "PASS " << name << '\n'; } }
    catch (const std::exception& error) { std::cerr << error.what() << '\n'; return 1; }
    return 0;
}
