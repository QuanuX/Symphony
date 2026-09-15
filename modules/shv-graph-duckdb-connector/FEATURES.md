# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/shv-graph-duckdb-connector/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Source evolution closure.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "Current owner surface routing.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Official publication remains separately authorized.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [],
      "evidence": [
        "tests/connector_test.py exercises durable replay, corruption and scope boundaries; process interruption evidence is recorded separately.",
        "SHV-19 focused inventory, legacy-reader and independent consumer conformance."
      ],
      "feature_id": "ssfv:symphony:shv-graph-duckdb-connector",
      "how": "C++26 and pinned DuckDB own structural persistence, scoped inventory and caller-selected exact-revision transfer planning. qxctl independently checks structural correspondence and owns durable copy execution and recovery.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns durable structural graph storage, verified inventory and revision-bound transfer planning."
        },
        {
          "language": "CMake",
          "role": "Builds and receipts exact independent package."
        }
      ],
      "implementation_paths": [
        "modules/shv-graph-duckdb-connector/CMakeLists.txt",
        "modules/shv-graph-duckdb-connector/src/connector.cpp",
        "modules/shv-graph-duckdb-connector/src/main.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No semantic interpretation, publisher authentication, canonical head selection or graph database default."
      ],
      "owner_contract": "modules/shv-graph-duckdb-connector/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Uses bounded strict process and canonical digest mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        },
        {
          "rationale": "Reuses exact generic graph structural validation without importing semantic authority.",
          "target_feature_id": "ssfv:symphony:shv-graph-adapter",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/shv-graph-duckdb-connector",
      "status": "experimental",
      "title": "SHV durable DuckDB graph storage",
      "what": "Prepare, atomically commit, inspect, query and export scoped immutable generic graph snapshots. Enumerates verified scoped inventory with revision-bound pagination and retained writer identities.",
      "when": "Runs only on explicit native process or qxctl invocation.",
      "where": "Explicit caller-owned private database root and exact installed connector.",
      "who": "Humans and agents using exact qxctl adapter selection; structural transport conveys no hardware authority.",
      "why": "Preserve graph evidence durably without selecting a canonical catalogue head."
    }
  ],
  "source_scope": "modules/shv-graph-duckdb-connector"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
