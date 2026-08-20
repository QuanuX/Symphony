# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sev-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SEV and SCSEV changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes SEV contracts, implementation, tests, and feature ownership.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "SEV derives non-authoritative transition plans; the coordinator persists sessions and qxctl performs separately authorized external actions.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator"
        },
        {
          "distinction": "SCSEV consumes the established qxctl and SSFV registries and never creates a second command registry.",
          "target_feature_id": "ssfv:symphony:qxctl.command-registry"
        }
      ],
      "evidence": [
        "The C++ unit and process tests exercise descriptor v2, evidence-bound case opening, status, disabled apply, and caller-neutral inspection.",
        "The implementation validates dependency DAGs, localized blocker closure, closed success predicates, and all fourteen SCSEV consequence families."
      ],
      "feature_id": "ssfv:symphony:sev-engine",
      "how": "The C++26 process creates content-addressed successor cases, validates exact caller-declared impacts and dispositions, computes dependency-safe ready sets, verifies complete reobservation, and emits proposal-only recovery, closure, SCSEV, compatibility, and graph results.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements SEV case, impact, disposition, verification, recalculation, recovery, closure, SCSEV, graph, and compatibility operations."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and conservatively uninstalls exact inactive-undocked versions."
        }
      ],
      "implementation_paths": [
        "modules/sev-engine/CMakeLists.txt",
        "modules/sev-engine/src/main.cpp",
        "modules/sev-engine/src/sev.cpp",
        "modules/sev-engine/tests/process_smoke.sh",
        "modules/sev-engine/tests/sev_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not apply plans, persist sessions, authorize callers, submit STAV events, mutate Maestro, invent identities or grammar, or bypass hard safety edges.",
        "Does not create an SCSEV engine/registry, publish novelty, require AI/database/network access, or enter hot/warm execution."
      ],
      "owner_contract": "modules/sev-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Durable evolution reuses the existing coordinator journal and dynamic ready-set recovery instead of creating parallel state.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator",
          "type": "composes_with"
        },
        {
          "rationale": "SCSEV reuses existing qxctl command truth and administrative coverage evidence.",
          "target_feature_id": "ssfv:symphony:qxctl.command-registry",
          "type": "composes_with"
        },
        {
          "rationale": "The engine uses shared bounded process, digest, operation, JSON, and STSC mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sev-engine",
      "status": "experimental",
      "title": "SEV evolution and SCSEV assessment engine",
      "what": "Provides an independently installable freezing-path engine for governed evolution cases, deterministic impacts and ready-set plans, evidence-based verification/recovery, and qxctl command-surface consequence assessment.",
      "when": "Runs only on explicit bounded invocation before or after separately controlled external actions and complete reobservation.",
      "where": "Runs from an exact inactive-undocked installation and consumes caller-supplied SAV, SSFV, qxctl, operation, invariant, and lifecycle evidence.",
      "who": "Any authenticated or otherwise host-authorized subject, reviewer, integration process, independent module developer, or agent using the same caller-neutral protocol.",
      "why": "Closes Symphony's evidence-to-plan-to-reobservation loop while preserving owner authority, safe dependency order, and explicit administrative consequences."
    }
  ],
  "source_scope": "modules/sev-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
