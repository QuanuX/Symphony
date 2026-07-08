#include "doctrine_vocab.hpp"
#include <fstream>
#include <sstream>
#include <regex>
#include <vector>
#include <string>
#include <algorithm>

namespace {
    const std::vector<std::string> CANONICAL_FILES = {
        "README.md",
        "INTENT.md",
        "modules/node-troll/INTENT.md",
        "modules/node-troll/MANIFEST.md",
        "modules/node-troll/INSTALL.md",
        "modules/node-troll/SKILL.md",
        "modules/bus-troll/INTENT.md",
        "modules/bus-troll/MANIFEST.md",
        "modules/bus-troll/INSTALL.md",
        "modules/bus-troll/SKILL.md",
        "modules/hotpath-runtime/INTENT.md",
        "modules/hotpath-runtime/MANIFEST.md",
        "modules/hotpath-runtime/INSTALL.md",
        "modules/hotpath-runtime/SKILL.md",
        "knowledge/INTENT.md",
        "knowledge/skvi/INTENT.md",
        "knowledge/skvi/MANIFEST.md",
        "knowledge/skvi/SKILL.md",
        "knowledge/skvi/SPEC.md",
        "knowledge/skvi/INDEX.md",
        "knowledge/sclv/INTENT.md",
        "knowledge/sclv/MANIFEST.md",
        "knowledge/sclv/SKILL.md",
        "knowledge/sclv/SPEC.md",
        "knowledge/sclv/CHANGELOG.md",
        "knowledge/sodv/INTENT.md",
        "knowledge/sodv/MANIFEST.md",
        "knowledge/sodv/SKILL.md",
        "knowledge/sodv/SPEC.md",
        "tools/symphony-validator/INTENT.md",
        "tools/symphony-validator/MANIFEST.md",
        "tools/symphony-validator/INSTALL.md",
        "tools/symphony-validator/SKILL.md",
        "tools/symphony-validator/SPEC.md"
    };

    const std::vector<std::string> STALE_NAMESPACES = {
        "execution-node",
        "native-execution",
        "bus-agent"
    };

    const std::string FORBIDDEN_ACTIVE_TERM = "core";

    const std::vector<std::string> REJECTED_TRUTH = {
        "Markdown always wins"
    };

    const std::vector<std::string> PROHIBITED_RUNTIME = {
        "contract seeds enforce runtime behavior",
        "contract seed enforces runtime behavior",
        "seeds enforce runtime behavior"
    };

    std::string normalize_phrase(const std::string& phrase) {
        std::string result = phrase;
        std::replace(result.begin(), result.end(), ' ', '_');
        return result;
    }
}

void check_doctrine_vocabulary(const std::string& repo_path, std::vector<std::string>& evidence) {
    std::regex core_regex("\\bcore\\b");

    for (const auto& rel_path : CANONICAL_FILES) {
        std::string full_path = repo_path + "/" + rel_path;
        std::ifstream file(full_path);
        
        if (!file.is_open()) {
            evidence.push_back(format_evidence(EvidenceCategory::Violation, "doctrine_vocab.unreadable", "path=" + rel_path));
            continue;
        }

        std::stringstream buffer;
        buffer << file.rdbuf();
        std::string content = buffer.str();

        bool has_violation = false;

        // Check stale namespaces
        for (const auto& term : STALE_NAMESPACES) {
            if (content.find(term) != std::string::npos) {
                evidence.push_back(format_evidence(EvidenceCategory::Violation, "doctrine_vocab.stale_namespace", "path=" + rel_path + " term=" + term));
                has_violation = true;
            }
        }

        // Check forbidden active term
        if (std::regex_search(content, core_regex)) {
            evidence.push_back(format_evidence(EvidenceCategory::Violation, "doctrine_vocab.forbidden_active_term", "path=" + rel_path + " term=core"));
            has_violation = true;
        }

        // Check rejected truth hierarchy
        for (const auto& phrase : REJECTED_TRUTH) {
            if (content.find(phrase) != std::string::npos) {
                evidence.push_back(format_evidence(EvidenceCategory::Violation, "doctrine_vocab.rejected_truth_hierarchy", "path=" + rel_path + " phrase=" + normalize_phrase(phrase)));
                has_violation = true;
            }
        }

        // Check prohibited runtime enforcement wording
        for (const auto& phrase : PROHIBITED_RUNTIME) {
            if (content.find(phrase) != std::string::npos) {
                evidence.push_back(format_evidence(EvidenceCategory::Violation, "doctrine_vocab.prohibited_runtime_enforcement_wording", "path=" + rel_path + " phrase=" + normalize_phrase(phrase)));
                has_violation = true;
            }
        }

        if (!has_violation) {
            evidence.push_back(format_evidence(EvidenceCategory::Pass, "doctrine_vocab.clean", "path=" + rel_path));
        }
    }
}
