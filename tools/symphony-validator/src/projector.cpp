#include "projector.hpp"

#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/json.hpp>

#include <algorithm>
#include <cctype>
#include <filesystem>
#include <map>
#include <sstream>
#include <stdexcept>
#include <string_view>

namespace {

namespace fs = std::filesystem;
namespace engine = symphony::knowledge::engine;

constexpr std::string_view evidence_prefix = "evidence ";
constexpr std::string_view validator_id = "symphony-validator";
constexpr std::string_view validator_version = "0.1.0-dev";

bool attribute_key(const std::string& value) {
    if (value.empty() || !std::isalpha(static_cast<unsigned char>(value.front())) || value.size() > 64U) {
        return false;
    }
    return std::all_of(value.begin() + 1, value.end(), [](const char character) {
        const auto byte = static_cast<unsigned char>(character);
        return std::isalnum(byte) || character == '_' || character == '.' || character == '-';
    });
}

std::map<std::string, std::string> attributes(const std::string& detail) {
    std::map<std::string, std::string> result;
    std::istringstream stream(detail);
    std::string token;
    while (stream >> token && result.size() < 32U) {
        const auto separator = token.find('=');
        if (separator == std::string::npos || separator == 0U || separator + 1U >= token.size()) {
            continue;
        }
        const auto key = token.substr(0U, separator);
        const auto value = token.substr(separator + 1U);
        if (attribute_key(key) && value.size() <= 4096U) {
            result.emplace(key, value);
        }
    }
    return result;
}

struct ParsedEvidence {
    std::string category;
    std::string rule_id;
    std::string detail;
};

ParsedEvidence parse_evidence(const std::string& message) {
    if (!message.starts_with(evidence_prefix)) {
        throw std::runtime_error("validator message is not structured evidence");
    }
    const auto category_begin = evidence_prefix.size();
    const auto category_end = message.find(' ', category_begin);
    if (category_end == std::string::npos) {
        throw std::runtime_error("validator evidence category is incomplete");
    }
    const auto rule_end = message.find(' ', category_end + 1U);
    ParsedEvidence result{
        message.substr(category_begin, category_end - category_begin),
        rule_end == std::string::npos
            ? message.substr(category_end + 1U)
            : message.substr(category_end + 1U, rule_end - category_end - 1U),
        rule_end == std::string::npos ? std::string{} : message.substr(rule_end + 1U),
    };
    if (result.rule_id.empty()) {
        throw std::runtime_error("validator evidence rule ID is empty");
    }
    return result;
}

std::string finding_scope(const ParsedEvidence& finding) {
    if (finding.rule_id == "sclv.affected_surface.provenance_summary" ||
        finding.rule_id == "sclv.affected_surface.unindexed") {
        return "historical";
    }
    if (finding.rule_id == "repository.path") {
        return "system";
    }
    return "active";
}

engine::Json finding_json(const ParsedEvidence& finding) {
    const auto values = attributes(finding.detail);
    const auto scope = finding_scope(finding);
    const auto occurrence = engine::tagged_sha256(
        finding.category + "\n" + finding.rule_id + "\n" + scope + "\n" + finding.detail);
    std::string subject_basis = finding.category + "\n" + finding.rule_id + "\n" + scope + "\n" + finding.detail;
    if (finding.rule_id == "sclv.affected_surface.unindexed") {
        const auto path = values.find("path");
        if (path != values.end()) {
            subject_basis = finding.rule_id + "\npath=" + path->second;
        }
    }
    return engine::Json{
        {"attributes", values},
        {"category", finding.category},
        {"detail", finding.detail},
        {"occurrence_id", occurrence},
        {"rule_id", finding.rule_id},
        {"scope", scope},
        {"subject_id", engine::tagged_sha256(subject_basis)},
    };
}

std::string repository_identity(const std::string& repository_path) {
    std::error_code error;
    auto canonical = fs::canonical(fs::path(repository_path), error);
    if (error) {
        canonical = fs::absolute(fs::path(repository_path), error).lexically_normal();
    }
    return engine::tagged_sha256(canonical.generic_string());
}

}

std::string project_validation_result(
    const std::string& repository_path,
    const std::vector<std::string>& messages,
    const int exit_code) {
    engine::Json findings = engine::Json::array();
    std::size_t pass = 0U;
    std::size_t warning = 0U;
    std::size_t violation = 0U;
    std::size_t other = 0U;
    for (const auto& message : messages) {
        const auto parsed = parse_evidence(message);
        pass += parsed.category == "pass" ? 1U : 0U;
        warning += parsed.category == "warning" ? 1U : 0U;
        violation += parsed.category == "violation" ? 1U : 0U;
        other += parsed.category != "pass" && parsed.category != "warning" &&
                         parsed.category != "violation"
                     ? 1U
                     : 0U;
        findings.push_back(finding_json(parsed));
    }
    engine::Json evidence{
        {"exit_code", exit_code},
        {"findings", std::move(findings)},
        {"outcome", exit_code == 0 ? "pass" : "violation"},
        {"repository_identity_digest", repository_identity(repository_path)},
        {"summary", engine::Json{
            {"other", other}, {"pass", pass},
            {"total", pass + warning + violation + other},
            {"violation", violation}, {"warning", warning},
        }},
        {"validator_id", validator_id},
        {"validator_version", validator_version},
    };
    evidence["evidence_digest"] = engine::tagged_sha256(evidence.dump());
    engine::Json result{
        {"evaluation", nullptr},
        {"evidence", std::move(evidence)},
        {"format_version", 1},
        {"protocol", "symphony.validation.result.v1"},
    };
    result["result_digest"] = engine::tagged_sha256(result.dump());
    return result.dump();
}
