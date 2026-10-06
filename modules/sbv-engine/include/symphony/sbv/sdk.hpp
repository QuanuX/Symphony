#pragma once
#include <memory>
#include <stdexcept>
#include <string>
#include <string_view>
#include <symphony/sbv/sdk.h>
namespace symphony::sbv::sdk {
inline constexpr std::string_view contract = "symphony.sbv.sdk.v1";
struct Response {
  int status;
  std::string json;
};
inline Response process(std::string_view request) {
  char *buffer = nullptr;
  std::size_t size = 0;
  const auto status = symphony_sbv_sdk_process_v1(
      request.data(), request.size(), &buffer, &size);
  std::unique_ptr<char, decltype(&symphony_sbv_sdk_release_v1)> owned(
      buffer, symphony_sbv_sdk_release_v1);
  if (!owned)
    throw std::runtime_error(
        "SBV SDK boundary unavailable; inspect output artifacts before retry");
  return {status, std::string(buffer, size)};
}
} // namespace symphony::sbv::sdk
