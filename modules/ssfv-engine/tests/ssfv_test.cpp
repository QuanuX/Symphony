#include "ssfv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <algorithm>
#include <chrono>
#include <filesystem>
#include <fstream>
#include <functional>
#include <iomanip>
#include <iostream>
#include <sstream>
#include <stdexcept>
#include <string>
#include <unistd.h>
#include <utility>
#include <vector>

namespace fs = std::filesystem;
namespace ssfv = symphony::knowledge::ssfv;
namespace engine = symphony::knowledge::engine;

namespace {

class TemporaryDirectory final {
public:
    TemporaryDirectory() {
        std::string pattern =
            (fs::canonical(fs::temp_directory_path()) / "symphony-ssfv-test-XXXXXX").string();
        pattern.push_back('\0');
        char* result = ::mkdtemp(pattern.data());
        if (result == nullptr) {
            throw std::runtime_error("mkdtemp failed");
        }
        path_ = result;
    }
    ~TemporaryDirectory() {
        std::error_code ignored;
        fs::remove_all(path_, ignored);
    }
    TemporaryDirectory(const TemporaryDirectory&) = delete;
    TemporaryDirectory& operator=(const TemporaryDirectory&) = delete;
    [[nodiscard]] const fs::path& path() const { return path_; }
private:
    fs::path path_;
};

class CurrentDirectory final {
public:
    explicit CurrentDirectory(const fs::path& path)
        : previous_(fs::current_path()) {
        fs::current_path(path);
    }
    ~CurrentDirectory() {
        std::error_code ignored;
        fs::current_path(previous_, ignored);
    }
    CurrentDirectory(const CurrentDirectory&) = delete;
    CurrentDirectory& operator=(const CurrentDirectory&) = delete;
private:
    fs::path previous_;
};

void require(bool condition, const std::string& message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

template <typename Function>
void require_error(Function&& function, const std::string& code) {
    try {
        function();
    } catch (const engine::Error& error) {
        require(error.code() == code, "expected " + code + ", got " + error.code());
        return;
    }
    throw std::runtime_error("expected Error with code " + code);
}

void write_file(const fs::path& path, const std::string& contents) {
    fs::create_directories(path.parent_path());
    std::ofstream output(path, std::ios::binary);
    if (!output.good()) {
        throw std::runtime_error("could not create fixture: " + path.string());
    }
    output << contents;
}

engine::Request request(std::string operation, engine::Json payload) {
    return engine::Request{"request-1", "correlation-1", std::move(operation), ssfv::engine_id,
                           engine::unix_time_ms() + 60000, std::move(payload)};
}

const std::vector<std::string> contract_paths = {
    "knowledge/SPEC.md",
    "knowledge/ssfv/INTENT.md",
    "knowledge/ssfv/MANIFEST.md",
    "knowledge/ssfv/SKILL.md",
    "knowledge/ssfv/SPEC.md",
    "knowledge/ssfv/NAMESPACES.md",
    "knowledge/ssfv/REGISTRY.md",
    "knowledge/ssfv/FEATURE-FILE-FORMAT.md",
    "knowledge/ssfv/schemas/v1/MANIFEST.md",
    "knowledge/ssfv/schemas/v1/check-result.schema.json",
    "knowledge/ssfv/schemas/v1/diff-input.schema.json",
    "knowledge/ssfv/schemas/v1/diff-result.schema.json",
    "knowledge/ssfv/schemas/v1/feature-file.schema.json",
    "knowledge/ssfv/schemas/v1/feature-record.schema.json",
    "knowledge/ssfv/schemas/v1/graph-input.schema.json",
    "knowledge/ssfv/schemas/v1/graph-projection.schema.json",
    "knowledge/ssfv/schemas/v1/namespace-entry.schema.json",
    "knowledge/ssfv/schemas/v1/proposal-input.schema.json",
    "knowledge/ssfv/schemas/v1/registry-entry.schema.json",
    "knowledge/ssfv/schemas/v1/semantic-snapshot.schema.json",
    "knowledge/ssfv/schemas/v2/MANIFEST.md",
    "knowledge/ssfv/schemas/v2/check-input.schema.json",
    "knowledge/ssfv/schemas/v2/check-result.schema.json",
    "knowledge/ssfv/schemas/v2/diff-input.schema.json",
    "knowledge/ssfv/schemas/v2/diff-result.schema.json",
    "knowledge/ssfv/schemas/v2/feature-record.schema.json",
    "knowledge/ssfv/schemas/v2/proposal-input.schema.json",
    "knowledge/ssfv/schemas/v2/registry-entry.schema.json",
    "knowledge/skvi/INDEX.md",
};

engine::Json feature_record(const std::string& feature_id, const std::string& title,
                            const std::string& kind,
                            const engine::Json& parent_feature_id) {
    return engine::Json{
        {"feature_id", feature_id},
        {"record_version", 2},
        {"title", title},
        {"kind", kind},
        {"status", "implemented"},
        {"parent_feature_id", parent_feature_id},
        {"owner_contract", "modules/example/SPEC.md"},
        {"source_scope", "modules/example"},
        {"implementation_paths", engine::Json::array({"modules/example/src/example.cpp"})},
        {"implementation_languages", engine::Json::array({
            engine::Json{{"language", "C++26"}, {"role", "Deterministic fixture execution."}},
        })},
        {"who", "A caller holding effective repository permissions."},
        {"what", "A deterministic semantic fixture capability."},
        {"how", "It binds explicit records to implementation evidence."},
        {"when", "During bounded freezing-path knowledge operations."},
        {"where", "Inside the fixture repository source scope."},
        {"why", "To verify semantic feature contracts without domain inference."},
        {"relationships", engine::Json::array()},
        {"distinctions", engine::Json::array()},
        {"cross_vector_references", engine::Json::array({
            engine::Json{{"vector", "skvi"}, {"applicability", "not_applicable"},
                         {"reference", nullptr}, {"reason", "No cross-vector fixture reference."}},
        })},
        {"evidence", engine::Json::array({"fixture-owner-ratification"})},
        {"non_claims", engine::Json::array({"The engine does not decide feature worthiness."})},
    };
}

std::string feature_file(const std::vector<engine::Json>& records,
                         const std::string& owner_contract = "modules/example/SPEC.md",
                         const std::string& source_scope = "modules/example") {
    engine::Json envelope{
        {"owner_contract", owner_contract},
        {"protocol", "symphony.ssfv.feature-file.v1"},
        {"records", records},
        {"source_scope", source_scope},
    };
    return "# Symphony Semantic Features\n\n"
           "<!-- symphony:ssfv:feature-file:v1:begin -->\n"
           "```json\n" + envelope.dump(2) + "\n```\n"
           "<!-- symphony:ssfv:feature-file:v1:end -->\n";
}

std::string registry_entry(const engine::Json& record,
                           const std::string& feature_file_path = "modules/example/FEATURES.md") {
    const auto parent = record.at("parent_feature_id").is_null()
        ? "none" : record.at("parent_feature_id").get<std::string>();
    return "- feature_id: `" + record.at("feature_id").get<std::string>() + "`\n"
           "- feature_file: `" + feature_file_path + "`\n"
           "- owner_contract: `" + record.at("owner_contract").get<std::string>() + "`\n"
           "- source_scope: `" + record.at("source_scope").get<std::string>() + "`\n"
           "- status: `" + record.at("status").get<std::string>() + "`\n"
           "- parent_feature_id: `" + parent + "`\n"
           "- record_digest: `" + engine::tagged_sha256(record.dump()) + "`\n"
           "- notes: deterministic fixture\n";
}

std::vector<engine::Json> fixture_records() {
    return {
        feature_record("ssfv:symphony:fixture-capability", "Fixture Capability", "capability", nullptr),
        feature_record("ssfv:symphony:fixture-feature", "Fixture Feature", "feature",
                       "ssfv:symphony:fixture-capability"),
        feature_record("ssfv:symphony:fixture-microfeature", "Fixture Microfeature", "microfeature",
                       "ssfv:symphony:fixture-subfeature"),
        feature_record("ssfv:symphony:fixture-subfeature", "Fixture Subfeature", "subfeature",
                       "ssfv:symphony:fixture-feature"),
    };
}

void write_registry(const fs::path& root, const std::vector<engine::Json>& records,
                    const std::string& feature_file_path = "modules/example/FEATURES.md") {
    std::string entries;
    for (const auto& record : records) {
        entries += registry_entry(record, feature_file_path) + "\n";
    }
    write_file(root / "knowledge/ssfv/REGISTRY.md",
        "# Registry\n\n"
        "Inline `## Canonical Entries` references do not select a section.\n\n"
        "## Canonical Entries\n\n" + entries + "## Prohibited Entries\n\nNone.\n");
}

void create_fixture(const fs::path& root, bool with_features = true) {
    for (const auto& path : contract_paths) {
        write_file(root / path, path.ends_with(".json") ? "{}\n" : "fixture\n");
    }
    write_file(root / "knowledge/ssfv/NAMESPACES.md",
        "# Namespace Registry\n\n"
        "Inline reference to `## Canonical Namespace Entries` is not a heading.\n\n"
        "## Canonical Namespace Entries\n\n"
        "- namespace: `symphony`\n"
        "- id_prefix: `ssfv:symphony:`\n"
        "- owner_contract: `knowledge/ssfv/SPEC.md`\n"
        "- scope: deterministic fixture identities\n"
        "- status: `canonical`\n"
        "- evidence: `knowledge/ssfv/INTENT.md`\n"
        "- notes: deterministic fixture\n");
    write_file(root / "modules/example/SPEC.md", "fixture owner contract\n");
    write_file(root / "modules/example/src/example.cpp", "int fixture() { return 1; }\n");

    if (!with_features) {
        write_file(root / "knowledge/ssfv/REGISTRY.md",
            "# Registry\n\n"
            "The literal `None.` beneath `## Canonical Entries` is the empty form.\n\n"
            "## Canonical Entries\n\nNone.\n\n## Prohibited Entries\n\nNone.\n");
        write_file(root / "knowledge/skvi/INDEX.md", "# Index\n");
        return;
    }

    const auto records = fixture_records();
    write_file(root / "modules/example/FEATURES.md", feature_file(records));
    write_registry(root, records);
    write_file(root / "knowledge/skvi/INDEX.md",
               "# Index\n\n- path: `modules/example/FEATURES.md`\n");
}

engine::Json disabled_check() {
    return engine::Json{{"expected_namespace_digest", nullptr},
                        {"expected_registry_digest", nullptr},
                        {"freshness", "disabled"},
                        {"baseline", nullptr}};
}

bool has_finding(const engine::Json& check, const std::string& code) {
    return std::any_of(check.at("evidence").begin(), check.at("evidence").end(),
                       [&](const engine::Json& finding) {
                           return finding.at("code") == code;
                       });
}

void refresh_snapshot_digest(engine::Json& snapshot) {
    snapshot.erase("snapshot_digest");
    snapshot["snapshot_digest"] = engine::tagged_sha256(snapshot.dump());
}

engine::Json proposal_payload(const engine::Json& state, const std::string& type,
                              const engine::Json& record,
                              const engine::Json& expected_feature_digest,
                              const std::vector<std::string>& target_paths,
                              const std::string& feature_file_path) {
    return engine::Json{
        {"repository", engine::Json{
            {"repository_id", "fixture"},
            {"revision", engine::Json{{"scheme", "git"}, {"value", "fixture-revision"}}},
            {"worktree_id", "fixture-worktree"},
            {"tree_digest", "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
        }},
        {"session_ref", nullptr},
        {"context_ref", "fixture-context"},
        {"created_at", "2026-07-29T12:00:00Z"},
        {"expires_at", "2026-07-29T13:00:00Z"},
        {"operation", engine::Json{
            {"type", type},
            {"expected_contract_digest", state.at("contract_snapshot").at("digest")},
            {"expected_namespace_digest", state.at("namespace_registry").at("digest")},
            {"expected_registry_digest", state.at("feature_registry").at("digest")},
            {"expected_feature_digest", expected_feature_digest},
            {"affected_feature_ids", engine::Json::array({record.at("feature_id")})},
            {"target_paths", target_paths},
            {"namespace_entry", nullptr},
            {"feature_file", feature_file_path},
            {"feature_record", record},
            {"registry_notes", "Architect-ratified feature proposal."},
            {"semantic_declaration", engine::Json{
                {"feature_worthiness_ratified", true},
                {"owner_ratified", true},
                {"rationale", "Fixture declaration supplied by the authorized caller."},
                {"evidence", engine::Json::array({"fixture-ratification"})},
            }},
            {"authorization_ref", "fixture-authorization"},
        }},
    };
}

engine::Json check_fixture_records(const std::vector<engine::Json>& records) {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
    write_registry(temporary.path(), records);
    CurrentDirectory current(temporary.path());
    return ssfv::handle_request(request("check", disabled_check()));
}

void test_descriptor_and_actual_repository(const fs::path& repository_root) {
    const auto descriptor = ssfv::descriptor();
    require(descriptor.at("language") == "C++26", "language contract mismatch");
    require(descriptor.at("thermal_path") == "freezing", "thermal-path contract mismatch");
    require(descriptor.at("install_state") == "installed_undocked", "install state mismatch");
    require(descriptor.at("canonical_apply_enabled") == false, "apply enabled");
    require(descriptor.at("session_mutation_enabled") == false, "session mutation enabled");
    require(descriptor.at("network_listener") == false, "network listener enabled");

    CurrentDirectory current(repository_root);
    const auto inspect = ssfv::handle_request(request("inspect", engine::Json::object()));
    require(inspect.at("engine_decides_feature_worthiness") == false,
            "engine claimed feature-worthiness authority");
    const auto check = ssfv::handle_request(request("check", disabled_check()));
    require(check.at("summary").at("state") == "valid", "canonical empty registry invalid");
    require(check.at("coverage_state") == "empty", "empty coverage state mismatch");
    require(check.at("feature_count") == 0U && check.at("feature_file_count") == 0U,
            "unexpected canonical feature records");
    const auto graph = ssfv::handle_request(
        request("graph", engine::Json{{"format", "json"}}));
    require(graph.at("node_count") == 0U && graph.at("edge_count") == 0U,
            "empty graph count mismatch");
    require(graph.at("noncanonical") == true && graph.at("rebuildable") == true,
            "graph authority escalated");
}

void test_valid_hierarchy_and_deterministic_graph() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    CurrentDirectory current(temporary.path());
    const auto check = ssfv::handle_request(request("check", disabled_check()));
    require(check.at("summary").at("state") == "valid", "valid hierarchy rejected");
    require(check.at("coverage_state") == "complete", "coverage state mismatch");
    require(check.at("feature_count") == 4U && check.at("feature_file_count") == 1U,
            "feature counts mismatch");
    const auto first = ssfv::handle_request(
        request("graph", engine::Json{{"format", "json"}}));
    const auto second = ssfv::handle_request(
        request("graph", engine::Json{{"format", "json"}}));
    require(first == second, "graph projection is nondeterministic");
    require(first.at("node_count") == 4U && first.at("edge_count") == 3U,
            "graph topology mismatch");
}

void test_root_scope_and_crosslinks() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path(), false);
    write_file(temporary.path() / "SPEC.md", "root feature owner\n");
    write_file(temporary.path() / "root-feature.cpp", "int root_feature() { return 1; }\n");

