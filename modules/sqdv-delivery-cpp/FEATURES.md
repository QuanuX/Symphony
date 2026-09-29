# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqdv-delivery-cpp/SPEC.md",
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
        "modules/sqdv-delivery-cpp/tests/delivery_test.cpp verifies exact resume, acknowledgement/lifetime separation, finite allowances, retention provenance and failure preservation.",
        "modules/sqdv-delivery-cpp/tests/sdk-consumer/main.cpp verifies installed consumer rejection and continuity through the public C++26 interface.",
        "modules/sqdv-delivery-cpp/tests/package_lifecycle_test.cmake verifies isolated install and guarded removal of the exact library package.",
        "The asynchronous profile accepts opaque queue-admission proofs for non-durable preview and uses actual SQPV reads for confirmed replay; processing checkpoints remain distinct from retention.",
        "Adds exact-view durable processing checkpoints and confirmed-prefix admission for retained replay. Native owner tests and the independently installed consumer verify the new boundary."
      ],
      "feature_id": "ssfv:symphony:sqdv-delivery-cpp",
      "how": "SQFV owns queued payload leases; a finite acknowledgement ledger tracks processing separately. Opaque retained proofs come from actual SQPV commits and exact reads, with SQMV-bound view identity.",
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
        "modules/sqdv-delivery-cpp/CMakeLists.txt",
        "modules/sqdv-delivery-cpp/include/symphony/sqdv/checkpoint.hpp",
        "modules/sqdv-delivery-cpp/include/symphony/sqdv/delivery.hpp",
        "modules/sqdv-delivery-cpp/src/checkpoint.cpp",
        "modules/sqdv-delivery-cpp/src/delivery.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No provider acquisition, projection or transformation, access authorization, recipient authentication, remote destination commit or exactly-once delivery.",
        "No universal storage backend, network or IPC transport, resident service, standalone C ABI or SQV qxctl command."
      ],
      "owner_contract": "modules/sqdv-delivery-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Uses exact SQFV port, cursor and lease mechanics for payload ownership and independent byte credits.",
          "target_feature_id": "ssfv:symphony:sqfv-batch-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Binds each complete view to its immutable resolved metadata manifest.",
          "target_feature_id": "ssfv:symphony:sqmv-metadata-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Derives retained evidence only from the actual owned store and its admitted local commit/recovery contract.",
          "target_feature_id": "ssfv:symphony:sqpv-local-store-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqdv-delivery-cpp",
      "status": "experimental",
      "title": "SQDV bounded native delivery and resume library",
      "what": "Provides exact same-process disposable, retention-before-delivery and asynchronous-preview profiles with independent processing acknowledgements and confirmed replay. Adds exact-view durable processing checkpoints and confirmed-prefix admission for retained replay.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "Inside the explicit native caller through the exact installed sqdv-delivery-cpp static library.",
      "who": "Trusted C++26 callers selecting disposable or retained-before-delivery full-batch views for exact local recipients.",
      "why": "Preserve consumer continuity and evidence across retained and live data while keeping recipient progress, retention and memory release distinct."
    }
  ],
  "source_scope": "modules/sqdv-delivery-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
