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
        "The C++ unit and direct-process tests exercise descriptor v2, complete and unknown CURRENT coverage, Named Version immutability, incomplete third-party Capsules, inverse Blueprint readiness and cycle rejection, disabled apply, and caller-neutral inspection.",
        "Receipt-v2 CMake surfaces prove independently installable inactive-undocked packaging."
      ],
      "feature_id": "ssfv:symphony:sav-engine",
      "how": "The C++26 process validates exact bounded JSON, tagged digests, source payload bindings, STSC timestamps, a closed rule algebra, coverage qualification, independent result axes, immutable Named Versions, incomplete Extension Capsules, acyclic two-way Blueprints, deterministic ordering, and disposable graphs.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements SAV validation, CURRENT resolution, accord evaluation, Named Version, Extension Capsule, Installation Blueprint, explanation, diff, graph, and compatibility operations."
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
      "what": "Provides an independently installable freezing-path engine that validates Accord References, derives immutable coverage-qualified CURRENT snapshots, evaluates relationship accord, validates Named Versions and Extension Capsules, plans two-way Installation Blueprints, and emits deterministic read-only projections.",
      "when": "Runs only on explicit bounded direct-process or qxctl invocation against caller-supplied evidence.",
      "where": "Runs from an exact inactive-undocked installation on Linux, macOS, or WSL/remote-node use without a repository requirement for evidence-only operations.",
      "who": "Any authenticated or otherwise host-authorized subject, reviewer, integration process, or agent using the same caller-neutral protocol.",
      "why": "Makes exact installed composition and cross-vector compatibility explainable without centralizing or rewriting source-vector truth."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SKVI routes the governing SAV contracts and schemas.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "CURRENT is an evidence-qualified snapshot, not an installed-version alias or mutable desired state.",
          "target_feature_id": "ssfv:symphony:sav-engine.named-version"
        }
      ],
      "evidence": [
        "Unit and process tests cover reference validation, complete and unknown CURRENT qualification, evaluation, diff, explanation, and graph projection."
      ],
      "feature_id": "ssfv:symphony:sav-engine.current-accord",
      "how": "Validates bounded source projections and closed relationship rules, then emits deterministic snapshots, evaluations, explanations, diffs, and disposable graphs.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded CURRENT and accord operations."
        }
      ],
      "implementation_paths": [
        "modules/sav-engine/src/sav.cpp",
        "modules/sav-engine/tests/sav_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not discover ambient state, persist snapshots, or authorize mutation."
      ],
      "owner_contract": "modules/sav-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sav-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Extends the SAV parent capability with its primary evidence-to-accord circuit.",
          "target_feature_id": "ssfv:symphony:sav-engine",
          "type": "extends"
        }
      ],
      "source_scope": "modules/sav-engine",
      "status": "experimental",
      "title": "SAV CURRENT and accord evaluation",
      "what": "Resolves coverage-qualified CURRENT state and evaluates declared relationships across exact source evidence.",
      "when": "On explicit direct-process or qxctl query and validation requests.",
      "where": "In the independently installed freezing-path SAV process.",
      "who": "Any host-authorized caller using the caller-neutral operation protocol.",
      "why": "Makes live composition evidence explicit, bounded, and explainable."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SKVI supplies canonical registration paths evaluated by capsule admission.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "Capsule checks assess third-party integration readiness; they do not install or dock packages.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.lifecycle-planning"
        }
      ],
      "evidence": [
        "Unit tests cover complete and incomplete capsules while prohibiting invented feature, command, and operation identities."
      ],
      "feature_id": "ssfv:symphony:sav-engine.extension-capsule",
      "how": "Checks receipt, semantic registration, command, operation, receptor, trait, and accord bindings as independent readiness axes.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements deterministic Extension Capsule admission checks."
        }
      ],
      "implementation_paths": [
        "knowledge/sav/schemas/v1/extension-capsule.schema.json",
        "modules/sav-engine/src/sav.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not install packages or invent missing identities and grammar."
      ],
      "owner_contract": "modules/sav-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sav-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Extends SAV composition checks to independently developed modules.",
          "target_feature_id": "ssfv:symphony:sav-engine",
          "type": "extends"
        }
      ],
      "source_scope": "modules/sav-engine",
      "status": "experimental",
      "title": "SAV Extension Capsule admission",
      "what": "Reports whether an independently developed module exposes enough package, semantic, administrative, and docking evidence to integrate.",
      "when": "Before a separately authorized module lifecycle action.",
      "where": "In the freezing-path SAV validation process.",
      "who": "Module developers, administrators, and agents with host-granted access.",
      "why": "Detects forgotten qxctl and semantic surfaces before deployment."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Maestro owns persistent receptor state consumed by lifecycle execution.",
          "reference": "modules/maestro/SPEC.md",
          "vector": "maestro"
        }
      ],
      "distinctions": [
        {
          "distinction": "Blueprint planning proposes a dynamic ready set; qxctl and the coordinator own separately authorized execution and recovery.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.lifecycle-apply-coordination"
        }
      ],
      "evidence": [
        "Unit tests prove forward/reverse inverse edges, cycle rejection, blocker localization, convergence, and disabled apply."
      ],
      "feature_id": "ssfv:symphony:sav-engine.installation-blueprint",
      "how": "Validates exact inverse dependency graphs and recalculates direction-aware ready and blocked sets after each supplied outcome.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements acyclic two-way Blueprint planning."
        }
      ],
      "implementation_paths": [
        "knowledge/sav/schemas/v1/installation-blueprint.schema.json",
        "modules/sav-engine/src/sav.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not apply, persist, install, remove, dock, or undock components."
      ],
      "owner_contract": "modules/sav-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sav-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Supplies composition-aware planning to the external lifecycle circuit.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.lifecycle-planning",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/sav-engine",
      "status": "experimental",
      "title": "SAV two-way Installation Blueprints",
      "what": "Produces deterministic forward or reverse dependency-safe ready sets that can replan around recoverable ordering blocks.",
      "when": "Before and between separately authorized lifecycle steps.",
      "where": "In the freezing-path SAV process using caller-supplied progress evidence.",
      "who": "The headless qxctl lifecycle circuit or any equivalent host-authorized caller.",
      "why": "Allows rolling module changes to converge safely even when upgrades occur out of the preferred order."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SODV may later bind reviewed publication evidence without changing SAV ownership.",
          "reference": "knowledge/sodv/INTENT.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "A Named Version is an immutable composition envelope, not the dynamically qualified CURRENT snapshot.",
          "target_feature_id": "ssfv:symphony:sav-engine.current-accord"
        }
      ],
      "evidence": [
        "Unit tests validate immutable envelopes, lineage, deterministic diffs, and disabled sealing authority."
      ],
      "feature_id": "ssfv:symphony:sav-engine.named-version",
      "how": "Validates exact content-addressed envelopes and compares successor lineage without storing or sealing them.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements Named Version validation and deterministic diffing."
        }
      ],
      "implementation_paths": [
        "knowledge/sav/schemas/v1/named-version.schema.json",
        "modules/sav-engine/src/sav.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not seal, publish, select, or persist a Named Version."
      ],
      "owner_contract": "modules/sav-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sav-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Extends SAV with immutable composition references.",
          "target_feature_id": "ssfv:symphony:sav-engine",
          "type": "extends"
        }
      ],
      "source_scope": "modules/sav-engine",
      "status": "experimental",
      "title": "SAV immutable Named Versions",
      "what": "Validates and compares immutable, content-addressed composition envelopes.",
      "when": "During explicit review, compatibility, or upgrade-planning requests.",
      "where": "In the freezing-path SAV process against caller-supplied envelopes.",
      "who": "Any host-authorized caller with exact Named Version evidence.",
      "why": "Gives rolling deployments a stable composition identity without confusing it with live state."
    }
  ],
  "source_scope": "modules/sav-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
