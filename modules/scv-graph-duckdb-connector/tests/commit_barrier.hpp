#pragma once
namespace symphony::scv::duckdb_connector::test_support {
// Test executable only. A private inherited pipe acknowledges the reached stage;
// the parent kills this process while it is stopped, without unwinding DuckDB.
void commit_barrier(const char* point);
}
