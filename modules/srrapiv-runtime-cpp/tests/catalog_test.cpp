#include "catalog.hpp"
#include "catalog_fixture.hpp"
#include "symphony/knowledge/engine/digest.hpp"

#include <algorithm>
#include <iostream>
#include <source_location>
#include <stdexcept>
#include <string>

using symphony::srrapiv::Json;
using symphony::srrapiv::catalog_operation;
using symphony::knowledge::engine::tagged_sha256;
using symphony::srrapiv::test_fixture::seal;
using symphony::srrapiv::test_fixture::artifact;
using symphony::srrapiv::test_fixture::binding;
using symphony::srrapiv::test_fixture::input;
namespace {
unsigned assertions_run = 0;
void check(bool value, const std::source_location at = std::source_location::current()) {
    ++assertions_run;
    if (!value) throw std::runtime_error("catalog assertion failed at line " + std::to_string(at.line()));
}
Json evidence(const Json& b, const std::string& capability, const std::string& result = "supported",
              const std::string& scope = "provider_test") {
    return {{"binding_id",b.at("binding_id")},{"capability",capability},{"result",result},
            {"evidence_id","evidence:" + capability + ":" + result + ":" + scope},
            {"scope",scope},{"observed_at","2026-11-06T00:00:00Z"}};
}
void error(const Json& result, const std::string& code) {
    check(!result.at("ok").get<bool>());
    check(result.at("errors").at(0).at("code") == code);
}
Json mapping(const std::string& id, const std::string& disposition = "implemented") {
    const bool mapped = disposition == "implemented" || disposition == "equivalent";
    return {{"element_id",id},{"disposition",disposition},
            {"handler_id",mapped ? Json("handler:" + id) : Json(nullptr)},
            {"operation_id",mapped ? Json("engop:srrapiv:" + id) : Json(nullptr)},
            {"qxctl_route",mapped ? Json("qxctl sqv srrapiv invoke") : Json(nullptr)},
            {"input_schema",mapped ? Json("input.v1") : Json(nullptr)},
            {"result_schema",mapped ? Json("result.v1") : Json(nullptr)},
            {"event_schemas",Json::array()},{"evidence_ids",mapped ? Json::array({"evidence:" + id}) : Json::array()}};
}
}

