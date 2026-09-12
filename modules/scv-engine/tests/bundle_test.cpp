#include "bundle.hpp"
#include "composition.hpp"
#include "knowledge.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"

#include <algorithm>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <vector>

namespace engine = symphony::knowledge::engine;
namespace scv = symphony::knowledge::scv;
using engine::Json;
namespace {
void require(bool condition, const std::string& message) { if (!condition) throw std::runtime_error(message); }
template<class F> void rejects(F&& action) {
    try { action(); } catch (const engine::Error&) { return; }
    throw std::runtime_error("expected bounded bundle rejection");
}
engine::Request request(const std::string& operation = "bundle_inspect", Json payload = Json::object()) {
    return {"bundle-test", "bundle-test", operation, "symphony-scv", engine::unix_time_ms() + 120000, std::move(payload)};
}
std::string hash(const Json& value) { return engine::tagged_sha256(value.dump()); }
Json pack(const Json& value) { return scv::encode_bundle(value, request()); }
scv::DecodedBundle unpack(const Json& value) { return scv::decode_bundle(value, request()); }
Json input(const Json& value, const std::string& operation = "composition_explore") {
    return {{"operation", operation}, {"owner", {{"domain", "scv"}, {"version", scv::version}}}, {"bundle", pack(value)}};
}
Json call(const std::string& operation, const Json& payload) { return scv::handle_bundle(request(operation, payload), "scv"); }
Json native(const std::string& operation, const Json& payload) { return scv::handle_composition(request(operation, payload), "scv"); }
void sort_nodes(Json& bundle) {
    std::sort(bundle["objects"].begin(), bundle["objects"].end(), [](const Json& a, const Json& b) { return a.at("digest") < b.at("digest"); });
    bundle = scv::sealed(bundle);
}
Json& root_node(Json& bundle) {
    for (auto& node : bundle["objects"]) if (node.at("digest") == bundle.at("root_digest")) return node;
    throw std::runtime_error("missing test root");
}
Json policy() { return {{"policy_id", "bundle-fixture"}, {"max_age_seconds", nullptr}, {"partial_capture", "exclude"}, {"allowed_statement_kinds", Json::array({"user_assertion"})}}; }
Json composition_input() {
    const Json value = {{"type", "integer"}, {"value", 10}, {"unit", "units"}};
    const Json claim = {{"claim_id", "limit"}, {"subject", "fixture"}, {"predicate", "capacity"}, {"value", value},
        {"scope", Json::object()}, {"statement_kind", "user_assertion"}, {"evidence", Json::array()}, {"dependencies", Json::array()}};
    const auto knowledge = scv::handle_knowledge(request("knowledge_interpret", {{"captures", Json::array()}, {"claims", Json::array({claim})},
        {"interpreter_version", "bundle-fixture"}, {"selection_policy", policy()}}), "scv");
    const Json resolution = {{"kind", "adapter"}, {"reference", "fixture.adapter"}, {"description", "Explicit synthetic adapter; no execution"}};
    const Json requirement = {{"requirement_id", "capacity"}, {"importance", "required"}, {"operator", "gte"},
        {"right", {{"kind", "literal"}, {"value", value}}}, {"resolution", resolution}};
    const Json recipe = {{"recipe_id", "first"}, {"provider_id", "cf"}, {"bindings", Json::array({Json{{"requirement_id", "capacity"},
        {"claim", {{"claim_id", "limit"}, {"subject", "fixture"}, {"scope", Json::object()}}}}})}, {"prerequisites", Json::array()},
        {"requires_interfaces", Json::array()}, {"supplies_interfaces", Json::array()}, {"guarantee_changes", Json::array()},
        {"implementation", {{"status", "unknown"}, {"reference", nullptr}}}, {"resolution", resolution}};
    return {{"interpretations", Json::array()}, {"additional_knowledge", Json::array({knowledge})}, {"provider_packs", Json::array()},
        {"query_time", "2026-09-10T00:00:00Z"}, {"requirements", Json::array({requirement})},
        {"slots", Json::array({Json{{"slot_id", "compute"}, {"allowed_provider_ids", Json::array({"cf"})}, {"recipes", Json::array({recipe})}}})},
        {"allowed_guarantee_changes", Json::array()}, {"counterfactuals", Json::array()}, {"bounds", {{"max_candidates", 32}}}};
}
Json submission(const Json& obligation) {
    return {{"submission_id", "opaque-attempt"}, {"obligation_id", obligation.at("obligation_id")},
        {"provenance", {{"kind", "adapter"}, {"producer", "caller"}, {"reference", "not-executed"}, {"content_digest", nullptr},
            {"recorded_at", "2026-09-10T00:00:01Z"}, {"description", "Opaque reference does not prove success"}}}};
}
void test_bundle_codec_producer_exact_sharing_and_metrics() {
    const Json array = Json::array({nullptr, true, 0});
    const Json value = {{"a", array}, {"b", {{"x", "Y"}}}, {"c", array}};
    const auto bundle = pack(value);
    const auto decoded = unpack(bundle);
    require(decoded.value == value && bundle == pack(value), "deterministic exact reconstruction");
    require(decoded.metrics.at("object_count") == 3 && decoded.metrics.at("reference_count") == 3 &&
        decoded.metrics.at("traversal_steps") == 6 && decoded.metrics.at("expanded_values") == 15 &&
        decoded.metrics.at("expanded_depth") == 2 && decoded.metrics.at("expanded_bytes") == value.dump().size(), "exact expanded accounting");
    require(decoded.metrics.at("materialized_bytes") == value.dump().size() + array.dump().size() + Json({{"x", "Y"}}).dump().size(), "unique materialization accounting");
    auto sealed = scv::sealed(Json{{"value", "retained"}});
    require(pack(sealed).at("root_digest") == hash(sealed) && sealed.at("digest") != hash(sealed), "full-value hash differs from native self-seal");
}
void test_bundle_codec_preserves_unicode_integer_and_reference_looking_values() {
    const auto value = engine::parse_bounded_json(R"({"maximum":9007199254740991,"minimum":-9007199254740991,"zero":-0,"ref":{"scalar":[],"ref":"sha256:not-an-instruction"},"empty":{},"unicode":"\u2028\u2029雪","escaped":"\\u2028","bool":false})", 4096);
    const auto decoded = unpack(pack(value));
    require(decoded.value == value && decoded.value.at("zero").dump() == "0", "native integral negative zero canonicalizes to zero");
    require(decoded.value.at("unicode").get<std::string>() == "\xE2\x80\xA8\xE2\x80\xA9雪", "Unicode source bytes retained");
    require(decoded.value.at("escaped") == "\\u2028", "literal escape spelling is data");
    require(unpack(pack(Json::object())).value == Json::object(), "empty object retained");
    rejects([] { static_cast<void>(pack(Json::array())); });
    rejects([] { static_cast<void>(pack(Json{{"float", 0.0}})); });
    rejects([] { static_cast<void>(pack(Json{{"too_large", 9007199254740992ULL}})); });
    rejects([] { static_cast<void>(pack(Json{{"too_small", -9007199254740992LL}})); });
    rejects([] { static_cast<void>(pack(Json{{"invalid", std::string(1, static_cast<char>(0xff))}})); });
}
void test_bundle_codec_rejects_missing_cycle_unused_and_duplicate_nodes() {
    const auto original = pack(Json{{"child", Json::array({1})}});
    auto b = original;
    root_node(b)["members"]["child"]["ref"] = "sha256:" + std::string(64, '0'); b = scv::sealed(b);
    rejects([&] { static_cast<void>(unpack(b)); });
    b = original; root_node(b)["members"]["child"]["ref"] = b.at("root_digest"); b = scv::sealed(b);
    rejects([&] { static_cast<void>(unpack(b)); });
    b = original; const auto extra = pack(Json{{"unused", 123}}); b["objects"].push_back(extra.at("objects").at(0)); sort_nodes(b);
    rejects([&] { static_cast<void>(unpack(b)); });
    b = original; b["objects"].push_back(b.at("objects").at(0)); sort_nodes(b);
    rejects([&] { static_cast<void>(unpack(b)); });
    b = original; std::reverse(b["objects"].begin(), b["objects"].end()); b = scv::sealed(b);
    rejects([&] { static_cast<void>(unpack(b)); });
}
void test_bundle_codec_rejects_resealed_hash_and_closed_shape_changes() {
    const auto original = pack(Json{{"scalar", "selected"}, {"array", Json::array()}});
    auto b = original; root_node(b)["members"]["scalar"]["scalar"] = "substituted"; b = scv::sealed(b);
    rejects([&] { static_cast<void>(unpack(b)); });
    b = original; b["unknown"] = nullptr; b = scv::sealed(b); rejects([&] { static_cast<void>(unpack(b)); });
    b = original; root_node(b)["members"]["scalar"]["ref"] = b.at("root_digest"); b = scv::sealed(b); rejects([&] { static_cast<void>(unpack(b)); });
    b = original; root_node(b)["members"]["scalar"]["scalar"] = Json::object(); b = scv::sealed(b); rejects([&] { static_cast<void>(unpack(b)); });
    b = original; root_node(b)["members"] = Json::array(); b = scv::sealed(b); rejects([&] { static_cast<void>(unpack(b)); });
    b = original; b["digest"] = "sha256:" + std::string(64, '0'); rejects([&] { static_cast<void>(unpack(b)); });
    rejects([] { static_cast<void>(engine::parse_bounded_json("{\"duplicate\":1,\"duplicate\":2}", 100)); });
}
void test_bundle_codec_rejects_exponential_fanout_before_materialization() {
    Json nodes = Json::array();
    std::string prior;
    for (int i = 0; i < 20; ++i) {
        const auto id = engine::tagged_sha256("synthetic-level-" + std::to_string(i));
        Json members = i == 0 ? Json{{"value", {{"scalar", 0}}}} : Json{{"left", {{"ref", prior}}}, {"right", {{"ref", prior}}}};
        nodes.push_back({{"digest", id}, {"kind", "object"}, {"members", members}}); prior = id;
    }
    auto b = scv::sealed(Json{{"protocol", "symphony.scv.evidence-bundle.v1"}, {"root_digest", prior}, {"objects", nodes}}); sort_nodes(b);
    try { static_cast<void>(unpack(b)); }
    catch (const engine::Error& error) { require(std::string(error.what()).find("budget") != std::string::npos, "fanout rejected by metrics before fake hash comparison"); return; }
    throw std::runtime_error("fanout was accepted");
}
void test_bundle_codec_logical_depth_values_and_object_bounds() {
    Json deep = Json::object(); for (int i = 0; i < 64; ++i) deep = Json{{"child", deep}};
    require(unpack(pack(deep)).metrics.at("expanded_depth") == 64, "exact logical depth boundary accepted");
    deep = Json{{"child", deep}}; rejects([&] { static_cast<void>(pack(deep)); });
    Json wide = Json::array(); for (int i = 0; i < 32766; ++i) wide.push_back(i);
    rejects([&] { static_cast<void>(pack(Json{{"values", wide}})); });
    Json objects = Json::array(); for (int i = 0; i < 2048; ++i) objects.push_back(Json{{"id", i}});
    rejects([&] { static_cast<void>(pack(Json{{"objects", objects}})); });
    rejects([] { static_cast<void>(pack(Json{{"oversized_string", std::string(65537, 'x')}})); });
}
void test_bundle_codec_byte_and_cumulative_materialization_bounds() {
    Json leaf = Json::object(); for (int i = 0; i < 34; ++i) leaf[std::to_string(i)] = std::string(65536, 'x');
    Json chain = leaf; for (int i = 0; i < 32; ++i) chain = Json{{"child", chain}};
    rejects([&] { static_cast<void>(pack(chain)); });
    Json too_large = Json::object(); for (int i = 0; i < 65; ++i) too_large[std::to_string(i)] = std::string(65536, 'x');
    rejects([&] { static_cast<void>(pack(too_large)); });
}
void test_bundle_inspection_is_transport_only_and_owner_pinned() {
    const auto payload = input(Json{{"not_composition", true}});
    const auto result = call("bundle_inspect", payload);
    require(result.at("validation") == "transport_only" && result.at("input_digest") == hash(payload), "inspection makes no semantic claim");
    rejects([&] { static_cast<void>(call("composition_bundle_evaluate", payload)); });
    auto bad = payload; bad["owner"]["domain"] = "schv-gcp"; rejects([&] { static_cast<void>(call("bundle_inspect", bad)); });
    bad = payload; bad["owner"]["version"] = "0.8.0-dev"; if (std::string(scv::version) == "0.8.0-dev") bad["owner"]["version"] = "0.1.0-dev";
    rejects([&] { static_cast<void>(call("bundle_inspect", bad)); });
    bad = payload; bad["operation"] = "composition_bundle_evaluate"; rejects([&] { static_cast<void>(call("bundle_inspect", bad)); });
    bad = payload; bad["operation"] = "source_apply"; rejects([&] { static_cast<void>(call("bundle_inspect", bad)); });
}
Json verify_native(const std::string& operation, const Json& value) {
    const auto payload = input(value, operation), result = call("composition_bundle_evaluate", payload);
    const auto logical = unpack(result.at("result_bundle"));
    require(logical.value == native(operation, value), "bundled logical result equals unchanged native owner result");
    require(result.at("native_result_digest") == logical.value.at("digest") && result.at("result_metrics") == logical.metrics &&
        result.at("input_digest") == hash(payload) && result.at("input_root_digest") == hash(value), "exact input/result/digest/metrics binding");
    return result;
}
void test_composition_bundle_evaluation_producer_preserves_all_four_operations() {
    const auto original = composition_input();
    static_cast<void>(verify_native("composition_explore", original));
    const auto before = native("composition_explore", original);
    auto changed = original; changed["query_time"] = "2026-09-10T00:00:01Z";
    const auto after = native("composition_explore", changed);
    static_cast<void>(verify_native("composition_reassess", {{"before", before}, {"after", after}}));
    const auto obligations = native("composition_obligations", {{"composition", before}});
    static_cast<void>(verify_native("composition_obligations", {{"composition", before}}));
    const auto followup = verify_native("composition_followup", {{"before", before}, {"after", after}, {"submissions", Json::array({submission(obligations.at("obligations").at(0))})}});
    require(unpack(followup.at("result_bundle")).value.at("change_axes").at("query_time") == true, "time differences preserved explicitly");
}
void test_composition_bundle_consumer_rejects_forgery_and_problem_changes() {
    const auto original = composition_input(), before = native("composition_explore", original);
    auto forged = before; forged["scenarios"][0]["candidates"][0]["status"] = "satisfied"; forged = scv::sealed(forged);
    rejects([&] { static_cast<void>(call("composition_bundle_evaluate", input(Json{{"before", before}, {"after", forged}}, "composition_reassess"))); });
    auto changed = original; changed["slots"][0]["allowed_provider_ids"] = Json::array({"aws"});
    const auto after = native("composition_explore", changed), obligations = native("composition_obligations", {{"composition", before}});
    const auto followup = input(Json{{"before", before}, {"after", after}, {"submissions", Json::array({submission(obligations.at("obligations").at(0))})}}, "composition_followup");
    rejects([&] { static_cast<void>(call("composition_bundle_evaluate", followup)); });
    forged = before; forged["domain"] = "schv-gcp"; forged = scv::sealed(forged);
    rejects([&] { static_cast<void>(call("composition_bundle_evaluate", input(Json{{"composition", forged}}, "composition_obligations"))); });
}
void test_bundle_large_logical_input_preserves_legacy_wire_limit() {
    Json repeated = Json::object(); for (int i = 0; i < 8; ++i) repeated[std::to_string(i)] = std::string(65536, 'x');
    const Json logical = {{"a", repeated}, {"b", repeated}, {"c", repeated}};
    require(logical.dump().size() > engine::Limits::max_request_bytes, "logical fixture crosses historical request bound");
    const auto payload = input(logical);
    const auto packed = payload.dump();
    require(packed.size() < engine::Limits::max_request_bytes && unpack(payload.at("bundle")).value == logical, "dedup fits unchanged process framing exactly");
    rejects([&] { static_cast<void>(engine::parse_bounded_json(logical.dump(), engine::Limits::max_request_bytes)); });
}
void test_bundle_deadline_and_shared_wire_constraints() {
    auto r = request(); r.deadline_unix_ms = 0;
    rejects([&] { static_cast<void>(scv::encode_bundle(Json::object(), r)); });
    const auto b = pack(Json::object()); rejects([&] { static_cast<void>(scv::decode_bundle(b, r)); });
    r.payload = input(Json::object()); rejects([&] { static_cast<void>(scv::handle_bundle(r, "scv")); });
}
}
int main(int argc, char** argv) { try {
    test_bundle_codec_producer_exact_sharing_and_metrics();
    test_bundle_codec_preserves_unicode_integer_and_reference_looking_values();
    test_bundle_codec_rejects_missing_cycle_unused_and_duplicate_nodes();
    test_bundle_codec_rejects_resealed_hash_and_closed_shape_changes();
    test_bundle_codec_rejects_exponential_fanout_before_materialization();
    test_bundle_codec_logical_depth_values_and_object_bounds();
    test_bundle_codec_byte_and_cumulative_materialization_bounds();
    test_bundle_inspection_is_transport_only_and_owner_pinned();
    test_composition_bundle_evaluation_producer_preserves_all_four_operations();
    test_composition_bundle_consumer_rejects_forgery_and_problem_changes();
    test_bundle_large_logical_input_preserves_legacy_wire_limit();
    test_bundle_deadline_and_shared_wire_constraints();
    std::cout << "12 bundle cases passed\n";
    if (argc == 2) {
        Json cases = Json::array(); const auto original = composition_input(), before = native("composition_explore", original);
        auto changed = original; changed["query_time"] = "2026-09-10T00:00:01Z"; const auto after = native("composition_explore", changed);
        const auto obligations = native("composition_obligations", {{"composition", before}});
        for (const auto& [operation, value] : std::vector<std::pair<std::string, Json>>{
            {"composition_explore", original}, {"composition_reassess", {{"before", before}, {"after", after}}},
            {"composition_obligations", {{"composition", before}}}, {"composition_followup", {{"before", before}, {"after", after}, {"submissions", Json::array({submission(obligations.at("obligations").at(0))})}}}}) {
            const auto payload = input(value, operation); cases.push_back({{"input", payload}, {"inspection", call("bundle_inspect", payload)}, {"result", call("composition_bundle_evaluate", payload)}, {"logical_input", value}, {"logical_result", native(operation, value)}});
        }
        const std::string raw_codec = R"({"maximum":9007199254740991,"minimum":-9007199254740991,"zero":-0,"ref":{"scalar":[],"ref":"sha256:not-an-instruction"},"empty":{},"unicode":"\u2028\u2029雪","escaped":"\\u2028","bool":false})";
        const auto codec_value = engine::parse_bounded_json(raw_codec, 4096), codec_bundle = pack(codec_value);
        const Json fixture = {{"operations", cases}, {"codec", {{"raw_input", raw_codec}, {"input", codec_value}, {"bundle", codec_bundle}, {"metrics", unpack(codec_bundle).metrics}}}};
        std::ofstream file(argv[1]); if (!file) throw std::runtime_error("cannot write bundle parity fixture"); file << fixture.dump(2) << '\n';
        if (!file) throw std::runtime_error("cannot finish bundle parity fixture");
    }
    return 0;
} catch (const std::exception& error) { std::cerr << error.what() << '\n'; return 1; } }
