# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes SSIAG contracts and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future SSIAG release.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG vector truth owns the identity, policy, capability, provider, and safeguard semantics.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns the committed safe audit evidence required before decisions are released.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "SSIAG owns identity and authorization decisions; qxctl is the caller-neutral administrative client and does not own policy truth.",
          "target_feature_id": "ssfv:symphony:qxctl"
        }
      ],
      "evidence": [
        "Peer-authentication tests cover Darwin/Linux credential extraction, exact mapping, ambiguity refusal, and endpoint mismatch.",
        "Policy, server, STAV producer, lifecycle, and supervision tests cover exact grants, non-transferable bindings, audit-before-release, and per-TOPS isolation."
      ],
      "feature_id": "ssfv:symphony:ssiag-foundation",
      "how": "Darwin and Linux kernel peer credentials map exact UID/GID identities to canonical subjects; exact grants evaluate operation, resource, audience, and scope; SSIAG releases a decision only after the corresponding safe STAV event commits.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free per-TOPS service, kernel peer authentication, canonical subject mapping, exact policy decisions, endpoint trust, STAV production, and supervision."
        }
      ],
      "implementation_paths": [
        "modules/secure-identity-access-governance/cmd/symphony-ssiag/main.go",
        "modules/secure-identity-access-governance/internal/peerauth/peerauth.go",
        "modules/secure-identity-access-governance/internal/policy/policy.go",
        "modules/secure-identity-access-governance/internal/server/server.go",
        "modules/secure-identity-access-governance/internal/server/server_test.go",
        "modules/secure-identity-access-governance/internal/stavproducer/producer.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not infer authority from whether a caller is human, AI, agentic, automated, a service, or an organization.",
        "The current record does not claim canonical knowledge apply, operational credential delivery, remote access, or operational Keychain use."
      ],
      "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Audited authorization decisions fail closed unless the per-TOPS STAV authority commits their safe outcome evidence.",
          "target_feature_id": "ssfv:symphony:stav-append-authority",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/secure-identity-access-governance",
      "status": "experimental",
      "title": "Caller-neutral secure identity and access governance foundation",
      "what": "Provides independently installable per-TOPS local identity, authentication, exact deny-by-default authorization, non-transferable capability evidence, safe provider metadata, and audited decision services.",
      "when": "Used for explicit administrative cold or freezing-path operations requiring fresh target-host authorization; it is not consulted inline by hot or warm trading work.",
      "where": "Runs on the target TOPS node over a protected Unix socket with isolated configuration, service identity, runtime state, and native supervision.",
      "who": "Any target-host-authorized caller, qxctl, protected Symphony administrative consumers, TOPS owners, maintainers, and provider adapters.",
      "why": "Makes host ownership and granted permission—not caller class—the uniform gate for protected Symphony administration while preventing reusable bearer authority or secret leakage."
    }
  ],
  "source_scope": "modules/secure-identity-access-governance"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
