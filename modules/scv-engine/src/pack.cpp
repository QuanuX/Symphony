#include "pack.hpp"
#include "interpretation.hpp"
#include "knowledge.hpp"
#include "scv.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include <algorithm>
#include <charconv>
#include <cstdint>
#include <map>
#include <optional>
#include <regex>
#include <set>
#include <string_view>
#include <vector>

namespace symphony::knowledge::scv {
namespace {
using Map = std::map<std::string, Json>;
[[noreturn]] void bad(const std::string& message) { throw engine::Error("pack.invalid", message, 4); }
void check_time(const engine::Request& r) { if (engine::unix_time_ms() > r.deadline_unix_ms) throw engine::Error("request.deadline_exceeded", "pack deadline exceeded", 4); }
void fields(const Json& j, std::initializer_list<std::string_view> keys) {
    if (!j.is_object() || j.size() != keys.size()) bad("unexpected pack fields");
    for (const auto key : keys) if (!j.contains(std::string(key))) bad("missing pack field: " + std::string(key));
}
std::string str(const Json& j, const std::string& key, std::size_t bound = 512, bool empty = false) {
    if (!j.contains(key) || !j.at(key).is_string()) bad("expected pack text: " + key);
    auto s = j.at(key).get<std::string>();
    if ((!empty && s.empty()) || s.size() > bound || s.find('\0') != std::string::npos) bad("pack text outside bounds: " + key);
    return s;
}
void arr(const Json& j, std::size_t bound) { if (!j.is_array() || j.size() > bound) bad("pack array outside bounds"); }
Json sorted(Json j) { std::sort(j.begin(), j.end(), [](const Json& a, const Json& b) { return a.dump() < b.dump(); }); return j; }
Json vals(const Map& m) { auto j = Json::array(); for (const auto& [k,v] : m) { static_cast<void>(k); j.push_back(v); } return j; }
void seal(const Json& j) { if (str(j, "digest", 71).size() != 71 || sealed(j) != j) bad("pack artifact seal mismatch"); }
Json call(const engine::Request& r, const std::string& op, Json p, const std::string& d) {
    auto q = r; q.operation = op; q.payload = std::move(p); check_time(q);
    if (op == "provider_onboard") return handle_source(q,d);
    if (op == "profile_prepare" || op == "provider_interpret") return handle_interpretation(q,d);
    return handle_knowledge(q,d);
}
Json validation_policy() { return {{"policy_id","pack-structure-validation"},{"max_age_seconds",nullptr},{"partial_capture","exclude"},{"allowed_statement_kinds",Json::array({"documented_fact","requirement","recommendation","observation","user_assertion","inference","hypothesis"})}}; }
Json claim(const Json& rule, const Json& value) {
    Json j = {{"value",value},{"evidence",Json::array()}};
    for (const auto* k : {"claim_id","subject","predicate","scope","statement_kind","dependencies"}) j[k] = rule.at(k);
    return j;
}
Json projection(const Json& j) { Json out = Json::object(); for (const auto* key : {"claim_id","subject","predicate","scope","statement_kind","value","dependencies"}) out[key] = j.at(key); return out; }
Json project_claims(const Json& claims) { auto a = Json::array(); for (auto c : claims) a.push_back(projection(std::move(c))); return sorted(a); }
// Validate exact value/claim semantics using the existing authoritative owner.
void validate_claims(const Json& claims, const engine::Request& r, const std::string& d) {
    static_cast<void>(call(r,"knowledge_interpret",{{"captures",Json::array()},{"claims",claims},{"interpreter_version","pack-validation-v1"},{"selection_policy",validation_policy()}},d));
}
Json placeholder(const Json& x) {
    const auto t = str(x,"type"); Json v;
    if (t == "integer") v = 0;
    else if (t == "decimal") v = "0";
    else if (t == "boolean") v = false;
    else if (t == "string" || t == "reference") v = "placeholder";
    else bad("unsupported structured value type");
    return {{"type",t},{"value",v},{"unit",x.at("unit")}};
}
std::vector<std::string> pointer(const std::string& s) {
    if (s.size() > 1024 || s.find('\0') != std::string::npos || (!s.empty() && s.front() != '/')) bad("invalid JSON pointer");
    std::vector<std::string> out;
    if (s.empty()) return out;
    std::size_t start = 1;
    for (;;) {
        const auto stop = s.find('/',start); const auto raw = s.substr(start, stop == std::string::npos ? stop : stop-start); std::string part;
        for (std::size_t i=0;i<raw.size();++i) {
            if (raw[i] != '~') part += raw[i];
            else { if (++i == raw.size() || (raw[i] != '0' && raw[i] != '1')) bad("invalid JSON pointer escape"); part += raw[i]=='0'?'~':'/'; }
        }
        out.push_back(part); if (out.size()>32) bad("JSON pointer depth exceeds bound");
        if (stop==std::string::npos) break; start=stop+1;
    }
    return out;
}
std::string escaped(const std::string& s) { std::string o; for (char c : s) { if(c=='~')o+="~0"; else if(c=='/')o+="~1"; else o+=c; } return o; }
// A bounded document parser preserves numeric lexemes and byte anchors. It does
// not feed document floats through the integer-only administrative JSON wire.
struct Node { std::string kind, raw; Json value; bool reference = false; };
struct Document {
    const std::string& body; std::size_t pos=0, visited=0; std::map<std::string,Node> nodes;
    void ws() { while(pos<body.size() && (body[pos]==' '||body[pos]=='\n'||body[pos]=='\r'||body[pos]=='\t')) ++pos; }
    char peek() { ws(); return pos<body.size()?body[pos]:'\0'; }
    void take(char c) { if(peek()!=c) bad("invalid structured JSON"); ++pos; }
    std::string string() {
        ws(); auto begin=pos; if(pos==body.size()||body[pos++]!='"')bad("invalid JSON string");
        bool end=false; while(pos<body.size()) { char c=body[pos++]; if(c=='"'){end=true;break;} if(c=='\\') {if(pos==body.size())bad("invalid JSON escape");++pos;} }
        if(!end)bad("unterminated JSON string");
        try { return Json::parse(body.substr(begin,pos-begin)).get<std::string>(); } catch(const Json::exception&) { bad("invalid JSON string encoding"); }
    }
    void parse(const std::string& path, unsigned depth) {
        if(depth>32 || ++visited>8192)bad("structured JSON parsing bound");
        ws(); const auto begin=pos; Node node; const auto c=peek();
        if(c=='{') {
            node.kind="object"; take('{'); std::set<std::string> keys;
            if(peek()!='}') for(;;) { auto key=string(); if(!keys.insert(key).second)bad("duplicate structured JSON key"); if(key=="$ref")node.reference=true; take(':'); parse(path+"/"+escaped(key),depth+1); if(peek()!=',')break;take(','); }
            take('}');
        } else if(c=='[') {
            node.kind="array";take('['); std::size_t index=0;
            if(peek()!=']') for(;;) {parse(path+"/"+std::to_string(index++),depth+1);if(peek()!=',')break;take(',');} take(']');
        } else if(c=='"') {node.kind="string";node.value=string();}
        else if(body.compare(pos,4,"true")==0){node.kind="boolean";node.value=true;pos+=4;}
        else if(body.compare(pos,5,"false")==0){node.kind="boolean";node.value=false;pos+=5;}
        else if(body.compare(pos,4,"null")==0){node.kind="null";node.value=nullptr;pos+=4;}
        else {
            node.kind="number";while(pos<body.size() && std::string_view("-+0123456789.eE").find(body[pos])!=std::string_view::npos)++pos;
            static const std::regex number("-?(0|[1-9][0-9]*)(\\.[0-9]+)?([eE][+-]?[0-9]+)?");
            if(!std::regex_match(body.substr(begin,pos-begin),number))bad("invalid JSON number");
        }
        node.raw=body.substr(begin,pos-begin); nodes.emplace(path,std::move(node));
    }
    explicit Document(const std::string& s):body(s){parse("",0);ws();if(pos!=body.size())bad("trailing structured JSON bytes");}
    const Node* select(const std::string& p, std::string& reason) const {
        const auto parts=pointer(p); std::string path;
        for(std::size_t i=0;;++i) {
            const auto found=nodes.find(path); if(found==nodes.end()){reason="pointer_missing";return nullptr;}
            if(found->second.reference){reason="reference_unresolved";return nullptr;}
            if(i==parts.size())return &found->second;
            if(found->second.kind!="array" && found->second.kind!="object"){reason="pointer_missing";return nullptr;}
            if(found->second.kind=="array") {
                const auto& x=parts[i]; if(x.empty() || (x.size()>1&&x.front()=='0') || !std::all_of(x.begin(),x.end(),[](char k){return k>='0'&&k<='9';})){reason="array_index_invalid";return nullptr;}
            }
            path+="/"+escaped(parts[i]);
        }
    }
};
std::string canonical_decimal(std::string s) {
    bool negative=!s.empty()&&s.front()=='-';if(negative)s.erase(0,1);
    const auto e=s.find_first_of("eE"); int exponent=0;
    if(e!=std::string::npos) {
        auto x=s.substr(e+1); if(!x.empty()&&x.front()=='+')x.erase(0,1);
        auto p=std::from_chars(x.data(),x.data()+x.size(),exponent);
        if(p.ec!=std::errc{} || p.ptr!=x.data()+x.size() || exponent>256 || exponent< -256)bad("numeric token outside exact decimal bound");s.resize(e);
    }
    const auto dot=s.find('.'); int point=static_cast<int>(dot==std::string::npos?s.size():dot)+exponent;
    if(dot!=std::string::npos)s.erase(dot,1);
    if(point<=0)s="0."+std::string(static_cast<std::size_t>(-point),'0')+s;
    else if(static_cast<std::size_t>(point)>=s.size())s+=std::string(static_cast<std::size_t>(point)-s.size(),'0');
    else s.insert(static_cast<std::size_t>(point),".");
    while(s.size()>1 && s[0]=='0' && s[1]!='.')s.erase(0,1);
    if(s.find('.')!=std::string::npos){while(s.back()=='0')s.pop_back();if(s.back()=='.')s.pop_back();}
    if(s.empty())s="0";if(negative&&s!="0")s="-"+s;if(s.size()>128)bad("numeric token outside exact decimal bound");return s;
}
Json extracted(const Node& n, const Json& descriptor) {
    const auto type=str(descriptor,"type"); Json value;
    if(type=="integer") {
        if(n.kind!="number" || n.raw.find_first_of(".eE")!=std::string::npos)bad("integer node required");
        std::int64_t number=0;const auto p=std::from_chars(n.raw.data(),n.raw.data()+n.raw.size(),number);
        if(p.ec!=std::errc{} || p.ptr!=n.raw.data()+n.raw.size() || number< -9007199254740991LL || number>9007199254740991LL)bad("integer node outside bound");value=number;
    } else if(type=="decimal") {if(n.kind!="number")bad("numeric decimal node required");value=canonical_decimal(n.raw);}
    else if(type=="boolean") {if(n.kind!="boolean")bad("boolean node required");value=n.value;}
    else if(type=="string"||type=="reference") {if(n.kind!="string"||n.value.get_ref<const std::string&>().empty()||n.value.get_ref<const std::string&>().size()>4096||n.value.get_ref<const std::string&>().find('\0')!=std::string::npos)bad("nonempty bounded string node required");value=n.value;}
    else bad("unsupported structured node type");
    return {{"type",type},{"value",value},{"unit",descriptor.at("unit")}};
}
void typed(const Json& v) {
    fields(v,{"type","value","unit"}); const auto t=str(v,"type");
    if(!v.at("unit").is_null())str(v,"unit",128);
    if(t=="integer") {
        const auto& n=v.at("value"); if(!n.is_number_integer() || (n.is_number_unsigned()&&n.get<std::uint64_t>()>9007199254740991ULL) || n.get<std::int64_t>()< -9007199254740991LL || n.get<std::int64_t>()>9007199254740991LL)bad("invalid typed integer");
    } else if(t=="decimal") {
        const auto s=str(v,"value",128);static const std::regex dec("-?(0|[1-9][0-9]*)(\\.[0-9]*[1-9])?");
        if(s=="-0"||!std::regex_match(s,dec))bad("noncanonical typed decimal");
    } else if(t=="boolean") {if(!v.at("value").is_boolean())bad("invalid typed boolean");}
    else if(t=="string"||t=="reference")str(v,"value",4096);else bad("invalid typed value");
}
Map profiles(const Json& pack, const engine::Request& r, const std::string& d) {
    arr(pack.at("profiles"),16);arr(pack.at("structured_profiles"),16);
    if(pack.at("profiles").size()+pack.at("structured_profiles").size()>16)bad("pack exceeds 16 profiles");
    Map out, sources; for(const auto& s:pack.at("provider").at("sources"))sources.emplace(str(s,"source_id"),s);
    std::set<std::string> ids;std::size_t rules=0;
    for(const auto* group:{"profiles","structured_profiles"})for(const auto& p:pack.at(group)) {
        check_time(r);auto draft=p;const bool structured=std::string_view(group)=="structured_profiles";
        if(structured) {
            fields(p,{"protocol","profile_id","profile_version","provider_id","source_id","locator_id","media_types","authored_by","rationale","rules"});
            if(p.at("protocol")!="symphony.scv.structured-profile.v1")bad("unsupported structured profile");
            draft["protocol"]="symphony.scv.interpretation-profile.v1";
            arr(p.at("rules"),128);
            for(std::size_t i=0;i<p.at("rules").size();++i) {
                const auto& rule=p.at("rules")[i];
                fields(rule,{"rule_id","claim_id","subject","predicate","scope","statement_kind","dependencies","context","extractor"});
                const auto& x=rule.at("extractor"); fields(x,{"kind","pointer","type","unit"});
                if(x.at("kind")!="json_pointer")bad("unsupported structured extractor");
                pointer(str(x,"pointer",1024,true)); const auto v=placeholder(x);typed(v);
                arr(rule.at("context"),8);
                for(const auto& c:rule.at("context")){fields(c,{"pointer","value"});pointer(str(c,"pointer",1024,true));typed(c.at("value"));}
                draft["rules"][i]["context"]=Json::array();draft["rules"][i]["extractor"]={{"kind","literal"},{"quote","structure-validation-only"},{"value",v}};
            }
        } else {seal(p);draft.erase("digest");}
        const auto prepared=call(r,"profile_prepare",{{"profile",draft}},d);
        if(!structured&&prepared!=p)bad("text profile cannot be replayed");
        const auto id=str(p,"profile_id");if(!out.emplace(id,p).second)bad("duplicate pack profile identity");
        if(p.at("provider_id")!=pack.at("provider").at("provider_id"))bad("pack profile provider mismatch");
        const auto source=sources.find(str(p,"source_id"));if(source==sources.end())bad("pack profile source not declared");
        bool found=false;for(const auto& l:source->second.at("locators"))if(l.at("locator_id")==p.at("locator_id"))found=true;
        if(!found)bad("pack profile locator not declared");
        for(const auto& rule:p.at("rules"))if(++rules>128||!ids.insert(str(rule,"claim_id")).second)bad("pack generated claim identity/rule bound");
    }
    return out;
}
void expected_extractions(const Json& a) {
    arr(a,128);std::set<std::string> unique;
    for(const auto& e:a) {
        fields(e,{"profile_id","capture_digest","rule_id","claim_id","status","reasons"});
        for(const auto* k:{"profile_id","rule_id","claim_id"})str(e,k);
        const auto dg=str(e,"capture_digest",71);static const std::regex digest("sha256:[0-9a-f]{64}");if(!std::regex_match(dg,digest))bad("fixture capture digest invalid");
        if(e.at("status")!="matched"&&e.at("status")!="unresolved")bad("fixture extraction status invalid");
        arr(e.at("reasons"),16);std::set<std::string> reasons;for(const auto& x:e.at("reasons"))if(!reasons.insert(str(Json{{"reason",x}},"reason",512)).second)bad("duplicate fixture reason");
        if(!unique.insert(Json::array({e.at("profile_id"),e.at("capture_digest"),e.at("rule_id")}).dump()).second)bad("duplicate expected extraction");
    }
}
Map validate_pack(const Json& pack,const engine::Request& r,const std::string& d,bool pending=false) {
    fields(pack,{"protocol","pack_id","pack_version","authored_by","provenance","provider","profiles","structured_profiles","fixtures","digest"});
    if(pack.at("protocol")!="symphony.scv.provider-pack.v1")bad("unsupported provider pack");seal(pack);
    str(pack,"pack_id");str(pack,"pack_version");str(pack,"authored_by",1024);
    arr(pack.at("provenance"),16);if(pack.at("provenance").empty())bad("pack requires authored provenance");
    for(const auto& p:pack.at("provenance"))str(Json{{"provenance",p}},"provenance",4096);
    static_cast<void>(call(r,"provider_onboard",pack.at("provider"),d));
    const auto ps=profiles(pack,r,d);arr(pack.at("fixtures"),16);if(pack.at("fixtures").empty())bad("pack requires a labeled conformance manifest");
    std::set<std::string> ids;
    for(const auto& f:pack.at("fixtures")) {
        fields(f,{"fixture_id","label","authored_by","rationale","input_digest","expected_claims","expected_extractions"});
        if(!ids.insert(str(f,"fixture_id")).second)bad("duplicate fixture identity");str(f,"label",1024);str(f,"authored_by",1024);str(f,"rationale",4096);
        if(!pending||!f.at("input_digest").is_null()) {static const std::regex digest("sha256:[0-9a-f]{64}");if(!std::regex_match(str(f,"input_digest",71),digest))bad("fixture input digest invalid");}
        arr(f.at("expected_claims"),128);Json cs=Json::array();
        for(auto c:f.at("expected_claims")){fields(c,{"claim_id","subject","predicate","scope","statement_kind","value","dependencies"});typed(c.at("value"));c["evidence"]=Json::array();cs.push_back(c);}
        validate_claims(cs,r,d);expected_extractions(f.at("expected_extractions"));
    }
    return ps;
}
Json selection(const Json& pack,const Map& ps,const Json& input,const engine::Request& r,const std::string& d) {
    arr(input.at("captures"),16);arr(input.at("bindings"),16);Map captures,sources;
    for(const auto& s:pack.at("provider").at("sources"))sources.emplace(str(s,"source_id"),s);
    for(const auto& c:input.at("captures")){validate_capture(c,d);if(!captures.emplace(str(c,"digest",71),c).second)bad("duplicate pack capture");}
    // The existing owner normalizes the policy even when every extraction fails.
    auto empty=call(r,"knowledge_interpret",{{"captures",vals(captures)},{"claims",Json::array()},{"interpreter_version","pack-policy-validation-v1"},{"selection_policy",input.at("selection_policy")}},d);
    const auto policy=empty.at("selection_policy");std::set<std::string> used_profiles,used_captures;
    Json claims=Json::array(),extractions=Json::array();
    for(const auto& b:sorted(input.at("bindings"))) {
        check_time(r);fields(b,{"profile_id","capture_digest"});const auto p=str(b,"profile_id"),c=str(b,"capture_digest",71);
        if(!ps.contains(p)||!captures.contains(c)||!used_profiles.insert(p).second)bad("pack binding must select one known profile/capture");used_captures.insert(c);
        const auto& profile=ps.at(p);const auto& cap=captures.at(c);const auto& source=cap.at("source");
        if(profile.at("provider_id")!=source.at("provider_id")||profile.at("source_id")!=source.at("source_id")||profile.at("locator_id")!=cap.at("locator_id"))bad("pack binding owner mismatch");
        bool declaration=true;const auto& desired=sources.at(str(profile,"source_id"));for(const auto& [key,v]:desired.items())if(!source.contains(key)||source.at(key)!=v)declaration=false;
        const bool structured=profile.at("protocol")=="symphony.scv.structured-profile.v1";
        if(!structured && declaration) {
            auto old=call(r,"provider_interpret",{{"captures",Json::array({cap})},{"profiles",Json::array({profile})},{"bindings",Json::array({Json{{"profile_digest",profile.at("digest")},{"capture_digest",c}}})},{"selection_policy",policy}},d);
            for(const auto& v:old.at("knowledge").at("claims"))claims.push_back(v);
            for(auto e:old.at("extractions")){e.erase("profile_digest");e["profile_id"]=p;extractions.push_back(e);}continue;
        }
        const auto& body=cap.at("body").get_ref<const std::string&>();
        std::string stage;
        if(!declaration)stage="source_declaration_differs";
        else if(cap.at("completeness")=="failed")stage="capture_failed";
        else if(cap.at("completeness")=="partial"&&policy.at("partial_capture")=="exclude")stage="partial_capture_excluded";
        else if(std::find(profile.at("media_types").begin(),profile.at("media_types").end(),cap.at("media_type"))==profile.at("media_types").end())stage="unsupported_media_type";
        std::optional<Document> doc;
        if(stage.empty())try{doc.emplace(body);}catch(const engine::Error&){stage="invalid_json";}
        for(const auto& rule:profile.at("rules")) {
            check_time(r);std::set<std::string> reasons,anchors;Json value=nullptr;
            if(!stage.empty())reasons.insert(stage);
            else {
                std::size_t i=0;
                for(const auto& context:rule.at("context")) {
                    std::string why;const auto* n=doc->select(str(context,"pointer",1024,true),why);
                    if(!n)reasons.insert("context_"+why+":"+std::to_string(i));
                    else try{if(extracted(*n,context.at("value"))!=context.at("value"))reasons.insert("context_value_differs:"+std::to_string(i));else if(n->raw.size()>4096)reasons.insert("context_quote_exceeds_bound:"+std::to_string(i));else anchors.insert(n->raw);}catch(const engine::Error&){reasons.insert("context_type_mismatch:"+std::to_string(i));}
                    ++i;
                }
                std::string why;const auto& x=rule.at("extractor");const auto* n=doc->select(str(x,"pointer",1024,true),why);
                if(!n)reasons.insert(why);
                else if(n->raw.size()>4096)reasons.insert("value_quote_exceeds_evidence_bound");
                else try{value=extracted(*n,x);anchors.insert(n->raw);}catch(const engine::Error&){reasons.insert("value_type_mismatch");}
            }
            const bool matched=reasons.empty();
            if(matched){auto v=claim(rule,value);for(const auto& a:anchors)v["evidence"].push_back({{"capture_digest",c},{"quote",a}});claims.push_back(v);if(cap.at("completeness")=="partial")reasons.insert("partial_capture_qualified");}
            extractions.push_back({{"profile_id",p},{"capture_digest",c},{"rule_id",rule.at("rule_id")},{"claim_id",rule.at("claim_id")},{"status",matched?"matched":"unresolved"},{"reasons",reasons}});
        }
    }
    if(used_captures.size()!=captures.size())bad("every pack capture requires explicit profile binding");
    const auto knowledge=call(r,"knowledge_interpret",{{"captures",vals(captures)},{"claims",claims},{"interpreter_version","provider-pack-v1:"+str(pack,"digest",71)},{"selection_policy",policy}},d);
    return {{"knowledge",knowledge},{"extractions",sorted(extractions)}};
}
Map fixture_inputs(const Json& inputs) {
    arr(inputs,16);Map out;std::size_t count=0;
    for(const auto& f:inputs){fields(f,{"fixture_id","captures","bindings","selection_policy"});arr(f.at("captures"),16);count+=f.at("captures").size();if(count>16)bad("selected fixture inputs exceed 16 total captures");if(!out.emplace(str(f,"fixture_id"),f).second)bad("duplicate selected fixture input");}
    return out;
}
Json prepare(const Json& input,const engine::Request& r,const std::string& d) {
    fields(input,{"pack","fixtures"});auto draft=input.at("pack");
    fields(draft,{"protocol","pack_id","pack_version","authored_by","provenance","provider","profiles","structured_profiles","fixtures"});
    auto pack=sealed(draft);const auto ps=validate_pack(pack,r,d,true);const auto supplied=fixture_inputs(input.at("fixtures"));std::set<std::string> used;
    for(auto& f:pack["fixtures"]) {
        const auto id=str(f,"fixture_id");const auto it=supplied.find(id);
        if(it!=supplied.end()) {
            const auto digest=engine::tagged_sha256(it->second.dump());if(!f.at("input_digest").is_null()&&f.at("input_digest")!=digest)bad("supplied fixture differs from precomputed manifest");
            f["input_digest"]=digest;used.insert(id);
            static_cast<void>(selection(pack,ps,it->second,r,d));
        } else if(f.at("input_digest").is_null())bad("fixture with null digest requires explicit fixture input");
    }
    if(used.size()!=supplied.size())bad("fixture input is not declared in pack");pack=sealed(pack);static_cast<void>(validate_pack(pack,r,d));return pack;
}
Json evaluate(const Json& input,const engine::Request& r,const std::string& d) {
    fields(input,{"pack","captures","bindings","selection_policy","fixtures"});const auto& pack=input.at("pack");const auto ps=validate_pack(pack,r,d);
    const auto result=selection(pack,ps,input,r,d);const auto supplied=fixture_inputs(input.at("fixtures"));std::set<std::string> used;
    Json cases=Json::array();std::size_t passed=0,failed=0,not_run=0;
    for(const auto& f:pack.at("fixtures")) {
        check_time(r);const auto id=str(f,"fixture_id");const auto it=supplied.find(id);Json ac=nullptr,ae=nullptr,axes=nullptr;std::string status="not_run";
        if(it!=supplied.end()) {
            if(f.at("input_digest")!=engine::tagged_sha256(it->second.dump()))bad("fixture body differs from declared input digest");used.insert(id);
            const auto actual=selection(pack,ps,it->second,r,d);ac=project_claims(actual.at("knowledge").at("claims"));ae=actual.at("extractions");
            const bool dc=ac!=sorted(f.at("expected_claims")),de=ae!=sorted(f.at("expected_extractions"));axes={{"claims",dc},{"extractions",de}};
            status=dc||de?"failed":"passed";if(status=="passed")++passed;else++failed;
        } else ++not_run;
        cases.push_back({{"fixture_id",id},{"status",status},{"expected_claims",f.at("expected_claims")},{"expected_extractions",f.at("expected_extractions")},{"actual_claims",ac},{"actual_extractions",ae},{"difference_axes",axes}});
    }
    if(used.size()!=supplied.size())bad("selected fixture is not declared in pack");
    return sealed(Json{{"protocol","symphony.scv.provider-pack-evaluation.v1"},{"domain",d},{"input",input},{"knowledge",result.at("knowledge")},{"extractions",result.at("extractions")},{"fixture_results",sorted(cases)},
        {"conformance",{{"passed",passed},{"failed",failed},{"not_run",not_run}}},{"limitations",Json::array({
            "pack mappings and fixture expectations are authored inputs; passing selected fixtures is not certification or provider endorsement",
            "structured extraction evaluates explicit JSON pointers and typed contexts only; references, API behavior and undeclared exceptions remain unevaluated",
            "document numeric lexemes are represented exactly within declared bounds; units, scope and statement kinds are never inferred or converted",
            "fixture selection and production selection are separate; not_run fixtures and unresolved rules provide no positive proof"})}});
}
}
Json handle_pack(const engine::Request& request,const std::string& domain) {
    check_time(request);
    if(request.operation=="provider_pack_prepare")return prepare(request.payload,request,domain);
    if(request.operation=="provider_pack_evaluate")return evaluate(request.payload,request,domain);
    bad("unsupported provider pack operation");
}
Json replay_pack(const Json& wrapper,const std::string& domain,const engine::Request& request) {
    fields(wrapper,{"protocol","domain","input","knowledge","extractions","fixture_results","conformance","limitations","digest"});
    if(wrapper.at("protocol")!="symphony.scv.provider-pack-evaluation.v1")bad("unsupported pack evaluation wrapper");seal(wrapper);
    // Validate the declared source family/provider against the consuming owner,
    // then replay under the original recorded owner without relabeling it.
    static_cast<void>(validate_pack(wrapper.at("input").at("pack"),request,domain));
    const auto original=str(wrapper,"domain");
    const std::set<std::string> known={"scv","schv","scev","schv-aws","schv-azure","schv-do","schv-gcp","scev-cf"};
    if(!known.contains(original))bad("unknown retained pack owner");
    if(evaluate(wrapper.at("input"),request,original)!=wrapper)bad("pack evaluation cannot be exactly replayed");
    return wrapper.at("knowledge");
}
}
