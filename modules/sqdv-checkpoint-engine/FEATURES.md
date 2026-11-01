# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqdv-checkpoint-engine/SPEC.md",
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
      "feature_id": "ssfv:symphony:sqdv-checkpoint-engine",
      "how": "Exact C++26 owner operations return bounded request-bound evidence. Receipt-owned declarations and schema resources bind process admission; reusable transport helpers hold no vector semantics. The shared SQPV reader preserves no-follow paths, shared locking, stat/inventory identity fences, finite bounds and dirty-state semantics on macOS local APFS and Linux.",
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
        "modules/sqdv-checkpoint-engine/src/admin.cpp",
        "modules/sqpv-inspection-engine/src/store_reader.cpp",
        "tools/sqv-administration-cpp/main.cpp",
        "tools/sqv-administration-cpp/request.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No live collection, provider access, credentials, paid request, store recovery, journal write, remote acknowledgment or execution grant.",
        "No Linux filesystem-name allowlist or filesystem durability certification; the caller selects the deployment filesystem.",
        "No native Windows support. Shared Linux ARM64 implementation does not establish physical DGX Spark, exact DGX OS image/kernel/driver/runtime or GPU qualification; that evidence remains separate and pending."
      ],
      "owner_contract": "modules/sqdv-checkpoint-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Uses the exact native owner contract or persisted format.",
          "target_feature_id": "ssfv:symphony:sqdv-delivery-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Uses the exact native owner contract or persisted format.",
          "target_feature_id": "ssfv:symphony:sqpv-inspection-engine",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqdv-checkpoint-engine",
      "status": "experimental",
      "title": "SQDV checkpoint administration",
      "what": "Inspect persisted local-processing checkpoints without repairing or advancing the journal.",
      "when": "On explicit local invocation with an exact installed release and bounded request.",
      "where": "Independently installed sqdv-checkpoint-engine process on macOS local APFS or Ubuntu LTS Linux x86_64/ARM64; DGX Spark shares the Linux ARM64 code path.",
      "who": "Native process callers and qxctl SQV administration.",
      "why": "Expose SQDV administration through its owner while sharing source discovery and schema/template routes."
    }
  ],
  "source_scope": "modules/sqdv-checkpoint-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