    auto capability = feature_record(
        "ssfv:symphony:root-capability", "Root Capability", "capability", nullptr);
    auto feature = feature_record(
        "ssfv:symphony:root-feature", "Root Feature", "feature",
        "ssfv:symphony:root-capability");
    for (auto* record : {&capability, &feature}) {
        (*record)["owner_contract"] = "SPEC.md";
        (*record)["source_scope"] = ".";
        (*record)["implementation_paths"] = engine::Json::array({"root-feature.cpp"});
    }
    capability["relationships"] = engine::Json::array({
        engine::Json{{"type", "enables"},
                     {"target_feature_id", "ssfv:symphony:root-feature"},
                     {"rationale", "The capability enables its explicit feature."}},
    });
    feature["distinctions"] = engine::Json::array({
        engine::Json{{"target_feature_id", "ssfv:symphony:root-capability"},
                     {"distinction", "The feature is an implementation facet, not the root capability."}},
    });
    const std::vector<engine::Json> records = {capability, feature};
    write_file(temporary.path() / "FEATURES.md", feature_file(records, "SPEC.md", "."));
    write_registry(temporary.path(), records, "FEATURES.md");
    write_file(temporary.path() / "knowledge/skvi/INDEX.md",
               "# Index\n\n- path: `FEATURES.md`\n");

