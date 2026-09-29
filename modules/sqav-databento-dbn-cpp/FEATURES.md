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
        "modules/sqav-databento-dbn-cpp/tests/dbn_test.cpp checks public v1/v3 provider fixture fields, exact encoding/version binding, private opt-in sample replay, malformed/bounded inputs, raw sentinels, allocation rollback and retained replay.",
        "modules/sqav-databento-dbn-cpp/tests/sdk-consumer/main.cpp checks installed fixture fidelity and atomic rejection.",
        "modules/sqav-databento-dbn-cpp/tests/package_lifecycle_test.cmake checks immutable installation and guarded removal.",
        "The new exact package admits the updated capture/metadata/flow dependency chain while retaining the specified DBNv1/v3 MBO file scope.",
        "modules/sqav-databento-dbn-cpp/tests/historical_test.cpp verifies request canonicalization, response conformance, interruption/cap/retry limits, allocation rollback and asynchronous retained replay; the installed consumer independently rejects interrupted and mismatched responses.",
        "Adds durable attempt/charge-ceiling admission and bounded native historical HTTP with an explicitly required SSIAG bridge. Native owner tests and the independently installed consumer verify the new boundary."
      ],
      "feature_id": "ssfv:symphony:sqav-databento-dbn-cpp",
      "how": "Explicit endian parsing and immutable request identities bind metadata and receive-time ranges; bounded response assembly distinguishes transport completion, record caps, unresolved symbols and conservative recovery.",
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
        "modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/attempts.hpp",
        "modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/dbn.hpp",
        "modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/historical.hpp",
        "modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/http.hpp",
        "modules/sqav-databento-dbn-cpp/src/attempts.cpp",
        "modules/sqav-databento-dbn-cpp/src/dbn.cpp",
        "modules/sqav-databento-dbn-cpp/src/historical.cpp",
        "modules/sqav-databento-dbn-cpp/src/http.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No installed operational SSIAG credential bridge, provider activation, live collector, reference client, reconstructed order book or actual invoice guarantee.",
        "No DBNv2 support, implicit wire-version upgrade, compression, source authenticity, destination commit or SQV qxctl surface."
      ],
      "owner_contract": "modules/sqav-databento-dbn-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Preserves the entire original file in an exact attributed capture for optional downstream composition.",
          "target_feature_id": "ssfv:symphony:sqav-capture-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Uses the exact SQPV retained store for persistent attempt and charge-ceiling accounting.",
          "target_feature_id": "ssfv:symphony:sqpv-local-store-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqav-databento-dbn-cpp",
      "status": "experimental",
      "title": "SQAV Databento DBN fidelity and bounded historical planning",
      "what": "Inspects exact uncompressed DBNv1/v3 MBO files, plans bounded historical requests, classifies transport/coverage/recovery evidence and preserves bound original-byte captures. Adds durable attempt/charge-ceiling admission and bounded native historical HTTP with an explicitly required SSIAG bridge.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "Inside a caller using the independently installed exact native library.",
      "who": "Trusted C++26 callers explicitly selecting the supported DBNv1/v3 MBO profile.",
      "why": "Preserve provider-native meaning and original file evidence through existing research-data owners."
    }
  ],
  "source_scope": "modules/sqav-databento-dbn-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
