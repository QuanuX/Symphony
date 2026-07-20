#include "validator_contracts.hpp"
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

ValidatorContractShapeResult check_validator_contract_shapes(const std::string& repo_root) {
    ValidatorContractShapeResult result;
    result.success = true;
    fs::path root(repo_root);

    std::vector<ContractFileTarget> targets = {
        {
            "tools/symphony-validator/INTENT.md",
            {
                {"Module_Intent", "# Symphony Validator Intent"},
                {"Purpose", "## Purpose"},
                {"Scope", "## Scope"},
                {"Non-Goals", "## Non-scope"},
                {"Authority", "## Non-authorization Statement"}
            }
        },
        {
            "tools/symphony-validator/MANIFEST.md",
            {
                {"Manifest", "# Symphony Validator Manifest"},
                {"Identity", "## Tool Identity"},
                {"Contract", "## Contract Files"},
                {"Boundaries", "## Non-goals"},
                {"Installability", "## Installability"}
            }
        },
        {
            "tools/symphony-validator/INSTALL.md",
            {
                {"Install", "# Symphony Validator Installation"},
                {"Requirements", "## Requirements"},
                {"Build", "## macOS Build Instructions"},
                {"Test", "## Smoke Tests"},
                {"C++26", "C++26"}
            }
        },
        {
            "tools/symphony-validator/SKILL.md",
            {
                {"Skill", "# Symphony Validator Skill"},
                {"Purpose", "## Skill Purpose"},
                {"Inputs", "## Planned Skill Surface"},
                {"Outputs", "## Output Consumption Behavior"},
                {"Boundaries", "## Deterministic, Non-Autonomous Behavior"}
            }
        },
        {
            "tools/symphony-validator/SPEC.md",
            {
                {"Specification", "# Symphony Validator Specification"},
                {"Parser_/_Checker_/_Projector_Contract", "## Parser / Checker / Projector Contract"},
                {"Parser_Boundary", "### Parser Boundary"},
                {"Checker_Boundary", "### Checker Boundary"},
                {"Projector_Boundary", "### Projector Boundary"},
                {"Evidence_Categories", "### Evidence Categories"},
                {"Authority_Boundaries", "### Authority Boundaries"},
                {"Explicit_Non-Authorizations", "### Explicit Non-Authorizations"}
            }
        }
    };

    for (const auto& file_target : targets) {
        fs::path p = root / file_target.path;
        if (!fs::exists(p)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_contract.unreadable", "path=" + file_target.path));
            continue;
        }

        std::ifstream file(p);
        if (!file.is_open()) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_contract.unreadable", "path=" + file_target.path));
            continue;
        }

        std::string content((std::istreambuf_iterator<char>(file)), std::istreambuf_iterator<char>());

        for (const auto& anchor : file_target.anchors) {
            if (content.find(anchor.search_text) != std::string::npos) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "validator_contract.anchor_present", "path=" + file_target.path + " anchor=" + anchor.identifier));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "validator_contract.anchor_missing", "path=" + file_target.path + " anchor=" + anchor.identifier));
            }
        }
    }

    return result;
}
