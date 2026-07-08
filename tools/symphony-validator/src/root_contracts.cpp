#include "root_contracts.hpp"
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

RootContractShapeResult check_root_contract_shapes(const std::string& repo_root) {
    RootContractShapeResult result;
    result.success = true;
    fs::path root(repo_root);

    std::vector<ContractFileTarget> targets = {
        {
            "README.md",
            {
                {"Identity", "## Identity"},
                {"Architecture", "## Architecture"},
                {"First_Runtime_Set", "## First Runtime Set"},
                {"Root-Level_Governance_Role", "## Root-Level Governance Role"},
                {"Doctrine", "## Doctrine"},
                {"Python_Doctrine", "## Python Doctrine"}
            }
        },
        {
            "INTENT.md",
            {
                {"Root_Intent", "# QuanuX Symphony Intent"},
                {"Purpose", "## Purpose"},
                {"Scope", "## Scope"},
                {"Non-Scope", "## Non-Scope"},
                {"Relationship_to_Modules", "## Relationship to Modules"},
                {"Relationships", "## Relationships"},
                {"Relationship_to_First_Runtime_Set", "## Relationship to First Runtime Set"},
                {"Installability_Expectations", "## Installability Expectations"},
                {"Owner_Ratification_Boundaries", "## Owner Ratification Boundaries"}
            }
        }
    };

    for (const auto& file_target : targets) {
        fs::path p = root / file_target.path;
        if (!fs::exists(p)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "root_contract.unreadable", "path=" + file_target.path));
            continue;
        }

        std::ifstream file(p);
        if (!file.is_open()) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "root_contract.unreadable", "path=" + file_target.path));
            continue;
        }

        std::string content((std::istreambuf_iterator<char>(file)), std::istreambuf_iterator<char>());

        for (const auto& anchor : file_target.anchors) {
            if (content.find(anchor.search_text) != std::string::npos) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "root_contract.anchor_present", "path=" + file_target.path + " anchor=" + anchor.identifier));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "root_contract.anchor_missing", "path=" + file_target.path + " anchor=" + anchor.identifier));
            }
        }
    }

    return result;
}
