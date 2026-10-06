#include <cstdlib>
#include <cstring>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/limits.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/sbv/sbv.hpp>
#include <symphony/sbv/sdk.h>
namespace e = symphony::knowledge::engine;
namespace s = symphony::sbv;
extern "C" {
uint32_t symphony_sbv_sdk_abi_v1(void) noexcept { return 1; }
const char *symphony_sbv_sdk_version_v1(void) noexcept { return s::version; }
void symphony_sbv_sdk_release_v1(void *response) noexcept {
  std::free(response);
}
int symphony_sbv_sdk_process_v1(const char *input, size_t size, char **output,
                                size_t *output_size) noexcept {
  if (output)
    *output = nullptr;
  if (output_size)
    *output_size = 0;
  if (!output || !output_size)
    return 64;
  if (!input && size)
    return 64;
  try {
    std::string response;
    int status = 0;
    try {
      if (size > e::Limits::max_request_bytes)
        throw e::Error("request.too_large", "request exceeds byte limit", 2);
      const auto request =
          e::parse_request(size ? std::string(input, size) : std::string{},
                           "symphony-sbv", e::unix_time_ms());
      try {
        if (request.operation == "descriptor" &&
            (!request.payload.is_object() || !request.payload.empty()))
          throw e::Error("descriptor.input", "empty payload required", 2);
        auto result = request.operation == "descriptor"
                          ? s::descriptor()
                          : s::dispatch(request.operation, request.payload,
                                        request.deadline_unix_ms);
        response = e::serialize_response(
            e::success_response(request, "symphony-sbv", s::version, result));
      } catch (const e::Error &error) {
        status = error.exit_status();
        response = e::serialize_response(e::error_response(
            request.request_id, request.correlation_id, request.operation,
            "symphony-sbv", s::version, error.code(),
            "SBV SDK operation failed; inspect output artifacts before retry"));
      } catch (...) {
        status = 4;
        response = e::serialize_response(e::error_response(
            request.request_id, request.correlation_id, request.operation,
            "symphony-sbv", s::version, "sbv.failure",
            "SBV SDK operation failed; inspect output artifacts before retry"));
      }
    } catch (const e::Error &error) {
      status = error.exit_status();
      response = e::serialize_response(e::error_response(
          "unavailable", "unavailable", "unavailable", "symphony-sbv",
          s::version, error.code(), "SBV SDK request rejected"));
    } catch (...) {
      status = 4;
      response = e::serialize_response(e::error_response(
          "unavailable", "unavailable", "unavailable", "symphony-sbv",
          s::version, "sbv.failure",
          "SBV SDK request failed; inspect output artifacts before retry"));
    }
    auto *buffer = static_cast<char *>(std::malloc(response.size() + 1));
    if (!buffer)
      return 70;
    std::memcpy(buffer, response.data(), response.size());
    buffer[response.size()] = '\0';
    *output = buffer;
    *output_size = response.size();
    return status;
  } catch (...) {
    return 70;
  }
}
}
