#include <iostream>
#include <string>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/sbv/sdk.h>
int main() {
  try {
    if (symphony_sbv_sdk_abi_v1() != 1 ||
        std::string(symphony_sbv_sdk_version_v1()) != "0.33.0-dev")
      return 64;
    const auto request =
        symphony::knowledge::engine::read_bounded(std::cin, 1048576);
    char *response = nullptr;
    std::size_t size = 0;
    const int result = symphony_sbv_sdk_process_v1(
        request.data(), request.size(), &response, &size);
    if (response) {
      std::cout.write(response, static_cast<std::streamsize>(size));
      std::cout << '\n';
      symphony_sbv_sdk_release_v1(response);
    }
    return result;
  } catch (...) {
    return 64;
  }
}
