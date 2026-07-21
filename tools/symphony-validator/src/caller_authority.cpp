#include "caller_authority.hpp"
#include "evidence.hpp"
#include <algorithm>
#include <cctype>
#include <compare>
#include <filesystem>
#include <fstream>
#include <system_error>
#include <utility>

namespace fs = std::filesystem;

namespace {

constexpr std::size_t MAX_PHYSICAL_LINE = 64 * 1024;
constexpr std::size_t MAX_NORMALIZED_PARAGRAPH = 256 * 1024;
constexpr std::size_t MAX_FILE_SIZE = 4 * 1024 * 1024;

struct ScanFaults {
    std::string tellg_failure_path;
    std::string metadata_failure_path;
    std::string iterator_construction_failure_path;
    std::string iterator_increment_failure_path;
};

struct DiscoveryFinding {
    std::string path;
    std::string rule_id;
    std::string detail;

    auto operator<=>(const DiscoveryFinding&) const = default;
};

bool is_sentence_boundary(const std::string& token) {
    return token == "." || token == "!" || token == "?";
}

std::string lexical_relative(const fs::path& path, const fs::path& repo_root) {
    const fs::path relative = path.lexically_relative(repo_root);
    return relative.empty() ? path.generic_string() : relative.generic_string();
}

std::string trim(const std::string& str) {
    auto start = str.find_first_not_of(" \t\r\n");
    if (start == std::string::npos) return "";
    auto end = str.find_last_not_of(" \t\r\n");
    return str.substr(start, end - start + 1);
}

struct Token {
    std::string text;
    int line;
};

std::vector<Token> tokenize_paragraph(const std::string& raw_paragraph, int start_line) {
    std::vector<Token> tokens;
    tokens.reserve(raw_paragraph.size() / 5);
    std::string current;
    current.reserve(32);
    int current_line = start_line;
    for (std::size_t i = 0; i < raw_paragraph.size(); ++i) {
        char c = raw_paragraph[i];
        if (c == '\n') {
            if (!current.empty()) {
                tokens.push_back({current, current_line});
                current.clear();
            }
            current_line++;
        } else {
            unsigned char uc = static_cast<unsigned char>(c);
            if (std::isalnum(uc)) {
                current.push_back(static_cast<char>(std::tolower(uc)));
            } else if (c == '-' || c == '_' || c == '/') {
                if (!current.empty()) {
                    tokens.push_back({current, current_line});
                    current.clear();
                }
            } else if (c == '.' || c == '!' || c == '?') {
                if (!current.empty()) {
                    tokens.push_back({current, current_line});
                    current.clear();
                }
                tokens.push_back({std::string(1, c), current_line});
            } else {
                if (!current.empty()) {
                    tokens.push_back({current, current_line});
                    current.clear();
                }
            }
        }
    }
    if (!current.empty()) {
        tokens.push_back({current, current_line});
    }
    return tokens;
}

bool match_phrase(const std::vector<Token>& tokens, std::size_t index, const std::vector<std::string>& phrase) {
    if (index + phrase.size() > tokens.size()) return false;
    for (std::size_t i = 0; i < phrase.size(); ++i) {
        if (tokens[index + i].text != phrase[i]) return false;
    }
    return true;
}

struct MatchedPhrase {
    bool matched = false;
    std::size_t length = 0;
    std::string value;
};

MatchedPhrase match_class_subject(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::pair<std::vector<std::string>, std::string>> subjects = {
        {{"artificial", "intelligence", "callers"}, "artificial_intelligence_caller"},
        {{"artificial", "intelligence", "caller"}, "artificial_intelligence_caller"},
        {{"artificial", "intelligence"}, "artificial_intelligence"},
        {{"human", "callers"}, "human_caller"},
        {{"human", "caller"}, "human_caller"},
        {{"humans"}, "human"},
        {{"human"}, "human"},
        {{"ai", "agents"}, "ai_agent"},
        {{"ai", "agent"}, "ai_agent"},
        {{"ai"}, "ai"},
        {{"agentic", "callers"}, "agentic_caller"},
        {{"agentic", "caller"}, "agentic_caller"},
        {{"agents"}, "agent"},
        {{"agent"}, "agent"},
        {{"nonhuman", "callers"}, "nonhuman_caller"},
        {{"nonhuman", "caller"}, "nonhuman_caller"},
        {{"machine", "callers"}, "machine_caller"},
        {{"machine", "caller"}, "machine_caller"},
        {{"service", "callers"}, "service_caller"},
        {{"service", "caller"}, "service_caller"},
        {{"workload", "callers"}, "workload_caller"},
        {{"workload", "caller"}, "workload_caller"},
        {{"organization", "callers"}, "organization_caller"},
        {{"organization", "caller"}, "organization_caller"},
        {{"future", "actors"}, "future_actor"},
        {{"future", "actor"}, "future_actor"}
    };
    for (const auto& [phrase, id] : subjects) {
        if (match_phrase(tokens, index, phrase)) return {true, phrase.size(), id};
    }
    return {false, 0, ""};
}

MatchedPhrase match_modal(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::vector<std::string>> modals = {
        {"are", "not", "authorized", "to"}, {"are", "not", "allowed", "to"}, {"are", "not", "permitted", "to"},
        {"is", "not", "authorized", "to"}, {"is", "not", "allowed", "to"}, {"is", "not", "permitted", "to"},
        {"are", "authorized", "to"}, {"are", "allowed", "to"}, {"are", "permitted", "to"},
        {"is", "authorized", "to"}, {"is", "allowed", "to"}, {"is", "permitted", "to"},
        {"are", "not", "authorized"}, {"are", "not", "allowed"}, {"are", "not", "permitted"},
        {"is", "not", "authorized"}, {"is", "not", "allowed"}, {"is", "not", "permitted"},
        {"are", "authorized"}, {"are", "allowed"}, {"are", "permitted"},
        {"is", "authorized"}, {"is", "allowed"}, {"is", "permitted"},
        {"may", "not"}, {"may", "never"}, {"may", "only"}, {"must", "not"}, {"can", "not"}, {"cannot"}, {"could", "not"}, {"should", "not"}, {"will", "not"}, {"shall", "not"},
        {"may"}, {"must"}, {"can"}, {"could"}, {"should"}, {"will"}, {"shall"}
    };
    for (const auto& phrase : modals) {
        if (match_phrase(tokens, index, phrase)) return {true, phrase.size(), ""};
    }
    return {false, 0, ""};
}

MatchedPhrase match_governed_op(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::string> ops = {
        "apply", "approve", "ratify", "ratification", "review", "authorize", "mutate", "write", "edit", "append", "truncate", "rotate", "repair", "query", "propose", "administer", "configure", "access", "receive", "release", "use", "invoke", "install", "uninstall", "own", "sign", "transact"
    };
    if (index < tokens.size()) {
        for (const auto& op : ops) {
            if (tokens[index].text == op) return {true, 1, ""};
        }
    }
    return {false, 0, ""};
}

MatchedPhrase match_status(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::vector<std::string>> statuses = {
        {"query", "only"}, {"proposal", "only"}, {"read", "only"},
        {"allowed"}, {"authorized"}, {"permitted"}, {"eligible"}, {"unauthorized"}, {"forbidden"}, {"prohibited"}, {"banned"}, {"denied"}, {"ineligible"}, {"restricted"}, {"limited"}
    };
    for (const auto& phrase : statuses) {
        if (match_phrase(tokens, index, phrase)) return {true, phrase.size(), ""};
    }
    return {false, 0, ""};
}

MatchedPhrase match_availability_predicate(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::string> avail = {
        "available", "unavailable", "granted", "withheld", "denied", "reserved"
    };
    if (index < tokens.size()) {
        for (const auto& a : avail) {
            if (tokens[index].text == a) return {true, 1, ""};
        }
    }
    return {false, 0, ""};
}

MatchedPhrase match_affirmative_gate(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::vector<std::string>> gates = {
        {"requires"}, {"must", "receive"}, {"subject", "to"}, {"approved", "by"}
    };
    for (const auto& phrase : gates) {
        if (match_phrase(tokens, index, phrase)) return {true, phrase.size(), ""};
    }
    return {false, 0, ""};
}

MatchedPhrase match_causality_verb(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::vector<std::string>> verbs = {
        {"changes", "according", "to"}, {"differs", "based", "on"}, {"depends", "on"}, {"varies", "by"}, {"vary", "by"}, {"uses"}, {"determines"}
    };
    for (const auto& phrase : verbs) {
        if (match_phrase(tokens, index, phrase)) return {true, phrase.size(), ""};
    }
    return {false, 0, ""};
}

MatchedPhrase match_caller_type(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::vector<std::string>> types = {
        {"whether", "the", "caller", "is"}, {"caller", "type"}, {"caller", "class"}, {"classification"}, {"caller", "is"}
    };
    for (const auto& phrase : types) {
        if (match_phrase(tokens, index, phrase)) return {true, phrase.size(), ""};
    }
    return {false, 0, ""};
}

MatchedPhrase match_authorization_concept(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::string> concepts = {
        "authorization", "authority", "permission", "permissions", "eligibility", "access"
    };
    if (index < tokens.size()) {
        for (const auto& c : concepts) {
            if (tokens[index].text == c) return {true, 1, ""};
        }
    }
    return match_governed_op(tokens, index);
}

MatchedPhrase match_human_governance_phrase(const std::vector<Token>& tokens, std::size_t index) {
    static const std::vector<std::pair<std::vector<std::string>, std::string>> phrases = {
        {{"humans", "ratify"}, "human_ratify"},
        {{"human", "ratified"}, "human_ratify"},
        {{"human", "approval"}, "human_approval"},
        {{"human", "review"}, "human_review"},
        {{"human", "authored", "truth"}, "human_authored"},
        {{"human", "authored", "canonical", "record"}, "human_authored"}
    };
    for (const auto& [phrase, type] : phrases) {
        if (match_phrase(tokens, index, phrase)) return {true, phrase.size(), type};
    }
    return {false, 0, ""};
}

bool check_paragraph_tokens(const std::vector<Token>& tokens, const std::string& rel_path, std::vector<std::string>& messages) {
    bool has_violation = false;

    std::vector<std::string> seen;
    auto emit_violation = [&](const std::string& rule_id, int start_line, int end_line, const std::string& matched_class = "") {
        std::string dedup = rule_id;
        if (std::find(seen.begin(), seen.end(), dedup) != seen.end()) return;
        seen.push_back(dedup);

        std::string loc = "path=" + rel_path + " line=";
        if (start_line == end_line) loc += std::to_string(start_line);
        else loc += std::to_string(start_line) + "-" + std::to_string(end_line);

        std::string extra = loc;
        if (!matched_class.empty()) extra += " class=" + matched_class;
        messages.push_back(format_evidence(EvidenceCategory::Violation, rule_id, extra));
        has_violation = true;
    };

    for (std::size_t i = 0; i < tokens.size(); ++i) {
        auto subj = match_class_subject(tokens, i);
        if (subj.matched) {
            auto mod = match_modal(tokens, i + subj.length);
            if (mod.matched) {
                std::size_t next_idx = i + subj.length + mod.length;
                bool found_op = false;
                for (std::size_t gap = 0; gap <= 3 && next_idx + gap < tokens.size(); ++gap) {
                    if (is_sentence_boundary(tokens[next_idx + gap].text)) break;
                    auto op = match_governed_op(tokens, next_idx + gap);
                    if (op.matched) {
                        emit_violation("caller_authority.class_subject_modal", tokens[i].line, tokens[next_idx + gap + op.length - 1].line, subj.value);
                        found_op = true;
                        break;
                    }
                }
                if (found_op) continue;
            }
        }

        if (subj.matched) {
            std::size_t next_idx = i + subj.length;
            if (next_idx < tokens.size()) {
                const std::string& t = tokens[next_idx].text;
                if (t == "are" || t == "is" || t == "were" || t == "was" || t == "be" || t == "being" || t == "been") {
                    for (std::size_t gap = 1; gap <= 2 && next_idx + gap < tokens.size(); ++gap) {
                        if (is_sentence_boundary(tokens[next_idx + gap].text)) break;
                        auto st = match_status(tokens, next_idx + gap);
                        if (st.matched) {
                            emit_violation("caller_authority.class_subject_status", tokens[i].line, tokens[next_idx + gap + st.length - 1].line, subj.value);
                            break;
                        }
                    }
                }
            }
        }

        bool exclusive_match = false;
        std::size_t excl_end = 0;
        std::string excl_class = "";
        if (tokens[i].text == "only") {
            auto c = match_class_subject(tokens, i + 1);
            if (c.matched) { exclusive_match = true; excl_end = i + 1 + c.length; excl_class = c.value; }
        } else {
            auto c = match_class_subject(tokens, i);
            if (c.matched && i + c.length < tokens.size() && tokens[i + c.length].text == "only") {
                exclusive_match = true; excl_end = i + c.length + 1; excl_class = c.value;
            }
        }
        if (exclusive_match) {
            for (std::size_t gap = 0; gap <= 12 && excl_end + gap < tokens.size(); ++gap) {
                if (is_sentence_boundary(tokens[excl_end + gap].text)) break;
                auto op = match_governed_op(tokens, excl_end + gap);
                if (op.matched) {
                    emit_violation("caller_authority.class_exclusive_operation", tokens[i].line, tokens[excl_end + gap + op.length - 1].line, excl_class);
                    break;
                }
                auto st = match_status(tokens, excl_end + gap);
                if (st.matched) {
                    emit_violation("caller_authority.class_exclusive_operation", tokens[i].line, tokens[excl_end + gap + st.length - 1].line, excl_class);
                    break;
                }
            }

            bool has_to_for = false;
            std::size_t to_for_idx = i;
            if (i > 0 && (tokens[i-1].text == "to" || tokens[i-1].text == "for")) {
                has_to_for = true;
                to_for_idx = i - 1;
            }
            if (has_to_for) {
                for (std::size_t j = (to_for_idx >= 12 ? to_for_idx - 12 : 0); j < to_for_idx; ++j) {
                    bool has_bound = false;
                    for (std::size_t k = j; k < to_for_idx; ++k) {
                        if (is_sentence_boundary(tokens[k].text)) {
                            has_bound = true;
                            break;
                        }
                    }
                    if (has_bound) continue;
                    auto op = match_governed_op(tokens, j);
                    if (op.matched && j + op.length <= to_for_idx) {
                        emit_violation("caller_authority.class_exclusive_operation", tokens[j].line, tokens[excl_end - 1].line, excl_class);
                        break;
                    }
                    auto st = match_status(tokens, j);
                    if (st.matched && j + st.length <= to_for_idx) {
                        emit_violation("caller_authority.class_exclusive_operation", tokens[j].line, tokens[excl_end - 1].line, excl_class);
                        break;
                    }
                }
            }
        }

        auto avail = match_availability_predicate(tokens, i);
        if (avail.matched) {
            std::size_t next_idx = i + avail.length;
            if (next_idx < tokens.size() && (tokens[next_idx].text == "to" || tokens[next_idx].text == "for")) {
                next_idx++;
                for (std::size_t gap = 0; gap <= 4 && next_idx + gap < tokens.size(); ++gap) {
                    if (is_sentence_boundary(tokens[next_idx + gap].text)) break;
                    auto c = match_class_subject(tokens, next_idx + gap);
                    if (c.matched) {
                        emit_violation("caller_authority.class_targeted_availability", tokens[i].line, tokens[next_idx + gap + c.length - 1].line, c.value);
                        break;
                    }
                }
            }
        }

        auto hum = match_human_governance_phrase(tokens, i);
        if (hum.matched) {
            if (hum.value == "human_ratify" || hum.value == "human_authored") {
                emit_violation("caller_authority.human_exclusive_governance", tokens[i].line, tokens[i + hum.length - 1].line, "human");
            } else if (hum.value == "human_approval" || hum.value == "human_review") {
                bool gated = false;
                std::size_t gate_idx = 0; std::size_t gate_len = 0;
                for (std::size_t j = i; j > (i >= 8 ? i - 8 : 0); --j) {
                    if (is_sentence_boundary(tokens[j - 1].text)) break;
                    auto g = match_affirmative_gate(tokens, j - 1);
                    if (g.matched) { gated = true; gate_idx = j - 1; gate_len = g.length; break; }
                }
                if (!gated) {
                    for (std::size_t gap = 0; gap <= 8 && i + hum.length + gap < tokens.size(); ++gap) {
                        if (is_sentence_boundary(tokens[i + hum.length + gap].text)) break;
                        auto g = match_affirmative_gate(tokens, i + hum.length + gap);
                        if (g.matched) { gated = true; gate_idx = i + hum.length + gap; gate_len = g.length; break; }
                    }
                }
                if (gated) {
                    std::size_t start_idx = std::min(i, gate_idx);
                    start_idx = (start_idx >= 2) ? start_idx - 2 : 0;
                    std::size_t end_idx = std::max(i + hum.length, gate_idx + gate_len);
                    bool negated = false;
                    for (std::size_t j = start_idx; j < end_idx; ++j) {
                        if (tokens[j].text == "not" || tokens[j].text == "never" || tokens[j].text == "no") {
                            negated = true; break;
                        }
                    }
                    if (!negated) {
                        emit_violation("caller_authority.human_exclusive_governance", tokens[i].line, tokens[i + hum.length - 1].line, "human");
                    }
                }
            }
        }

        auto verb = match_causality_verb(tokens, i);
        if (verb.matched) {
            std::size_t sent_start = i;
            while (sent_start > 0 && !is_sentence_boundary(tokens[sent_start - 1].text)) sent_start--;
            std::size_t sent_end = i + verb.length;
            while (sent_end < tokens.size() && !is_sentence_boundary(tokens[sent_end].text)) sent_end++;
            bool has_concept = false;
            std::size_t min_dist_c = 1000;
            for (std::size_t j = std::max(sent_start, i >= 12 ? i - 12 : 0); j <= std::min(sent_end, i + verb.length + 12) && j < tokens.size(); ++j) {
                auto c = match_authorization_concept(tokens, j);
                if (c.matched) {
                    std::size_t dist = (j < i) ? (i - j) : (j - i);
                    if (dist < min_dist_c) {
                        min_dist_c = dist;
                        has_concept = true;
                    }
                }
            }
            if (has_concept) {
                bool has_type = false;
                std::size_t min_dist_t = 1000;
                for (std::size_t j = std::max(sent_start, i >= 12 ? i - 12 : 0); j <= std::min(sent_end, i + verb.length + 12) && j < tokens.size(); ++j) {
                    auto t = match_caller_type(tokens, j);
                    if (t.matched) {
                        std::size_t dist = (j < i) ? (i - j) : (j - i);
                        if (dist < min_dist_t) {
                            min_dist_t = dist;
                            has_type = true;
                        }
                    }
                }
                if (has_type) {
                    bool negated = false;
                    const std::size_t negation_start = std::max(sent_start, i >= 2 ? i - 2 : 0);
                    for (std::size_t j = negation_start; j < i; ++j) {
                        if (tokens[j].text == "not" || tokens[j].text == "never" || tokens[j].text == "no") {
                            negated = true; break;
                        }
                    }
                    if (!negated) {
                        emit_violation("caller_authority.caller_type_decision", tokens[i].line, tokens[i + verb.length - 1].line, "");
                    }
                }
            }
        }
    }

    return has_violation;
}

void process_file(
    const std::string& repo_root,
    const std::string& rel_path,
    CallerAuthorityCheckResult& result,
    std::size_t& paragraphs_count,
    const ScanFaults& faults) {
    std::string full_path = repo_root + "/" + rel_path;
    std::ifstream file(full_path, std::ios::binary | std::ios::ate);
    if (!file.is_open()) {
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "caller_authority.unreadable", "path=" + rel_path));
        result.success = false;
        return;
    }

    const std::streampos end_position = rel_path == faults.tellg_failure_path
        ? std::streampos{-1}
        : file.tellg();
    if (end_position == std::streampos{-1}) {
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "caller_authority.unreadable", "path=" + rel_path));
        result.success = false;
        return;
    }
    const std::streamoff size = static_cast<std::streamoff>(end_position);
    file.seekg(0, std::ios::beg);
    if (!file) {
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "caller_authority.unreadable", "path=" + rel_path));
        result.success = false;
        return;
    }

    if (rel_path != "knowledge/sclv/CHANGELOG.md" && rel_path != "knowledge/sodv/RELEASES.md") {
        if (size > static_cast<std::streamoff>(MAX_FILE_SIZE)) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "caller_authority.file_size_exceeded", "path=" + rel_path));
            result.success = false;
            return;
        }
    }

    std::string line;
    int line_number = 0;

    std::string current_paragraph_raw;
    int current_start_line = 0;
    bool has_violation_in_file = false;

    auto flush_paragraph = [&]() {
        if (current_paragraph_raw.empty()) return;
        if (current_paragraph_raw.length() > MAX_NORMALIZED_PARAGRAPH) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "caller_authority.paragraph_size_exceeded", "path=" + rel_path));
            result.success = false;
            has_violation_in_file = true;
        } else {
            std::vector<Token> tokens = tokenize_paragraph(current_paragraph_raw, current_start_line);
            if (!tokens.empty()) {
                paragraphs_count++;
                if (check_paragraph_tokens(tokens, rel_path, result.messages)) {
                    result.success = false;
                    has_violation_in_file = true;
                }
            }
        }
        current_paragraph_raw.clear();
        current_start_line = 0;
    };

    while (std::getline(file, line)) {
        line_number++;
        if (line.length() > MAX_PHYSICAL_LINE) {
            result.messages.push_back(format_evidence(EvidenceCategory::Violation, "caller_authority.line_length_exceeded", "path=" + rel_path + " line=" + std::to_string(line_number)));
            result.success = false;
            has_violation_in_file = true;
            return;
        }

        std::string trimmed = trim(line);

        if (rel_path == "knowledge/sclv/CHANGELOG.md" && trimmed.find("- record_id:") == 0) {
            flush_paragraph();
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "caller_authority.historical_region_exempt", "path=" + rel_path + " start_line=" + std::to_string(line_number) + " boundary=record_id"));
            if (!has_violation_in_file) {
                 result.messages.push_back(format_evidence(EvidenceCategory::Pass, "caller_authority.clean", "path=" + rel_path));
            }
            return;
        }
        if (rel_path == "knowledge/sodv/RELEASES.md" && trimmed.find("- release_record_id:") == 0) {
            flush_paragraph();
            result.messages.push_back(format_evidence(EvidenceCategory::Pass, "caller_authority.historical_region_exempt", "path=" + rel_path + " start_line=" + std::to_string(line_number) + " boundary=release_record_id"));
            if (!has_violation_in_file) {
                 result.messages.push_back(format_evidence(EvidenceCategory::Pass, "caller_authority.clean", "path=" + rel_path));
            }
            return;
        }

        if (trimmed.empty() || trimmed[0] == '#') {
            flush_paragraph();
            if (!trimmed.empty() && trimmed[0] == '#') {
                current_start_line = line_number;
                current_paragraph_raw = line + "\n";
            }
        } else {
            if (current_paragraph_raw.empty()) {
                current_start_line = line_number;
            }
            current_paragraph_raw += line + "\n";
            if (current_paragraph_raw.length() > MAX_NORMALIZED_PARAGRAPH) {
                flush_paragraph();
            }
        }
    }

    if (file.bad()) {
        result.messages.push_back(format_evidence(EvidenceCategory::Violation, "caller_authority.unreadable", "path=" + rel_path));
        result.success = false;
        return;
    }

    flush_paragraph();
    if (!has_violation_in_file) {
        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "caller_authority.clean", "path=" + rel_path));
    }
}

}

