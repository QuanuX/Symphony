# SHV Generic Graph Adapter INSTALL

Build standalone with CMake and C++26. Configure `SYMPHONY_KVE_USE_INSTALLED=ON` and an explicit `SymphonyKnowledgeVectorEngine_DIR` to use the installed 0.1.0-dev foundation; its CMake package version is `0.1.0` and the dependency request is exact. Alternatively the repository foundation may be built alongside the adapter. Use a single-configuration generator and an explicit installation prefix. Receipt v2 owns the executable, public header, static core archive, CMake package exports/configuration, schemas, licenses and documentation. Guarded `uninstall-shv-graph-adapter` removes only this receipt's unchanged owned files; it does not remove the independently installed foundation.

## Installed C++ SDK

The SDK is usable without a source checkout. Install the foundation independently, then point CMake at its exact configuration directory and the adapter's exact `lib/cmake/SymphonyShvGraphAdapter-0.1.0-dev` directory. Standard `GNUInstallDirs` overrides may change the `lib` directory.

```cmake
cmake_minimum_required(VERSION 3.25)
project(OwnGraphAdapter LANGUAGES CXX)
find_package(SymphonyShvGraphAdapter 0.1.0 EXACT REQUIRED CONFIG)
add_executable(consumer main.cpp)
target_link_libraries(consumer PRIVATE Symphony::ShvGraphAdapter)
```

The imported target propagates its versioned public include directory, C++26 requirement, static archive and `Symphony::KnowledgeVectorEngine` dependency. Include `<symphony/graph/adapter.hpp>` to implement `Adapter`, call the linkable `validate_exchange`, or use `PortableReference::roundtrip` and `query`. The installed CMake configuration depends on the installed foundation; it never fetches packages or references a repository checkout.

The CMake package version is `0.1.0`; `SymphonyShvGraphAdapter_RELEASE_VERSION` identifies the exact `0.1.0-dev` package. Select exact prefixes/configuration directories rather than an ambient newer installation. Build-platform and compiler ABI compatibility remain the consumer's responsibility; this static archive is not a universal cross-platform binary.

No SCV binary, provider account, vendor database or network service is required. The reference adapter checks the portable graph's structure and exact values. It does not authenticate a publisher, replay hardware source semantics or provide physical persistence. The executable/schema protocols are unchanged by the SDK packaging.

## Focused installed-SDK regression

`tests/sdk-consumer/` is a standalone consumer project. It uses only the installed CMake target and public headers, checks the linkable validator and virtual adapter port, and verifies roundtrip/query correspondence plus dangling-endpoint rejection. It is deliberately not a default CTest dependency on an ambient installation. For a checkout-free check, copy its `CMakeLists.txt` and `main.cpp` to a temporary directory and configure that directory with both exact package locations:

```sh
cmake -S /tmp/copied-sdk-consumer -B /tmp/shv-sdk-consumer-build \
  -DSymphonyShvGraphAdapter_DIR=/exact/adapter-prefix/lib/cmake/SymphonyShvGraphAdapter-0.1.0-dev \
  -DSymphonyKnowledgeVectorEngine_DIR=/exact/foundation-prefix/lib/cmake/SymphonyKnowledgeVectorEngine-0.1.0-dev
cmake --build /tmp/shv-sdk-consumer-build
/tmp/shv-sdk-consumer-build/consumer
```

Apple builds use `ZERO_AR_DATE=1` for the adapter archive's `ar` and `ranlib` commands so timestamps do not change the installed SDK bytes between otherwise identical builds.

## Mechanical interface release 0.2.0-dev

OWNER-INTERFACE.json declares exact metadata. INTERFACE-GENERATOR.json selects generation paths and frozen history; the shared registration-driven generator produces src/interface.generated.hpp, Go admission and CMake inventory. qxctl verifies the installed declaration through receipt-v2 and compiled admission. Existing command identities and defaults remain unchanged. Old packages remain independently selectable; artifacts and retained writers keep their exact original identity. Metadata generation is separate from native semantic handlers and independent Go result replay.

This inventory is not exhaustive. New owners can use `tools/shv-interface-codegen/EXTENDING.md` without extending a fixed generator whitelist. They must supply their own semantic contracts and qxctl integration.
