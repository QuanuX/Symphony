#include "scv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"

#include <cstdlib>
#include <iostream>
#include <string>

namespace scv = symphony::knowledge::scv;
namespace engine = symphony::knowledge::engine;
using engine::Json;
namespace {
void require(bool condition, const std::string& name) {
    if (!condition) { std::cerr << name << '\n'; std::exit(1); }
}
Json call(const std::string& op, Json payload, const std::string& domain = "scv") {
    return scv::handle_source(engine::Request{"test", "test", op, "symphony-" + domain,
        engine::unix_time_ms() + 5000, std::move(payload)}, domain);
}
void rejects(const std::string& op, const Json& payload, const std::string& code, const std::string& domain = "scv") {
    try { (void)call(op, payload, domain); }
    catch (const engine::Error& error) { require(error.code() == code, "wrong failure: " + error.code() + " expected " + code); return; }
    require(false, "request was not rejected: " + code);
}
Json desired() {
    return Json{{"source_id", "fixture-docs"}, {"provider_id", "cf"}, {"family_id", "scev"}, {"publisher", "Fixture publisher"},
        {"authority_role", "user_declared"}, {"scope", "Synthetic workflow fixture"},
        {"locators", Json::array({Json{{"locator_id", "docs"}, {"uri", "https://example.invalid/docs.md"},
            {"role", "preferred"}, {"format", "markdown"}, {"selector", "revision-a"}}})},
        {"continuity_evidence", Json::array({"fixture-owned publication"})}};
}
Json plan(const Json& current, const Json& wanted, std::string id = "fixture-change") {
    return call("source_plan", Json{{"operation_id", id}, {"current", current}, {"desired", wanted}, {"reason", "Fixture transition"}});
}
Json capture(const Json& source) {
    return call("capture_import", Json{{"source", source}, {"locator_id", "docs"}, {"resolved_uri", source.at("locators")[0].at("uri")},
        {"redirects", Json::array()}, {"observed_at", "2026-09-10T12:00:00Z"}, {"upstream_revision", Json{{"scheme", "fixture"}, {"value", "a"}}},
        {"media_type", "text/markdown"}, {"body", "# Synthetic product\nDefault retention: 7 days.\n"},
        {"completeness", "complete"}, {"issues", Json::array()}});
}
}
void test_source_revision_and_capture_contracts() {
    const auto initial = plan(nullptr, desired());
    const auto source = initial.at("source");
    require(source.at("generation") == 1 && source.at("predecessor_digest").is_null(), "initial lineage");
    require(call("source_apply", Json{{"plan", initial}, {"current", nullptr}}).at("state") == source, "pure onboarding reduction");
    const auto before = capture(source);
    auto relocated = desired(); relocated["locators"][0]["uri"] = "https://example.invalid/new/docs.md";
    const auto next = plan(source, relocated, "move-one");
    require(next.at("change_kind") == "relocate", "relocation classified");
    require(next.at("source").at("source_id") == source.at("source_id"), "source identity retained");
    require(next.at("source").at("predecessor_digest") == source.at("digest"), "predecessor retained");
    require(call("source_apply", Json{{"plan", next}, {"current", source}}).at("state") == next.at("source"), "relocation reducer");
    rejects("source_apply", Json{{"plan", next}, {"current", next.at("source")}}, "scv.stale_state");
    const auto after = capture(next.at("source"));
    require(before.at("body_digest") == after.at("body_digest") && before.at("digest") != after.at("digest"), "same bytes, distinct provenance");
    auto comparison = call("capture_compare", Json{{"before", before}, {"after", after}});
    require(comparison.at("source_identity_preserved") && comparison.at("configuration_changed"), "source move comparison");
    auto rollback = plan(next.at("source"), desired(), "restore-one");
    require(rollback.at("source").at("generation") == 3 && rollback.at("source").at("predecessor_digest") == next.at("source").at("digest"), "rollback moves forward");
    auto authority = desired(); authority["publisher"] = "Transferred fixture publication";
    require(plan(source, authority).at("change_kind") == "authority_change", "authority change cannot masquerade as relocation");
    auto other = desired(); other["source_id"] = "replacement-docs";
    rejects("source_plan", Json{{"operation_id", "change"}, {"current", source}, {"desired", other}, {"reason", "replacement"}}, "scv.identity_change");
    auto tampered = next; tampered["change_kind"] = "revise"; tampered = scv::sealed(tampered, "plan_digest");
    rejects("source_apply", Json{{"plan", tampered}, {"current", source}}, "scv.plan_mismatch");
    auto corrupted = before; corrupted["body"] = "different bytes"; corrupted = scv::sealed(corrupted);
    try { scv::validate_capture(corrupted, "scv"); require(false, "tampered capture accepted"); }
    catch (const engine::Error&) {}
    auto failed = before; failed["completeness"] = "failed"; failed["issues"] = Json::array({"retrieval_timeout"}); failed = scv::sealed(failed);
    scv::validate_capture(failed, "scev-cf");
    require(call("capture_compare", Json{{"before", before}, {"after", failed}}).at("coverage_after") == "failed", "acquisition failure preserves evidence");
    auto missing = failed; missing["issues"] = Json::array(); missing = scv::sealed(missing);
    rejects("capture_compare", Json{{"before", before}, {"after", missing}}, "scv.completeness");
    rejects("source_status", Json{{"source", source}}, "scv.domain", "schv");
    rejects("source_status", Json{{"source", source}}, "scv.domain", "schv-gcp");
    require(call("source_status", Json{{"source", source}}, "scev-cf").at("state_digest") == source.at("digest"), "provider standalone scope");
    auto extension = desired(); extension["provider_id"] = "newco"; extension["family_id"] = "schv";
    auto provider = call("provider_onboard", Json{{"provider_id", "newco"}, {"family_id", "schv"}, {"display_name", "New provider fixture"}, {"sources", Json::array({extension})}}, "schv");
    require(provider.at("disposition") == "candidate", "new provider discovered without code switch or false expertise");
    auto secret = desired(); secret["locators"][0]["uri"] = "https://user:password@example.invalid/docs";
    rejects("source_plan", Json{{"operation_id", "unsafe"}, {"current", nullptr}, {"desired", secret}, {"reason", "fixture"}}, "scv.locator");
    for (const auto* value : {":", "1invalid:doc", "bad_scheme:doc"}) {
        auto invalid = desired(); invalid["locators"][0]["uri"] = value;
        rejects("source_plan", Json{{"operation_id", "scheme"}, {"current", nullptr}, {"desired", invalid}, {"reason", "fixture"}}, "scv.locator");
    }
    auto offline = desired(); offline["locators"][0]["uri"] = "custom+offline.v1:docs/revision-1";
    require(plan(nullptr, offline).at("source").at("locators") == offline.at("locators"), "custom offline source scheme is preserved");
    auto wrong = desired(); wrong["unknown"] = true;
    rejects("source_plan", Json{{"operation_id", "unknown"}, {"current", nullptr}, {"desired", wrong}, {"reason", "fixture"}}, "scv.fields");
    std::cout << "SCV source identity, transition, capture and onboarding checks passed\n";
}
int main() { test_source_revision_and_capture_contracts(); }
