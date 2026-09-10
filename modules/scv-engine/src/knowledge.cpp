#include "knowledge.hpp"
#include "scv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/temporal.hpp"

#include <algorithm>
#include <chrono>
#include <cstdint>
#include <functional>
#include <map>
#include <regex>
#include <set>
#include <sstream>
#include <string_view>
#include <utility>
#include <vector>

namespace symphony::knowledge::scv {
namespace {
using engine::Json;
constexpr std::size_t max_captures = 16;
constexpr std::size_t max_claims = 128;
constexpr std::size_t max_nodes = 512;
constexpr std::size_t max_edges = 1024;
const std::set<std::string> kinds = {"documented_fact", "requirement", "recommendation",
    "observation", "user_assertion", "inference", "hypothesis"};

[[noreturn]] void invalid(const std::string& message) {
    throw engine::Error("knowledge.invalid", message, 4);
}
void check_deadline(const engine::Request& request) {
    if (engine::unix_time_ms() > request.deadline_unix_ms) {
        throw engine::Error("request.deadline_exceeded", "knowledge operation deadline exceeded", 4);
    }
}
void fields(const Json& value, const std::set<std::string>& required,
            const std::set<std::string>& optional = {}) {
    if (!value.is_object()) invalid("expected object");
    for (const auto& key : required) if (!value.contains(key)) invalid("missing field: " + key);
    for (const auto& [key, unused] : value.items()) {
        static_cast<void>(unused);
        if (!required.contains(key) && !optional.contains(key)) invalid("unknown field: " + key);
    }
}
std::string string_at(const Json& value, const std::string& key, std::size_t limit = 512) {
    if (!value.contains(key) || !value.at(key).is_string()) invalid("expected string: " + key);
    const auto result = value.at(key).get<std::string>();
    if (result.empty() || result.size() > limit || result.find('\0') != std::string::npos)
        invalid("empty, oversized or NUL-bearing string: " + key);
    return result;
}
void array_bound(const Json& value, std::size_t bound, const std::string& name) {
    if (!value.is_array() || value.size() > bound) invalid("invalid or oversized array: " + name);
}
void unique_strings(const Json& value, std::size_t bound, const std::string& name) {
    array_bound(value, bound, name);
    std::set<std::string> seen;
    for (const auto& entry : value) {
        if (!entry.is_string() || entry.get_ref<const std::string&>().empty() ||
            entry.get_ref<const std::string&>().size() > 512 ||
            !seen.insert(entry.get<std::string>()).second) invalid("invalid or duplicate " + name);
    }
}
Json sorted_array(Json value) {
    std::sort(value.begin(), value.end(), [](const Json& a, const Json& b) { return a.dump() < b.dump(); });
    return value;
}
void verify_digest(const Json& value) {
    const auto digest = string_at(value, "digest", 71);
    auto unsealed = value;
    unsealed.erase("digest");
    if (engine::tagged_sha256(unsealed.dump()) != digest) invalid("digest mismatch");
}
std::int64_t utc_seconds(const Json& value) {
    if (!value.is_string() || !engine::is_utc_seconds(value.get_ref<const std::string&>()))
        invalid("expected canonical STSC whole-second UTC");
    const auto& s = value.get_ref<const std::string&>();
    const auto year = std::stoi(s.substr(0, 4));
    const auto month = static_cast<unsigned>(std::stoi(s.substr(5, 2)));
    const auto day = static_cast<unsigned>(std::stoi(s.substr(8, 2)));
    const auto date = std::chrono::sys_days(std::chrono::year(year) / month / day);
    return std::chrono::duration_cast<std::chrono::seconds>(date.time_since_epoch()).count() +
        std::stoi(s.substr(11, 2)) * 3600 + std::stoi(s.substr(14, 2)) * 60 + std::stoi(s.substr(17, 2));
}
Json policy(const Json& input) {
    fields(input, {"policy_id", "max_age_seconds", "allowed_statement_kinds", "partial_capture"});
    static_cast<void>(string_at(input, "policy_id"));
    const auto& age = input.at("max_age_seconds");
    if (!age.is_null() && (!age.is_number_integer() || age.get<std::int64_t>() < 0 ||
        age.get<std::int64_t>() > 3155760000LL)) invalid("max_age_seconds must be null or 0..3155760000");
    unique_strings(input.at("allowed_statement_kinds"), kinds.size(), "allowed_statement_kinds");
    for (const auto& kind : input.at("allowed_statement_kinds"))
        if (!kinds.contains(kind.get<std::string>())) invalid("unknown statement kind");
    const auto partial = string_at(input, "partial_capture");
    if (partial != "exclude" && partial != "include_qualified") invalid("unsupported partial_capture policy");
    auto result = input;
    result["allowed_statement_kinds"] = sorted_array(result["allowed_statement_kinds"]);
    return result;
}
using CaptureMap = std::map<std::string, Json>;
CaptureMap captures(const Json& input, const std::string& domain) {
    array_bound(input, max_captures, "captures");
    CaptureMap result;
    std::map<std::string, Json> source_owners;
    for (const auto& capture : input) {
        validate_capture(capture, domain);
        const auto& source = capture.at("source");
        const auto owner = Json::array({source.at("family_id"), source.at("provider_id")});
        const auto [prior, inserted] = source_owners.emplace(source.at("source_id").get<std::string>(), owner);
        if (!inserted && prior->second != owner) invalid("source_id is ambiguous across owner domains");
        const auto digest = string_at(capture, "digest", 71);
        if (!result.emplace(digest, capture).second) invalid("duplicate selected capture");
    }
    return result;
}
Json map_values(const CaptureMap& values) {
    auto result = Json::array();
    for (const auto& [key, value] : values) { static_cast<void>(key); result.push_back(value); }
    return result;
}
Json evidence(const Json& input, const CaptureMap& selected) {
    array_bound(input, 16, "evidence");
    std::set<std::string> seen;
    for (const auto& item : input) {
        fields(item, {"capture_digest", "quote"});
        const auto digest = string_at(item, "capture_digest", 71);
        const auto quote = string_at(item, "quote", 4096);
        const auto found = selected.find(digest);
        if (found == selected.end()) invalid("evidence references an unselected capture");
        if (found->second.at("body").get_ref<const std::string&>().find(quote) == std::string::npos)
            invalid("evidence quote not found in captured bytes");
        if (!seen.insert(item.dump()).second) invalid("duplicate evidence anchor");
    }
    return sorted_array(input);
}
Json dependencies(const Json& input) {
    array_bound(input, 16, "dependencies");
    std::set<std::string> seen;
    for (const auto& item : input) {
        fields(item, {"claim_id", "role"});
        static_cast<void>(string_at(item, "claim_id"));
        const auto role = string_at(item, "role");
        if (role != "support" && role != "requires" && role != "scope") invalid("unsupported dependency role");
        if (!seen.insert(item.dump()).second) invalid("duplicate dependency");
    }
    return sorted_array(input);
}
Json normalize_claim(const Json& input, const CaptureMap& selected) {
    fields(input, {"claim_id", "subject", "predicate", "value", "scope", "statement_kind", "evidence", "dependencies"},
        {"valid_from", "valid_until", "alternative_supports", "scope_dependencies"});
    for (const auto* name : {"claim_id", "subject", "predicate"}) static_cast<void>(string_at(input, name));
    if (!kinds.contains(string_at(input, "statement_kind"))) invalid("unknown statement_kind");
    const auto& value = input.at("value");
    fields(value, {"type", "value"}, {"unit"});
    const auto type = string_at(value, "type");
    if (type == "string" || type == "reference") static_cast<void>(string_at(value, "value", 4096));
    else if (type == "boolean") { if (!value.at("value").is_boolean()) invalid("boolean value required"); }
    else if (type == "integer") {
        if (!value.at("value").is_number_integer() ||
            (value.at("value").is_number_unsigned() && value.at("value").get<std::uint64_t>() > 9007199254740991ULL) ||
            value.at("value").get<std::int64_t>() < -9007199254740991LL ||
            value.at("value").get<std::int64_t>() > 9007199254740991LL) invalid("interoperable integer required");
    } else if (type == "decimal") {
        const auto exact = string_at(value, "value", 128);
        static const std::regex decimal("-?(0|[1-9][0-9]*)(\\.[0-9]*[1-9])?");
        if (exact == "-0" || !std::regex_match(exact, decimal)) invalid("canonical exact decimal string required");
    } else invalid("unsupported value type");
    if (value.contains("unit") && !value.at("unit").is_null()) static_cast<void>(string_at(value, "unit", 128));
    if (!input.at("scope").is_object() || input.at("scope").size() > 16) invalid("invalid claim scope");
    for (const auto& [key, item] : input.at("scope").items()) {
        if (key.empty() || key.size() > 128 || !item.is_string() || item.get_ref<const std::string&>().size() > 512)
            invalid("scope must contain bounded string qualifiers");
    }
    auto result = input;
    if (!result["value"].contains("unit")) result["value"]["unit"] = nullptr;
    for (const auto* time : {"valid_from", "valid_until"}) {
        if (!result.contains(time)) result[time] = nullptr;
        if (!result.at(time).is_null()) static_cast<void>(utc_seconds(result.at(time)));
    }
    if (!result.at("valid_from").is_null() && !result.at("valid_until").is_null() &&
        utc_seconds(result.at("valid_from")) >= utc_seconds(result.at("valid_until"))) invalid("empty validity interval");
    result["evidence"] = evidence(input.at("evidence"), selected);
    result["dependencies"] = dependencies(input.at("dependencies"));
    if (!result.contains("alternative_supports")) result["alternative_supports"] = Json::array();
    array_bound(result.at("alternative_supports"), 7, "alternative_supports");
    std::set<std::string> support_ids = {"primary"};
    for (auto& support : result["alternative_supports"]) {
        fields(support, {"support_id", "evidence", "dependencies"});
        if (!support_ids.insert(string_at(support, "support_id")).second) invalid("duplicate support_id");
        support["evidence"] = evidence(support.at("evidence"), selected);
        support["dependencies"] = dependencies(support.at("dependencies"));
    }
    result["alternative_supports"] = sorted_array(result.at("alternative_supports"));
    if (!result.contains("scope_dependencies")) result["scope_dependencies"] = Json::array();
    array_bound(result.at("scope_dependencies"), 16, "scope_dependencies");
    std::set<std::string> scope_ids;
    for (auto& scope : result["scope_dependencies"]) {
        fields(scope, {"source_id", "capture_digests"});
        if (!scope_ids.insert(string_at(scope, "source_id")).second) invalid("duplicate source scope dependency");
        unique_strings(scope.at("capture_digests"), max_captures, "capture_digests");
        for (const auto& digest : scope.at("capture_digests")) {
            static const std::regex sha("sha256:[0-9a-f]{64}");
            if (!std::regex_match(digest.get<std::string>(), sha)) invalid("invalid scope capture digest");
        }
        scope["capture_digests"] = sorted_array(scope.at("capture_digests"));
    }
    result["scope_dependencies"] = sorted_array(result.at("scope_dependencies"));
    return result;
}
CaptureMap claims(const Json& input, const CaptureMap& selected) {
    array_bound(input, max_claims, "claims");
    CaptureMap result;
    for (const auto& item : input) {
        const auto claim = normalize_claim(item, selected);
        if (!result.emplace(claim.at("claim_id").get<std::string>(), claim).second) invalid("duplicate claim_id");
    }
    return result;
}
Json support_sets(const Json& claim) {
    auto result = Json::array({Json{{"support_id", "primary"}, {"evidence", claim.at("evidence")},
                                  {"dependencies", claim.at("dependencies")}}});
    for (const auto& support : claim.at("alternative_supports")) result.push_back(support);
    return result;
}

struct Native final {
    CaptureMap nodes;
    CaptureMap edges;
    std::set<std::string> limitations = {
        "native extraction describes captured structure; it does not validate provider semantics",
        "caller-proposed claims retain their statement kinds; quote anchoring is not factual verification",
        "coverage is limited to selected captures; absence is not provider-wide unavailability"};

