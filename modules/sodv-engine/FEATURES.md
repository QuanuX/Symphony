# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sodv-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed engine and vector changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes engine contracts, implementation, and feature ownership.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future engine release.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine interprets one vector read-only; the coordinator persists cross-session noncanonical administrative evidence and never decides vector truth.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator"
        }
      ],
      "evidence": [
        "modules/sodv-engine tests exercise the implemented operation set, deterministic results, bounds, invalid input, and disabled apply.",
        "The module CMake and receipt-v2 surfaces prove independent versioned installation and conservative uninstall."
      ],
      "feature_id": "ssfv:symphony:sodv-engine",
      "how": "The C++ process uses the shared bounded engine envelope, no-follow reads, canonical digests, deterministic ordering, strict schemas, explicit resource ceilings, and disabled canonical apply.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded SODV inspection, validation, evidence, proposal, and vector-authorized projection operations without canonical writes."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and uninstalls the exact inactive-undocked engine package."
        }
      ],
      "implementation_paths": [
        "modules/sodv-engine/CMakeLists.txt",
        "modules/sodv-engine/src/main.cpp",
        "modules/sodv-engine/src/sodv.cpp",
        "modules/sodv-engine/tests/process_smoke.sh",
        "modules/sodv-engine/tests/sodv_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not mutate canonical knowledge, decide semantic membership or ratification, start a listener, or infer permission from caller type.",
        "Does not activate or dock itself, publish artifacts, or claim an overall production release."
      ],
      "owner_contract": "modules/sodv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The engine statically links the common bounded process, digest, path, temporal, and snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sodv-engine",
      "status": "experimental",
      "title": "SODV independently installed knowledge engine",
      "what": "Provides the independently installable freezing-path engine that reads canonical SODV truth and emits deterministic bounded evidence, proposals, and disposable projections under that vector's contract.",
      "when": "Runs only on explicit qxctl or exact local process invocation under a bounded deadline and against an exact repository snapshot.",
      "where": "Executes from an inactive-undocked versioned installation and reads repository truth owned by knowledge/sodv/ without a network listener.",
      "who": "Any host-authorized caller using qxctl, vector maintainers, reviewers, integration tooling, and tests.",
      "why": "Gives SODV application-owned programmatic behavior without transferring semantic, ratification, mutation, or publication authority to tooling."
    }
  ],
  "source_scope": "modules/sodv-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
