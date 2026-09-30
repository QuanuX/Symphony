#pragma once
#include <cstddef>
#include <cstdint>
#include <string>
#include <string_view>
#include <symphony/knowledge/engine/json.hpp>
#include <symphony/knowledge/engine/operation.hpp>
#include <vector>

namespace symphony::snv::local_observer {
using Json = symphony::knowledge::engine::Json;
inline constexpr const char *version = "0.1.0-dev";
inline constexpr const char *engine_id = "symphony-snv-local-observer";
inline constexpr std::size_t max_json_values = 262144;
enum class Source { boot, cpu_present, cpu_online, meminfo };
enum class ReadStatus {
  ok,
  unavailable,
  permission_denied,
  unsupported_source,
  too_large,
  io_error
};
struct ReadResult {
  ReadStatus status;
  std::string bytes;
};
struct Platform {
  bool available;
  std::string architecture, kernel_release;
};
// Explicit SDK dependency injection for deterministic tests/custom integration.
// The finite executable only uses its private native reader and fixed sources.
class SourceReader {
public:
  virtual ~SourceReader() = default;
  virtual Platform platform() const = 0;
  virtual ReadResult read(Source source, std::size_t byte_limit,
                          std::int64_t deadline_unix_ms) = 0;
};
std::vector<symphony::knowledge::engine::OperationSpec> operations();
Json descriptor();
Json collect(const Json &request, SourceReader &reader,
             std::int64_t deadline_unix_ms);
Json handle(std::string_view operation, const Json &request,
            std::int64_t deadline_unix_ms);
} // namespace symphony::snv::local_observer
