# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/shv-source-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Completed material source changes require reviewed append-only closure.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes current owner contracts and distributed feature files.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
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
        "Focused source-process, independent consumer and installed qxctl acceptance is recorded at SHV-02 closure."
      ],
      "feature_id": "ssfv:symphony:shv-source-engine",
      "how": "Independently installed C++ source reducer and no-follow capture replay with exact receipt selection and independent Go consumer.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns source revision, capture and graph provenance semantics."
        },
        {
          "language": "CMake",
          "role": "Builds and receipts the exact independently selected engine and its declared dependencies."
        }
      ],
      "implementation_paths": [
        "modules/shv-source-engine/CMakeLists.txt",
        "modules/shv-source-engine/src/descriptor.cpp",
        "modules/shv-source-engine/src/main.cpp",
        "modules/shv-source-engine/src/source.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No registry activation, approval, publisher authentication or network acquisition.",
        "No new hardware mapping, rebrand equivalence inference, whole-system enumeration or vendor database persistence."
      ],
      "owner_contract": "modules/shv-source-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Authority-free process, path and digest mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/shv-source-engine",
      "status": "experimental",
      "title": "SHV component source lifecycle and provenance",
      "what": "Plans explicit source revisions, validates finite lineage, binds consumed capture bytes and projects source-replayed provenance graphs.",
      "when": "Explicit bounded read-only operations only.",
      "where": "Independent C++ installation with caller-provided retained source root.",
      "who": "Agents and humans invoking the exact installed SHV through qxctl.",
      "why": "Preserves source and caller authority while making finite hardware evidence queryable and portable."
    }
  ],
  "source_scope": "modules/shv-source-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
