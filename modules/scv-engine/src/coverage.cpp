#include "coverage.hpp"
#include "corpus.hpp"
#include "interpretation.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/error.hpp"

#include <algorithm>
#include <map>
#include <set>
#include <string_view>
#include <tuple>

namespace symphony::knowledge::scv {
namespace {
using Map = std::map<std::string, Json>;
using RowKey = std::pair<std::string, std::string>;
[[noreturn]] void invalid(const std::string& message) { throw engine::Error("coverage.invalid", message, 4); }
void deadline(const engine::Request& request) {
    if (engine::unix_time_ms() > request.deadline_unix_ms)
        throw engine::Error("request.deadline_exceeded", "provider coverage deadline exceeded", 4);
}
void fields(const Json& value, std::initializer_list<std::string_view> keys) {
    if (!value.is_object() || value.size() != keys.size()) invalid("unexpected provider coverage fields");
    for (const auto key : keys) if (!value.contains(std::string(key))) invalid("missing provider coverage field");
}
std::string text(const Json& value, const std::string& key) {
    if (!value.contains(key) || !value.at(key).is_string()) invalid("missing text: " + key);
    const auto s = value.at(key).get<std::string>();
    if (s.empty() || s.size() > 512 || s.find('\0') != std::string::npos) invalid("invalid text: " + key);
    return s;
}
bool known_domain(const std::string& domain) {
    return domain == "scv" || domain == "schv" || domain == "scev" || domain == "schv-aws" ||
        domain == "schv-azure" || domain == "schv-do" || domain == "schv-gcp" || domain == "scev-cf";
}
bool admits_domain(const std::string& domain, const std::string& owner) {
    if (!known_domain(owner)) return false;
    return domain == "scv" || domain == owner ||
        ((domain == "schv" || domain == "scev") && owner.starts_with(domain + "-"));
}
engine::Request nested(const engine::Request& request, const std::string& operation, Json payload) {
    auto result = request; result.operation = operation; result.payload = std::move(payload); return result;
}
std::string identity(const Json& index) {
    return Json::array({index.at("family_id"), index.at("provider_id"), index.at("source_id"), index.at("locator_id")}).dump();
}
void increment(Json& object, const std::string& key) { object[key] = object.at(key).get<std::size_t>() + 1; }
auto binding_key(const Json& value) {
    return std::tuple{text(value, "interpretation_digest"), text(value, "profile_digest"), text(value, "capture_digest")};
}
void sort_bindings(Json& values) {
    std::sort(values.begin(), values.end(), [](const Json& a, const Json& b) { return binding_key(a) < binding_key(b); });
}
Json replay_provider(const Json& provider, const engine::Request& request, const std::string& domain) {
    fields(provider, {"provider_id", "family_id", "display_name", "sources", "protocol", "disposition", "interpretation_scope", "digest"});
    const auto rebuilt = handle_source(nested(request, "provider_onboard", {{"provider_id", provider.at("provider_id")},
        {"family_id", provider.at("family_id")}, {"display_name", provider.at("display_name")}, {"sources", provider.at("sources")}}), domain);
    if (rebuilt != provider) invalid("provider declaration cannot be exactly replayed");
    return rebuilt;
}
void replay_wrapper(const Json& wrapper, const engine::Request& request, const std::string& domain) {
    fields(wrapper, {"protocol", "domain", "profiles", "bindings", "knowledge", "extractions", "limitations", "digest"});
    const auto owner = text(wrapper, "domain");
    if (!known_domain(owner)) invalid("unsupported interpretation domain");
    const auto& knowledge = wrapper.at("knowledge");
    if (!knowledge.is_object() || !knowledge.contains("captures") || !knowledge.contains("selection_policy") ||
        !knowledge.at("captures").is_array() || knowledge.at("captures").size() > 16) invalid("missing bounded interpretation inputs");
    for (const auto& capture : knowledge.at("captures")) validate_capture(capture, domain);
    const auto rebuilt = handle_interpretation(nested(request, "provider_interpret", {{"captures", knowledge.at("captures")},
        {"profiles", wrapper.at("profiles")}, {"bindings", wrapper.at("bindings")}, {"selection_policy", knowledge.at("selection_policy")}}), owner);
    if (rebuilt != wrapper) invalid("provider interpretation cannot be exactly replayed");
}
Json coverage(const engine::Request& request, const std::string& domain) {
    const auto& input = request.payload;
    fields(input, {"provider", "corpus_query", "interpretations"});
    if (!known_domain(domain)) invalid("unsupported provider coverage domain");
    if (!input.at("interpretations").is_array() || input.at("interpretations").size() > 16) invalid("coverage admits at most 16 interpretation wrappers");
    const auto provider = replay_provider(input.at("provider"), request, domain);
    if (!input.at("corpus_query").is_object() || !input.at("corpus_query").contains("corpus")) invalid("missing corpus query input");
    const auto& corpus = input.at("corpus_query").at("corpus");
    const auto corpus_domain = text(corpus, "domain");
    if (!admits_domain(domain, corpus_domain)) invalid("corpus outside provider coverage domain");
    const auto query = handle_corpus(nested(request, "corpus_query", input.at("corpus_query")), corpus_domain);
    Map members, selected, selected_captures, declarations;
    for (const auto& source : provider.at("sources")) declarations.emplace(text(source, "source_id"), source);
    for (const auto& member : corpus.at("members")) members.emplace(identity(member.at("latest_attempt")), member);
    for (const auto& member : query.at("members")) {
        selected.emplace(text(member, "member_id"), member);
        if (!member.at("selected").is_null()) selected_captures.emplace(text(member.at("selected"), "capture_digest"), member);
    }
    std::map<RowKey, Json> rows;
    std::map<std::string, RowKey> declared_members;
    Json summary = {{"declared_sources", provider.at("sources").size()}, {"declared_locators", 0},
        {"selected_declared_members", 0}, {"unselected_declared_locators", 0}, {"selected_unlisted_members", 0},
        {"replayed_selected_captures", 0}, {"selected_profile_bindings", 0}, {"matched_rule_attempts", 0},
        {"unresolved_rule_attempts", 0}, {"unselected_profile_bindings", 0},
        {"selected_declared_status", {{"complete", 0}, {"partial", 0}, {"failed", 0}, {"unavailable", 0}}},
        {"selected_declared_freshness", {{"current", 0}, {"expired", 0}, {"future", 0}, {"not_selected", 0}}}};
    for (const auto& source : provider.at("sources")) for (const auto& locator : source.at("locators")) {
        deadline(request);
        const RowKey key{text(source, "source_id"), text(locator, "locator_id")};
        Json row = {{"source_id", key.first}, {"locator_id", key.second}, {"member_id", nullptr},
            {"selection_status", "not_selected"}, {"selected_capture_digest", nullptr}, {"declaration_match", "not_available"},
            {"declaration_difference_fields", Json::array()}, {"interpretations", Json::array()}};
        const auto tuple = Json::array({source.at("family_id"), source.at("provider_id"), source.at("source_id"), locator.at("locator_id")}).dump();
        const auto member = members.find(tuple);
        if (member != members.end()) {
            const auto id = text(member->second, "member_id"); row["member_id"] = id; declared_members.emplace(id, key);
            const auto chosen = selected.find(id);
            if (chosen != selected.end()) {
                increment(summary, "selected_declared_members");
                increment(summary.at("selected_declared_status"), text(chosen->second, "status"));
                increment(summary.at("selected_declared_freshness"), text(chosen->second, "freshness"));
                row["selection_status"] = chosen->second.at("selected").is_null() ? "unavailable" : "selected";
                if (!chosen->second.at("selected").is_null()) row["selected_capture_digest"] = chosen->second.at("selected").at("capture_digest");
            }
        }
        if (row.at("selection_status") == "not_selected") increment(summary, "unselected_declared_locators");
        increment(summary, "declared_locators"); rows.emplace(key, std::move(row));
    }
    Json unlisted = Json::array();
    Map all_by_id;
    for (const auto& member : corpus.at("members")) all_by_id.emplace(text(member, "member_id"), member);
    for (const auto& [id, member] : selected) {
        static_cast<void>(member);
        if (declared_members.contains(id)) continue;
        const auto& index = all_by_id.at(id).at("latest_attempt");
        const auto reason = index.at("provider_id") != provider.at("provider_id") || index.at("family_id") != provider.at("family_id") ? "provider_not_declared" :
            !declarations.contains(text(index, "source_id")) ? "source_not_declared" : "locator_not_declared";
        unlisted.push_back({{"member_id", id}, {"family_id", index.at("family_id")}, {"provider_id", index.at("provider_id")},
            {"source_id", index.at("source_id")}, {"locator_id", index.at("locator_id")}, {"reason", reason}});
    }
    std::set<std::string> seen_wrappers, replayed_captures;
    Json unselected = Json::array();
    for (const auto& wrapper : input.at("interpretations")) {
        deadline(request); replay_wrapper(wrapper, request, domain);
        const auto wrapper_digest = text(wrapper, "digest");
        if (!seen_wrappers.insert(wrapper_digest).second) invalid("duplicate selected interpretation wrapper");
        Map captures;
        for (const auto& capture : wrapper.at("knowledge").at("captures")) captures.emplace(text(capture, "digest"), capture);
        // A metadata seal cannot borrow a real capture digest while changing its
        // source or body projection. Validate every selected supplied capture,
        // including selected members outside this provider's declaration.
        for (const auto& [digest, capture] : captures) {
            const auto chosen = selected_captures.find(digest);
            if (chosen == selected_captures.end()) continue;
            const auto& index = chosen->second.at("selected");
            const auto regenerated = handle_corpus(nested(request, "capture_index", {{"capture", capture}}), text(index, "domain"));
            if (regenerated != index) invalid("selected index does not match replayed capture bytes");
            const auto declared = declared_members.find(text(chosen->second, "member_id"));
            if (declared == declared_members.end()) continue;
            auto& row = rows.at(declared->second);
            const auto& desired = declarations.at(declared->second.first);
            const auto& actual = capture.at("source");
            Json differences = Json::array();
            for (const auto& [key, value] : desired.items()) if (actual.at(key) != value) differences.push_back(key);
            row["declaration_match"] = differences.empty() ? "matches" : "differs";
            row["declaration_difference_fields"] = differences;
            replayed_captures.insert(digest);
        }
        for (const auto& binding : wrapper.at("bindings")) {
            deadline(request);
            const auto p = text(binding, "profile_digest"), c = text(binding, "capture_digest");
            const auto chosen = selected_captures.find(c);
            const auto declared = chosen == selected_captures.end() ? declared_members.end() : declared_members.find(text(chosen->second, "member_id"));
            if (declared == declared_members.end()) {
                unselected.push_back({{"interpretation_digest", wrapper_digest}, {"profile_digest", p}, {"capture_digest", c},
                    {"reason", chosen == selected_captures.end() ? "capture_not_selected" : "selected_member_not_declared"}});
                continue;
            }
            auto& row = rows.at(declared->second);
            std::set<std::string> matched, unresolved;
            for (const auto& extraction : wrapper.at("extractions")) if (extraction.at("profile_digest") == p && extraction.at("capture_digest") == c) {
                const auto rule = text(extraction, "rule_id");
                if (extraction.at("status") == "matched") matched.insert(rule); else unresolved.insert(rule);
            }
            row["interpretations"].push_back({{"interpretation_digest", wrapper_digest}, {"profile_digest", p}, {"capture_digest", c},
                {"matched_rule_ids", matched}, {"unresolved_rule_ids", unresolved}});
            increment(summary, "selected_profile_bindings");
            summary["matched_rule_attempts"] = summary.at("matched_rule_attempts").get<std::size_t>() + matched.size();
            summary["unresolved_rule_attempts"] = summary.at("unresolved_rule_attempts").get<std::size_t>() + unresolved.size();
        }
    }
    Json sources = Json::array();
    for (auto& [key, row] : rows) { static_cast<void>(key); sort_bindings(row["interpretations"]); sources.push_back(std::move(row)); }
    sort_bindings(unselected);
    summary["selected_unlisted_members"] = unlisted.size();
    summary["replayed_selected_captures"] = replayed_captures.size();
    summary["unselected_profile_bindings"] = unselected.size();
    return sealed(Json{{"protocol", "symphony.scv.provider-coverage.v1"}, {"domain", domain}, {"input", input},
        {"corpus_query_result", query}, {"sources", sources}, {"unlisted_members", unlisted}, {"unselected_bindings", unselected}, {"summary", summary},
        {"limitations", Json::array({"coverage is relative to the caller's declared sources and exact selected corpus members; it is not provider-wide completeness",
            "corpus indexes describe captured evidence but do not prove retained body availability; supplied interpretation captures are replayed and matched to their selected indexes",
            "matched and unresolved rules count wrapper-bound extraction attempts, not unique concepts, semantic truth or independent corroboration",
            "declaration comparison uses supplied captured source fields, not a current source head or verified publisher authority",
            "extraction matching does not establish current claim support, runtime compatibility, account availability or permission; each wrapper retains its own evidence policy"})}});
}
}
Json handle_coverage(const engine::Request& request, const std::string& domain) {
    deadline(request);
    if (request.operation != "provider_coverage") throw engine::Error("operation.unsupported", "unsupported provider coverage operation", 4);
    auto result = coverage(request, domain); deadline(request); return result;
}
}
