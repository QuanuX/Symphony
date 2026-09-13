# SCV DuckDB Graph Connector Installation

This is an optional, independently installed C++26 integration adapter. Existing SCV domain-engine installations do not acquire a DuckDB dependency or change version when this package is installed.

## Build Requirements

The pinned reference recipe supports macOS x86_64, CMake 3.25 or later, a C++26 compiler, the selected DuckDB 1.5.5 `duckdb.h` C API header and matching C++ engine shared library, and the shared `knowledge-vector-engine-cpp` foundation. The C++26 adapter calls the stable C ABI of the actual C++ DuckDB engine. The receipt-owned `DUCKDB-PROVENANCE.json` pins the selected release, commit, archive and library hashes; the standard descriptor does not carry ad hoc dependency fields. No previous SQLite setting applies.

From a source checkout, choose explicit build and install directories:

```sh
cmake -S modules/scv-graph-duckdb-connector -B /absolute/connector-build -DDUCKDB_ROOT=/absolute/verified-duckdb-1.5.5 -DDUCKDB_LIBRARY=/absolute/verified-duckdb-1.5.5/libduckdb.dylib -DCMAKE_INSTALL_PREFIX=/absolute/connector-prefix
cmake --build /absolute/connector-build
ctest --test-dir /absolute/connector-build --output-on-failure
cmake --install /absolute/connector-build
```

The current CMake recipe rejects other platforms and architectures. A separately verified dependency recipe would be needed before claiming another platform works. CMake performs no dependency download or global installation. The recorded reference dependency is the official 1.5.5 macOS universal archive, SHA256 `7b5b8915cc382d0708636fe6385c0cdad5a61c9ff8ba2638b3e2141640783155`, with the x86_64 library extracted by Apple `lipo`; that platform evidence must not be relabeled as a verified Linux build. The exact selected library is installed beside the executable and loaded using `@loader_path` on macOS. [Official release](https://github.com/duckdb/duckdb/releases/tag/v1.5.5).

The default source build links the shared foundation from this checkout. An already installed compatible foundation can be selected with `-DSYMPHONY_KVE_USE_INSTALLED=ON` and an explicit `CMAKE_PREFIX_PATH`. Python is needed for the process acceptance test when `BUILD_TESTING` is enabled; it is not a runtime dependency of the connector.

## Exact Installed Files

The versioned executable is `libexec/symphony/scv-graph-duckdb-connector/0.2.0-dev/symphony-scv-graph-duckdb-connector`. Contracts are under `share/doc/symphony/scv-graph-duckdb-connector/0.2.0-dev`; the closed schema is under `share/symphony/schemas/scv-graph-duckdb-connector/0.2.0-dev`. The receipt is `share/symphony/receipts/scv-graph-duckdb-connector/0.2.0-dev/install-receipt.json`, with component kind `adapter`, vector `scv`, and the process-v1 entry point. Inspect its exact owned-file list rather than assuming ownership from a directory name.

No active executable alias, background process, database, source/graph selection or network service is installed. Reinstalling over different receipt-owned bytes is not an upgrade mechanism; select another exact release/prefix according to the package lifecycle contract.

## Uninstall and Caller Data

The `uninstall-scv-graph-duckdb-connector` build target removes only the selected receipt-owned package files after the installer's ownership checks. Caller databases and their journal/import state are outside package ownership and must remain intact. Uninstalling does not grant permission to delete them or to remove another connector release. The colocated dependency library is receipt owned; external source/dependency inputs and unrelated system installations are not.
