#include "catalog.hpp"
#include "runtime.hpp"
#include "private_io.hpp"
#include "vendor.hpp"
#include "symphony/knowledge/engine/operation.hpp"
#include <algorithm>
#include <iostream>

namespace owner = symphony::srrapiv;
namespace engine = symphony::knowledge::engine;
using Json=engine::Json;
constexpr auto engine_id="symphony-srrapiv";
constexpr auto version="0.1.0-dev";
std::vector<engine::OperationSpec> operations() {
  std::vector<engine::OperationSpec> ops;
  for(const auto *name:{"catalog_inspect","binding_validate","binding_compare","coverage_assess","api_describe","runtime_start","runtime_status","runtime_invoke","runtime_events","runtime_stop","runtime_attempts"}) {
    std::string operation(name),key(name);std::replace(key.begin(),key.end(),'_','.');
    bool mutation=operation=="runtime_start"||operation=="runtime_stop"||operation=="runtime_invoke";
    std::string interaction="inspect";
    if(operation=="api_describe")interaction="discover";
    if(operation=="binding_validate")interaction="validate";
    if(operation=="runtime_start"||operation=="runtime_stop")interaction="lifecycle";
    if(operation=="runtime_invoke")interaction="invoke";
    if(operation=="runtime_attempts")interaction="recover";
    ops.push_back({"engop:symphony:srrapiv."+key,operation,"implemented",false,true,
      {"ssfv:symphony:srrapiv-runtime-cpp"},{interaction},"qxctl_required",
      "symphony.srrapiv.request.v1","symphony.srrapiv.result.v1",
      mutation?"permission_backed_mutation":"read_only",operation=="runtime_invoke"?"idempotent_with_invocation_id":mutation?"non_idempotent":"idempotent",
      operation=="runtime_invoke"||operation=="runtime_stop",operation.starts_with("runtime_")?"target_host_permission":"none",
      operation=="runtime_invoke"?"engop:symphony:srrapiv.runtime.attempts":mutation?"engop:symphony:srrapiv.runtime.status":"","supported","freezing"});
  }
  engine::validate_operation_specs(ops);return ops;
}
Json seal(Json object,const char *key) { object[key]=engine::tagged_sha256(object.dump());return object; }
Json descriptor() {
  return seal({{"protocol",engine::descriptor_protocol_v2},{"format_version",2},
    {"module_id","srrapiv-runtime-cpp"},{"engine_id",engine_id},{"vector_id","srrapiv"},
    {"engine_version",version},{"process_protocols",Json::array({engine::process_protocol_v1})},
    {"contract_versions",Json::array({"modules/srrapiv-runtime-cpp/SPEC.md@v1","symphony.srrapiv.request.v1","symphony.srrapiv.result.v1"})},
    {"operations",engine::administration_operation_descriptors(operations())},
    {"limits",{{"request_bytes",1048576},{"response_bytes",4194304},{"json_depth",64},{"json_values",65536},{"deadline_ahead_ms",30000}}},
    {"supported_scopes",Json::array({"user"})},{"language","C++26"},{"thermal_path","freezing"},
    {"canonical_apply_enabled",false},{"network_listener",false},{"resident_local_control","explicit_start_only"}},"descriptor_digest");
}
Json handle(const engine::Request &r) {
  const auto specs=operations();owner::private_io::require(engine::find_operation(specs,r.operation)!=nullptr,"operation.unsupported");
  owner::private_io::require(r.payload.is_object()&&r.payload.value("protocol","")=="symphony.srrapiv.request.v1");
  owner::private_io::require(r.deadline_unix_ms<=engine::unix_time_ms()+30000,"request.invalid_deadline");
  Json payload=r.payload;payload.erase("protocol");Json result;
  if(r.operation=="api_describe") {
    owner::private_io::fields(payload,{});
    owner::Vendor vendor(Json::object(),[](Json){});result=vendor.describe();
    result["runtime_build"]=owner::runtime_build_identity();
    // Native owner descriptors contain the complete version-specific schema.
    // Selection is implemented by the vendor bridge if its schema permits it.
  } else if(r.operation.starts_with("runtime_")) result=owner::runtime_operation(r.operation,payload,r.deadline_unix_ms);
  else result=owner::catalog_operation(r.operation,payload);
  return seal({{"protocol","symphony.srrapiv.result.v1"},{"operation",r.operation},{"engine_version",version},
    {"request_digest",engine::tagged_sha256(r.payload.dump())},{"result",result}},"result_digest");
}
int main(int argc,char **argv) {
  if(argc==6&&std::string(argv[1])=="--serve")return owner::serve(argv[2],argv[3],argv[4],argv[5]);
  try {
    if(argc==2) {
      if(std::string(argv[1])=="--descriptor") {std::cout<<descriptor().dump()<<'\n';return 0;}
      if(std::string(argv[1])=="--version") {std::cout<<engine_id<<' '<<version<<'\n';return 0;}
      if(std::string(argv[1])=="--help") {std::cout<<"SRRAPIV native administration: one engine-process.v1 request on stdin. Use qxctl sqv rithmic schema for contracts.\n";return 0;}
    }
    owner::private_io::require(argc==1,"argument.unsupported");
    const auto input=engine::read_bounded(std::cin,1048576);
    const auto request=engine::parse_request(input,engine_id,engine::unix_time_ms(),65536);
    try { std::cout<<engine::serialize_response(engine::success_response(request,engine_id,version,handle(request)),65536);return 0; }
    catch(const engine::Error &error) {
      std::cout<<engine::serialize_response(engine::error_response(request.request_id,request.correlation_id,request.operation,engine_id,version,error.code(),"SRRAPIV operation refused; reconcile uncertain external attempts"));return error.exit_status();
    } catch(...) {
      std::cout<<engine::serialize_response(engine::error_response(request.request_id,request.correlation_id,request.operation,engine_id,version,"srrapiv.request.invalid","SRRAPIV operation failed; no automatic retry"));return 3;
    }
  } catch(const engine::Error &error) {
    std::cout<<engine::serialize_response(engine::error_response("unavailable","unavailable","unavailable",engine_id,version,error.code(),"SRRAPIV request refused"));return error.exit_status();
  } catch(...) {return 5;}
}
