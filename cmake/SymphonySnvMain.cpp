#include SNV_HEADER
#ifdef SNV_GENERATED_INTERFACE
#include "interface.generated.hpp"
#include <symphony/knowledge/engine/error.hpp>
#endif
#ifndef SNV_EMBEDDED
#define SNV_EMBEDDED symphony::snv::Json::array()
#endif
int main(int argc,char** argv) {
  auto operations=symphony::snv::SNV_OWNER::operations();
#ifdef SNV_GENERATED_INTERFACE
  const auto specs=symphony::snv::SNV_OWNER::interface::interface_operations();
  std::vector<symphony::snv::Operation> admitted;
  for(const auto& op:specs)admitted.push_back({op.operation_name,*op.input_protocol,*op.output_protocol,
    op.administrative_interactions,op.mutability,op.idempotency,op.expected_state_required,
    op.authorization_requirement,op.recovery_operation_id.empty()?"none":op.recovery_operation_id});
  if(operations!=admitted) return 2;
#endif
  return symphony::snv::run(argc,argv,SNV_OWNER_TEXT,symphony::snv::SNV_OWNER::version,
    operations,symphony::snv::SNV_OWNER::handle,SNV_EMBEDDED);
}
