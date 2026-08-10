#pragma once

#include "symphony/knowledge/engine/json.hpp"
#include "symphony/knowledge/engine/protocol.hpp"

#include <string>

namespace symphony::knowledge::session {

[[nodiscard]] engine::Json authority_session_capabilities();
[[nodiscard]] engine::Json handle_authority_session(const engine::Request& request);
[[nodiscard]] engine::Json validate_ssiag_authorization(
    const engine::Json& decision,
    const std::string& expected_operation,
    const std::string& tops_id,
    const std::string& expected_resource);

}
