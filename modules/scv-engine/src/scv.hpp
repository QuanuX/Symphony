#pragma once

#include "symphony/knowledge/engine/protocol.hpp"

#include <string>

namespace symphony::knowledge::scv {
using engine::Json;
#ifndef SYMPHONY_SCV_VERSION
#error "SCV package identity must be supplied by its CMake build"
#endif
inline constexpr const char* version = SYMPHONY_SCV_VERSION;

[[nodiscard]] Json sealed(Json value, std::string field = "digest");
void validate_source(const Json& source, const std::string& domain);
void validate_capture(const Json& capture, const std::string& domain);
[[nodiscard]] Json handle_source(const engine::Request& request, const std::string& domain);
[[nodiscard]] Json handle_request(const engine::Request& request, const std::string& domain);
[[nodiscard]] Json descriptor(const std::string& domain);
}
