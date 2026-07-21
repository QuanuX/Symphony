#pragma once

#include "symphony/knowledge/engine/json.hpp"

#include <cstddef>
#include <initializer_list>
#include <string>
#include <string_view>

namespace symphony::knowledge::sclv::provider {

inline constexpr const char* evidence_protocol = "symphony.knowledge.provider-evidence.v1";
inline constexpr const char* adapter_version = "0.1.0-dev";

[[nodiscard]] bool safe_token(std::string_view value, std::size_t maximum = 128U);
[[nodiscard]] bool bounded_text(std::string_view value, std::size_t maximum = 4096U, bool allow_newlines = false);
[[nodiscard]] bool strict_utc(std::string_view value);
[[nodiscard]] bool tagged_digest(std::string_view value);
void require_exact_fields(const engine::Json& value, std::initializer_list<const char*> fields);
void validate_change_request(const engine::Json& value);
void validate_ratification(const engine::Json& value, bool require_asserted);
void validate_evidence(const engine::Json& value);
[[nodiscard]] engine::Json descriptor(const std::string& adapter_id);
[[nodiscard]] engine::Json normalize_airgap(const engine::Json& payload);

}
