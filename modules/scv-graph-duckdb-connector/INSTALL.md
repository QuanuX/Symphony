# SCV DuckDB Graph Connector Installation

This is an optional, independently installed C++26 integration adapter. Existing SCV domain-engine installations do not acquire a DuckDB dependency or change version when this package is installed.

## Build Requirements

The pinned dependency recipe supports macOS arm64/x86_64 and Linux glibc arm64/x86_64, CMake 3.25 or later, a C++26 compiler, the selected DuckDB 1.5.5 `duckdb.h` C API header and matching C++ engine shared library, and the shared `knowledge-vector-engine-cpp` foundation. The C++26 adapter calls the stable C ABI of the actual C++ DuckDB engine. The receipt-owned `DUCKDB-PROVENANCE.json` pins the selected release, commit, archive and library hashes; the standard descriptor does not carry ad hoc dependency fields. No previous SQLite setting applies.

From a source checkout, choose explicit build and install directories:

```sh
cmake -S modules/scv-graph-duckdb-connector -B /absolute/connector-build -DDUCKDB_ROOT=/absolute/verified-duckdb-1.5.5 -DDUCKDB_LIBRARY=/absolute/verified-duckdb-1.5.5/libduckdb.dylib -DCMAKE_INSTALL_PREFIX=/absolute/connector-prefix
cmake --build /absolute/connector-build
ctest --test-dir /absolute/connector-build --output-on-failure
cmake --install /absolute/connector-build
```

The preparation script `.github/scripts/prepare-duckdb-dependency.sh /absolute/fresh-duckdb` downloads the pinned official 1.5.5 platform archive, exact-commit MIT license and source archive license appendix, verifies hashes, and selects the unchanged macOS slice when applicable. Pass `-C /absolute/fresh-duckdb/dependency.cmake` to configure. Module CMake performs no downloads. The supplied header, library, license and third-party notices are all hash checked, including object architecture. The library is installed beside the executable with only `@loader_path` on macOS or `$ORIGIN` on Linux; a private build directory is not a runtime fallback. [Official release](https://github.com/duckdb/duckdb/releases/tag/v1.5.5).

The default source build links the shared foundation from this checkout. An already installed compatible foundation can be selected with `-DSYMPHONY_KVE_USE_INSTALLED=ON` and an explicit `CMAKE_PREFIX_PATH`. Process acceptance tests are compiled C++ executables when `BUILD_TESTING` is enabled. The connector and its tests require no Python interpreter.

## Exact Installed Files

The versioned executable is `libexec/symphony/scv-graph-duckdb-connector/0.2.0-dev/symphony-scv-graph-duckdb-connector`. Contracts are under `share/doc/symphony/scv-graph-duckdb-connector/0.2.0-dev`; the closed schema is under `share/symphony/schemas/scv-graph-duckdb-connector/0.2.0-dev`. The receipt is `share/symphony/receipts/scv-graph-duckdb-connector/0.2.0-dev/install-receipt.json`, with component kind `adapter`, vector `scv`, and the process-v1 entry point. Inspect its exact owned-file list rather than assuming ownership from a directory name.

No active executable alias, background process, database, source/graph selection or network service is installed. Reinstalling over different receipt-owned bytes is not an upgrade mechanism; select another exact release/prefix according to the package lifecycle contract.

## Uninstall and Caller Data

The `uninstall-scv-graph-duckdb-connector` build target removes only the selected receipt-owned package files after the installer's ownership checks. Caller databases and their journal/import state are outside package ownership and must remain intact. Uninstalling does not grant permission to delete them or to remove another connector release. The colocated dependency library is receipt owned; external source/dependency inputs and unrelated system installations are not.

Installed dependency mode (`-DSYMPHONY_KVE_USE_INSTALLED=ON`) requires the exact `SymphonyKnowledgeVectorEngine` CMake package `0.2.0` from the independently installed `knowledge-vector-engine-cpp` package `0.2.0-dev`. Select its explicit configuration directory with `SymphonyKnowledgeVectorEngine_DIR`; a missing or different version fails configuration without adding foundation source. The module's own release and protocol versions are unchanged.

## Platform and installed qualification

The required sprint profiles are declared in `cmake/mvp-platform-profiles.json`: macOS 15 ARM64, Ubuntu 24.04/26.04 x86_64 and Ubuntu 24.04 ARM64. Native Windows is unsupported. The optional Intel recipe remains independently identifiable. Ubuntu ARM64 is accepted for this sprint; no physical DGX Spark or GPU qualification is claimed. `tests/duckdb-connectors` copies process consumers outside the source tree, imports the installed foundation, tests six native SIGKILL transaction boundaries per connector using installed production recovery, and verifies adjacent loading, missing-library refusal and exact receipt-owned removal. Caller data remains caller owned. These process tests do not simulate power loss or establish filesystem durability.

The source archive license appendix contains 20 third-party notices and is installed as `DuckDB-THIRD-PARTY-LICENSES.txt` alongside `DuckDB-LICENSE.MIT`; both are receipt owned. See `DUCKDB-PROVENANCE.json` for exact archive/library/header/license hashes, upstream ABI observations and the documented Linux glibc 2.28 minimum.

Loader evidence applies to the recorded clean process environment. Standard OS loader override variables are not a security boundary; callers must preserve their selected package and environment. The gate clears loader override variables and checks the observed adjacent library path.
