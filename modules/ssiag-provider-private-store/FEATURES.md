# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/ssiag-provider-private-store/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Source evolution and scope claims remain reviewable through SCLV.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "The exact module contract, implementation and focused evidence have independent indexed routes.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [],
      "evidence": [
        "modules/ssiag-provider-private-store/tests/contract.cpp",
        "modules/sov-cloudflare-adapter/tests/execution.cpp",
        "modules/secure-identity-access-governance/internal/server/cloudflare_use_test.go"
      ],
      "feature_id": "ssfv:symphony:ssiag-provider-private-store",
      "how": "Kernel peer and executable binding, exact SSIAG permission and audit, purpose/scope/generation/expiry checks and private framing.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns native semantics."
        }
      ],
      "implementation_paths": [
        "modules/ssiag-provider-private-store/include/symphony/ssiag/private_channel.hpp",
        "modules/ssiag-provider-private-store/src/main.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No OS-keychain encryption, hostile same-UID/root isolation, live Cloudflare qualification or provider-side token revocation."
      ],
      "owner_contract": "modules/ssiag-provider-private-store/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [],
      "source_scope": "modules/ssiag-provider-private-store",
      "status": "experimental",
      "title": "Native private credential channels",
      "what": "Opt-in user-owned private-file credential backend and authenticated native use/import channels.",
      "when": "Explicit native backend selection and admitted Cloudflare use.",
      "where": "Independent SSIAG native provider and private user store.",
      "who": "Humans, agents and independent native consumers.",
      "why": "Keep credential bytes out of qxctl and Go while retaining exact use authority."
    }
  ],
  "source_scope": "modules/ssiag-provider-private-store"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
