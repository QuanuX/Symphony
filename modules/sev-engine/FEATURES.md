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
        "The C++ unit and process tests exercise descriptor v2, evidence-bound case opening, exact lifecycle-session binding, watch/novelty boundaries, trigger coalescing, disabled apply, and caller-neutral inspection.",
        "The implementation validates dependency DAGs, dynamic successor ready sets, localized blocker closure, closed success predicates, Accordare self-assessment, incomplete third-party command surfaces, and all fourteen SCSEV consequence families."
      ],
      "feature_id": "ssfv:symphony:sev-engine",
      "how": "The C++26 process creates content-addressed successor cases, validates exact caller-declared impacts and dispositions, recomputes dependency-safe ready sets after every outcome, binds cases to the shared lifecycle stream, checks watch and novelty artifacts, coalesces bounded events, verifies complete reobservation, and emits proposal-only recovery, closure, SCSEV, compatibility, and graph results.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements SEV case, impact, disposition, verification, recalculation, lifecycle-session binding, watch, novelty, trigger, recovery, closure, SCSEV, graph, and compatibility operations."
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
      "what": "Provides an independently installable freezing-path engine for governed evolution cases, dynamic deterministic impacts and ready-set plans, exact shared-journal bindings, opt-in watch and private novelty assessment, evidence-based verification/recovery, and qxctl command-surface consequence assessment.",
      "when": "Runs only on explicit bounded invocation before or after separately controlled external actions and complete reobservation.",
      "where": "Runs from an exact inactive-undocked installation and consumes caller-supplied SAV, SSFV, qxctl, operation, invariant, and lifecycle evidence.",
      "who": "Any authenticated or otherwise host-authorized subject, reviewer, integration process, independent module developer, or agent using the same caller-neutral protocol.",
      "why": "Closes Symphony's evidence-to-plan-to-reobservation loop while preserving owner authority, safe dependency order, and explicit administrative consequences."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed evolution-engine changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        }
      ],
      "distinctions": [
        {
          "distinction": "SEV derives plans and recovery advice; the coordinator persists sessions and qxctl performs authorized actions.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator"
        }
      ],
      "evidence": [
        "Unit and process tests cover evidence-bound cases, impact, DAG planning, recalculation, verification, status, recovery, closure, and graph projection."
      ],
      "feature_id": "ssfv:symphony:sev-engine.dynamic-evolution",
      "how": "Uses content-addressed cases and dependency-safe successor ready sets, recomputing after every observed outcome while preserving hard safety edges.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements deterministic evolution analysis, planning, verification, and recovery advice."
        }
      ],
      "implementation_paths": [
        "modules/sev-engine/src/sev.cpp",
        "modules/sev-engine/tests/sev_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not apply plans, persist sessions, or authorize external action."
      ],
      "owner_contract": "modules/sev-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sev-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Extends SEV with its evidence-to-plan-to-reobservation circuit.",
          "target_feature_id": "ssfv:symphony:sev-engine",
          "type": "extends"
        }
      ],
      "source_scope": "modules/sev-engine",
      "status": "experimental",
      "title": "SEV dynamic evolution circuit",
      "what": "Creates and evaluates change cases, deterministic impact and disposition plans, reobservation results, and forward-only recovery advice.",
      "when": "Before, between, and after separately authorized lifecycle actions.",
      "where": "In the independently installed freezing-path SEV process.",
      "who": "Any host-authorized caller using exact evolution evidence.",
      "why": "Makes change convergence explainable and recoverable under unplanned execution order."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Maestro owns the persistent receptor state referenced by the shared lifecycle journal.",
          "reference": "modules/maestro/SPEC.md",
          "vector": "maestro"
        }
      ],
      "distinctions": [
        {
          "distinction": "The binding is immutable evidence; the coordinator owns journal persistence and lifecycle transitions.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator"
        }
      ],
      "evidence": [
        "Tests bind a case to exact lifecycle profile, report-journal, desired-state, direction, and STSC time evidence with apply disabled."
      ],
      "feature_id": "ssfv:symphony:sev-engine.lifecycle-binding",
      "how": "Validates exact digests and UTC seconds, then produces a content-addressed noncanonical binding to the existing lifecycle stream.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements evolution-to-lifecycle evidence binding."
        }
      ],
      "implementation_paths": [
        "knowledge/sev/schemas/v1/evolution-session-binding.schema.json",
        "modules/sev-engine/src/sev.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not create, mutate, persist, close, or recover the lifecycle session."
      ],
      "owner_contract": "modules/sev-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sev-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Connects SEV evidence to the one shared durable lifecycle stream.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/sev-engine",
      "status": "experimental",
      "title": "SEV shared lifecycle-session binding",
      "what": "Binds an evolution case to exact lifecycle profile, report-journal, desired-state, direction, and time evidence.",
      "when": "Before external lifecycle execution begins or resumes.",
      "where": "In SEV, with persistence remaining in the existing coordinator and Maestro surfaces.",
      "who": "The qxctl lifecycle circuit or any equivalent host-authorized caller.",
      "why": "Prevents a parallel evolution journal while preserving complete recovery provenance."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "not_applicable",
          "reason": "Watch and novelty artifacts are private offline projections and do not publish through SODV.",
          "reference": null,
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "Watch checks validate opt-in policy and coalesce events; they do not run ambient monitoring or export novelty.",
          "target_feature_id": "ssfv:symphony:sev-engine.dynamic-evolution"
        }
      ],
      "evidence": [
        "Tests cover default-disabled watch policy, bounded event coalescing, forbidden secret keys, redaction integrity, and disabled network transfer."
      ],
      "feature_id": "ssfv:symphony:sev-engine.novelty-watch",
      "how": "Validates explicit generation-linked watch policy, private redacted novelty bundles, and sorted bounded event sets that only propose a case kind.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements offline novelty validation and opt-in trigger coalescing."
        }
      ],
      "implementation_paths": [
        "knowledge/sev/schemas/v1/novelty-bundle.schema.json",
        "knowledge/sev/schemas/v1/watch-policy.schema.json",
        "modules/sev-engine/src/sev.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not watch by default, mutate ambient state, export data, open cases, or transfer over a network."
      ],
      "owner_contract": "modules/sev-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sev-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Supplies bounded encountered-novelty evidence to later explicit evolution requests.",
          "target_feature_id": "ssfv:symphony:sev-engine.dynamic-evolution",
          "type": "enables"
        }
      ],
      "source_scope": "modules/sev-engine",
      "status": "experimental",
      "title": "SEV private novelty and opt-in watch policy",
      "what": "Checks private offline novelty bundles and coalesces explicitly supplied watch events under a default-disabled policy.",
      "when": "Only after explicit policy enablement and an explicit operation request.",
      "where": "In the freezing path; never in hot or warm trading execution.",
      "who": "Any host-authorized caller under the same caller-neutral policy.",
      "why": "Captures encountered change signals without adding ambient surveillance or hidden mutation."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SKVI routes the command and feature contracts assessed by SCSEV.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "SCSEV assesses established qxctl and SSFV truth and never creates a parallel registry.",
          "target_feature_id": "ssfv:symphony:qxctl.command-registry"
        }
      ],
      "evidence": [
        "Tests exercise all fourteen command-surface consequence families and incomplete third-party registrations."
      ],
      "feature_id": "ssfv:symphony:sev-engine.scsev",
      "how": "Compares proposed semantic capability changes with stable commands, operation bindings, manifests, recovery, and authority metadata.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements deterministic command-surface consequence assessment."
        }
      ],
      "implementation_paths": [
        "knowledge/sev/schemas/v1/command-surface-assessment.schema.json",
        "modules/sev-engine/src/sev.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not invent command names, grammar, feature identities, or implementation patches."
      ],
      "owner_contract": "modules/sev-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sev-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Checks administrative consequences of semantic evolution.",
          "target_feature_id": "ssfv:symphony:qxctl.command-registry",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/sev-engine",
      "status": "experimental",
      "title": "SCSEV command-surface assessment",
      "what": "Reports whether a feature change has complete stable qxctl, engine-operation, manifest, recovery, and authority coverage.",
      "when": "During module admission and before ratifying capability changes.",
      "where": "Inside SEV as a read-only freezing-path operation.",
      "who": "Independent module developers, reviewers, administrators, and agents.",
      "why": "Catches powerful but administratively unreachable features before they ship."
    }
  ],
  "source_scope": "modules/sev-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
