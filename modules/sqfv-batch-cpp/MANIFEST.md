# SQFV Batch C++ Manifest

## Identity

- Module, component, and package ID: `sqfv-batch-cpp`
- Owner: `sqfv`
- First exact development release: `0.1.0-dev`
- Package kind: `module`; library-only receipt-v2 installation
- Public ABI: C, exported from a native C++20 static archive

## Canonical Surfaces

- `modules/sqfv-batch-cpp/FEATURES.md`
- `modules/sqfv-batch-cpp/INSTALL.md`
- `modules/sqfv-batch-cpp/INTENT.md`
- `modules/sqfv-batch-cpp/MANIFEST.md`
- `modules/sqfv-batch-cpp/SKILL.md`
- `modules/sqfv-batch-cpp/SPEC.md`
- `modules/sqfv-batch-cpp/CMakeLists.txt`
- `modules/sqfv-batch-cpp/include/symphony/sqfv/batch.h`
- `modules/sqfv-batch-cpp/src/batch.cpp`
- `modules/sqfv-batch-cpp/src/batch_internal.hpp`
- `modules/sqfv-batch-cpp/src/frame.cpp`
- `modules/sqfv-batch-cpp/cmake/SymphonySqfvBatchConfig.cmake.in`
- `modules/sqfv-batch-cpp/cmake/uninstall.cmake.in`
- `modules/sqfv-batch-cpp/tests/batch_test.cpp`
- `modules/sqfv-batch-cpp/tests/frame_test.cpp`
- `modules/sqfv-batch-cpp/tests/sdk-consumer/CMakeLists.txt`
- `modules/sqfv-batch-cpp/tests/sdk-consumer/main.c`

The public header and SPEC define the source ABI and frame contract. Source and tests are implementation and focused verification surfaces. The generated CMake exports and installed receipt are exact-version package artifacts, not independent source contracts.

## Boundary

No `sqfv:` colon namespace, process-engine identity, qxctl command, provider operation, access grant, durable storage, network transport, or SSFV registration follows from this module identity. The installed receipt has `engine_id=null` and an empty `entry_points` array. Use the exact installed version; a selected static archive does not activate a background process.
