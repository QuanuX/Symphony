#include <symphony/snv/common.hpp>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/knowledge/engine/operation.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <algorithm>
#include <iostream>

namespace symphony::snv {
namespace engine = symphony::knowledge::engine;
Json descriptor(std::string_view owner, std::string_view version,
                const std::vector<Operation>& operations, const Json& embedded) {
  std::vector<engine::OperationSpec> specs;
  for (const auto& op : operations) {
    auto suffix = op.name;
    std::replace(suffix.begin(), suffix.end(), '_', '-');
    specs.push_back({"engop:symphony:" + std::string(owner) + "." + suffix,
      op.name,"implemented",op.mutability=="canonical_mutation",true,
      {"ssfv:symphony:"+std::string(owner)+"-engine"},op.interactions,
      "qxctl_required",op.input_protocol,op.output_protocol,op.mutability,
      op.idempotency,op.expected_state_required,op.authorization_requirement,
      op.recovery_operation_id=="none"?"":op.recovery_operation_id,"supported","freezing"});
  }
  engine::validate_operation_specs(specs);
  Json result{{"protocol",engine::descriptor_protocol_v2},{"format_version",2},
    {"module_id",std::string(owner)+"-engine"},{"engine_id","symphony-"+std::string(owner)},
    {"vector_id",owner},{"engine_version",version},
    {"process_protocols",Json::array({engine::process_protocol_v1})},
    {"contract_versions",Json::array({"modules/"+std::string(owner)+"-engine/SPEC.md@v1"})},
    {"operations",engine::administration_operation_descriptors(specs)},
    {"embedded_dependencies",embedded},
    {"limits",{{"request_bytes",engine::Limits::max_request_bytes},
               {"response_bytes",engine::Limits::max_response_bytes},
               {"json_depth",engine::Limits::max_json_depth},{"json_values",max_json_values},
               {"records",max_records},{"deadline_ahead_ms",engine::Limits::max_deadline_ahead_ms}}},
    {"supported_scopes",Json::array({"user"})},{"language","C++26"},
    {"thermal_path","freezing"},{"canonical_apply_enabled",false},
    {"session_mutation_enabled",false},{"network_listener",false}};
  for (const auto& op : operations) {
    result["contract_versions"].push_back(op.input_protocol);
    result["contract_versions"].push_back(op.output_protocol);
  }
  result["descriptor_digest"] = engine::tagged_sha256(result.dump());
  return result;
}
int run(int argc,char** argv,std::string_view owner,std::string_view version,
        const std::vector<Operation>& operations,const Handler& handler,const Json& embedded) {
  const std::string engine_id="symphony-"+std::string(owner);
  try {
    if(argc==2) {
      const std::string_view flag=argv[1];
      if(flag=="--descriptor") { std::cout<<descriptor(owner,version,operations,embedded).dump()<<'\n'; return 0; }
      if(flag=="--version") { std::cout<<engine_id<<' '<<version<<'\n'; return 0; }
      if(flag=="--help") { std::cout<<engine_id<<": one bounded engine-process.v1 request on stdin; --descriptor, --version\n"; return 0; }
    }
    if(argc!=1) throw engine::Error("invocation.arguments","unsupported invocation",2);
    const auto input=engine::read_bounded(std::cin,engine::Limits::max_request_bytes);
    const auto request=engine::parse_request(input,engine_id,engine::unix_time_ms(),max_json_values);
    try {
      Json result;
      if(request.operation=="descriptor") {
        if(!request.payload.is_object()||!request.payload.empty()) throw engine::Error("descriptor.input","descriptor requires empty payload",2);
        result=descriptor(owner,version,operations,embedded);
      } else result=handler(request.operation,request.payload,request.deadline_unix_ms);
      if(engine::unix_time_ms()>request.deadline_unix_ms) throw engine::Error("deadline.exceeded","operation deadline exceeded",3);
      std::cout<<engine::serialize_response(engine::success_response(request,engine_id,std::string(version),result),max_json_values)<<'\n';
      return 0;
    } catch(const engine::Error& e) {
      std::cout<<engine::serialize_response(engine::error_response(request.request_id,request.correlation_id,request.operation,engine_id,std::string(version),e.code(),e.what()),max_json_values)<<'\n';
      return e.exit_status();
    } catch(...) {
      // The envelope is admitted already: retain its routing identity even if
      // an owner or SDK consumer throws an unexpected exception. Exception
      // text can contain supplied evidence, so return only the fixed message.
      std::cout<<engine::serialize_response(engine::error_response(request.request_id,request.correlation_id,request.operation,engine_id,std::string(version),"internal.failure","bounded operation failed"),max_json_values)<<'\n';
      return 1;
    }
  } catch(const engine::Error& e) {
    std::cout<<engine::serialize_response(engine::error_response("unavailable","unavailable","unavailable",engine_id,std::string(version),e.code(),e.what()))<<'\n';
    return e.exit_status();
  } catch(...) {
    std::cout<<engine::serialize_response(engine::error_response("unavailable","unavailable","unavailable",engine_id,std::string(version),"internal.failure","bounded operation failed"))<<'\n';
    return 1;
  }
}
}
