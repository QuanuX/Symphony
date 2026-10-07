#pragma once
#include <cstdint>
#include <string>
#include <symphony/knowledge/engine/json.hpp>
#include <vector>

namespace symphony::sbv {
using Json = knowledge::engine::Json;
inline constexpr auto version = "0.22.0-dev";
inline constexpr auto result_protocol = "symphony.sbv.result.v1";
// Owns no process-global dataset cache. Explicit dataset_load starts a
// companion native host with caller-controlled lifecycle. Callers may invoke independent jobs
// concurrently. Paths must be absolute, traverse no symlinks and name private
// local artifacts. A run creates exactly one new result; existing files are
// never replaced.
Json dispatch(const std::string &operation, const Json &input,
              std::int64_t deadline_ms);
Json descriptor();
// Shared native contract used by direct SDK clients and the process boundary.
void validate_result(const Json &result);
Json seal_result(Json result);

// Capability seam, not a claim that a device implementation is present.
// No pointer may outlive owner; strides are bytes; completion must be awaited
// before owner/release or crossing streams. No implicit host/device transfer.
struct TensorView {
  const void *data = nullptr;
  std::uint64_t bytes = 0;
  std::string dtype, device, owner, stream, completion;
  std::vector<std::uint64_t> shape, byte_strides;
};
} // namespace symphony::sbv
