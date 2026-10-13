# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "tools/sov-ssh-adapter/MANIFEST.md",
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
        "tools/sov-ssh-adapter/main_test.go",
        "tools/sov-ssh-adapter/contract/contract_test.go",
        "knowledge/sov/SSH-CARRIER.md"
      ],
      "feature_id": "ssfv:symphony:sov-ssh-adapter",
      "how": "Digest-bound public endpoint selection, staged carrier executable and complete typed native response validation with conservative lost-response outcomes.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Exact owner installation discovery, bounded process consumption, retained administrative state and agentic command grammar."
        }
      ],
      "implementation_paths": [
        "tools/sov-ssh-adapter/contract/contract.go",
        "tools/sov-ssh-adapter/main.go"
      ],
      "kind": "feature",
      "non_claims": [
        "No qualified two-host deployment, deployed SSIAG allow/expiry/revocation, cluster scheduling or remote provisioning.",
        "No native checkpoint resume, automatic retry, private-key extraction or proof of remote termination from disconnection."
      ],
      "owner_contract": "tools/sov-ssh-adapter/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl owns every endpoint field and explicit carrier invocation.",
          "target_feature_id": "ssfv:symphony:qxctl-sov-ssh-carrier",
          "type": "depends_on"
        }
      ],
      "source_scope": "tools/sov-ssh-adapter",
      "status": "experimental",
      "title": "Optional independently installed SOV SSH adapter",
      "what": "Exact caller-selected authenticated SSH carrier invocation and complete endpoint inspection through qxctl.",
      "when": "Explicit selected invocation or target configuration lifecycle.",
      "where": "Selected controller qxctl and independently installed network carrier; existing target qxctl.",
      "who": "Humans, agents and external consumers without caller-class distinctions.",
      "why": "Keep durable native calculations recoverable across controller boundaries without hidden replay."
    }
  ],
  "source_scope": "tools/sov-ssh-adapter"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
