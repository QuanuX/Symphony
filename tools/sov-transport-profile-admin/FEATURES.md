# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "tools/sov-transport-profile-admin/SPEC.md",
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
        "tools/sov-transport-profile-admin/contract/contract_test.go",
        "tools/sov-transport-profile-admin/internal/store/store_test.go",
        "tools/sov-transport-profile-admin/internal/store/packages_test.go",
        "tools/sov-transport-profile-admin/installed_test.go"
      ],
      "feature_id": "ssfv:symphony:sov-transport-profile-admin",
      "how": "Independent Go helper, fixed sudo invocation, root-owned descriptor-relative store, coherent snapshots and committed SSIAG/STAV audit.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Owns protected local administration and validates exact authenticated audit."
        }
      ],
      "implementation_paths": [
        "tools/sov-transport-profile-admin/contract/types.go",
        "tools/sov-transport-profile-admin/contract/validate.go",
        "tools/sov-transport-profile-admin/internal/auditclient/client.go",
        "tools/sov-transport-profile-admin/internal/store/store_unix.go",
        "tools/sov-transport-profile-admin/main.go"
      ],
      "kind": "feature",
      "non_claims": [
        "No I3 connection admission, endpoint-v2 execution, live Cloudflare session or provider resource operations.",
        "No deployed sudo/SSIAG/STAV qualification follows from local fixtures; no host policy is installed automatically."
      ],
      "owner_contract": "tools/sov-transport-profile-admin/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [],
      "source_scope": "tools/sov-transport-profile-admin",
      "status": "experimental",
      "title": "Protected local transport profile administration",
      "what": "Closed local profile/policy changes with exact generations, protected package trust, authenticated audit and explicit recovery.",
      "when": "Explicit local profile inspection, planning, mutation or recovery.",
      "where": "Selected local host and TOPS protected store.",
      "who": "Humans, agents and external consumers without caller-class distinctions.",
      "why": "Permit optional transport composition while keeping privilege limited to protected administration."
    }
  ],
  "source_scope": "tools/sov-transport-profile-admin"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
