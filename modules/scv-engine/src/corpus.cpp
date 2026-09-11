#include "corpus.hpp"
#include "scv.hpp"

#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/temporal.hpp"

#include <algorithm>
#include <chrono>
#include <cstdint>
#include <map>
#include <set>
#include <string_view>

namespace symphony::knowledge::scv {
namespace {
constexpr std::size_t maximum_members = 128;
constexpr std::int64_t maximum_integer = 9007199254740991LL;
constexpr std::int64_t maximum_age = 3155760000LL;
using Members = std::map<std::string, Json>;

[[noreturn]] void invalid(const std::string& message) {
    throw engine::Error("corpus.invalid", message, 4);
}
void fields(const Json& value, std::initializer_list<std::string_view> names) {
    if (!value.is_object() || value.size() != names.size()) invalid("unexpected object fields");
    for (const auto name : names) if (!value.contains(std::string(name))) invalid("missing required field: " + std::string(name));
}
std::string text(const Json& value, const std::string& key, std::size_t bound = 128) {
    if (!value.contains(key) || !value.at(key).is_string()) invalid("expected text: " + key);
    const auto result = value.at(key).get<std::string>();
    if (result.empty() || result.size() > bound || result.find('\0') != std::string::npos) invalid("text bound: " + key);
    return result;
}
void opaque_id(const Json& value, const std::string& key) {
    const auto s = text(value, key);
    if (std::any_of(s.begin(), s.end(), [](unsigned char c) { return c < 32 || c == 127; })) invalid("control character in identity");
}
void source_id(const Json& value, const std::string& key) {
    const auto s = text(value, key);
    if (!std::all_of(s.begin(), s.end(), [](unsigned char c) {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
               c == '-' || c == '_' || c == '.' || c == ':';
    })) invalid("invalid source identity: " + key);
}
std::int64_t integer(const Json& value, std::int64_t minimum, std::int64_t maximum) {
    if (!value.is_number_integer() ||
        (value.is_number_unsigned() && value.get<std::uint64_t>() > static_cast<std::uint64_t>(maximum)))
        invalid("expected bounded integer");
    const auto number = value.get<std::int64_t>();
    if (number < minimum || number > maximum) invalid("integer outside bounds");
    return number;
}
void digest(const Json& value, const std::string& key) {
    const auto s = text(value, key, 71);
    if (s.size() != 71 || !s.starts_with("sha256:") || !std::all_of(s.begin() + 7, s.end(), [](unsigned char c) {
        return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
    })) invalid("invalid tagged digest: " + key);
}
void seal(const Json& value) {
    digest(value, "digest");
    if (sealed(value) != value) invalid("content does not match digest");
}
std::int64_t utc(const Json& value) {
    if (!value.is_string() || !engine::is_utc_seconds(value.get_ref<const std::string&>())) invalid("expected canonical UTC seconds");
    const auto& s = value.get_ref<const std::string&>();
    const auto date = std::chrono::sys_days(std::chrono::year(std::stoi(s.substr(0, 4))) /
        static_cast<unsigned>(std::stoi(s.substr(5, 2))) / static_cast<unsigned>(std::stoi(s.substr(8, 2))));
    return std::chrono::duration_cast<std::chrono::seconds>(date.time_since_epoch()).count() +
        std::stoi(s.substr(11, 2)) * 3600 + std::stoi(s.substr(14, 2)) * 60 + std::stoi(s.substr(17, 2));
}
void array(const Json& value, std::size_t maximum) {
    if (!value.is_array() || value.size() > maximum) invalid("expected bounded array");
}
void uri(const std::string& s) {
    const auto colon = s.find(':');
    const auto alpha = [](unsigned char c) { return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'); };
    if (colon == std::string::npos || colon == 0 || !alpha(s.front()) ||
        !std::all_of(s.begin(), s.begin() + colon, [&](unsigned char c) {
            return alpha(c) || (c >= '0' && c <= '9') || c == '+' || c == '.' || c == '-';
        }) || std::any_of(s.begin(), s.end(), [](unsigned char c) { return c <= 32 || c == 127; })) invalid("invalid requested URI");
    const auto authority = s.find("://");
    if (authority != std::string::npos) {
        const auto end = s.find_first_of("/?#", authority + 3);
        if (s.substr(authority + 3, end == std::string::npos ? end : end - authority - 3).find('@') != std::string::npos)
            invalid("credential-bearing requested URI");
    }
}
void scope(const Json& index, const std::string& domain) {
    if (domain == "scv") return;
    const auto family = text(index, "family_id"), provider = text(index, "provider_id");
    if (domain == "schv" || domain == "scev") {
        if (family != domain) invalid("index outside family domain");
    } else {
        const auto split = domain.find('-');
        if (split == std::string::npos || family != domain.substr(0, split) || provider != domain.substr(split + 1))
            invalid("index outside provider domain");
    }
}
Json member_identity(const Json& index) {
    return Json::array({index.at("family_id"), index.at("provider_id"), index.at("source_id"), index.at("locator_id")});
}
struct References {
    std::map<std::string, Json> owners, captures, sources, locators;
    void add(const Json& index) {
        const auto bind = [](auto& map, const std::string& key, const Json& value, const std::string& message) {
            const auto [found, inserted] = map.emplace(key, value);
            if (!inserted && found->second != value) invalid(message);
        };
        bind(owners, text(index, "source_id"), Json::array({index.at("family_id"), index.at("provider_id")}),
            "source_id is ambiguous across owner domains");
        auto projection = index;
        projection.erase("domain"); projection.erase("digest");
        bind(captures, text(index, "capture_digest", 71), projection, "capture digest has inconsistent index projections");
        bind(sources, text(index, "source_digest", 71), Json::array({index.at("family_id"), index.at("provider_id"),
            index.at("source_id"), index.at("source_generation")}), "source digest has inconsistent source identity");
        bind(locators, Json::array({index.at("source_digest"), index.at("locator_id")}).dump(),
            index.at("requested_uri"), "source locator has inconsistent requested URI");
    }
};
Members members(const Json& corpus) {
    Members result;
    for (const auto& member : corpus.at("members")) result.emplace(text(member, "member_id"), member);
    return result;
}
Json coverage(const Json& selected) {
    std::size_t complete = 0, partial = 0, failed = 0, retained = 0;
    for (const auto& member : selected) {
        const auto& attempt = member.at("latest_attempt");
        if (attempt.at("completeness") == "complete") ++complete;
        else if (attempt.at("completeness") == "partial") ++partial;
        else ++failed;
        if (attempt.at("completeness") != "complete" && !member.at("last_complete").is_null()) ++retained;
    }
    return {{"requested_members", selected.size()}, {"complete_attempts", complete}, {"partial_attempts", partial},
        {"failed_attempts", failed}, {"retained_complete_members", retained}};
}
Json capture_index(const Json& payload, const std::string& domain) {
    fields(payload, {"capture"});
    const auto& capture = payload.at("capture");
    validate_capture(capture, domain);
    const auto& source = capture.at("source");
    Json result = {{"protocol", "symphony.scv.capture-index.v1"}, {"domain", domain},
        {"source_digest", source.at("digest")}, {"source_generation", source.at("generation")},
        {"capture_digest", capture.at("digest")}};
    for (const auto* field : {"source_id", "provider_id", "family_id"}) result[field] = source.at(field);
    for (const auto* field : {"locator_id", "body_digest", "byte_size", "observed_at", "upstream_revision", "media_type", "completeness", "issues"})
        result[field] = capture.at(field);
    for (const auto& locator : source.at("locators"))
        if (locator.at("locator_id") == capture.at("locator_id")) result["requested_uri"] = locator.at("uri");
    result = sealed(result);
    validate_capture_index(result, domain);
    return result;
}
Json build(const Json& payload, const std::string& domain) {
    fields(payload, {"corpus_id", "previous", "snapshot_time", "attempts"});
    opaque_id(payload, "corpus_id");
    const auto time = utc(payload.at("snapshot_time"));
    const auto& previous = payload.at("previous");
    std::int64_t generation = 1;
    Members old;
    References references;
    if (!previous.is_null()) {
        validate_corpus(previous, domain);
        if (previous.at("corpus_id") != payload.at("corpus_id")) invalid("predecessor corpus identity differs");
        if (utc(previous.at("snapshot_time")) > time) invalid("snapshot precedes predecessor time");
        generation = integer(previous.at("generation"), 1, maximum_integer - 1) + 1;
        old = members(previous);
        for (const auto& [id, member] : old) {
            static_cast<void>(id);
            references.add(member.at("latest_attempt"));
            if (!member.at("last_complete").is_null()) references.add(member.at("last_complete"));
        }
    }
    array(payload.at("attempts"), maximum_members);
    Members selected;
    std::set<std::string> identities;
    for (const auto& item : payload.at("attempts")) {
        fields(item, {"member_id", "capture"}); opaque_id(item, "member_id");
        const auto id = text(item, "member_id");
        const auto& index = item.at("capture");
        validate_capture_index(index, domain); references.add(index);
        if (utc(index.at("observed_at")) > time) invalid("snapshot precedes an acquisition observation");
        const auto identity = member_identity(index);
        if (!identities.insert(identity.dump()).second) invalid("duplicate member source/locator identity");
        const auto prior = old.find(id);
        if (prior != old.end() && member_identity(prior->second.at("latest_attempt")) != identity)
            invalid("member_id cannot be rebound to another source or locator");
        Json last = nullptr;
        if (index.at("completeness") == "complete") last = index;
        else if (prior != old.end()) last = prior->second.at("last_complete");
        if (!selected.emplace(id, Json{{"member_id", id}, {"latest_attempt", index}, {"last_complete", last}}).second)
            invalid("duplicate member_id");
    }
    Json list = Json::array();
    for (const auto& [id, value] : selected) { static_cast<void>(id); list.push_back(value); }
    auto result = sealed(Json{{"protocol", "symphony.scv.corpus.v1"}, {"domain", domain}, {"corpus_id", payload.at("corpus_id")},
        {"generation", generation}, {"parent_digest", previous.is_null() ? Json(nullptr) : previous.at("digest")},
        {"snapshot_time", payload.at("snapshot_time")}, {"members", list}, {"coverage", coverage(list)}});
    validate_corpus(result, domain);
    return result;
}
Json query(const Json& payload, const std::string& domain) {
    fields(payload, {"corpus", "query_time", "member_ids", "selection", "max_age_seconds"});
    const auto& corpus = payload.at("corpus"); validate_corpus(corpus, domain);
    const auto time = utc(payload.at("query_time"));
    const auto selection = text(payload, "selection");
    if (selection != "latest_attempt" && selection != "last_complete") invalid("explicit supported selection required");
    const auto& age = payload.at("max_age_seconds");
    const auto maximum = age.is_null() ? maximum_age : integer(age, 0, maximum_age);
    array(payload.at("member_ids"), maximum_members);
    const auto available = members(corpus);
    std::set<std::string> requested;
    for (const auto& id : payload.at("member_ids")) {
        opaque_id(Json{{"id", id}}, "id");
        const auto name = id.get<std::string>();
        if (!available.contains(name)) invalid("unknown requested member_id: " + name);
        if (!requested.insert(name).second) invalid("duplicate requested member_id");
    }
    if (requested.empty()) for (const auto& [id, unused] : available) { static_cast<void>(unused); requested.insert(id); }
    Json result_members = Json::array(), subset = Json::array();
    for (const auto& id : requested) {
        const auto& member = available.at(id); subset.push_back(member);
        const auto& attempt = member.at("latest_attempt");
        const auto& selected = member.at(selection);
        Json reasons = Json::array();
        std::string status = "unavailable", freshness = "not_selected";
        Json source_matches = nullptr;
        if (attempt.at("completeness") == "partial") reasons.push_back("latest_attempt_partial");
        if (attempt.at("completeness") == "failed") reasons.push_back("latest_attempt_failed");
        if (selected.is_null()) reasons.push_back("no_complete_capture");
        else {
            status = selected.at("completeness").get<std::string>();
            source_matches = selected.at("source_digest") == attempt.at("source_digest");
            if (selected.at("capture_digest") != attempt.at("capture_digest")) reasons.push_back("selected_previous_complete");
            if (source_matches == false) reasons.push_back("source_revision_differs_from_latest_attempt");
            const auto observed = utc(selected.at("observed_at"));
            if (observed > time) { freshness = "future"; reasons.push_back("observation_after_query_time"); }
            else if (!age.is_null() && time - observed > maximum) { freshness = "expired"; reasons.push_back("maximum_age_exceeded"); }
            else freshness = "current";
        }
        result_members.push_back(Json{{"member_id", id}, {"latest_attempt_digest", attempt.at("capture_digest")},
            {"selected", selected}, {"status", status}, {"freshness", freshness},
            {"source_revision_matches_latest", source_matches}, {"reasons", reasons}});
    }
    return sealed(Json{{"protocol", "symphony.scv.corpus-query.v1"}, {"domain", domain}, {"corpus_digest", corpus.at("digest")},
        {"query_time", payload.at("query_time")}, {"selection", selection}, {"max_age_seconds", age},
        {"members", result_members}, {"coverage", coverage(subset)}});
}
Json difference(const Json& payload, const std::string& domain) {
    fields(payload, {"before", "after"});
    const auto& before = payload.at("before"); const auto& after = payload.at("after");
    validate_corpus(before, domain); validate_corpus(after, domain);
    if (before.at("corpus_id") != after.at("corpus_id")) invalid("cannot compare different corpus identities");
    const auto old = members(before), current = members(after);
    References references;
    for (const auto* snapshot : {&before, &after}) for (const auto& member : snapshot->at("members")) {
        references.add(member.at("latest_attempt"));
        if (!member.at("last_complete").is_null()) references.add(member.at("last_complete"));
    }
    Json added = Json::array(), removed = Json::array(), changes = Json::array();
    std::set<std::string> affected;
    for (const auto& [id, member] : old) {
        static_cast<void>(member);
        if (!current.contains(id)) { removed.push_back(id); affected.insert(id); }
    }
    for (const auto& [id, member] : current) {
        const auto prior = old.find(id);
        if (prior == old.end()) { added.push_back(id); affected.insert(id); continue; }
        const auto& a = prior->second.at("latest_attempt"); const auto& b = member.at("latest_attempt");
        if (member_identity(a) != member_identity(b)) invalid("member_id has incompatible source/locator identity");
        const bool body = a.at("body_digest") != b.at("body_digest") || a.at("byte_size") != b.at("byte_size");
        const bool source = a.at("source_digest") != b.at("source_digest");
        const bool observation = a.at("capture_digest") != b.at("capture_digest") || a.at("domain") != b.at("domain");
        const bool covered = a.at("completeness") != b.at("completeness") || a.at("issues") != b.at("issues");
        const bool last = prior->second.at("last_complete") != member.at("last_complete");
        if (body || source || observation || covered || last) {
            changes.push_back(Json{{"member_id", id}, {"body_changed", body}, {"source_changed", source},
                {"observation_changed", observation}, {"coverage_changed", covered}, {"last_complete_changed", last}});
            affected.insert(id);
        }
    }
    Json affected_ids = Json::array(); for (const auto& id : affected) affected_ids.push_back(id);
    return sealed(Json{{"protocol", "symphony.scv.corpus-diff.v1"}, {"domain", domain}, {"before_digest", before.at("digest")},
        {"after_digest", after.at("digest")}, {"added_member_ids", added}, {"removed_member_ids", removed},
        {"changes", changes}, {"affected_member_ids", affected_ids}});
}
}

void validate_capture_index(const Json& index, const std::string& domain) {
    fields(index, {"protocol", "domain", "source_id", "provider_id", "family_id", "source_digest", "source_generation",
        "locator_id", "requested_uri", "capture_digest", "body_digest", "byte_size", "observed_at", "upstream_revision",
        "media_type", "completeness", "issues", "digest"});
    if (index.at("protocol") != "symphony.scv.capture-index.v1") invalid("unsupported capture index version");
    for (const auto* field : {"source_id", "provider_id", "family_id", "locator_id"}) source_id(index, field);
    for (const auto* field : {"source_digest", "capture_digest", "body_digest"}) digest(index, field);
    integer(index.at("source_generation"), 1, maximum_integer);
    integer(index.at("byte_size"), 0, engine::Limits::max_string_bytes);
    uri(text(index, "requested_uri", 4096)); utc(index.at("observed_at"));
    text(index, "media_type");
    const auto completeness = text(index, "completeness", 32);
    if (completeness != "complete" && completeness != "partial" && completeness != "failed") invalid("unknown capture completeness");
    array(index.at("issues"), 64);
    for (const auto& issue : index.at("issues")) text(Json{{"issue", issue}}, "issue", 4096);
    if (completeness != "complete" && index.at("issues").empty()) invalid("incomplete capture requires an issue");
    if (!index.at("upstream_revision").is_null()) {
        const auto& revision = index.at("upstream_revision");
        fields(revision, {"scheme", "value"}); source_id(revision, "scheme"); text(revision, "value", 4096);
    }
    scope(index, text(index, "domain")); scope(index, domain); seal(index);
}
void validate_corpus(const Json& corpus, const std::string& domain) {
    fields(corpus, {"protocol", "domain", "corpus_id", "generation", "parent_digest", "snapshot_time", "members", "coverage", "digest"});
    if (corpus.at("protocol") != "symphony.scv.corpus.v1" || corpus.at("domain") != domain) invalid("corpus protocol or domain mismatch");
    opaque_id(corpus, "corpus_id");
    const auto generation = integer(corpus.at("generation"), 1, maximum_integer);
    if (generation == 1) { if (!corpus.at("parent_digest").is_null()) invalid("initial corpus cannot have a predecessor"); }
    else digest(corpus, "parent_digest");
    const auto time = utc(corpus.at("snapshot_time"));
    array(corpus.at("members"), maximum_members);
    std::string previous_id;
    bool first = true;
    std::set<std::string> identities;
    References references;
    for (const auto& member : corpus.at("members")) {
        fields(member, {"member_id", "latest_attempt", "last_complete"}); opaque_id(member, "member_id");
        const auto id = text(member, "member_id");
        if (!first && id <= previous_id) invalid("corpus member IDs must be unique and sorted");
        previous_id = id; first = false;
        const auto& latest = member.at("latest_attempt");
        const auto& last = member.at("last_complete");
        validate_capture_index(latest, domain); references.add(latest);
        if (utc(latest.at("observed_at")) > time) invalid("snapshot precedes latest attempt");
        if (!identities.insert(member_identity(latest).dump()).second) invalid("duplicate corpus member source/locator");
        if (latest.at("completeness") == "complete") {
            if (last != latest) invalid("complete latest attempt must be the last complete capture");
        } else if (generation == 1 && !last.is_null()) invalid("initial incomplete attempt cannot invent retained evidence");
        if (!last.is_null()) {
            validate_capture_index(last, domain); references.add(last);
            if (last.at("completeness") != "complete" || member_identity(last) != member_identity(latest))
                invalid("retained complete capture must belong to the same member");
            if (utc(last.at("observed_at")) > time) invalid("snapshot precedes retained evidence");
        }
    }
    if (corpus.at("coverage") != coverage(corpus.at("members"))) invalid("corpus coverage does not match selected attempts");
    seal(corpus);
}
Json handle_corpus(const engine::Request& request, const std::string& domain) {
    const auto deadline = [&] {
        if (engine::unix_time_ms() > request.deadline_unix_ms)
            throw engine::Error("request.deadline_exceeded", "corpus operation deadline exceeded", 4);
    };
    deadline();
    Json result;
    if (request.operation == "capture_index") result = capture_index(request.payload, domain);
    else if (request.operation == "corpus_build") result = build(request.payload, domain);
    else if (request.operation == "corpus_query") result = query(request.payload, domain);
    else if (request.operation == "corpus_diff") result = difference(request.payload, domain);
    else throw engine::Error("operation.unsupported", "unsupported corpus operation", 4);
    deadline();
    return result;
}
}
