# SQFV Batch C++ Install

Configure and build `modules/sqfv-batch-cpp` with CMake 3.25 or newer and a C++20 compiler. The initial supported development target is macOS `amd64` with AppleClang 21. Use a single-configuration generator and an explicit prefix. The first exact package release is `0.1.0-dev`; its CMake package version is `0.1.0`.

```sh
cmake -S modules/sqfv-batch-cpp -B /tmp/sqfv-batch-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/sqfv-batch-build
ctest --test-dir /tmp/sqfv-batch-build --output-on-failure
cmake --install /tmp/sqfv-batch-build --prefix /tmp/sqfv-prefix
```

The versioned installation contains `lib/symphony/sqfv-batch-cpp/0.1.0-dev/libsqfv-batch.a`, the public C header under a versioned include root, an exact CMake export, six contract documents, the AGPL license, and a receipt-v2 file. `GNUInstallDirs` settings may change `lib` and `include`. The receipt has `component_kind=module`, `vector_id=sqfv`, `engine_id=null`, and no entry points. Installing this package does not start or bind a runtime.

An independent C consumer can compile only against the installed public header and archive. Its project is [tests/sdk-consumer/](tests/sdk-consumer/); copy those two files outside the source tree, then configure with the exact package directory:

```sh
cmake -S /tmp/copied-sqfv-consumer -B /tmp/sqfv-consumer-build \
  -DSymphonySqfvBatch_DIR=/tmp/sqfv-prefix/lib/cmake/SymphonySqfvBatch-0.1.0-dev
cmake --build /tmp/sqfv-consumer-build
/tmp/sqfv-consumer-build/consumer
```

The C source is compiled as C11; the final executable uses the C++ linker because the installed implementation is a C++ static archive. `find_package(SymphonySqfvBatch 0.1.0 EXACT CONFIG)` and `SymphonySqfvBatch_RELEASE_VERSION` enforce the exact development package. The exported target is `Symphony::SqfvBatch`.

For an installation made with `cmake --install ... --prefix`, pass that same exact prefix to the configured uninstaller:

```sh
cmake -DINSTALL_PREFIX=/tmp/sqfv-prefix -P /tmp/sqfv-batch-build/uninstall.cmake
```

The `uninstall-sqfv-batch-cpp` build target uses the prefix selected at CMake configuration time. Both routes remove only unchanged files owned by this exact receipt. They refuse altered owned files or a shared root administered by qxctl lifecycle. Different package versions can coexist under distinct versioned paths; the exact same version cannot be installed over its receipt. The installed archive is a platform/toolchain build, not a universal binary ABI across operating systems and compilers.
