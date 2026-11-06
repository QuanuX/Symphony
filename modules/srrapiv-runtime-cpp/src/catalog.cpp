#include "catalog.hpp"
#include "symphony/knowledge/engine/digest.hpp"

#include <algorithm>
#include <cctype>
#include <initializer_list>
#include <map>
#include <set>
#include <stdexcept>
#include <string_view>
#include <vector>

namespace symphony::srrapiv {
namespace {
constexpr std::size_t max_artifacts = 256;
constexpr std::size_t max_elements = 16384;
constexpr std::size_t max_evidence = 32768;
using Strings = std::set<std::string>;

struct Invalid : std::exception {
    std::string code, path;
    Invalid(std::string c, std::string p) : code(std::move(c)), path(std::move(p)) {}
};
[[noreturn]] void fail(const std::string& code, const std::string& path) {
    throw Invalid(code, path);
}
void object(const Json& v, const std::string& path,
            std::initializer_list<std::string_view> required,
            std::initializer_list<std::string_view> optional = {}) {
    if (!v.is_object()) fail("object_required", path);
    for (auto key : required)
        if (!v.contains(std::string(key))) fail("field_required", path + "/" + std::string(key));
    for (const auto& [key, ignored] : v.items()) {
        static_cast<void>(ignored);
        auto matches = [&](auto keys) { return std::find(keys.begin(), keys.end(), key) != keys.end(); };
        if (!matches(required) && !matches(optional))
            // Do not echo an unknown key: it may itself contain secret material.
            fail("unknown_field", path);
    }
}
std::string text(const Json& v, const std::string& path, std::size_t limit = 256,
                 bool spaces = false) {
    if (!v.is_string()) fail("string_required", path);
    const auto s = v.get<std::string>();
    if (s.empty() || s.size() > limit) fail("string_size", path);
    for (unsigned char c : s) {
        if (c < 0x21 || c > 0x7e) {
            if (!(spaces && c == ' ')) fail("identifier_character", path);
        }
    }
    if (!spaces && s.find_first_not_of("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._:-/@+") != std::string::npos)
        fail("identifier_character", path);
    return s;
}
std::string exact(const Json& v, const std::string& path) {
    auto s = text(v, path);
    auto lower = s;
    std::transform(lower.begin(), lower.end(), lower.begin(), [](unsigned char c){return static_cast<char>(std::tolower(c));});
    if (lower == "latest" || lower == "current" || lower == "default" || lower == "auto")
        fail("mutable_alias_rejected", path);
    return s;
}
std::string digest(const Json& v, const std::string& path) {
    auto s = text(v, path, 71);
    if (s.size() != 71 || !s.starts_with("sha256:") ||
        s.find_first_not_of("0123456789abcdef", 7) != std::string::npos)
        fail("sha256_required", path);
    return s;
}
void literal(const Json& v, std::string_view wanted, const std::string& path) {
    if (!v.is_string() || v.get_ref<const std::string&>() != wanted) fail("unsupported_schema", path);
}
void one_of(const Json& v, const std::string& path, std::initializer_list<std::string_view> choices) {
    const auto s = text(v, path);
    if (std::find(choices.begin(), choices.end(), s) == choices.end()) fail("unsupported_value", path);
}
void array(const Json& v, const std::string& path, std::size_t maximum, bool empty_allowed = true) {
    if (!v.is_array()) fail("array_required", path);
    if (v.size() > maximum || (!empty_allowed && v.empty())) fail("array_size", path);
}
Strings strings(const Json& v, const std::string& path, bool empty_allowed = true,
                std::size_t maximum = max_elements) {
    array(v, path, maximum, empty_allowed);
    Strings result;
    for (std::size_t i = 0; i < v.size(); ++i) {
        auto s = exact(v[i], path + "/" + std::to_string(i));
        if (!result.insert(s).second) fail("duplicate_value", path + "/" + std::to_string(i));
    }
    return result;
}
std::string reference(const Json& v, const std::string& path) {
    const auto s = exact(v, path);
    if (s.find(':') == std::string::npos || s.find("://") != std::string::npos)
        fail("opaque_reference_required", path);
    return s;
}
void content_identity(const Json& v, const char* field, const std::string& path) {
    const auto given = digest(v.at(field), path + "/" + field);
    auto payload = v;
    payload.erase(field);
    if (given != knowledge::engine::tagged_sha256(payload.dump()))
        fail("content_identity_mismatch", path + "/" + field);
}
void version(const Json& v, const std::string& path) {
    auto s = exact(v, path);
    unsigned parts = 0;
    bool digit_seen = false;
    for (const auto c : s) {
        if (c >= '0' && c <= '9') digit_seen = true;
        else if (c == '.' && digit_seen) {++parts; digit_seen = false;}
        else fail("sdk_version_required", path);
    }
    if (!digit_seen || parts != 3) fail("sdk_version_required", path);
}

void artifact(const Json& v, const std::string& p) {
    object(v, p, {"artifact_id","sdk_version","product","archive_digest","header_digest","library_digest",
                 "adapter_version","adapter_digest","platform","contracts","connection","capabilities"});
    version(v.at("sdk_version"), p + "/sdk_version");
    literal(v.at("product"), "rithmic-rapi-plus-cpp", p + "/product");
    for (const auto* k : {"archive_digest","header_digest","library_digest","adapter_digest"})
        digest(v.at(k), p + "/" + k);
    exact(v.at("adapter_version"), p + "/adapter_version");
    const auto& platform = v.at("platform");
    object(platform, p + "/platform", {"os","architecture","compiler","abi","dependency_digest"});
    one_of(platform.at("os"), p + "/platform/os", {"linux","darwin"});
    for (const auto* k : {"architecture","compiler","abi"}) exact(platform.at(k), p + "/platform/" + k);
    digest(platform.at("dependency_digest"), p + "/platform/dependency_digest");
    const auto& c = v.at("contracts");
    object(c, p + "/contracts", {"api_mapping","control","event","state","journal","state_reader",
                                "readable_state_schemas","readable_journal_schemas"});
    for (const auto* k : {"api_mapping","control","event","state","journal","state_reader"})
        exact(c.at(k), p + "/contracts/" + k);
    const auto readable_states = strings(c.at("readable_state_schemas"), p + "/contracts/readable_state_schemas", false);
    const auto readable_journals = strings(c.at("readable_journal_schemas"), p + "/contracts/readable_journal_schemas", false);
    if (!readable_states.contains(c.at("state").get<std::string>()) ||
        !readable_journals.contains(c.at("journal").get<std::string>()))
        fail("own_writer_reader_mismatch", p + "/contracts");
    const auto& connection = v.at("connection");
    object(connection, p + "/connection", {"profile_digest","trust_digest"});
    digest(connection.at("profile_digest"), p + "/connection/profile_digest");
    digest(connection.at("trust_digest"), p + "/connection/trust_digest");
    strings(v.at("capabilities"), p + "/capabilities", false);
    content_identity(v, "artifact_id", p);
}
using Catalog = std::map<std::string, const Json*>;
Catalog catalog(const Json& v, const std::string& p) {
    object(v, p, {"schema","artifacts"});
    literal(v.at("schema"), "symphony.srrapiv.catalog.v1", p + "/schema");
    array(v.at("artifacts"), p + "/artifacts", max_artifacts);
    Catalog result;
    for (std::size_t i = 0; i < v.at("artifacts").size(); ++i) {
        const auto& a = v.at("artifacts")[i];
        const auto ap = p + "/artifacts/" + std::to_string(i);
        artifact(a, ap);
        if (!result.emplace(a.at("artifact_id").get<std::string>(), &a).second)
            fail("duplicate_artifact", ap + "/artifact_id");
    }
    return result;
}
void binding(const Json& b, const std::string& p) {
    object(b, p, {"schema","binding_id","artifact_id","strategy_id","environment","account_ref",
                 "credential_reference","required_capabilities","required_evidence_scope","feed_profile_digest","metadata_digest"});
    literal(b.at("schema"), "symphony.srrapiv.binding.v1", p + "/schema");
    digest(b.at("artifact_id"), p + "/artifact_id");
    exact(b.at("strategy_id"), p + "/strategy_id");
    // Provider system/environment names are exact native text and can contain
    // spaces (for example Rithmic Test); they are not namespace identifiers.
    text(b.at("environment"), p + "/environment", 256, true);
    reference(b.at("account_ref"), p + "/account_ref");
    const auto& cr = b.at("credential_reference");
    object(cr, p + "/credential_reference", {"provider_id","resource_id"});
    reference(cr.at("provider_id"), p + "/credential_reference/provider_id");
    reference(cr.at("resource_id"), p + "/credential_reference/resource_id");
    strings(b.at("required_capabilities"), p + "/required_capabilities", false);
    one_of(b.at("required_evidence_scope"), p + "/required_evidence_scope", {"native_fixture","provider_test","production"});
    digest(b.at("feed_profile_digest"), p + "/feed_profile_digest");
    digest(b.at("metadata_digest"), p + "/metadata_digest");
    content_identity(b, "binding_id", p);
}
void assertions(const Json& list) {
    array(list, "/compatibility_evidence", max_evidence);
    Strings seen;
    for (std::size_t i = 0; i < list.size(); ++i) {
        const auto& a = list[i];
        const auto p = "/compatibility_evidence/" + std::to_string(i);
        object(a, p, {"binding_id","capability","result","evidence_id","scope","observed_at"});
        digest(a.at("binding_id"), p + "/binding_id");
        exact(a.at("capability"), p + "/capability");
        one_of(a.at("result"), p + "/result", {"supported","incompatible"});
        reference(a.at("evidence_id"), p + "/evidence_id");
        one_of(a.at("scope"), p + "/scope", {"native_fixture","provider_test","production"});
        // Observation is retained as an opaque, bounded timestamp; this pure
        // operation never infers freshness or current provider acceptance.
        exact(a.at("observed_at"), p + "/observed_at");
        if (!seen.insert(a.dump()).second) fail("duplicate_assertion", p);
    }
}
const Json& selected(const Catalog& c, const Json& id, const std::string& path) {
    const auto key = digest(id, path);
    const auto it = c.find(key);
    if (it == c.end()) fail("exact_artifact_not_found", path);
    return *it->second;
}
Json assess(const Catalog& c, const Json& b, const Json& evidence, const std::string& path) {
    binding(b, path);
    const auto& a = selected(c, b.at("artifact_id"), path + "/artifact_id");
    const auto available = strings(a.at("capabilities"), "/selected/capabilities");
    const auto required = strings(b.at("required_capabilities"), path + "/required_capabilities");
    Json caps = Json::array();
    bool any_support = false, any_unknown = false, any_bad = false, any_conflict = false;
    for (const auto& cap : required) {
        Strings results;
        Json refs = Json::array();
        for (const auto& e : evidence) {
            if (e.at("binding_id") == b.at("binding_id") && e.at("capability") == cap &&
                e.at("scope") == b.at("required_evidence_scope")) {
                results.insert(e.at("result").get<std::string>());
                refs.push_back(e.at("evidence_id"));
            }
        }
        std::string state;
        if (!available.contains(cap)) state = "unavailable";
        else if (results.size() > 1) state = "conflicted";
        else if (results.contains("incompatible")) state = "incompatible";
        else if (results.contains("supported")) state = "supported";
        else state = "unknown";
        any_support |= state == "supported";
        any_unknown |= state == "unknown";
        any_bad |= state == "unavailable" || state == "incompatible";
        any_conflict |= state == "conflicted";
        caps.push_back({{"capability",cap},{"state",state},{"evidence_ids",refs}});
    }
    const std::string compatibility = any_conflict ? "conflicted" : any_bad ? "incompatible" :
        any_support && any_unknown ? "partial" : any_unknown ? "unknown" : "supported";
    return {{"valid",true},{"binding_id",b.at("binding_id")},{"artifact_id",b.at("artifact_id")},
            {"sdk_version",a.at("sdk_version")},{"compatibility",compatibility},{"capabilities",caps},
            {"evidence_scope",b.at("required_evidence_scope")},{"assertions_verified",false},
            {"live_qualified",false},{"provider_activity_authorized",false}};
}
void diff(const Json& from, const Json& to, const std::string& p, Json& changes) {
    if (from == to) return;
    if (from.is_object() && to.is_object()) {
        Strings keys;
        for (const auto& [k,v] : from.items()) {static_cast<void>(v);keys.insert(k);}
        for (const auto& [k,v] : to.items()) {static_cast<void>(v);keys.insert(k);}
        for (const auto& k : keys) {
            if (!from.contains(k) || !to.contains(k)) changes.push_back(p + "/" + k);
            else diff(from.at(k), to.at(k), p + "/" + k, changes);
        }
    } else changes.push_back(p);
}
bool reads(const Json& reader, const Json& writer, const char* readable, const char* schema) {
    const auto list = strings(reader.at("contracts").at(readable), "/reader");
    return list.contains(writer.at("contracts").at(schema).get<std::string>());
}
Json coverage(const Json& input) {
    object(input, "", {"schema","sdk_artifact_id","universe","mappings"});
    literal(input.at("schema"), "symphony.srrapiv.coverage.v1", "/schema");
    digest(input.at("sdk_artifact_id"), "/sdk_artifact_id");
    const auto universe = strings(input.at("universe"), "/universe", false);
    array(input.at("mappings"), "/mappings", max_elements);
    Strings seen;
    Json unresolved = Json::array();
    std::size_t mapped = 0;
    for (std::size_t i = 0; i < input.at("mappings").size(); ++i) {
        const auto& m = input.at("mappings")[i];
        const auto p = "/mappings/" + std::to_string(i);
        object(m, p, {"element_id","disposition","handler_id","operation_id","qxctl_route",
                      "input_schema","result_schema","event_schemas","evidence_ids"}, {"equivalence_evidence_id"});
        const auto id = exact(m.at("element_id"), p + "/element_id");
        if (!universe.contains(id)) fail("element_outside_universe", p + "/element_id");
        if (!seen.insert(id).second) fail("duplicate_mapping", p + "/element_id");
        one_of(m.at("disposition"), p + "/disposition", {"implemented","equivalent","unresolved","provider_unavailable"});
        const auto disposition = m.at("disposition").get<std::string>();
        const bool implemented = disposition == "implemented" || disposition == "equivalent";
        for (const auto* k : {"handler_id","operation_id","input_schema","result_schema"}) {
            if (implemented || !m.at(k).is_null()) exact(m.at(k), p + "/" + k);
        }
        if (implemented || !m.at("qxctl_route").is_null()) text(m.at("qxctl_route"), p + "/qxctl_route",256,true);
        strings(m.at("event_schemas"), p + "/event_schemas");
        const auto evidence = strings(m.at("evidence_ids"), p + "/evidence_ids", !implemented);
        static_cast<void>(evidence);
        if (disposition == "equivalent" && !m.contains("equivalence_evidence_id"))
            fail("equivalence_evidence_required", p);
        if (m.contains("equivalence_evidence_id")) reference(m.at("equivalence_evidence_id"), p + "/equivalence_evidence_id");
        if (implemented) ++mapped;
        else unresolved.push_back({{"element_id",id},{"reason",disposition}});
    }
    for (const auto& id : universe)
        if (!seen.contains(id)) unresolved.push_back({{"element_id",id},{"reason","unmapped"}});
    return {{"sdk_artifact_id",input.at("sdk_artifact_id")},{"universe_digest",knowledge::engine::tagged_sha256(input.at("universe").dump())},
            {"total_elements",universe.size()},{"mapped_elements",mapped},{"unresolved",unresolved},
            {"complete_for_supplied_universe",unresolved.empty()},{"sdk_universe_independently_verified",false},
            {"implementation_evidence_verified",false},{"live_qualified",false}};
}
}

Json catalog_operation(const std::string& operation, const Json& input) {
    try {
        Json result;
        if (operation == "coverage_assess") result = coverage(input);
        else if (operation == "catalog_inspect") {
            object(input, "", {"catalog"}, {"artifact_id"});
            const auto c = catalog(input.at("catalog"), "/catalog");
            Json rows = Json::array();
            if (input.contains("artifact_id")) rows.push_back(selected(c,input.at("artifact_id"),"/artifact_id"));
            else for (const auto& [id,a] : c) {static_cast<void>(id);rows.push_back(*a);}
            result = {{"artifacts",rows},{"catalog_digest",knowledge::engine::tagged_sha256(input.at("catalog").dump())},
                      {"selection",input.contains("artifact_id") ? "exact" : "inventory"},{"substitution_performed",false},
                      {"artifact_bytes_verified",false},{"live_qualified",false}};
        } else if (operation == "binding_validate") {
            object(input, "", {"catalog","binding","compatibility_evidence"});
            const auto c = catalog(input.at("catalog"), "/catalog");
            assertions(input.at("compatibility_evidence"));
            result = assess(c,input.at("binding"),input.at("compatibility_evidence"),"/binding");
        } else if (operation == "binding_compare") {
            object(input, "", {"catalog","from_binding","to_binding","compatibility_evidence"});
            const auto c = catalog(input.at("catalog"), "/catalog");
            assertions(input.at("compatibility_evidence"));
            const auto& from = input.at("from_binding");
            const auto& to = input.at("to_binding");
            const auto from_result = assess(c,from,input.at("compatibility_evidence"),"/from_binding");
            const auto to_result = assess(c,to,input.at("compatibility_evidence"),"/to_binding");
            if (from.at("strategy_id") != to.at("strategy_id")) fail("strategy_identity_mismatch", "/to_binding/strategy_id");
            const auto& a = selected(c,from.at("artifact_id"),"/from_binding/artifact_id");
            const auto& b = selected(c,to.at("artifact_id"),"/to_binding/artifact_id");
            Json changes = Json::array();
            diff(a,b,"/artifact",changes);
            diff(from,to,"/binding",changes);
            const bool forward_state = reads(b,a,"readable_state_schemas","state");
            const bool forward_journal = reads(b,a,"readable_journal_schemas","journal");
            const bool reverse_state = reads(a,b,"readable_state_schemas","state");
            const bool reverse_journal = reads(a,b,"readable_journal_schemas","journal");
            Strings before = strings(a.at("capabilities"),"/before/capabilities");
            Strings after = strings(b.at("capabilities"),"/after/capabilities");
            Json added = Json::array(), removed = Json::array();
            for (const auto& x : after) if (!before.contains(x)) added.push_back(x);
            for (const auto& x : before) if (!after.contains(x)) removed.push_back(x);
            result = {{"from",from_result},{"to",to_result},{"changed_paths",changes},
                      {"added_capabilities",added},{"removed_capabilities",removed},
                      {"forward_state_readable",forward_state},{"forward_journal_readable",forward_journal},
                      {"rollback_state_readable",reverse_state},{"rollback_journal_readable",reverse_journal},
                      {"reader_claims_verified",false},{"migration_required",!forward_state || !forward_journal},
                      {"rollback_reader_gap",!reverse_state || !reverse_journal},
                      {"selection_changed",from.at("binding_id") != to.at("binding_id")},
                      {"provider_state_rollback",false},{"automatically_applied",false},{"live_qualified",false}};
        } else fail("unknown_operation", "/operation");
        result["ok"] = true;
        result["operation"] = operation;
        return result;
    } catch (const Invalid& e) {
        return {{"ok",false},{"operation",operation},{"errors",Json::array({{{"code",e.code},{"path",e.path}}})}};
    }
}
}
