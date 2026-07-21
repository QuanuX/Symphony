#include "artifacts.hpp"
#include <filesystem>
#include <iostream>
#include <array>
#include <algorithm>

namespace fs = std::filesystem;

#include "evidence.hpp"

bool is_authorized_canonical_json(const std::string& relative_path) {
    // Exact, Architect-ratified STAV v1, common SKV, SKVI, and SCLV protocol artifacts. Directory-prefix
    // allowlisting would silently admit unreviewed JSON and is prohibited.
    static const std::array<std::string, 43> authorized_paths = {
        "knowledge/stav/schemas/v1/common.schema.json",
        "knowledge/stav/schemas/v1/candidate.schema.json",
        "knowledge/stav/schemas/v1/event.schema.json",
        "knowledge/stav/schemas/v1/receipt.schema.json",
        "knowledge/stav/schemas/v1/query.schema.json",
        "knowledge/stav/schemas/v1/query-page.schema.json",
        "knowledge/stav/schemas/v1/verification.schema.json",
        "knowledge/stav/schemas/v1/append-authority-config.schema.json",
        "knowledge/stav/schemas/v1/append-authority-status.schema.json",
        "knowledge/stav/schemas/v1/local-request.schema.json",
        "knowledge/stav/schemas/v1/local-response.schema.json",
        "knowledge/stav/fixtures/v1/valid/candidate.json",
        "knowledge/stav/fixtures/v1/valid/event.json",
        "knowledge/stav/fixtures/v1/valid/receipt-rejected.json",
        "knowledge/stav/fixtures/v1/valid/query.json",
        "knowledge/stav/fixtures/v1/valid/query-page.json",
        "knowledge/stav/fixtures/v1/valid/verification.json",
        "knowledge/stav/fixtures/v1/valid/append-authority-config.json",
        "knowledge/stav/fixtures/v1/valid/append-authority-status.json",
        "knowledge/stav/fixtures/v1/valid/local-request-status.json",
        "knowledge/stav/fixtures/v1/valid/local-response-status.json",
        "knowledge/stav/fixtures/v1/invalid/candidate-duplicate-key.json",
        "knowledge/stav/fixtures/v1/invalid/candidate-null.json",
        "knowledge/stav/fixtures/v1/invalid/query-float.json",
        "knowledge/stav/fixtures/v1/invalid/query-unsafe-integer.json",
        "knowledge/stav/fixtures/v1/invalid/query-unknown-field.json",
        "knowledge/stav/fixtures/v1/invalid/local-request-multiple-payloads.json",
        "knowledge/stav/fixtures/v1/invalid/local-response-wrong-payload.json",
        "knowledge/schemas/v1/engine-process-request.schema.json",
        "knowledge/schemas/v1/engine-process-response.schema.json",
        "knowledge/schemas/v1/engine-descriptor.schema.json",
        "knowledge/schemas/v1/install-receipt.schema.json",
        "knowledge/schemas/v1/proposal.schema.json",
        "knowledge/schemas/v1/provider-evidence.schema.json",
        "knowledge/skvi/schemas/v1/entry.schema.json",
        "knowledge/skvi/schemas/v1/operation-payload.schema.json",
        "knowledge/skvi/schemas/v1/check-result.schema.json",
        "knowledge/skvi/schemas/v1/projection.schema.json",
        "knowledge/sclv/schemas/v3/record.schema.json",
        "knowledge/sclv/schemas/v3/proposal-input.schema.json",
        "knowledge/sclv/schemas/v3/recovery-input.schema.json",
        "knowledge/sclv/schemas/v3/check-result.schema.json",
        "knowledge/sclv/schemas/v3/projection.schema.json"
    };
    return std::find(authorized_paths.begin(), authorized_paths.end(), relative_path) != authorized_paths.end();
}

