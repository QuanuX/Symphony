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
      "how": "Strict bounded canonicalization, typed codecs, domain-separated digests, semantic validation, and four-byte length-prefixed local IPC framing are compiled into each consumer without granting storage, append, or durable ledger-frame authority.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free canonical STAV codec, validation, digest, framing, identifier, and conformance mechanics."
        }
      ],
      "implementation_paths": [
        "libraries/stav-protocol-go/canonical.go",
        "libraries/stav-protocol-go/canonical_test.go",
        "libraries/stav-protocol-go/codec.go",
        "libraries/stav-protocol-go/digest.go",
        "libraries/stav-protocol-go/frame.go",
        "libraries/stav-protocol-go/validate.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not implement the checksummed durable ledger frame, append, store, authorize, supervise, query a live ledger, or own producer identity.",
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
      "what": "Provides reusable authority-free Go types, canonical bytes and digests, deterministic semantic validation, typed codecs, and bounded local IPC framing for STAV candidates, events, receipts, queries, and request or response envelopes.",
      "when": "Used whenever a Go component constructs, validates, persists, verifies, or queries STAV v1 evidence.",
      "where": "Compiled as an independent Go module and linked into Go STAV producers, clients, and the append authority outside hot and warm trading paths.",
      "who": "STAV producers, append authorities, qxctl clients, maintainers, and tests that require one exact safe audit-event protocol.",
      "why": "Prevents audit producers and consumers from inventing incompatible framing, validation, or digest rules while keeping authority outside the library."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed STAV protocol-kernel changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the protocol contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV owns release truth for the published Go source module and future documentation projections.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns the canonical byte, digest, codec, and local framing semantics implemented here.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Implements canonical document bytes, digests, and local IPC length framing; checksummed durable ledger records belong to append-authority ledger durability.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.ledger-durability"
        }
      ],
      "evidence": [
        "Canonical, codec, digest, frame, and fixture tests cover I-JSON and RFC 8785 constraints, byte equality, tagged hashes, and bounded frame lengths."
      ],
      "feature_id": "ssfv:symphony:stav-protocol-kernel.canonical-wire-representation",
      "how": "Parses the STAV I-JSON subset; rejects unsafe Unicode, numbers, null, duplicate names, and trailing data; emits RFC 8785 ordering; requires typed decode bytes to equal canonical re-encoding; applies tagged SHA-256 domains; and checks announced frame length before allocation.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements standard-library-only canonical JSON, typed codecs, domain-separated digests, and bounded four-byte local IPC frames without cgo."
        }
      ],
      "implementation_paths": [
        "libraries/stav-protocol-go/canonical.go",
        "libraries/stav-protocol-go/canonical_test.go",
        "libraries/stav-protocol-go/codec.go",
        "libraries/stav-protocol-go/digest.go",
        "libraries/stav-protocol-go/fixtures_test.go",
        "libraries/stav-protocol-go/frame.go",
        "libraries/stav-protocol-go/frame_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not open a transport, write storage, implement the ledger checksum record, assign ordering, authenticate a caller, or grant authority."
      ],
      "owner_contract": "libraries/stav-protocol-go/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:stav-protocol-kernel",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The durable ledger uses canonical event bytes but owns its separate checksummed file-frame contract.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.ledger-durability",
          "type": "distinguished_from"
        },
        {
          "rationale": "Canonical bytes, digests, and frames enable all append-authority request and evidence processing.",
          "target_feature_id": "ssfv:symphony:stav-append-authority",
          "type": "enables"
        }
      ],
      "source_scope": "libraries/stav-protocol-go",
      "status": "experimental",
      "title": "Canonical STAV bytes, digests, and local frames",
      "what": "Produces and requires one strict canonical byte representation, domain-separated candidate, event, and genesis digests, and bounded four-byte-length local IPC frames.",
      "when": "Whenever a Go consumer encodes, decodes, hashes, or locally transports STAV content.",
      "where": "In the authority-free stavprotocol library linked into producers, clients, tests, and the authority.",
      "who": "STAV producers, append authorities, local clients, qxctl consumers, fixtures, and maintainers.",
      "why": "Byte-identical representations and identity digests prevent consumers from accepting semantically ambiguous or incompatibly encoded audit evidence."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed STAV protocol-kernel changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the protocol contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV owns release truth for the published Go source module and future documentation projections.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns the exact content, identifier, presence, grant, query, and response semantics implemented here.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Determines whether content conforms to STAV v1; serialized append proves authority provenance, assigns trusted fields, and commits accepted content.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.serialized-append"
        }
      ],
      "evidence": [
        "Identifier, validation, fixture, and codec tests cover exact event groups, not-applicable variants, bounds, unions, grants, query grammar, ordering fields, and response binding."
      ],
      "feature_id": "ssfv:symphony:stav-protocol-kernel.semantic-validation",
      "how": "Typed model validators enforce all ten event groups, explicit not-applicable variants, canonical UUID, identifier, digest, and time forms, safe integer ceilings, exact query and grant bounds, unique permissions, strict payload unions, and response binding.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements standard-library-only STAV models, identifiers, exact validators, presence rules, unions, and canonical conformance fixtures without cgo."
        }
      ],
      "implementation_paths": [
        "libraries/stav-protocol-go/fixtures_test.go",
        "libraries/stav-protocol-go/identifiers.go",
        "libraries/stav-protocol-go/identifiers_test.go",
        "libraries/stav-protocol-go/model.go",
        "libraries/stav-protocol-go/validate.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not authorize a producer or reader, assign event fields, persist or query a ledger, supervise a process, or make a caller trusted."
      ],
      "owner_contract": "libraries/stav-protocol-go/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:stav-protocol-kernel",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Content conformance is distinct from authenticated assignment and durable commitment by the append authority.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.serialized-append",
          "type": "distinguished_from"
        },
        {
          "rationale": "Shared exact validation enables the authority and its producers and consumers to accept one STAV v1 language.",
          "target_feature_id": "ssfv:symphony:stav-append-authority",
          "type": "enables"
        }
      ],
      "source_scope": "libraries/stav-protocol-go",
      "status": "experimental",
      "title": "Exact STAV content and identifier validation",
      "what": "Validates the exact closed content, presence rules, identifiers, bounds, unions, ordering fields, query grammar, configuration, and local request and response shapes for STAV v1.",
      "when": "Before encoding, after decoding, before dispatch, during conformance testing, and whenever canonical STAV evidence is constructed or consumed.",
      "where": "In the authority-free stavprotocol model, identifier, validator, and canonical fixture code.",
      "who": "STAV producers, append authorities, readers, qxctl clients, conformance tests, and maintainers.",
      "why": "Shared exact acceptance and rejection rules prevent each producer, authority, and client from inventing its own STAV semantics."
    }
  ],
  "source_scope": "libraries/stav-protocol-go"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
