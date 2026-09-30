#pragma once
#include <symphony/snv/common.hpp>
namespace symphony::snv::snv {
inline constexpr char engine_id[] = "symphony-snv";
inline constexpr char version[] = "0.1.0-dev";
std::vector<Operation> operations();
Json handle(std::string_view operation,const Json& payload,std::int64_t deadline_unix_ms);
Json embedded_dependencies();
}
