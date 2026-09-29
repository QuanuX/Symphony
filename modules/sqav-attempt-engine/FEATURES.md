# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqav-attempt-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Source evolution and scope claims remain reviewable through SCLV.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "The exact module contract, implementation and focused evidence have independent indexed routes.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [],
      "evidence": [
        "tests/sqv-administration/validation_test.cpp",
        "tests/sqv-administration/reader_test.cpp",
        "tools/qxctl/internal/knowledgeengine/sqv_administration_test.go"
      ],
      "feature_id": "ssfv:symphony:sqav-attempt-engine",
      "how": "Exact C++26 owner operations return bounded request-bound evidence. Receipt-owned declarations and schema resources bind process admission; reusable transport helpers hold no vector semantics.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the module owner contract and focused native verification."
        },
        {
          "language": "CMake",
          "role": "Builds, receipts and guarded-uninstalls the exact process package."
        }
      ],
      "implementation_paths": [
        "modules/sqav-attempt-engine/src/admin.cpp",
        "modules/sqpv-inspection-engine/src/store_reader.cpp",
        "tools/sqv-administration-cpp/main.cpp",
        "tools/sqv-administration-cpp/request.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No live collection, provider access, credentials, paid request, store recovery, journal write, remote acknowledgment or execution grant."
      ],
      "owner_contract": "modules/sqav-attempt-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Uses the exact native owner contract or persisted format.",
          "target_feature_id": "ssfv:symphony:sqav-databento-dbn-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Uses the exact native owner contract or persisted format.",
          "target_feature_id": "ssfv:symphony:sqpv-inspection-engine",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqav-attempt-engine",
      "status": "experimental",
      "title": "SQAV attempt administration",
      "what": "Inspect persisted acquisition attempts and conservative charge ceilings without recovering records or reissuing execution capability.",
      "when": "On explicit local invocation with an exact installed release and bounded request.",
      "where": "Independently installed sqav-attempt-engine process.",
      "who": "Native process callers and qxctl SQV administration.",
      "why": "Expose SQAV administration through its owner while sharing source discovery and schema/template routes."
    }
  ],
  "source_scope": "modules/sqav-attempt-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
