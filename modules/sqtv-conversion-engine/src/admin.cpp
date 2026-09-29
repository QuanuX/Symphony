#include "../../sqtv-integer-conversion-cpp/src/limits.hpp"
#include "request.hpp"
namespace symphony::sqtv::administration {
using namespace symphony::sqv_admin;
Json administer(std::string_view, const Json &p, std::int64_t) {
  fields(p, {"protocol", "input_layout", "output_layout", "element_count",
             "limits"});
  Format input, output;
  accepted(parse_layout(text(p, "input_layout", 64), input));
  accepted(parse_layout(text(p, "output_layout", 64), output));
  const auto &l = p.at("limits");
  fields(l, {"max_elements", "max_input_bytes", "max_output_bytes"});
  Limits limits{u64(l, "max_elements"), u64(l, "max_input_bytes"),
                u64(l, "max_output_bytes")};
  require(detail::valid_limits(limits));
  const auto count = u64(p, "element_count");
  require(count > 0 && count <= limits.max_elements &&
          count <= limits.max_input_bytes / (input.bits / 8) &&
          count <= limits.max_output_bytes / (output.bits / 8));
  std::string ref;
  accepted(operation_reference(input, output, ref));
  return {{"operation_reference", ref},
          {"converter_identity", converter_identity},
          {"input_bytes", std::to_string(count * (input.bits / 8))},
          {"output_bytes", std::to_string(count * (output.bits / 8))},
          {"validation_scope", "formats_count_and_limits_only"},
          {"value_range_check", "required_at_conversion"},
          {"input_observation", "not_performed"}};
}
} // namespace symphony::sqtv::administration