    CurrentDirectory current(temporary.path());
    const auto check = ssfv::handle_request(request("check", disabled_check()));
    require(check.at("summary").at("state") == "valid", "valid root scope rejected");
    require(check.at("feature_count") == 2U && check.at("feature_file_count") == 1U,
            "root-scope counts mismatch");
    const auto graph = ssfv::handle_request(
        request("graph", engine::Json{{"format", "json"}}));
    require(graph.at("edge_count") == 3U,
            "primary parent, crosslink, and distinction were not projected");
}

void test_freshness_and_diff() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path());
    CurrentDirectory current(temporary.path());
    const auto baseline_check = ssfv::handle_request(request("check", disabled_check()));
    const auto baseline = baseline_check.at("semantic_snapshot");
    write_file(temporary.path() / "modules/example/src/example.cpp",
               "int fixture() { return 2; }\n");

    const auto report = ssfv::handle_request(request("check", engine::Json{
        {"expected_namespace_digest", nullptr}, {"expected_registry_digest", nullptr},
        {"freshness", "report"}, {"baseline", baseline},
    }));
    require(report.at("summary").at("state") == "valid", "report mode became a hard failure");
    require(report.at("semantic_freshness_state") == "stale", "stale evidence not detected");
    require(report.at("summary").at("warning").get<std::size_t>() > 0U,
            "report mode omitted warning");

    const auto required = ssfv::handle_request(request("check", engine::Json{
        {"expected_namespace_digest", nullptr}, {"expected_registry_digest", nullptr},
        {"freshness", "require"}, {"baseline", baseline},
    }));
    require(required.at("summary").at("state") == "invalid",
            "require mode did not fail unresolved semantic freshness");

    const auto diff = ssfv::handle_request(request("diff", engine::Json{
        {"baseline", baseline},
        {"expected_current_snapshot_digest", nullptr},
        {"scope_feature_ids", engine::Json::array()},
        {"include_semantic_candidates", true},
    }));
    require(diff.at("state") == "review_required", "stale source was not review-required");
    require(diff.at("stale_references").size() == 1U, "stale path count mismatch");
    require(!diff.at("semantic_candidates").empty(), "semantic candidate omitted");
}

