#include "knowledge_contracts.hpp"
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

KnowledgeContractShapeResult check_knowledge_contract_shapes(const std::string& repo_root) {
    KnowledgeContractShapeResult result;
    result.success = true;
    fs::path root(repo_root);

    std::vector<ContractFileTarget> targets = {
        {
            "knowledge/INTENT.md",
            {
                {"Knowledge", "Symphony Knowledge Vector Intent"},
                {"Purpose", "### Purpose"},
                {"Scope", "### Scope"},
                {"Boundaries", "### Non-scope"},
                {"Authority", "### Truth Hierarchy"}
            }
        },
        {
            "knowledge/skvi/INTENT.md",
            {
                {"Intent", "Intent"},
                {"Purpose", "### Purpose"},
                {"Scope", "### Scope"},
                {"Non-Goals", "### Non-scope"},
                {"Authority", "### Non-authorization Statement"}
            }
        },
        {
            "knowledge/skvi/MANIFEST.md",
            {
                {"Manifest", "Manifest"},
                {"Identity", "Identity"},
                {"Contract", "## Declared Contract Truth Role"},
                {"Boundaries", "## Non-Authorization Statement"},
                {"Installability", "## Installability Considerations"}
            }
        },
        {
            "knowledge/skvi/SKILL.md",
            {
                {"Skill", "Skill"},
                {"Purpose", "## Purpose"},
                {"Boundaries", "## Non-Authorization Statement"}
            }
        },
        {
            "knowledge/skvi/SPEC.md",
            {
                {"Specification", "Specification"},
                {"Purpose", "## Purpose"},
                {"Required", "## Layer 0 Canonical"},
                {"Boundaries", "## Non-Authorization Statement"},
                {"Non-Authorizations", "## Non-Authorization Statement"}
            }
        },
        {
            "knowledge/skvi/INDEX.md",
            {
                {"Symphony Knowledge Vector Index", "Symphony Knowledge Vector Index"},
                {"Purpose", "## Purpose"},
                {"Entry Model", "## Entry Model"},
                {"Relationship", "## Relationship Model"},
                {"Deferred Projections", "## Projection Doctrine"}
            }
        },
        {
            "knowledge/sclv/INTENT.md",
            {
                {"Intent", "Intent"},
                {"Purpose", "### Purpose"},
                {"Scope", "### Scope"},
                {"Non-Goals", "### Non-scope"},
                {"Authority", "### Non-authorization Statement"}
            }
        },
        {
            "knowledge/sclv/MANIFEST.md",
            {
                {"Manifest", "Manifest"},
                {"Identity", "Identity"},
                {"Contract", "## Declared Contract Truth Role"},
                {"Boundaries", "## Non-Authorization Statement"},
                {"Installability", "## Installability Considerations"}
            }
        },
        {
            "knowledge/sclv/SKILL.md",
            {
                {"Skill", "Skill"},
                {"Purpose", "## Purpose"},
                {"Boundaries", "## Non-Authorization Statement"}
            }
        },
        {
            "knowledge/sclv/SPEC.md",
            {
                {"Specification", "Specification"},
                {"Purpose", "## Purpose"},
                {"Required", "## Layer 0 Canonical"},
                {"Boundaries", "## Non-Authorization Statement"},
                {"Non-Authorizations", "## Non-Authorization Statement"}
            }
        },
        {
            "knowledge/sclv/CHANGELOG.md",
            {
                {"Symphony Change Log Vector", "Symphony Change Log Vector Ledger"},
                {"Purpose", "## Source-Truth Doctrine"},
                {"Record Model", "## Record Model"},
                {"Records", "## Canonical Change Records"},
                {"Non-Authorizations", "## Non-Authorized Artifacts"}
            }
        },
        {
            "knowledge/sodv/INTENT.md",
            {
                {"Intent", "Intent"},
                {"Purpose", "### Purpose"},
                {"Scope", "### Scope"},
                {"Non-Goals", "### Non-scope"},
                {"Authority", "### Non-authorization Statement"}
            }
        },
        {
            "knowledge/sodv/MANIFEST.md",
            {
                {"Manifest", "Manifest"},
                {"Identity", "Identity"},
                {"Contract", "## Declared Contract Truth Role"},
                {"Boundaries", "## Non-Authorization Statement"},
                {"Installability", "## Installability Considerations"}
            }
        },
        {
            "knowledge/sodv/SKILL.md",
            {
                {"Skill", "Skill"},
                {"Purpose", "## Purpose"},
                {"Boundaries", "## Non-Authorization Statement"}
            }
        },
        {
            "knowledge/sodv/SPEC.md",
            {
                {"Specification", "Specification"},
                {"Purpose", "## Purpose"},
                {"Required", "## Layer 0 Canonical"},
                {"Boundaries", "## Non-Authorization Statement"},
                {"Non-Authorizations", "## Non-Authorization Statement"}
            }
        }
    };

    if (fs::exists(root / "go.work") || fs::exists(root / "knowledge/sacv")) {
        const std::vector<ContractFileTarget> sacv_targets = {
            {"knowledge/sacv/INTENT.md", {{"Intent", "Intent"}, {"Purpose", "## Purpose"},
                {"Source_Truth", "## Source-Truth Boundary"}, {"Scope", "## Scope"},
                {"Boundaries", "## Non-Authorization Statement"}}},
            {"knowledge/sacv/MANIFEST.md", {{"Manifest", "Manifest"}, {"Identity", "## Identity"},
                {"Contract", "## Declared Contract Truth Role"}, {"Installability", "## Installability Considerations"},
                {"Boundaries", "## Non-Authorization Statement"}}},
            {"knowledge/sacv/SKILL.md", {{"Skill", "Skill"}, {"Purpose", "## Purpose"},
                {"Authority", "## Caller Authority"}, {"Boundaries", "## Non-Authorization Statement"}}},
            {"knowledge/sacv/SPEC.md", {{"Specification", "Specification"}, {"Purpose", "## Purpose"},
                {"Registry", "## Registry Contract"}, {"Operations", "## SACV Engine Operations"},
                {"Boundaries", "## Non-Authorization Statement"}}},
            {"knowledge/sacv/REGISTRY.md", {{"Registry", "Registry"}, {"Purpose", "## Purpose"},
                {"Entry_Model", "## Entry Model"}, {"Entries", "## Canonical Entries"},
                {"Prohibited", "## Prohibited Entries"}}}
        };
        targets.insert(targets.end(), sacv_targets.begin(), sacv_targets.end());
    }

    for (const auto& file_target : targets) {
        fs::path p = root / file_target.path;
        if (!fs::exists(p)) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "knowledge_contract.unreadable", "path=" + file_target.path));
            continue;
        }

        std::ifstream file(p);
        if (!file.is_open()) {
            result.success = false;
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "knowledge_contract.unreadable", "path=" + file_target.path));
            continue;
        }

        std::string content((std::istreambuf_iterator<char>(file)), std::istreambuf_iterator<char>());

        for (const auto& anchor : file_target.anchors) {
            if (content.find(anchor.search_text) != std::string::npos) {
                result.messages.push_back(format_evidence(EvidenceCategory::Pass, "knowledge_contract.anchor_present", "path=" + file_target.path + " anchor=" + anchor.identifier));
            } else {
                result.success = false;
                result.messages.push_back(format_evidence(EvidenceCategory::Violation, "knowledge_contract.anchor_missing", "path=" + file_target.path + " anchor=" + anchor.identifier));
            }
        }
    }

    return result;
}
