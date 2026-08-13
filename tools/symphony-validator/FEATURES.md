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
        "Feature-administration tests exercise dynamic SSFV registry/profile closure, omit-self digests, expected qxctl command ordering and bindings, reference resolution, enforcement gates, and no-follow rejection without executing qxctl."
      ],
      "feature_id": "ssfv:symphony:symphony-validator",
      "how": "Bounded no-follow discovery and purpose-built detectors inspect canonical and implementation surfaces; feature-administration assurance derives the current SSFV set from its registry and validates the digest-bound profile and expected qxctl command inventory; every finding receives deterministic rule, subject, and occurrence identity before line or structured projection.",
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
        "tools/symphony-validator/tests/smoke.sh"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not mutate scanned files, downgrade violations, suppress detector execution, auto-remediate, or treat baselines as ratification.",
        "Does not yet claim CI wiring, Markdown projection, runtime AST analysis, or an overall production release."
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
      "what": "Provides a deterministic read-only repository checker that executes the complete configured rule set, including static feature-to-administration closure, and emits stable finding identities, summaries, JSON evidence, and exit status.",
      "when": "Runs on explicit direct or qxctl invocation for repository review, debugging, baselining, or a separately wired gate.",
      "where": "Executes locally against one repository root from an exact receipt-v2 installation and writes no scanned content.",
      "who": "Repository maintainers, reviewers, qxctl callers, agentic tools, release gates, and debugging workflows.",
      "why": "Makes architectural and contract drift visible through reproducible evidence without allowing the detector, policy profiles, or baselines to change truth."
    }
  ],
  "source_scope": "tools/symphony-validator"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