int main(int argc, char** argv) {
    const auto a = artifact();
    const auto original = input(a);
    if (argc == 2 && std::string(argv[1]) == "--sample-input") {
        std::cout << original.dump(2) << '\n';
        return 0;
    }
    const auto baseline = original.dump();
    auto r = catalog_operation("binding_validate",original);
    check(r.at("ok") == true && r.at("valid") == true);
    check(r.at("compatibility") == "unknown");
    check(r.at("assertions_verified") == false && r.at("live_qualified") == false);
    check(r.at("provider_activity_authorized") == false);
    check(original.dump() == baseline);

    auto x = original;
    x["compatibility_evidence"].push_back(evidence(x["binding"],"getVersion"));
    check(catalog_operation("binding_validate",x).at("compatibility") == "partial");
    x["compatibility_evidence"].push_back(evidence(x["binding"],"subscribe"));
    r = catalog_operation("binding_validate",x);
    check(r.at("compatibility") == "supported" && r.at("live_qualified") == false);
    x["compatibility_evidence"].push_back(evidence(x["binding"],"subscribe","incompatible"));
    check(catalog_operation("binding_validate",x).at("compatibility") == "conflicted");
    x["compatibility_evidence"] = Json::array({evidence(x["binding"],"subscribe","incompatible")});
    check(catalog_operation("binding_validate",x).at("compatibility") == "incompatible");
    x["compatibility_evidence"] = Json::array({evidence(x["binding"],"subscribe","supported","native_fixture")});
    check(catalog_operation("binding_validate",x).at("compatibility") == "unknown");

    // Evidence binds the entire immutable consumer/environment/account selection.
    x = original;
    x["compatibility_evidence"] = Json::array({evidence(x["binding"],"getVersion"), evidence(x["binding"],"subscribe")});
    x["binding"]["environment"] = "DifferentEnvironment";
    seal(x["binding"],"binding_id");
    check(catalog_operation("binding_validate",x).at("compatibility") == "unknown");

    x = original;
    x["binding"]["required_capabilities"].push_back("notAvailable");
    seal(x["binding"],"binding_id");
    r = catalog_operation("binding_validate",x);
    check(r.at("valid") == true && r.at("compatibility") == "incompatible");
    check(r.at("capabilities").at(1).at("state") == "unavailable");

    // Every exact identity is checked; no alias, matching-release or newest fallback.
    x = original;
    x["binding"]["artifact_id"] = tagged_sha256("missing"); seal(x["binding"],"binding_id");
    error(catalog_operation("binding_validate",x),"exact_artifact_not_found");
    x = original; x["binding"]["artifact_id"] = "latest"; seal(x["binding"],"binding_id");
    error(catalog_operation("binding_validate",x),"sha256_required");
    x = original; x["binding"]["account_ref"] = "account:changed";
    error(catalog_operation("binding_validate",x),"content_identity_mismatch");
    x = original; x["catalog"]["artifacts"][0]["sdk_version"] = "13.8.0.0";
    error(catalog_operation("binding_validate",x),"content_identity_mismatch");
    x = original; x["catalog"]["artifacts"].push_back(a);
    error(catalog_operation("binding_validate",x),"duplicate_artifact");
    x = original; x["binding"]["required_capabilities"].push_back("subscribe"); seal(x["binding"],"binding_id");
    error(catalog_operation("binding_validate",x),"duplicate_value");

    // Unknown and secret-bearing keys are not accepted or reflected in diagnostics.
    x = original; x["binding"]["credential_reference"]["password"] = "DO-NOT-RETURN-ME";
    r = catalog_operation("binding_validate",x);
    error(r,"unknown_field"); check(r.dump().find("DO-NOT-RETURN-ME") == std::string::npos);
    x = original; x["binding"]["credential_reference"] = "DO-NOT-RETURN-ME";
    error(catalog_operation("binding_validate",x),"object_required");
    x = original; x["binding"]["credential_reference"]["resource_id"] = "https://host/password";
    error(catalog_operation("binding_validate",x),"opaque_reference_required");
    x = original; x["binding"]["credential_reference"]["DO-NOT-RETURN-ME"] = true;
    r = catalog_operation("binding_validate",x);
    error(r,"unknown_field"); check(r.dump().find("DO-NOT-RETURN-ME") == std::string::npos);

    auto older = artifact("13.6.0.0");
    auto newer = artifact("13.7.0.0","state.v2","journal.v2");
    newer["contracts"]["readable_state_schemas"].push_back("state.v1");
    newer["contracts"]["readable_journal_schemas"].push_back("journal.v1");
    newer["capabilities"].push_back("newCapability");
    seal(newer,"artifact_id");
    Json compare = {{"catalog",{{"schema","symphony.srrapiv.catalog.v1"},{"artifacts",Json::array({older,newer})}}},
                    {"from_binding",binding(older)},{"to_binding",binding(newer)},
                    {"compatibility_evidence",Json::array()}};
    const auto compare_original = compare.dump();
    r = catalog_operation("binding_compare",compare);
    check(r.at("ok") == true && r.at("selection_changed") == true);
    check(r.at("forward_state_readable") == true && r.at("forward_journal_readable") == true);
    check(r.at("rollback_state_readable") == false && r.at("rollback_journal_readable") == false);
    check(r.at("migration_required") == false && r.at("rollback_reader_gap") == true);
    check(r.at("automatically_applied") == false && r.at("provider_state_rollback") == false);
    check(r.at("added_capabilities") == Json::array({"newCapability"}));
    check(compare.dump() == compare_original);
    std::swap(compare["from_binding"], compare["to_binding"]);
    r = catalog_operation("binding_compare",compare);
    check(r.at("migration_required") == true && r.at("forward_state_readable") == false);
    check(r.at("rollback_state_readable") == true);
    check(r.at("removed_capabilities") == Json::array({"newCapability"}));

    // Same SDK label can have different immutable compiler/trust/mapping artifacts.
    auto variant = a;
    variant["platform"]["compiler"] = "clang-18";
    variant["connection"]["trust_digest"] = tagged_sha256("rotated-trust");
    variant["contracts"]["api_mapping"] = "rapi-mapping.v2";
    seal(variant,"artifact_id");
    compare = {{"catalog",{{"schema","symphony.srrapiv.catalog.v1"},{"artifacts",Json::array({a,variant})}}},
               {"from_binding",binding(a)},{"to_binding",binding(variant)},
               {"compatibility_evidence",Json::array()}};
    compare["to_binding"]["metadata_digest"] = tagged_sha256("metadata-changed");
    compare["to_binding"]["feed_profile_digest"] = tagged_sha256("aggregated-feed");
    compare["to_binding"]["credential_reference"]["resource_id"] = "credential:rotated";
    seal(compare["to_binding"],"binding_id");
    r = catalog_operation("binding_compare",compare);
    for (const auto* path : {"/artifact/platform/compiler","/artifact/connection/trust_digest",
                             "/artifact/contracts/api_mapping","/binding/metadata_digest",
                             "/binding/feed_profile_digest","/binding/credential_reference/resource_id"}) {
        check(std::find(r.at("changed_paths").begin(),r.at("changed_paths").end(),path) != r.at("changed_paths").end());
    }
    auto changed_strategy = compare;
    changed_strategy["to_binding"]["strategy_id"] = "strategy:another";
    seal(changed_strategy["to_binding"],"binding_id");
    error(catalog_operation("binding_compare",changed_strategy),"strategy_identity_mismatch");

    // Exact selection is stable irrespective of catalogue order or newer releases.
    Json inspect = {{"catalog",compare.at("catalog")},{"artifact_id",a.at("artifact_id")}};
    r = catalog_operation("catalog_inspect",inspect);
    check(r.at("artifacts").size() == 1 && r.at("artifacts")[0] == a);
    check(r.at("substitution_performed") == false && r.at("artifact_bytes_verified") == false);
    std::reverse(inspect["catalog"]["artifacts"].begin(), inspect["catalog"]["artifacts"].end());
    check(catalog_operation("catalog_inspect",inspect).at("artifacts")[0] == a);
    inspect["artifact_id"] = tagged_sha256("absent");
    error(catalog_operation("catalog_inspect",inspect),"exact_artifact_not_found");
    inspect.erase("artifact_id");
    check(catalog_operation("catalog_inspect",inspect).at("artifacts").size() == 2);

    Json cov = {{"schema","symphony.srrapiv.coverage.v1"},{"sdk_artifact_id",a.at("artifact_id")},
                {"universe",Json::array({"method:sendOrder:legacy","callback:LineInfo","field:quantity"})},
                {"mappings",Json::array({mapping("method:sendOrder:legacy"),mapping("callback:LineInfo"),mapping("field:quantity")})}};
    r = catalog_operation("coverage_assess",cov);
    check(r.at("complete_for_supplied_universe") == true && r.at("total_elements") == 3);
    check(r.at("sdk_universe_independently_verified") == false && r.at("implementation_evidence_verified") == false);
    auto partial = cov; partial["mappings"].erase(2);
    r = catalog_operation("coverage_assess",partial);
    check(r.at("complete_for_supplied_universe") == false && r.at("unresolved")[0].at("reason") == "unmapped");
    partial["mappings"].push_back(mapping("field:quantity","provider_unavailable"));
    r = catalog_operation("coverage_assess",partial);
    check(r.at("complete_for_supplied_universe") == false && r.at("total_elements") == 3);
    partial = cov; partial["mappings"][0]["disposition"] = "equivalent";
    error(catalog_operation("coverage_assess",partial),"equivalence_evidence_required");
    partial["mappings"][0]["equivalence_evidence_id"] = "evidence:equivalent";
    check(catalog_operation("coverage_assess",partial).at("complete_for_supplied_universe") == true);
    partial = cov; partial["mappings"][0]["evidence_ids"] = Json::array();
    error(catalog_operation("coverage_assess",partial),"array_size");
    partial = cov; partial["mappings"].push_back(mapping("method:outside"));
    error(catalog_operation("coverage_assess",partial),"element_outside_universe");
    partial = cov; partial["mappings"].push_back(mapping("field:quantity"));
    error(catalog_operation("coverage_assess",partial),"duplicate_mapping");
    partial = cov; partial["universe"] = Json::array();
    error(catalog_operation("coverage_assess",partial),"array_size");

    // Bounded structural rejection: malformed types, future schema, arrays, platforms.
    error(catalog_operation("unknown",Json::object()),"unknown_operation");
    error(catalog_operation("catalog_inspect",nullptr),"object_required");
    x = original; x.erase("compatibility_evidence");
    error(catalog_operation("binding_validate",x),"field_required");
    x = original; x["binding"]["schema"] = "symphony.srrapiv.binding.v2";
    error(catalog_operation("binding_validate",x),"unsupported_schema");
    x = original; x["binding"]["required_capabilities"] = "subscribe";
    error(catalog_operation("binding_validate",x),"array_required");
    x = original; x["catalog"]["artifacts"][0]["platform"]["os"] = "windows";
    error(catalog_operation("binding_validate",x),"unsupported_value");
    x = original; x["catalog"]["artifacts"][0]["contracts"]["readable_state_schemas"] = Json::array({"other.v1"});
    error(catalog_operation("binding_validate",x),"own_writer_reader_mismatch");
    x = original; x["catalog"]["artifacts"] = Json::array();
    for (unsigned i = 0; i < 257; ++i) x["catalog"]["artifacts"].push_back(a);
    error(catalog_operation("binding_validate",x),"array_size");
    std::cout << "catalog: " << assertions_run << " focused assertions passed\n";
}