void test_snapshot_diff_variants_and_bounds() {
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto first = ssfv::handle_request(request("check", disabled_check()));
        const auto second = ssfv::handle_request(request("check", disabled_check()));
        require(first.at("semantic_snapshot") == second.at("semantic_snapshot"),
                "unchanged inputs produced different semantic snapshots");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        auto records = fixture_records();
        records.at(1)["title"] = "Updated Fixture Feature";
        write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
        write_registry(temporary.path(), records);
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", true},
        }));
        require(diff.at("changed_feature_ids") ==
                    engine::Json::array({"ssfv:symphony:fixture-feature"}),
                "record-only change was not isolated");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto records = fixture_records();
        records.at(0)["cross_vector_references"] = engine::Json::array({
            engine::Json{{"vector", "skvi"}, {"applicability", "applicable"},
                         {"reference", "knowledge/skvi/INDEX.md"},
                         {"reason", "The feature owner file is indexed here."}},
        });
        write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
        write_registry(temporary.path(), records);
        CurrentDirectory current(temporary.path());
        const auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        write_file(temporary.path() / "knowledge/skvi/INDEX.md",
                   "# Index\n\n- path: `modules/example/FEATURES.md`\n\nChanged fixture note.\n");
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", true},
        }));
        require(std::find(diff.at("stale_references").begin(),
                          diff.at("stale_references").end(),
                          engine::Json("knowledge/skvi/INDEX.md")) !=
                    diff.at("stale_references").end(),
                "cross-vector evidence change was not stale");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        write_file(temporary.path() / "modules/example/SPEC.md",
                   "fixture owner contract changed\n");
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", true},
        }));
        require(diff.at("state") == "review_required" &&
                    !diff.at("semantic_candidates").empty(),
                "owner-contract-only change was not review-required");
        const auto required = ssfv::handle_request(request("check", engine::Json{
            {"expected_namespace_digest", nullptr},
            {"expected_registry_digest", nullptr},
            {"freshness", "require"},
            {"baseline", baseline},
        }));
        require(required.at("summary").at("state") == "invalid" &&
                    has_finding(required, "ssfv.semantic_freshness.review_required"),
                "required freshness accepted changed owner evidence");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto initial = fixture_records();
        initial.at(0)["implementation_paths"] =
            engine::Json::array({"modules/example/src/example.cpp",
                                 "modules/example/src/second.cpp"});
        write_file(temporary.path() / "modules/example/src/second.cpp",
                   "int second_fixture() { return 2; }\n");
        write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(initial));
        write_registry(temporary.path(), initial);
        CurrentDirectory current(temporary.path());
        const auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        auto current_records = initial;
        current_records.at(0)["implementation_paths"] =
            engine::Json::array({"modules/example/src/example.cpp"});
        write_file(temporary.path() / "modules/example/FEATURES.md",
                   feature_file(current_records));
        write_registry(temporary.path(), current_records);
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", false},
        }));
        require(diff.at("uncovered_paths") ==
                    engine::Json::array({"modules/example/src/second.cpp"}),
                "removed implementation coverage was not reported");
        require(diff.at("state") == "review_required",
                "removed implementation coverage did not require review");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto records = fixture_records();
        records.at(0)["title"] = "Scoped Capability Change";
        records.at(1)["title"] = "Out-of-Scope Feature Change";
        write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
        write_registry(temporary.path(), records);
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", state.at("semantic_snapshot")},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array({
                "ssfv:symphony:fixture-capability",
            })},
            {"include_semantic_candidates", true},
        }));
        require(diff.at("changed_feature_ids") ==
                    engine::Json::array({"ssfv:symphony:fixture-capability"}),
                "bounded feature scope leaked another change");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", state.at("semantic_snapshot")},
                {"expected_current_snapshot_digest",
                 "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.diff.current_digest_mismatch");
        const auto stale_check = ssfv::handle_request(request("check", engine::Json{
            {"expected_namespace_digest", nullptr},
            {"expected_registry_digest",
             "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
            {"freshness", "disabled"},
            {"baseline", nullptr},
        }));
        require(stale_check.at("summary").at("state") == "invalid" &&
                    has_finding(stale_check, "ssfv.registry.expected_digest_mismatch"),
                "stale expected registry digest accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["contract_digest"] =
            "sha256:0000000000000000000000000000000000000000000000000000000000000000";
        refresh_snapshot_digest(baseline);
        const auto diff = ssfv::handle_request(request("diff", engine::Json{
            {"baseline", baseline},
            {"expected_current_snapshot_digest", nullptr},
            {"scope_feature_ids", engine::Json::array()},
            {"include_semantic_candidates", true},
        }));
        require(diff.at("state") == "review_required" &&
                    !diff.at("semantic_candidates").empty(),
                "baseline contract mismatch omitted review requirement");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["records"] = engine::Json::array();
        for (std::size_t index = 0U; index < 8193U; ++index) {
            baseline["records"].push_back(engine::Json::object());
        }
        refresh_snapshot_digest(baseline);
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", baseline},
                {"expected_current_snapshot_digest", nullptr},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.snapshot.count");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["records"].at(1)["parent_feature_id"] = nullptr;
        refresh_snapshot_digest(baseline);
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", baseline},
                {"expected_current_snapshot_digest", nullptr},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.snapshot.record");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["records"].at(1)["parent_feature_id"] =
            "ssfv:symphony:missing-capability";
        refresh_snapshot_digest(baseline);
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", baseline},
                {"expected_current_snapshot_digest", nullptr},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.snapshot.hierarchy");

        baseline = ssfv::handle_request(
            request("check", disabled_check())).at("semantic_snapshot");
        baseline["feature_files"] = engine::Json::array();
        refresh_snapshot_digest(baseline);
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("diff", engine::Json{
                {"baseline", baseline},
                {"expected_current_snapshot_digest", nullptr},
                {"scope_feature_ids", engine::Json::array()},
                {"include_semantic_candidates", false},
            })));
        }, "ssfv.snapshot.record");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        auto expired = request("check", disabled_check());
        expired.deadline_unix_ms = engine::unix_time_ms() - 1;
        require_error([&] {
            static_cast<void>(ssfv::handle_request(expired));
        }, "request.deadline_expired");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        write_file(temporary.path() / contract_paths.front(),
                   std::string((4U << 20) + 1U, 'x'));
        CurrentDirectory current(temporary.path());
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("check", disabled_check())));
        }, "path.file_too_large");
    }
}

