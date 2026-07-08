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