bool check_path_absence(const fs::path& repo_root, const std::string& relative_path, const std::string& reason, ArtifactCheckResult& result) {
    fs::path p = repo_root / relative_path;
    if (fs::exists(p)) {
        result.success = false;
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "artifact.unauthorized", "path=" + relative_path + " reason=" + reason));
        return false;
    } else {
        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "artifact.absent", "path=" + relative_path));
        return true;
    }
}

ArtifactCheckResult check_unauthorized_artifacts(const std::string& repo_root) {
    ArtifactCheckResult result;
    result.success = true;
    fs::path root(repo_root);

    // A. Publication/docs artifacts
    const std::array<std::string, 4> pub_paths = {
        "docs", "mint.json", "mintlify.json", ".mintlify"
    };
    for (const auto& path : pub_paths) {
        check_path_absence(root, path, "publication_not_authorized", result);
    }

    // B. Generated projection directories
    const std::array<std::string, 12> gen_paths = {
        "generated", "projections",
        "knowledge/generated", "knowledge/projections",
        "knowledge/skvi/generated", "knowledge/skvi/projections",
        "knowledge/sclv/generated", "knowledge/sclv/projections",
        "knowledge/sodv/generated", "knowledge/sodv/projections",
        "tools/symphony-validator/generated", "tools/symphony-validator/projections"
    };
    for (const auto& path : gen_paths) {
        check_path_absence(root, path, "generated_projection_not_authorized", result);
    }

    // C. Projection files under knowledge/
    const std::array<std::string, 10> proj_exts = {
        ".json", ".jsonl", ".duckdb", ".db", ".sqlite", ".h5", ".hdf5", ".graphml", ".gexf", ".dot"
    };
    
    fs::path knowledge_dir = root / "knowledge";
    bool found_projection = false;
    if (fs::exists(knowledge_dir) && fs::is_directory(knowledge_dir)) {
        for (auto const& dir_entry : fs::recursive_directory_iterator(knowledge_dir)) {
            if (dir_entry.is_regular_file()) {
                std::string ext = dir_entry.path().extension().string();
                for (const auto& proj_ext : proj_exts) {
                    if (ext == proj_ext) {
                        std::string rel_path = fs::relative(dir_entry.path(), root).string();
                        if (ext == ".json" && is_authorized_canonical_json(rel_path)) {
                            const std::string authority = rel_path.starts_with("knowledge/stav/")
                                ? "knowledge/stav/SPEC.md"
                                : rel_path.starts_with("knowledge/skvi/")
                                    ? "knowledge/skvi/SPEC.md"
                                    : rel_path.starts_with("knowledge/sclv/")
                                        ? "knowledge/sclv/SPEC.md"
                                    : "knowledge/SPEC.md";
                            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "artifact.canonical_json_authorized", "path=" + rel_path + " authority=" + authority));
                            break;
                        }
                        found_projection = true;
                        result.success = false;
                        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "artifact.unauthorized", "path=" + rel_path + " reason=projection_file_not_authorized"));
                        break;
                    }
                }
            }
        }
    }
    
    if (!found_projection) {
        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "artifact.projection_files_absent", "root=knowledge"));
    }

    // D. qxctl integration surfaces
    const std::array<std::string, 7> qxctl_paths = {
        "tools/symphony-validator/qxctl",
        "tools/symphony-validator/qxctl.cpp",
        "tools/symphony-validator/qxctl.hpp",
        "tools/symphony-validator/src/qxctl.cpp",
        "tools/symphony-validator/src/qxctl.hpp",
        "tools/symphony-validator/src/qxctl_integration.cpp",
        "tools/symphony-validator/src/qxctl_integration.hpp"
    };
    for (const auto& path : qxctl_paths) {
        check_path_absence(root, path, "qxctl_integration_not_authorized", result);
    }

    // E. Schema/template artifacts
    const std::array<std::string, 6> schema_paths = {
        "schemas", "schema", "templates",
        "tools/symphony-validator/schemas",
        "tools/symphony-validator/schema",
        "tools/symphony-validator/templates"
    };
    for (const auto& path : schema_paths) {
        check_path_absence(root, path, "schema_template_not_authorized", result);
    }

    return result;
}
