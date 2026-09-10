#include "scv.hpp"

#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"
#include "symphony/knowledge/engine/temporal.hpp"

#include <algorithm>
#include <cstdint>
#include <set>
#include <string_view>

namespace symphony::knowledge::scv {
namespace {
[[noreturn]] void fail(const std::string& code, const std::string& message) {
    throw engine::Error("scv." + code, message, 3);
}
void fields(const Json& value, std::initializer_list<std::string_view> names) {
    if (!value.is_object() || value.size() != names.size()) fail("fields", "unexpected object fields");
    for (auto name : names) if (!value.contains(std::string(name))) fail("fields", "missing required field");
}
std::string str(const Json& value, const std::string& key, std::size_t maximum = 4096, bool empty = false) {
    if (!value.contains(key) || !value.at(key).is_string()) fail("type", key + " must be text");
    auto text = value.at(key).get<std::string>();
    if ((!empty && text.empty()) || text.size() > maximum) fail("bounds", key + " exceeds text bounds");
    return text;
}
bool token(const std::string& value) {
    return !value.empty() && value.size() <= 128 &&
        std::all_of(value.begin(), value.end(), [](unsigned char c) {
            return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
                (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':';
        });
}
void identity(const Json& value, const std::string& name) {
    if (!token(str(value, name, 128))) fail("identity", name + " is not a bounded identity");
}
void texts(const Json& value, const std::string& name, std::size_t bound = 64) {
    const auto& a = value.at(name);
    if (!a.is_array() || a.size() > bound) fail("bounds", name + " must be a bounded array");
    for (const auto& x : a) {
        if (!x.is_string() || x.get_ref<const std::string&>().empty() || x.get_ref<const std::string&>().size() > 4096)
            fail("type", name + " contains invalid text");
    }
}
void uri(const std::string& value) {
    const auto colon = value.find(':');
    const auto alpha = [](unsigned char c) { return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'); };
    if (value.empty() || value.size() > 4096 || colon == std::string::npos || colon == 0 || !alpha(value.front()) ||
        !std::all_of(value.begin(), value.begin() + colon, [&](unsigned char c) {
            return alpha(c) || (c >= '0' && c <= '9') || c == '+' || c == '.' || c == '-';
        }) ||
        std::any_of(value.begin(), value.end(), [](unsigned char c) { return c <= 32 || c == 127; }))
        fail("locator", "locator must have an explicit scheme and no whitespace or controls");
    auto authority = value.find("://");
    if (authority != std::string::npos) {
        auto end = value.find_first_of("/?#", authority + 3);
        if (value.substr(authority + 3, end == std::string::npos ? end : end - authority - 3).find('@') != std::string::npos)
            fail("locator", "credential-bearing locator authority is not supported");
    }
}
void digest(const Json& value, const std::string& field) {
    auto d = str(value, field, 71);
    if (d.size() != 71 || !d.starts_with("sha256:") ||
        !std::all_of(d.begin() + 7, d.end(), [](unsigned char c) {
            return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
        })) fail("digest", "invalid tagged digest");
}
void check_seal(const Json& value, const std::string& field = "digest") {
    digest(value, field);
    if (sealed(value, field).at(field) != value.at(field)) fail("digest", "content does not match its digest");
}
void domain_scope(const Json& value, const std::string& domain) {
    const auto family = str(value, "family_id", 128), provider = str(value, "provider_id", 128);
    if (domain == "scv") return;
    if (domain == "schv" || domain == "scev") {
        if (family != domain) fail("domain", "source is outside this family engine");
        return;
    }
    const auto split = domain.find('-');
    if (split == std::string::npos || family != domain.substr(0, split) || provider != domain.substr(split + 1))
        fail("domain", "source is outside this provider engine");
}
Json desired_part(Json value) {
    for (const auto* name : {"protocol", "generation", "predecessor_digest", "digest"}) value.erase(name);
    return value;
}
void validate_desired(const Json& value, const std::string& domain) {
    fields(value, {"source_id", "provider_id", "family_id", "publisher", "authority_role", "scope", "locators", "continuity_evidence"});
    for (const auto* name : {"source_id", "provider_id", "family_id", "authority_role"}) identity(value, name);
    str(value, "publisher", 1024); str(value, "scope"); texts(value, "continuity_evidence");
    const auto& locators = value.at("locators");
    if (!locators.is_array() || locators.empty() || locators.size() > 16) fail("bounds", "source requires 1 to 16 locators");
    std::set<std::string> ids;
    for (const auto& locator : locators) {
        fields(locator, {"locator_id", "uri", "role", "format", "selector"});
        identity(locator, "locator_id"); identity(locator, "role"); identity(locator, "format");
        uri(str(locator, "uri")); str(locator, "selector", 4096, true);
        if (!ids.insert(str(locator, "locator_id")).second) fail("duplicate", "duplicate locator identity");
    }
    domain_scope(value, domain);
}
Json plan(const Json& payload, const std::string& domain) {
    fields(payload, {"operation_id", "current", "desired", "reason"});
    identity(payload, "operation_id"); str(payload, "reason");
    const auto& current = payload.at("current");
    auto source = payload.at("desired");
    validate_desired(source, domain);
    Json previous = nullptr;
    std::int64_t generation = 1;
    std::string kind = "onboard";
    if (!current.is_null()) {
        validate_source(current, domain);
        for (const auto* key : {"source_id", "provider_id", "family_id"})
            if (current.at(key) != source.at(key)) fail("identity_change", "replacement source/provider requires a new source chain");
        previous = current.at("digest");
        generation = current.at("generation").get<std::int64_t>() + 1;
        if (generation > 9007199254740991LL) fail("generation", "generation is exhausted");
        kind = "revise";
        if (current.at("locators") != source.at("locators")) kind = "relocate";
        if (current.at("publisher") != source.at("publisher") || current.at("authority_role") != source.at("authority_role") ||
            current.at("scope") != source.at("scope")) kind = "authority_change";
    }
    source["protocol"] = "symphony.scv.source.v1";
    source["generation"] = generation;
    source["predecessor_digest"] = previous;
    source = sealed(source);
    return sealed(Json{{"protocol", "symphony.scv.source-plan.v1"}, {"operation_id", payload.at("operation_id")},
        {"expected_state_digest", previous}, {"change_kind", kind}, {"reason", payload.at("reason")}, {"source", source}}, "plan_digest");
}
Json apply(const Json& payload, const std::string& domain) {
    fields(payload, {"plan", "current"});
    const auto& proposal = payload.at("plan");
    fields(proposal, {"protocol", "operation_id", "expected_state_digest", "change_kind", "reason", "source", "plan_digest"});
    if (proposal.at("protocol") != "symphony.scv.source-plan.v1") fail("protocol", "unsupported source plan");
    check_seal(proposal, "plan_digest");
    validate_source(proposal.at("source"), domain);
    const auto& current = payload.at("current");
    Json expected = nullptr;
    if (!current.is_null()) { validate_source(current, domain); expected = current.at("digest"); }
    if (proposal.at("expected_state_digest") != expected) fail("stale_state", "source head differs from the confirmed plan");
    auto rebuilt = plan(Json{{"operation_id", proposal.at("operation_id")}, {"current", current},
        {"desired", desired_part(proposal.at("source"))}, {"reason", proposal.at("reason")}}, domain);
    if (rebuilt != proposal) fail("plan_mismatch", "plan does not describe this exact source transition");
    return Json{{"protocol", "symphony.scv.source-transition.v1"}, {"operation_id", proposal.at("operation_id")},
        {"expected_state_digest", expected}, {"state", proposal.at("source")}, {"state_digest", proposal.at("source").at("digest")}};
}
Json import_capture(const Json& payload, const std::string& domain) {
    fields(payload, {"source", "locator_id", "resolved_uri", "redirects", "observed_at", "upstream_revision",
        "media_type", "body", "completeness", "issues"});
    auto result = payload;
    const auto body = str(payload, "body", engine::Limits::max_string_bytes, true);
    result["protocol"] = "symphony.scv.capture.v1";
    result["body_digest"] = engine::tagged_sha256(body);
    result["byte_size"] = body.size();
    result = sealed(result);
    validate_capture(result, domain);
    return result;
}
Json onboard(const Json& payload, const std::string& domain) {
    fields(payload, {"provider_id", "family_id", "display_name", "sources"});
    identity(payload, "provider_id"); identity(payload, "family_id"); str(payload, "display_name", 1024);
    domain_scope(payload, domain);
    if (!payload.at("sources").is_array() || payload.at("sources").empty() || payload.at("sources").size() > 32)
        fail("bounds", "provider onboarding requires 1 to 32 selected sources");
    std::set<std::string> sources;
    for (const auto& source : payload.at("sources")) {
        validate_desired(source, domain);
        if (source.at("provider_id") != payload.at("provider_id") || source.at("family_id") != payload.at("family_id"))
            fail("domain", "provider source belongs to another domain");
        if (!sources.insert(str(source, "source_id")).second) fail("duplicate", "duplicate source identity");
    }
    auto result = payload;
    result["protocol"] = "symphony.scv.provider.v1";
    result["disposition"] = "candidate";
    result["interpretation_scope"] = "captured-text-and-explicit-assertions";
    return sealed(result);
}
}

Json sealed(Json value, std::string field) {
    value.erase(field);
    value[field] = engine::tagged_sha256(value.dump());
    return value;
}
void validate_source(const Json& source, const std::string& domain) {
    fields(source, {"protocol", "source_id", "provider_id", "family_id", "publisher", "authority_role", "scope",
        "locators", "continuity_evidence", "generation", "predecessor_digest", "digest"});
    if (source.at("protocol") != "symphony.scv.source.v1") fail("protocol", "unsupported source version");
    validate_desired(desired_part(source), domain);
    if (!source.at("generation").is_number_integer()) fail("generation", "source generation must be an integer");
    const auto generation = source.at("generation").get<std::int64_t>();
    if (generation < 1 || generation > 9007199254740991LL) fail("generation", "invalid source generation");
    if (generation == 1 && !source.at("predecessor_digest").is_null()) fail("generation", "initial source cannot have a predecessor");
    if (generation > 1) digest(source, "predecessor_digest");
    check_seal(source);
}
void validate_capture(const Json& capture, const std::string& domain) {
    fields(capture, {"protocol", "source", "locator_id", "resolved_uri", "redirects", "observed_at", "upstream_revision",
        "media_type", "body", "body_digest", "byte_size", "completeness", "issues", "digest"});
    if (capture.at("protocol") != "symphony.scv.capture.v1") fail("protocol", "unsupported capture version");
    validate_source(capture.at("source"), domain);
    identity(capture, "locator_id");
    const auto& locators = capture.at("source").at("locators");
    if (std::none_of(locators.begin(), locators.end(), [&](const auto& l) { return l.at("locator_id") == capture.at("locator_id"); }))
        fail("locator", "capture locator is not in the selected source revision");
    uri(str(capture, "resolved_uri")); texts(capture, "redirects", 8);
    for (const auto& location : capture.at("redirects")) uri(location.get<std::string>());
    if (!engine::is_utc_seconds(str(capture, "observed_at", 20))) fail("time", "observed_at must be strict UTC seconds");
    if (!capture.at("upstream_revision").is_null()) {
        fields(capture.at("upstream_revision"), {"scheme", "value"});
        identity(capture.at("upstream_revision"), "scheme"); str(capture.at("upstream_revision"), "value");
    }
    str(capture, "media_type", 128);
    const auto body = str(capture, "body", engine::Limits::max_string_bytes, true);
    if (!capture.at("byte_size").is_number_integer() || capture.at("byte_size") != Json(body.size()))
        fail("capture_size", "capture byte size does not bind its body");
    if (body.find('\0') != std::string::npos) fail("capture_encoding", "capture profile accepts UTF-8 text, not binary content");
    digest(capture, "body_digest");
    if (capture.at("body_digest") != engine::tagged_sha256(body)) fail("capture_digest", "body digest mismatch");
    const auto completeness = str(capture, "completeness", 32);
    if (completeness != "complete" && completeness != "partial" && completeness != "failed") fail("completeness", "invalid capture completeness");
    texts(capture, "issues");
    if (completeness != "complete" && capture.at("issues").empty()) fail("completeness", "incomplete capture requires an explicit issue");
    check_seal(capture);
}
Json handle_source(const engine::Request& request, const std::string& domain) {
    if (engine::unix_time_ms() > request.deadline_unix_ms) throw engine::Error("request.deadline", "request deadline exceeded", 4);
    const auto& p = request.payload;
    if (request.operation == "provider_onboard") return onboard(p, domain);
    if (request.operation == "source_plan") return plan(p, domain);
    if (request.operation == "source_apply") return apply(p, domain);
    if (request.operation == "source_status") {
        fields(p, {"source"});
        if (!p.at("source").is_null()) validate_source(p.at("source"), domain);
        return Json{{"protocol", "symphony.scv.source-status.v1"}, {"source", p.at("source")},
            {"state_digest", p.at("source").is_null() ? Json(nullptr) : p.at("source").at("digest")}};
    }
    if (request.operation == "capture_import") return import_capture(p, domain);
    if (request.operation == "capture_compare") {
        fields(p, {"before", "after"});
        const auto& a = p.at("before"); const auto& b = p.at("after");
        validate_capture(a, domain); validate_capture(b, domain);
        const bool identity_preserved = a.at("source").at("source_id") == b.at("source").at("source_id") &&
            a.at("source").at("provider_id") == b.at("source").at("provider_id") && a.at("source").at("family_id") == b.at("source").at("family_id");
        const bool changed = a.at("body_digest") != b.at("body_digest");
        const bool config = a.at("source").at("digest") != b.at("source").at("digest");
        const bool representation = a.at("media_type") != b.at("media_type");
        auto kind = !identity_preserved ? "source_changed" : representation ? "representation_changed" : changed ? "content_changed" : "retrieval_only";
        return sealed(Json{{"protocol", "symphony.scv.capture-diff.v1"}, {"before_digest", a.at("digest")},
            {"after_digest", b.at("digest")}, {"change_kind", kind}, {"source_identity_preserved", identity_preserved},
            {"body_changed", changed}, {"configuration_changed", config}, {"coverage_before", a.at("completeness")},
            {"coverage_after", b.at("completeness")}, {"requires_reinterpretation", changed || config || representation || a.at("completeness") != b.at("completeness")}});
    }
    throw engine::Error("operation.unsupported", "unsupported source operation", 4);
}
}