void test_structural_failures_and_no_follow() {
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto contents = feature_file({
            feature_record("ssfv:symphony:fixture-capability", "Fixture Capability",
                           "capability", nullptr),
        });
        contents += "<!-- symphony:ssfv:feature-file:v1:begin -->\n";
        write_file(temporary.path() / "modules/example/FEATURES.md", contents);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "duplicate managed marker accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "modules/example/FEATURES.md");
        write_file(temporary.path() / "outside.md", "unsafe target\n");
        fs::create_symlink(temporary.path() / "outside.md",
                           temporary.path() / "modules/example/FEATURES.md");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "symlinked feature owner accepted");
    }
}

void test_record_graph_and_evidence_failures() {
    {
        auto records = fixture_records();
        records.at(3)["parent_feature_id"] = "ssfv:symphony:fixture-capability";
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.hierarchy.parent_kind"),
                "invalid primary-parent kind accepted");
    }
    {
        auto records = fixture_records();
        records.at(1)["parent_feature_id"] = "ssfv:symphony:missing-capability";
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.hierarchy.parent_missing"),
                "missing primary parent accepted");
    }
    {
        auto records = fixture_records();
        records.at(1)["parent_feature_id"] = "ssfv:symphony:fixture-microfeature";
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.hierarchy.cycle"),
                "primary-parent cycle accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["relationships"] = engine::Json::array({
            engine::Json{{"type", "depends_on"},
                         {"target_feature_id", "ssfv:symphony:missing-target"},
                         {"rationale", "Fixture missing-target relation."}},
        });
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.relationship.target_missing"),
                "missing relationship target accepted");
    }
    {
        auto records = fixture_records();
        const engine::Json relation{
            {"type", "depends_on"},
            {"target_feature_id", "ssfv:symphony:fixture-feature"},
            {"rationale", "Fixture repeated relation."},
        };
        records.at(0)["relationships"] = engine::Json::array({relation, relation});
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.record.duplicate_value"),
                "repeated relationship accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["distinctions"] = engine::Json::array({
            engine::Json{{"target_feature_id", "ssfv:symphony:fixture-feature"},
                         {"distinction", ""}},
        });
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "distinction without factual text accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto registry = registry_entry(fixture_records().at(0));
        const auto digest_at = registry.find("sha256:");
        require(digest_at != std::string::npos, "fixture registry digest absent");
        registry.replace(digest_at, 71U,
                         "sha256:0000000000000000000000000000000000000000000000000000000000000000");
        for (std::size_t index = 1U; index < fixture_records().size(); ++index) {
            registry += "\n" + registry_entry(fixture_records().at(index));
        }
        write_file(temporary.path() / "knowledge/ssfv/REGISTRY.md",
            "# Registry\n\n## Canonical Entries\n\n" + registry +
            "\n## Prohibited Entries\n\nNone.\n");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.registry.record_mismatch"),
                "registry-to-record digest mismatch accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "modules/example/src/example.cpp");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.record.evidence_unreadable"),
                "missing implementation evidence accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "modules/example/src/example.cpp");
        write_file(temporary.path() / "outside.cpp", "int outside() { return 0; }\n");
        fs::create_symlink(temporary.path() / "outside.cpp",
                           temporary.path() / "modules/example/src/example.cpp");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "symlinked implementation evidence accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        fs::remove(temporary.path() / "modules/example/SPEC.md");
        write_file(temporary.path() / "outside-spec.md", "outside owner\n");
        fs::create_symlink(temporary.path() / "outside-spec.md",
                           temporary.path() / "modules/example/SPEC.md");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "symlinked owner contract accepted");
    }
    {
        auto records = fixture_records();
        for (auto& record : records) {
            record["implementation_paths"] = engine::Json::array({"modules/example/src"});
        }
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "non-regular implementation evidence accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["cross_vector_references"] = engine::Json::array({
            engine::Json{{"vector", "skvi"}, {"applicability", "applicable"},
                         {"reference", "knowledge/skvi/MISSING.md"},
                         {"reason", "Fixture applicable reference."}},
        });
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "missing applicable cross-vector reference accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        write_file(temporary.path() / "modules/example/FEATURES.md",
            "# Features\n\n<!-- symphony:ssfv:feature-file:v1:begin -->\n"
            "```json\n{\"protocol\":\n```\n"
            "<!-- symphony:ssfv:feature-file:v1:end -->\n");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.feature_file.json"),
                "malformed embedded JSON accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto contents = feature_file(fixture_records());
        const auto object = contents.find("{\n");
        require(object != std::string::npos, "fixture JSON object absent");
        contents.insert(object + 2U,
                        "  \"protocol\": \"symphony.ssfv.feature-file.v1\",\n");
        write_file(temporary.path() / "modules/example/FEATURES.md", contents);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.feature_file.json"),
                "duplicate embedded JSON field accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto contents = feature_file(fixture_records());
        const auto title = contents.find("Fixture Capability");
        require(title != std::string::npos, "fixture title absent");
        contents.insert(title, 1U, static_cast<char>(0xff));
        write_file(temporary.path() / "modules/example/FEATURES.md", contents);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.feature_file.encoding"),
                "invalid UTF-8 in embedded JSON accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto contents = feature_file(fixture_records());
        const std::string opening =
            "<!-- symphony:ssfv:feature-file:v1:begin -->\n```json\n";
        const std::string closing =
            "\n```\n<!-- symphony:ssfv:feature-file:v1:end -->";
        const auto begin = contents.find(opening);
        const auto end = contents.find(closing);
        require(begin != std::string::npos && end != std::string::npos,
                "fixture managed region absent");
        const auto json_begin = begin + opening.size();
        const auto parsed = engine::Json::parse(
            contents.substr(json_begin, end - json_begin));
        contents.replace(json_begin, end - json_begin, parsed.dump());
        write_file(temporary.path() / "modules/example/FEATURES.md", contents);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid" &&
                    has_finding(check, "ssfv.feature_file.canonical_format"),
                "noncanonical feature-file rendering accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["implementation_paths"] = engine::Json::array({"../outside.cpp"});
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "traversal implementation path accepted");
    }
    {
        auto records = fixture_records();
        records.at(0)["implementation_paths"] =
            engine::Json::array({"modules/example/src/*.cpp"});
        const auto check = check_fixture_records(records);
        require(check.at("summary").at("state") == "invalid",
                "glob implementation path accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        auto registry = registry_entry(fixture_records().at(0));
        const std::string feature_line = "- feature_file: `modules/example/FEATURES.md`\n";
        const std::string owner_line = "- owner_contract: `modules/example/SPEC.md`\n";
        const auto feature_at = registry.find(feature_line);
        const auto owner_at = registry.find(owner_line);
        require(feature_at != std::string::npos && owner_at != std::string::npos,
                "fixture registry fields absent");
        registry.replace(feature_at, feature_line.size() + owner_line.size(),
                         owner_line + feature_line);
        write_file(temporary.path() / "knowledge/ssfv/REGISTRY.md",
                   "# Registry\n\n## Canonical Entries\n\n" + registry +
                   "\n## Prohibited Entries\n\nNone.\n");
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "reordered or incomplete registry block accepted");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        std::string registry =
            "# Registry\n\n## Canonical Entries\n\n" +
            registry_entry(fixture_records().at(0)) +
            "\n## Prohibited Entries\n\nNone.\n";
        const auto notes = registry.find("deterministic fixture");
        require(notes != std::string::npos, "fixture registry notes absent");
        registry.insert(notes, 1U, static_cast<char>(0xff));
        write_file(temporary.path() / "knowledge/ssfv/REGISTRY.md", registry);
        CurrentDirectory current(temporary.path());
        const auto check = ssfv::handle_request(request("check", disabled_check()));
        require(check.at("summary").at("state") == "invalid",
                "invalid UTF-8 in registry accepted");
    }
}

