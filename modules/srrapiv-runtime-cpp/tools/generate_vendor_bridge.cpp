// Regenerate Symphony-owned bindings from a reviewed interoperability schema.
// The licensed vendor header is supplied by the operator and must match the
// selected schema's exact digest. A new SDK requires a reviewed schema and
// compiler/ABI qualification; matching declaration counts are not approval.
#include <nlohmann/json.hpp>
#include <symphony/knowledge/engine/digest.hpp>
#include <algorithm>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <map>
#include <set>
#include <sstream>
#include <stdexcept>
#include <string>
#include <vector>
using Json=nlohmann::json;
using Lines=std::vector<std::string>;
std::string str(const Json& j,const char* key){return j.at(key).get<std::string>();}
std::string quote(const std::string& s){return Json(s).dump();}
std::string join(const Lines& values,const std::string& separator=", "){std::string out;for(const auto& v:values){if(!out.empty())out+=separator;out+=v;}return out;}
std::string read(const std::filesystem::path& p){std::ifstream in(p,std::ios::binary);if(!in)throw std::runtime_error("input cannot be opened");std::ostringstream text;text<<in.rdbuf();return text.str();}
void write(const std::filesystem::path& p,const std::string& text){std::ofstream out(p,std::ios::binary|std::ios::trunc);if(!out||!(out<<text))throw std::runtime_error("output write failed");}
void write(const std::filesystem::path& p,const Lines& lines){write(p,join(lines,"\n")+"\n");}
std::string base_type(std::string t){t.erase(std::remove(t.begin(),t.end(),'*'),t.end());for(auto p=t.find("const");p!=std::string::npos;p=t.find("const"))t.erase(p,5);auto first=t.find_first_not_of(' ');auto last=t.find_last_not_of(' ');return t.substr(first,last-first+1);}
bool has(const std::string& s,const std::string& part){return s.find(part)!=std::string::npos;}
int main(int argc,char** argv){try {
 std::filesystem::path header,schema,output;
 for(int i=1;i<argc;i+=2){if(i+1==argc)throw std::runtime_error("missing option value");std::string key=argv[i];if(key=="--header")header=argv[i+1];else if(key=="--schema")schema=argv[i+1];else if(key=="--output")output=argv[i+1];else throw std::runtime_error("unknown option");}
 if(header.empty()||schema.empty()||output.empty())throw std::runtime_error("required: --header licensed/RApiPlus.h --schema vendor_schema_13_7.json --output generated-directory");
 auto raw=read(header);auto manifest=Json::parse(read(schema));
 if(symphony::knowledge::engine::sha256_hex(std::string_view(raw))!=str(manifest,"header_sha256"))throw std::runtime_error("header digest differs from reviewed version schema; review new SDK before regeneration");
 if(manifest.at("sdk_version")!="13.7.0.0"||manifest.at("operations").size()!=118||manifest.at("callbacks").size()!=95)throw std::runtime_error("unsupported schema census");
 manifest["generator"]="generate_vendor_bridge.cpp/v1";
 auto data=manifest.at("classes");auto engine=manifest.at("operations");auto all_callbacks=manifest.at("callbacks");
 std::map<std::string,Json> index;std::size_t fields=0;for(const auto& c:data){index.emplace(str(c,"name"),c);fields+=c.at("fields").size();}
 if(fields!=1231||data.size()!=113)throw std::runtime_error("unsupported class/field census");
 auto ancestors=[&](const Json& cls){Lines names;auto c=cls;while(true){for(const auto& f:c.at("fields"))names.push_back(str(f,"name"));if(c.at("base").is_null())break;c=index.at(str(c,"base"));}return names;};
 auto quoted=[](const Lines& input){Lines out;for(const auto& v:input)out.push_back(quote(v));return join(out);};
 std::filesystem::create_directories(output);
 Lines initializers{"// Initialize public interchange fields explicitly; native constructors still own private state."};
 for(const auto& c:data)initializers.push_back("void initialize_public("+str(c,"name")+"& value);");
 for(const auto& c:data){initializers.push_back("void initialize_public("+str(c,"name")+"& value) {");if(!c.at("base").is_null())initializers.push_back("  initialize_public(static_cast<"+str(c,"base")+"&>(value));");for(const auto& f:c.at("fields")){auto t=str(f,"type"),n=str(f,"name");initializers.push_back(index.contains(t)?"  initialize_public(value."+n+");":"  value."+n+" = {};");}initializers.push_back("}");}
 write(output/"vendor_initializers.inc",initializers);
 write(output/"vendor_surface.json",manifest.dump(2)+"\n");
 // Each string literal is under the implementation's minimum supported size.
 Lines metadata{"// Generated interoperability metadata; do not edit.","static const std::string vendor_metadata = [] { std::string value;"};auto compact=manifest.dump();
 for(std::size_t i=0;i<compact.size();i+=30000)metadata.push_back("  value += "+quote(compact.substr(i,30000))+";");
 metadata.push_back("  return value; }();");write(output/"vendor_metadata.inc",metadata);
 Lines declarations,code{"// Generated typed codecs. No vendor implementation is included."};
 for(const auto& c:data){auto name=str(c,"name");declarations.push_back("Json encode(const "+name+"& v, Encoding& e);");declarations.push_back("void decode(const Json& j, "+name+"& v, Arena& a);");}
 code.insert(code.end(),declarations.begin(),declarations.end());
 for(const auto& c:data){auto name=str(c,"name");auto base=c.at("base").is_null()?std::string():str(c,"base");
  code.push_back("Json encode(const "+name+"& v, Encoding& e) {");code.push_back(base.empty()?"  Json j = Json::object();":"  Json j = encode(static_cast<const "+base+"&>(v), e);");code.push_back("  e.consume("+std::to_string(c.at("fields").size())+", 0);");
  for(const auto& f:c.at("fields")){auto n=str(f,"name"),t=str(f,"type"),count=f.at("array_count").is_null()?std::string():str(f,"array_count");std::string expr;
   if(name=="EnvironmentVariable"&&n=="sValue")expr="encode_environment_value(v.sName, v.sValue, e)";else if(has(n,"Password"))expr="Json{{\"redacted\", true}}";else if(t=="char * *")expr="encode_environment(v."+n+", e)";else if(t=="RCallbacks *"||t=="AdmCallbacks *")expr="e.state.handle(v."+n+", "+quote(base_type(t))+")";else if(!count.empty())expr="encode_array(v."+n+", v."+count+", e)";else if(has(t,"*"))expr="encode_pointer(v."+n+", e)";else expr="encode(v."+n+", e)";
   if(f.contains("presence")){const auto& condition=f.at("presence");expr="(v."+str(condition,"field")+" == "+condition.at("equals").dump()+") ? "+expr+" : Json{{\"state\",\"native-value-unavailable\"},{\"presence_field\","+quote(str(condition,"field"))+"}}";}
   code.push_back("  j["+quote(n)+"] = "+expr+";");
  }
  code.push_back("  return j;\n}");code.push_back("void decode(const Json& j, "+name+"& v, Arena& a) {\n  require_object(j);");
  if(!base.empty()){code.push_back("  Json base_fields = select_keys(j, {"+quoted(ancestors(index.at(base)))+"});");code.push_back("  decode(base_fields, static_cast<"+base+"&>(v), a);");}
  code.push_back("  validate_keys(j, {"+quoted(ancestors(c))+"});");
  std::map<std::string,Lines> count_groups;for(const auto& field:c.at("fields"))if(!field.at("array_count").is_null())count_groups[str(field,"array_count")].push_back(str(field,"name"));
  for(const auto& group:count_groups)code.push_back("  v."+group.first+" = validate_array_group(j, "+quote(group.first)+", {"+quoted(group.second)+"}, v."+group.first+", a);");
  for(const auto& f:c.at("fields")){auto n=str(f,"name"),t=str(f,"type"),count=f.at("array_count").is_null()?std::string():str(f,"array_count");auto value="j.at("+quote(n)+")";std::string statement;
   if(has(n,"Password"))statement="v."+n+" = a.secret_string("+value+");";else if(t=="char * *")statement="v."+n+" = a.environment("+value+");";else if(t=="void *")statement="v."+n+" = a.state.context("+value+");";else if(t=="RCallbacks *"||t=="AdmCallbacks *")statement="v."+n+" = static_cast<"+t+">(a.state.callback_handle("+quote(base_type(t))+", "+value+"));";else if(!count.empty())statement="v."+n+" = decode_array<"+base_type(t)+">("+value+", a);";else if(has(t,"*"))statement="v."+n+" = decode_pointer<"+base_type(t)+">("+value+", a);";else statement="decode("+value+", v."+n+", a);";
   code.push_back("  if (j.contains("+quote(n)+")) {\n    "+statement+"\n  }");
  }
  if(name=="LoginParams")code.push_back("  if (!j.contains(\"pCallbacks\")) v.pCallbacks = static_cast<RCallbacks*>(a.state.callback_handle(\"RCallbacks\", nullptr));");
  if(name=="REngineParams")code.push_back("  if (!j.contains(\"pAdmCallbacks\")) v.pAdmCallbacks = static_cast<AdmCallbacks*>(a.state.callback_handle(\"AdmCallbacks\", nullptr));");code.push_back("}");
 }
 write(output/"vendor_declarations.inc",declarations);write(output/"vendor_codecs.inc",code);
 Lines callback_code{"// Generated callback adapters; every overload is distinct."},checks;
 for(const auto& pair:std::vector<std::pair<std::string,std::string>>{{"Callbacks","RCallbacks"},{"AdminCallbacks","AdmCallbacks"}}){const auto& cls=pair.first;const auto& native=pair.second;
  callback_code.push_back("class "+cls+" final : public "+native+" {\npublic:\n  explicit "+cls+"(State& s) : state(s) {}\n  State& state;");
  for(const auto& m:all_callbacks){auto id=str(m,"id");if(id.rfind(native+".",0)!=0)continue;Lines signature,args;
   for(const auto& p:m.at("parameters"))signature.push_back(str(p,"type")+" "+str(p,"name"));
   callback_code.push_back("  int "+str(m,"name")+"("+join(signature)+") override {\n    try { Encoding e(state); Json payload = Json::object();");
   checks.push_back("{\n  "+cls+"& cb = static_cast<"+cls+"&>(*state."+(cls=="Callbacks"?"callbacks":"admin")+");");
   for(const auto& p:m.at("parameters")){auto n=str(p,"name"),t=str(p,"type");if(n=="aiCode"){checks.push_back("  int code = -1;");args.push_back("&code");continue;}auto expr=has(t,"*")?"encode_pointer("+n+", e)":"encode("+n+", e)";callback_code.push_back("      payload["+quote(n)+"] = "+expr+";");if(t=="void *")args.push_back("nullptr");else{checks.push_back("  "+base_type(t)+" "+n+"{};");if(index.contains(base_type(t)))checks.push_back("  initialize_public("+n+");");args.push_back("&"+n);}}
   callback_code.push_back("      state.deliver("+quote(id)+", std::move(payload));\n    } catch (const std::exception&) { state.encoding_failure("+quote(id)+", \"callback payload failed bounded encoding\"); } catch (...) { state.encoding_failure("+quote(id)+", \"non-standard callback exception\"); }\n    if (aiCode) *aiCode = API_OK; return OK;\n  }");
   checks.push_back("  assert(cb."+str(m,"name")+"("+join(args)+") == OK); assert(code == API_OK);\n  assert(events.back().at(\"event\") == "+quote(id)+");\n}");
  }callback_code.push_back("};");
 }
 write(output/"vendor_callbacks.inc",callback_code);write(output/"vendor_callback_checks.inc",checks);
 const std::set<std::string> getters{"getAdminName","getAppName","getAppVersion","getIhCnnctPt","getMdCnnctPt","getPnlCnnctPt","getTsCnnctPt","getUser","getVersion"};
 const std::set<std::string> special{"getPassword","lock","tryLock","unlock","addTimer"};
 const std::map<std::string,std::set<std::string>> arrays{{"cancelOrderList",{"aoAccountArray","asOrderNumArray","asEntryTypeArray","asTradingAlgorithmArray","asUserMsgArray","apContextArray"}},{"cancelQuoteList",{"pParams"}},{"createUserDefinedSpread",{"asLegTickers","aiLegRatios"}},{"linkOrders",{"aoAccountArray","asOrderNumArray"}},{"modifyOrderList",{"pParams"}},{"searchInstrument",{"asTerms"}},{"sendBracketOrder",{"pTargetQty","pTargetTicks","pStopQty","pStopTicks"}},{"sendOcaList",{"pParamsArray"}},{"sendOrderList",{"pParamsArray"}},{"submitQuoteList",{"pParams"}}};
 Lines dispatch{"// Generated typed dispatch; overloads resolved by native argument types."};
 for(const auto& m:engine){auto name=str(m,"name"),id=str(m,"id");if(special.contains(name))continue;Lines keys,call,posts,array_names;
  for(const auto& p:m.at("parameters")){auto n=str(p,"name"),t=str(p,"type");if(n!="aiCode"&&!getters.contains(name)&&!(has(t,"* *")&&name!="cancelOrderList")&&n!="aulPendingBytes")keys.push_back(n);}
  dispatch.push_back("if (id == "+quote(id)+") {\n  validate_keys(p, {"+quoted(keys)+"});\n  Arena arena(state); int native_code = API_OK; Json outputs = Json::object();");
  for(const auto& p:m.at("parameters")){auto n=str(p,"name"),t=str(p,"type"),bare=base_type(t),required="required(p, "+quote(n)+")";
   if(n=="aiCode"){call.push_back("&native_code");continue;}
   if(getters.contains(name)){dispatch.push_back("  tsNCharcb "+n+"{};");call.push_back("&"+n);posts.push_back("outputs["+quote(n)+"] = encode("+n+", encoding);");continue;}
   if(has(t,"* *")&&name!="cancelOrderList"){dispatch.push_back("  "+bare+"* "+n+" = nullptr;");call.push_back("&"+n);posts.push_back("outputs["+quote(n)+"] = state.handle("+n+", "+quote(name=="getOmne"?"native-engine":bare=="void"?"context":bare)+");");continue;}
   if(n=="aulPendingBytes"){dispatch.push_back("  unsigned long int "+n+" = 0;");call.push_back("&"+n);posts.push_back("outputs["+quote(n)+"] = encode("+n+", encoding);");continue;}
   if(arrays.contains(name)&&arrays.at(name).contains(n)){auto base=std::count(t.begin(),t.end(),'*')==1?bare:"void*";dispatch.push_back("  auto "+n+" = decode_array<"+base+">("+required+", arena);");call.push_back(n);array_names.push_back(n);continue;}
   if(t=="void *"){dispatch.push_back("  void* "+n+" = state.context(p.value("+quote(n)+", Json()));");call.push_back(n);continue;}
   if(t=="RCallbacks *"){dispatch.push_back("  auto "+n+" = static_cast<RCallbacks*>(state.callback_handle(\"RCallbacks\", p.value("+quote(n)+", Json())));");call.push_back(n);continue;}
   if(bare=="tsNCharcb"){if(has(n,"Password")){dispatch.push_back("  tsNCharcb "+n+" = arena.secret_string("+required+");");call.push_back("&"+n);}else{dispatch.push_back("  tsNCharcb* "+n+" = decode_pointer<tsNCharcb>("+required+", arena);");call.push_back(n);}continue;}
   if(has(t,"*")){dispatch.push_back("  auto "+n+" = decode_pointer<"+bare+">("+required+", arena); if (!"+n+") throw std::invalid_argument(\"null native object: "+n+"\");");call.push_back(n);continue;}
   dispatch.push_back("  "+t+" "+n+"{}; decode("+required+", "+n+", arena);");call.push_back(n);
  }
  for(const auto& n:array_names){std::string count="iArrayLen";if(name=="createUserDefinedSpread")count="iNumLegs";if(name=="searchInstrument")count="iNumTerms";if(name=="sendBracketOrder")count=has(n,"Target")?"iTargetArrayLen":"iStopArrayLen";dispatch.push_back("  require_array_count(p.at("+quote(n)+"), "+count+");");}
  dispatch.push_back("  state.native_call_started = true; int native_return = "+std::string(name=="getVersion"?"REngine::":"state.engine->")+name+"("+join(call)+");\n  Encoding encoding(state); if (native_return == OK) { "+join(posts," ")+" }\n  return native_result(id, native_return, native_code, std::move(outputs));\n}");
 }
 write(output/"vendor_dispatch.inc",dispatch);
 Lines objects{"// Generated native object helpers."};
 for(const auto& c:data){auto name=str(c,"name");std::set<std::string> helpers;auto current=c;while(true){for(const auto& m:current.at("methods"))helpers.insert(str(m,"name"));if(current.at("base").is_null())break;current=index.at(str(current,"base"));}
  objects.push_back("if (type == "+quote(name)+") {\n  auto* value = arena.make<"+name+">();\n  if (p.contains(\"value\")) decode(p.at(\"value\"), *value, arena);");
  if(helpers.contains("clearHandles"))objects.push_back("  if (action == \"clear\") { int code=API_OK; int rc=value->clearHandles(&code); if (rc != OK) return native_result(id,rc,code,Json::object()); }");else objects.push_back("  if (action == \"clear\") throw std::invalid_argument(\"this native type has no clearHandles method\");");
  objects.push_back("  Encoding encoding(state); return Json{{\"status\",\"ok\"},{\"type\",type},{\"action\",action},{\"value\",encode(*value,encoding)},{\"diagnostic_adaptation\",action==\"dump\" ? \"structured-inspection\" : \"native-object-lifecycle\"}};\n}");
 }
 write(output/"vendor_objects.inc",objects);
 std::cout<<Json{{"operations",118},{"callbacks",95},{"direct_fields",fields},{"header_verified",true}}.dump()<<'\n';return 0;
 }catch(const std::exception& error){std::cerr<<"vendor binding generation failed: "<<error.what()<<'\n';return 1;}}
