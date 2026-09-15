# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/shv-graph-adapter/SPEC.md",
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
        "Independent arbitrary-owner roundtrip/query and negative structural/process examples in tests/conformance.cpp; actual execution evidence belongs to SHV-01 closure."
      ],
      "feature_id": "ssfv:symphony:shv-graph-adapter",
      "how": "C++26 validates canonical structural graph invariants and self-seals; qxctl independently verifies the result.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns structural graph exchange and portable-reference implementation."
        },
        {
          "language": "CMake",
          "role": "Builds and receipts exact independent package."
        }
      ],
      "implementation_paths": [
        "modules/shv-graph-adapter/CMakeLists.txt",
        "modules/shv-graph-adapter/src/adapter.cpp",
        "modules/shv-graph-adapter/src/main.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No hardware meaning, source authentication, persistent database, vendor driver, network service or universal graph technology."
      ],
      "owner_contract": "modules/shv-graph-adapter/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Uses bounded strict process and canonical digest mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/shv-graph-adapter",
      "status": "experimental",
      "title": "SHV generic graph exchange adapter",
      "what": "Pure generic graph exchange, exact lossless roundtrip and bounded ID query.",
      "when": "Runs only on explicit native process or qxctl invocation.",
      "where": "An exact independent adapter installation; graph values remain in memory.",
      "who": "Humans and agents using exact qxctl adapter selection; structural transport conveys no hardware authority.",
      "why": "Supply an open backend-independent graph port without selecting a vendor database."
    }
  ],
  "source_scope": "modules/shv-graph-adapter"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
