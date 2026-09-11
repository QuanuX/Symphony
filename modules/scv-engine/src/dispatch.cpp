#include "scv.hpp"
#include "knowledge.hpp"
#include "corpus.hpp"
#include "interpretation.hpp"

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
const std::vector<Operation>& registry() {
    static const std::vector<Operation> operations{
        {"inspect", engine::descriptor_protocol_v2, {"inspect"}, "read_only", false, inspect},
        {"provider_onboard", "symphony.scv.provider.v1", {"discover"}, "proposal_only", false, handle_source},
        {"source_plan", "symphony.scv.source-plan.v1", {"propose"}, "proposal_only", true, handle_source},
        {"source_apply", "symphony.scv.source-transition.v1", {"apply", "recover"}, "proposal_only", true, handle_source},
        {"source_status", "symphony.scv.source-status.v1", {"inspect"}, "read_only", false, handle_source},
        {"capture_import", "symphony.scv.capture.v1", {"invoke"}, "evidence_only", false, handle_source},
        {"capture_compare", "symphony.scv.capture-diff.v1", {"validate"}, "evidence_only", false, handle_source},
        {"knowledge_interpret", "symphony.scv.knowledge.v1", {"invoke"}, "evidence_only", false, handle_knowledge},
        {"graph_build", "symphony.scv.graph.v1", {"invoke"}, "evidence_only", false, handle_knowledge},
        {"graph_query", "symphony.scv.query-result.v1", {"query"}, "read_only", false, handle_knowledge},
        {"graph_diff", "symphony.scv.diff-result.v1", {"validate"}, "evidence_only", false, handle_knowledge},
        {"graph_explain", "symphony.scv.explain-result.v1", {"query"}, "read_only", false, handle_knowledge},
        {"graph_evaluate", "symphony.scv.evaluate-result.v1", {"validate"}, "evidence_only", false, handle_knowledge},
        {"capture_index", "symphony.scv.capture-index.v1", {"invoke"}, "evidence_only", false, handle_corpus},
        {"corpus_build", "symphony.scv.corpus.v1", {"invoke"}, "evidence_only", false, handle_corpus},
        {"corpus_query", "symphony.scv.corpus-query.v1", {"query"}, "read_only", false, handle_corpus},
        {"corpus_diff", "symphony.scv.corpus-diff.v1", {"validate"}, "evidence_only", false, handle_corpus},
        {"provider_interpret", "symphony.scv.provider-interpretation.v1", {"invoke"}, "evidence_only", false, handle_interpretation},
        {"connection_evaluate", "symphony.scv.connection-evaluation.v1", {"validate"}, "evidence_only", false, handle_interpretation},
        {"connection_reassess", "symphony.scv.connection-reassessment.v1", {"validate"}, "evidence_only", false, handle_interpretation},
        {"profile_prepare", "symphony.scv.interpretation-profile.v1", {"propose"}, "proposal_only", false, handle_interpretation}
    };
    return operations;
}
std::vector<engine::OperationSpec> specs(const std::string& domain) {
    std::vector<engine::OperationSpec> output;
    for (const auto& op : registry()) {
        auto suffix = std::string(op.name);
        std::replace(suffix.begin(), suffix.end(), '_', '.');
        auto protocol_name = std::string(op.name);
        std::replace(protocol_name.begin(), protocol_name.end(), '_', '-');
        output.push_back(engine::OperationSpec{"engop:symphony:" + domain + "." + suffix, op.name,
            "implemented", false, true, {"ssfv:symphony:" + domain + "-engine"}, op.interactions,
            "qxctl_required", "symphony.scv." + protocol_name + "-input.v1", op.output,
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
        {"contract_versions", Json::array({"knowledge/SPEC.md@v1", "knowledge/scv/SOURCE-KNOWLEDGE.md@v1", "knowledge/scv/CORPUS.md@v1", "knowledge/scv/INTERPRETATION.md@v1", "knowledge/scv/AGENT-WORKFLOWS.md@v1"})},
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
