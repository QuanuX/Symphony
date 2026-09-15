# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/shv-profile-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SKVI routes current owner contracts and distributed feature files.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Completed material source changes require reviewed append-only closure.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "Future official publication follows a separate authorized release boundary.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [],
      "evidence": [
        "Focused native, Go, installed conformance and schema checks are recorded in the SHV-21 evidence packet."
      ],
      "feature_id": "ssfv:symphony:shv-profile-engine",
      "how": "C++ validates profile and recipe semantics; the exact compiled kernel 0.3 reader replays selected bytes on binding. qxctl independently checks correspondence.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns profile, declaration conformance and universe recipe semantics."
        },
        {
          "language": "CMake",
          "role": "Builds and receipts the independent profile engine and compiled exact reader."
        }
      ],
      "implementation_paths": [
        "modules/shv-profile-engine/CMakeLists.txt",
        "modules/shv-profile-engine/src/profile.cpp",
        "modules/shv-profile-engine/src/descriptor.cpp",
        "modules/shv-profile-engine/src/main.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Mapping conformance is a caller declaration check, not hardware qualification or publisher authentication.",
        "No acquisition, active source mutation, catalogue publication or graph database default."
      ],
      "owner_contract": "modules/shv-profile-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 1,
      "relationships": [
        {
          "rationale": "Authority-free process, path and digest mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/shv-profile-engine",
      "status": "experimental",
      "title": "SHV caller hardware profiles and portable universes",
      "what": "Caller-defined class profiles, mapping conformance diagnostics and portable hardware-universe recipes.",
      "when": "Explicit qxctl profile, mapping and universe operations.",
      "where": "Independent receipt-owned engine and caller-selected local evidence roots.",
      "who": "Agents and humans invoking the exact installed SHV through qxctl.",
      "why": "Make comparable declarations and portable evidence selection available without imposing a hardware census or architecture."
    }
  ],
  "source_scope": "modules/shv-profile-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
