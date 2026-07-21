#pragma once

#include "symphony/knowledge/engine/json.hpp"

#include <cstdint>

namespace symphony::knowledge::sclv::provider {

[[nodiscard]] engine::Json normalize_local_git(
    const engine::Json& payload,
    std::int64_t deadline_unix_ms);

}
