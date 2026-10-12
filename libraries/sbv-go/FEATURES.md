# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "libraries/sbv-go/MANIFEST.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SBV owns quantitative meaning and replay/results.",
          "reference": "knowledge/sbv/SPEC.md",
          "vector": "sbv"
        },
        {
          "applicability": "applicable",
          "reason": "Local source closure retains implementation and verification provenance.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "Routes the independently selected module and evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "The adapter only transports and decodes; it does not implement native backtesting mathematics.",
          "target_feature_id": "ssfv:symphony:sbv-engine"
        }
      ],
      "evidence": [
        "Focused corruption, precision, ownership, cancellation and race checks.",
        "Independent installed consumer: retained Databento legacy/partitioned replay, external model/proposal admission and zero-work resume."
      ],
      "feature_id": "ssfv:symphony:sbv-go-adapter",
      "how": "Pins qxctl bytes, delegates exact owner requests, validates copied pointer/node streams and logical identities, and projects typed rows in caller feature order.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Standard-library transport and exact-value streaming/consumer shape validation; no scientific calculation."
        }
      ],
      "implementation_paths": [
        "libraries/sbv-go/client.go",
        "libraries/sbv-go/stream.go",
        "libraries/sbv-go/rows.go",
        "libraries/sbv-go/client_test.go",
        "libraries/sbv-go/stream_test.go",
        "libraries/sbv-go/examples/roundtrip/main.go"
      ],
      "kind": "feature",
      "non_claims": [
        "No scientific owner, calibrated predictions, adaptive optimizer or vendor ML integration.",
        "No borrowed Arrow/DLPack buffers, GPU runtime, native installation authority, remote execution or total dataset-size policy cap."
      ],
      "owner_contract": "libraries/sbv-go/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Native C++ owns scientific calculation and source/result admission.",
          "target_feature_id": "ssfv:symphony:sbv-engine",
          "type": "depends_on"
        },
        {
          "rationale": "All operation and discovery transport uses the same qxctl-first exact installed surface.",
          "target_feature_id": "ssfv:symphony:qxctl-sbv-administration",
          "type": "depends_on"
        }
      ],
      "source_scope": "libraries/sbv-go",
      "status": "experimental",
      "title": "Independent SBV Go data and control adapter",
      "what": "Transports every selected native SBV operation through qxctl and consumes exact streamed results/replay for independent Go model, optimizer and GUI pipelines.",
      "when": "When callers select the exact native installation and a Go consumer.",
      "where": "Separately versioned pure-Go source module compiled into a caller program; no standalone service or native installation ownership.",
      "who": "Independent Go consumers, user-authored model/proposal producers, agents and future GUI adapters.",
      "why": "Provides an optional language boundary without duplicating C++ calculations or imposing an external framework."
    }
  ]
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