    std::string node(const std::string& kind, const std::string& capture,
                     const std::string& pointer, const Json& attributes) {
        Json value = {{"kind", kind}, {"capture_digest", capture}, {"pointer", pointer}, {"attributes", attributes}};
        const auto id = "native:" + engine::sha256_hex(value.dump());
        value["node_id"] = id;
        if (nodes.size() >= max_nodes && !nodes.contains(id)) {
            limitations.insert("native_node_limit: extraction is partial");
            return {};
        }
        nodes.emplace(id, std::move(value));
        return id;
    }
    void edge(const std::string& from, const std::string& to, const std::string& relation) {
        if (from.empty() || to.empty()) return;
        const Json value = {{"from", from}, {"to", to}, {"relation", relation}};
        const auto key = value.dump();
        if (edges.size() >= max_edges && !edges.contains(key)) {
            limitations.insert("native_edge_limit: extraction is partial");
            return;
        }
        edges.emplace(key, value);
    }
};
std::string pointer_escape(const std::string& text) {
    std::string result;
    for (const auto c : text) {
        if (c == '~') result += "~0";
        else if (c == '/') result += "~1";
        else result += c;
    }
    return result;
}
Json parse_native_json(const std::string& body) {
    std::map<int, std::set<std::string>> keys;
    const auto callback = [&keys](int depth, Json::parse_event_t event, Json& parsed) {
        if (depth > 32) invalid("native JSON depth exceeds 32");
        if (event == Json::parse_event_t::object_start) keys[depth + 1].clear();
        if (event == Json::parse_event_t::key && !keys[depth].insert(parsed.get<std::string>()).second)
            invalid("native JSON duplicate key");
        return true;
    };
    return Json::parse(body, callback);
}
Native native_graph(const CaptureMap& selected) {
    Native native;
    for (const auto& [digest, capture] : selected) {
        const auto source_node = native.node("source", digest, "", {
            {"source_id", capture.at("source").at("source_id")},
            {"publisher", capture.at("source").at("publisher")},
            {"authority_role", capture.at("source").at("authority_role")},
            {"scope", capture.at("source").at("scope")},
            {"resolved_uri", capture.at("resolved_uri")},
            {"completeness", capture.at("completeness")}});
        if (capture.at("completeness") == "failed") {
            native.limitations.insert("failed capture retained without native interpretation: " + digest);
            continue;
        }
        if (capture.at("completeness") == "partial") native.limitations.insert("partial capture: " + digest);
        const auto media = capture.at("media_type").get<std::string>();
        const auto& body = capture.at("body").get_ref<const std::string&>();
        if (media == "text/markdown" || media == "text/plain") {
            std::istringstream lines(body);
            std::string line;
            std::size_t number = 0;
            static const std::regex heading("^#{1,6} +(.+)$");
            while (std::getline(lines, line)) {
                ++number;
                if (native.nodes.size() >= max_nodes) {
                    native.limitations.insert("native_node_limit: extraction is partial"); break;
                }
                std::smatch match;
                if (std::regex_match(line, match, heading)) {
                    const auto term = match[1].str();
                    if (term.size() <= 4096) native.edge(source_node,
                        native.node("section", digest, "line:" + std::to_string(number), {{"term", term}}), "contains");
                    else native.limitations.insert("oversized heading retained but not extracted: " + digest);
                }
                std::size_t cursor = 0;
                while (cursor < line.size()) {
                    const auto start = line.find('[', cursor);
                    if (start == std::string::npos) break;
                    const auto middle = line.find("](", start + 1);
                    if (middle == std::string::npos) break;
                    const auto end = line.find(')', middle + 2);
                    if (end == std::string::npos) break;
                    cursor = end + 1;
                    const auto label = line.substr(start + 1, middle - start - 1);
                    const auto target = line.substr(middle + 2, end - middle - 2);
                    if (label.empty() || label.size() > 256 || target.empty() || target.size() > 2048 ||
                        target.find_first_of(" \t\r\n") != std::string::npos) continue;
                    native.edge(source_node, native.node("reference", digest,
                        "line:" + std::to_string(number) + ":byte:" + std::to_string(start),
                        {{"label", label}, {"target", target}, {"retrieved", false}}), "links_to");
                }
            }
            native.limitations.insert("text extraction is limited to headings and Markdown links; prose meaning remains proposed");
        } else if (media == "application/json" || media == "application/schema+json" || media == "application/vnd.oai.openapi+json") {
            Json document;
            try { document = parse_native_json(body); }
            catch (const std::exception&) {
                native.limitations.insert("invalid_or_unsupported_native_json: " + digest); continue;
            }
            if (!document.is_object()) { native.limitations.insert("unsupported native JSON root: " + digest); continue; }
            bool supported = false;
            if (document.contains("openapi") && document.at("openapi").is_string()) {
                const auto version = document.at("openapi").get<std::string>();
                static const std::regex profile("3\\.(0|1|2)\\.[0-9]+");
                if (std::regex_match(version, profile)) {
                    supported = true;
                    Json attributes = {{"description_language_version", version}};
                    if (document.contains("info") && document.at("info").is_object()) {
                        for (const auto* field : {"title", "version"})
                            if (document.at("info").contains(field) && document.at("info").at(field).is_string())
                                attributes[std::string("info_") + field] = document.at("info").at(field);
                    }
                    const auto api = native.node("openapi_document", digest, "", attributes);
                    native.edge(source_node, api, "describes");
                    if (document.contains("paths") && document.at("paths").is_object()) {
                        const std::set<std::string> methods = {"get", "put", "post", "delete", "options", "head", "patch", "trace", "query"};
                        for (const auto& [path, path_item] : document.at("paths").items()) {
                            if (!path_item.is_object()) continue;
                            for (const auto& [method, operation] : path_item.items()) {
                                if (!methods.contains(method) || !operation.is_object()) continue;
                                Json attrs = {{"path", path}, {"method", method}};
                                for (const auto* field : {"operationId", "summary"})
                                    if (operation.contains(field) && operation.at(field).is_string()) attrs[field] = operation.at(field);
                                native.edge(api, native.node("api_operation", digest,
                                    "/paths/" + pointer_escape(path) + "/" + method, attrs), "declares");
                            }
                        }
                    }
                    native.limitations.insert("OpenAPI extraction covers info and operation structure only; schemas/security/runtime compatibility remain unevaluated");
                }
            }
            if (document.contains("$schema") && document.at("$schema") == "https://json-schema.org/draft/2020-12/schema") {
                supported = true;
                std::function<void(const Json&, const std::string&, const std::string&)> walk;
                walk = [&](const Json& object, const std::string& pointer, const std::string& parent) {
                    if (!object.is_object()) return;
                    if (native.nodes.size() >= max_nodes) {
                        native.limitations.insert("native_node_limit: extraction is partial"); return;
                    }
                    Json attrs = Json::object();
                    for (const auto* field : {"$schema", "$id", "title", "type", "$ref"})
                        if (object.contains(field) && object.at(field).is_string()) attrs[field] = object.at(field);
                    const auto node = native.node("schema_structure", digest, pointer, attrs);
                    native.edge(parent, node, "declares");
                    if (object.contains("properties") && object.at("properties").is_object()) {
                        for (const auto& [key, child] : object.at("properties").items())
                            walk(child, pointer + "/properties/" + pointer_escape(key), node);
                    }
                };
                walk(document, "", source_node);
                native.limitations.insert("JSON Schema 2020-12 extraction covers named property structure; no instance validation or reference resolution is performed");
            }
            if (!supported) native.limitations.insert("unsupported JSON dialect retained without semantic interpretation: " + digest);
        } else native.limitations.insert("unsupported media type retained without native interpretation: " + media);
    }
    return native;
}
Json interpret(const Json& payload, const std::string& domain) {
    fields(payload, {"captures", "claims", "interpreter_version", "selection_policy"});
    const auto selected = captures(payload.at("captures"), domain);
    const auto assertions = claims(payload.at("claims"), selected);
    const auto native = native_graph(selected);
    return sealed(Json{{"protocol", "symphony.scv.knowledge.v1"}, {"domain", domain},
        {"interpreter_version", string_at(payload, "interpreter_version")},
        {"selection_policy", policy(payload.at("selection_policy"))},
        {"captures", map_values(selected)}, {"claims", map_values(assertions)},
        {"native_nodes", map_values(native.nodes)}, {"native_edges", map_values(native.edges)},
        {"limitations", native.limitations}});
}
void validate_knowledge(const Json& input, const std::string& domain) {
    fields(input, {"protocol", "domain", "interpreter_version", "selection_policy", "captures", "claims",
                  "native_nodes", "native_edges", "limitations", "digest"});
    if (input.at("protocol") != "symphony.scv.knowledge.v1") invalid("unsupported knowledge protocol");
    static_cast<void>(string_at(input, "domain"));
    verify_digest(input);
    static_cast<void>(captures(input.at("captures"), input.at("domain").get<std::string>()));
    Json replay_payload;
    for (const auto* field : {"captures", "claims", "interpreter_version", "selection_policy"}) replay_payload[field] = input.at(field);
    auto replay = interpret(replay_payload, domain);
    replay["domain"] = input.at("domain");
    replay.erase("digest");
    if (sealed(replay) != input) invalid("knowledge is not a reproducible normalized interpretation");
}
Json build_graph(const Json& payload, const std::string& domain) {
    fields(payload, {"knowledge"});
    array_bound(payload.at("knowledge"), 16, "knowledge");
    if (payload.at("knowledge").empty()) invalid("graph requires at least one selected knowledge artifact");
    CaptureMap selected, assertions;
    std::set<std::string> digests;
    Json selected_policy;
    Json interpretations = Json::array();
    for (const auto& knowledge : payload.at("knowledge")) {
        validate_knowledge(knowledge, domain);
        if (selected_policy.is_null()) selected_policy = knowledge.at("selection_policy");
        else if (selected_policy != knowledge.at("selection_policy")) invalid("composition requires an explicit common selection policy");
        const auto digest = knowledge.at("digest").get<std::string>();
        if (!digests.insert(digest).second) invalid("duplicate knowledge selection");
        interpretations.push_back({{"knowledge_digest", digest}, {"domain", knowledge.at("domain")},
            {"interpreter_version", knowledge.at("interpreter_version")}});
        for (const auto& capture : knowledge.at("captures")) selected.emplace(capture.at("digest").get<std::string>(), capture);
        for (const auto& claim : knowledge.at("claims")) {
            const auto [it, inserted] = assertions.emplace(claim.at("claim_id").get<std::string>(), claim);
            if (!inserted && it->second != claim) invalid("claim_id has conflicting revisions; choose or namespace explicitly");
        }
    }
    if (selected.size() > max_captures || assertions.size() > max_claims) invalid("composed graph exceeds capture/claim bound");
    static_cast<void>(captures(map_values(selected), domain));
    const auto native = native_graph(selected);
    return sealed(Json{{"protocol", "symphony.scv.graph.v1"}, {"domain", domain},
        {"selection_policy", selected_policy}, {"knowledge_digests", digests},
        {"interpretations", sorted_array(interpretations)}, {"captures", map_values(selected)},
        {"claims", map_values(assertions)}, {"native_nodes", map_values(native.nodes)},
        {"native_edges", map_values(native.edges)}, {"limitations", native.limitations}});
}
void validate_graph(const Json& graph, const std::string& domain) {
    fields(graph, {"protocol", "domain", "selection_policy", "knowledge_digests", "interpretations", "captures",
                   "claims", "native_nodes", "native_edges", "limitations", "digest"});
    if (graph.at("protocol") != "symphony.scv.graph.v1" || graph.at("domain") != domain) invalid("graph protocol/domain mismatch");
    verify_digest(graph);
    if (policy(graph.at("selection_policy")) != graph.at("selection_policy")) invalid("noncanonical policy");
    unique_strings(graph.at("knowledge_digests"), 16, "knowledge_digests");
    if (graph.at("knowledge_digests").empty()) invalid("graph requires selected knowledge provenance");
    static const std::regex sha("sha256:[0-9a-f]{64}");
    for (const auto& digest : graph.at("knowledge_digests"))
        if (!std::regex_match(digest.get<std::string>(), sha)) invalid("invalid knowledge digest");
    array_bound(graph.at("interpretations"), 16, "interpretations");
    std::set<std::string> interpretations;
    for (const auto& item : graph.at("interpretations")) {
        fields(item, {"knowledge_digest", "domain", "interpreter_version"});
        for (const auto* key : {"domain", "interpreter_version"}) static_cast<void>(string_at(item, key));
        if (!interpretations.insert(string_at(item, "knowledge_digest", 71)).second) invalid("duplicate interpretation");
    }
    if (Json(interpretations) != graph.at("knowledge_digests")) invalid("interpretation provenance does not match selection");
    const auto selected = captures(graph.at("captures"), domain);
    if (map_values(selected) != graph.at("captures")) invalid("noncanonical captures");
    if (map_values(claims(graph.at("claims"), selected)) != graph.at("claims")) invalid("noncanonical claims");
    const auto native = native_graph(selected);
    if (map_values(native.nodes) != graph.at("native_nodes") || map_values(native.edges) != graph.at("native_edges") ||
        Json(native.limitations) != graph.at("limitations")) invalid("native graph does not match retained captures");
}

bool usable(const Json& finding) {
    return finding.at("status") == "supported" || finding.at("status") == "conditional";
}
void boundary(Json& current, std::int64_t candidate) {
    if (current.is_null() || candidate < current.get<std::int64_t>()) current = candidate;
}
bool assumption(const Json& claim) {
    return claim.at("statement_kind") == "user_assertion" || claim.at("statement_kind") == "hypothesis";
}
Json assess_claim(const Json& claim, const CaptureMap& selected, const Json& selected_policy,
                  std::int64_t now, const CaptureMap& previous, const std::set<std::string>& disputed) {
    std::set<std::string> reasons, roots, used_dependencies;
    Json validity = nullptr;
    std::string blocked;
    const auto id = claim.at("claim_id").get<std::string>();
    const auto& allowed = selected_policy.at("allowed_statement_kinds");
    if (std::find(allowed.begin(), allowed.end(), claim.at("statement_kind")) == allowed.end()) {
        blocked = "excluded"; reasons.insert("statement_kind_excluded_by_selected_policy");
    }
    if (!claim.at("valid_from").is_null() && now < utc_seconds(claim.at("valid_from"))) {
        blocked = "inactive"; reasons.insert("claim_not_yet_effective");
        boundary(validity, utc_seconds(claim.at("valid_from")));
    }
    if (!claim.at("valid_until").is_null()) {
        const auto until = utc_seconds(claim.at("valid_until"));
        boundary(validity, until);
        if (now >= until) { blocked = "stale"; reasons.insert("claim_validity_expired"); }
    }
    if (disputed.contains(id)) { blocked = "disputed"; reasons.insert("unresolved_typed_conflict"); }
    for (const auto& dependency : claim.at("scope_dependencies")) {
        std::set<std::string> current;
        for (const auto& [digest, capture] : selected) {
            if (capture.at("source").at("source_id") != dependency.at("source_id")) continue;
            current.insert(digest);
            if (capture.at("completeness") != "complete") {
                blocked = "unsupported"; reasons.insert("scope_coverage_incomplete");
            }
            const auto observed = utc_seconds(capture.at("observed_at"));
            if (now < observed) { blocked = "inactive"; reasons.insert("scope_capture_not_yet_observed"); boundary(validity, observed); }
            if (!selected_policy.at("max_age_seconds").is_null()) {
                const auto expires = observed + selected_policy.at("max_age_seconds").get<std::int64_t>() + 1;
                boundary(validity, expires);
                if (now >= expires) { blocked = "stale"; reasons.insert("scope_evidence_stale"); }
            }
        }
        if (Json(current) != dependency.at("capture_digests")) {
            blocked = "unsupported"; reasons.insert("scope_capture_selection_changed");
        }
        if (current.empty()) { blocked = "unsupported"; reasons.insert("scope_coverage_unknown"); }
    }
    Json evaluated_sets = Json::array();
    bool any_supported = false, any_conditional = false;
    for (const auto& support : support_sets(claim)) {
        std::set<std::string> set_reasons, set_roots, set_dependencies;
        bool set_usable = true, conditional = assumption(claim) || claim.at("statement_kind") == "inference";
        Json set_validity = nullptr;
        if (support.at("evidence").empty() && support.at("dependencies").empty()) {
            if (assumption(claim)) set_reasons.insert("selected_assumption_without_grounded_evidence");
            else { set_usable = false; set_reasons.insert("no_grounded_support"); }
        }
        for (const auto& anchor : support.at("evidence")) {
            const auto& capture = selected.at(anchor.at("capture_digest").get<std::string>());
            set_roots.insert(capture.at("body_digest").get<std::string>());
            if (capture.at("completeness") == "failed") { set_usable = false; set_reasons.insert("capture_failed"); }
            if (capture.at("completeness") == "partial") {
                set_reasons.insert("partial_capture");
                if (selected_policy.at("partial_capture") == "exclude") set_usable = false;
                else conditional = true;
            }
            if (!capture.at("issues").empty()) { conditional = true; set_reasons.insert("capture_has_reported_issues"); }
            const auto observed = utc_seconds(capture.at("observed_at"));
            if (now < observed) {
                set_usable = false; set_reasons.insert("evidence_not_yet_observed"); boundary(set_validity, observed);
            }
            if (!selected_policy.at("max_age_seconds").is_null()) {
                const auto expires = observed + selected_policy.at("max_age_seconds").get<std::int64_t>() + 1;
                boundary(set_validity, expires);
                if (now >= expires) { set_usable = false; set_reasons.insert("evidence_stale"); }
            } else set_reasons.insert("freshness_unbounded_by_selected_policy");
        }
        for (const auto& dependency : support.at("dependencies")) {
            const auto dependency_id = dependency.at("claim_id").get<std::string>();
            set_dependencies.insert(dependency_id);
            const auto found = previous.find(dependency_id);
            if (found == previous.end()) {
                set_usable = false; set_reasons.insert("missing_dependency:" + dependency_id); continue;
            }
            if (!usable(found->second)) { set_usable = false; set_reasons.insert("dependency_not_eligible:" + dependency_id); }
            if (found->second.at("status") == "conditional") conditional = true;
            for (const auto& root : found->second.at("evidence_roots")) set_roots.insert(root.get<std::string>());
            if (!found->second.at("valid_until_unix_seconds").is_null())
                boundary(set_validity, found->second.at("valid_until_unix_seconds").get<std::int64_t>());
        }
        if (!blocked.empty()) set_usable = false;
        const auto status = set_usable ? (conditional ? "conditional" : "supported") : "unsupported";
        evaluated_sets.push_back({{"support_id", support.at("support_id")}, {"status", status},
            {"evidence", support.at("evidence")}, {"dependencies", support.at("dependencies")},
            {"evidence_roots", set_roots}, {"reasons", set_reasons}, {"valid_until_unix_seconds", set_validity}});
        if (set_usable) {
            any_supported = any_supported || !conditional;
            any_conditional = any_conditional || conditional;
            roots.insert(set_roots.begin(), set_roots.end());
            used_dependencies.insert(set_dependencies.begin(), set_dependencies.end());
        }
        // A boundary on an inactive alternative is still a mandatory reassessment
        // point: its future admission may change the available support paths.
        if (!set_validity.is_null()) boundary(validity, set_validity.get<std::int64_t>());
    }
    std::string status = blocked.empty() ? (any_supported ? "supported" : any_conditional ? "conditional" : "unsupported") : blocked;
    if (status == "unsupported") reasons.insert("no_current_eligible_support_set");
    if (assumption(claim)) reasons.insert("user_assumption_or_hypothesis_preserved");
    if (claim.at("statement_kind") == "recommendation") reasons.insert("recommendation_is_not_a_requirement");
    if (claim.at("statement_kind") == "inference") reasons.insert("caller_proposed_inference_rule_not_verified");
    reasons.insert("semantic_claim_not_empirically_verified");
    return Json{{"claim", claim}, {"status", status}, {"reasons", reasons},
        {"support_sets", evaluated_sets}, {"evidence_roots", roots},
        {"dependent_claim_ids", used_dependencies}, {"valid_until_unix_seconds", validity}};
}
CaptureMap settle(const CaptureMap& assertions, const CaptureMap& selected, const Json& selected_policy,
                  std::int64_t now, const std::set<std::string>& disputed, const engine::Request& request) {
    CaptureMap previous;
    for (const auto& [id, claim] : assertions) {
        static_cast<void>(claim);
        previous[id] = {{"status", "unsupported"}, {"evidence_roots", Json::array()}, {"valid_until_unix_seconds", nullptr}};
    }
    // Least fixed point: a cycle begins without eligible premises. It can inherit
    // existing grounded support, but cannot create a new evidence root.
    for (std::size_t iteration = 0; iteration <= assertions.size() * 3 + 2; ++iteration) {
        check_deadline(request);
        CaptureMap next;
        for (const auto& [id, claim] : assertions) next[id] = assess_claim(claim, selected, selected_policy, now, previous, disputed);
        if (next == previous) return next;
        previous = std::move(next);
    }
    throw engine::Error("knowledge.evaluation_limit", "support evaluation did not reach a bounded fixed point", 4);
}
Json typed_conflicts(const CaptureMap& assertions, const CaptureMap& findings) {
    Json result = Json::array();
    for (auto left = assertions.begin(); left != assertions.end(); ++left) {
        if (!usable(findings.at(left->first))) continue;
        for (auto right = std::next(left); right != assertions.end(); ++right) {
            if (!usable(findings.at(right->first))) continue;
            const auto& a = left->second;
            const auto& b = right->second;
            if (a.at("subject") != b.at("subject") || a.at("predicate") != b.at("predicate") || a.at("scope") != b.at("scope")) continue;
            if (a.at("value") == b.at("value")) continue;
            std::string type;
            bool blocking = false;
            if (a.at("statement_kind") != b.at("statement_kind")) type = "statement_kind_distinction";
            else if (a.at("value").at("unit") != b.at("value").at("unit")) type = "unit_mismatch";
            else if (assumption(a) || a.at("statement_kind") == "recommendation") type = "alternative_proposals";
            else {
                blocking = true;
                type = a.at("value").at("type") != b.at("value").at("type") ? "value_type_mismatch" : "incompatible_values";
            }
            result.push_back({{"left_claim_id", left->first}, {"right_claim_id", right->first},
                {"type", type}, {"blocking", blocking}, {"resolution", "unresolved; no precedence rule applied"}});
        }
    }
    return result;
}
bool reaches(const CaptureMap& assertions, const std::string& from, const std::string& target, std::set<std::string>& visited) {
    if (!visited.insert(from).second) return false;
    const auto found = assertions.find(from);
    if (found == assertions.end()) return false;
    for (const auto& support : support_sets(found->second)) {
        for (const auto& dependency : support.at("dependencies")) {
            const auto id = dependency.at("claim_id").get<std::string>();
            if (id == target || reaches(assertions, id, target, visited)) return true;
        }
    }
    return false;
}
Json evaluate_graph(const Json& graph, const Json& time, const std::string& protocol, const engine::Request& request) {
    const auto now = utc_seconds(time);
    const auto selected = captures(graph.at("captures"), graph.at("domain").get<std::string>());
    const auto assertions = claims(graph.at("claims"), selected);
    auto findings = settle(assertions, selected, graph.at("selection_policy"), now, {}, request);
    const auto conflicts = typed_conflicts(assertions, findings);
    std::set<std::string> disputed;
    for (const auto& conflict : conflicts) if (conflict.at("blocking") == true) {
        disputed.insert(conflict.at("left_claim_id").get<std::string>());
        disputed.insert(conflict.at("right_claim_id").get<std::string>());
    }
    if (!disputed.empty()) findings = settle(assertions, selected, graph.at("selection_policy"), now, disputed, request);
    for (auto& [id, finding] : findings) {
        std::set<std::string> visited;
        finding["support_cycle_present"] = reaches(assertions, id, id, visited);
        finding["semantic_validation"] = "caller_proposed; anchors and bounded support structure checked only";
    }
    std::map<std::string, std::set<std::string>> root_sources, root_captures;
    for (const auto& [digest, capture] : selected) {
        const auto root = capture.at("body_digest").get<std::string>();
        root_sources[root].insert(capture.at("source").at("source_id").get<std::string>());
        root_captures[root].insert(digest);
    }
    Json origins = Json::array();
    for (const auto& [root, sources] : root_sources) origins.push_back({{"body_digest", root},
        {"source_ids", sources}, {"capture_digests", root_captures.at(root)}, {"independence", "not_asserted"}});
    Json result = {{"protocol", protocol}, {"domain", graph.at("domain")}, {"graph_digest", graph.at("digest")},
        {"query_time", time}, {"selection_policy", graph.at("selection_policy")}, {"findings", map_values(findings)},
        {"conflicts", conflicts}, {"evidence_origins", origins}, {"limitations", graph.at("limitations")},
        {"coverage", {{"selected_captures", selected.size()}, {"selected_claims", assertions.size()},
                      {"matched_claims", assertions.size()}, {"native_nodes", graph.at("native_nodes").size()},
                      {"extent", "bounded_selected_corpus"}, {"provider_completeness", "not_asserted"}}}};
    result["limitations"].push_back("evidence roots deduplicate identical body bytes; independent corroboration and arbitrary inference rules are not established");
    return result;
}
void filter_findings(Json& result, const Json& payload) {
    std::set<std::string> ids;
    if (payload.contains("claim_ids")) {
        unique_strings(payload.at("claim_ids"), max_claims, "claim_ids");
        for (const auto& id : payload.at("claim_ids")) ids.insert(id.get<std::string>());
    }
    for (const auto* field : {"subject", "predicate"}) if (payload.contains(field)) static_cast<void>(string_at(payload, field));
    auto filtered = Json::array();
    for (const auto& finding : result.at("findings")) {
        const auto& claim = finding.at("claim");
        if (payload.contains("claim_ids") && !ids.contains(claim.at("claim_id").get<std::string>())) continue;
        if (payload.contains("subject") && payload.at("subject") != claim.at("subject")) continue;
        if (payload.contains("predicate") && payload.at("predicate") != claim.at("predicate")) continue;
        filtered.push_back(finding);
    }
    result["findings"] = filtered;
    result["coverage"]["matched_claims"] = filtered.size();
    result["query_selection"] = payload;
    result["query_selection"].erase("graph");
    if (filtered.empty()) result["absence"] = "no matching claim in the bounded selected corpus; no availability conclusion";
}
Json query(const Json& payload, const std::string& domain, const std::string& protocol, const engine::Request& request) {
    fields(payload, {"graph", "query_time"}, {"claim_ids", "subject", "predicate"});
    validate_graph(payload.at("graph"), domain);
    auto result = evaluate_graph(payload.at("graph"), payload.at("query_time"), protocol, request);
    filter_findings(result, payload);
    return sealed(result);
}
Json explain(const Json& payload, const std::string& domain, const engine::Request& request) {
    fields(payload, {"graph", "query_time", "claim_id"});
    validate_graph(payload.at("graph"), domain);
    const auto id = string_at(payload, "claim_id");
    const auto& graph = payload.at("graph");
    const auto assertions = claims(graph.at("claims"), captures(graph.at("captures"), domain));
    if (!assertions.contains(id)) invalid("claim_id is not in selected graph");
    std::set<std::string> closure;
    std::vector<std::string> pending = {id};
    while (!pending.empty()) {
        const auto current = pending.back(); pending.pop_back();
        if (!closure.insert(current).second || !assertions.contains(current)) continue;
        for (const auto& support : support_sets(assertions.at(current)))
            for (const auto& dependency : support.at("dependencies")) pending.push_back(dependency.at("claim_id").get<std::string>());
    }
    auto result = evaluate_graph(graph, payload.at("query_time"), "symphony.scv.explain-result.v1", request);
    filter_findings(result, Json{{"query_time", payload.at("query_time")}, {"claim_ids", closure}});
    result["claim_id"] = id;
    result["dependency_closure"] = closure;
    return sealed(result);
}
Json difference(const Json& payload, const std::string& domain, const engine::Request& request) {
    fields(payload, {"before", "after", "query_time"});
    const auto& before = payload.at("before");
    const auto& after = payload.at("after");
    validate_graph(before, domain); validate_graph(after, domain);
    const auto old_captures = captures(before.at("captures"), domain), new_captures = captures(after.at("captures"), domain);
    const auto old_claims = claims(before.at("claims"), old_captures), new_claims = claims(after.at("claims"), new_captures);
    std::set<std::string> added, removed, changed, affected, changed_sources;
    Json capture_changes = Json::array();
    for (const auto& [digest, capture] : old_captures) if (!new_captures.contains(digest)) {
        capture_changes.push_back({{"capture_digest", digest}, {"change", "removed"}});
        changed_sources.insert(capture.at("source").at("source_id").get<std::string>());
    }
    for (const auto& [digest, capture] : new_captures) if (!old_captures.contains(digest)) {
        capture_changes.push_back({{"capture_digest", digest}, {"change", "added"}});
        changed_sources.insert(capture.at("source").at("source_id").get<std::string>());
    }
    for (const auto& [id, claim] : old_claims) {
        if (!new_claims.contains(id)) removed.insert(id);
        else if (new_claims.at(id) != claim) changed.insert(id);
    }
    for (const auto& [id, claim] : new_claims) { static_cast<void>(claim); if (!old_claims.contains(id)) added.insert(id); }
    affected.insert(added.begin(), added.end()); affected.insert(removed.begin(), removed.end()); affected.insert(changed.begin(), changed.end());
    CaptureMap all = old_claims;
    for (const auto& [id, claim] : new_claims) all[id] = claim;
    const bool policy_changed = before.at("selection_policy") != after.at("selection_policy");
    const bool interpretation_changed = before.at("interpretations") != after.at("interpretations");
    std::set<std::string> old_profiles, new_profiles;
    for (const auto& item : before.at("interpretations"))
        old_profiles.insert(Json::array({item.at("domain"), item.at("interpreter_version")}).dump());
    for (const auto& item : after.at("interpretations"))
        new_profiles.insert(Json::array({item.at("domain"), item.at("interpreter_version")}).dump());
    for (const auto& [id, claim] : all) {
        if (policy_changed || old_profiles != new_profiles) affected.insert(id);
        for (const auto& scope : claim.at("scope_dependencies"))
            if (changed_sources.contains(scope.at("source_id").get<std::string>())) affected.insert(id);
    }
    bool progress = true;
    while (progress) {
        progress = false;
        for (const auto& [id, claim] : all) {
            if (affected.contains(id)) continue;
            for (const auto& support : support_sets(claim)) for (const auto& dependency : support.at("dependencies")) {
                if (affected.contains(dependency.at("claim_id").get<std::string>())) progress = affected.insert(id).second || progress;
            }
        }
    }
    const auto old_evaluation = sealed(evaluate_graph(before, payload.at("query_time"), "symphony.scv.evaluate-result.v1", request));
    const auto new_evaluation = sealed(evaluate_graph(after, payload.at("query_time"), "symphony.scv.evaluate-result.v1", request));
    return sealed(Json{{"protocol", "symphony.scv.diff-result.v1"}, {"domain", domain},
        {"before_digest", before.at("digest")}, {"after_digest", after.at("digest")}, {"query_time", payload.at("query_time")},
        {"changes", {{"added_claim_ids", added}, {"removed_claim_ids", removed}, {"changed_claim_ids", changed},
                     {"captures", capture_changes}, {"selection_policy_changed", policy_changed},
                     {"interpretation_selection_changed", interpretation_changed}}},
        {"affected_claim_ids", affected}, {"before_evaluation", old_evaluation}, {"after_evaluation", new_evaluation},
        {"limitations", Json::array({"impact is conservative within retained support/scope dependencies; source byte change is not a proven behavioral change",
            "no user selection, provider substitution or deployment is performed"})}});
}

} // namespace

engine::Json handle_knowledge(const engine::Request& request, const std::string& domain) {
    check_deadline(request);
    Json result;
    if (request.operation == "knowledge_interpret") result = interpret(request.payload, domain);
    else if (request.operation == "graph_build") result = build_graph(request.payload, domain);
    else if (request.operation == "graph_query") result = query(request.payload, domain, "symphony.scv.query-result.v1", request);
    else if (request.operation == "graph_evaluate") result = query(request.payload, domain, "symphony.scv.evaluate-result.v1", request);
    else if (request.operation == "graph_explain") result = explain(request.payload, domain, request);
    else if (request.operation == "graph_diff") result = difference(request.payload, domain, request);
    else throw engine::Error("operation.unsupported", "unsupported source-knowledge operation", 4);
    check_deadline(request);
    return result;
}
} // namespace symphony::knowledge::scv
