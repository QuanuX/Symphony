#pragma once
#include "symphony/knowledge/engine/protocol.hpp"
#include <set>
#include <string>
namespace symphony::knowledge::shv {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
inline constexpr auto version = "0.1.0-dev";
inline constexpr auto engine_id = "symphony-shv";
[[noreturn]] void invalid(const std::string &why);
void fields(const Json &, std::initializer_list<const char *>);
std::string text(const Json &, const char *, std::size_t maximum = 4096);
std::string ident(const Json &, const char *);
void array(const Json &, std::size_t);
Json seal(Json, const std::string &key = "digest");
void check_seal(const Json &);
void deadline(const engine::Request &);
bool date_valid(const std::string &);
void summary(const Json &);
Json coverage_default(const Json &);
Json coverage_plan(const Json &);
std::string html_text(const std::string &, const std::string &model,
                      const std::string &heading_section,
                      const std::string &field_section);
Json catalogue_build(const engine::Request &, const Json &);
void catalogue_replay(const engine::Request &, const Json &,
                      const std::string &);
Json project(const Json &);
Json descriptor();
Json handle_request(const engine::Request &);
} // namespace symphony::knowledge::shv
