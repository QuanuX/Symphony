#include "interpretation.hpp"
#include "composition.hpp"
#include "knowledge.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/error.hpp"

#include <functional>
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
void test_composition_exploration_producer(){const auto r=compose();require(r.at("search").at("requested_combinations")==4&&r.at("search").at("evaluated_candidates")==4,"actual Cartesian product");require(first(r).at("status")=="satisfied","documented requirement matches");require(first(r).at("obligations").size()==2,"declared adapters still need verification");require(compose()==r,"deterministic replay");}
void test_composition_consumer_rejects_resealed_result(){const auto a=compose();auto b=a;b["search"]["eligible_combinations"]=9;b=scv::sealed(b);rejects([&]{static_cast<void>(reconsider(a,b));});}
void test_composition_partial_search_is_explicit(){auto i=composition_input();i["bounds"]["max_candidates"]=1;const auto r=compose(i);require(r.at("search").at("stop_reason")=="candidate_limit"&&r.at("search").at("eligible_combinations")==4,"partial search exposes full finite space");}
void test_composition_provider_and_guarantee_permissions(){auto i=composition_input();i["slots"][0]["allowed_provider_ids"]=Json::array({"aws"});require(compose(i).at("search").at("stop_reason")=="no_eligible_recipe","provider pin never relaxed");i=composition_input();i["slots"][0]["recipes"][0]["guarantee_changes"]=Json::array({"checkpoint-resume"});require(compose(i).at("search").at("eligible_combinations")==2,"unpermitted transformation excluded");i["allowed_guarantee_changes"]=Json::array({"checkpoint-resume"});require(compose(i).at("search").at("eligible_combinations")==4,"explicit permission restores recipe");}
void test_composition_duplicate_and_missing_requirement_bindings(){auto i=composition_input();i["requirements"].push_back(requirement());rejects([&]{static_cast<void>(compose(i));});i=composition_input();i["slots"][0]["recipes"][0]["bindings"]=Json::array();auto r=compose(i);require(first(r).at("status")=="unresolved"&&first(r).at("unbound_requirements")[0].at("reason")=="missing_binding","required missing binding unresolved");i=composition_input();i["slots"][1]["recipes"][0]["bindings"]=i["slots"][0]["recipes"][0]["bindings"];require(first(compose(i)).at("unbound_requirements")[0].at("reason")=="ambiguous_binding","ambiguous cross-slot binding never picks winner");}
void test_composition_implementation_and_interface_gaps(){auto i=composition_input();i["slots"][0]["recipes"][0]["implementation"]={{"status","unimplemented"},{"reference",nullptr}};i["slots"][1]["recipes"][0]["supplies_interfaces"]=Json::array();const auto r=compose(i);require(first(r).at("implementation_status")=="unimplemented"&&first(r).at("status")=="unresolved","implementation and missing interface distinct");}
void test_composition_counterfactual_pins_evidence_and_candidates(){auto i=composition_input();i["counterfactuals"]=Json::array({Json{{"counterfactual_id","longer"},{"requirements",Json::array({requirement(12)})}}});const auto r=compose(i);const auto& a=r.at("scenarios")[0].at("candidates")[0];const auto& b=r.at("scenarios")[1].at("candidates")[0];require(a.at("candidate_id")==b.at("candidate_id")&&a.at("status")=="satisfied"&&b.at("status")=="contradicted","same candidate different requirement");require(r.at("input").at("interpretations")==i.at("interpretations"),"evidence pinned");}
void test_composition_time_only_reassessment(){auto i=composition_input();i["interpretations"]=Json::array({interpret(capture(),profile(),policy(1))});const auto a=compose(i);i["query_time"]="2026-09-10T00:00:02Z";const auto b=compose(i),d=reconsider(a,b);require(first(b).at("status")=="unresolved","expiry unknown not impossible");require(d.at("change_axes").at("query_time")==true&&d.at("change_axes").at("evidence")==false,"time-only attribution");require(d.at("candidates")[0].at("affected")==true,"affected expiry");}
void test_composition_invalid_excluded_recipe_still_rejected(){auto i=composition_input();i["slots"][0]["allowed_provider_ids"]=Json::array({"aws"});i["slots"][0]["recipes"][0]["bindings"][0]["claim"]["scope"]=Json::array();rejects([&]{static_cast<void>(compose(i));});}
void test_composition_unknown_claim_and_prerequisite_gap(){auto i=composition_input();i["slots"][0]["recipes"][0]["bindings"][0]["claim"]["claim_id"]="unknown";require(first(compose(i)).at("status")=="unresolved","unknown evidence remains unresolved");i=composition_input();auto r=requirement();Json prerequisite={{"check_id","independent"},{"importance","required"},{"operator","gte"},{"right",r.at("right")},{"left",i["slots"][0]["recipes"][0]["bindings"][0]["claim"]}};i["slots"][0]["recipes"][0]["bindings"]=Json::array();i["slots"][0]["recipes"][0]["prerequisites"]=Json::array({prerequisite});require(first(compose(i)).at("status")=="unresolved","satisfied prerequisite cannot hide missing requirement");}
void test_composition_unbound_references_remain_change_dependencies(){
    for (const bool ambiguous : {true,false}) {
        auto i=composition_input();
        i["interpretations"]=Json::array({interpret(capture(),profile(),policy(1))});
        if (ambiguous) i["slots"][1]["recipes"][0]["bindings"]=i["slots"][0]["recipes"][0]["bindings"];
        else {
            auto ref=i["slots"][0]["recipes"][0]["bindings"][0]["claim"];ref["kind"]="claim";
            i["requirements"][0]["right"]=ref;i["slots"][0]["recipes"][0]["bindings"]=Json::array();
        }
        const auto a=compose(i);i["query_time"]="2026-09-10T00:00:02Z";const auto b=compose(i),d=reconsider(a,b);
        require(first(a).at("status")=="unresolved"&&first(a).at("claim_ids")==Json::array({"limit"}),"unbound authored references retained without credit");
        bool found=false;for(const auto& c:d.at("candidates"))if(c.at("candidate_id")==first(a).at("candidate_id")){
            found=true;require(c.at("changed")==false&&c.at("affected")==true,"unbound reference evidence change remains visible");
        }
        require(found,"reassessment candidate present");
    }
    auto i=composition_input();i["interpretations"]=Json::array();
    Json limit={{"claim_id","limit"},{"subject","fixture-service"},{"predicate","maximum"},{"scope",{{"plan","fixture"}}},{"statement_kind","user_assertion"},{"value",value(10)},{"evidence",Json::array()},{"dependencies",Json::array({Json{{"claim_id","premise"},{"role","requires"}}})}};
    auto premise=limit;premise["claim_id"]="premise";premise["predicate"]="premise";premise["dependencies"]=Json::array();
    const auto knowledge=[&](){return scv::handle_knowledge(request("knowledge_interpret",{{"captures",Json::array()},{"claims",Json::array({limit,premise})},{"interpreter_version","composition-dependency-fixture"},{"selection_policy",policy()}}),"scv");};
    i["additional_knowledge"]=Json::array({knowledge()});const auto a=compose(i);
    premise["value"]=value(11);i["additional_knowledge"]=Json::array({knowledge()});const auto b=compose(i),d=reconsider(a,b);
    require(first(a)==first(b),"same candidate direct finding remains qualified");
    for(const auto& c:d.at("candidates"))require(c.at("changed")==false&&c.at("affected")==true,"transitive dependency finding changes remain visible");
}
void test_composition_bounds_and_deadline(){auto i=composition_input();i["bounds"]["max_candidates"]=0;rejects([&]{static_cast<void>(compose(i));});auto r=request("composition_explore",composition_input());r.deadline_unix_ms=0;rejects([&]{static_cast<void>(scv::handle_composition(r,"scv"));});}
}
int main(){try{test_composition_exploration_producer();test_composition_consumer_rejects_resealed_result();test_composition_partial_search_is_explicit();test_composition_provider_and_guarantee_permissions();test_composition_duplicate_and_missing_requirement_bindings();test_composition_implementation_and_interface_gaps();test_composition_counterfactual_pins_evidence_and_candidates();test_composition_time_only_reassessment();test_composition_invalid_excluded_recipe_still_rejected();test_composition_bounds_and_deadline();test_composition_unknown_claim_and_prerequisite_gap();test_composition_unbound_references_remain_change_dependencies();std::cout<<"12 composition cases passed\n";return 0;}catch(const std::exception& e){std::cerr<<e.what()<<'\n';return 1;}}
