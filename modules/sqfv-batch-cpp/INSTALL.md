# SQFV Batch C++ Install

Configure and build `modules/sqfv-batch-cpp` with CMake 3.30 or newer and a compiler that accepts the C++26 mode used by this package. The initial development target is macOS `amd64` with AppleClang 21. Use a single-configuration generator and an explicit prefix. This release is `0.2.0-dev`; its CMake package version is `0.2.0`. The preceding `0.1.0-dev` C ABI is a different exact package and is not source-compatible with this native C++ API.

```sh
cmake -S modules/sqfv-batch-cpp -B /tmp/sqfv-batch-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/sqfv-batch-build
ctest --test-dir /tmp/sqfv-batch-build --output-on-failure
cmake --install /tmp/sqfv-batch-build --prefix /tmp/sqfv-prefix
```

The versioned installation contains `lib/symphony/sqfv-batch-cpp/0.2.0-dev/libsqfv-batch.a`, the public C++26 header under a versioned include root, an exact CMake export, six contract documents, the AGPL license, and a receipt-v2 file. `GNUInstallDirs` settings may change `lib` and `include` to normalized relative paths inside the selected prefix; absolute paths and parent traversal are rejected during configuration. The receipt has `component_kind=module`, `vector_id=sqfv`, `engine_id=null`, and no entry points. Installing this package does not start or bind a runtime.

An independent C++26 consumer can compile only against the installed public header and archive. Its project is [tests/sdk-consumer/](tests/sdk-consumer/); copy those two files outside the source tree, then configure with the exact package directory:

```sh
cmake -S /tmp/copied-sqfv-consumer -B /tmp/sqfv-consumer-build \
  -DSymphonySqfvBatch_DIR=/tmp/sqfv-prefix/lib/cmake/SymphonySqfvBatch-0.2.0-dev
cmake --build /tmp/sqfv-consumer-build
/tmp/sqfv-consumer-build/consumer
```

`find_package(SymphonySqfvBatch 0.2.0 EXACT CONFIG)` and `SymphonySqfvBatch_RELEASE_VERSION` select the exact development package. The exported target is `Symphony::SqfvBatch`; it propagates the C++26 language requirement. A consumer must use the C++ API in `symphony/sqfv/batch.hpp` and match the installed platform/compiler runtime.

For an installation made with `cmake --install ... --prefix`, pass that same exact prefix to the configured uninstaller:

```sh
cmake -DINSTALL_PREFIX=/tmp/sqfv-prefix -P /tmp/sqfv-batch-build/uninstall.cmake
```

The `uninstall-sqfv-batch-cpp` build target uses the prefix selected at CMake configuration time. Both routes remove only unchanged files owned by this exact receipt. They refuse altered owned files or a shared root administered by qxctl lifecycle. Different package versions can coexist under distinct versioned paths; the exact same version cannot be installed over its receipt. The installed archive is a platform/toolchain build, not a universal binary ABI across operating systems and compilers.

Build-local lifecycle scripts require an administrator-controlled prefix with no concurrent filesystem mutation. They resolve the selected prefix explicitly (including platform aliases such as `/tmp`), reject a symlink at the prefix itself or any existing component of an owned or receipt path, and verify the receipt self-digest, configured package identity, and owned-file content before removal. These CMake path checks do not provide descriptor-relative protection against a hostile process replacing directories concurrently. The generated uninstaller uses its build-local helper and identity files; retain them together if moving the build directory. Removing or moving the original source checkout does not prevent uninstall.
