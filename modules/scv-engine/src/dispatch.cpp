#include "scv.hpp"
#include "knowledge.hpp"
#include "corpus.hpp"
#include "coverage.hpp"
#include "interpretation.hpp"
#include "pack.hpp"
#include "composition.hpp"

#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/operation.hpp"

#include <algorithm>
#include <string>
#include <vector>

namespace symphony::knowledge::scv {
namespace {
using Handler = Json (*)(const engine::Request&, const std::string&);
struct Operation {
    const char* name;
    const char* input;
    const char* output;
    std::vector<std::string> interactions;
    const char* mutability;
    bool expected_state;
    Handler handler;
};
Json inspect(const engine::Request& request, const std::string& domain) {
    if (!request.payload.is_object() || !request.payload.empty())
        throw engine::Error("scv.fields", "inspect requires an empty object", 3);
    return descriptor(domain);
}
#include "interface.generated.inc"
std::vector<engine::OperationSpec> specs(const std::string& domain) {
    std::vector<engine::OperationSpec> output;
    for (const auto& op : registry()) {
        auto suffix = std::string(op.name);
        std::replace(suffix.begin(), suffix.end(), '_', '.');
        output.push_back(engine::OperationSpec{"engop:symphony:" + domain + "." + suffix, op.name,
            "implemented", false, true, {"ssfv:symphony:" + domain + "-engine"}, op.interactions,
            "qxctl_required", op.input, op.output,
            op.mutability, "idempotent", op.expected_state, "none", "", "supported", "freezing"});
    }
    engine::validate_operation_specs(output);
    return output;
}
}
Json descriptor(const std::string& domain) {
    const auto operations = specs(domain);
    return sealed(Json{{"protocol", engine::descriptor_protocol_v2}, {"format_version", 2},
        {"module_id", domain + "-engine"}, {"engine_id", "symphony-" + domain}, {"vector_id", domain},
        {"engine_version", version}, {"process_protocols", Json::array({engine::process_protocol_v1})},
        {"contract_versions", interface_contract_versions()},
        {"operations", engine::administration_operation_descriptors(operations)},
        {"limits", Json{{"request_bytes", engine::Limits::max_request_bytes}, {"response_bytes", engine::Limits::max_response_bytes},
            {"json_depth", engine::Limits::max_json_depth}, {"json_values", engine::Limits::max_json_values},
            {"path_bytes", engine::Limits::max_path_bytes}, {"snapshot_files", engine::Limits::max_snapshot_files},
            {"snapshot_file_bytes", engine::Limits::max_snapshot_file_bytes}, {"deadline_ahead_ms", engine::Limits::max_deadline_ahead_ms}}},
        {"supported_scopes", Json::array({"user", "tops"})}, {"language", "C++26"}, {"thermal_path", "freezing"},
        {"canonical_apply_enabled", false}, {"session_mutation_enabled", false}, {"network_listener", false}}, "descriptor_digest");
}
Json handle_request(const engine::Request& request, const std::string& domain) {
    const auto& entries = registry();
    auto found = std::find_if(entries.begin(), entries.end(), [&](const auto& op) { return request.operation == op.name; });
    if (found == entries.end()) throw engine::Error("operation.unsupported", "operation is unsupported", 4);
    if (engine::unix_time_ms() > request.deadline_unix_ms) throw engine::Error("request.deadline", "request deadline exceeded", 4);
    return found->handler(request, domain);
}
}