void test_add_proposal_and_reserved_apply() {
    TemporaryDirectory temporary;
    create_fixture(temporary.path(), false);
    CurrentDirectory current(temporary.path());
    const auto state = ssfv::handle_request(request("check", disabled_check()));
    const auto record = feature_record(
        "ssfv:symphony:new-capability", "New Capability", "capability", nullptr);
    const auto payload = proposal_payload(
        state, "add_feature", record, nullptr,
        {"knowledge/ssfv/REGISTRY.md", "knowledge/skvi/INDEX.md",
         "modules/example/FEATURES.md"},
        "modules/example/FEATURES.md");
    const auto first = ssfv::handle_request(request("propose", payload));
    const auto second = ssfv::handle_request(request("propose", payload));
    require(first == second, "proposal output is nondeterministic");
    require(first.at("canonical_apply_enabled") == false, "proposal enabled apply");
    require(first.at("authority").at("engine_decided_domain_truth") == false,
            "proposal escalated engine authority");
    require(first.at("write_set").size() == 3U, "proposal write set mismatch");
    require(fs::exists(temporary.path() / "modules/example/FEATURES.md") == false,
            "proposal mutated canonical state");

    require_error([&] {
        static_cast<void>(ssfv::handle_request(request("apply", engine::Json::object())));
    }, "operation.unsupported");
}

