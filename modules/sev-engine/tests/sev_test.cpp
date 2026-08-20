#include "sev.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <cassert>
#include <iostream>

namespace sev = symphony::knowledge::sev;
namespace engine = symphony::knowledge::engine;

namespace {
engine::Request request(std::string operation, engine::Json payload) {
    return {"test", "test", std::move(operation), sev::engine_id, 1, std::move(payload)};
}
engine::Json current() {
    return {{"snapshot_digest", "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
        {"coverage_state", "complete"}, {"sources", engine::Json::array()}};
}
}

int main() {
    const auto descriptor = sev::descriptor();
    assert(descriptor.at("protocol") == engine::descriptor_protocol_v2);
    assert(descriptor.at("separate_scsev_registry") == false);
    engine::Json input{{"protocol", "symphony.sev.case-open-input.v1"},
        {"case_id", "sevcase:symphony:test"}, {"case_kind", "planned_change"},
        {"source_current", current()}, {"target", {{"version", "v2"}}},
        {"created_at", "2026-08-20T12:00:00Z"}};
    const auto opened = sev::handle_request(request("case_open", input));
    assert(opened.at("state") == "open");
    assert(opened.at("canonical") == false);
    const auto status = sev::handle_request(request("case_status", {{"case", opened}}));
    assert(status.at("case_id") == "sevcase:symphony:test");
    bool rejected = false;
    try { static_cast<void>(sev::handle_request(request("apply", {}))); }
    catch (const engine::Error& error) { rejected = error.code() == "operation.unsupported"; }
    assert(rejected);
    std::cout << "SEV engine tests passed\n";
}
