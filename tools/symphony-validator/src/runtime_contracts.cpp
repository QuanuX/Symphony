#include "runtime_contracts.hpp"
#include "evidence.hpp"
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

namespace fs = std::filesystem;

struct AnchorTarget {
    std::string identifier;
    std::string search_text;
};

struct ContractFileTarget {
    std::string path;
    std::vector<AnchorTarget> anchors;
};

RuntimeContractShapeResult check_runtime_contract_shapes(const std::string& repo_root) {
    RuntimeContractShapeResult result;
    result.success = true;
    fs::path root(repo_root);

    std::vector<std::string> modules = {
        "node-troll",
        "bus-troll",
        "hotpath-runtime"
    };

    std::vector<ContractFileTarget> file_templates = {
        {
            "INTENT.md",
            {
                {"Intent", "Intent"},
                {"Identity", "## Identity"},
                {"Doctrine", "Doctrine"},
                {"Role", "## Role"},
                {"Purpose", "## Purpose"}
            }
        },
        {
            "MANIFEST.md",
            {
                {"Manifest", "Manifest"},
                {"Identity", "## Module Identity"},
                {"Doctrine", "Doctrine"},
                {"Scope", "## Scope"},
                {"Non-Scope", "## Non-Scope"},
                {"Installability", "## Installability"}
            }
        },
        {
            "INSTALL.md",
            {
                {"Install", "Install"},
                {"Install_Status", "## Install Status"},
                {"Install_Scope", "## Install Scope"},
                {"Installation_Modes", "## Supported Installation Modes"},
                {"Non-Requirements", "## Explicit Non-Requirements"}
            }
        },
        {
            "SKILL.md",
            {
                {"Skill", "Skill"},
                {"Purpose", "## Purpose"}
            }
        }
    };

    std::vector<ContractFileTarget> targets;
    for (const auto& mod : modules) {
        for (const auto& tmpl : file_templates) {
            targets.push_back({
                "modules/" + mod + "/" + tmpl.path,
                tmpl.anchors
            });
        }
    }

    if (fs::exists(root / "modules/ssfv-engine")) {
        const std::vector<ContractFileTarget> ssfv_targets = {
            {"modules/ssfv-engine/INTENT.md", {
                {"Intent", "Intent"}, {"Purpose", "## Purpose"},
                {"Authority", "## Authority Boundary"}, {"Non_Goals", "## Non-Goals"}}},
            {"modules/ssfv-engine/MANIFEST.md", {
                {"Manifest", "Manifest"}, {"Identity", "## Identity"},
                {"Contract", "## Contract"}, {"Installation", "## Installation"},
                {"Boundaries", "## Boundaries"}}},
            {"modules/ssfv-engine/INSTALL.md", {
                {"Install", "Installation"}, {"Requirements", "## Requirements"},
                {"Build", "## Build, Test, and Install"}, {"Lifecycle", "## Lifecycle"}}},
            {"modules/ssfv-engine/SKILL.md", {
                {"Skill", "Skill"}, {"Purpose", "## Purpose"},
                {"Procedure", "## Procedure"}, {"Boundaries", "## Boundaries"}}},
            {"modules/ssfv-engine/SPEC.md", {
                {"Specification", "Specification"}, {"Status", "## Status"},
                {"Process", "## Process Contract"}, {"Operations", "## Operations"},
                {"Non_Authorization", "## Non-Authorization"}}},
            {"modules/ssfv-engine/CMakeLists.txt", {
                {"Project", "project(SymphonySsfvEngine"}, {"CXX26", "CXX_STANDARD 26"},
                {"Receipt", "install-receipt.json"}, {"Uninstall", "uninstall-ssfv-engine"}}},
            {"tools/qxctl/README.md", {
                {"SSFV_Group", "qxctl ssfv"}, {"SSFV_Check", "ssfv check"},
                {"SSFV_Graph", "ssfv graph"}}},
            {"tools/qxctl/cmd/qxctl/commands.go", {
                {"SSFV_Command", "newSSFVCommand"}, {"SSFV_Group", "\"ssfv\""},
                {"SSFV_Run", "runSSFV"}}},
            {"tools/qxctl/internal/knowledgeengine/client.go", {
                {"SSFV_Invoke", "InvokeSSFV"}, {"SSFV_Module", "ssfv-engine"},
                {"SSFV_Engine", "symphony-ssfv"}}}
        };
        targets.insert(targets.end(), ssfv_targets.begin(), ssfv_targets.end());
    }

    if (fs::exists(root / "modules/maestro")) {
        const std::vector<ContractFileTarget> maestro_targets = {
            {"modules/maestro/INTENT.md", {{"Intent", "Intent"}, {"Purpose", "## Purpose"},
                {"Authority", "## Authority"}, {"Non_Goals", "## Non-Goals"}}},
            {"modules/maestro/MANIFEST.md", {{"Manifest", "Manifest"}, {"Identity", "## Identity"},
                {"Contract", "## Contract"}, {"Installation", "## Installation"}, {"Boundaries", "## Boundaries"}}},
            {"modules/maestro/INSTALL.md", {{"Install", "Install"}, {"Requirements", "## Requirements"},
                {"Build", "## Build and Test"}, {"Uninstall", "## Uninstall"}}},
            {"modules/maestro/SKILL.md", {{"Skill", "Skill"}, {"Purpose", "## Purpose"},
                {"Procedure", "## Procedure"}, {"Boundaries", "## Boundaries"}}},
            {"modules/maestro/SPEC.md", {{"Specification", "Specification"}, {"Status", "## Status"},
                {"Process", "## Process"}, {"Operations", "## Operations"}, {"Authorization", "## Authorization"}}},
            {"modules/maestro/CMakeLists.txt", {{"Project", "project(SymphonyMaestro"},
                {"CXX26", "CXX_STANDARD 26"}, {"Receipt", "install-receipt.json"},
                {"Uninstall", "uninstall-maestro"}}},
            {"tools/qxctl/cmd/qxctl/commands.go", {{"Maestro_Command", "newMaestroCommand"},
                {"Maestro_Group", "\"maestro\""}}},
            {"tools/qxctl/internal/knowledgeengine/client.go", {{"Maestro_Invoke", "InvokeMaestro"},
                {"Maestro_Module", "maestro"}, {"Maestro_Engine", "symphony-maestro"}}},
            {"tools/qxctl/internal/maestroclient/client.go", {{"Maestro_Protocol", "CommandProtocol"},
                {"Maestro_Receptor", "ReceptorKind"}, {"Maestro_Resource", "func Resource"}}}
        };
        targets.insert(targets.end(), maestro_targets.begin(), maestro_targets.end());
    }

    for (const auto& file_target : targets) {
        fs::path p = root / file_target.path;
        if (!fs::exists(p)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "runtime_contract.unreadable", "path=" + file_target.path));
            continue;
        }

        std::ifstream file(p);
        if (!file.is_open()) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "runtime_contract.unreadable", "path=" + file_target.path));
            continue;
        }

        std::string content((std::istreambuf_iterator<char>(file)), std::istreambuf_iterator<char>());

        for (const auto& anchor : file_target.anchors) {
            if (content.find(anchor.search_text) != std::string::npos) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "runtime_contract.anchor_present", "path=" + file_target.path + " anchor=" + anchor.identifier));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "runtime_contract.anchor_missing", "path=" + file_target.path + " anchor=" + anchor.identifier));
            }
        }
    }

    return result;
}
