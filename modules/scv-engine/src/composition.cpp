#include "composition.hpp"
#include "interpretation.hpp"
#include "pack.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <algorithm>
#include <functional>
#include <map>
#include <set>
#include <string_view>

namespace symphony::knowledge::scv {
namespace {
using Map = std::map<std::string, Json>;
[[noreturn]] void invalid(const std::string& why) { throw engine::Error("composition.invalid", why, 4); }
void deadline(const engine::Request& r) { if (engine::unix_time_ms() > r.deadline_unix_ms) throw engine::Error("request.deadline_exceeded", "composition deadline exceeded", 4); }
void fields(const Json& v, std::initializer_list<std::string_view> keys) {
    if (!v.is_object() || v.size() != keys.size()) invalid("unexpected composition fields");
    for (const auto key : keys) if (!v.contains(std::string(key))) invalid("missing composition field");
}
std::string text(const Json& v, const std::string& key, std::size_t bound = 512) {
    if (!v.is_object() || !v.contains(key) || !v.at(key).is_string()) invalid("expected composition text: " + key);
    const auto s=v.at(key).get<std::string>();
    if (s.empty() || s.size()>bound) invalid("composition text outside bounds");
    for (unsigned char c : s) if (c < 32 || c == 127) invalid("composition identity/control text");
    return s;
}
void array(const Json& v, std::size_t max, std::size_t min=0) { if (!v.is_array() || v.size()<min || v.size()>max) invalid("composition array outside bounds"); }
std::set<std::string> strings(const Json& v, std::size_t max, std::size_t min=0) {
    array(v,max,min); std::set<std::string> out;
    for (const auto& x:v) if (!out.insert(text(Json{{"v",x}},"v")).second) invalid("duplicate composition string");
    return out;
}
void resolution(const Json& v) {
    fields(v,{"kind","reference","description"}); const auto kind=text(v,"kind");
    if (kind!="source" && kind!="observation" && kind!="adapter" && kind!="caller_decision") invalid("unknown resolution kind");
    text(v,"reference"); text(v,"description",2048);
}
Map requirements(const Json& values) {
    array(values,32,1); Map out;
    for (const auto& v:values) {
        fields(v,{"requirement_id","importance","operator","right","resolution"}); resolution(v.at("resolution"));
        if (!out.emplace(text(v,"requirement_id"),v).second) invalid("duplicate common requirement");
    }
    return out;
}
Json check(const Json& requirement, const Json& claim) {
    return {{"check_id",requirement.at("requirement_id")},{"importance",requirement.at("importance")},{"operator",requirement.at("operator")},{"right",requirement.at("right")},{"left",claim}};
}
Json connection(const std::string& id, const Json& checks) {
    return {{"connection_id",id},{"from_subject","caller-composition"},{"to_subject","caller-requirements"},{"checks",checks}};
}
std::string aggregate(const std::set<std::string>& statuses) {
    if (statuses.empty()) return "unresolved";
    for (const auto* s : {"contradicted","unresolved","conditional"}) if (statuses.contains(s)) return s;
    return "satisfied";
}
Json obligation(const std::string& kind, Json requirement, Json slot, Json recipe, Json claims, const Json& how, const std::string& detail) {
    return {{"kind",kind},{"requirement_id",requirement},{"slot_id",slot},{"recipe_id",recipe},{"claim_ids",claims},{"resolution",how},{"detail",detail}};
}
Json evidence_input(const Json& input, const std::string& domain, const engine::Request& request) {
    auto extra=input.at("additional_knowledge"); array(extra,16); array(input.at("interpretations"),16);array(input.at("provider_packs"),16);
    std::set<std::string> packs;
    for (const auto& p:input.at("provider_packs")) {
        deadline(request); if (!packs.insert(text(p,"digest")).second) invalid("duplicate provider pack evaluation");
        extra.push_back(replay_pack(p,domain,request));
    }
    return {{"interpretations",input.at("interpretations")},{"additional_knowledge",extra},{"query_time",input.at("query_time")},{"connections",Json::array()}};
}
struct Choice { std::string slot; Json recipe; };
using Choices=std::vector<Choice>;
Json candidate(const Choices& selected, const Map& reqs, const Json& assessment, const Json& evidence_obligations) {
    Json choices=Json::array(),checks=Json::array(),connections=Json::array(),unbound=Json::array(),obligations=Json::array();
    std::map<std::string,std::vector<Json>> bindings; std::set<std::string> supplied,required_statuses,claim_ids;
    bool unknown=false,unimplemented=false;
    for (const auto& c:selected) {
        choices.push_back({{"slot_id",c.slot},{"recipe_id",c.recipe.at("recipe_id")},{"provider_id",c.recipe.at("provider_id")}});
        for (const auto& b:c.recipe.at("bindings")) {
            bindings[text(b,"requirement_id")].push_back(b.at("claim"));
            claim_ids.insert(b.at("claim").at("claim_id").get<std::string>());
        }
        for (const auto& x:strings(c.recipe.at("supplies_interfaces"),16)) supplied.insert(x);
        const auto state=text(c.recipe.at("implementation"),"status"); unknown|=state=="unknown";unimplemented|=state=="unimplemented";
        obligations.push_back(obligation(state=="available"?"verify_implementation":"implementation_gap",nullptr,c.slot,c.recipe.at("recipe_id"),Json::array(),c.recipe.at("resolution"),"implementation status/reference is caller declared; verify exact adapter behavior and runtime installation"));
        connections.push_back(connection("prerequisites:"+c.slot,c.recipe.at("prerequisites")));
    }
    for (const auto& [id,r]:reqs) {
        // References remain dependencies even when no unambiguous comparison
        // can be made. They also link conformance obligations to the candidate.
        if (r.at("right").at("kind")=="claim") claim_ids.insert(r.at("right").at("claim_id").get<std::string>());
        const auto& options=bindings[id];
        if (options.size()!=1) {
            const auto reason=options.empty()?"missing_binding":"ambiguous_binding";
            unbound.push_back({{"requirement_id",id},{"reason",reason},{"claim_references",options}});
            Json ids=Json::array();for(const auto& ref:options)ids.push_back(ref.at("claim_id"));
            obligations.push_back(obligation(reason,id,nullptr,nullptr,ids,r.at("resolution"),"supply one unambiguous scoped claim binding for this common requirement"));
            if(r.at("importance")=="required")required_statuses.insert("unresolved");
        } else checks.push_back(check(r,options.front()));
    }
    connections.push_back(connection("requirements",checks));
    const auto evaluated=evaluate_connections(connections,assessment);
    for(const auto& c:evaluated)for(const auto& item:c.at("checks")) {
        const auto& spec=item.at("specification");const auto status=text(item,"status");
        for(const auto& id:item.at("claim_ids"))claim_ids.insert(id.get<std::string>());
        if(spec.at("importance")=="required")required_statuses.insert(status);
        if(status=="satisfied")continue;
        const auto cid=text(c,"connection_id"),id=text(spec,"check_id");
        Json how,slot=nullptr,recipe=nullptr,req=nullptr;
        if(cid=="requirements") {how=reqs.at(id).at("resolution");req=id;}
        else {const auto sid=cid.substr(std::string("prerequisites:").size());for(const auto& choice:selected)if(choice.slot==sid){how=choice.recipe.at("resolution");slot=sid;recipe=choice.recipe.at("recipe_id");}}
        obligations.push_back(obligation("check_"+status,req,slot,recipe,item.at("claim_ids"),how,"inspect the native check and exact evidence qualification; a contradiction is scoped to this requirement and selected evidence"));
    }
    Json interfaces=Json::array();
    for(const auto& c:selected)for(const auto& id:strings(c.recipe.at("requires_interfaces"),16)) {
        const bool matched=supplied.contains(id);interfaces.push_back({{"slot_id",c.slot},{"recipe_id",c.recipe.at("recipe_id")},{"interface_ref",id},{"declared_match",matched}});
        if(!matched){required_statuses.insert("unresolved");obligations.push_back(obligation("missing_interface",nullptr,c.slot,c.recipe.at("recipe_id"),Json::array(),c.recipe.at("resolution"),"no selected recipe supplies exact interface contract: "+id));}
    }
    std::sort(obligations.begin(),obligations.end(),[](const Json&a,const Json&b){return a.dump()<b.dump();});
    Json evidence_refs=Json::array();
    for(const auto& issue:evidence_obligations){bool relevant=false;for(const auto& id:issue.at("claim_ids"))relevant|=claim_ids.contains(id.get<std::string>());if(relevant)evidence_refs.push_back({{"pack_evaluation_digest",issue.at("pack_evaluation_digest")},{"fixture_id",issue.at("fixture_id")},{"status",issue.at("status")}});}
    return {{"candidate_id",sealed(Json{{"choices",choices}}).at("digest")},{"choices",choices},{"status",aggregate(required_statuses)},
        {"implementation_status",unimplemented?"unimplemented":unknown?"unknown":"available"},{"connections",evaluated},{"unbound_requirements",unbound},
        {"interfaces",interfaces},{"claim_ids",claim_ids},{"obligations",obligations},{"evidence_obligation_refs",evidence_refs}};
}
Json explore(const Json& input,const std::string& domain,const engine::Request& request) {
    fields(input,{"interpretations","additional_knowledge","provider_packs","query_time","requirements","slots","allowed_guarantee_changes","counterfactuals","bounds"});
    const auto baseline=requirements(input.at("requirements"));const auto permitted=strings(input.at("allowed_guarantee_changes"),32);array(input.at("slots"),4,1);array(input.at("counterfactuals"),4);
    fields(input.at("bounds"),{"max_candidates"});const auto& raw_max=input.at("bounds").at("max_candidates");
    if(!raw_max.is_number_integer()||raw_max<1||raw_max>32)invalid("max_candidates outside 1..32");const auto max=raw_max.get<std::size_t>();
    Map slots,scenarios;scenarios.emplace("baseline",input.at("requirements"));
    for(const auto& c:input.at("counterfactuals")){fields(c,{"counterfactual_id","requirements"});const auto id=text(c,"counterfactual_id");auto rs=requirements(c.at("requirements"));if(rs.size()!=baseline.size())invalid("counterfactual must preserve requirement identities");for(const auto& [rid,r]:baseline){static_cast<void>(r);if(!rs.contains(rid))invalid("counterfactual changes requirement identity");}if(!scenarios.emplace(id,c.at("requirements")).second)invalid("duplicate counterfactual identity");}
    Json excluded=Json::array();std::size_t requested_space=1,eligible_space=1;std::map<std::string,std::vector<Json>> eligible;
    for(const auto& s:input.at("slots")) {
        fields(s,{"slot_id","allowed_provider_ids","recipes"});const auto sid=text(s,"slot_id",128);if(!slots.emplace(sid,s).second)invalid("duplicate slot identity");const auto providers=strings(s.at("allowed_provider_ids"),16,1);array(s.at("recipes"),8,1);requested_space*=s.at("recipes").size();Map recipes;
        for(const auto& r:s.at("recipes")) {
            fields(r,{"recipe_id","provider_id","bindings","prerequisites","requires_interfaces","supplies_interfaces","guarantee_changes","implementation","resolution"});const auto rid=text(r,"recipe_id");if(!recipes.emplace(rid,r).second)invalid("duplicate recipe identity in slot");text(r,"provider_id");resolution(r.at("resolution"));array(r.at("bindings"),32);array(r.at("prerequisites"),16);for(const auto& prerequisite:r.at("prerequisites"))text(prerequisite,"check_id");strings(r.at("requires_interfaces"),16);strings(r.at("supplies_interfaces"),16);const auto changes=strings(r.at("guarantee_changes"),16);
            fields(r.at("implementation"),{"status","reference"});const auto status=text(r.at("implementation"),"status");if(status!="available"&&status!="unimplemented"&&status!="unknown")invalid("invalid implementation status");if(!r.at("implementation").at("reference").is_null())text(r.at("implementation"),"reference");if(status=="available"&&r.at("implementation").at("reference").is_null())invalid("declared available implementation requires reference");
            std::set<std::string> seen;for(const auto& b:r.at("bindings")){fields(b,{"requirement_id","claim"});const auto id=text(b,"requirement_id");if(!baseline.contains(id)||!seen.insert(id).second)invalid("unknown or duplicate recipe requirement binding");}
            Json reasons=Json::array();if(!providers.contains(text(r,"provider_id")))reasons.push_back("provider_not_selected");for(const auto& id:changes)if(!permitted.contains(id))reasons.push_back("guarantee_change_not_permitted:"+id);
            if(!reasons.empty())excluded.push_back({{"slot_id",sid},{"recipe_id",rid},{"reasons",reasons}});else eligible[sid].push_back(r);
        }
        std::sort(eligible[sid].begin(),eligible[sid].end(),[](const Json&a,const Json&b){return a.at("recipe_id")<b.at("recipe_id");});eligible_space*=eligible[sid].size();
    }
    auto r=request;r.operation="connection_evaluate";r.payload=evidence_input(input,domain,request);const auto evidence=handle_interpretation(r,domain);const auto& assessment=evidence.at("graph_evaluation");
    Json evidence_obligations=Json::array();
    for(const auto& pack:input.at("provider_packs")){std::set<std::string> ids;for(const auto& claim:pack.at("knowledge").at("claims"))ids.insert(claim.at("claim_id").get<std::string>());for(const auto& fixture:pack.at("fixture_results"))if(fixture.at("status")!="passed")evidence_obligations.push_back({{"pack_evaluation_digest",pack.at("digest")},{"fixture_id",fixture.at("fixture_id")},{"status",fixture.at("status")},{"claim_ids",ids},{"detail","review the retained conformance fixture and authored mapping; failure or absence is not provider impossibility"}});}
    std::sort(evidence_obligations.begin(),evidence_obligations.end(),[](const Json&a,const Json&b){return a.dump()<b.dump();});
    // Validate every authored requirement, binding and prerequisite, including
    // excluded recipes and counterfactuals, before claiming a bounded search.
    for(const auto& [sid,s]:slots)for(const auto& recipe:s.at("recipes")) {
        Json checks=Json::array();for(const auto& b:recipe.at("bindings"))checks.push_back(check(baseline.at(text(b,"requirement_id")),b.at("claim")));
        static_cast<void>(evaluate_connections(Json::array({connection("bindings",checks),connection("prerequisites",recipe.at("prerequisites"))}),assessment));static_cast<void>(sid);
    }
    const Json dummy={{"claim_id","unbound"},{"subject","unbound"},{"scope",Json::object()}};
    for(const auto& [id,rs]:scenarios){Json checks=Json::array();for(const auto& [rid,rq]:requirements(rs)){static_cast<void>(rid);checks.push_back(check(rq,dummy));}static_cast<void>(evaluate_connections(Json::array({connection(id,checks)}),assessment));}
    std::sort(excluded.begin(),excluded.end(),[](const Json&a,const Json&b){return std::pair{a.at("slot_id"),a.at("recipe_id")}<std::pair{b.at("slot_id"),b.at("recipe_id")};});
    std::vector<std::string> keys;for(const auto& [key,s]:slots){static_cast<void>(s);keys.push_back(key);}std::vector<Choices> products;Choices path;
    std::function<void(std::size_t)> walk=[&](std::size_t depth){deadline(request);if(products.size()>=max)return;if(depth==keys.size()){products.push_back(path);return;}for(const auto& recipe:eligible.at(keys[depth])){path.push_back({keys[depth],recipe});walk(depth+1);path.pop_back();if(products.size()>=max)return;}};walk(0);
    Json evaluations=Json::array();for(const auto& [id,rs]:scenarios){Json candidates=Json::array(),changed=Json::array();const auto requirements_map=requirements(rs);for(const auto& [rid,rq]:requirements_map)if(rq!=baseline.at(rid))changed.push_back(rid);for(const auto& choices:products){deadline(request);candidates.push_back(candidate(choices,requirements_map,assessment,evidence_obligations));}evaluations.push_back({{"scenario_id",id},{"changed_requirement_ids",changed},{"candidates",candidates}});}
    return sealed(Json{{"protocol","symphony.scv.composition-exploration.v1"},{"domain",domain},{"input",input},{"evidence_evaluation",evidence},{"evidence_obligations",evidence_obligations},
        {"search",{{"requested_combinations",requested_space},{"eligible_combinations",eligible_space},{"evaluated_candidates",products.size()},{"exhaustive",products.size()==eligible_space},{"stop_reason",eligible_space==0?"no_eligible_recipe":products.size()<eligible_space?"candidate_limit":"exhausted"},{"excluded_recipes",excluded}}},
        {"scenarios",evaluations},{"limitations",Json::array({"finite caller-declared recipe products only; no result establishes global impossibility or selects a provider",
            "recipe-to-requirement bindings and interface/guarantee declarations are authored mappings, not proof that the recipe realizes those semantics",
            "status aggregates selected documented checks and missing interface bindings; implementation availability is caller declared and requires independent verification",
            "exact interface references describe declared contracts, not deployed routing, permissions, payload delivery or resource guarantees",
            "counterfactuals change only caller-supplied requirements over the same evidence and recipe selections; no relaxation is applied and no minimality is asserted",
            "resolution descriptors are opaque caller-authored instructions for review; no source acquisition, probe, credential or deployment is executed"})}});
}
Map candidates(const Json& result){Map out;for(const auto& s:result.at("scenarios"))for(const auto& c:s.at("candidates"))out.emplace(Json::array({s.at("scenario_id"),c.at("candidate_id")}).dump(),c);return out;}
Map finding_map(const Json& result){Map out;for(const auto& f:result.at("evidence_evaluation").at("graph_evaluation").at("findings"))out.emplace(f.at("claim").at("claim_id").get<std::string>(),f);return out;}
void reference_closure(const Map& findings,const std::string& id,std::set<std::string>& visited) {
    if(!visited.insert(id).second||!findings.contains(id))return;
    const auto& claim=findings.at(id).at("claim");
    const auto visit=[&](const Json& dependencies){for(const auto& d:dependencies)reference_closure(findings,d.at("claim_id").get<std::string>(),visited);};
    visit(claim.at("dependencies"));for(const auto& support:claim.at("alternative_supports"))visit(support.at("dependencies"));
}
Json recipes(const Json& input,bool selection){Json out=Json::array();Map ordered;for(const auto& s:input.at("slots"))ordered.emplace(text(s,"slot_id"),s);for(const auto& [id,s]:ordered)out.push_back({{"slot_id",id},{selection?"allowed_provider_ids":"recipes",s.at(selection?"allowed_provider_ids":"recipes")}});return out;}
Json reassess(const Json& input,const std::string& domain,const engine::Request& request){
    fields(input,{"before","after"});const auto& before=input.at("before");const auto& after=input.at("after");
    for(const auto* x:{&before,&after}){if(!x->is_object()||!x->contains("input")||explore(x->at("input"),domain,request)!=*x)invalid("composition result cannot be exactly replayed");}
    const auto old=candidates(before),current=candidates(after),old_findings=finding_map(before),new_findings=finding_map(after);std::set<std::string> keys;for(const auto& [key,v]:old){static_cast<void>(v);keys.insert(key);}for(const auto& [key,v]:current){static_cast<void>(v);keys.insert(key);}Json changes=Json::array();
    for(const auto& key:keys){const auto a=old.contains(key)?old.at(key):Json(nullptr),b=current.contains(key)?current.at(key):Json(nullptr);bool affected=a!=b;std::set<std::string> refs;for(const auto* c:{&a,&b})if(!c->is_null())for(const auto& id:c->at("claim_ids")){std::set<std::string> previous,current;reference_closure(old_findings,id.get<std::string>(),previous);reference_closure(new_findings,id.get<std::string>(),current);refs.insert(previous.begin(),previous.end());refs.insert(current.begin(),current.end());}for(const auto& id:refs)if((old_findings.contains(id)?old_findings.at(id):Json(nullptr))!=(new_findings.contains(id)?new_findings.at(id):Json(nullptr)))affected=true;const auto id=Json::parse(key);changes.push_back({{"scenario_id",id.at(0)},{"candidate_id",id.at(1)},{"before_status",a.is_null()?Json(nullptr):a.at("status")},{"after_status",b.is_null()?Json(nullptr):b.at("status")},{"changed",a!=b},{"affected",affected}});}
    const auto& a=before.at("input");const auto& b=after.at("input");bool evidence=false;for(const auto* key:{"interpretations","additional_knowledge","provider_packs"})evidence|=a.at(key)!=b.at(key);
    return sealed(Json{{"protocol","symphony.scv.composition-reassessment.v1"},{"domain",domain},{"input",input},{"before_digest",before.at("digest")},{"after_digest",after.at("digest")},
        {"change_axes",{{"evidence",evidence},{"policy",before.at("evidence_evaluation").at("graph_evaluation").at("selection_policy")!=after.at("evidence_evaluation").at("graph_evaluation").at("selection_policy")},{"requirements",a.at("requirements")!=b.at("requirements")||a.at("counterfactuals")!=b.at("counterfactuals")},{"query_time",a.at("query_time")!=b.at("query_time")},{"recipes",recipes(a,false)!=recipes(b,false)},{"selections",recipes(a,true)!=recipes(b,true)||a.at("allowed_guarantee_changes")!=b.at("allowed_guarantee_changes")},{"bounds",a.at("bounds")!=b.at("bounds")}}},
        {"candidates",changes},{"search_changed",before.at("search")!=after.at("search")},{"limitations",Json::array({"axes identify exact input differences and may overlap; they do not prove counterfactual causation",
            "affected candidates include referenced evidence changes; absent candidates may reflect search bounds or caller selections rather than impossibility"})}});
}
}
Json handle_composition(const engine::Request& request,const std::string& domain){deadline(request);Json result;if(request.operation=="composition_explore")result=explore(request.payload,domain,request);else if(request.operation=="composition_reassess")result=reassess(request.payload,domain,request);else throw engine::Error("operation.unsupported","unsupported composition operation",4);deadline(request);return result;}
}
