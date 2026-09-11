#include "interpretation.hpp"
#include "knowledge.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"

#include <algorithm>
#include <charconv>
#include <cstdint>
#include <map>
#include <set>
#include <string_view>
#include <vector>

namespace symphony::knowledge::scv {
namespace {
using Map = std::map<std::string, Json>;
constexpr std::int64_t safe_integer = 9007199254740991LL;
[[noreturn]] void invalid(const std::string& message) { throw engine::Error("interpretation.invalid", message, 4); }
void deadline(const engine::Request& request) {
    if (engine::unix_time_ms() > request.deadline_unix_ms)
        throw engine::Error("request.deadline_exceeded", "interpretation operation deadline exceeded", 4);
}
void fields(const Json& value, std::initializer_list<std::string_view> names) {
    if (!value.is_object() || value.size() != names.size()) invalid("unexpected object fields");
    for (const auto name : names) if (!value.contains(std::string(name))) invalid("missing field: " + std::string(name));
}
std::string text(const Json& value, const std::string& key, std::size_t bound = 512) {
    if (!value.contains(key) || !value.at(key).is_string()) invalid("expected text: " + key);
    const auto result = value.at(key).get<std::string>();
    if (result.empty() || result.size() > bound || result.find('\0') != std::string::npos) invalid("text outside bounds: " + key);
    return result;
}
void array(const Json& value, std::size_t bound) { if (!value.is_array() || value.size() > bound) invalid("invalid bounded array"); }
void token(const Json& value, const std::string& key) {
    const auto s = text(value, key, 128);
    if (!std::all_of(s.begin(), s.end(), [](unsigned char c) {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' || c == ':';
    })) invalid("invalid source token: " + key);
}
void seal(const Json& value) {
    const auto digest = text(value, "digest", 71);
    if (digest.size() != 71 || !digest.starts_with("sha256:") || sealed(value) != value) invalid("invalid sealed artifact");
}
Json sorted(Json values) { std::sort(values.begin(), values.end(), [](const Json& a, const Json& b) { return a.dump() < b.dump(); }); return values; }
Json values(const Map& entries) { Json result = Json::array(); for (const auto& [key, value] : entries) { static_cast<void>(key); result.push_back(value); } return result; }
Json core(const engine::Request& request, const std::string& operation, Json payload, const std::string& domain) {
    auto nested = request; nested.operation = operation; nested.payload = std::move(payload);
    return handle_knowledge(nested, domain);
}
bool decimal(const std::string& s) {
    if (s.empty() || s.size() > 128 || s == "-0") return false;
    std::size_t start = s.front() == '-' ? 1 : 0;
    if (start == s.size()) return false;
    const auto dot = s.find('.', start), end = dot == std::string::npos ? s.size() : dot;
    if (end == start || (end - start > 1 && s[start] == '0')) return false;
    for (auto i = start; i < end; ++i) if (s[i] < '0' || s[i] > '9') return false;
    if (dot != std::string::npos) {
        if (dot + 1 == s.size() || s.back() == '0') return false;
        for (auto i = dot + 1; i < s.size(); ++i) if (s[i] < '0' || s[i] > '9') return false;
    }
    return true;
}
void typed_value(const Json& value) {
    fields(value, {"type", "value", "unit"});
    const auto type = text(value, "type");
    if (type == "integer") {
        const auto& number = value.at("value");
        if (!number.is_number_integer() || (number.is_number_unsigned() && number.get<std::uint64_t>() > static_cast<std::uint64_t>(safe_integer)) ||
            number.get<std::int64_t>() < -safe_integer || number.get<std::int64_t>() > safe_integer) invalid("invalid interoperable integer");
    } else if (type == "decimal") { if (!decimal(text(value, "value", 128))) invalid("noncanonical decimal"); }
    else if (type == "string" || type == "reference") static_cast<void>(text(value, "value", 4096));
    else if (type == "boolean") { if (!value.at("value").is_boolean()) invalid("boolean value required"); }
    else invalid("unsupported value type");
    if (!value.at("unit").is_null()) static_cast<void>(text(value, "unit", 128));
}
void scope(const Json& value) {
    if (!value.is_object() || value.size() > 16) invalid("invalid scope");
    for (const auto& [key, item] : value.items())
        if (key.empty() || key.size() > 128 || !item.is_string() || item.get_ref<const std::string&>().size() > 512) invalid("invalid scope qualifier");
}
Json rule_claim(const Json& rule, const Json& value) {
    Json claim = {{"value", value}, {"evidence", Json::array()}};
    for (const auto* key : {"claim_id", "subject", "predicate", "scope", "statement_kind", "dependencies"}) claim[key] = rule.at(key);
    return claim;
}
Json validate_profile(const Json& profile, std::set<std::string>& claim_ids, std::size_t& rule_count) {
    fields(profile, {"protocol", "profile_id", "profile_version", "provider_id", "source_id", "locator_id", "media_types", "authored_by", "rationale", "rules", "digest"});
    if (profile.at("protocol") != "symphony.scv.interpretation-profile.v1") invalid("unsupported interpretation profile");
    seal(profile);
    for (const auto* key : {"profile_id", "profile_version"}) static_cast<void>(text(profile, key));
    for (const auto* key : {"provider_id", "source_id", "locator_id"}) token(profile, key);
    text(profile, "authored_by", 1024); text(profile, "rationale", 4096);
    array(profile.at("media_types"), 16);
    if (profile.at("media_types").empty()) invalid("profile requires a media type");
    std::set<std::string> media;
    for (const auto& item : profile.at("media_types"))
        if (!media.insert(text(Json{{"media", item}}, "media", 128)).second) invalid("duplicate profile media type");
    array(profile.at("rules"), 128);
    std::set<std::string> rule_ids;
    Json validation = Json::array();
    for (const auto& rule : profile.at("rules")) {
        if (++rule_count > 128) invalid("selected profiles exceed 128 total rules");
        fields(rule, {"rule_id", "claim_id", "subject", "predicate", "scope", "statement_kind", "dependencies", "context", "extractor"});
        if (!rule_ids.insert(text(rule, "rule_id")).second || !claim_ids.insert(text(rule, "claim_id")).second) invalid("duplicate rule or generated claim identity");
        array(rule.at("context"), 8);
        for (const auto& context : rule.at("context")) text(Json{{"context", context}}, "context", 4096);
        const auto& extractor = rule.at("extractor");
        const auto kind = text(extractor, "kind");
        Json value;
        if (kind == "literal") {
            fields(extractor, {"kind", "quote", "value"}); text(extractor, "quote", 4096);
            value = extractor.at("value"); typed_value(value);
        } else if (kind == "delimited") {
            fields(extractor, {"kind", "prefix", "suffix", "type", "unit"});
            text(extractor, "prefix", 4096); text(extractor, "suffix", 4096);
            const auto type = text(extractor, "type");
            if (type != "integer" && type != "decimal" && type != "string") invalid("unsupported delimited token type");
            value = {{"type", type}, {"value", type == "integer" ? Json(0) : type == "decimal" ? Json("0") : Json("placeholder")}, {"unit", extractor.at("unit")}};
            typed_value(value);
        } else invalid("unsupported extractor kind");
        validation.push_back(rule_claim(rule, value));
    }
    return validation;
}
std::size_t occurrences(const std::string& body, const std::string& needle) {
    std::size_t count = 0, cursor = 0;
    while ((cursor = body.find(needle, cursor)) != std::string::npos) { if (++count >= 2) break; ++cursor; }
    return count;
}
Json token_value(const Json& extractor, const std::string& token_text) {
    const auto type = extractor.at("type").get<std::string>();
    Json value = {{"type", type}, {"unit", extractor.at("unit")}, {"value", token_text}};
    if (type == "integer") {
        if (!decimal(token_text) || token_text.find('.') != std::string::npos) invalid("invalid integer token");
        std::int64_t number = 0;
        const auto parsed = std::from_chars(token_text.data(), token_text.data() + token_text.size(), number);
        if (parsed.ec != std::errc{} || parsed.ptr != token_text.data() + token_text.size()) invalid("integer token overflow");
        value["value"] = number;
    }
    typed_value(value);
    return value;
}
Json interpretation(const Json& payload, const std::string& domain, const engine::Request& request) {
    fields(payload, {"captures", "profiles", "bindings", "selection_policy"});
    array(payload.at("captures"), 16); array(payload.at("profiles"), 16); array(payload.at("bindings"), 16);
    Map captures, profiles;
    for (const auto& capture : payload.at("captures")) {
        validate_capture(capture, domain);
        if (!captures.emplace(text(capture, "digest", 71), capture).second) invalid("duplicate selected capture");
    }
    std::set<std::string> claim_ids, profile_ids;
    std::size_t rule_count = 0;
    Json validation_claims = Json::array();
    for (const auto& profile : payload.at("profiles")) {
        deadline(request);
        const auto validated = validate_profile(profile, claim_ids, rule_count);
        if (!profiles.emplace(text(profile, "digest", 71), profile).second || !profile_ids.insert(text(profile, "profile_id")).second)
            invalid("duplicate selected profile identity");
        for (const auto& claim : validated) validation_claims.push_back(claim);
    }
    const auto normalized = core(request, "knowledge_interpret", {{"captures", values(captures)}, {"claims", validation_claims},
        {"interpreter_version", "profile-validation-v1"}, {"selection_policy", payload.at("selection_policy")}}, domain);
    Map bases;
    for (const auto& claim : normalized.at("claims")) bases.emplace(text(claim, "claim_id"), claim);
    const auto selected_policy = normalized.at("selection_policy");
    std::set<std::string> bound_profiles, bound_captures;
    const auto bindings = sorted(payload.at("bindings"));
    for (const auto& binding : bindings) {
        fields(binding, {"profile_digest", "capture_digest"});
        const auto p = text(binding, "profile_digest", 71), c = text(binding, "capture_digest", 71);
        if (!profiles.contains(p) || !captures.contains(c) || !bound_profiles.insert(p).second) invalid("binding must select one exact known profile and capture");
        bound_captures.insert(c);
        const auto& profile = profiles.at(p); const auto& capture = captures.at(c); const auto& source = capture.at("source");
        if (profile.at("provider_id") != source.at("provider_id") || profile.at("source_id") != source.at("source_id") || profile.at("locator_id") != capture.at("locator_id"))
            invalid("profile source/provider/locator differs from selected capture");
    }
    if (bound_profiles.size() != profiles.size() || bound_captures.size() != captures.size()) invalid("every profile and capture requires explicit binding");
    Json claims = Json::array(), extractions = Json::array();
    for (const auto& binding : bindings) {
        const auto p = binding.at("profile_digest").get<std::string>(), c = binding.at("capture_digest").get<std::string>();
        const auto& profile = profiles.at(p); const auto& capture = captures.at(c);
        const auto& body = capture.at("body").get_ref<const std::string&>();
        for (const auto& rule : profile.at("rules")) {
            deadline(request);
            std::set<std::string> reasons, anchors;
            const auto& extractor = rule.at("extractor");
            Json value = nullptr;
            if (capture.at("completeness") == "failed") reasons.insert("capture_failed");
            else if (capture.at("completeness") == "partial" && selected_policy.at("partial_capture") == "exclude") reasons.insert("partial_capture_excluded");
            else if (std::find(profile.at("media_types").begin(), profile.at("media_types").end(), capture.at("media_type")) == profile.at("media_types").end())
                reasons.insert("unsupported_media_type");
            else {
                std::size_t n = 0;
                for (const auto& context : rule.at("context")) {
                    const auto anchor = context.get<std::string>(); const auto count = occurrences(body, anchor);
                    if (count != 1) reasons.insert(std::string(count == 0 ? "context_missing:" : "context_ambiguous:") + std::to_string(n));
                    else anchors.insert(anchor);
                    ++n;
                }
                if (extractor.at("kind") == "literal") {
                    const auto quote = extractor.at("quote").get<std::string>(); const auto count = occurrences(body, quote);
                    if (count != 1) reasons.insert(count == 0 ? "value_missing" : "value_ambiguous");
                    else { anchors.insert(quote); value = extractor.at("value"); }
                } else {
                    const auto prefix = extractor.at("prefix").get<std::string>(), suffix = extractor.at("suffix").get<std::string>();
                    std::size_t cursor = 0, count = 0, begin = 0, end = 0;
                    while ((cursor = body.find(prefix, cursor)) != std::string::npos) {
                        const auto close = body.find(suffix, cursor + prefix.size());
                        if (close != std::string::npos) { begin = cursor; end = close; if (++count >= 2) break; }
                        ++cursor;
                    }
                    if (count != 1) reasons.insert(count == 0 ? "value_missing" : "value_ambiguous");
                    else if (end + suffix.size() - begin > 4096) reasons.insert("value_quote_exceeds_evidence_bound");
                    else {
                        const auto extracted = body.substr(begin + prefix.size(), end - begin - prefix.size());
                        try { value = token_value(extractor, extracted); anchors.insert(body.substr(begin, end + suffix.size() - begin)); }
                        catch (const engine::Error&) { reasons.insert("invalid_token"); }
                    }
                }
            }
            const bool matched = reasons.empty();
            if (matched) {
                auto claim = bases.at(rule.at("claim_id").get<std::string>()); claim["value"] = value;
                Json evidence = Json::array();
                for (const auto& anchor : anchors) evidence.push_back({{"capture_digest", c}, {"quote", anchor}});
                claim["evidence"] = evidence; claims.push_back(claim);
                if (capture.at("completeness") == "partial") reasons.insert("partial_capture_qualified");
            }
            extractions.push_back({{"profile_digest", p}, {"capture_digest", c}, {"rule_id", rule.at("rule_id")}, {"claim_id", rule.at("claim_id")},
                {"status", matched ? "matched" : "unresolved"}, {"reasons", reasons}});
        }
    }
    Json profile_digests = Json::array(); for (const auto& [id, profile] : profiles) { static_cast<void>(profile); profile_digests.push_back(id); }
    const auto identity = "profile-extraction-v1:" + engine::tagged_sha256(profile_digests.dump());
    auto knowledge = core(request, "knowledge_interpret", {{"captures", values(captures)}, {"claims", claims},
        {"interpreter_version", identity}, {"selection_policy", selected_policy}}, domain);
    return sealed(Json{{"protocol", "symphony.scv.provider-interpretation.v1"}, {"domain", domain}, {"profiles", values(profiles)},
        {"bindings", bindings}, {"knowledge", knowledge}, {"extractions", sorted(extractions)},
        {"limitations", Json::array({"profiles are explicit authored mappings; anchored occurrence is not semantic or empirical verification",
            "interpretation covers selected rules and contextual fragments only; changes outside them remain unassessed",
            "failed and unresolved extraction produces no new claim; prior results are not silently refreshed"})}});
}
Json replay_interpretation(const Json& value, const std::string& domain, const engine::Request& request) {
    fields(value, {"protocol", "domain", "profiles", "bindings", "knowledge", "extractions", "limitations", "digest"});
    if (value.at("protocol") != "symphony.scv.provider-interpretation.v1") invalid("unsupported interpretation wrapper");
    seal(value);
    const auto owner = text(value, "domain");
    const auto& knowledge = value.at("knowledge");
    if (!knowledge.is_object() || !knowledge.contains("captures") || !knowledge.contains("selection_policy")) invalid("missing retained knowledge input");
    array(knowledge.at("captures"), 16);
    for (const auto& capture : knowledge.at("captures")) validate_capture(capture, domain);
    const auto replay = interpretation({{"captures", knowledge.at("captures")}, {"profiles", value.at("profiles")},
        {"bindings", value.at("bindings")}, {"selection_policy", knowledge.at("selection_policy")}}, owner, request);
    if (replay != value) invalid("interpretation wrapper cannot be exactly replayed");
    return value.at("knowledge");
}
void reference(const Json& value, bool right = false) {
    if (right) fields(value, {"kind", "claim_id", "subject", "scope"});
    else fields(value, {"claim_id", "subject", "scope"});
    text(value, "claim_id"); text(value, "subject"); scope(value.at("scope"));
}
Map connections(const Json& input) {
    array(input, 32);
    Map result;
    std::size_t count = 0;
    for (const auto& connection : input) {
        fields(connection, {"connection_id", "from_subject", "to_subject", "checks"});
        const auto id = text(connection, "connection_id"); text(connection, "from_subject"); text(connection, "to_subject");
        array(connection.at("checks"), 128);
        std::set<std::string> check_ids;
        for (const auto& check : connection.at("checks")) {
            if (++count > 128) invalid("connections exceed 128 total checks");
            fields(check, {"check_id", "importance", "left", "operator", "right"});
            if (!check_ids.insert(text(check, "check_id")).second) invalid("duplicate check identity");
            const auto importance = text(check, "importance"), op = text(check, "operator");
            if (importance != "required" && importance != "optional") invalid("unknown check importance");
            if (op != "eq" && op != "gte" && op != "lte") invalid("unknown comparison operator");
            reference(check.at("left"));
            const auto& right = check.at("right"); const auto kind = text(right, "kind");
            if (kind == "claim") reference(right, true);
            else if (kind == "literal") { fields(right, {"kind", "value"}); typed_value(right.at("value")); }
            else invalid("unknown right operand kind");
        }
        if (!result.emplace(id, connection).second) invalid("duplicate connection identity");
    }
    return result;
}
int numeric_compare(const Json& a, const Json& b) {
    if (a.at("type") == "integer") {
        const auto x = a.at("value").get<std::int64_t>(), y = b.at("value").get<std::int64_t>();
        return x < y ? -1 : x > y ? 1 : 0;
    }
    auto x = a.at("value").get<std::string>(), y = b.at("value").get<std::string>();
    const bool negative_x = x.front() == '-', negative_y = y.front() == '-';
    if (negative_x != negative_y) return negative_x ? -1 : 1;
    if (negative_x) { x.erase(0, 1); y.erase(0, 1); }
    const auto split = [](const std::string& s) {
        const auto dot = s.find('.'); return std::pair{s.substr(0, dot), dot == std::string::npos ? std::string{} : s.substr(dot + 1)};
    };
    auto [xi, xf] = split(x); auto [yi, yf] = split(y);
    int result = 0;
    if (xi.size() != yi.size()) result = xi.size() < yi.size() ? -1 : 1;
    else if (xi != yi) result = xi < yi ? -1 : 1;
    else { const auto width = std::max(xf.size(), yf.size()); xf.resize(width, '0'); yf.resize(width, '0'); result = xf < yf ? -1 : xf > yf ? 1 : 0; }
    return negative_x ? -result : result;
}
Map findings(const Json& evaluation) { Map result; for (const auto& finding : evaluation.at("findings")) result.emplace(text(finding.at("claim"), "claim_id"), finding); return result; }
Json checked_connection(const Json& connection, const Map& evaluated) {
    Map checks;
    for (const auto& check : connection.at("checks")) checks.emplace(text(check, "check_id"), check);
    Json results = Json::array();
    std::set<std::string> required;
    for (const auto& [id, check] : checks) {
        static_cast<void>(id);
        std::set<std::string> reasons, claim_ids;
        bool conditional = false, unresolved = false;
        const auto operand = [&](const Json& ref) -> Json {
            const auto claim_id = ref.at("claim_id").get<std::string>(); claim_ids.insert(claim_id);
            const auto found = evaluated.find(claim_id);
            if (found == evaluated.end()) { unresolved = true; reasons.insert("claim_missing:" + claim_id); return nullptr; }
            const auto& finding = found->second; const auto& claim = finding.at("claim");
            if (claim.at("subject") != ref.at("subject")) { unresolved = true; reasons.insert("subject_mismatch:" + claim_id); }
            if (claim.at("scope") != ref.at("scope")) { unresolved = true; reasons.insert("scope_mismatch:" + claim_id); }
            if (finding.at("status") != "supported" && finding.at("status") != "conditional") {
                unresolved = true; reasons.insert("claim_not_eligible:" + claim_id + ":" + finding.at("status").get<std::string>());
            }
            if (finding.at("status") == "conditional" || (claim.at("statement_kind") != "documented_fact" && claim.at("statement_kind") != "requirement")) {
                conditional = true; reasons.insert("conditional_evidence:" + claim_id);
            }
            return claim.at("value");
        };
        const auto left = operand(check.at("left"));
        const auto right = check.at("right").at("kind") == "claim" ? operand(check.at("right")) : check.at("right").at("value");
        Json comparison = nullptr;
        if (!left.is_null() && !right.is_null()) {
            if (left.at("type") != right.at("type")) { unresolved = true; reasons.insert("value_type_mismatch"); }
            else if (left.at("unit") != right.at("unit")) { unresolved = true; reasons.insert("unit_mismatch"); }
            else if (check.at("operator") != "eq" && left.at("type") != "integer" && left.at("type") != "decimal") {
                unresolved = true; reasons.insert("ordering_requires_numeric_type");
            }
            if (!unresolved) {
                const auto op = check.at("operator").get<std::string>();
                const bool matched = op == "eq" ? left.at("value") == right.at("value") : op == "gte" ? numeric_compare(left, right) >= 0 : numeric_compare(left, right) <= 0;
                comparison = matched; reasons.insert(matched ? "comparison_matched" : "comparison_not_matched");
            }
        }
        const auto status = unresolved ? "unresolved" : conditional ? "conditional" : comparison == true ? "satisfied" : "contradicted";
        if (check.at("importance") == "required") required.insert(status);
        results.push_back({{"specification", check}, {"status", status}, {"reasons", reasons}, {"claim_ids", claim_ids}, {"comparison", comparison}});
    }
    const auto status = required.empty() ? "unresolved" : required.contains("contradicted") ? "contradicted" : required.contains("unresolved") ? "unresolved" : required.contains("conditional") ? "conditional" : "satisfied";
    return {{"connection_id", connection.at("connection_id")}, {"from_subject", connection.at("from_subject")},
        {"to_subject", connection.at("to_subject")}, {"status", status}, {"checks", results}};
}
Json evaluate(const Json& payload, const std::string& domain, const engine::Request& request) {
    fields(payload, {"interpretations", "additional_knowledge", "query_time", "connections"});
    array(payload.at("interpretations"), 16); array(payload.at("additional_knowledge"), 16);
    if (payload.at("interpretations").size() + payload.at("additional_knowledge").size() > 16) invalid("combined knowledge exceeds 16 artifacts");
    const auto selected_connections = connections(payload.at("connections"));
    Json knowledge = Json::array();
    std::set<std::string> profile_ids;
    for (const auto& wrapper : payload.at("interpretations")) {
        deadline(request);
        knowledge.push_back(replay_interpretation(wrapper, domain, request));
        for (const auto& profile : wrapper.at("profiles"))
            if (!profile_ids.insert(text(profile, "profile_id")).second) invalid("profile_id must be unique across selected interpretation wrappers");
    }
    for (const auto& extra : payload.at("additional_knowledge")) knowledge.push_back(extra);
    const auto graph = core(request, "graph_build", {{"knowledge", knowledge}}, domain);
    const auto assessment = core(request, "graph_evaluate", {{"graph", graph}, {"query_time", payload.at("query_time")}}, domain);
    const auto evaluated = findings(assessment);
    Json results = Json::array();
    for (const auto& [id, connection] : selected_connections) { static_cast<void>(id); deadline(request); results.push_back(checked_connection(connection, evaluated)); }
    return sealed(Json{{"protocol", "symphony.scv.connection-evaluation.v1"}, {"domain", domain}, {"input", payload},
        {"graph_digest", graph.at("digest")}, {"graph_evaluation", assessment}, {"connections", results},
        {"limitations", Json::array({"satisfied means selected documented checks match; live route, account, permissions and payload agreement are not thereby tested",
            "endpoint labels and requirements belong to the caller; optional checks do not impose a topology or deployment policy",
            "recommendations, observations and assumptions remain conditional; source attribution is not empirical proof"})}});
}
void replay_evaluation(const Json& value, const std::string& domain, const engine::Request& request) {
    fields(value, {"protocol", "domain", "input", "graph_digest", "graph_evaluation", "connections", "limitations", "digest"});
    if (value.at("protocol") != "symphony.scv.connection-evaluation.v1" || value.at("domain") != domain) invalid("evaluation protocol/domain mismatch");
    seal(value);
    if (evaluate(value.at("input"), domain, request) != value) invalid("connection evaluation cannot be exactly replayed");
}
Json capture_selection(const Json& evaluation) {
    std::set<std::string> result;
    Json bindings = Json::array();
    for (const auto& wrapper : evaluation.at("input").at("interpretations")) {
        for (const auto& capture : wrapper.at("knowledge").at("captures")) result.insert(text(capture, "digest", 71));
        Map profile_ids;
        for (const auto& profile : wrapper.at("profiles")) profile_ids.emplace(text(profile, "digest", 71), profile.at("profile_id"));
        for (const auto& binding : wrapper.at("bindings")) bindings.push_back({{"profile_id", profile_ids.at(binding.at("profile_digest").get<std::string>())},
            {"capture_digest", binding.at("capture_digest")}});
    }
    for (const auto& knowledge : evaluation.at("input").at("additional_knowledge"))
        for (const auto& capture : knowledge.at("captures")) result.insert(text(capture, "digest", 71));
    return {{"captures", result}, {"bindings", sorted(bindings)}};
}
Json profile_selection(const Json& evaluation) {
    std::set<std::string> result;
    for (const auto& wrapper : evaluation.at("input").at("interpretations"))
        for (const auto& profile : wrapper.at("profiles")) result.insert(text(profile, "digest", 71));
    return result;
}
Json additional_selection(const Json& evaluation) {
    std::set<std::string> result; for (const auto& knowledge : evaluation.at("input").at("additional_knowledge")) result.insert(text(knowledge, "digest", 71)); return result;
}
Json requirements(const Json& evaluation) {
    Json result = Json::array();
    for (const auto& [id, connection] : connections(evaluation.at("input").at("connections"))) {
        static_cast<void>(id); auto normalized = connection;
        normalized["checks"] = sorted(normalized.at("checks")); result.push_back(normalized);
    }
    return result;
}
Map check_map(const Json& evaluation) {
    Map result;
    for (const auto& connection : evaluation.at("connections")) for (const auto& check : connection.at("checks"))
        result.emplace(Json::array({connection.at("connection_id"), check.at("specification").at("check_id")}).dump(), check);
    return result;
}
void dependency_closure(const Map& evaluated, const std::string& id, std::set<std::string>& visited) {
    if (!visited.insert(id).second) return;
    const auto found = evaluated.find(id); if (found == evaluated.end()) return;
    const auto& claim = found->second.at("claim");
    const auto visit = [&](const Json& deps) { for (const auto& dep : deps) dependency_closure(evaluated, dep.at("claim_id").get<std::string>(), visited); };
    visit(claim.at("dependencies"));
    for (const auto& support : claim.at("alternative_supports")) visit(support.at("dependencies"));
}
Json reassess(const Json& payload, const std::string& domain, const engine::Request& request) {
    fields(payload, {"before", "after"});
    const auto& before = payload.at("before"); const auto& after = payload.at("after");
    replay_evaluation(before, domain, request); replay_evaluation(after, domain, request);
    const auto a = check_map(before), b = check_map(after);
    const auto old_findings = findings(before.at("graph_evaluation")), new_findings = findings(after.at("graph_evaluation"));
    std::set<std::string> keys; for (const auto& [key, unused] : a) { static_cast<void>(unused); keys.insert(key); }
    for (const auto& [key, unused] : b) { static_cast<void>(unused); keys.insert(key); }
    Json checks = Json::array();
    for (const auto& key : keys) {
        deadline(request);
        const auto old = a.contains(key) ? a.at(key) : Json(nullptr), current = b.contains(key) ? b.at(key) : Json(nullptr);
        const bool changed = old != current;
        bool affected = changed;
        std::set<std::string> closure;
        for (const auto* check : {&old, &current}) if (!check->is_null()) for (const auto& id : check->at("claim_ids")) {
            std::set<std::string> old_closure, new_closure;
            dependency_closure(old_findings, id.get<std::string>(), old_closure);
            dependency_closure(new_findings, id.get<std::string>(), new_closure);
            closure.insert(old_closure.begin(), old_closure.end());
            closure.insert(new_closure.begin(), new_closure.end());
        }
        for (const auto& id : closure) {
            const auto previous = old_findings.contains(id) ? old_findings.at(id) : Json(nullptr);
            const auto latest = new_findings.contains(id) ? new_findings.at(id) : Json(nullptr);
            if (previous != latest) affected = true;
        }
        const auto identity = Json::parse(key);
        checks.push_back({{"connection_id", identity.at(0)}, {"check_id", identity.at(1)},
            {"before_status", old.is_null() ? Json(nullptr) : old.at("status")}, {"after_status", current.is_null() ? Json(nullptr) : current.at("status")},
            {"changed", changed}, {"affected", affected}});
    }
    return sealed(Json{{"protocol", "symphony.scv.connection-reassessment.v1"}, {"domain", domain}, {"before_digest", before.at("digest")}, {"after_digest", after.at("digest")},
        {"change_axes", {{"captures", capture_selection(before) != capture_selection(after)}, {"profiles", profile_selection(before) != profile_selection(after)},
            {"additional_knowledge", additional_selection(before) != additional_selection(after)},
            {"selection_policy", before.at("graph_evaluation").at("selection_policy") != after.at("graph_evaluation").at("selection_policy")},
            {"requirements", requirements(before) != requirements(after)}, {"query_time", before.at("input").at("query_time") != after.at("input").at("query_time")}}},
        {"checks", checks}, {"limitations", Json::array({"change axes identify differing inputs, not counterfactual causes; simultaneous changes are not attributed solely to the provider",
            "affected checks include exact referenced evidence and dependency changes; old results and user selections remain unchanged"})}});
}
}
Json handle_interpretation(const engine::Request& request, const std::string& domain) {
    deadline(request); Json result;
    if (request.operation == "provider_interpret") result = interpretation(request.payload, domain, request);
    else if (request.operation == "connection_evaluate") result = evaluate(request.payload, domain, request);
    else if (request.operation == "connection_reassess") result = reassess(request.payload, domain, request);
    else throw engine::Error("operation.unsupported", "unsupported provider interpretation operation", 4);
    deadline(request); return result;
}
}
