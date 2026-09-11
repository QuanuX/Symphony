#include "coverage.hpp"
#include "corpus.hpp"
#include "interpretation.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/error.hpp"

#include <functional>
#include <iostream>
#include <set>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

namespace engine = symphony::knowledge::engine;
namespace scv = symphony::knowledge::scv;
using engine::Json;
namespace {
const std::string time0 = "2026-09-10T00:00:00Z";
const std::string time1 = "2026-09-10T00:01:00Z";
void require(bool condition, const std::string& message) { if (!condition) throw std::runtime_error(message); }
template<class F> void rejects(F&& function) {
    try { function(); } catch (const engine::Error&) { return; }
    throw std::runtime_error("expected bounded provider coverage rejection");
}
engine::Request request(const std::string& operation, Json payload) {
    return {"req-coverage-test", "corr-coverage-test", operation, "symphony-scv", engine::unix_time_ms() + 60000, std::move(payload)};
}
Json desired(const std::string& id = "docs", const std::string& provider = "cf", const std::string& family = "scev") {
    return {{"source_id", id}, {"provider_id", provider}, {"family_id", family}, {"publisher", "Fixture publisher"},
        {"authority_role", "documentation"}, {"scope", "fixture scope"},
        {"locators", Json::array({Json{{"locator_id", "main"}, {"uri", "https://fixture.invalid/" + id}, {"role", "primary"}, {"format", "markdown"}, {"selector", "v1"}}})},
        {"continuity_evidence", Json::array()}};
}
Json provider(Json sources = Json::array({desired()}), const std::string& provider_id = "cf", const std::string& family = "scev") {
    return scv::handle_source(request("provider_onboard", {{"provider_id", provider_id}, {"family_id", family},
        {"display_name", "Fixture provider"}, {"sources", sources}}), "scv");
}
Json capture(Json specification = desired(), const std::string& disposition = "complete", const std::string& time = time0, const std::string& body = "Scope: fixture\nLimit: 10 units\n", const std::string& locator = "main") {
    auto source = specification; source["protocol"] = "symphony.scv.source.v1"; source["generation"] = 1;
    source["predecessor_digest"] = nullptr; source = scv::sealed(source);
    return scv::handle_source(request("capture_import", {{"source", source}, {"locator_id", locator},
        {"resolved_uri", "https://fixture.invalid/" + specification.at("source_id").get<std::string>()}, {"redirects", Json::array()},
        {"observed_at", time}, {"upstream_revision", nullptr}, {"media_type", "text/markdown"}, {"body", body},
        {"completeness", disposition}, {"issues", disposition == "complete" ? Json::array() : Json::array({"fixture acquisition issue"})}}), "scv");
}
Json index(const Json& cap, const std::string& domain = "scv") { return scv::handle_corpus(request("capture_index", {{"capture", cap}}), domain); }
Json corpus(Json captures = Json::array({capture()}), Json previous = nullptr, const std::string& time = time0, const std::string& domain = "scv") {
    Json attempts = Json::array();
    for (const auto& cap : captures) attempts.push_back({{"member_id", cap.at("source").at("source_id").get<std::string>() + ":" + cap.at("locator_id").get<std::string>()}, {"capture", index(cap, domain)}});
    return scv::handle_corpus(request("corpus_build", {{"corpus_id", "fixture-corpus"}, {"previous", previous}, {"snapshot_time", time}, {"attempts", attempts}}), domain);
}
Json policy(const std::string& partial = "exclude", const std::string& id = "fixture-policy") {
    return {{"policy_id", id}, {"max_age_seconds", nullptr}, {"partial_capture", partial},
        {"allowed_statement_kinds", Json::array({"documented_fact", "requirement", "recommendation", "observation", "user_assertion", "inference", "hypothesis"})}};
}
Json interpret(Json cap = capture(), const std::string& partial = "exclude", const std::string& policy_id = "fixture-policy", bool empty_rules = false, const std::string& domain = "scv") {
    const auto& source = cap.at("source");
    Json rules = Json::array();
    if (!empty_rules) rules.push_back({{"rule_id", "maximum"}, {"claim_id", "maximum"}, {"subject", "fixture-service"}, {"predicate", "maximum"},
        {"scope", {{"plan", "fixture"}}}, {"statement_kind", "documented_fact"}, {"dependencies", Json::array()}, {"context", Json::array({"Scope: fixture"})},
        {"extractor", {{"kind", "delimited"}, {"prefix", "Limit: "}, {"suffix", " units"}, {"type", "integer"}, {"unit", "units"}}}});
    const auto profile = scv::sealed(Json{{"protocol", "symphony.scv.interpretation-profile.v1"}, {"profile_id", "fixture-profile"}, {"profile_version", "fixture-1"},
        {"provider_id", source.at("provider_id")}, {"source_id", source.at("source_id")}, {"locator_id", cap.at("locator_id")}, {"media_types", Json::array({"text/markdown"})},
        {"authored_by", "Fixture author"}, {"rationale", "Explicit fixture mapping; not publisher endorsement"}, {"rules", rules}});
    return scv::handle_interpretation(request("provider_interpret", {{"captures", Json::array({cap})}, {"profiles", Json::array({profile})},
        {"bindings", Json::array({Json{{"profile_digest", profile.at("digest")}, {"capture_digest", cap.at("digest")}}})}, {"selection_policy", policy(partial, policy_id)}}), domain);
}
Json input(Json chosen = corpus(), Json wrappers = Json::array({interpret()}), Json declaration = provider(), Json members = Json::array(),
    const std::string& selection = "latest_attempt", const std::string& time = time0, Json age = nullptr) {
    return {{"provider", declaration}, {"corpus_query", {{"corpus", chosen}, {"member_ids", members}, {"selection", selection},
        {"query_time", time}, {"max_age_seconds", age}}}, {"interpretations", wrappers}};
}
Json coverage(Json selected = input(), const std::string& domain = "scv") { return scv::handle_coverage(request("provider_coverage", selected), domain); }
const Json& row(const Json& result, const std::string& source = "docs", const std::string& locator = "main") {
    for (const auto& value : result.at("sources")) if (value.at("source_id") == source && value.at("locator_id") == locator) return value;
    throw std::runtime_error("missing declared source row");
}

void test_provider_coverage_producer() {
    const auto selected = input(); const auto result = coverage(selected);
    require(result.at("protocol") == "symphony.scv.provider-coverage.v1" && scv::sealed(result) == result, "sealed result identity");
    require(result.at("input") == selected, "exact caller input retained");
    require(row(result).at("declaration_match") == "matches", "selected bytes establish exact declaration comparison");
    require(result.at("summary").at("matched_rule_attempts") == 1 && result.at("summary").at("replayed_selected_captures") == 1, "finite extraction and byte counts");
    require(result.at("corpus_query_result") == scv::handle_corpus(request("corpus_query", selected.at("corpus_query")), "scv"), "native query exactly reused");
}
void test_empty_corpus_preserves_declared_inventory() {
    const auto result = coverage(input(corpus(Json::array()), Json::array(), provider(Json::array({desired(), desired("second")}))));
    require(result.at("sources").size() == 2 && result.at("summary").at("unselected_declared_locators") == 2, "empty selection cannot erase declared coverage denominator");
    require(row(result).at("member_id").is_null() && row(result).at("selection_status") == "not_selected", "unacquired declaration is explicit");
}
void test_metadata_does_not_claim_body_or_interpretation() {
    const auto result = coverage(input(corpus(), Json::array()));
    require(row(result).at("selection_status") == "selected" && row(result).at("declaration_match") == "not_available", "metadata index does not establish declaration bytes");
    require(result.at("summary").at("replayed_selected_captures") == 0 && result.at("summary").at("selected_profile_bindings") == 0, "no fake body or semantic proof");
}
void test_explicit_subset_preserves_unselected_member_identity() {
    const auto a = capture(), b = capture(desired("second"));
    const auto result = coverage(input(corpus(Json::array({a, b})), Json::array({interpret(b)}), provider(Json::array({desired(), desired("second")})), Json::array({"docs:main"})));
    require(row(result, "second").at("member_id") == "second:main" && row(result, "second").at("selection_status") == "not_selected", "existing but unselected member remains distinct");
    require(result.at("unselected_bindings").at(0).at("reason") == "capture_not_selected" && result.at("summary").at("matched_rule_attempts") == 0, "unselected wrapper cannot credit selected coverage");
}
void test_partial_policies_remain_independent() {
    const auto cap = capture(desired(), "partial");
    const auto a = interpret(cap, "exclude", "exclude-policy"), b = interpret(cap, "include_qualified", "include-policy");
    const auto result = coverage(input(corpus(Json::array({cap})), Json::array({a, b})));
    require(result.at("summary").at("selected_profile_bindings") == 2 && result.at("summary").at("replayed_selected_captures") == 1, "two mappings do not duplicate captured source evidence");
    require(result.at("summary").at("matched_rule_attempts") == 1 && result.at("summary").at("unresolved_rule_attempts") == 1, "independent policy attempts retained without composition");
    require(result.at("summary").at("selected_declared_status").at("partial") == 1, "matched mapping does not make partial capture complete");
}
void test_failed_refresh_preserves_old_age_and_revision() {
    const auto good = capture(); const auto previous = corpus();
    auto changed = desired(); changed["publisher"] = "Changed fixture publisher";
    const auto failed = capture(changed, "failed", time1, "");
    const auto current = corpus(Json::array({failed}), previous, time1);
    const auto result = coverage(input(current, Json::array({interpret(good)}), provider(Json::array({changed})), Json::array(), "last_complete", time1, 30));
    require(result.at("summary").at("selected_declared_freshness").at("expired") == 1, "failed refresh cannot update prior complete age");
    require(result.at("corpus_query_result").at("members").at(0).at("source_revision_matches_latest") == false, "retained revision remains distinct");
    require(row(result).at("declaration_match") == "differs" && row(result).at("declaration_difference_fields") == Json::array({"publisher"}), "exact old source compared to supplied new declaration");
}
void test_failed_initial_attempt_has_no_last_complete() {
    const auto cap = capture(desired(), "failed", time0, "");
    const auto result = coverage(input(corpus(Json::array({cap})), Json::array({interpret(cap)}), provider(), Json::array(), "last_complete"));
    require(row(result).at("selection_status") == "unavailable" && result.at("summary").at("selected_declared_members") == 1, "selected member differs from available capture");
    require(result.at("summary").at("selected_declared_status").at("unavailable") == 1 && result.at("unselected_bindings").size() == 1, "failed latest wrapper cannot stand in for absent complete evidence");
}
void test_unlisted_members_and_bindings_stay_visible() {
    auto extra_locator = desired(); extra_locator["locators"].push_back({{"locator_id", "other"}, {"uri", "https://fixture.invalid/other"}, {"role", "mirror"}, {"format", "markdown"}, {"selector", "v1"}});
    const auto other_locator = capture(extra_locator, "complete", time0, "Scope: fixture\nLimit: 10 units\n", "other");
    const auto other_source = capture(desired("other-source"));
    const auto other_provider = capture(desired("gcp-docs", "gcp", "schv"));
    const auto result = coverage(input(corpus(Json::array({capture(), other_locator, other_source, other_provider})), Json::array({interpret(other_source)})));
    require(result.at("unlisted_members").size() == 3 && result.at("summary").at("selected_unlisted_members") == 3, "all unlisted selection remains inspectable");
    std::set<std::string> reasons;
    for (const auto& member : result.at("unlisted_members")) reasons.insert(member.at("reason").get<std::string>());
    require(reasons == std::set<std::string>{"provider_not_declared", "source_not_declared", "locator_not_declared"}, "unlisted identity causes distinguished");
    require(result.at("unselected_bindings").at(0).at("reason") == "selected_member_not_declared" && result.at("summary").at("matched_rule_attempts") == 0, "unlisted wrapper not promoted into declaration coverage");
}
void test_different_capture_revision_is_not_credited() {
    const auto old = capture(); const auto fresh = capture(desired(), "complete", time1, "Scope: fixture\nLimit: 20 units\n");
    const auto result = coverage(input(corpus(Json::array({fresh}), nullptr, time1), Json::array({interpret(old)}), provider(), Json::array(), "latest_attempt", time1));
    require(row(result).at("declaration_match") == "not_available" && result.at("unselected_bindings").at(0).at("reason") == "capture_not_selected", "old wrapper cannot validate newer selected body");
}
void test_provider_coverage_rejects_forged_interpretation() {
    auto wrapper = interpret(); wrapper["extractions"][0]["status"] = "unresolved"; wrapper = scv::sealed(wrapper);
    rejects([&] { static_cast<void>(coverage(input(corpus(), Json::array({wrapper})))); });
    wrapper = interpret(); wrapper["profiles"] = Json::array(); wrapper["bindings"] = Json::array(); wrapper = scv::sealed(wrapper);
    rejects([&] { static_cast<void>(coverage(input(corpus(), Json::array({wrapper})))); });
}
void test_borrowed_capture_digest_cannot_forge_index_metadata() {
    auto selected = corpus();
    auto forged = selected["members"][0]["latest_attempt"]; forged["byte_size"] = 1; forged = scv::sealed(forged);
    selected["members"][0]["latest_attempt"] = forged; selected["members"][0]["last_complete"] = forged; selected = scv::sealed(selected);
    rejects([&] { static_cast<void>(coverage(input(selected))); });
    const auto metadata_only = coverage(input(selected, Json::array()));
    require(row(metadata_only).at("declaration_match") == "not_available", "standalone metadata does not assert absent-body correspondence");
}
void test_duplicate_wrapper_rejected_without_global_profile_restriction() {
    const auto wrapper = interpret();
    rejects([&] { static_cast<void>(coverage(input(corpus(), Json::array({wrapper, wrapper})))); });
    const auto different = interpret(capture(), "exclude", "another-policy");
    const auto result = coverage(input(corpus(), Json::array({wrapper, different})));
    require(result.at("summary").at("matched_rule_attempts") == 2 && result.at("summary").at("replayed_selected_captures") == 1, "distinct contextual attempts are permitted and honestly counted");
}
void test_zero_rule_profile_is_not_interpreted_completeness() {
    const auto result = coverage(input(corpus(), Json::array({interpret(capture(), "exclude", "fixture-policy", true)})));
    require(result.at("summary").at("selected_profile_bindings") == 1 && result.at("summary").at("matched_rule_attempts") == 0 && result.at("summary").at("unresolved_rule_attempts") == 0, "zero rules provide no semantic coverage denominator");
    require(result.at("summary").at("replayed_selected_captures") == 1 && row(result).at("declaration_match") == "matches", "zero rules still replay selected bytes and declaration");
}
void test_parent_scope_preserves_child_corpus_and_rejects_other_domains() {
    const auto cap = capture(); const auto child = corpus(Json::array({cap}), nullptr, time0, "scev-cf");
    const auto selected = input(child, Json::array({interpret(cap, "exclude", "fixture-policy", false, "scev-cf")}));
    const auto result = coverage(selected);
    require(result.at("domain") == "scv" && result.at("corpus_query_result").at("domain") == "scev-cf", "parent query retains exact child identity");
    require(coverage(selected, "scev").at("domain") == "scev", "family accepts declared child");
    require(coverage(selected, "scev-cf").at("domain") == "scev-cf", "leaf accepts own corpus");
    rejects([&] { static_cast<void>(coverage(input(), "scev-cf")); });
    rejects([&] { static_cast<void>(coverage(selected, "schv")); });
    auto forged = child; forged["domain"] = "schv-gcp"; forged = scv::sealed(forged);
    rejects([&] { static_cast<void>(coverage(input(forged))); });
}
void test_time_only_query_changes_freshness_without_rewriting_evidence() {
    const auto current = coverage(input(corpus(), Json::array({interpret()}), provider(), Json::array(), "latest_attempt", time0, 30));
    const auto expired = coverage(input(corpus(), Json::array({interpret()}), provider(), Json::array(), "latest_attempt", time1, 30));
    require(current.at("summary").at("selected_declared_freshness").at("current") == 1 && expired.at("summary").at("selected_declared_freshness").at("expired") == 1, "explicit query time controls freshness");
    require(row(current).at("interpretations") == row(expired).at("interpretations"), "extraction attempt not silently replaced by freshness result");
}
void test_strict_bounds_and_request_shape() {
    auto selected = input(); selected["unused_policy"] = nullptr;
    rejects([&] { static_cast<void>(coverage(selected)); });
    selected = input(); selected["corpus_query"] = nullptr;
    rejects([&] { static_cast<void>(coverage(selected)); });
    selected = input(); selected["provider"]["disposition"] = "adopted"; selected["provider"] = scv::sealed(selected["provider"]);
    rejects([&] { static_cast<void>(coverage(selected)); });
    selected = input(); selected["interpretations"] = Json::array();
    for (int i = 0; i < 17; ++i) selected["interpretations"].push_back(interpret());
    rejects([&] { static_cast<void>(coverage(selected)); });
    auto expired = request("provider_coverage", input()); expired.deadline_unix_ms = 1;
    rejects([&] { static_cast<void>(scv::handle_coverage(expired, "scv")); });
}
void test_derived_ordering_is_stable_and_exact_input_order_retained() {
    const auto a = capture(), b = capture(desired("second"));
    const auto one = interpret(a), two = interpret(b);
    const auto selected_corpus = corpus(Json::array({a, b}));
    const auto first = coverage(input(selected_corpus, Json::array({two, one}), provider(Json::array({desired("second"), desired()}))));
    const auto second = coverage(input(selected_corpus, Json::array({one, two}), provider(Json::array({desired(), desired("second")}))));
    require(first.at("sources") == second.at("sources") && first.at("summary") == second.at("summary"), "derived rows do not depend on enumeration order");
    require(first.at("input") != second.at("input") && first.at("digest") != second.at("digest"), "caller input enumeration remains exactly bound");
}
}
int main() {
    const std::vector<std::pair<std::string, std::function<void()>>> tests = {
        {"test_provider_coverage_producer", test_provider_coverage_producer},
        {"test_empty_corpus_preserves_declared_inventory", test_empty_corpus_preserves_declared_inventory},
        {"test_metadata_does_not_claim_body_or_interpretation", test_metadata_does_not_claim_body_or_interpretation},
        {"test_explicit_subset_preserves_unselected_member_identity", test_explicit_subset_preserves_unselected_member_identity},
        {"test_partial_policies_remain_independent", test_partial_policies_remain_independent},
        {"test_failed_refresh_preserves_old_age_and_revision", test_failed_refresh_preserves_old_age_and_revision},
        {"test_failed_initial_attempt_has_no_last_complete", test_failed_initial_attempt_has_no_last_complete},
        {"test_unlisted_members_and_bindings_stay_visible", test_unlisted_members_and_bindings_stay_visible},
        {"test_different_capture_revision_is_not_credited", test_different_capture_revision_is_not_credited},
        {"test_provider_coverage_rejects_forged_interpretation", test_provider_coverage_rejects_forged_interpretation},
        {"test_borrowed_capture_digest_cannot_forge_index_metadata", test_borrowed_capture_digest_cannot_forge_index_metadata},
        {"test_duplicate_wrapper_rejected_without_global_profile_restriction", test_duplicate_wrapper_rejected_without_global_profile_restriction},
        {"test_zero_rule_profile_is_not_interpreted_completeness", test_zero_rule_profile_is_not_interpreted_completeness},
        {"test_parent_scope_preserves_child_corpus_and_rejects_other_domains", test_parent_scope_preserves_child_corpus_and_rejects_other_domains},
        {"test_time_only_query_changes_freshness_without_rewriting_evidence", test_time_only_query_changes_freshness_without_rewriting_evidence},
        {"test_strict_bounds_and_request_shape", test_strict_bounds_and_request_shape},
        {"test_derived_ordering_is_stable_and_exact_input_order_retained", test_derived_ordering_is_stable_and_exact_input_order_retained}};
    for (const auto& [name, test] : tests) {
        try { test(); std::cout << "PASS " << name << '\n'; }
        catch (const std::exception& error) { std::cerr << "FAIL " << name << ": " << error.what() << '\n'; return 1; }
    }
    std::cout << tests.size() << " provider coverage cases passed\n";
}
