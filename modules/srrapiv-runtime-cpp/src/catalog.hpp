#pragma once

#include <nlohmann/json.hpp>
#include <string>

namespace symphony::srrapiv {
using Json = nlohmann::json;

// Pure, bounded owner operations. Neither a valid binding nor supplied
// qualification assertions authorize provider activity or prove live readiness.
[[nodiscard]] Json catalog_operation(const std::string& operation, const Json& input);
}
