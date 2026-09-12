#include "interpretation.hpp"
#include "composition.hpp"
#include "knowledge.hpp"
#include "pack.hpp"
#include <set>
#include "scv.hpp"
#include "symphony/knowledge/engine/error.hpp"

#include <functional>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <utility>
#include <vector>

namespace engine = symphony::knowledge::engine;
namespace scv = symphony::knowledge::scv;
using engine::Json;
namespace {
const std::string time0 = "2026-09-10T00:00:00Z";
void require(bool condition, const std::string& message) { if (!condition) throw std::runtime_error(message); }
template<class F> void rejects(F&& function) {
    try { function(); } catch (const engine::Error&) { return; }
    throw std::runtime_error("expected bounded owner rejection");
}
engine::Request request(const std::string& operation, Json payload) {
    return {"req-interpret-test", "corr-interpret-test", operation, "symphony-scv", engine::unix_time_ms() + 60000, std::move(payload)};
}
Json call(const std::string& op, Json payload, const std::string& domain = "scv") { return scv::handle_interpretation(request(op, std::move(payload)), domain); }
Json policy(Json age = nullptr, const std::string& partial = "exclude") {
    return {{"policy_id", "fixture-policy"}, {"max_age_seconds", age}, {"partial_capture", partial},
        {"allowed_statement_kinds", Json::array({"documented_fact", "requirement", "recommendation", "observation", "user_assertion", "inference", "hypothesis"})}};
}
Json source(const std::string& id = "docs", const std::string& provider = "cf", const std::string& family = "scev") {
    return scv::sealed(Json{{"protocol", "symphony.scv.source.v1"}, {"source_id", id}, {"provider_id", provider}, {"family_id", family},
        {"publisher", "Fixture publisher"}, {"authority_role", "documentation"}, {"scope", "fixture scope"},
        {"locators", Json::array({Json{{"locator_id", "main"}, {"uri", "https://fixture.invalid/" + id}, {"role", "primary"}, {"format", "markdown"}, {"selector", "v1"}}})},
        {"continuity_evidence", Json::array()}, {"generation", 1}, {"predecessor_digest", nullptr}});
}
Json capture(const std::string& body = "Scope: fixture\nLimit: 10 units\n", const std::string& id = "docs", const std::string& disposition = "complete", const std::string& media = "text/markdown", const std::string& provider = "cf", const std::string& family = "scev") {
    return scv::handle_source(request("capture_import", {{"source", source(id, provider, family)}, {"locator_id", "main"},
        {"resolved_uri", "https://fixture.invalid/" + id}, {"redirects", Json::array()}, {"observed_at", time0}, {"upstream_revision", nullptr},
        {"media_type", media}, {"body", body}, {"completeness", disposition},
        {"issues", disposition == "complete" ? Json::array() : Json::array({"fixture acquisition gap"})}}), "scv");
}
Json value(Json number = 8, const std::string& type = "integer", Json unit = "units") { return {{"type", type}, {"value", number}, {"unit", unit}}; }
Json rule(const std::string& id = "limit", const std::string& kind = "documented_fact", const std::string& type = "integer") {
    return {{"rule_id", "rule-" + id}, {"claim_id", id}, {"subject", "fixture-service"}, {"predicate", "maximum"},
        {"scope", {{"plan", "fixture"}}}, {"statement_kind", kind}, {"dependencies", Json::array()}, {"context", Json::array({"Scope: fixture"})},
        {"extractor", {{"kind", "delimited"}, {"prefix", "Limit: "}, {"suffix", " units"}, {"type", type}, {"unit", "units"}}}};
}
Json profile(Json rules = Json::array({rule()}), const std::string& id = "docs", const std::string& provider = "cf") {
    return scv::sealed(Json{{"protocol", "symphony.scv.interpretation-profile.v1"}, {"profile_id", "profile-" + id}, {"profile_version", "fixture-1"},
        {"provider_id", provider}, {"source_id", id}, {"locator_id", "main"}, {"media_types", Json::array({"text/markdown"})},
        {"authored_by", "Fixture author"}, {"rationale", "Explicit fixture mapping; not publisher endorsement"}, {"rules", rules}});
}
Json binding(const Json& cap, const Json& prof) { return {{"profile_digest", prof.at("digest")}, {"capture_digest", cap.at("digest")}}; }
Json interpret(Json cap = capture(), Json prof = profile(), Json selected = policy(), const std::string& domain = "scv") {
    return call("provider_interpret", {{"captures", Json::array({cap})}, {"profiles", Json::array({prof})}, {"bindings", Json::array({binding(cap, prof)})}, {"selection_policy", selected}}, domain);
}
Json how(){return {{"kind","adapter"},{"reference","fixture.adapter.v1"},{"description","Verify exact synthetic adapter"}};}
Json requirement(Json expected=8){return {{"requirement_id","capacity"},{"importance","required"},{"operator","gte"},{"right",{{"kind","literal"},{"value",value(expected)}}},{"resolution",how()}};}
Json recipe(const std::string& id,bool binds=true){return {{"recipe_id",id},{"provider_id","cf"},{"bindings",binds?Json::array({Json{{"requirement_id","capacity"},{"claim",{{"claim_id","limit"},{"subject","fixture-service"},{"scope",{{"plan","fixture"}}}}}}}):Json::array()},{"prerequisites",Json::array()},{"requires_interfaces",binds?Json::array({"fixture.store.v1"}):Json::array()},{"supplies_interfaces",binds?Json::array():Json::array({"fixture.store.v1"})},{"guarantee_changes",Json::array()},{"implementation",{{"status","available"},{"reference","fixture.adapter.v1"}}},{"resolution",how()}};}
Json composition_input(){return {{"interpretations",Json::array({interpret()})},{"additional_knowledge",Json::array()},{"provider_packs",Json::array()},{"query_time",time0},{"requirements",Json::array({requirement()})},{"slots",Json::array({Json{{"slot_id","compute"},{"allowed_provider_ids",Json::array({"cf"})},{"recipes",Json::array({recipe("first"),recipe("second")})}},Json{{"slot_id","store"},{"allowed_provider_ids",Json::array({"cf"})},{"recipes",Json::array({recipe("local",false),recipe("remote",false)})}}})},{"allowed_guarantee_changes",Json::array()},{"counterfactuals",Json::array()},{"bounds",{{"max_candidates",32}}}};}
Json compose(Json input=composition_input()){return scv::handle_composition(request("composition_explore",input),"scv");}
Json reconsider(const Json& a,const Json& b){return scv::handle_composition(request("composition_reassess",{{"before",a},{"after",b}}),"scv");}
const Json& first(const Json& result){for(const auto& s:result.at("scenarios"))if(s.at("scenario_id")=="baseline")return s.at("candidates").at(0);throw std::runtime_error("missing baseline");}

Json inventory(const Json& composition) { return scv::handle_composition(request("composition_obligations",{{"composition",composition}}),"scv"); }
Json provenance(const std::string& reference="fixture.unverified-observation") {
    return {{"kind","observation"},{"producer","fixture.author"},{"reference",reference},{"content_digest",nullptr},
        {"recorded_at",time0},{"description","Caller-supplied reference; neither content nor causation is verified"}};
}
Json submit(const Json& obligation,const std::string& id="submission-1") {
    return {{"submission_id",id},{"obligation_id",obligation.at("obligation_id")},{"provenance",provenance()}};
}
Json follow(const Json& before,const Json& after,Json submissions) {
    return scv::handle_composition(request("composition_followup",{{"before",before},{"after",after},{"submissions",submissions}}),"scv");
}
Json only_one(Json input=composition_input()) {
    for(auto& slot:input["slots"])slot["recipes"]=Json::array({slot.at("recipes").at(0)});
    return input;
}
Json unsatisfied_input() {auto input=only_one();input["requirements"][0]["right"]["value"]["value"]=12;return input;}
Json obligation_of(const Json& result,const std::string& kind) {
    for(const auto& entry:result.at("obligations"))if(entry.at("kind")==kind)return entry;
    throw std::runtime_error("missing obligation kind: "+kind);
}
Json requirement_check(const Json& input,const std::string& id) {
    return {{"check_id",id},{"importance","required"},{"operator","gte"},{"right",{{"kind","literal"},{"value",value(12)}}},
        {"left",input.at("slots").at(0).at("recipes").at(0).at("bindings").at(0).at("claim")}};
}
void test_composition_obligations_producer() {
    const auto composition=compose(unsatisfied_input()),result=inventory(composition);
    require(result.at("protocol")=="symphony.scv.composition-obligations.v1"&&result.at("input").at("composition")==composition,"exact composition retained");
    require(result.at("obligations").size()==3,"one failed check and two implementation declarations");
    std::string previous;
    for(const auto& entry:result.at("obligations")) {
        const auto id=entry.at("obligation_id").get<std::string>();require(previous.empty()||previous<id,"lexical inventory ordering");previous=id;
        require(entry.at("target").size()==10,"closed precise target");
        require(scv::sealed(Json{{"composition_digest",composition.at("digest")},{"kind",entry.at("kind")},{"target",entry.at("target")}}).at("digest")==entry.at("obligation_id"),"identity binds exact composition/kind/target");
    }
    const auto check=obligation_of(result,"check");require(check.at("prior_status")=="contradicted"&&check.at("target").at("connection_id")=="requirements"&&check.at("target").at("check_id")=="capacity","precise common check identity");
    require(check.at("target").at("requirement_id").is_null(),"check target does not invent redundant requirement target");
    require(inventory(composition)==result,"deterministic exact replay");
}
void test_composition_obligations_distinguish_identical_prerequisite_summaries() {
    auto input=only_one();input["slots"][0]["recipes"][0]["prerequisites"]=Json::array({requirement_check(input,"alpha"),requirement_check(input,"beta")});
    const auto composition=compose(input),result=inventory(composition);
    Json prior=Json::array();for(const auto& item:first(composition).at("obligations"))if(item.at("kind")=="check_contradicted")prior.push_back(item);
    require(prior.size()==2&&prior[0]==prior[1],"fixture reproduces ambiguous anonymous legacy summaries");
    std::set<std::string> ids,checks;
    for(const auto& item:result.at("obligations"))if(item.at("kind")=="check") {ids.insert(item.at("obligation_id"));checks.insert(item.at("target").at("check_id"));}
    require(ids.size()==2&&checks==std::set<std::string>{"alpha","beta"},"precise prerequisite identities remain distinct");
}
void test_composition_obligations_include_binding_interface_and_implementation() {
    auto input=only_one();input["slots"][0]["recipes"][0]["bindings"]=Json::array();input["slots"][1]["recipes"][0]["supplies_interfaces"]=Json::array();
    input["slots"][0]["recipes"][0]["implementation"]={{"status","unimplemented"},{"reference",nullptr}};
    auto result=inventory(compose(input));
    require(obligation_of(result,"binding").at("prior_status")=="missing_binding","missing binding inventoried");
    require(obligation_of(result,"interface").at("target").at("interface_ref")=="fixture.store.v1","exact missing interface inventoried");
    std::set<std::string> states;for(const auto& item:result.at("obligations"))if(item.at("kind")=="implementation")states.insert(item.at("prior_status"));
    require(states==std::set<std::string>{"available","unimplemented"},"available still requires separate verification");
    input=only_one();input["slots"][1]["recipes"][0]["bindings"]=input["slots"][0]["recipes"][0]["bindings"];
    result=inventory(compose(input));const auto ambiguous=obligation_of(result,"binding");
    require(ambiguous.at("prior_status")=="ambiguous_binding"&&ambiguous.at("claim_ids")==Json::array({"limit"}),"ambiguous refs deduplicated without credit");
}
void test_composition_followup_consumer_rejects_resealed_result() {
    const auto before=compose(unsatisfied_input()),obligation=obligation_of(inventory(before),"check");
    auto forged=before;forged["scenarios"][0]["candidates"][0]["status"]="satisfied";forged=scv::sealed(forged);
    rejects([&]{static_cast<void>(inventory(forged));});
    rejects([&]{static_cast<void>(follow(before,forged,Json::array({submit(obligation)})));});
    rejects([&]{static_cast<void>(follow(forged,before,Json::array({submit(obligation)})));});
    forged=before;forged["digest"]="sha256:"+std::string(64,'0');rejects([&]{static_cast<void>(inventory(forged));});
}
void test_composition_followup_criterion_change_does_not_credit_reference() {
    auto input=unsatisfied_input();const auto before=compose(input),obligation=obligation_of(inventory(before),"check");
    input["interpretations"]=Json::array({interpret(capture("Scope: fixture\nLimit: 14 units\n"))});const auto after=compose(input);
    auto submission=submit(obligation);submission["provenance"]["reference"]="https://unrelated.invalid/not-fetched";
    const auto result=follow(before,after,Json::array({submission}));const auto& entry=result.at("entries").at(0);
    require(entry.at("prior_status")=="contradicted"&&entry.at("current_status")=="satisfied"&&entry.at("outcome")=="criterion_satisfied","new native evidence satisfies exact check");
    require(entry.at("provenance_validation")=="reference_only"&&entry.at("causation")=="not_established","unrelated opaque reference never credited");
    require(result.at("input").at("submissions").at(0)==submission,"opaque provenance retained exactly");
    require(result.at("candidate_changes")==reconsider(before,after).at("candidates"),"reassessment candidate changes retained exactly");
}
void test_composition_followup_unchanged_and_missing_evidence_remain_unsatisfied() {
    auto input=unsatisfied_input();const auto before=compose(input),obligation=obligation_of(inventory(before),"check");
    require(follow(before,before,Json::array({submit(obligation)})).at("entries").at(0).at("outcome")=="criterion_not_satisfied","unchanged contradiction stays");
    input["interpretations"]=Json::array({interpret(capture("Scope: fixture\nNo documented limit\n"))});
    const auto result=follow(before,compose(input),Json::array({submit(obligation)}));
    require(result.at("entries").at(0).at("current_status")=="unresolved"&&result.at("entries").at(0).at("outcome")=="criterion_not_satisfied","missing evidence is unresolved not satisfaction");
}
void test_composition_followup_partial_evidence_stays_conditional() {
    auto input=unsatisfied_input();input["interpretations"]=Json::array({interpret(capture(),profile(),policy(nullptr,"include_qualified"))});
    const auto before=compose(input),obligation=obligation_of(inventory(before),"check");
    input["interpretations"]=Json::array({interpret(capture("Scope: fixture\nLimit: 14 units\n","docs","partial"),profile(),policy(nullptr,"include_qualified"))});
    const auto result=follow(before,compose(input),Json::array({submit(obligation)}));
    require(result.at("entries").at(0).at("current_status")=="conditional"&&result.at("entries").at(0).at("outcome")=="criterion_not_satisfied","partial qualification survives matched comparison");
}
void test_composition_followup_noncheck_needs_separate_verification() {
    auto input=only_one();input["slots"][0]["recipes"][0]["bindings"]=Json::array();input["slots"][1]["recipes"][0]["supplies_interfaces"]=Json::array();
    const auto before=compose(input),items=inventory(before);Json submissions=Json::array();
    for(const auto* kind:{"binding","interface","implementation"})submissions.push_back(submit(obligation_of(items,kind),kind));
    const auto result=follow(before,before,submissions);
    for(const auto& entry:result.at("entries"))require(entry.at("current_status").is_null()&&entry.at("outcome")=="requires_separate_verification","non-check opaque reference never closes native obligation");
}
void test_composition_followup_preserves_problem_and_policy() {
    auto input=unsatisfied_input();const auto before=compose(input),obligation=obligation_of(inventory(before),"check");
    const auto rejected=[&](Json changed){const auto after=compose(changed);rejects([&]{static_cast<void>(follow(before,after,Json::array({submit(obligation)})));});};
    auto changed=input;changed["requirements"][0]["right"]["value"]["value"]=8;rejected(changed);
    changed=input;changed["slots"][0]["recipes"][0]["implementation"]["reference"]="different.adapter";rejected(changed);
    changed=input;changed["allowed_guarantee_changes"]=Json::array({"permitted-change"});rejected(changed);
    changed=input;changed["counterfactuals"]=Json::array({Json{{"counterfactual_id","other"},{"requirements",changed.at("requirements")}}});rejected(changed);
    changed=input;changed["bounds"]["max_candidates"]=1;rejected(changed);
    changed=input;auto selected=policy();selected["policy_id"]="new-policy";changed["interpretations"]=Json::array({interpret(capture(),profile(),selected)});rejected(changed);
}
void test_composition_followup_reports_time_without_implying_chronology() {
    auto input=unsatisfied_input();const auto before=compose(input),obligation=obligation_of(inventory(before),"check");
    input["query_time"]="2026-09-09T00:00:00Z";const auto after=compose(input),result=follow(before,after,Json::array({submit(obligation)}));
    require(result.at("change_axes").at("query_time")==true&&result.at("change_axes").at("evidence")==false,"earlier query time allowed and explicit");
    require(result.at("entries").at(0).at("current_status")=="unresolved"&&result.at("entries").at(0).at("causation")=="not_established","future relative observation remains unresolved");
}
void test_composition_followup_rejects_duplicate_and_wrong_targets() {
    const auto before=compose(unsatisfied_input()),items=inventory(before),check=obligation_of(items,"check"),implementation=obligation_of(items,"implementation");
    auto a=submit(check),b=submit(implementation);rejects([&]{static_cast<void>(follow(before,before,Json::array({a,b})));});
    b=submit(check,"different-id");rejects([&]{static_cast<void>(follow(before,before,Json::array({a,b})));});
    a["obligation_id"]="sha256:"+std::string(64,'0');rejects([&]{static_cast<void>(follow(before,before,Json::array({a})));});
    auto otherInput=unsatisfied_input();otherInput["query_time"]="2026-09-10T00:00:01Z";
    a=submit(obligation_of(inventory(compose(otherInput)),"check"));rejects([&]{static_cast<void>(follow(before,before,Json::array({a})));});
}
void test_composition_followup_validates_provenance_and_closed_shapes() {
    const auto before=compose(unsatisfied_input()),check=obligation_of(inventory(before),"check");
    for(const auto& key:std::vector<std::string>{"producer","reference","recorded_at","description"}) {
        auto item=submit(check);item["provenance"][key]=nullptr;rejects([&]{static_cast<void>(follow(before,before,Json::array({item})));});
    }
    for(const auto& bad:std::vector<std::pair<std::string,Json>>{{"kind","execute"},{"recorded_at","2026-09-10T00:00:00+00:00"},{"recorded_at","2026-02-30T00:00:00Z"},{"content_digest","sha256:"+std::string(64,'A')},{"reference","bad\nreference"},{"producer",std::string(513,'x')},{"description",std::string(2049,'x')}}) {
        auto item=submit(check);item["provenance"][bad.first]=bad.second;rejects([&]{static_cast<void>(follow(before,before,Json::array({item})));});
    }
    auto item=submit(check);item["unexpected"]=true;rejects([&]{static_cast<void>(follow(before,before,Json::array({item})));});
    item=submit(check);item["provenance"]["execute"]=false;rejects([&]{static_cast<void>(follow(before,before,Json::array({item})));});
    item=submit(check);item["provenance"]["content_digest"]="sha256:"+std::string(64,'a');
    require(follow(before,before,Json::array({item})).at("entries").at(0).at("provenance_validation")=="reference_only","well-shaped hash remains unverified content reference");
}
void test_composition_obligations_and_followup_bounds_and_deadline() {
    const auto before=compose(unsatisfied_input()),check=obligation_of(inventory(before),"check");
    rejects([&]{static_cast<void>(follow(before,before,Json::array()));});Json too_many=Json::array();for(int i=0;i<33;i++)too_many.push_back(submit(check,"item-"+std::to_string(i)));rejects([&]{static_cast<void>(follow(before,before,too_many));});
    auto r=request("composition_obligations",{{"composition",before}});r.deadline_unix_ms=0;rejects([&]{static_cast<void>(scv::handle_composition(r,"scv"));});
    r=request("composition_followup",{{"before",before},{"after",before},{"submissions",Json::array({submit(check)})}});r.deadline_unix_ms=0;rejects([&]{static_cast<void>(scv::handle_composition(r,"scv"));});
    rejects([&]{static_cast<void>(scv::handle_composition(request("composition_obligations",{{"composition",before},{"extra",true}}),"scv"));});
}
void test_composition_followup_sorts_entries_but_retains_input() {
    const auto before=compose(unsatisfied_input()),items=inventory(before);const auto submissions=Json::array({submit(obligation_of(items,"check"),"z"),submit(obligation_of(items,"implementation"),"a")});
    const auto result=follow(before,before,submissions);
    require(result.at("entries").at(0).at("submission_id")=="a"&&result.at("entries").at(1).at("submission_id")=="z","submission output sorted by identity");
    require(result.at("input").at("submissions")==submissions,"authored submission order retained");
}
void test_composition_obligations_fixture_inventory_and_separate_verification() {
    const auto cap=capture(),prof=profile();
    Json fixture={{"fixture_id","mapping-fixture"},{"captures",Json::array({cap})},{"bindings",Json::array({Json{{"profile_id",prof.at("profile_id")},{"capture_digest",cap.at("digest")}}})},{"selection_policy",policy()}};
    // A manually specified fixture projection deliberately disagrees with the
    // actual extracted number. The resulting failure must remain inspectable.
    Json expected={{"claim_id","limit"},{"subject","fixture-service"},{"predicate","maximum"},{"scope",{{"plan","fixture"}}},{"statement_kind","documented_fact"},{"value",value(999)},{"dependencies",Json::array()}};
    auto desired=source();for(const auto* key:{"protocol","generation","predecessor_digest","digest"})desired.erase(key);
    Json manifest={{"fixture_id","mapping-fixture"},{"label","Independently authored mismatch"},{"authored_by","Fixture author"},{"rationale","Expected999 deliberately differs from documentary10"},
        {"input_digest",nullptr},{"expected_claims",Json::array({expected})},{"expected_extractions",Json::array({Json{{"profile_id",prof.at("profile_id")},{"capture_digest",cap.at("digest")},
        {"rule_id","rule-limit"},{"claim_id","limit"},{"status","matched"},{"reasons",Json::array()}}})}};
    Json draft={{"protocol","symphony.scv.provider-pack.v1"},{"pack_id","obligation-fixture-pack"},{"pack_version","v1"},{"authored_by","Fixture author"},{"provenance",Json::array({"Independent expected mismatch"})},
        {"provider",{{"provider_id","cf"},{"family_id","scev"},{"display_name","Fixture CF"},{"sources",Json::array({desired})}}},
        {"profiles",Json::array({prof})},{"structured_profiles",Json::array()},{"fixtures",Json::array({manifest})}};
    auto prepared=scv::handle_pack(request("provider_pack_prepare",{{"pack",draft},{"fixtures",Json::array({fixture})}}),"scv");
    auto evaluated=scv::handle_pack(request("provider_pack_evaluate",{{"pack",prepared},{"captures",Json::array({cap})},{"bindings",fixture.at("bindings")},{"selection_policy",policy()},{"fixtures",Json::array({fixture})}}),"scv");
    auto input=only_one();input["interpretations"]=Json::array();input["provider_packs"]=Json::array({evaluated});const auto before=compose(input),issue=obligation_of(inventory(before),"fixture");
    require(issue.at("prior_status")=="failed"&&issue.at("target").at("pack_evaluation_digest")==evaluated.at("digest")&&issue.at("claim_ids")==Json::array({"limit"}),"global exact failed fixture retained");
    const auto result=follow(before,before,Json::array({submit(issue)}));require(result.at("entries").at(0).at("current_status").is_null()&&result.at("entries").at(0).at("outcome")=="requires_separate_verification","fixture is not implicitly closed");
    draft["fixtures"][0]["expected_claims"][0]["value"]=value(10);
    prepared=scv::handle_pack(request("provider_pack_prepare",{{"pack",draft},{"fixtures",Json::array({fixture})}}),"scv");
    evaluated=scv::handle_pack(request("provider_pack_evaluate",{{"pack",prepared},{"captures",Json::array({cap})},{"bindings",fixture.at("bindings")},{"selection_policy",policy()},{"fixtures",Json::array({fixture})}}),"scv");
    input["provider_packs"]=Json::array({evaluated});const auto after=compose(input);require(after.at("evidence_obligations").empty(),"later selected pack fixture passes independently");
    const auto later=follow(before,after,Json::array({submit(issue)}));require(later.at("entries").at(0).at("current_status").is_null()&&later.at("entries").at(0).at("outcome")=="requires_separate_verification","changed pack does not silently map or close old fixture target");
    evaluated=scv::handle_pack(request("provider_pack_evaluate",{{"pack",prepared},{"captures",Json::array({cap})},{"bindings",fixture.at("bindings")},{"selection_policy",policy()},{"fixtures",Json::array()}}),"scv");
    input["provider_packs"]=Json::array({evaluated});require(obligation_of(inventory(compose(input)),"fixture").at("prior_status")=="not_run","unrun fixture remains global obligation");
}
}
int main(int argc,char** argv){try{
    test_composition_obligations_producer();
    test_composition_obligations_distinguish_identical_prerequisite_summaries();
    test_composition_obligations_include_binding_interface_and_implementation();
    test_composition_followup_consumer_rejects_resealed_result();
    test_composition_followup_criterion_change_does_not_credit_reference();
    test_composition_followup_unchanged_and_missing_evidence_remain_unsatisfied();
    test_composition_followup_partial_evidence_stays_conditional();
    test_composition_followup_noncheck_needs_separate_verification();
    test_composition_followup_preserves_problem_and_policy();
    test_composition_followup_reports_time_without_implying_chronology();
    test_composition_followup_rejects_duplicate_and_wrong_targets();
    test_composition_followup_validates_provenance_and_closed_shapes();
    test_composition_obligations_and_followup_bounds_and_deadline();
    test_composition_followup_sorts_entries_but_retains_input();
    test_composition_obligations_fixture_inventory_and_separate_verification();
    std::cout<<"15 obligation cases passed\n";
    if(argc==2){
        auto input=unsatisfied_input();input["slots"][0]["recipes"][0]["prerequisites"]=Json::array({requirement_check(input,"alpha"),requirement_check(input,"beta")});
        const auto before=compose(input);input["interpretations"]=Json::array({interpret(capture("Scope: fixture\nLimit: 14 units\n"))});const auto after=compose(input),items=inventory(before);
        Json submissions=Json::array();int count=0;for(const auto& item:items.at("obligations"))submissions.push_back(submit(item,"submission-"+std::to_string(++count)));
        std::ofstream file(argv[1]);if(!file)throw std::runtime_error("cannot write obligation parity fixture");
        file<<Json{{"before",before},{"after",after},{"obligations_input",{{"composition",before}}},{"obligations_result",items},
            {"followup_input",{{"before",before},{"after",after},{"submissions",submissions}}},{"followup_result",follow(before,after,submissions)}}.dump(2)<<'\n';
        if(!file)throw std::runtime_error("cannot finish obligation parity fixture");
    }
    return 0;
}catch(const std::exception& e){std::cerr<<e.what()<<'\n';return 1;}}
