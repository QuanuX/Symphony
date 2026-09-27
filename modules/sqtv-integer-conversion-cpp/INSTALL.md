# SQTV C++26 Library Installation

`sqtv-integer-conversion-cpp` is an independently selectable, library-only `0.1.0-dev` package. Its CMake package version is `0.1.0`; the public target is `Symphony::SqtvIntegerConversion`. It requires CMake 3.30 or newer, C++26, and one single-configuration generator. The initial verified development target is macOS `amd64` with AppleClang 21; consumer compiler/runtime compatibility remains explicit.

Exact runtime dependencies are foundation `0.2.0-dev`, SQFV batch `0.2.0-dev`, and SQMV metadata `0.1.0-dev`. The default source build compiles those repository dependencies with their tests disabled and excludes their installation rules. Native composition tests additionally build SQPV and SQDV `0.1.0-dev`; those are not exported runtime dependencies. Install dependency packages separately before configuring an installed SDK consumer. A source dependency build does not make its package present in an installation prefix.

```sh
cmake -S modules/sqtv-integer-conversion-cpp -B /tmp/sqtv-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/sqtv-build
ctest --test-dir /tmp/sqtv-build --output-on-failure
cmake --install /tmp/sqtv-build --prefix /tmp/sqv-prefix
```

To build this module against already installed exact dependencies, add `-DSYMPHONY_SQTV_USE_INSTALLED_DEPENDENCIES=ON -DCMAKE_PREFIX_PATH=/tmp/sqv-prefix` at configuration. Use `-DBUILD_TESTING=OFF` for a library build requiring only the three runtime dependencies; enabled native tests additionally require installed SQPV/SQDV. The exported package uses exact `find_dependency` versions and propagates the C++26 compiler requirement.

The installation contains a versioned static archive, public header, four CMake package/export files, six contract documents, AGPL license, and immutable receipt-v2. The receipt owns exactly 13 files, identifies vector `sqtv`, has `component_kind=module`, `engine_id=null`, and no process entry points. Installation starts no service. `GNUInstallDirs` may select normalized relative library/include paths inside the prefix; absolute paths, traversal, and existing symlink components are rejected.

The current Apple archiver may vary unused symbol-table padding across builds even when timestamps are cleared. Archive byte-for-byte reproducibility is therefore not a package guarantee; each receipt binds the exact installed archive.

Copy `tests/sdk-consumer/CMakeLists.txt` and `tests/sdk-consumer/main.cpp` outside the checkout, then configure with the prefix containing all exact dependency packages:

```sh
cmake -S /tmp/sqtv-consumer -B /tmp/sqtv-consumer-build -DCMAKE_PREFIX_PATH=/tmp/sqv-prefix
cmake --build /tmp/sqtv-consumer-build
/tmp/sqtv-consumer-build/consumer
```

The consumer requests `find_package(SymphonySqtvIntegerConversion 0.1.0 EXACT CONFIG)` and verifies `SymphonySqtvIntegerConversion_RELEASE_VERSION` is `0.1.0-dev`. It uses the exported C++26 requirement without setting its own language standard.

Guarded removal uses the same prefix that received installation:

```sh
cmake -DINSTALL_PREFIX=/tmp/sqv-prefix -P /tmp/sqtv-build/uninstall.cmake
```

Retain the generated uninstaller, shared helper scripts, and configured identity sidecar together when moving its build directory. Removing the source checkout does not prevent removal. The build target `uninstall-sqtv-integer-conversion-cpp` instead uses the prefix chosen at configuration.

Build-local lifecycle operations require an administrator-controlled prefix without concurrent filesystem mutation. They canonicalize the selected root (including `/tmp` aliases), reject symlinks and special files on relevant paths, strictly parse and verify the receipt digest/identity, and verify every present owned file before deleting content. CMake checks provide no descriptor-relative defense against hostile concurrent directory replacement. A qxctl-administered shared root is refused in favor of its authenticated lifecycle. Same-version receipt overwrite is refused; other exact package versions and dependency packages are preserved by removal.

Native payloads are explicitly declared dense integers under the module SPEC; conversion performs no provider parsing.
