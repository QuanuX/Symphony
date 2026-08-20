#include "sav.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"

#include <algorithm>
#include <cassert>
#include <iostream>
#include <set>
#include <string_view>

namespace sav = symphony::knowledge::sav;
namespace engine = symphony::knowledge::engine;

namespace {

engine::Request request(std::string operation, engine::Json payload) {
    return {"test", "test", std::move(operation), sav::engine_id, 1, std::move(payload)};
}

engine::Json source() {
    engine::Json value{{"source_id", "source:test"}, {"owner_vector", "test"},
        {"owner_contract", "knowledge/sav/SPEC.md"}, {"protocol", "test.v1"},
        {"authority_role", "derived_evidence"}, {"collection_state", "available"},
        {"content_digest", nullptr}, {"observation_digest", nullptr}, {"observed_at", "2026-08-20T12:00:00Z"},
        {"freshness", "current"}, {"payload", {{"ready", true}}}};
    value["content_digest"] = engine::tagged_sha256(value.at("payload").dump());
    return value;
}

engine::Json current_input() {
    return {{"protocol", "symphony.sav.current-resolution-input.v1"}, {"tops_id", "tops:test"},
        {"operation_id", "operation:test"}, {"snapshot_started_at", "2026-08-20T12:00:00Z"},
        {"snapshot_completed_at", "2026-08-20T12:00:01Z"}, {"named_version_id", nullptr},
        {"named_version_digest", nullptr}, {"declared_scope", engine::Json::array({"scope:test"})},
        {"required_source_ids", engine::Json::array({"source:test"})}, {"sources", engine::Json::array({source()})}};
}

void finalize(engine::Json& value, const char* field) {
    value.erase(field);
    value[field] = engine::tagged_sha256(value.dump());
}

engine::Json requirement(const std::string& id) {
    return {{"id", id}, {"version", "1.0.0"},
            {"digest", engine::tagged_sha256(id)}, {"required", true}};
}

template <typename Function>
void require_error(Function&& function, std::string_view code) {
    bool rejected = false;
    try { function(); }
    catch (const engine::Error& error) { rejected = error.code() == code; }
    assert(rejected);
}

}

