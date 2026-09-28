# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqav-request-engine/SPEC.md",
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
        "modules/sqav-request-engine/tests/test.cpp verifies native parity and refusal boundaries."
      ],
      "feature_id": "ssfv:symphony:sqav-request-engine",
      "how": "Exact tagged payloads call existing Plan::create functions; canonical request/result digests and receipt-owned interface/schema admission bind the response.",
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
        "modules/sqav-request-engine/src/main.cpp",
        "modules/sqav-request-engine/src/request.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No acquisition, live feed, credential retrieval, provider observation or store mutation."
      ],
      "owner_contract": "modules/sqav-request-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Calls the existing native request validator.",
          "target_feature_id": "ssfv:symphony:sqav-databento-dbn-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Calls the existing native request validator.",
          "target_feature_id": "ssfv:symphony:sqav-databento-reference-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Calls the existing native request validator.",
          "target_feature_id": "ssfv:symphony:sqav-fred-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqav-request-engine",
      "status": "experimental",
      "title": "SQAV native request validation process",
      "what": "One bounded process operation validates three native source request variants.",
      "when": "Only on explicit invocation; no provider traffic or credential use.",
      "where": "Independently installed sqav-request-engine executable.",
      "who": "Native process callers and qxctl SQV administration.",
      "why": "Expose native request validation without duplicating provider semantics in qxctl."
    }
  ],
  "source_scope": "modules/sqav-request-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
