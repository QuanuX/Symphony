# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "tools/sov-stdio-adapter/MANIFEST.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Indexes the exact SOV handoff contract and implementation paths; SOV is not an admitted cross-vector enum in this selected SSFV v1 schema.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Optional target-local fresh authority; controller decisions never grant access.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        }
      ],
      "distinctions": [],
      "evidence": [
        "tools/sov-stdio-adapter/main_test.go",
        "tools/sov-stdio-adapter/README.md"
      ],
      "feature_id": "ssfv:symphony:sov-stdio-adapter",
      "how": "Streams one opaque request and reply through a private staged exact target executable.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Exact owner installation discovery, bounded process consumption, retained administrative state and agentic command grammar."
        }
      ],
      "implementation_paths": [
        "tools/sov-stdio-adapter/main.go",
        "tools/sov-stdio-adapter/main_test.go"
      ],
      "kind": "feature",
      "non_claims": [
        "No qualified remote network/cluster/shared-storage deployment or delegated remote principal.",
        "No native checkpoint resume, automatic retry, provisioning, live activity or calibrated performance superiority."
      ],
      "owner_contract": "tools/sov-stdio-adapter/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl owns all target selection/configuration/invocation control.",
          "target_feature_id": "ssfv:symphony:qxctl-sov-process-execution",
          "type": "depends_on"
        }
      ],
      "source_scope": "tools/sov-stdio-adapter",
      "status": "experimental",
      "title": "Optional SOV local process adapter",
      "what": "Separately built exact-digest local process carrier for one qxctl target invocation.",
      "when": "Explicit selected invocation or target configuration lifecycle.",
      "where": "Controller and target qxctl; stock adapter uses local processes.",
      "who": "Humans, agents and external consumers without caller-class distinctions.",
      "why": "Keep durable native calculations recoverable across controller boundaries without hidden replay."
    }
  ],
  "source_scope": "tools/sov-stdio-adapter"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
