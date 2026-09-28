# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqfv-batch-cpp/SPEC.md",
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
          "reason": "The source module Quad, C++26 API, build, and focused tests have individual discovery routes.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [],
      "evidence": [
        "modules/sqfv-batch-cpp/tests/batch_test.cpp verifies immutable copied payloads, exact binding and scope checks, independent bounded port cursors and credits, leases after owner release, stalls, and resource boundaries in focused native CTest.",
        "modules/sqfv-batch-cpp/tests/frame_test.cpp verifies the exact independent frame and content-ID golden bytes, round trips, corruption and length rejection, and configured bounds in focused native CTest.",
        "modules/sqfv-batch-cpp/tests/sdk-consumer/main.cpp compiles as C++26 outside the checkout against the installed exact CMake package and exercises frozen copy, offer/take, lease lifetime, and reservation accounting.",
        "modules/sqfv-batch-cpp/CMakeLists.txt installs a library-only receipt-v2 package with no process entry points and guarded exact-version uninstall.",
        "Independently retained contexts share the same charged allocation ledger and frame identity domain; the integrated pipeline releases producer handles while asynchronous work retains ownership."
      ],
      "feature_id": "ssfv:symphony:sqfv-batch-cpp",
      "how": "A C++26 static library exposes move-only RAII handles, makes one immutable payload copy, tracks finite charged reservation units and independent per-port byte credits, and uses a bounded SHA-256 frame codec.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements trusted same-process batches, leases, bounded consumer ports and cursors, and frame encoding and decoding."
        },
        {
          "language": "CMake",
          "role": "Builds, exports, receipts, and guarded-uninstalls the exact library package."
        }
      ],
      "implementation_paths": [
        "modules/sqfv-batch-cpp/CMakeLists.txt",
        "modules/sqfv-batch-cpp/include/symphony/sqfv/batch.hpp",
        "modules/sqfv-batch-cpp/src/batch.cpp",
        "modules/sqfv-batch-cpp/src/frame.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No provider acquisition, SQMV interpretation, private access enforcement, recipient authorization, durable commit or replay, cross-process or network transport, device lifetime, exactly-once processing, or throughput guarantee.",
        "No standalone C ABI, resident process, engine operation, or SQV qxctl command is supplied by this library-only package."
      ],
      "owner_contract": "modules/sqfv-batch-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [],
      "source_scope": "modules/sqfv-batch-cpp",
      "status": "experimental",
      "title": "SQFV trusted same-process batch library",
      "what": "Provides immutable copied batches, explicit read leases, independently credited consumer ports, and a bounded versioned local frame codec.",
      "when": "Runs only when a trusted native caller explicitly links and invokes the exact installed library; installation alone starts nothing.",
      "where": "Inside one caller address space through the versioned sqfv-batch-cpp static archive.",
      "who": "Trusted same-process C++26 consumers using an exact SQFV metadata binding and finite resource configuration.",
      "why": "Establish explicit memory lifetime, ordering context, and backpressure for composable SQV research-data movement without assigning provider, meaning, access, or persistence decisions to SQFV."
    }
  ],
  "source_scope": "modules/sqfv-batch-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
