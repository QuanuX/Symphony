# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/stav-append-authority/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed append-authority changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the STAV authority contracts and implementation.",
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
          "reason": "STAV owns ledger, authorization, recovery, and projection semantics.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "STAV records operational audit events; SCLV validates canonical reviewed repository-change truth.",
          "target_feature_id": "ssfv:symphony:sclv-engine"
        }
      ],
      "evidence": [
        "Storage tests cover append ordering, checksums, digest-chain verification, restart idempotency, incomplete tails, and corruption refusal.",
        "Server and supervision tests cover kernel peer authorization, bounded IPC, endpoint identity, restart cadence, and conservative lifecycle."
      ],
      "feature_id": "ssfv:symphony:stav-append-authority",
      "how": "Kernel peer credentials, exact producer and reader grants, exclusive ledger locking, checksum framing, a preceding-event digest chain, fsync-before-receipt, startup reconstruction, bounded tail recovery, and conservative native supervision protect one append sequence.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free service, authenticated local IPC, serialized durable ledger, query projection, lifecycle, and native supervision descriptors."
        }
      ],
      "implementation_paths": [
        "modules/stav-append-authority/cmd/symphony-stav-append-authority/main.go",
        "modules/stav-append-authority/internal/server/server.go",
        "modules/stav-append-authority/internal/storage/ledger.go",
        "modules/stav-append-authority/internal/storage/ledger_test.go",
        "modules/stav-append-authority/internal/supervision/supervision.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not provide remote transport, signed non-repudiation, automatic retention deletion, rotation, or arbitrary repair.",
        "Does not accept raw qxctl append requests, secret values, provider payloads, or ungranted producer events."
      ],
      "owner_contract": "modules/stav-append-authority/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "All accepted candidates, frames, receipts, and queries use the canonical Go protocol kernel.",
          "target_feature_id": "ssfv:symphony:stav-protocol-kernel",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/stav-append-authority",
      "status": "experimental",
      "title": "Durable authenticated STAV append authority",
      "what": "Provides one independently installable per-TOPS authority that authenticates local peers, serializes append operations, assigns canonical ledger fields, commits tamper-evident events, and serves bounded verification and query projections.",
      "when": "Runs as a supervised foundational service or explicit development process and handles only bounded local STAV requests outside hot and warm paths.",
      "where": "Operates locally per TOPS with isolated configuration, socket, service identity, state directory, and append-only ledger.",
      "who": "Authorized local producers, authorized readers through qxctl, TOPS administrators, SSIAG, maintainers, and recovery tooling.",
      "why": "Creates one durable ordering and integrity authority for safe audit metadata without allowing producers, qxctl, or agents to edit ledger files."
    }
  ],
  "source_scope": "modules/stav-append-authority"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
