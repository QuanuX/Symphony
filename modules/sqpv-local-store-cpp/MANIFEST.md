# SQPV Local Store C++ Manifest

## Identity

- Module, component and package: `sqpv-local-store-cpp`
- Owner: `sqpv`; current development release: `0.2.0-dev`
- Native C++26 static library; library-only receipt-v2 installation
- Exact dependencies: SQMV metadata 0.2.0-dev, SQFV batch 0.3.0-dev,
  knowledge-vector C++ foundation 0.2.0-dev

## Canonical Surfaces

- `modules/sqpv-local-store-cpp/INTENT.md`
- `modules/sqpv-local-store-cpp/MANIFEST.md`
- `modules/sqpv-local-store-cpp/SPEC.md`
- `modules/sqpv-local-store-cpp/SKILL.md`
- `modules/sqpv-local-store-cpp/FEATURES.md`
- `modules/sqpv-local-store-cpp/INSTALL.md`
- `modules/sqpv-local-store-cpp/CMakeLists.txt`
- `modules/sqpv-local-store-cpp/include/symphony/sqpv/local_store.hpp`
- `modules/sqpv-local-store-cpp/src/local_store.cpp`
- `modules/sqpv-local-store-cpp/src/async_store.cpp`
- `modules/sqpv-local-store-cpp/include/symphony/sqpv/async_store.hpp`
- `modules/sqpv-local-store-cpp/tests/async_store_test.cpp`
- `modules/sqpv-local-store-cpp/cmake/SymphonySqpvLocalStoreConfig.cmake.in`
- `modules/sqpv-local-store-cpp/cmake/uninstall.cmake.in`
- `modules/sqpv-local-store-cpp/tests/hook.hpp`
- `modules/sqpv-local-store-cpp/tests/local_store_test.cpp`
- `modules/sqpv-local-store-cpp/tests/package_lifecycle_test.cmake`
- `modules/sqpv-local-store-cpp/tests/sdk-consumer/CMakeLists.txt`
- `modules/sqpv-local-store-cpp/tests/sdk-consumer/main.cpp`

## Extent

One exact local APFS append/read/recovery module is implemented. The installed
receipt has no executable entry point and no engine identity. Private crash/fault
hooks exist only in the separately instrumented test archive and are absent from
the installed production archive. The SPEC defines storage interpretation;
installation alone creates no user store or active process.

No qxctl surface, universal backend choice, data deletion, provider connection,
replication, power-loss certification or `sqpv:` namespace follows from this module.
