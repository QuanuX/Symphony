# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sav-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SAV changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes SAV contracts, implementation, tests, and feature ownership.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "SAV composes and evaluates caller-supplied owner evidence; qxctl selects an exact installed process and assembles host evidence without owning SAV semantics.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        }
      ],
      "evidence": [
        "The C++ unit and direct-process tests exercise descriptor v2, complete and unknown CURRENT coverage, disabled apply, and caller-neutral inspection.",
        "Receipt-v2 CMake surfaces prove independently installable inactive-undocked packaging."
      ],
      "feature_id": "ssfv:symphony:sav-engine",
      "how": "The C++26 process validates exact bounded JSON, tagged digests, source payload bindings, STSC timestamps, a closed rule algebra, coverage qualification, independent result axes, deterministic ordering, and disposable graphs.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements SAV validation, CURRENT resolution, accord evaluation, explanation, diff, graph, and compatibility operations."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and conservatively uninstalls exact inactive-undocked versions."
        }
      ],
      "implementation_paths": [
        "modules/sav-engine/CMakeLists.txt",
        "modules/sav-engine/src/main.cpp",
        "modules/sav-engine/src/sav.cpp",
        "modules/sav-engine/tests/process_smoke.sh",
        "modules/sav-engine/tests/sav_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not discover ambient host state, mutate canonical knowledge, install, select, dock, authorize, persist, or listen on a network.",
        "Does not execute prose or model output, claim completeness without exact required evidence, or enter hot/warm execution."
      ],
      "owner_contract": "modules/sav-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl is the preferred headless binding and evidence-assembly surface while direct IPC remains supported.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "The engine uses shared bounded process, digest, operation, JSON, and STSC mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sav-engine",
      "status": "experimental",
      "title": "SAV composition and accord engine",
      "what": "Provides an independently installable freezing-path engine that validates Accord References, derives immutable coverage-qualified CURRENT snapshots, evaluates relationship accord, and emits deterministic read-only projections.",
      "when": "Runs only on explicit bounded direct-process or qxctl invocation against caller-supplied evidence.",
      "where": "Runs from an exact inactive-undocked installation on Linux, macOS, or WSL/remote-node use without a repository requirement for evidence-only operations.",
      "who": "Any authenticated or otherwise host-authorized subject, reviewer, integration process, or agent using the same caller-neutral protocol.",
      "why": "Makes exact installed composition and cross-vector compatibility explainable without centralizing or rewriting source-vector truth."
    }
  ],
  "source_scope": "modules/sav-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
