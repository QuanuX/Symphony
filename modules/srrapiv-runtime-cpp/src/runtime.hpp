#pragma once
#include <nlohmann/json.hpp>
#include <cstdint>
#include <string>
namespace symphony::srrapiv {
using Json = nlohmann::json;
Json runtime_operation(const std::string &, const Json &, std::int64_t deadline_ms);
Json runtime_build_identity();
int serve(const std::string &directory, const std::string &config_file,
          const std::string &generation, const std::string &config_digest);
}
