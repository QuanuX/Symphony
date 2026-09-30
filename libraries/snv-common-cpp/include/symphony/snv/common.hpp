#pragma once
#include <symphony/knowledge/engine/json.hpp>
#include <cstdint>
#include <functional>
#include <string>
#include <string_view>
#include <vector>

namespace symphony::snv {
using Json = symphony::knowledge::engine::Json;
struct Operation {
  std::string name, input_protocol, output_protocol;
  std::vector<std::string> interactions{"inspect"};
  std::string mutability{"read_only"}, idempotency{"idempotent"};
  bool expected_state_required{false};
  std::string authorization_requirement{"none"}, recovery_operation_id{"none"};
  bool operator==(const Operation&) const = default;
};
inline constexpr std::size_t max_json_values = 262144;
inline constexpr std::size_t max_records = 2048;
using Handler = std::function<Json(std::string_view, const Json&, std::int64_t)>;
Json descriptor(std::string_view owner, std::string_view version,
                const std::vector<Operation>& operations, const Json& embedded = Json::array());
int run(int argc, char** argv, std::string_view owner, std::string_view version,
        const std::vector<Operation>& operations, const Handler& handler,
        const Json& embedded = Json::array());
}
