#include "corpus.hpp"
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
const std::string day1 = "2026-09-10T00:00:00Z", day2 = "2026-09-11T00:00:00Z", day3 = "2026-09-12T00:00:00Z";
void require(bool condition, const std::string& message) {
    if (!condition) throw std::runtime_error(message);
}
template<class F> void rejected(F&& function, const std::string& code = "corpus.invalid") {
    try { function(); }
    catch (const engine::Error& error) {
        require(error.code() == code, "expected " + code + ", got " + error.code() + ": " + error.what());
        return;
    }
    throw std::runtime_error("expected rejection: " + code);
}
engine::Request request(const std::string& operation, Json payload) {
    return {"req-corpus-test", "corr-corpus-test", operation, "symphony-scv", engine::unix_time_ms() + 60000, std::move(payload)};
}
Json call(const std::string& operation, Json payload, const std::string& domain = "scv") {
    return scv::handle_corpus(request(operation, std::move(payload)), domain);
}
Json source(const std::string& id = "fixture-doc", const std::string& provider = "cf", const std::string& family = "scev") {
    return scv::sealed(Json{{"protocol", "symphony.scv.source.v1"}, {"source_id", id}, {"provider_id", provider},
        {"family_id", family}, {"publisher", "Fixture publisher"}, {"authority_role", "documentation"}, {"scope", "fixture scope"},
        {"locators", Json::array({Json{{"locator_id", "main"}, {"uri", "https://fixture.invalid/docs?view=v1#scope"},
            {"role", "primary"}, {"format", "markdown"}, {"selector", "v1"}}})},
        {"continuity_evidence", Json::array()}, {"generation", 1}, {"predecessor_digest", nullptr}});
}
Json relocate(Json current) {
    current["predecessor_digest"] = current.at("digest");
    current["generation"] = current.at("generation").get<int>() + 1;
    current["locators"][0]["uri"] = "custom+offline.v1:retained-doc";
    current["continuity_evidence"] = Json::array({"fixture explicit continuity"});
    return scv::sealed(current);
}
Json capture(Json selected = source(), const std::string& body = "# Fixture\nA retained document.\n",
             const std::string& observed = day1, const std::string& completeness = "complete") {
    return scv::handle_source(request("capture_import", Json{{"source", selected}, {"locator_id", "main"},
        {"resolved_uri", selected.at("locators")[0].at("uri")}, {"redirects", Json::array()}, {"observed_at", observed},
        {"upstream_revision", Json{{"scheme", "etag"}, {"value", "fixture-validator"}}}, {"media_type", "text/markdown"},
        {"body", body}, {"completeness", completeness},
        {"issues", completeness == "complete" ? Json::array() : Json::array({"fixture interrupted acquisition"})}}), "scv");
}
Json index(Json cap = capture(), const std::string& domain = "scv") {
    return call("capture_index", {{"capture", cap}}, domain);
}
Json attempt(const std::string& id, Json idx) { return {{"member_id", id}, {"capture", std::move(idx)}}; }
Json build(Json attempts, Json previous = nullptr, const std::string& time = day2, const std::string& domain = "scv") {
    return call("corpus_build", {{"corpus_id", "fixture corpus"}, {"previous", previous}, {"snapshot_time", time}, {"attempts", attempts}}, domain);
}
Json query(Json corpus, const std::string& selection = "last_complete", const std::string& time = day2,
           Json age = nullptr, Json ids = Json::array(), const std::string& domain = "scv") {
    return call("corpus_query", {{"corpus", corpus}, {"query_time", time}, {"member_ids", ids},
        {"selection", selection}, {"max_age_seconds", age}}, domain);
}
Json diff(const Json& before, const Json& after) { return call("corpus_diff", {{"before", before}, {"after", after}}); }
bool reason(const Json& member, const std::string& expected) {
    for (const auto& value : member.at("reasons")) if (value == expected) return true;
    return false;
}

