# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqmv-metadata-cpp/SPEC.md",
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
        "modules/sqmv-metadata-cpp/tests/metadata_test.cpp verifies bounded admission, exact identity, failure behavior and the stated native contract.",
        "modules/sqmv-metadata-cpp/tests/sdk-consumer/main.cpp verifies the installed public C++26 interface independently of private implementation headers.",
        "modules/sqmv-metadata-cpp/tests/package_lifecycle_test.cmake verifies isolated install and guarded removal of the exact library package.",
        "The exact dependency update preserves SQM1 metadata while the six-owner pipeline verifies source, capture lineage, time and coverage evidence through conversion and retained replay."
      ],
      "feature_id": "ssfv:symphony:sqmv-metadata-cpp",
      "how": "Canonical SQM1 bytes bind exact dataset, revision, schema, layout and access assertions to producer-attributed evidence references; resolution verifies the expected content reference before publishing an immutable handle.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the module owner contract and focused native verification."
        },
        {
          "language": "CMake",
          "role": "Builds, exports, receipts, and guarded-uninstalls the exact library package."
        }
      ],
      "implementation_paths": [
        "modules/sqmv-metadata-cpp/CMakeLists.txt",
        "modules/sqmv-metadata-cpp/include/symphony/sqmv/metadata.hpp",
        "modules/sqmv-metadata-cpp/src/metadata.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No provider acquisition, external evidence dereference, source authenticity, semantic schema interpretation, access authorization, metadata catalogue, database, transformation or consumer delivery.",
        "No resident process, standalone C ABI, IPC transport, engine operation or SQV qxctl command."
      ],
      "owner_contract": "modules/sqmv-metadata-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Reuses the exact native SHA-256 helper without using knowledge-process envelopes for research payloads.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        },
        {
          "rationale": "Constructs and verifies the exact public SQFV binding through its pinned C++26 contract.",
          "target_feature_id": "ssfv:symphony:sqfv-batch-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqmv-metadata-cpp",
      "status": "experimental",
      "title": "SQMV immutable metadata binding library",
      "what": "Provides bounded immutable metadata manifests, exact content references and verified SQFV bindings.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "Inside the explicit native caller through the exact installed sqmv-metadata-cpp static library.",
      "who": "Trusted C++26 callers resolving the exact metadata needed by a selected research composition.",
      "why": "Establish an attributable metadata boundary that can be resolved once and retained without a resident catalogue or per-record lookup."
    }
  ],
  "source_scope": "modules/sqmv-metadata-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
