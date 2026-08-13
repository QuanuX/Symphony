# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "tools/symphony-validator/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to this implemented capability.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the canonical contracts, implementation evidence, and this distributed feature owner.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future release or publication of this active-development capability.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The validator checks repository-wide rule families; SKVI engine behavior is limited to canonical index truth and vector-authorized proposals/projections.",
          "target_feature_id": "ssfv:symphony:skvi-engine"
        }
      ],
      "evidence": [
        "Validator CTests and the smoke matrix exercise deterministic line/JSON parity, path safety, resource bounds, caller-authority rules, temporal rules, release rules, installation, and uninstall.",
        "qxctl validation tests verify exact installed invocation, complete-scan policy evaluation, baselines, and display-only debugging filters.",
        "Feature-administration tests exercise dynamic SSFV registry/profile closure, omit-self digests, expected qxctl command ordering and bindings, reference resolution, enforcement gates, and no-follow rejection without executing qxctl.",
        "Root-summary tests exercise deterministic Markdown and JSON projection, exact managed-region freshness, canonical source changes, and fail-closed malformed-state handling."
      ],
      "feature_id": "ssfv:symphony:symphony-validator",
      "how": "Bounded no-follow discovery and purpose-built detectors inspect canonical and implementation surfaces; feature-administration assurance derives the current SSFV set from its registry and validates the digest-bound profile and expected qxctl command inventory; root-summary assurance derives a bounded README region only after its canonical sources validate; every finding receives deterministic rule, subject, and occurrence identity before line or structured projection.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded no-follow repository discovery, deterministic rule evaluation, line and JSON evidence, and stable exit behavior."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and uninstalls the exact validator package."
        }
      ],
      "implementation_paths": [
        "tools/symphony-validator/CMakeLists.txt",
        "tools/symphony-validator/src/caller_authority.cpp",
        "tools/symphony-validator/src/cli.cpp",
        "tools/symphony-validator/src/feature_administration.cpp",
        "tools/symphony-validator/src/projector.cpp",
        "tools/symphony-validator/src/root_summary.cpp",
        "tools/symphony-validator/tests/smoke.sh"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not mutate scanned files, downgrade violations, suppress detector execution, auto-remediate, or treat baselines as ratification.",
        "Does not yet claim CI wiring, runtime AST analysis, or an overall production release."
      ],
      "owner_contract": "tools/symphony-validator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl administers exact validator invocation, presentation policy, baselines, and post-scan debug filters.",
          "target_feature_id": "ssfv:symphony:qxctl",
          "type": "composes_with"
        },
        {
          "rationale": "The validator uses the shared C++ packaging and bounded implementation conventions while retaining its own detector contract.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "tools/symphony-validator",
      "status": "experimental",
      "title": "Deterministic repository validation tool",
      "what": "Provides a deterministic read-only repository checker that executes the complete configured rule set, including static feature-to-administration closure and root-summary freshness, and emits stable finding identities, summaries, JSON evidence, Markdown root-summary evidence, and exit status.",
      "when": "Runs on explicit direct or qxctl invocation for repository review, debugging, baselining, or a separately wired gate.",
      "where": "Executes locally against one repository root from an exact receipt-v2 installation and writes no scanned content.",
      "who": "Repository maintainers, reviewers, qxctl callers, agentic tools, release gates, and debugging workflows.",
      "why": "Makes architectural and contract drift visible through reproducible evidence without allowing the detector, policy profiles, or baselines to change truth."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed invariant-assurance implementation changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the invariant contracts, machine registry, validator implementation, and evidence tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "Root-summary assurance projects public repository counts; invariant ownership assurance validates exact owner and regression routing and the implemented-module admission boundary.",
          "target_feature_id": "ssfv:symphony:symphony-validator.root-summary-assurance"
        }
      ],
      "evidence": [
        "The invariant-ownership CTest covers exact identifiers, recursive omit-self digest, ordering, adapter closure, evidence-role separation, named regressions, no-follow paths, IPC process mechanics, and malformed inputs.",
        "The feature-administration CTest proves an implemented independent module without FEATURES, SSFV routing, and profile mapping is rejected while documentation-only proposal seeds remain excluded.",
        "The smoke suite reserves exit 26 for invariant failure and proves direct apply rejection plus read-only repository-byte preservation."
      ],
      "feature_id": "ssfv:symphony:symphony-validator.invariant-ownership-assurance",
      "how": "A purpose-built C++ checker validates the incremental registry's exact constants, identifiers, digest, ordering, single ownership, referenced files and named tests, finite adapters, and IPC real-process evidence; the feature-administration checker separately enumerates implemented module roots from bounded build/source markers and rejects undeclared admission.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements deterministic read-only invariant and module-admission evidence with bounded no-follow filesystem access and stable exit behavior."
        }
      ],
      "implementation_paths": [
        "tools/symphony-validator/src/feature_administration.cpp",
        "tools/symphony-validator/src/invariant_ownership.cpp",
        "tools/symphony-validator/src/invariant_ownership.hpp",
        "tools/symphony-validator/tests/feature_administration_test.cpp",
        "tools/symphony-validator/tests/invariant_ownership_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not execute tests merely by finding named regression definitions; the build and test campaign remains separate execution evidence.",
        "Does not claim a complete legacy-invariant catalog, discover installed-host packages, authorize generic IPC adapters, mutate canonical knowledge, or remediate failures."
      ],
      "owner_contract": "tools/symphony-validator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:symphony-validator",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl supplies the stable headless query and exact installed-validator command surface.",
          "target_feature_id": "ssfv:symphony:qxctl.invariant-assurance",
          "type": "composes_with"
        },
        {
          "rationale": "Invariant assurance is one purpose-built rule family in the complete deterministic validator.",
          "target_feature_id": "ssfv:symphony:symphony-validator",
          "type": "extends"
        }
      ],
      "source_scope": "tools/symphony-validator",
      "status": "experimental",
      "title": "Invariant ownership and module-admission assurance",
      "what": "Verifies common cross-component invariant ownership and evidence routing and detects implemented source modules that omitted their same-change semantic and administrative declarations.",
      "when": "Runs during every complete repository validation and through the focused qxctl knowledge invariant check surface.",
      "where": "Executes locally inside the independently installed validator against one repository without following links or writing source.",
      "who": "Repository maintainers, independent module developers, reviewers, qxctl callers, and agentic integration workflows.",
      "why": "Closes the omission case where a new module supplies no descriptor or feature declaration while keeping semantic invention out of the engine, validator, qxctl, and AI."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to this implemented capability.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the canonical inputs, validator implementation evidence, and this distributed feature owner.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Completed SODV publication records are one of the validated source inputs projected into the root summary.",
          "reference": "knowledge/sodv/RELEASES.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The parent validator feature executes the full repository rule set and evidence projection; this subfeature narrowly derives and verifies one bounded repository-root README evidence region from already validated canonical inputs.",
          "target_feature_id": "ssfv:symphony:symphony-validator"
        }
      ],
      "evidence": [
        "The root-summary CTest rejects stale, missing, malformed, and source-divergent managed regions while proving deterministic JSON and Markdown projection.",
        "The complete repository scan runs root-summary freshness only after authoritative source-contract, SKVI, SCLV, SSFV administration, artifact, and build-integrity gates pass.",
        "qxctl invokes the exact receipt-backed validator root-summary operation and verifies the bounded digest-bearing JSON result before presentation."
      ],
      "feature_id": "ssfv:symphony:symphony-validator.root-summary-assurance",
      "how": "The C++ validator validates canonical SSFV routing and coverage, the digest-bound feature-administration profile and qxctl command registry, and completed SODV publication records; it then emits deterministic JSON or Markdown and compares the exact line-bounded README managed region without writing it.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Validates exact source evidence, derives the content-addressed projection, and enforces README managed-region freshness without mutation."
        },
        {
          "language": "Go",
          "role": "Provides the qxctl read-only administrative route and validates the exact installed validator result before presentation."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/validation.go",
        "tools/symphony-validator/src/cli.cpp",
        "tools/symphony-validator/src/root_summary.cpp",
        "tools/symphony-validator/src/root_summary.hpp",
        "tools/symphony-validator/tests/root_summary_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not make the README canonical, publish documentation, select release currentness, or infer feature-worthiness.",
        "Does not rewrite README, repair canonical sources, suppress underlying failures, or introduce a runtime service."
      ],
      "owner_contract": "tools/symphony-validator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:symphony-validator",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl provides the headless exact-installation administration route for this projection.",
          "target_feature_id": "ssfv:symphony:qxctl.governed-validation",
          "type": "composes_with"
        },
        {
          "rationale": "The root-summary operation is a bounded read-only assurance and projection surface within the complete validator.",
          "target_feature_id": "ssfv:symphony:symphony-validator",
          "type": "extends"
        }
      ],
      "source_scope": "tools/symphony-validator",
      "status": "experimental",
      "title": "Root summary assurance",
      "what": "Produces a deterministic, digest-bearing snapshot of registered semantic capabilities, reviewed administrative coverage, qxctl command identity, and completed source publications, and rejects a stale or malformed root README managed region.",
      "when": "Runs explicitly for JSON or Markdown projection and as the final derived gate in a complete repository validation after all authoritative source checks succeed.",
      "where": "Executes locally inside the exact installed validator against one repository and projects only the bounded root README evidence region.",
      "who": "Headless qxctl callers, repository maintainers, release reviewers, agents, and documentation workflows consuming current implemented-capability evidence.",
      "why": "Prevents the public repository summary from drifting into stale counts, missing implemented capabilities, or unsupported publication claims while preserving canonical truth in its owning SKV sources."
    }
  ],
  "source_scope": "tools/symphony-validator"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
