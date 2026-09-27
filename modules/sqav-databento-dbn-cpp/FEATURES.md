# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqav-databento-dbn-cpp/SPEC.md",
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
        "modules/sqav-databento-dbn-cpp/tests/dbn_test.cpp checks public provider fixture fields, malformed/bounded inputs, raw sentinels, allocation rollback and retained replay.",
        "modules/sqav-databento-dbn-cpp/tests/sdk-consumer/main.cpp checks installed fixture fidelity and atomic rejection.",
        "modules/sqav-databento-dbn-cpp/tests/package_lifecycle_test.cmake checks immutable installation and guarded removal."
      ],
      "feature_id": "ssfv:symphony:sqav-databento-dbn-cpp",
      "how": "Allocation-free explicit little-endian parsing checks metadata grammar and record framing; an exact provider/adapter binding admits the original-byte capture.",
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
        "modules/sqav-databento-dbn-cpp/CMakeLists.txt",
        "modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/dbn.hpp",
        "modules/sqav-databento-dbn-cpp/src/dbn.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No provider network session, credential storage, dataset entitlement, billable request, live mixed-record stream, reference client or reconstructed order book.",
        "No DBNv1/v2 upgrade, compression, source completeness/authenticity, stateful transformation or SQV qxctl surface."
      ],
      "owner_contract": "modules/sqav-databento-dbn-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Preserves the entire original file in an exact attributed capture for optional downstream composition.",
          "target_feature_id": "ssfv:symphony:sqav-capture-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqav-databento-dbn-cpp",
      "status": "experimental",
      "title": "SQAV Databento DBNv3 MBO fidelity library",
      "what": "Inspects bounded uncompressed DBNv3 single-schema MBO files and preserves complete original bytes in an attributed SQAV capture.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "Inside a caller using the independently installed exact native library.",
      "who": "Trusted C++26 callers explicitly selecting the supported DBNv3 MBO profile.",
      "why": "Preserve provider-native meaning and original file evidence through existing research-data owners."
    }
  ],
  "source_scope": "modules/sqav-databento-dbn-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
