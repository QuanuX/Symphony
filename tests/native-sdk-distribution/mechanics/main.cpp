#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/manifest_discovery.hpp>
#include <symphony/knowledge/engine/operation.hpp>
#include <iostream>
#include <stdexcept>
#ifdef SDK_INCLUDE_SNV_COMMON
#include <symphony/snv/common.hpp>
#endif
int main() {
  if (symphony::knowledge::engine::sha256_hex(std::string_view("abc")) !=
      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
    throw std::runtime_error("installed foundation digest differs");
#ifdef SDK_INCLUDE_SNV_COMMON
  const auto descriptor = symphony::snv::descriptor("fixture", "0.1.0-dev",
      {{"inspect", "symphony.fixture.input.v1", "symphony.fixture.result.v1"}});
  if (descriptor.at("engine_id") != "symphony-fixture" ||
      descriptor.at("canonical_apply_enabled") != false ||
      descriptor.at("network_listener") != false)
    throw std::runtime_error("installed neutral descriptor changes authority");
#endif
  std::cout << "PASS exact installed mechanics SDKs\n";
}
