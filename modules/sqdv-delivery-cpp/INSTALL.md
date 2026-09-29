# SQDV C++26 Library Installation

`sqdv-delivery-cpp` is an independently selectable, library-only `0.3.0-dev` package. Its CMake package version is `0.3.0`; the public target is `Symphony::SqdvDelivery`. It requires CMake 3.30 or newer, C++26, and one single-configuration generator. The initial verified development target is macOS `amd64` with AppleClang 21; consumer compiler/runtime compatibility remains explicit. Retained delivery uses the SQPV backend selected for macOS local APFS. Direct delivery is a trusted same-process interface; the package supplies no network transport or remote destination acknowledgement.

Exact dependencies are SQPV local store `0.2.0-dev`, SQMV metadata `0.2.0-dev`, SQFV batch `0.3.0-dev`, and foundation `0.2.0-dev`. The default source build compiles those repository dependencies with their tests disabled and excludes their installation rules. Install dependency packages separately before configuring an installed SDK consumer. A source dependency build does not make its package present in an installation prefix.

```sh
cmake -S modules/sqdv-delivery-cpp -B /tmp/sqdv-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/sqdv-build
ctest --test-dir /tmp/sqdv-build --output-on-failure
cmake --install /tmp/sqdv-build --prefix /tmp/sqv-prefix
```

To build this module against already installed exact dependencies, add `-DSYMPHONY_SQDV_USE_INSTALLED_DEPENDENCIES=ON -DCMAKE_PREFIX_PATH=/tmp/sqv-prefix` at configuration. The exported package uses exact `find_dependency` versions and propagates the C++26 compiler requirement.

The installation contains a versioned static archive, two public headers, four CMake package/export files, six contract documents, AGPL license, and immutable receipt-v2. The receipt owns exactly 14 files, identifies vector `sqdv`, has `component_kind=module`, `engine_id=null`, and no process entry points. Installation starts no service. `GNUInstallDirs` may select normalized relative library/include paths inside the prefix; absolute paths, traversal, and existing symlink components are rejected.

The current Apple archiver may vary unused symbol-table padding across builds even when timestamps are cleared. Archive byte-for-byte reproducibility is therefore not a package guarantee; each receipt binds the exact installed archive.

Copy `tests/sdk-consumer/CMakeLists.txt` and `tests/sdk-consumer/main.cpp` outside the checkout, then configure with the prefix containing all exact dependency packages:

```sh
cmake -S /tmp/sqdv-consumer -B /tmp/sqdv-consumer-build -DCMAKE_PREFIX_PATH=/tmp/sqv-prefix
cmake --build /tmp/sqdv-consumer-build
/tmp/sqdv-consumer-build/consumer
```

The consumer requests `find_package(SymphonySqdvDelivery 0.3.0 EXACT CONFIG)` and verifies `SymphonySqdvDelivery_RELEASE_VERSION` is `0.3.0-dev`. It uses the exported C++26 requirement without setting its own language standard.

Its public API checks cover identity-bound resume refusal with an existing output session preserved, exact retained/live cutover and reopened-source replay, independent processing acknowledgements and payload byte credit, bounded unacknowledged deliveries, and terminal sequence exhaustion. Retained fixtures use fresh private temporary APFS directories.

Guarded removal uses the same prefix that received installation:

```sh
cmake -DINSTALL_PREFIX=/tmp/sqv-prefix -P /tmp/sqdv-build/uninstall.cmake
```

Retain the generated uninstaller, shared helper scripts, and configured identity sidecar together when moving its build directory. Removing the source checkout does not prevent removal. The build target `uninstall-sqdv-delivery-cpp` instead uses the prefix chosen at configuration.

Build-local lifecycle operations require an administrator-controlled prefix without concurrent filesystem mutation. They canonicalize the selected root (including `/tmp` aliases), reject symlinks and special files on relevant paths, strictly parse and verify the receipt digest/identity, and verify every present owned file before deleting content. CMake checks provide no descriptor-relative defense against hostile concurrent directory replacement. A qxctl-administered shared root is refused in favor of its authenticated lifecycle. Same-version receipt overwrite is refused; other exact package versions and dependency packages are preserved by removal.
