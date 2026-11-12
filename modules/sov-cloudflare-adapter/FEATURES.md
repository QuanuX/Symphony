# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sov-cloudflare-adapter/SPEC.md",
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
        "modules/sov-cloudflare-adapter/tests/installed.cpp",
        "modules/sov-cloudflare-adapter/tests/contract.cpp",
        "modules/sov-cloudflare-adapter/tests/qualification.cpp",
        "modules/sov-cloudflare-adapter/tests/execution.cpp",
        "modules/sov-cloudflare-adapter/tests/connector.cpp",
        "modules/sov-cloudflare-adapter/tests/guard.mjs",
        "modules/sov-cloudflare-adapter/tests/access.cpp"
      ],
      "feature_id": "ssfv:symphony:sov-cloudflare-adapter",
      "how": "Closed receipt-owned schemas, exact runtime selection, SSIAG pinned admission and committed audit, one-request guard, private journals and observation-only recovery.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Owns native semantics."
        }
      ],
      "implementation_paths": [
        "modules/sov-cloudflare-adapter/src/access.cpp",
        "modules/sov-cloudflare-adapter/src/access_main.cpp",
        "modules/sov-cloudflare-adapter/src/adapter.cpp",
        "modules/sov-cloudflare-adapter/src/connector.cpp",
        "modules/sov-cloudflare-adapter/src/credential_channel.cpp",
        "modules/sov-cloudflare-adapter/src/execution.cpp",
        "modules/sov-cloudflare-adapter/src/http_runner.cpp",
        "modules/sov-cloudflare-adapter/src/main.cpp",
        "modules/sov-cloudflare-adapter/src/project.cpp",
        "modules/sov-cloudflare-adapter/src/resource.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No live provider or SSH qualification, token-rotation command, automatic root service installation or protection against hostile same-UID/root processes."
      ],
      "owner_contract": "modules/sov-cloudflare-adapter/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [],
      "source_scope": "modules/sov-cloudflare-adapter",
      "status": "experimental",
      "title": "Native Cloudflare administration",
      "what": "Exact cf/Node lifecycle, typed Tunnel/DNS/Access operations, native credential channels, persisted contexts and user-mode connector lifecycle. Explicit private Access user-session lifecycle and fixed SSH proxy helper. Selected project file observations and exact persisted preparation/migration proposals without configuration execution.",
      "when": "Explicitly planned user-owned Cloudflare administration.",
      "where": "Selected independently installed native adapter and private local store.",
      "who": "Humans, agents and independent native consumers.",
      "why": "Administer an independent Cloudflare CLI without ambient selection or premature provider authority."
    }
  ],
  "source_scope": "modules/sov-cloudflare-adapter"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
