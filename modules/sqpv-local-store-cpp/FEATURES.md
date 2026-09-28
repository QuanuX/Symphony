# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqpv-local-store-cpp/SPEC.md",
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
        "modules/sqpv-local-store-cpp/tests/local_store_test.cpp verifies bounded admission, exact identity, failure behavior and the stated native contract.",
        "modules/sqpv-local-store-cpp/tests/sdk-consumer/main.cpp verifies the installed public C++26 interface independently of private implementation headers.",
        "modules/sqpv-local-store-cpp/tests/package_lifecycle_test.cmake verifies isolated install and guarded removal of the exact library package.",
        "The optional AsyncStore uses one owned worker with bounded frame bytes and slots including active writes, explicit lag/failure, drain and committed-prefix recovery. Native tests cover paused writes, conflicts, queue exhaustion, uncertain failure and two actual SIGKILL boundaries."
      ],
      "feature_id": "ssfv:symphony:sqpv-local-store-cpp",
      "how": "Exclusive writer ownership, immutable frames and hash-linked commit records, an atomic checked head and explicit file/directory synchronization keep retained evidence distinct from staged work.",
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
        "modules/sqpv-local-store-cpp/CMakeLists.txt",
        "modules/sqpv-local-store-cpp/include/symphony/sqpv/async_store.hpp",
        "modules/sqpv-local-store-cpp/include/symphony/sqpv/local_store.hpp",
        "modules/sqpv-local-store-cpp/src/async_store.cpp",
        "modules/sqpv-local-store-cpp/src/local_store.cpp",
        "modules/sqpv-local-store-cpp/tests/async_store_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No provider acquisition, power-loss certification, replication, hostile same-user filesystem mutation tolerance, access authorization, pruning, generation replacement or universal storage default.",
        "No destination acknowledgement, live-to-retained delivery cutover, remote transport, standalone C ABI, resident process or SQV qxctl command."
      ],
      "owner_contract": "modules/sqpv-local-store-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Stores and restores frames through the exact SQFV codec and caller-bounded context.",
          "target_feature_id": "ssfv:symphony:sqfv-batch-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Validates the resolved exact metadata reference and binding before retained admission.",
          "target_feature_id": "ssfv:symphony:sqmv-metadata-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqpv-local-store-cpp",
      "status": "experimental",
      "title": "SQPV bounded local retention library",
      "what": "Provides exact local retained streams and an optional bounded asynchronous writer with separate admission, confirmed progress and failure evidence.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "Inside the explicit native caller through the exact installed sqpv-local-store-cpp static library.",
      "who": "Trusted C++26 callers selecting a private local store for one exact manifest, partition and producer generation.",
      "why": "Provide the first independently selectable retention path while preserving exact source bindings, observable capacity failure and honest commit/recovery evidence."
    }
  ],
  "source_scope": "modules/sqpv-local-store-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
