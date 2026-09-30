#include <symphony/snv/sciv.hpp>
#include <symphony/knowledge/engine/error.hpp>

#include <chrono>
#include <functional>
#include <iostream>
#include <limits>
#include <stdexcept>
#include <string>

using symphony::snv::Json;
namespace sciv = symphony::snv::sciv;
namespace engine = symphony::knowledge::engine;

namespace {
std::size_t cases = 0;
std::int64_t future() {
    return std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count() + 60000;
}
void check(bool condition, const std::string& message) {
    if (!condition) throw std::runtime_error(message);
}
void test(const char* name, const std::function<void()>& run) {
    run(); ++cases; std::cout << "pass " << name << '\n';
}
void rejected(const std::string& code, const std::function<void()>& run) {
    try { run(); } catch (const engine::Error& error) {
        check(error.code() == code, "wrong error: " + error.code() + " expected " + code); return;
    }
    throw std::runtime_error("expected error " + code);
}
Json source(std::string id = "observer") {
    return {{"source_id", id}, {"source_revision", "capture-1"}, {"method", "supplied-fixture"}, {"method_version", "1"}};
}
Json node(const std::string& id) {
    return {{"node_id", id}, {"system_id", "research"}, {"incarnation_id", "episode-" + id}};
}
Json membership(std::string id, std::string state = "joined", std::int64_t time = 10000, std::string record = "") {
    return {{"record_id", record.empty() ? "member-" + id : record}, {"node_id", id},
        {"incarnation_id", "episode-" + id}, {"state", state}, {"effective_unix_ms", time}, {"source", source("membership-observer")}};
}
Json connection(std::string a, std::string b, std::string state = "connected", std::int64_t time = 10000,
                std::string bus = "fabric-1", std::string record = "") {
    return {{"record_id", record.empty() ? bus + "-" + a + "-" + b : record}, {"bus_id", bus},
        {"from_node_id", a}, {"to_node_id", b}, {"state", state}, {"coverage", "complete"},
        {"observed_unix_ms", time}, {"source", source("link-observer")}};
}
Json fixture(std::size_t count = 2) {
    auto nodes = Json::array(), members = Json::array();
    for (std::size_t i = 0; i < count; ++i) {
        const auto id = std::string(1, static_cast<char>('a' + i));
        nodes.push_back(node(id)); members.push_back(membership(id));
    }
    return {{"protocol", "symphony.snv.sciv.evidence.v1"}, {"record_limit", 2048}, {"at_unix_ms", 10000},
        {"profile", {{"direction", "undirected"}, {"transitive", true}, {"max_age_ms", 1000}}},
        {"cluster", {{"cluster_id", "cluster-alpha"}, {"system_id", "research"}}},
        {"nodes", nodes}, {"memberships", members}, {"observations", Json::array({connection("a", "b")})}};
}
Json run(const Json& input) { return sciv::handle("sciv_validate", input, future()); }
void state(const Json& input, const char* expected) {
    const auto result = run(input);
    check(result.at("cluster_state") == expected, "expected cluster " + std::string(expected) + " got " + result.at("cluster_state").get<std::string>());
    check(result.at("connected_cluster") == (std::string(expected) == "connected"), "inconsistent connectivity boolean");
}
}

