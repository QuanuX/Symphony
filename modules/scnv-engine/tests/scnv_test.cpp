#include <symphony/snv/scnv.hpp>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <functional>
#include <iostream>
#include <stdexcept>
#include <string>

namespace {
using symphony::snv::Json;
namespace scnv = symphony::snv::scnv;
namespace engine = symphony::knowledge::engine;
std::size_t tests = 0;
void expect(bool condition, const char* label) {
  if (!condition) throw std::runtime_error(label);
  ++tests;
}
template <class F> void rejected(F operation, const char* label,
                                const char* expected_code = nullptr) {
  try { operation(); } catch (const engine::Error& error) {
    if (expected_code && error.code() != expected_code) throw std::runtime_error(label);
    ++tests; return;
  }
  throw std::runtime_error(label);
}
Json association(const std::string& id, const std::string& subject,
                 const std::string& name = "annex", const std::string& scope = "cluster-1") {
  return {{"record_id", id}, {"association_id", id}, {"kind", "assignment"},
    {"previous_record_id", nullptr}, {"origin_record_id", nullptr},
    {"subject", {{"kind", "node"}, {"id", subject}}}, {"name", name},
    {"name_kind", "short_name"}, {"namespace", "user-names"},
    {"source", {{"kind", "user"}, {"id", "duncan"}}},
    {"scope", {{"kind", "cluster"}, {"id", scope}}},
    {"interval", {{"from_unix_ms", 100}, {"until_unix_ms", nullptr}}},
    {"observed_unix_ms", 100}, {"recorded_unix_ms", 110}};
}
Json validation(Json records = Json::array()) {
  return {{"protocol", "scnv.names.v1"}, {"mode", "records"},
    {"records", records}, {"coverage", "complete"}, {"limits", {{"max_records", 512}}}};
}
Json query(Json records, const std::string& name = "annex", const std::string& scope = "cluster-1",
           std::int64_t at = 150) {
  return {{"protocol", "scnv.resolve.v1"}, {"records", records}, {"coverage", "complete"},
    {"limits", {{"max_records", 512}}}, {"query", {
      {"mode", "exact"}, {"name", name}, {"scope", {{"kind", "cluster"}, {"id", scope}}},
      {"namespace", nullptr}, {"name_kind", nullptr}, {"as_of_unix_ms", at},
      {"offset", 0}, {"limit", 128}, {"expected_evidence_digest", nullptr}}}};
}
Json invoke(const char* operation, const Json& payload) {
  return scnv::handle(operation, payload, engine::unix_time_ms() + 30000);
}
Json validate(const Json& payload) { return invoke("names_validate", payload); }
Json resolve(const Json& payload) { return invoke("names_resolve", payload); }
Json transition(Json records, const char* action, Json proposed) {
  auto p = validation(records);
  p["mode"] = "transition";
  p["policy"] = {{"id", "user-naming-policy"}, {"version", "1"},
    {"comparison", "utf8_bytes"}, {"scope", "exact_scope"},
    {"collision", "report"}, {"interval", "half_open"}};
  p["intent"] = {{"action", action},
    {"expected_evidence_digest", validate(validation(records)).at("evidence_digest")},
    {"new_record", proposed}, {"reason", "explicit caller-selected evidence change"}};
  return p;
}
}
int main() {
  try {
    const auto a = association("a", "node-a");
    auto b = association("b", "node-b");
    auto p = validation(Json::array({a}));
    const auto initial = validate(p);
    expect(initial.at("valid") == true && initial.at("owner") == "scnv" &&
      initial.at("owner_version") == "0.1.0-dev", "owner metadata");
    expect(initial.at("subject_ids") == Json::array({"node-a"}), "subject evidence IDs");
    expect(initial.at("source_digest") == engine::tagged_sha256(p.dump()), "source digest binds whole input");
    expect(resolve(query(Json::array({a}))).at("resolved_subject").at("id") == "node-a", "nickname is usable selector");
    expect(resolve(query(Json::array({a}), "missing")).at("status") == "absent", "absent result");
    auto partial = query(Json::array({a})); partial["coverage"] = "partial";
    expect(resolve(partial).at("status") == "insufficient", "partial evidence cannot certify uniqueness");
    partial["records"] = Json::array();
    expect(resolve(partial).at("status") == "insufficient", "partial absence stays insufficient");
    b["scope"]["id"] = "cluster-2";
    expect(validate(validation(Json::array({a,b}))).at("valid") == true, "same spelling distinct clusters");
    expect(resolve(query(Json::array({a,b}), "annex", "cluster-2")).at("resolved_subject").at("id") == "node-b", "explicit cluster qualification");
    b["scope"] = {{"kind", "tops"}, {"id", "cluster-1"}};
    expect(validate(validation(Json::array({a,b}))).at("valid") == true, "scope kinds do not collapse");
    auto q = query(Json::array({a,b})); q["query"]["scope"] = b.at("scope");
    expect(resolve(q).at("resolved_subject").at("id") == "node-b", "TOPS scope selector");
    b["scope"]["kind"] = "trog"; q["records"] = Json::array({a,b}); q["query"]["scope"] = b.at("scope");
    expect(resolve(q).at("resolved_subject").at("id") == "node-b", "TROG scope selector");
    b["scope"] = a.at("scope");
    const auto collision = validate(validation(Json::array({a,b})));
    expect(collision.at("valid") == false && collision.at("collisions").size() == 1, "active collision explicit");
    expect(resolve(query(Json::array({a,b}))).at("status") == "ambiguous", "ambiguous never first match");
    b["namespace"] = "provider-labels"; b["name_kind"] = "provider_resource_id";
    expect(resolve(query(Json::array({a,b}))).at("status") == "ambiguous", "unqualified namespace collision");
    q = query(Json::array({a,b})); q["query"]["namespace"] = "provider-labels";
    expect(resolve(q).at("resolved_subject").at("id") == "node-b", "exact source namespace qualification");
    b = association("b", "node-a");
    expect(resolve(query(Json::array({a,b}))).at("status") == "unique", "multiple names for one subject remain unique");
    b = association("b", "node-b", "Annex");
    expect(validate(validation(Json::array({a,b}))).at("valid") == true, "case comparison exact");
    expect(resolve(query(Json::array({a,b}), "Annex")).at("resolved_subject").at("id") == "node-b", "case preserved");
    auto unicode_a = association("unicode-a", "node-a", "caf\xc3\xa9");
    auto unicode_b = association("unicode-b", "node-b", "cafe\xcc\x81");
    expect(validate(validation(Json::array({unicode_a,unicode_b}))).at("valid") == true, "no invented Unicode normalization");
    expect(resolve(query(Json::array({unicode_a,unicode_b}), "cafe\xcc\x81")).at("resolved_subject").at("id") == "node-b", "decomposed Unicode spelling retained");
    const auto escaped = association("escaped", "node-a", "cluster/annex\\\"::x");
    expect(resolve(query(Json::array({escaped}), "cluster/annex\\\"::x")).at("status") == "unique", "punctuation is name data");
    q = query(Json::array({a})); q["query"]["scope"]["kind"] = "global";
    expect(resolve(q).at("status") == "unsupported_scope", "unsupported scope explicit");
    q = query(Json::array({a}), "annex", "cluster-1", 99);
    expect(resolve(q).at("status") == "absent", "before applicable interval");
    auto closed = a; closed["interval"]["until_unix_ms"] = 200;
    b = association("b", "node-b"); b["interval"]["from_unix_ms"] = 200;
    expect(validate(validation(Json::array({closed,b}))).at("valid") == true, "half-open retirement reuse boundary");
    expect(resolve(query(Json::array({closed,b}), "annex", "cluster-1", 199)).at("resolved_subject").at("id") == "node-a", "historical predecessor");
    expect(resolve(query(Json::array({closed,b}), "annex", "cluster-1", 200)).at("resolved_subject").at("id") == "node-b", "successor at reuse boundary");
    const auto first_binding = resolve(query(Json::array({a}))).at("resolution_binding");
    const auto later_binding = resolve(query(Json::array({closed,b}), "annex", "cluster-1", 200)).at("resolution_binding");
    expect(first_binding.at("subject").at("id") == "node-a" && later_binding.at("subject").at("id") == "node-b" &&
      first_binding.at("binding_digest") != later_binding.at("binding_digest"), "alias reassignment cannot redirect prior binding");
    auto malformed = p; malformed["ignored"] = true;
    rejected([&]{validate(malformed);}, "unknown payload field rejected");
    malformed = p; malformed["records"][0]["ignored"] = true;
    rejected([&]{validate(malformed);}, "unknown record field rejected");
    malformed = p; malformed["protocol"] = "scnv.names.v2";
    rejected([&]{validate(malformed);}, "unknown version rejected");
    malformed = p; malformed["records"][0]["name"] = "";
    rejected([&]{validate(malformed);}, "empty name rejected");
    malformed["records"][0]["name"] = std::string(513, 'x');
    rejected([&]{validate(malformed);}, "name byte bound");
    malformed["records"][0]["name"] = "bad\nname";
    rejected([&]{validate(malformed);}, "embedded control rejected");
    malformed["records"][0]["name"] = std::string("\xc0\xaf", 2);
    rejected([&]{validate(malformed);}, "overlong UTF-8 rejected in SDK");
    malformed["records"][0]["name"] = std::string("\xed\xa0\x80", 3);
    rejected([&]{validate(malformed);}, "UTF-8 surrogate rejected");
    malformed = p; malformed["records"][0]["interval"]["until_unix_ms"] = 100;
    rejected([&]{validate(malformed);}, "empty interval rejected");
    malformed = p; malformed["records"][0]["observed_unix_ms"] = -1;
    rejected([&]{validate(malformed);}, "negative time rejected");
    malformed["records"][0]["observed_unix_ms"] = 1.5;
    rejected([&]{validate(malformed);}, "fractional time rejected");
    malformed["records"][0]["observed_unix_ms"] = std::uint64_t{18446744073709551615ULL};
    rejected([&]{validate(malformed);}, "time conversion overflow rejected");
    malformed = p; malformed["records"].push_back(a);
    rejected([&]{validate(malformed);}, "duplicate record IDs rejected");
    malformed = p; malformed["limits"]["max_records"] = 513;
    rejected([&]{validate(malformed);}, "unsupported record profile");
    auto retire = a; retire["record_id"] = "retired-a"; retire["kind"] = "retirement";
    retire["previous_record_id"] = "a"; retire["interval"]["until_unix_ms"] = 200;
    const auto retirement = validate(transition(Json::array({a}), "retire", retire));
    expect(retirement.at("proposed_records") == Json::array({retire}) && retirement.at("valid") == true,
      "retirement produces exact supplied candidate");
    expect(resolve(query(Json::array({a,retire}), "annex", "cluster-1", 200)).at("status") == "absent", "superseded open record cannot keep retired name active");
    expect(resolve(query(Json::array({a,retire}), "annex", "cluster-1", 199)).at("resolved_subject").at("id") == "node-a", "retirement preserves earlier applicability");
    auto reuse = b; reuse["kind"] = "reuse"; reuse["origin_record_id"] = "retired-a";
    expect(validate(transition(Json::array({a,retire}), "reuse", reuse)).at("valid") == true, "explicit reuse after retirement");
    auto restore = association("restored-a", "node-a"); restore["kind"] = "restoration";
    restore["origin_record_id"] = "retired-a"; restore["interval"]["from_unix_ms"] = 300;
    expect(validate(transition(Json::array({a,retire}), "restore", restore)).at("valid") == true, "restoration retains separated applicability");
    expect(resolve(query(Json::array({a,retire,restore}), "annex", "cluster-1", 250)).at("status") == "absent", "restoration cannot bridge retired gap");
    restore["subject"]["id"] = "node-b";
    rejected([&]{validate(transition(Json::array({a,retire}), "restore", restore));}, "restoration same subject");
    auto correction = a; correction["record_id"] = "corrected-a"; correction["kind"] = "correction";
    correction["previous_record_id"] = "a"; correction["name"] = "annex-corrected";
    expect(validate(transition(Json::array({a}), "correct", correction)).at("proposed_records")[0].at("name") == "annex-corrected", "typed correction preserves supplied spelling");
    expect(resolve(query(Json::array({a,correction}), "annex-corrected")).at("status") == "unique", "corrected association head resolves");
    expect(resolve(query(Json::array({a,correction}), "annex")).at("status") == "absent", "superseded correction cannot remain live");
    auto assign = transition(Json::array(), "assign", a);
    expect(validate(assign).at("proposed_records") == Json::array({a}), "user assignment no name generation");
    assign["intent"]["expected_evidence_digest"] = "sha256:" + std::string(64, '0');
    rejected([&]{validate(assign);}, "stale predecessor refuses", "scnv.expected_evidence_conflict");
    auto bad_policy = transition(Json::array(), "assign", a); bad_policy["policy"]["comparison"] = "case_fold";
    rejected([&]{validate(bad_policy);}, "policy normalization unsupported");
    auto bad_retire = retire; bad_retire["subject"]["id"] = "node-b";
    rejected([&]{validate(transition(Json::array({a}), "retire", bad_retire));}, "retirement cannot reassign subject");
    auto fork = correction; fork["record_id"] = "competing";
    rejected([&]{validate(validation(Json::array({a,correction,fork})));}, "lineage fork rejected", "scnv.lineage_conflict");
    malformed = validation(Json::array({correction}));
    rejected([&]{validate(malformed);}, "missing predecessor rejected");
    auto cyclic = correction; cyclic["record_id"] = "cycle-b"; cyclic["previous_record_id"] = "cycle-a";
    auto cyclic_a = cyclic; cyclic_a["record_id"] = "cycle-a"; cyclic_a["previous_record_id"] = "cycle-b";
    rejected([&]{validate(validation(Json::array({cyclic,cyclic_a})));}, "revision cycle rejected");
    auto two_roots = a; two_roots["record_id"] = "second-root";
    rejected([&]{validate(validation(Json::array({a,two_roots})));}, "association may not have competing roots");
    auto wrong_origin = reuse; wrong_origin["origin_record_id"] = "a";
    rejected([&]{validate(validation(Json::array({a,retire,wrong_origin})));}, "reuse requires retired terminal origin");
    auto premature = reuse; premature["interval"]["from_unix_ms"] = 199;
    rejected([&]{validate(validation(Json::array({a,retire,premature})));}, "reuse may not overlap retired origin");
    auto enumeration = query(Json::array({a,association("c", "node-c", "storage")}));
    enumeration["query"]["mode"] = "scoped_candidates"; enumeration["query"]["name"] = nullptr;
    enumeration["query"]["limit"] = 1;
    const auto page = resolve(enumeration);
    expect(page.at("status") == "candidates" && page.at("match_count") == 2 && page.at("candidates").size() == 1 && page.at("next_offset") == 1,
      "bounded candidate enumeration explicit continuation");
    enumeration["query"]["offset"] = 1;
    rejected([&]{resolve(enumeration);}, "unbound continuation rejected", "scnv.snapshot_required");
    enumeration["query"]["expected_evidence_digest"] = page.at("evidence_digest");
    const auto next = resolve(enumeration);
    expect(next.at("candidates")[0].at("name") == "storage" && next.at("next_offset").is_null(), "snapshot bound second page");
    enumeration["records"][0]["name"] = "new-alias";
    rejected([&]{resolve(enumeration);}, "changed evidence pagination refuses", "scnv.expected_evidence_conflict");
    auto large = validation();
    for (std::size_t i = 0; i < 512; ++i)
      large["records"].push_back(association("record-" + std::to_string(i), "node-" + std::to_string(i), "name-" + std::to_string(i)));
    expect(validate(large).at("record_count") == 512, "512 profile full capacity");
    large["records"].push_back(association("overflow", "overflow", "overflow"));
    rejected([&]{validate(large);}, "capacity never truncates", "scnv.capacity_exceeded");
    large["limits"]["max_records"] = 1024;
    expect(validate(large).at("record_count") == 513, "explicit next capacity profile");
    rejected([&]{scnv::handle("names_validate", p, 0);}, "deadline expired", "request.deadline_expired");
    rejected([&]{invoke("invented", p);}, "unknown operation", "operation.unsupported");
    expect(scnv::operations().size() == 2 && scnv::operations()[1].name == "names_resolve", "exact operation metadata");
    std::cout << "SCNV: " << tests << " independent semantic expectations passed\n";
    return 0;
  } catch (const std::exception& error) {
    std::cerr << error.what() << '\n'; return 1;
  }
}