void test_proposal_operation_matrix() {
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path(), false);
        write_file(temporary.path() / "knowledge/ssfv/new-namespace.md",
                   "namespace evidence\n");
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto payload = proposal_payload(
            state, "add_feature",
            feature_record("ssfv:symphony:unused", "Unused", "capability", nullptr),
            nullptr, {"knowledge/ssfv/NAMESPACES.md"}, "modules/example/FEATURES.md");
        auto& operation = payload["operation"];
        operation["type"] = "allocate_namespace";
        operation["affected_feature_ids"] = engine::Json::array();
        operation["target_paths"] = engine::Json::array({"knowledge/ssfv/NAMESPACES.md"});
        operation["expected_feature_digest"] = nullptr;
        operation["namespace_entry"] = engine::Json{
            {"namespace", "enterprise"},
            {"id_prefix", "ssfv:enterprise:"},
            {"owner_contract", "knowledge/ssfv/SPEC.md"},
            {"scope", "Enterprise extension feature identities."},
            {"status", "canonical"},
            {"evidence", engine::Json::array({"knowledge/ssfv/new-namespace.md"})},
            {"notes", "Architect-ratified fixture allocation."},
        };
        operation["feature_file"] = nullptr;
        operation["feature_record"] = nullptr;
        operation["registry_notes"] = nullptr;
        const auto result = ssfv::handle_request(request("propose", payload));
        require(result.at("write_set").size() == 1U &&
                    result.at("operations").at(0).at("type") == "allocate_namespace",
                "namespace allocation proposal shape mismatch");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        const auto record = feature_record(
            "ssfv:symphony:fixture-second-feature", "Second Fixture Feature", "feature",
            "ssfv:symphony:fixture-capability");
        const auto payload = proposal_payload(
            state, "add_feature", record, nullptr,
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        const auto result = ssfv::handle_request(request("propose", payload));
        require(result.at("write_set").size() == 2U,
                "add to existing owner file unexpectedly changed SKVI");
    }
    for (const auto& [type, status] : std::vector<std::pair<std::string, std::string>>{
             {"update_feature", "implemented"},
             {"deprecate_feature", "deprecated"},
             {"retire_feature", "retired"},
         }) {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto record = fixture_records().at(1);
        record["title"] = "Proposed " + type;
        record["status"] = status;
        const auto prior_digest = engine::tagged_sha256(fixture_records().at(1).dump());
        const auto payload = proposal_payload(
            state, type, record, prior_digest,
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        const auto result = ssfv::handle_request(request("propose", payload));
        require(result.at("write_set").size() == 2U,
                type + " proposal write-set mismatch");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        write_file(temporary.path() / "modules/other/SPEC.md", "other owner\n");
        write_file(temporary.path() / "modules/other/src/feature.cpp",
                   "int moved_feature() { return 3; }\n");
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto record = fixture_records().at(1);
        record["owner_contract"] = "modules/other/SPEC.md";
        record["source_scope"] = "modules/other";
        record["implementation_paths"] =
            engine::Json::array({"modules/other/src/feature.cpp"});
        const auto payload = proposal_payload(
            state, "move_feature", record,
            engine::tagged_sha256(fixture_records().at(1).dump()),
            {"knowledge/ssfv/REGISTRY.md", "knowledge/skvi/INDEX.md",
             "modules/example/FEATURES.md", "modules/other/FEATURES.md"},
            "modules/other/FEATURES.md");
        const auto result = ssfv::handle_request(request("propose", payload));
        require(result.at("write_set").size() == 4U,
                "move proposal did not atomically bind old/new owners, registry, and SKVI");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path(), false);
        write_file(temporary.path() / "modules/example/FEATURES.md", "");
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        const auto record = feature_record(
            "ssfv:symphony:new-capability", "New Capability", "capability", nullptr);
        const auto payload = proposal_payload(
            state, "add_feature", record, nullptr,
            {"knowledge/ssfv/REGISTRY.md", "knowledge/skvi/INDEX.md",
             "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "proposal.target_exists_unregistered");
    }
    {
        TemporaryDirectory temporary;
        create_fixture(temporary.path());
        CurrentDirectory current(temporary.path());
        const auto state = ssfv::handle_request(request("check", disabled_check()));
        auto record = fixture_records().at(1);
        record["title"] = "Stale Proposal";
        auto payload = proposal_payload(
            state, "update_feature", record,
            "sha256:0000000000000000000000000000000000000000000000000000000000000000",
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "proposal.expected_feature_mismatch");
        payload = proposal_payload(
            state, "update_feature", record,
            engine::tagged_sha256(fixture_records().at(1).dump()),
            {"README.md"}, "modules/example/FEATURES.md");
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "proposal.target_set_mismatch");
        payload = proposal_payload(
            state, "update_feature", record,
            engine::tagged_sha256(fixture_records().at(1).dump()),
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        payload["operation"]["semantic_declaration"]["evidence"] = engine::Json::array();
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "payload.semantic_declaration");
        payload = proposal_payload(
            state, "update_feature", record,
            engine::tagged_sha256(fixture_records().at(1).dump()),
            {"knowledge/ssfv/REGISTRY.md", "modules/example/FEATURES.md"},
            "modules/example/FEATURES.md");
        payload["operation"]["feature_record"]["implementation_paths"] =
            engine::Json::array({"../unsafe.cpp"});
        require_error([&] {
            static_cast<void>(ssfv::handle_request(request("propose", payload)));
        }, "ssfv.record.implementation_path");
    }
}

