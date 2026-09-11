#include "pack.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <functional>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <vector>
using symphony::knowledge::engine::Json;
namespace engine=symphony::knowledge::engine;
namespace scv=symphony::knowledge::scv;
namespace {
void require(bool c,const std::string& m){if(!c)throw std::runtime_error(m);}
template<class F>void rejects(F f){try{f();}catch(const engine::Error&){return;}throw std::runtime_error("expected rejection");}
engine::Request req(const std::string& op,Json p){return {"pack-test","pack-test",op,"symphony-scv",engine::unix_time_ms()+60000,std::move(p)};}
Json desired(const std::string& provider="independent",const std::string& family="schv") {
 return {{"source_id","docs"},{"provider_id",provider},{"family_id",family},{"publisher","Fixture publisher"},{"authority_role","reference"},{"scope","authored fixture"},
 {"locators",Json::array({Json{{"locator_id","main"},{"uri","https://fixture.invalid/docs"},{"role","primary"},{"format","json"},{"selector","fixture-1"}}})},{"continuity_evidence",Json::array()}};
}
Json policy(const std::string& partial="exclude"){return {{"policy_id","fixture-policy"},{"partial_capture",partial},{"max_age_seconds",60},{"allowed_statement_kinds",Json::array({"documented_fact","recommendation"})}};}
Json cap(const std::string& body="{\"plan\":\"basic\",\"maximum\":10}",const std::string& status="complete",Json source=desired()) {
 source["protocol"]="symphony.scv.source.v1";source["generation"]=1;source["predecessor_digest"]=nullptr;source=scv::sealed(source);
 return scv::handle_source(req("capture_import",{{"source",source},{"locator_id","main"},{"resolved_uri","https://fixture.invalid/docs"},{"redirects",Json::array()},{"observed_at","2026-09-10T00:00:00Z"},{"upstream_revision",nullptr},{"media_type","application/json"},{"body",body},{"completeness",status},{"issues",status=="complete"?Json::array():Json::array({"fixture issue"})}}),"scv");
}
Json value(const std::string& type="integer",Json v=10){return {{"type",type},{"value",v},{"unit","units"}};}
Json rule(){return {{"rule_id","maximum"},{"claim_id","maximum"},{"subject","service"},{"predicate","maximum"},{"scope",{{"plan","basic"},{"version","fixture-1"}}},{"statement_kind","documented_fact"},{"dependencies",Json::array()},
 {"context",Json::array({Json{{"pointer","/plan"},{"value",{{"type","string"},{"value","basic"},{"unit",nullptr}}}}})},{"extractor",{{"kind","json_pointer"},{"pointer","/maximum"},{"type","integer"},{"unit","units"}}}};}
Json profile(){return {{"protocol","symphony.scv.structured-profile.v1"},{"profile_id","limits"},{"profile_version","fixture-1"},{"provider_id","independent"},{"source_id","docs"},{"locator_id","main"},{"media_types",Json::array({"application/json"})},{"authored_by","Mapping author"},{"rationale","Exact basic-plan maximum; default and exceptions are separate fields"},{"rules",Json::array({rule()})}};}
Json binding(const Json& c){return {{"profile_id","limits"},{"capture_digest",c.at("digest")}};}
Json fixture(Json c=cap()){return {{"fixture_id","basic-limit"},{"captures",Json::array({c})},{"bindings",Json::array({binding(c)})},{"selection_policy",policy()}};}
Json expected_claim(){return {{"claim_id","maximum"},{"subject","service"},{"predicate","maximum"},{"scope",{{"plan","basic"},{"version","fixture-1"}}},{"statement_kind","documented_fact"},{"value",value()},{"dependencies",Json::array()}};}
Json draft() {
 const auto extraction=Json{{"profile_id","limits"},{"capture_digest",cap().at("digest")},{"rule_id","maximum"},{"claim_id","maximum"},{"status","matched"},{"reasons",Json::array()}};
 const auto manifest=Json{{"fixture_id","basic-limit"},{"label","Independent expected maximum"},{"authored_by","Fixture author"},{"rationale","Expectation authored as 10 before extraction"},{"input_digest",nullptr},{"expected_claims",Json::array({expected_claim()})},{"expected_extractions",Json::array({extraction})}};
 return {{"protocol","symphony.scv.provider-pack.v1"},{"pack_id","independent-fixture"},{"pack_version","fixture-1"},{"authored_by","Pack author"},{"provenance",Json::array({"Independent synthetic fixture, not a vendor capability"})},
 {"provider",{{"provider_id","independent"},{"family_id","schv"},{"display_name","Independent fixture"},{"sources",Json::array({desired()})}}},{"profiles",Json::array()},{"structured_profiles",Json::array({profile()})},{"fixtures",Json::array({manifest})}};
}
Json prepare(Json p=draft(),Json f=Json::array({fixture()}),const std::string& d="scv"){return scv::handle_pack(req("provider_pack_prepare",{{"pack",p},{"fixtures",f}}),d);}
Json input(Json c=cap(),Json p=prepare(),Json f=Json::array()){return {{"pack",p},{"captures",Json::array({c})},{"bindings",Json::array({binding(c)})},{"selection_policy",policy()},{"fixtures",f}};}
Json eval(Json i=input(),const std::string& d="scv"){return scv::handle_pack(req("provider_pack_evaluate",i),d);}
void unresolved(const Json& result,const std::string& reason){require(result.at("knowledge").at("claims").empty(),"unresolved yields no claim");const auto& e=result.at("extractions")[0];require(e.at("status")=="unresolved","status unresolved");require(std::find(e.at("reasons").begin(),e.at("reasons").end(),reason)!=e.at("reasons").end(),"reason "+reason);}
void test_provider_pack_producer(){const auto i=input(cap(),prepare(),Json::array({fixture()}));const auto r=eval(i);require(r.at("input")==i&&scv::sealed(r)==r,"exact sealed input");require(r.at("conformance").at("passed")==1,"independently authored fixture passed");require(r.at("knowledge").at("claims")[0].at("value")==value(),"maximum value");require(scv::replay_pack(r,"schv",req("replay",{}))==r.at("knowledge"),"parent family replay");}
void test_detached_fixture_not_run(){auto p=prepare();require(!p.dump().contains("\"body\""),"sealed pack has no fixture body");const auto r=eval(input(cap(),p));require(r.at("conformance").at("not_run")==1&&r.at("fixture_results")[0].at("actual_claims").is_null(),"unselected fixture no proof");}
void test_fixture_failure_retained(){auto d=draft();d["fixtures"][0]["expected_claims"][0]["value"]["value"]=11;const auto p=prepare(d);const auto r=eval(input(cap(),p,Json::array({fixture()})));require(r.at("conformance").at("failed")==1&&r.at("fixture_results")[0].at("difference_axes").at("claims")==true,"independent mismatch remains evidence");}
void test_fixture_expected_unresolved() { const auto failed=cap("","failed"); auto d=draft(); d["fixtures"][0]["expected_claims"]=Json::array(); auto& e=d["fixtures"][0]["expected_extractions"][0]; e["capture_digest"]=failed.at("digest"); e["status"]="unresolved"; e["reasons"]=Json::array({"capture_failed"}); const auto f=fixture(failed); const auto p=prepare(d,Json::array({f})); const auto r=eval(input(cap(),p,Json::array({f}))); require(r.at("conformance").at("passed")==1 && r.at("knowledge").at("claims").size()==1,"expected unresolved fixture remains separate from production knowledge"); }
void test_fixture_digest_wrong_rejected(){auto f=fixture(cap("{\"plan\":\"basic\",\"maximum\":11}"));rejects([&]{eval(input(cap(),prepare(),Json::array({f})));});}
void test_precomputed_fixture_prepare(){auto p=prepare();p.erase("digest");require(prepare(p,Json::array())==prepare(),"portable precomputed manifest supported");}
void test_provider_pack_rejects_forged_evaluation(){auto r=eval();r["knowledge"]["claims"][0]["value"]["value"]=99;r["knowledge"]=scv::sealed(r["knowledge"]);r=scv::sealed(r);rejects([&]{static_cast<void>(scv::replay_pack(r,"scv",req("replay",{})));});r=eval();r["conformance"]["passed"]=1;r=scv::sealed(r);rejects([&]{static_cast<void>(scv::replay_pack(r,"scv",req("replay",{})));});}
void test_duplicate_keys_unresolved(){unresolved(eval(input(cap("{\"plan\":\"basic\",\"maximum\":10,\"maximum\":20}"))),"invalid_json");}
void test_invalid_json_unresolved(){unresolved(eval(input(cap("{\"maximum\":10,"))),"invalid_json");}
void test_missing_pointer_unresolved(){unresolved(eval(input(cap("{\"plan\":\"basic\",\"default\":10}"))),"pointer_missing");}
void test_context_exception_unresolved(){unresolved(eval(input(cap("{\"plan\":\"enterprise\",\"maximum\":10}"))),"context_value_differs:0");}
void test_reference_unresolved(){unresolved(eval(input(cap("{\"$ref\":\"#/other\",\"plan\":\"basic\",\"maximum\":10}"))),"reference_unresolved");}
void test_exact_decimal_lexeme(){auto d=draft();d["structured_profiles"][0]["rules"][0]["extractor"]["type"]="decimal";const auto r=eval(input(cap("{\"plan\":\"basic\",\"maximum\":9.007199254740993e15}"),prepare(d)));require(r.at("knowledge").at("claims")[0].at("value").at("value")=="9007199254740993","no float roundtrip");}
void test_decimal_type_not_string(){auto d=draft();d["structured_profiles"][0]["rules"][0]["extractor"]["type"]="decimal";unresolved(eval(input(cap("{\"plan\":\"basic\",\"maximum\":\"10\"}"),prepare(d))),"value_type_mismatch");}
void test_large_integer_unresolved(){unresolved(eval(input(cap("{\"plan\":\"basic\",\"maximum\":9007199254740992}"))),"value_type_mismatch");}
void test_pointer_escape_and_array_indices(){auto d=draft();d["structured_profiles"][0]["rules"][0]["extractor"]["pointer"]="/a~1b~0c/0";const auto p=prepare(d);const auto c=cap("{\"plan\":\"basic\",\"a/b~c\":[10]}");require(eval(input(c,p)).at("knowledge").at("claims").size()==1,"escaped pointer");d["structured_profiles"][0]["rules"][0]["extractor"]["pointer"]="/a~1b~0c/00";unresolved(eval(input(c,prepare(d))),"array_index_invalid");}
void test_partial_policy(){const auto c=cap("{\"plan\":\"basic\",\"maximum\":10}","partial");unresolved(eval(input(c)),"partial_capture_excluded");auto i=input(c);i["selection_policy"]=policy("include_qualified");const auto r=eval(i);require(r.at("extractions")[0].at("reasons")==Json::array({"partial_capture_qualified"}),"partial qualification remains");}
void test_failed_capture(){unresolved(eval(input(cap("","failed"))),"capture_failed");}
void test_source_authority_change(){auto s=desired();s["publisher"]="Different publisher";unresolved(eval(input(cap("{\"plan\":\"basic\",\"maximum\":10}","complete",s))),"source_declaration_differs");}
void test_recommendation_scope_and_units(){auto d=draft();d["structured_profiles"][0]["rules"][0]["statement_kind"]="recommendation";const auto r=eval(input(cap(),prepare(d)));const auto& c=r.at("knowledge").at("claims")[0];require(c.at("statement_kind")=="recommendation"&&c.at("scope")==rule().at("scope")&&c.at("value").at("unit")=="units","kind scope units preserved");}
void test_arbitrary_provider_and_wrong_family(){require(eval(input(),"schv").at("domain")=="schv","arbitrary provider through family");rejects([]{eval(input(),"scev");});rejects([]{eval(input(),"schv-gcp");});}
void test_profile_and_binding_identity_rejected(){auto d=draft();d["structured_profiles"].push_back(profile());rejects([&]{prepare(d);});auto i=input();i["bindings"].push_back(i["bindings"][0]);rejects([&]{eval(i);});}
void test_closed_fields_and_pointer_syntax(){auto d=draft();d["execution"]="never";rejects([&]{prepare(d);});d=draft();d["structured_profiles"][0]["rules"][0]["extractor"]["pointer"]="/bad~2";rejects([&]{prepare(d);});}
void test_text_profile_reuse(){auto d=draft();auto p=profile();p["protocol"]="symphony.scv.interpretation-profile.v1";p["rules"][0]["context"]=Json::array();p["rules"][0]["extractor"]={{"kind","literal"},{"quote","\"maximum\":10"},{"value",value()}};d["structured_profiles"]=Json::array();d["profiles"]=Json::array({scv::sealed(p)});require(eval(input(cap(),prepare(d))).at("knowledge").at("claims").size()==1,"existing text profile replay reused");}
}
int main(int argc,char** argv){
 std::vector<std::pair<std::string,std::function<void()>>> tests={
#define CASE(x) {#x,x}
 CASE(test_provider_pack_producer),CASE(test_detached_fixture_not_run),CASE(test_fixture_failure_retained),CASE(test_fixture_digest_wrong_rejected),CASE(test_fixture_expected_unresolved),CASE(test_precomputed_fixture_prepare),CASE(test_provider_pack_rejects_forged_evaluation),CASE(test_duplicate_keys_unresolved),CASE(test_invalid_json_unresolved),CASE(test_missing_pointer_unresolved),CASE(test_context_exception_unresolved),CASE(test_reference_unresolved),CASE(test_exact_decimal_lexeme),CASE(test_decimal_type_not_string),CASE(test_large_integer_unresolved),CASE(test_pointer_escape_and_array_indices),CASE(test_partial_policy),CASE(test_failed_capture),CASE(test_source_authority_change),CASE(test_recommendation_scope_and_units),CASE(test_arbitrary_provider_and_wrong_family),CASE(test_profile_and_binding_identity_rejected),CASE(test_closed_fields_and_pointer_syntax),CASE(test_text_profile_reuse)
#undef CASE
 };
 try{for(const auto& [name,f]:tests){f();std::cout<<"PASS "<<name<<'\n';}if(argc==2){const auto i=input(cap(),prepare(),Json::array({fixture()}));std::ofstream out(argv[1]);out<<Json{{"prepare_input",{{"pack",draft()},{"fixtures",Json::array({fixture()})}}},{"prepare_result",prepare()},{"input",i},{"result",eval(i)}}.dump(2)<<'\n';}}catch(const std::exception& e){std::cerr<<"FAIL "<<e.what()<<'\n';return 1;}return 0;
}
