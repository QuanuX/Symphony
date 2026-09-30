#include <symphony/snv/sciv.hpp>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>

#include <algorithm>
#include <chrono>
#include <limits>
#include <map>
#include <set>
#include <string>
#include <vector>

namespace symphony::snv::sciv {
namespace {
namespace engine = symphony::knowledge::engine;

[[noreturn]] void fail(std::string code, std::string message) {
    throw engine::Error(std::move(code), std::move(message), 2);
}
void deadline(std::int64_t limit) {
    const auto now = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    if (limit <= now) fail("deadline_exceeded", "SCIV operation deadline expired");
}
void fields(const Json& value, std::initializer_list<std::string_view> required,
            std::initializer_list<std::string_view> optional = {}) {
    if (!value.is_object()) fail("invalid_payload", "SCIV expects an object");
    for (const auto field : required)
        if (!value.contains(std::string(field))) fail("invalid_payload", "SCIV required field missing: " + std::string(field));
    for (const auto& item : value.items()) {
        const auto allowed = [&](auto list) { return std::find(list.begin(), list.end(), item.key()) != list.end(); };
        if (!allowed(required) && !allowed(optional)) fail("invalid_payload", "SCIV unknown field: " + item.key());
    }
}
std::string text(const Json& value, std::string_view field, std::size_t max = 256) {
    const auto& item = value.at(std::string(field));
    if (!item.is_string()) fail("invalid_payload", "SCIV string field required: " + std::string(field));
    auto result = item.get<std::string>();
    if (result.empty() || result.size() > max) fail("invalid_payload", "SCIV string length outside bound: " + std::string(field));
    for (const unsigned char c : result)
        if (c < 0x20 || c == 0x7f) fail("invalid_payload", "SCIV identifier contains a control byte");
    return result;
}
std::int64_t integer(const Json& value, std::string_view field) {
    const auto& item = value.at(std::string(field));
    if (!item.is_number_integer() || (item.is_number_unsigned() && item.get<std::uint64_t>() > 9007199254740991ULL))
        fail("invalid_payload", "SCIV requires a bounded integer: " + std::string(field));
    const auto result = item.get<std::int64_t>();
    if (result < 0 || result > 9007199254740991LL) fail("invalid_payload", "SCIV requires nonnegative interoperable time and limits");
    return result;
}
std::string choice(const Json& value, std::string_view field, std::initializer_list<std::string_view> choices) {
    const auto result = text(value, field);
    if (std::find(choices.begin(), choices.end(), result) == choices.end()) fail("invalid_payload", "SCIV unsupported field value: " + std::string(field));
    return result;
}
void digest(const Json& value, std::string_view field) {
    const auto str = text(value, field, 71);
    if (str.size() != 71 || !str.starts_with("sha256:") ||
        !std::all_of(str.begin() + 7, str.end(), [](char c) { return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'); }))
        fail("invalid_payload", "SCIV digest must use lowercase tagged SHA256");
}
void source(const Json& value) {
    fields(value, {"source_id", "source_revision", "method", "method_version"}, {"source_digest"});
    for (const auto field : {"source_id", "source_revision", "method", "method_version"}) (void)text(value, field);
    if (value.contains("source_digest")) digest(value, "source_digest");
}
std::string source_key(const Json& value) {
    // Capture revisions change with time; they never make another source's
    // competing observation disappear. Method identity scopes one time series.
    return Json::array({value.at("source_id"), value.at("method"), value.at("method_version")}).dump();
}
void array(const Json& value, std::string_view field) {
    if (!value.at(std::string(field)).is_array()) fail("invalid_payload", "SCIV array required: " + std::string(field));
}

struct Input {
    const Json& raw;
    std::size_t capacity = 2048;
    std::int64_t at = 0;
    bool timed = false;
    std::int64_t max_age = 0;
    bool directed = false;
    bool transitive = false;
    std::string cluster;
    std::string system;
    std::map<std::string, Json> nodes;
    std::set<std::string> buses;
    std::set<std::string> record_ids;

    explicit Input(const Json& value, std::int64_t limit) : raw(value) {
        fields(raw, {"protocol", "profile", "cluster", "nodes", "memberships", "observations"}, {"record_limit", "at_unix_ms"});
        if (text(raw, "protocol") != "symphony.snv.sciv.evidence.v1") fail("unsupported_protocol", "SCIV evidence protocol unsupported");
        if (raw.contains("record_limit")) {
            const auto count = integer(raw, "record_limit");
            if (count != 512 && count != 1024 && count != 2048) fail("invalid_payload", "SCIV record_limit must be 512, 1024 or 2048");
            capacity = static_cast<std::size_t>(count);
        }
        if (raw.contains("at_unix_ms")) { at = integer(raw, "at_unix_ms"); timed = true; }
        fields(raw.at("profile"), {"direction", "transitive", "max_age_ms"});
        directed = choice(raw.at("profile"), "direction", {"directed", "undirected"}) == "directed";
        if (!raw.at("profile").at("transitive").is_boolean()) fail("invalid_payload", "SCIV transitive profile requires boolean");
        transitive = raw.at("profile").at("transitive").get<bool>();
        max_age = integer(raw.at("profile"), "max_age_ms");
        fields(raw.at("cluster"), {"cluster_id", "system_id"});
        cluster = text(raw.at("cluster"), "cluster_id"); system = text(raw.at("cluster"), "system_id");
        for (const auto field : {"nodes", "memberships", "observations"}) array(raw, field);
        if (raw.at("nodes").size() + raw.at("memberships").size() + raw.at("observations").size() > capacity)
            fail("capacity_exceeded", "SCIV aggregate record capacity exceeded");
        std::set<std::string> incarnations;
        for (const auto& node : raw.at("nodes")) {
            deadline(limit);
            fields(node, {"node_id", "system_id"}, {"incarnation_id"});
            const auto id = text(node, "node_id");
            if (text(node, "system_id") != system) fail("reference_mismatch", "SCIV Node system differs from cluster system");
            if (!nodes.emplace(id, node).second) fail("reference_conflict", "SCIV duplicate Node identifier");
            if (node.contains("incarnation_id") && !incarnations.insert(text(node, "incarnation_id")).second)
                fail("reference_conflict", "SCIV incarnation refers to multiple Nodes in one system");
        }
        for (const auto& record : raw.at("memberships")) { deadline(limit); membership(record); }
        for (const auto& record : raw.at("observations")) { deadline(limit); observation(record); }
    }
    void record_id(const Json& record) {
        if (!record_ids.insert(text(record, "record_id")).second) fail("reference_conflict", "SCIV duplicate evidence record identifier");
        source(record.at("source"));
    }
    void node_ref(const Json& record, std::string_view field) const {
        if (!nodes.contains(text(record, field))) fail("reference_mismatch", "SCIV record refers to an undeclared Node");
    }
    void membership(const Json& record) {
        fields(record, {"record_id", "node_id", "state", "source"}, {"incarnation_id", "effective_unix_ms"});
        record_id(record); node_ref(record, "node_id");
        (void)choice(record, "state", {"intended", "joined", "removed"});
        if (record.contains("effective_unix_ms")) (void)integer(record, "effective_unix_ms");
        if (record.contains("incarnation_id")) {
            const auto& node = nodes.at(text(record, "node_id"));
            if (!node.contains("incarnation_id") || text(record, "incarnation_id") != text(node, "incarnation_id"))
                fail("reference_mismatch", "SCIV membership incarnation differs from its Node reference");
        }
    }
    void observation(const Json& record) {
        fields(record, {"record_id", "bus_id", "from_node_id", "to_node_id", "state", "coverage", "source"},
               {"observed_unix_ms", "valid_until_unix_ms"});
        record_id(record); node_ref(record, "from_node_id"); node_ref(record, "to_node_id");
        if (text(record, "from_node_id") == text(record, "to_node_id")) fail("invalid_payload", "SCIV self link does not establish a cluster");
        buses.insert(text(record, "bus_id"));
        (void)choice(record, "state", {"connected", "disconnected", "unknown"});
        (void)choice(record, "coverage", {"complete", "partial"});
        if (record.contains("observed_unix_ms")) (void)integer(record, "observed_unix_ms");
        if (record.contains("valid_until_unix_ms")) {
            if (!record.contains("observed_unix_ms") || integer(record, "valid_until_unix_ms") <= integer(record, "observed_unix_ms"))
                fail("invalid_payload", "SCIV observation validity must follow its observation time");
        }
    }
};

struct Current {
    std::string state = "unobserved";
    std::vector<const Json*> records;
    bool stale = false;
    bool undated = false;
    bool partial = false;
};

Current current(const std::vector<const Json*>& records, const Input& input, std::string_view time_field, bool freshness) {
    Current result;
    if (!input.timed) { result.state = "evaluation_time_unknown"; return result; }
    std::map<std::string, std::vector<const Json*>> by_source;
    for (const auto* record : records) {
        if (!record->contains(std::string(time_field))) { result.undated = true; continue; }
        const auto time = integer(*record, time_field);
        if (time > input.at) continue;
        auto& selected = by_source[source_key(record->at("source"))];
        if (!selected.empty()) {
            const auto previous = integer(*selected.front(), time_field);
            if (time < previous) continue;
            if (time > previous) selected.clear();
        }
        selected.push_back(record);
    }
    std::set<std::string> states;
    for (const auto& [key, selected] : by_source) {
        (void)key;
        for (const auto* record : selected) {
            const auto time = integer(*record, time_field);
            if (freshness && (input.at - time > input.max_age ||
                (record->contains("valid_until_unix_ms") && input.at >= integer(*record, "valid_until_unix_ms")))) {
                result.stale = true; continue;
            }
            states.insert(record->at("state").get<std::string>());
            if (freshness && record->at("coverage") == "partial") result.partial = true;
            result.records.push_back(record);
        }
    }
    if (states.size() > 1) result.state = "conflicting";
    else if (!states.empty()) result.state = *states.begin();
    else if (result.stale) result.state = "stale";
    else if (result.undated) result.state = "undated";
    std::sort(result.records.begin(), result.records.end(), [](const auto* a, const auto* b) { return a->at("record_id") < b->at("record_id"); });
    return result;
}

Json record_refs(const Current& value) {
    auto output = Json::array();
    for (const auto* record : value.records) output.push_back(record->at("record_id"));
    return output;
}

Json validate(const Json& value, std::int64_t limit) {
    Input input(value, limit);
    std::set<std::string> subject_ids{input.cluster, input.system};
    auto memberships = Json::array();
    std::vector<std::string> eligible;
    bool membership_incomplete = false;
    for (const auto& [id, node] : input.nodes) {
        deadline(limit); subject_ids.insert(id);
        if (node.contains("incarnation_id")) subject_ids.insert(text(node, "incarnation_id"));
        std::vector<const Json*> records;
        for (const auto& record : value.at("memberships")) if (record.at("node_id") == id) records.push_back(&record);
        const auto finding = current(records, input, "effective_unix_ms", false);
        memberships.push_back({{"node_id", id}, {"incarnation_id", node.value("incarnation_id", Json(nullptr))},
            {"state", finding.state}, {"source_record_ids", record_refs(finding)}, {"undated_evidence", finding.undated}});
        if (finding.state == "joined") eligible.push_back(id);
        else if (finding.state != "removed") membership_incomplete = true;
    }
    auto fabrics = Json::array();
    bool proved_connected = false;
    bool negative_complete = !input.buses.empty();
    bool partitioned = false;
    using Pair = std::pair<std::string, std::string>;
    for (const auto& bus : input.buses) {
        deadline(limit); subject_ids.insert(bus);
        std::map<Pair, std::vector<const Json*>> pairs;
        for (const auto& record : value.at("observations")) if (record.at("bus_id") == bus) {
            auto a = text(record, "from_node_id"), b = text(record, "to_node_id");
            if (!input.directed && b < a) std::swap(a, b);
            pairs[{a, b}].push_back(&record);
        }
        std::map<Pair, Current> pair_findings;
        for (const auto& [pair, records] : pairs) pair_findings.emplace(pair, current(records, input, "observed_unix_ms", true));
        const auto count = eligible.size();
        std::vector<std::vector<std::size_t>> edges(count);
        std::size_t unknown_pairs = 0, stale_pairs = 0, conflict_pairs = 0, disconnected_pairs = 0;
        auto connections = Json::array();
        for (const auto& [pair, finding] : pair_findings) {
            connections.push_back({{"from_node_id", pair.first}, {"to_node_id", pair.second},
                {"state", finding.state}, {"coverage", finding.partial ? "partial" : "complete"},
                {"source_record_ids", record_refs(finding)}, {"stale_evidence", finding.stale}, {"undated_evidence", finding.undated}});
        }
        for (std::size_t a = 0; a < count; ++a) {
            deadline(limit);
            for (std::size_t b = 0; b < count; ++b) {
                if (a == b || (!input.directed && a > b)) continue;
                auto key = Pair{eligible[a], eligible[b]};
                const auto found = pair_findings.find(key);
                if (found == pair_findings.end()) { ++unknown_pairs; continue; }
                const auto& finding = found->second;
                if (finding.state == "connected") {
                    edges[a].push_back(b); if (!input.directed) edges[b].push_back(a);
                } else if (finding.state == "disconnected" && !finding.partial) ++disconnected_pairs;
                else { ++unknown_pairs; if (finding.state == "stale") ++stale_pairs; if (finding.state == "conflicting") ++conflict_pairs; }
            }
        }
        // Reachability stays inside one exact bus. No adjacency from another
        // bus can manufacture an undocumented bridge.
        std::vector<std::vector<bool>> reaches(count, std::vector<bool>(count));
        for (std::size_t start = 0; start < count; ++start) {
            deadline(limit);
            reaches[start][start] = true;
            std::vector<std::size_t> queue{start};
            for (std::size_t cursor = 0; cursor < queue.size(); ++cursor) {
                for (const auto to : edges[queue[cursor]]) if (!reaches[start][to]) {
                    reaches[start][to] = true;
                    if (input.transitive) queue.push_back(to);
                }
            }
        }
        bool all_connected = count >= 2;
        for (const auto& reach : reaches) for (const auto reachable : reach) all_connected = all_connected && reachable;
        auto groups = Json::array();
        std::vector<bool> assigned(count);
        for (std::size_t a = 0; a < count; ++a) if (!assigned[a]) {
            auto group = Json::array({eligible[a]}); assigned[a] = true;
            // In a non-transitive profile these are pairwise complete groups,
            // avoiding accidental transitive interpretation in diagnostics.
            for (std::size_t b = a + 1; b < count; ++b) if (!assigned[b]) {
                bool joins = true;
                for (const auto& member : group) {
                    const auto index = static_cast<std::size_t>(std::lower_bound(eligible.begin(), eligible.end(), member.get<std::string>()) - eligible.begin());
                    joins = joins && reaches[index][b] && reaches[b][index];
                }
                if (joins) { group.push_back(eligible[b]); assigned[b] = true; }
            }
            groups.push_back(group);
        }
        std::string state;
        if (!input.timed) state = "evaluation_time_unknown";
        else if (count < 2) state = "insufficient_membership";
        else if (all_connected) state = "connected";
        else if (unknown_pairs != 0) state = "insufficient";
        else if (groups.size() > 1 && groups.size() < count) state = "partitioned";
        else state = "disconnected";
        proved_connected = proved_connected || state == "connected";
        negative_complete = negative_complete && (state == "partitioned" || state == "disconnected");
        partitioned = partitioned || state == "partitioned";
        fabrics.push_back({{"bus_id", bus}, {"state", state}, {"eligible_nodes", eligible},
            {"reachable_groups", groups}, {"connection_findings", connections}, {"unknown_pair_count", unknown_pairs},
            {"stale_pair_count", stale_pairs}, {"conflicting_pair_count", conflict_pairs}, {"disconnected_pair_count", disconnected_pairs}});
    }
    std::string state;
    if (!input.timed) state = "evaluation_time_unknown";
    else if (eligible.size() < 2) state = membership_incomplete ? "insufficient_membership" : "not_cluster";
    else if (proved_connected) state = "connected";
    else if (input.buses.empty()) state = "unobserved";
    else if (negative_complete) state = partitioned ? "partitioned" : "disconnected";
    else state = "insufficient";
    return {{"protocol", "symphony.snv.sciv.result.v1"}, {"owner", "sciv"}, {"owner_version", version},
        {"source_digest", engine::tagged_sha256(value.dump())}, {"subject_ids", subject_ids},
        {"cluster_id", input.cluster}, {"system_id", input.system}, {"at_unix_ms", input.timed ? Json(input.at) : Json(nullptr)},
        {"profile", value.at("profile")}, {"cluster_state", state}, {"connected_cluster", state == "connected"},
        {"membership_findings", memberships}, {"fabric_findings", fabrics},
        {"history", {{"memberships", value.at("memberships")}, {"observations", value.at("observations")}}},
        {"cross_fabric_bridge_inferred", false}, {"record_count", input.nodes.size() + value.at("memberships").size() + value.at("observations").size()}};
}

Json transition(const Json& value, std::int64_t limit) {
    fields(value, {"protocol", "evidence", "expected_source_digest", "change"});
    if (text(value, "protocol") != "symphony.snv.sciv.transition.v1") fail("unsupported_protocol", "SCIV transition protocol unsupported");
    digest(value, "expected_source_digest");
    const auto original = validate(value.at("evidence"), limit);
    if (original.at("source_digest") != value.at("expected_source_digest")) fail("conflict", "SCIV transition evidence digest differs from expected state");
    fields(value.at("change"), {"kind", "record"});
    const auto kind = choice(value.at("change"), "kind", {"membership", "connection"});
    const auto& record = value.at("change").at("record");
    if (!record.is_object() || !record.contains("record_id")) fail("invalid_payload", "SCIV transition requires typed record");
    const auto id = text(record, "record_id");
    auto candidate = value.at("evidence");
    bool present = false;
    for (const auto field : {"memberships", "observations"}) for (const auto& previous : candidate.at(field)) if (previous.at("record_id") == id) {
        if (previous != record || field != std::string(kind == "membership" ? "memberships" : "observations"))
            fail("conflict", "SCIV transition cannot overwrite retained evidence identifier");
        present = true;
    }
    if (!present) candidate.at(kind == "membership" ? "memberships" : "observations").push_back(record);
    const auto result = validate(candidate, limit);
    return {{"protocol", "symphony.snv.sciv.transition-result.v1"}, {"owner", "sciv"}, {"owner_version", version},
        {"source_digest", engine::tagged_sha256(value.dump())}, {"subject_ids", result.at("subject_ids")},
        {"base_source_digest", original.at("source_digest")}, {"candidate_source_digest", result.at("source_digest")},
        {"candidate_evidence", candidate}, {"candidate_result", result}, {"status", present ? "already_present" : "candidate"}};
}
}

std::vector<symphony::snv::Operation> operations() {
    return {
        {"sciv_validate", "symphony.snv.sciv.evidence.v1", "symphony.snv.sciv.result.v1", {"validate", "inspect"}},
        {"sciv_transition", "symphony.snv.sciv.transition.v1", "symphony.snv.sciv.transition-result.v1", {"propose"}, "proposal_only", "idempotent", true}
    };
}

Json handle(std::string_view operation, const Json& payload, std::int64_t deadline_unix_ms) {
    deadline(deadline_unix_ms);
    if (operation == "sciv_validate") return validate(payload, deadline_unix_ms);
    if (operation == "sciv_transition") return transition(payload, deadline_unix_ms);
    fail("unsupported_operation", "SCIV operation unsupported");
}

}
