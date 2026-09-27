# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqav-capture-cpp/SPEC.md",
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
        "modules/sqav-capture-cpp/tests/capture_test.cpp verifies original-byte fidelity, exact identities, finite decoding and allocation rollback, and five-owner retained replay.",
        "modules/sqav-capture-cpp/tests/sdk-consumer/main.cpp verifies installed capture round-trip and corruption/binding rejection.",
        "modules/sqav-capture-cpp/tests/package_lifecycle_test.cmake verifies exact isolated installation and guarded removal."
      ],
      "feature_id": "ssfv:symphony:sqav-capture-cpp",
      "how": "Canonical SQA1 encoding and SHA-256 bind original bytes and bounded evidence; exact SQMV source evidence admits the optional SQFV envelope bridge.",
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
        "modules/sqav-capture-cpp/CMakeLists.txt",
        "modules/sqav-capture-cpp/include/symphony/sqav/capture.hpp",
        "modules/sqav-capture-cpp/src/capture.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No provider connection, native-schema parser, authentication, source assertion verification, credentials, retry scheduler, source completeness or redistribution grant.",
        "No compulsory normalized representation, standalone C ABI, background collector, transformation, new storage backend or SQV qxctl command."
      ],
      "owner_contract": "modules/sqav-capture-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Uses exact SQFV copied immutable batches and lease ownership for optional capture movement.",
          "target_feature_id": "ssfv:symphony:sqfv-batch-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Verifies exact dataset, revision, capture representation, access scope and attributed source evidence.",
          "target_feature_id": "ssfv:symphony:sqmv-metadata-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqav-capture-cpp",
      "status": "experimental",
      "title": "SQAV bounded original-byte capture library",
      "what": "Preserves original bytes with exact source/interface/adapter identity, acquisition attempts, native positions, time-role evidence and coverage assertions in an optional bounded capture.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "Inside the explicit native caller through the exact installed sqav-capture-cpp static library.",
      "who": "Trusted C++26 callers explicitly selecting the offline capture representation.",
      "why": "Keep source evidence and original bytes recoverable through selected flow, retention and delivery without conflating provider position and transfer order."
    }
  ],
  "source_scope": "modules/sqav-capture-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
