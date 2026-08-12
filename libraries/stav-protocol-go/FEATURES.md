# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "libraries/stav-protocol-go/MANIFEST.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed protocol-kernel changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes STAV protocol contracts and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV owns release truth for the published Go source module.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns the canonical protocol semantics implemented by this library.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "This kernel implements STAV-specific Go protocol truth; the C++ foundation implements cross-vector administrative engine mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation"
        }
      ],
      "evidence": [
        "The module contains strict codecs, digest and frame implementations, conformance fixtures, and Go tests.",
        "The published v0.2.0 source module is independently importable without an authority process."
      ],
      "feature_id": "ssfv:symphony:stav-protocol-kernel",
      "how": "Strict bounded codecs, canonical digests, checksummed framing, exact identifier validation, and fixture-backed conformance are compiled into each consumer without granting storage or append authority.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free canonical STAV codec, validation, digest, framing, identifier, and conformance mechanics."
        }
      ],
      "implementation_paths": [
        "libraries/stav-protocol-go/canonical_test.go",
        "libraries/stav-protocol-go/codec.go",
        "libraries/stav-protocol-go/digest.go",
        "libraries/stav-protocol-go/frame.go",
        "libraries/stav-protocol-go/validate.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not append, store, authorize, supervise, query a live ledger, or own producer identity.",
        "Does not make library callers trusted and does not carry credential or secret material."
      ],
      "owner_contract": "libraries/stav-protocol-go/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The append authority uses the protocol kernel for all accepted and emitted STAV evidence.",
          "target_feature_id": "ssfv:symphony:stav-append-authority",
          "type": "enables"
        }
      ],
      "source_scope": "libraries/stav-protocol-go",
      "status": "experimental",
      "title": "Authority-free STAV protocol kernel",
      "what": "Provides reusable authority-free Go types and deterministic validation for STAV candidates, events, receipts, local IPC envelopes, queries, and ledger frames.",
      "when": "Used whenever a Go component constructs, validates, persists, verifies, or queries STAV v1 evidence.",
      "where": "Compiled as an independent Go module and linked into Go STAV producers, clients, and the append authority outside hot and warm trading paths.",
      "who": "STAV producers, append authorities, qxctl clients, maintainers, and tests that require one exact safe audit-event protocol.",
      "why": "Prevents audit producers and consumers from inventing incompatible framing, validation, or digest rules while keeping authority outside the library."
    }
  ],
  "source_scope": "libraries/stav-protocol-go"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