int main() {
    const auto descriptor = sav::descriptor();
    assert(descriptor.at("protocol") == engine::descriptor_protocol_v2);
    assert(descriptor.at("format_version") == 2);
    assert(descriptor.size() == 17U);
    const std::set<std::string> descriptor_fields{
        "protocol", "format_version", "module_id", "engine_id", "vector_id", "engine_version",
        "process_protocols", "contract_versions", "operations", "limits", "supported_scopes",
        "language", "thermal_path", "canonical_apply_enabled", "session_mutation_enabled",
        "network_listener", "descriptor_digest"};
    for (const auto& [field, value] : descriptor.items()) {
        static_cast<void>(value); assert(descriptor_fields.contains(field));
    }
    assert(descriptor.at("canonical_apply_enabled") == false);
    assert(descriptor.at("limits").at("json_values") == engine::Limits::max_json_values);
    auto descriptor_without_digest = descriptor;
    descriptor_without_digest.erase("descriptor_digest");
    assert(descriptor.at("descriptor_digest") == engine::tagged_sha256(descriptor_without_digest.dump()));
    const auto& operations = descriptor.at("operations");
    const auto named_descriptor = std::find_if(operations.begin(), operations.end(), [](const auto& value) {
        return value.at("engine_operation_id") == "engop:symphony:sav.named-version.validate";
    });
    assert(named_descriptor != operations.end());
    assert(named_descriptor->at("input_protocol") == "symphony.sav.named-version-validation-input.v1");
    assert(named_descriptor->at("feature_ids") == engine::Json::array({"ssfv:symphony:sav-engine.named-version"}));
    const auto current_descriptor = std::find_if(operations.begin(), operations.end(), [](const auto& value) {
        return value.at("operation_name") == "current_resolve";
    });
    assert(current_descriptor != operations.end());
    assert(current_descriptor->at("engine_operation_id") == "engop:symphony:sav.current.resolve");
    const auto current = sav::handle_request(request("current_resolve", current_input()));
    assert(current.at("coverage_state") == "complete");
    assert(current.at("canonical") == false);
    assert(current.at("snapshot_digest").get<std::string>().starts_with("sha256:"));
    auto absent = current_input();
    absent["sources"] = engine::Json::array();
    const auto unknown = sav::handle_request(request("current_resolve", absent));
    assert(unknown.at("coverage_state") == "unknown");

    engine::Json named{{"protocol", "symphony.sav.named-version.v1"},
        {"named_version_id", "savver:symphony:baseline"}, {"alias", "Baseline"},
        {"predecessor_digest", nullptr}, {"component_requirements", engine::Json::array({requirement("component:a")})},
        {"contract_requirements", engine::Json::array()}, {"accord_reference_ids", engine::Json::array()},
        {"required_traits", engine::Json::array()}, {"extension_points", engine::Json::array()},
        {"platform_bounds", engine::Json::array({"linux:amd64"})}, {"thermal_restriction", "freezing_only"},
        {"sealed_at", "2026-08-20T12:00:00Z"}, {"composition_authority_reference", "ssiag:decision:test"},
        {"sodv_publication_reference", nullptr}, {"named_version_digest", nullptr}};
    finalize(named, "named_version_digest");
    const auto named_result = sav::handle_request(request("named_version_validate", {{"named_version", named}}));
    assert(named_result.at("state") == "valid_immutable_envelope" && named_result.at("seal_authorized") == false);

    engine::Json capsule{{"protocol", "symphony.sav.extension-capsule.v1"},
        {"capsule_id", "savcapsule:thirdparty:module"}, {"namespace", "thirdparty"},
        {"module_id", "thirdparty-module"}, {"version", "1.0.0"}, {"receipt_digest", engine::tagged_sha256("receipt")},
        {"feature_ids", engine::Json::array({"ssfv:thirdparty:module"})}, {"command_ids", engine::Json::array()},
        {"engine_operation_ids", engine::Json::array()}, {"compatible_receptors", engine::Json::array({"symphony.maestro.knowledge-engine.v1"})},
        {"accord_reference_ids", engine::Json::array()}, {"required_traits", engine::Json::array()},
        {"extension_point_id", "savext:symphony:modules"}, {"created_at", "2026-08-20T12:00:00Z"},
        {"canonical", false}, {"capsule_digest", nullptr}};
    finalize(capsule, "capsule_digest");
    const auto capsule_result = sav::handle_request(request("extension_capsule_check", {{"capsule", capsule}}));
    assert(capsule_result.at("package_inspectable") == true && capsule_result.at("integration_ready") == false &&
           capsule_result.at("gaps").size() == 2U && capsule_result.at("invented_grammar") == false);

    engine::Json blueprint{{"protocol", "symphony.sav.installation-blueprint.v1"},
        {"blueprint_id", "savblueprint:symphony:test"}, {"tops_id", "tops:test"},
        {"named_version_digest", named.at("named_version_digest")},
        {"component_requirements", engine::Json::array({requirement("component:a"), requirement("component:b")})},
        {"capsule_digests", engine::Json::array({capsule.at("capsule_digest")})},
        {"forward_edges", engine::Json::array({{{"source", "component:a"}, {"target", "component:b"}, {"kind", "hard_safety"}}})},
        {"reverse_edges", engine::Json::array({{{"source", "component:b"}, {"target", "component:a"}, {"kind", "hard_safety"}}})},
        {"default_receptors", engine::Json::array()}, {"created_at", "2026-08-20T12:00:00Z"},
        {"canonical", false}, {"apply_authorized", false}, {"blueprint_digest", nullptr}};
    finalize(blueprint, "blueprint_digest");
    const auto forward = sav::handle_request(request("installation_blueprint_plan", {
        {"blueprint", blueprint}, {"direction", "forward"},
        {"completed_component_ids", engine::Json::array()}, {"blocked_component_ids", engine::Json::array()}}));
    const auto reverse = sav::handle_request(request("installation_blueprint_plan", {
        {"blueprint", blueprint}, {"direction", "reverse"},
        {"completed_component_ids", engine::Json::array()}, {"blocked_component_ids", engine::Json::array()}}));
    assert(forward.at("ready_component_ids") == engine::Json::array({"component:a"}));
    assert(reverse.at("ready_component_ids") == engine::Json::array({"component:b"}));
    assert(forward.at("dynamic_replanning") == true && forward.at("hard_safety_edges_preserved") == true);
    auto cyclic = blueprint;
    cyclic["forward_edges"].push_back({{"source", "component:b"}, {"target", "component:a"}, {"kind", "semantic_dependency"}});
    cyclic["reverse_edges"].push_back({{"source", "component:a"}, {"target", "component:b"}, {"kind", "semantic_dependency"}});
    finalize(cyclic, "blueprint_digest");
    require_error([&] { static_cast<void>(sav::handle_request(request("installation_blueprint_plan", {
        {"blueprint", cyclic}, {"direction", "forward"}, {"completed_component_ids", engine::Json::array()},
        {"blocked_component_ids", engine::Json::array()}}))); }, "sav.blueprint_cycle");
    auto wrong_named_namespace = named;
    wrong_named_namespace["named_version_id"] = "other:symphony:baseline";
    finalize(wrong_named_namespace, "named_version_digest");
    require_error([&] { static_cast<void>(sav::handle_request(request("named_version_validate", {{"named_version", wrong_named_namespace}}))); }, "sav.named_version_id");
    const auto exact_compatibility = sav::handle_request(request("compatibility", {
        {"reader_versions", engine::Json::array({"v1"})}, {"writer_version", "v1"}}));
    const auto old_reader = sav::handle_request(request("compatibility", {
        {"reader_versions", engine::Json::array({"v0"})}, {"writer_version", "v1"}}));
    const auto new_writer = sav::handle_request(request("compatibility", {
        {"reader_versions", engine::Json::array({"v1"})}, {"writer_version", "v2"}}));
    assert(exact_compatibility.at("compatible") == true && exact_compatibility.at("reason") == "exact_v1_overlap");
    assert(old_reader.at("compatible") == false && old_reader.at("reason") == "no_supported_overlap");
    assert(new_writer.at("compatible") == false && new_writer.at("reason") == "no_supported_overlap");
    assert(exact_compatibility == sav::handle_request(request("compatibility", {
        {"reader_versions", engine::Json::array({"v1"})}, {"writer_version", "v1"}})));
    require_error([&] { static_cast<void>(sav::handle_request(request("compatibility", {
        {"reader_versions", engine::Json::array({"v1", "v0"})}, {"writer_version", "v1"}}))); }, "sav.order");
    bool rejected = false;
    try { static_cast<void>(sav::handle_request(request("apply", {}))); }
    catch (const engine::Error& error) { rejected = error.code() == "operation.unsupported"; }
    assert(rejected);
    std::cout << "SAV engine tests passed\n";
}
