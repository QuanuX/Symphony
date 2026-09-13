# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/scv-graph-duckdb-connector/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCV owns source/evidence meaning; the separately packaged connector owns this bounded persistence mapping.",
          "reference": "knowledge/scv/GRAPH-INDEX.md",
          "vector": "scv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes current owner contracts and distributed feature files.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Completed material source changes require reviewed append-only closure.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "Future official publication follows a separate authorized release boundary.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [],
      "evidence": [
        "The new connector and independent qxctl consumer have focused producer, persistence, projection and original-owner boundary checks. Test names identify coverage; actual execution and dependency provenance belong to the increment closure."
      ],
      "feature_id": "ssfv:symphony:scv-graph-duckdb-connector",
      "how": "Uses bounded C++ process requests, pinned DuckDB transactions and independent qxctl projection verification. Semantic queries replay the exact selected SCV owner against the complete retained graph.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns bounded mechanical graph projection, local transactional persistence and retrieval through the stable DuckDB C ABI to the actual C++ DuckDB engine."
        },
        {
          "language": "CMake",
          "role": "Builds and receipts the exact independently selected connector and dependency."
        }
      ],
      "implementation_paths": [
        "modules/scv-graph-duckdb-connector/CMakeLists.txt",
        "modules/scv-graph-duckdb-connector/src/connector.cpp",
        "modules/scv-graph-duckdb-connector/src/main.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not replace SCV inference, choose a selected graph head, modify source/corpus authority, or certify source truth.",
        "Does not adopt a universal graph database, expose arbitrary SQL, claim native graph traversal performance, or implement SHV.",
        "Does not claim cross-process DuckDB writers outside its invocation lock, backend migration, or general distributed storage."
      ],
      "owner_contract": "modules/scv-graph-duckdb-connector/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 1,
      "relationships": [
        {
          "rationale": "Uses authority-free bounded process, canonical JSON and digest mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        },
        {
          "rationale": "Preserves exact SCV graph identity and requires explicit SCV owner validation at the qxctl semantic boundary.",
          "target_feature_id": "ssfv:symphony:scv-engine",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/scv-graph-duckdb-connector",
      "status": "experimental",
      "title": "SCV DuckDB graph index connector",
      "what": "Independently installed DuckDB adapter for immutable SCV graph snapshots, exact relational row projection, bounded indexed retrieval and export, and recoverable prepare/commit publication.",
      "when": "Runs only on explicit local qxctl or native process invocation; status observes durable connector state without invoking a semantic owner.",
      "where": "A versioned integration_adapter installation and a caller-selected private local index root; no active alias or service is installed.",
      "who": "Humans and agents using qxctl with explicit installations, namespace and TOPS identity; connector publication does not authorize a selected graph head.",
      "why": "Makes caller-selected graphs persistently retrievable while preserving native SCV semantic ownership and explicit backend and namespace selection."
    }
  ]
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
