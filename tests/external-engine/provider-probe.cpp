#include <cstdint>
#include <dlfcn.h>
#include <iostream>
#include <stdexcept>
#include <string>
#include <symphony/knowledge/engine/json.hpp>
#include <symphony/sbv/provider.h>
using J = symphony::knowledge::engine::Json;
void require(bool p) {
  if (!p)
    throw std::runtime_error("provider ABI probe failed");
}
void drop(sbv_owned_bytes_v1 &b) {
  if (b.release)
    b.release(b.owner, b.data, b.size);
  b = {};
}
uint32_t cancel(void *) { return SBV_CALL_CANCEL_REQUESTED; }
int main(int argc, char **argv) {
  try {
    require(argc == 2);
    void *lib = ::dlopen(argv[1], RTLD_NOW | RTLD_LOCAL);
    require(lib);
    auto api_fn = reinterpret_cast<sbv_provider_get_api_v1>(
        ::dlsym(lib, "symphony_sbv_provider_api_v1"));
    require(api_fn);
    sbv_provider_api_v1 api{};
    api.struct_size = sizeof(api);
    api.abi_version = 2;
    require(api_fn(&api) == SBV_PROVIDER_INVALID_INPUT && api.abi_version == 2);
    api.abi_version = 1;
    api.struct_size = sizeof(api) - 1;
    require(api_fn(&api) == SBV_PROVIDER_INVALID_INPUT);
    api.struct_size = sizeof(api);
    require(api_fn(&api) == SBV_PROVIDER_OK &&
            api.role_bits == SBV_PROVIDER_MODEL);
    const std::string config = R"({"version":"1","window":"4"})";
    sbv_bytes_v1 bytes{sizeof(bytes), 1,
                       reinterpret_cast<const uint8_t *>(config.data()),
                       config.size()};
    void *instance = nullptr;
    sbv_owned_bytes_v1 out{}, error{};
    sbv_call_context_v1 context{sizeof(context), 1, 0, 0, nullptr, nullptr};
    require(api.model_create(&context, &bytes, &instance, &error) ==
                SBV_PROVIDER_OK &&
            instance);
    context.cancel_state = cancel;
    require(api.model_evaluate(instance, &context, nullptr, &out, &error) ==
                SBV_PROVIDER_CANCELLED &&
            !out.data && !error.data);
    api.model_destroy(instance);
    instance = nullptr;
    require(api.model_create(&context, &bytes, &instance, &error) ==
                SBV_PROVIDER_CANCELLED &&
            !instance);
    context.cancel_state = nullptr;
    const std::string wrong = R"({"version":"2","window":"4"})";
    bytes.data = reinterpret_cast<const uint8_t *>(wrong.data());
    bytes.size = wrong.size();
    require(api.model_create(&context, &bytes, &instance, &error) ==
                SBV_PROVIDER_INVALID_INPUT &&
            !instance);
    drop(error);
    require(api.descriptor(&context, &out, &error) == SBV_PROVIDER_OK);
    auto descriptor = J::parse(out.data, out.data + out.size);
    require(descriptor.at("version") == BETA_EXAMPLE_VERSION);
    drop(out);
    require(::dlclose(lib) == 0);
    std::cout
        << "{\"public_abi_checks\":7,\"release_acceptance_complete\":false}\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 2;
  }
}