int main() {
    try {
        test("two identified joined Nodes and observed fabric", [] {
            const auto value = fixture(); const auto result = run(value);
            check(result.at("owner") == "sciv" && result.at("owner_version") == "0.1.0-dev", "owner provenance");
            check(result.at("cluster_state") == "connected", "two-Node cluster");
            check(result.at("history").at("observations") == value.at("observations"), "original observations retained");
            check(result.at("subject_ids").size() == 7, "subject IDs include distinct Node/episode/system/cluster/bus");
            check(result == run(value), "deterministic canonical output");
        });
        test("wire and direct SDK numeric bounds agree", [] {
            auto value = fixture();
            value["at_unix_ms"] = 9007199254740991LL;
            value["profile"]["max_age_ms"] = 9007199254740991LL;
            state(value, "connected");
            for (const auto field : {"at_unix_ms", "max_age_ms", "effective_unix_ms", "observed_unix_ms", "valid_until_unix_ms"}) {
                value = fixture();
                if (std::string(field) == "max_age_ms") value["profile"][field] = 9007199254740992ULL;
                else if (std::string(field) == "effective_unix_ms") value["memberships"][0][field] = 9007199254740992ULL;
                else if (std::string(field) == "observed_unix_ms" || std::string(field) == "valid_until_unix_ms") value["observations"][0][field] = 9007199254740992ULL;
                else value[field] = 9007199254740992ULL;
                rejected("invalid_payload", [&] { (void)run(value); });
            }
        });
        test("desired membership alone", [] { auto v = fixture(); v["observations"] = Json::array(); state(v, "unobserved"); });
        test("intent does not prove membership", [] { auto v = fixture(); for (auto& m : v["memberships"]) m["state"] = "intended"; state(v, "insufficient_membership"); });
        test("a single Node is no cluster", [] { auto v = fixture(1); v["observations"] = Json::array(); state(v, "not_cluster"); });
        test("multiple buses support one cluster", [] {
            auto v = fixture(); v["observations"].push_back(connection("a", "b", "connected", 10000, "fabric-2"));
            auto r = run(v); check(r["fabric_findings"].size() == 2 && r["cluster_id"] == "cluster-alpha" && r["connected_cluster"] == true, "multi-fabric identity");
        });
        test("one disconnected fabric does not erase a connected alternative", [] {
            auto v = fixture(); v["observations"].push_back(connection("a", "b", "disconnected", 10000, "fabric-2")); state(v, "connected");
        });
        test("no inferred cross-fabric bridge", [] {
            auto v = fixture(3); v["observations"].push_back(connection("b", "c", "connected", 10000, "fabric-2"));
            auto r = run(v); check(r["cluster_state"] == "insufficient" && r["cross_fabric_bridge_inferred"] == false, "unproved bridge");
        });
        test("observed disconnect retains historical records", [] {
            auto v = fixture(); v["at_unix_ms"] = 10050; v["observations"].push_back(connection("a", "b", "disconnected", 10050, "fabric-1", "disconnect"));
            auto r = run(v); check(r["cluster_state"] == "disconnected" && r["history"]["observations"].size() == 2, "disconnect history");
            check(r["membership_findings"][0]["incarnation_id"] == "episode-a", "disconnect changed incarnation");
        });
        test("reconnect preserves supplied incarnation", [] {
            auto v = fixture(); v["at_unix_ms"] = 10100;
            v["observations"].push_back(connection("a", "b", "disconnected", 10050, "fabric-1", "disconnect"));
            v["observations"].push_back(connection("a", "b", "connected", 10100, "fabric-1", "reconnect")); state(v, "connected");
        });
        test("membership removal differs from disconnect", [] {
            auto v = fixture(); v["at_unix_ms"] = 10050; v["memberships"].push_back(membership("b", "removed", 10050, "remove-b"));
            auto r = run(v); check(r["cluster_state"] == "not_cluster" && r["membership_findings"][1]["state"] == "removed", "explicit removal");
        });
        test("as-of view precedes explicit removal", [] {
            auto v = fixture(); v["at_unix_ms"] = 10025; v["memberships"].push_back(membership("b", "removed", 10050, "remove-b")); state(v, "connected");
        });
        test("missing evaluation time is truthful", [] { auto v = fixture(); v.erase("at_unix_ms"); state(v, "evaluation_time_unknown"); });
        test("unknown source time is retained without fabricated current claim", [] {
            auto v = fixture(); v["observations"][0].erase("observed_unix_ms");
            auto r = run(v); check(r["cluster_state"] == "insufficient" && r["fabric_findings"][0]["connection_findings"][0]["state"] == "undated", "undated evidence");
        });
        test("stale observation is insufficient", [] {
            auto v = fixture(); v["at_unix_ms"] = 11001; auto r = run(v);
            check(r["cluster_state"] == "insufficient" && r["fabric_findings"][0]["stale_pair_count"] == 1, "stale coverage");
        });
        test("freshness exact boundary", [] { auto v = fixture(); v["at_unix_ms"] = 11000; state(v, "connected"); });
        test("validity end is exclusive", [] { auto v = fixture(); v["observations"][0]["valid_until_unix_ms"] = 10050; v["at_unix_ms"] = 10050; state(v, "insufficient"); });
        test("future observation does not prove now", [] { auto v = fixture(); v["observations"][0]["observed_unix_ms"] = 10050; state(v, "insufficient"); });
        test("equal-time competing source conflict", [] {
            auto v = fixture(); auto other = connection("a", "b", "disconnected", 10000, "fabric-1", "contradiction"); other["source"]["source_id"] = "other-observer";
            v["observations"].push_back(other); auto r = run(v); check(r["fabric_findings"][0]["conflicting_pair_count"] == 1 && r["cluster_state"] == "insufficient", "source conflict");
        });
        test("newer different source does not create precedence", [] {
            auto v = fixture(); v["at_unix_ms"] = 10050; auto other = connection("a", "b", "disconnected", 10050, "fabric-1", "contradiction");
            other["source"]["source_id"] = "other-observer"; v["observations"].push_back(other); state(v, "insufficient");
        });
        test("source-scoped temporal correction", [] {
            auto v = fixture(); v["at_unix_ms"] = 10050; auto newer = connection("a", "b", "disconnected", 10050, "fabric-1", "correction");
            newer["source"]["source_revision"] = "capture-2"; v["observations"].push_back(newer); state(v, "disconnected");
        });
        test("membership sources conflict", [] {
            auto v = fixture(); auto conflict = membership("b", "removed", 10000, "other-member"); conflict["source"]["source_id"] = "other-membership";
            v["memberships"].push_back(conflict); auto r = run(v); check(r["membership_findings"][1]["state"] == "conflicting", "membership conflict erased"); state(v, "insufficient_membership");
        });
        test("partial disconnect does not prove global negative", [] { auto v = fixture(); v["observations"][0]["state"] = "disconnected"; v["observations"][0]["coverage"] = "partial"; state(v, "insufficient"); });
        test("partial collection can prove an observed positive edge", [] { auto v = fixture(); v["observations"][0]["coverage"] = "partial"; state(v, "connected"); });
        test("transitive undirected chain", [] { auto v = fixture(3); v["observations"].push_back(connection("b", "c")); state(v, "connected"); });
        test("non-transitive chain remains insufficient", [] { auto v = fixture(3); v["observations"].push_back(connection("b", "c")); v["profile"]["transitive"] = false; state(v, "insufficient"); });
        test("directed chain is not mutually connected", [] { auto v = fixture(3); v["observations"].push_back(connection("b", "c")); v["profile"]["direction"] = "directed"; state(v, "insufficient"); });
        test("directed cycle has mutual transitive reachability", [] {
            auto v = fixture(3); v["observations"].push_back(connection("b", "c")); v["observations"].push_back(connection("c", "a")); v["profile"]["direction"] = "directed"; state(v, "connected");
        });
        test("directed evidence never manufactures reverse edge", [] { auto v = fixture(); v["profile"]["direction"] = "directed"; state(v, "insufficient"); });
        test("undirected reverse edge round trip", [] { auto v = fixture(); std::swap(v["observations"][0]["from_node_id"], v["observations"][0]["to_node_id"]); state(v, "connected"); });
        test("complete partition differs from missing observations", [] {
            auto v = fixture(4); v["observations"] = Json::array();
            for (char a = 'a'; a <= 'd'; ++a) for (char b = static_cast<char>(a + 1); b <= 'd'; ++b)
                v["observations"].push_back(connection(std::string(1,a), std::string(1,b), (a == 'a' && b == 'b') || (a == 'c' && b == 'd') ? "connected" : "disconnected"));
            auto r = run(v); check(r["cluster_state"] == "partitioned" && r["fabric_findings"][0]["reachable_groups"].size() == 2, "partition proof");
        });
        test("unknown observation is never disconnected", [] { auto v = fixture(); v["observations"][0]["state"] = "unknown"; state(v, "insufficient"); });
        test("unknown incarnation remains absent", [] {
            auto v = fixture(); for (auto& n : v["nodes"]) n.erase("incarnation_id"); for (auto& m : v["memberships"]) m.erase("incarnation_id");
            check(run(v)["membership_findings"][0]["incarnation_id"].is_null(), "fabricated incarnation");
        });
        test("capacity profiles and exact boundary", [] {
            for (const auto capacity : {512, 1024, 2048}) { auto v = fixture(); v["record_limit"] = capacity; state(v, "connected"); }
            auto v = fixture(); v["record_limit"] = 512; v["memberships"] = Json::array(); v["observations"] = Json::array(); v["nodes"] = Json::array();
            for (int i = 0; i < 512; ++i) v["nodes"].push_back(node("node-" + std::to_string(i)));
            check(run(v)["record_count"] == 512, "exact capacity rejected"); v["nodes"].push_back(node("over-limit")); rejected("capacity_exceeded", [&] { (void)run(v); });
        });
        test("unknown fields and protocols rejected", [] {
            auto v = fixture(); v["extra"] = true; rejected("invalid_payload", [&] { (void)run(v); }); v.erase("extra");
            v["protocol"] = "symphony.snv.sciv.evidence.v2"; rejected("unsupported_protocol", [&] { (void)run(v); });
        });
        test("adversarial typed fields", [] {
            auto v = fixture(); v["profile"]["transitive"] = "true"; rejected("invalid_payload", [&] { (void)run(v); });
            v = fixture(); v["at_unix_ms"] = 10000.5; rejected("invalid_payload", [&] { (void)run(v); });
            v = fixture(); v["at_unix_ms"] = std::numeric_limits<std::uint64_t>::max(); rejected("invalid_payload", [&] { (void)run(v); });
            v = fixture(); v["profile"]["direction"] = "bidirectional-guessed"; rejected("invalid_payload", [&] { (void)run(v); });
            v = fixture(); v["record_limit"] = 4096; rejected("invalid_payload", [&] { (void)run(v); });
        });
        test("foreign and duplicate references", [] {
            auto v = fixture(); v["nodes"][0]["system_id"] = "other-system"; rejected("reference_mismatch", [&] { (void)run(v); });
            v = fixture(); v["observations"][0]["to_node_id"] = "foreign"; rejected("reference_mismatch", [&] { (void)run(v); });
            v = fixture(); v["nodes"].push_back(v["nodes"][0]); rejected("reference_conflict", [&] { (void)run(v); });
            v = fixture(); v["memberships"][0]["incarnation_id"] = "other-episode"; rejected("reference_mismatch", [&] { (void)run(v); });
            v = fixture(); v["nodes"][1]["incarnation_id"] = "episode-a"; rejected("reference_conflict", [&] { (void)run(v); });
            v = fixture(); v["observations"].push_back(v["observations"][0]); rejected("reference_conflict", [&] { (void)run(v); });
        });
        test("deadline and unsupported operation", [] {
            rejected("deadline_exceeded", [] { (void)sciv::handle("sciv_validate", fixture(), 1); });
            rejected("unsupported_operation", [] { (void)sciv::handle("configure_bus", fixture(), future()); });
        });
        test("digest-bound membership transition is candidate only", [] {
            const auto v = fixture(); auto input = Json{{"protocol", "symphony.snv.sciv.transition.v1"}, {"evidence", v},
                {"expected_source_digest", run(v)["source_digest"]}, {"change", {{"kind", "membership"}, {"record", membership("b", "removed", 10000, "retirement")}}}};
            const auto r = sciv::handle("sciv_transition", input, future()); check(r["status"] == "candidate", "transition status");
            check(v["memberships"].size() == 2 && r["candidate_evidence"]["memberships"].size() == 3, "transition overwrote original");
            check(r["candidate_result"]["membership_findings"][1]["state"] == "conflicting", "equal-time conflict discarded");
            input["expected_source_digest"] = "sha256:" + std::string(64,'0'); rejected("conflict", [&] { (void)sciv::handle("sciv_transition", input, future()); });
        });
        test("typed transition has replay and collision protection", [] {
            const auto v = fixture(); auto input = Json{{"protocol", "symphony.snv.sciv.transition.v1"}, {"evidence", v},
                {"expected_source_digest", run(v)["source_digest"]}, {"change", {{"kind", "connection"}, {"record", v["observations"][0]}}}};
            check(sciv::handle("sciv_transition", input, future())["status"] == "already_present", "exact retry not idempotent");
            input["change"]["record"]["state"] = "disconnected"; rejected("conflict", [&] { (void)sciv::handle("sciv_transition", input, future()); });
            input["change"]["record"]["record_id"] = "new-invalid"; input["change"]["record"]["bus_id"] = "";
            rejected("invalid_payload", [&] { (void)sciv::handle("sciv_transition", input, future()); });
        });
        test("canonical input order affects evidence digest, never connectivity", [] {
            auto v = fixture(); auto reordered = v; std::swap(reordered["nodes"][0], reordered["nodes"][1]);
            check(run(v)["source_digest"] != run(reordered)["source_digest"], "input sequence evidence lost"); state(reordered, "connected");
        });
        std::cout << cases << " SCIV scenarios passed\n";
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "SCIV failure: " << error.what() << '\n'; return 1;
    }
}