void test_capture_index_producer() {
    const auto cap = capture(), result = index(cap, "scev-cf");
    require(result.at("protocol") == "symphony.scv.capture-index.v1" && result.at("domain") == "scev-cf", "index owner/protocol");
    require(result.at("capture_digest") == cap.at("digest") && result.at("body_digest") == cap.at("body_digest"), "exact capture and body references");
    require(result.at("requested_uri") == cap.at("source").at("locators")[0].at("uri"), "configured version and fragment retained");
    require(result.at("source_digest") == cap.at("source").at("digest") && result.at("observed_at") == day1, "source and observation unchanged");
    require(!result.contains("body") && result == scv::sealed(result), "index sealed but cannot substitute for retained body");
    scv::validate_capture_index(result, "scev"); scv::validate_capture_index(result, "scv");
}
void test_capture_index_rejects_changed_evidence() {
    auto cap = capture(); cap["body"] = "tampered";
    rejected([&] { static_cast<void>(index(cap)); }, "scv.capture_size");
    auto idx = index(); idx["observed_at"] = day2;
    rejected([&] { scv::validate_capture_index(idx, "scv"); });
    idx = index(); idx["byte_size"] = 65537; idx = scv::sealed(idx);
    rejected([&] { scv::validate_capture_index(idx, "scv"); });
}
void test_deterministic_corpus_selection() {
    const auto a = attempt("a", index()), b = attempt("b", index(capture(source("other"))));
    const auto one = build(Json::array({a, b})), two = build(Json::array({b, a}));
    require(one == two && one.at("generation") == 1 && one.at("parent_digest").is_null(), "enumeration independent initial snapshot");
    require(one.at("members")[0].at("member_id") == "a" && one.at("coverage").at("complete_attempts") == 2, "sorted complete members");
    const auto empty = build(Json::array());
    require(empty.at("members").empty() && empty.at("coverage").at("requested_members") == 0, "explicit empty selection supported");
}
void test_failed_and_partial_refresh_preserves_complete_evidence() {
    const auto a = index(), b = index(capture(source("other")));
    const auto before = build(Json::array({attempt("a", a), attempt("b", b)}), nullptr, day1);
    const auto failed = index(capture(source(), "", day2, "failed"));
    const auto partial = index(capture(source("other"), "# Incomplete", day2, "partial"));
    const auto after = build(Json::array({attempt("a", failed), attempt("b", partial)}), before, day2);
    require(after.at("generation") == 2 && after.at("parent_digest") == before.at("digest"), "successor binds exact predecessor");
    require(after.at("members")[0].at("latest_attempt") == failed && after.at("members")[0].at("last_complete") == a, "failed attempt and complete evidence distinct");
    require(after.at("members")[1].at("last_complete") == b, "partial attempt cannot replace complete evidence");
    require(after.at("coverage") == Json{{"requested_members", 2}, {"complete_attempts", 0}, {"partial_attempts", 1},
        {"failed_attempts", 1}, {"retained_complete_members", 2}}, "coverage does not relabel retained captures fresh");
    const auto selected = query(after, "last_complete", day2, 60);
    require(selected.at("members")[0].at("freshness") == "expired" && selected.at("members")[0].at("selected").at("observed_at") == day1, "retained evidence stays honestly aged");
    require(reason(selected.at("members")[0], "selected_previous_complete"), "retention disposition explicit");
    require(query(after, "latest_attempt").at("members")[0].at("status") == "failed", "latest attempt selection never silently falls back");
}
void test_historical_source_revision_is_explicit() {
    const auto original = source(), revised = relocate(original);
    const auto old = index(capture(original));
    const auto before = build(Json::array({attempt("doc", old)}), nullptr, day1);
    const auto failure = index(capture(revised, "", day2, "failed"));
    const auto after = build(Json::array({attempt("doc", failure)}), before, day2);
    const auto selected = query(after).at("members")[0];
    require(selected.at("source_revision_matches_latest") == false && selected.at("selected").at("source_generation") == 1, "relocation does not rewrite old complete provenance");
    require(reason(selected, "source_revision_differs_from_latest_attempt"), "source mismatch visible");
    const auto historical = build(Json::array({attempt("doc", old)}), after, day3);
    require(historical.at("members")[0].at("latest_attempt") == old, "explicit historical source revision remains selectable");
    require(query(historical, "last_complete", day3, 60).at("members")[0].at("freshness") == "expired", "historical selection not new observation");
}
void test_selection_removal_is_not_retirement() {
    const auto before = build(Json::array({attempt("doc", index())}));
    const auto after = build(Json::array(), before, day3), result = diff(before, after);
    require(result.at("removed_member_ids") == Json::array({"doc"}) && result.at("affected_member_ids") == Json::array({"doc"}), "removed selection explicit");
    require(result.at("changes").empty() && !result.contains("retired"), "no retirement inference");
    require(before.at("members").size() == 1, "old artifact retained unchanged");
}
void test_corpus_query_freshness_and_explicit_selection() {
    const auto corpus = build(Json::array({attempt("doc", index())}));
    require(query(corpus, "last_complete", "2026-09-10T00:00:10Z", 10).at("members")[0].at("freshness") == "current", "age boundary inclusive");
    require(query(corpus, "last_complete", "2026-09-10T00:00:11Z", 10).at("members")[0].at("freshness") == "expired", "age boundary expires next second");
    require(query(corpus, "last_complete", "2026-09-09T23:59:59Z").at("members")[0].at("freshness") == "future", "future observation never current");
    require(query(corpus, "last_complete", day3).at("members")[0].at("freshness") == "current", "no age bound is explicit temporal policy");
    const auto failed = build(Json::array({attempt("doc", index(capture(source(), "", day1, "failed")))}));
    const auto missing = query(failed).at("members")[0];
    require(missing.at("selected").is_null() && missing.at("status") == "unavailable" && missing.at("freshness") == "not_selected" &&
        missing.at("source_revision_matches_latest").is_null(), "missing prior complete evidence remains absent");
    rejected([&] { static_cast<void>(query(corpus, "implicit")); });
    rejected([&] { static_cast<void>(query(corpus, "last_complete", day2, nullptr, Json::array({"unknown"}))); });
    rejected([&] { static_cast<void>(query(corpus, "last_complete", day2, nullptr, Json::array({"doc", "doc"}))); });
}
void test_query_subset_coverage_is_finite() {
    const auto corpus = build(Json::array({attempt("a", index()), attempt("b", index(capture(source("other"), "", day1, "failed")))}));
    const auto selected = query(corpus, "latest_attempt", day2, nullptr, Json::array({"a"}));
    require(selected.at("members").size() == 1 && selected.at("coverage").at("requested_members") == 1 &&
        selected.at("coverage").at("failed_attempts") == 0, "coverage restricted to requested subset, independent of entire corpus");
}
void test_corpus_diff_separates_change_dimensions() {
    const auto a = index(), before = build(Json::array({attempt("doc", a)}), nullptr, day1);
    const auto refetch = build(Json::array({attempt("doc", index(capture(source(), "# Fixture\nA retained document.\n", day2)))}), before, day2);
    const auto retrieval = diff(before, refetch).at("changes")[0];
    require(retrieval.at("body_changed") == false && retrieval.at("source_changed") == false && retrieval.at("observation_changed") == true &&
        retrieval.at("coverage_changed") == false && retrieval.at("last_complete_changed") == true, "same body remains distinct observation");
    const auto changed = build(Json::array({attempt("doc", index(capture(relocate(source()), "partial different", day2, "partial")))}), before, day2);
    const auto change = diff(before, changed).at("changes")[0];
    require(change.at("body_changed") == true && change.at("source_changed") == true && change.at("coverage_changed") == true &&
        change.at("last_complete_changed") == false, "content/config/coverage separate from retained complete selection");
    const auto same = build(Json::array({attempt("doc", a)}), before, day2);
    require(diff(before, same).at("affected_member_ids").empty(), "snapshot generation/time alone does not alter evidence");
}
void test_member_identity_and_owner_collisions() {
    const auto a = index(), before = build(Json::array({attempt("doc", a)}));
    rejected([&] { static_cast<void>(build(Json::array({attempt("doc", a), attempt("doc", a)}))); });
    rejected([&] { static_cast<void>(build(Json::array({attempt("doc", a), attempt("alias", a)}))); });
    rejected([&] { static_cast<void>(build(Json::array({attempt("doc", index(capture(source("different"))))}), before)); });
    const auto conflicting = index(capture(source("fixture-doc", "do", "schv")));
    rejected([&] { static_cast<void>(build(Json::array({attempt("a", a), attempt("b", conflicting)}))); });
    auto second_locator = source();
    auto locator = second_locator["locators"][0]; locator["locator_id"] = "other";
    second_locator["locators"].push_back(locator); second_locator = scv::sealed(second_locator);
    auto second_cap = capture(second_locator); second_cap["locator_id"] = "other"; second_cap = scv::sealed(second_cap);
    require(build(Json::array({attempt("main", index(capture(second_locator))), attempt("other", index(second_cap))})).at("members").size() == 2,
        "distinct locators of one source remain distinct members");
}
void test_corpus_consumer_rejects_resealed_inconsistency() {
    const auto initial = build(Json::array({attempt("doc", index())}));
    auto bad = initial; bad["coverage"]["requested_members"] = 2; bad = scv::sealed(bad);
    rejected([&] { static_cast<void>(query(bad)); });
    bad = initial; bad["members"][0]["last_complete"] = nullptr; bad = scv::sealed(bad);
    rejected([&] { scv::validate_corpus(bad, "scv"); });
    bad = initial; bad["members"][0]["latest_attempt"]["byte_size"] = 1; bad = scv::sealed(bad);
    rejected([&] { scv::validate_corpus(bad, "scv"); });
    bad = initial; bad["generation"] = 2; bad = scv::sealed(bad);
    rejected([&] { scv::validate_corpus(bad, "scv"); });
    const auto failed = index(capture(source(), "", day1, "failed"));
    bad = build(Json::array({attempt("doc", failed)})); bad["members"][0]["last_complete"] = index();
    bad["coverage"]["retained_complete_members"] = 1; bad = scv::sealed(bad);
    rejected([&] { scv::validate_corpus(bad, "scv"); });
}
void test_index_references_cannot_disagree() {
    const auto original = index(), before = build(Json::array({attempt("doc", original)}));
    auto forged = original; forged["observed_at"] = day2; forged = scv::sealed(forged);
    rejected([&] { static_cast<void>(build(Json::array({attempt("doc", forged)}), before)); });
    forged = original; forged["source_generation"] = 2; forged["capture_digest"] = index(capture(source("different"))).at("capture_digest");
    forged = scv::sealed(forged);
    rejected([&] { static_cast<void>(build(Json::array({attempt("doc", forged)}), before)); });
}
void test_parent_composition_and_exact_corpus_domain() {
    const auto cf = index(capture(), "scev-cf"), other = index(capture(source("do-doc", "do", "schv")), "schv-do");
    const auto combined = build(Json::array({attempt("cf", cf), attempt("do", other)}));
    require(combined.at("members").size() == 2, "SCV composes subordinate provider indexes");
    require(build(Json::array({attempt("cf", cf)}), nullptr, day2, "scev").at("domain") == "scev", "family consumes own provider index");
    rejected([&] { static_cast<void>(build(Json::array({attempt("do", other)}), nullptr, day2, "scev")); });
    rejected([&] { static_cast<void>(query(combined, "last_complete", day2, nullptr, Json::array(), "scev")); });
    auto forged = cf; forged["domain"] = "schv-do"; forged = scv::sealed(forged);
    rejected([&] { scv::validate_capture_index(forged, "scv"); });
}
void test_snapshot_time_and_generation_bounds() {
    const auto idx = index();
    rejected([&] { static_cast<void>(build(Json::array({attempt("doc", idx)}), nullptr, "2026-09-09T00:00:00Z")); });
    const auto previous = build(Json::array({attempt("doc", idx)}), nullptr, day2);
    rejected([&] { static_cast<void>(build(Json::array({attempt("doc", idx)}), previous, day1)); });
    auto maximum = previous; maximum["generation"] = 9007199254740991LL; maximum["parent_digest"] = previous.at("digest"); maximum = scv::sealed(maximum);
    rejected([&] { static_cast<void>(build(Json::array(), maximum)); });
    rejected([&] { static_cast<void>(query(previous, "last_complete", day2, 18446744073709551615ULL)); });
    rejected([&] { static_cast<void>(query(previous, "last_complete", day2, -1)); });
    rejected([&] { static_cast<void>(query(previous, "last_complete", "2026-09-10T00:00:00+00:00")); });
}
void test_corpus_bounds_and_deadline() {
    Json attempts = Json::array();
    for (int i = 0; i < 128; ++i) attempts.push_back(attempt("member-" + std::to_string(i), index(capture(source("source-" + std::to_string(i))))));
    require(build(attempts).at("members").size() == 128, "bounded corpus can exceed one graph slice");
    attempts.push_back(attempt("overflow", index(capture(source("overflow")))));
    rejected([&] { static_cast<void>(build(attempts)); });
    rejected([&] { static_cast<void>(build(Json::array({attempt(std::string(129, 'x'), index())}))); });
    auto expired = request("capture_index", {{"capture", capture()}}); expired.deadline_unix_ms = 0;
    rejected([&] { static_cast<void>(scv::handle_corpus(expired, "scv")); }, "request.deadline_exceeded");
    rejected([&] { static_cast<void>(call("unknown", Json::object())); }, "operation.unsupported");
}
void test_corpus_real_request_round_trip() {
    const auto now = engine::unix_time_ms();
    const Json envelope = {{"protocol", engine::process_protocol_v1}, {"request_id", "corpus-real-request"},
        {"correlation_id", "corpus-real-correlation"}, {"operation", "corpus_build"}, {"target_engine", "symphony-scv"},
        {"deadline_unix_ms", now + 60000}, {"payload", {{"corpus_id", "fixture corpus"}, {"previous", nullptr},
            {"snapshot_time", day2}, {"attempts", Json::array({attempt("doc", index())})}}}};
    const auto parsed = engine::parse_request(envelope.dump(), "symphony-scv", now);
    const auto result = scv::handle_corpus(parsed, "scv");
    const auto wire = engine::serialize_response(engine::success_response(parsed, "symphony-scv", scv::version, result));
    const auto response = engine::parse_bounded_json(wire, engine::Limits::max_response_bytes);
    require(response.at("result") == result && result.at("members").size() == 1, "real bounded process request and response preserve corpus");
    auto unknown = envelope; unknown["payload"]["automatic_selection"] = true;
    rejected([&] { static_cast<void>(scv::handle_corpus(engine::parse_request(unknown.dump(), "symphony-scv", now), "scv")); });
}
}
int main() {
    const std::vector<std::pair<std::string, std::function<void()>>> tests = {
        {"capture_index_producer", test_capture_index_producer},
        {"capture_index_rejects_changed_evidence", test_capture_index_rejects_changed_evidence},
        {"deterministic_corpus_selection", test_deterministic_corpus_selection},
        {"failed_and_partial_refresh_preserves_complete_evidence", test_failed_and_partial_refresh_preserves_complete_evidence},
        {"historical_source_revision_is_explicit", test_historical_source_revision_is_explicit},
        {"selection_removal_is_not_retirement", test_selection_removal_is_not_retirement},
        {"corpus_query_freshness_and_explicit_selection", test_corpus_query_freshness_and_explicit_selection},
        {"query_subset_coverage_is_finite", test_query_subset_coverage_is_finite},
        {"corpus_diff_separates_change_dimensions", test_corpus_diff_separates_change_dimensions},
        {"member_identity_and_owner_collisions", test_member_identity_and_owner_collisions},
        {"corpus_consumer_rejects_resealed_inconsistency", test_corpus_consumer_rejects_resealed_inconsistency},
        {"index_references_cannot_disagree", test_index_references_cannot_disagree},
        {"parent_composition_and_exact_corpus_domain", test_parent_composition_and_exact_corpus_domain},
        {"snapshot_time_and_generation_bounds", test_snapshot_time_and_generation_bounds},
        {"corpus_bounds_and_deadline", test_corpus_bounds_and_deadline},
        {"corpus_real_request_round_trip", test_corpus_real_request_round_trip},
    };
    try {
        for (const auto& [name, test] : tests) { test(); std::cout << "PASS " << name << '\n'; }
    } catch (const std::exception& error) { std::cerr << error.what() << '\n'; return 1; }
    return 0;
}
