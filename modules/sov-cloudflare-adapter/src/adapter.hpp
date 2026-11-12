#pragma once
#include <symphony/knowledge/engine/json.hpp>
#include <filesystem>
#include <string>

namespace cloudflare {
using Json = symphony::knowledge::engine::Json;
namespace fs = std::filesystem;
constexpr std::size_t frame_limit = 8 * 1024 * 1024;
Json parse(const std::string& text);
std::string digest(const Json& value, const std::string& domain);
Json owner();
Json resource_preview(const Json& input);
Json project_inspect(const Json& input);
Json process(const Json& request, const fs::path& executable);
void validate(const Json& value, const Json& schema);
std::string file_digest(const fs::path& path);
}