namespace {

CallerAuthorityCheckResult check_caller_authority_impl(
    const std::string& repo_root,
    const ScanFaults& faults) {
    CallerAuthorityCheckResult result;
    result.success = true;

    const fs::path root_path(repo_root);
    std::vector<std::string> target_paths;
    std::vector<DiscoveryFinding> pending_discovery;

    auto add_discovery_finding = [&](const std::string& rule_id, const std::string& rel_path) {
        pending_discovery.push_back({rel_path, rule_id, "path=" + rel_path});
        result.success = false;
    };

    auto add_file_if_markdown = [&](const fs::path& path, bool missing_allowed) {
        const std::string relative_path = lexical_relative(path, root_path);
        if (relative_path == faults.metadata_failure_path) {
            add_discovery_finding("caller_authority.discovery_failed", relative_path);
            return;
        }

        std::error_code ec;
        const fs::file_status status = fs::symlink_status(path, ec);
        if (ec) {
            if (missing_allowed && ec == std::errc::no_such_file_or_directory) return;
            add_discovery_finding("caller_authority.discovery_failed", relative_path);
            return;
        }

        if (fs::is_symlink(status)) {
            if (path.extension() == ".md") {
                add_discovery_finding("caller_authority.symlink_unsupported", relative_path);
            }
            return;
        }

        if (fs::is_regular_file(status) && path.extension() == ".md") {
            target_paths.push_back(relative_path);
        }
    };

    add_file_if_markdown(root_path / "README.md", true);
    add_file_if_markdown(root_path / "INTENT.md", true);

    const std::vector<std::string> directories = {
        "knowledge", "modules", "libraries", "tools/qxctl", "tools/symphony-validator"
    };
    for (const auto& directory : directories) {
        const fs::path directory_path = root_path / directory;
        if (directory == faults.metadata_failure_path) {
            add_discovery_finding("caller_authority.discovery_failed", directory);
            continue;
        }

        std::error_code ec;
        const fs::file_status directory_status = fs::symlink_status(directory_path, ec);
        if (ec) {
            if (ec == std::errc::no_such_file_or_directory) continue;
            add_discovery_finding("caller_authority.discovery_failed", directory);
            continue;
        }
        if (!fs::is_directory(directory_status)) continue;

        if (directory == faults.iterator_construction_failure_path) {
            add_discovery_finding("caller_authority.discovery_failed", directory);
            continue;
        }

        fs::recursive_directory_iterator iterator(
            directory_path,
            fs::directory_options::none,
            ec);
        if (ec) {
            add_discovery_finding("caller_authority.discovery_failed", directory);
            continue;
        }

        const fs::recursive_directory_iterator end;
        while (iterator != end) {
            const fs::path current_path = iterator->path();
            const std::string relative_path = lexical_relative(current_path, root_path);

            if (relative_path == faults.metadata_failure_path) {
                add_discovery_finding("caller_authority.discovery_failed", relative_path);
            } else {
                std::error_code status_error;
                const fs::file_status status = fs::symlink_status(current_path, status_error);
                if (status_error) {
                    add_discovery_finding("caller_authority.discovery_failed", relative_path);
                } else if (fs::is_symlink(status)) {
                    if (current_path.extension() == ".md") {
                        add_discovery_finding("caller_authority.symlink_unsupported", relative_path);
                    }
                } else if (fs::is_directory(status)) {
                    if (current_path.filename() == "build" ||
                        relative_path == "tools/symphony-validator/tests") {
                        iterator.disable_recursion_pending();
                    }
                } else if (fs::is_regular_file(status) && current_path.extension() == ".md") {
                    target_paths.push_back(relative_path);
                }
            }

            if (relative_path == faults.iterator_increment_failure_path) {
                add_discovery_finding("caller_authority.discovery_failed", relative_path);
                break;
            }

            iterator.increment(ec);
            if (ec) {
                add_discovery_finding("caller_authority.discovery_failed", relative_path);
                break;
            }
        }
    }

    std::sort(pending_discovery.begin(), pending_discovery.end());
    pending_discovery.erase(std::unique(pending_discovery.begin(), pending_discovery.end()), pending_discovery.end());
    for (const auto& finding : pending_discovery) {
        result.messages.push_back(format_evidence(
            EvidenceCategory::Violation,
            finding.rule_id,
            finding.detail));
    }

    std::sort(target_paths.begin(), target_paths.end());
    target_paths.erase(std::unique(target_paths.begin(), target_paths.end()), target_paths.end());

    std::size_t files_count = target_paths.size();
    std::size_t paragraphs_count = 0;

    for (const auto& rel : target_paths) {
        process_file(repo_root, rel, result, paragraphs_count, faults);
    }

    std::size_t exemptions = 0;
    std::size_t findings = 0;
    for (const auto& msg : result.messages) {
        if (msg.find("evidence violation") != std::string::npos) findings++;
        if (msg.find("historical_region_exempt") != std::string::npos) exemptions++;
    }

    if (result.success && findings == 0) {
        std::string summary = "files=" + std::to_string(files_count) +
                              " paragraphs=" + std::to_string(paragraphs_count) +
                              " exemptions=" + std::to_string(exemptions) +
                              " findings=" + std::to_string(findings);

        result.messages.push_back(format_evidence(EvidenceCategory::Pass, "caller_authority.scan_complete", summary));
    }

    return result;
}

}

CallerAuthorityCheckResult check_caller_authority(const std::string& repo_root) {
    return check_caller_authority_impl(repo_root, {});
}

#ifdef SYMPHONY_VALIDATOR_TESTING
CallerAuthorityCheckResult check_caller_authority_with_test_fault(
    const std::string& repo_root,
    CallerAuthorityTestFault fault,
    const std::string& relative_path) {
    ScanFaults faults;
    switch (fault) {
        case CallerAuthorityTestFault::tellg_failure:
            faults.tellg_failure_path = relative_path;
            break;
        case CallerAuthorityTestFault::metadata_failure:
            faults.metadata_failure_path = relative_path;
            break;
        case CallerAuthorityTestFault::iterator_construction_failure:
            faults.iterator_construction_failure_path = relative_path;
            break;
        case CallerAuthorityTestFault::iterator_increment_failure:
            faults.iterator_increment_failure_path = relative_path;
            break;
    }
    return check_caller_authority_impl(repo_root, faults);
}
#endif