void measure_empty_and_populated_operations(const fs::path& repository_root) {
    const auto measure = [](const std::function<void()>& operation) {
        const auto started = std::chrono::steady_clock::now();
        operation();
        return std::chrono::duration_cast<std::chrono::microseconds>(
            std::chrono::steady_clock::now() - started).count();
    };
    std::int64_t empty_check_us = 0;
    std::int64_t empty_graph_us = 0;
    {
        CurrentDirectory current(repository_root);
        empty_check_us = measure([] {
            static_cast<void>(ssfv::handle_request(request("check", disabled_check())));
        });
        empty_graph_us = measure([] {
            static_cast<void>(ssfv::handle_request(
                request("graph", engine::Json{{"format", "json"}})));
        });
    }

    TemporaryDirectory temporary;
    create_fixture(temporary.path(), false);
    std::vector<engine::Json> records;
    records.push_back(feature_record(
        "ssfv:symphony:performance-capability", "Performance Capability",
        "capability", nullptr));
    for (std::size_t index = 0U; index < 256U; ++index) {
        std::ostringstream identifier;
        identifier << "ssfv:symphony:performance-feature-"
                   << std::setw(3) << std::setfill('0') << index;
        records.push_back(feature_record(
            identifier.str(), "Bounded Performance Feature", "feature",
            "ssfv:symphony:performance-capability"));
    }
    write_file(temporary.path() / "modules/example/FEATURES.md", feature_file(records));
    write_registry(temporary.path(), records);
    write_file(temporary.path() / "knowledge/skvi/INDEX.md",
               "# Index\n\n- path: `modules/example/FEATURES.md`\n");
    std::int64_t populated_check_us = 0;
    std::int64_t populated_graph_us = 0;
    {
        CurrentDirectory current(temporary.path());
        populated_check_us = measure([&] {
            const auto result = ssfv::handle_request(request("check", disabled_check()));
            require(result.at("feature_count") == 257U,
                    "bounded performance fixture count mismatch");
        });
        populated_graph_us = measure([&] {
            const auto result = ssfv::handle_request(
                request("graph", engine::Json{{"format", "json"}}));
            require(result.at("node_count") == 257U && result.at("edge_count") == 256U,
                    "bounded performance graph count mismatch");
            static_cast<void>(engine::serialize_response(engine::success_response(
                request("graph", engine::Json{{"format", "json"}}),
                ssfv::engine_id, ssfv::engine_version, result)));
        });
    }
    std::cout << "SSFV performance microseconds"
              << " empty_check=" << empty_check_us
              << " empty_graph=" << empty_graph_us
              << " populated_257_check=" << populated_check_us
              << " populated_257_graph_and_serialize=" << populated_graph_us << '\n';
}

}

int main(int argc, char** argv) {
    try {
        if (argc != 2) {
            throw std::runtime_error("repository root argument is required");
        }
        test_descriptor_and_actual_repository(fs::canonical(argv[1]));
        test_valid_hierarchy_and_deterministic_graph();
        test_root_scope_and_crosslinks();
        test_freshness_and_diff();
        test_snapshot_diff_variants_and_bounds();
        test_structural_failures_and_no_follow();
        test_record_graph_and_evidence_failures();
        test_add_proposal_and_reserved_apply();
        test_proposal_operation_matrix();
        measure_empty_and_populated_operations(fs::canonical(argv[1]));
        std::cout << "SSFV engine tests passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "SSFV engine tests failed: " << error.what() << '\n';
        return 1;
    }
}
