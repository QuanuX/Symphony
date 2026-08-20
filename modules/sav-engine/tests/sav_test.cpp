#include "sav.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"

#include <cassert>
#include <iostream>

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

}

int main() {
    const auto descriptor = sav::descriptor();
    assert(descriptor.at("protocol") == engine::descriptor_protocol_v2);
    assert(descriptor.at("canonical_apply_enabled") == false);
    const auto current = sav::handle_request(request("current_resolve", current_input()));
    assert(current.at("coverage_state") == "complete");
    assert(current.at("canonical") == false);
    assert(current.at("snapshot_digest").get<std::string>().starts_with("sha256:"));
    auto absent = current_input();
    absent["sources"] = engine::Json::array();
    const auto unknown = sav::handle_request(request("current_resolve", absent));
    assert(unknown.at("coverage_state") == "unknown");
    bool rejected = false;
    try { static_cast<void>(sav::handle_request(request("apply", {}))); }
    catch (const engine::Error& error) { rejected = error.code() == "operation.unsupported"; }
    assert(rejected);
    std::cout << "SAV engine tests passed\n";
}
