# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/shv-partition-engine/SPEC.md",
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
        "Focused SHV-07 native, independent consumer and installed qxctl evidence accompanies local closure."
      ],
      "feature_id": "ssfv:symphony:shv-partition-engine",
      "how": "Independent C++ partition engine with exact receipt selection and independent Go correspondence checks.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns immutable partition, manifest and bounded reference-query semantics."
        },
        {
          "language": "CMake",
          "role": "Builds and receipts the exact independently selected connector and dependency."
        }
      ],
      "implementation_paths": [
        "modules/shv-partition-engine/CMakeLists.txt",
        "modules/shv-partition-engine/src/partition.cpp",
        "modules/shv-partition-engine/src/descriptor.cpp",
        "modules/shv-partition-engine/src/main.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Direct native dependency references do not authenticate bytes, publisher claims or hardware compatibility.",
        "No durable catalogue head, network acquisition, graph vendor or resumable materialization checkpoints."
      ],
      "owner_contract": "modules/shv-partition-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Authority-free process, path and digest mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/shv-partition-engine",
      "status": "experimental",
      "title": "SHV immutable partition manifests",
      "what": "Builds immutable dependency identities, explicit incomplete inventories and bounded cursor-bound reference queries.",
      "when": "Explicit bounded read-only operations only.",
      "where": "Independent C++ installation with caller-provided retained source root.",
      "who": "Agents and humans invoking the exact installed SHV through qxctl.",
      "why": "Scale explicit documentary inventories without selecting hardware or hiding missing partitions."
    }
  ],
  "source_scope": "modules/shv-partition-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->

## Explicit composition release 0.4.0-dev

Partition 0.4.0-dev explicitly admits source 0.1/0.2 and kernel 0.1/0.2/0.3/0.4. Historical partition 0.1/0.2/0.3 retain their prior source/kernel admission. qxctl passes the selected partition owner through build, manifest, query, materialization checkpoint and resolution replay. The embedded previous-version macro retains the partition 0.2 semantics.

Use a fresh install prefix and explicit qxctl version selection. Version 0.3 declarations remain checked against their original compiled digest. No command or default changes; metadata is not semantic authorization and future versions are not automatically admitted.
