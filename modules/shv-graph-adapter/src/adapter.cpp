#include "adapter.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/operation.hpp"
#include <algorithm>
#include <set>
namespace symphony::graph {
namespace engine=knowledge::engine;
namespace {
[[noreturn]] void fail(const char* message) { throw engine::Error("graph.invalid_request",message,3); }
void require(bool good,const char* message) { if(!good) fail(message); }
void fields(const Json& j,std::initializer_list<const char*> names) { require(j.is_object() && j.size()==names.size(),"unexpected fields"); for(auto n:names) require(j.contains(n),"missing field"); }
std::string text(const Json& j,std::size_t limit=4096) { require(j.is_string(),"text required"); auto s=j.get<std::string>(); require(!s.empty() && s.size()<=limit,"text bound"); for(unsigned char c:s) require(c>=32 && c!=127,"control in identity"); return s; }
Json seal(Json j,const char* key="digest") { j[key]=engine::tagged_sha256(j.dump()); return j; }
void sealed(const Json& j) { require(j.is_object() && j.contains("digest"),"sealed object required"); auto copy=j; auto d=text(copy.at("digest"),71); copy.erase("digest"); require(engine::tagged_sha256(copy.dump())==d,"digest mismatch"); }
void ordered_strings(const Json& a,std::size_t maximum) { require(a.is_array() && a.size()<=maximum,"array bound"); std::string previous; for(const auto& v:a) { auto s=text(v); require(previous.empty() || previous<s,"identities not sorted unique"); previous=s; } }
void wire_subset(const Json& j) { try { static_cast<void>(engine::parse_bounded_json(j.dump(),engine::Limits::max_request_bytes)); } catch(const engine::Error&) { fail("graph exceeds canonical process subset"); } }
}
void validate_exchange(const Json& graph) {
 wire_subset(graph);
 fields(graph,{"protocol","owner","owner_artifact","nodes","edges","digest"});
 require(graph.at("protocol")==exchange_protocol,"unsupported exchange protocol"); sealed(graph);
 const auto& owner=graph.at("owner"); fields(owner,{"engine_id","engine_version","artifact_protocol","artifact_digest"});
 for(auto k:{"engine_id","engine_version","artifact_protocol"}) static_cast<void>(text(owner.at(k),128));
 const auto& artifact=graph.at("owner_artifact"); sealed(artifact);
 require(artifact.contains("protocol") && artifact.at("protocol")==owner.at("artifact_protocol") && artifact.at("digest")==owner.at("artifact_digest"),"owner artifact identity mismatch");
 const auto& nodes=graph.at("nodes"); require(nodes.is_array() && nodes.size()<=1024,"node bound");
 std::set<std::string> node_ids; std::string previous;
 for(const auto& n:nodes) { fields(n,{"id","labels","properties"}); auto id=text(n.at("id"),512); require(previous.empty() || previous<id,"node IDs not sorted unique"); previous=id; node_ids.insert(id); ordered_strings(n.at("labels"),32); require(n.at("properties").is_object(),"node properties must be object"); }
 const auto& edges=graph.at("edges"); require(edges.is_array() && edges.size()<=2048,"edge bound"); previous.clear();
 for(const auto& e:edges) { fields(e,{"id","from","to","label","properties"}); auto id=text(e.at("id"),512); require(previous.empty() || previous<id,"edge IDs not sorted unique"); previous=id; require(node_ids.contains(text(e.at("from"),512)) && node_ids.contains(text(e.at("to"),512)),"dangling endpoint"); static_cast<void>(text(e.at("label"),128)); require(e.at("properties").is_object(),"edge properties must be object"); }
}
Json PortableReference::roundtrip(const Json& graph) const { validate_exchange(graph); return seal(Json{{"protocol","symphony.graph.adapter-result.v1"},{"backend","portable-reference"},{"operation","roundtrip"},{"graph",graph}}); }
Json PortableReference::query(const Json& graph,const std::string& kind,const std::vector<std::string>& ids) const {
 validate_exchange(graph); require(kind=="nodes" || kind=="edges","unsupported row kind"); require(ids.size()<=2048,"query ID bound"); std::set<std::string> requested;
 for(const auto& id:ids) { static_cast<void>(text(Json(id),512)); require(requested.insert(id).second,"duplicate query ID"); }
 Json rows=Json::array(),missing=Json::array(); std::set<std::string> found;
 for(const auto& row:graph.at(kind)) if(ids.empty() || requested.contains(row.at("id").get<std::string>())) { rows.push_back(row); found.insert(row.at("id").get<std::string>()); }
 for(const auto& id:ids) if(!found.contains(id)) missing.push_back(id);
 return seal(Json{{"protocol","symphony.graph.adapter-result.v1"},{"backend","portable-reference"},{"operation","query"},{"graph_digest",graph.at("digest")},{"kind",kind},{"ids",ids},{"rows",rows},{"missing_ids",missing}});
}
Json descriptor() {
 std::vector<engine::OperationSpec> ops;
 for(const auto* name:{"inspect","roundtrip","query"}) ops.push_back({std::string("engop:symphony:shv-graph-adapter.")+name,name,"implemented",false,true,{"ssfv:symphony:shv-graph-adapter"},{name==std::string("inspect")?"inspect":name==std::string("query")?"query":"invoke"},"qxctl_required",std::string("symphony.graph.adapter-")+name+"-input.v1",name==std::string("inspect")?engine::descriptor_protocol_v2:"symphony.graph.adapter-result.v1","read_only","idempotent",false,"none","","supported","freezing"});
 engine::validate_operation_specs(ops);
 return seal(Json{{"protocol",engine::descriptor_protocol_v2},{"format_version",2},{"module_id","shv-graph-adapter"},{"engine_id",engine_id},{"vector_id","shv"},{"engine_version",version},{"process_protocols",Json::array({engine::process_protocol_v1})},{"contract_versions",Json::array({"shv-graph-adapter/SPEC.md@v1"})},{"operations",engine::administration_operation_descriptors(ops)},{"limits",{{"request_bytes",engine::Limits::max_request_bytes},{"response_bytes",engine::Limits::max_response_bytes},{"json_depth",engine::Limits::max_json_depth},{"json_values",engine::Limits::max_json_values},{"path_bytes",engine::Limits::max_path_bytes},{"snapshot_files",engine::Limits::max_snapshot_files},{"snapshot_file_bytes",engine::Limits::max_snapshot_file_bytes},{"deadline_ahead_ms",engine::Limits::max_deadline_ahead_ms}}},{"supported_scopes",Json::array({"user"})},{"language","C++26"},{"thermal_path","freezing"},{"canonical_apply_enabled",false},{"session_mutation_enabled",false},{"network_listener",false}},"descriptor_digest");
}
Json handle(const engine::Request& request) {
 require(engine::unix_time_ms()<request.deadline_unix_ms,"expired request");
 if(request.operation=="inspect") { fields(request.payload,{}); return descriptor(); }
 PortableReference adapter;
 if(request.operation=="roundtrip") { fields(request.payload,{"graph"}); return adapter.roundtrip(request.payload.at("graph")); }
 if(request.operation=="query") { fields(request.payload,{"graph","kind","ids"}); require(request.payload.at("ids").is_array(),"query IDs must be array"); std::vector<std::string> ids; for(const auto& id:request.payload.at("ids")) ids.push_back(text(id,512)); return adapter.query(request.payload.at("graph"),text(request.payload.at("kind")),ids); }
 fail("unsupported operation");
}
}
