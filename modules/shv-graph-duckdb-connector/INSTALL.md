# Independent installation

Build this module with CMake 3.25+, C++26, explicit DUCKDB_ROOT and DUCKDB_LIBRARY for the hash-pinned 1.5.5 macOS x86_64 dependency. No network acquisition occurs. Use a new CMAKE_INSTALL_PREFIX; install produces a v2 receipt owning the executable, adjacent DuckDB library, schemas, documentation and licenses. The uninstall-shv-graph-duckdb-connector target validates receipt ownership and retains unrelated caller files. The source build reuses the exact generic adapter 0.1 structural contract. Existing SCV/SHV installations are not changed.
