# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "INTENT.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to the platform capability and its semantic record.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes canonical root governance and the registered root feature file without owning its semantics.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication derived from this canonical capability record.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [],
      "evidence": [
        "INTENT.md defines the root purpose, modular-sovereignty boundary, installability expectations, and caller-neutral host authority.",
        "README.md identifies the implemented foundations, proposal-only runtime seeds, active-development state, and module-based release posture.",
        "knowledge/SPEC.md defines the common independently installed vector-engine architecture and thermal isolation rules.",
        "go.work composes the current Go administration and service modules for monorepo development without creating a monolithic runtime dependency."
      ],
      "feature_id": "ssfv:symphony:platform",
      "how": "Root governance and canonical cross-vector contracts establish shared invariants, while the monorepo co-locates implementation and evidence for inspection. Separately installable modules retain their own identity, state, version, configuration, and lifecycle boundaries.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "The root Go workspace composes current Go modules for repository development without making the workspace a runtime dependency."
        },
        {
          "language": "Markdown",
          "role": "Canonical root governance and project orientation define the platform boundary, implemented state, relationships, and non-claims."
        }
      ],
      "implementation_paths": [
        "INTENT.md",
        "README.md",
        "go.work",
        "knowledge/SPEC.md"
      ],
      "kind": "capability",
      "non_claims": [
        "Does not claim an overall production release, monolithic installation, or universal infrastructure dependency.",
        "Does not claim proposal-only node-troll, bus-troll, or hotpath-runtime seeds are implemented.",
        "Does not establish market-data, order-flow, or broader trading-node doctrine.",
        "Does not classify caller type as a source of authority."
      ],
      "owner_contract": "INTENT.md",
      "parent_feature_id": null,
      "record_version": 2,
      "relationships": [],
      "source_scope": ".",
      "status": "experimental",
      "title": "Modular Symphony platform boundary",
      "what": "Provides the open-source, modular platform boundary that unifies canonical knowledge, implementation, integration contracts, and validation evidence while preserving independently installable module sovereignty.",
      "when": "Applies whenever Symphony source, contracts, modules, or tools are developed, reviewed, integrated, installed, or operated. The current lifecycle is active development rather than an overall production launch.",
      "where": "Owned at the repository root and realized through the monorepo's canonical governance and module boundaries. Runtime deployment occurs only through separately selected and installed components.",
      "who": "Target-host owners, administrators, maintainers, integrators, agentic tools, and independently deployed Symphony components that need one inspectable platform boundary.",
      "why": "Keeps the system coherent and agent-inspectable without sacrificing modular deployment, bounded authority, provider neutrality, or the ability to compose bespoke installations."
    }
  ],
  "source_scope": "."
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
