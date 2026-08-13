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
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed append-authority changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the STAV authority contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs future append-authority release and documentation truth.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns reader grants, verification, query, and projection semantics.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Produces authorized read projections; qxctl presents them but owns neither reader grants nor ledger semantics.",
          "target_feature_id": "ssfv:symphony:qxctl.stav-administration"
        }
      ],
      "evidence": [
        "Server and ledger tests cover reader resolution, event-class filtering, bounded status and verification, ascending conjunctive query, paging, and strict response binding."
      ],
      "feature_id": "ssfv:symphony:stav-append-authority.authorized-query",
      "how": "Authenticates kernel peer credentials to one reader grant, validates bounded status, verify, and query requests, applies event-class filtering before other projection logic, preserves source event identity, digest, and sequence, and emits only strict response envelopes.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements cgo-free authenticated read dispatch, event-class-scoped ledger projections, verification, and typed client responses."
        }
      ],
      "implementation_paths": [
        "modules/stav-append-authority/client/client.go",
        "modules/stav-append-authority/internal/server/server.go",
        "modules/stav-append-authority/internal/server/server_test.go",
        "modules/stav-append-authority/internal/storage/ledger.go",
        "modules/stav-append-authority/internal/storage/ledger_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not provide append, raw expressions, SQL, JSONPath, regex, descending or unbounded query, direct file access, remote export, or canonical projection identity."
      ],
      "owner_contract": "modules/stav-append-authority/SPEC.md",
      "parent_feature_id": "ssfv:symphony:stav-append-authority",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl composes and presents the authority response without acquiring ledger authority.",
          "target_feature_id": "ssfv:symphony:qxctl.stav-administration",
          "type": "composes_with"
        },
        {
          "rationale": "All reader projections depend on a verified durable canonical ledger chain.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.ledger-durability",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/stav-append-authority",
      "status": "experimental",
      "title": "Event-class-scoped STAV query and verification",
      "what": "Provides bounded status, chain verification, and ascending conjunctive query projections while omitting events outside the reader event-class allowlist.",
      "when": "During explicit cold or freezing-path audit inspection, doctor composition, verification, and paged evidence retrieval.",
      "where": "At the read operations of the per-TOPS append-authority socket over the verified in-memory canonical chain.",
      "who": "Exactly granted local readers, qxctl STAV administration, TOPS operators, and maintainers.",
      "why": "Audit consumers need safe evidence access without direct ledger-file authority, unbounded expressions, misleading partial redaction, or a competing source of truth."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed append-authority changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the STAV authority contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs future append-authority release and documentation truth.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns durable ledger, integrity, and bounded recovery semantics.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Owns checksummed file framing, integrity, and recovery; serialized append owns producer authentication, authorization, ordering assignment, and commitment dispatch.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.serialized-append"
        }
      ],
      "evidence": [
        "Ledger tests cover checksums, predecessor chains, sequence and TOPS binding, fsync commitment, restart idempotency, incomplete tails, evidence preservation, and corruption refusal."
      ],
      "feature_id": "ssfv:symphony:stav-append-authority.ledger-durability",
      "how": "Holds an exclusive operating-system file lock; writes length, canonical event, and SHA-256 of event bytes; synchronizes before commitment; scans every frame, chain link, sequence, TOPS binding, and reconstructed request identity at startup; preserves and truncates only an incomplete final frame; and refuses complete corruption.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free checksummed file frame, exclusive ledger lock, integrity scan, durable append, idempotency reconstruction, and bounded tail recovery."
        }
      ],
      "implementation_paths": [
        "modules/stav-append-authority/internal/storage/ledger.go",
        "modules/stav-append-authority/internal/storage/ledger_test.go",
        "modules/stav-append-authority/internal/storage/platform_unix.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Provides tamper evidence, not non-repudiation; it does not sign checkpoints, rotate, delete by retention, export remotely, repair complete corruption, or salvage middle frames."
      ],
      "owner_contract": "modules/stav-append-authority/SPEC.md",
      "parent_feature_id": "ssfv:symphony:stav-append-authority",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Canonical event bytes, event digests, and semantic validation come from the authority-free protocol kernel.",
          "target_feature_id": "ssfv:symphony:stav-protocol-kernel",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/stav-append-authority",
      "status": "experimental",
      "title": "Tamper-evident STAV ledger durability and bounded recovery",
      "what": "Persists one finite per-TOPS append-only ledger with checksummed records, a predecessor digest chain, fsync-before-receipt durability, restart idempotency reconstruction, and evidence-preserving incomplete-tail recovery.",
      "when": "At authority startup, every successful append, verification, and recovery from an interrupted final write.",
      "where": "In one private ledger file and private recovery-evidence directory per TOPS.",
      "who": "The append authority, authorized verification consumers, TOPS administrators, and recovery maintainers.",
      "why": "A committed receipt must mean durable ordered evidence, while self-healing must never hide complete corruption or manufacture history."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed append-authority changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the STAV authority contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs future append-authority release and documentation truth.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns native supervision and socket lifecycle boundaries.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Manages STAV process liveness only; the SSIAG sibling supervises a separate identity and authorization service and neither starts the other.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.native-supervision"
        }
      ],
      "evidence": [
        "Supervision and socket-lock tests cover exact launchd and systemd descriptors, authority identity, atomic replacement, stale-socket proof, restart cadence, and graceful shutdown."
      ],
      "feature_id": "ssfv:symphony:stav-append-authority.native-supervision",
      "how": "Renders identity-bound launchd or systemd descriptors, verifies configured authority UID and GID and exact descriptor content, commits changes atomically, applies bounded restart and shutdown settings, serializes socket ownership through a persistent lock, and removes only provably stale sockets.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements cgo-free native descriptor management, authority identity checks, socket lifecycle locking, stale recovery, and graceful shutdown."
        }
      ],
      "implementation_paths": [
        "modules/stav-append-authority/cmd/symphony-stav-append-authority/main.go",
        "modules/stav-append-authority/internal/foundation/engine.go",
        "modules/stav-append-authority/internal/foundation/engine_test.go",
        "modules/stav-append-authority/internal/foundation/transaction.go",
        "modules/stav-append-authority/internal/server/socketlock_test.go",
        "modules/stav-append-authority/internal/server/socketlock_unix.go",
        "modules/stav-append-authority/internal/supervision/supervision.go",
        "modules/stav-append-authority/internal/supervision/supervision_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not create principals, grant producer, reader, or ledger authority, use supervisor socket activation, make supervised mode a credential, or add a dependency on SSIAG."
      ],
      "owner_contract": "modules/stav-append-authority/SPEC.md",
      "parent_feature_id": "ssfv:symphony:stav-append-authority",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "STAV and SSIAG have independent supervision and neither service starts the other.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.native-supervision",
          "type": "distinguished_from"
        }
      ],
      "source_scope": "modules/stav-append-authority",
      "status": "experimental",
      "title": "Per-TOPS native STAV append-authority supervision",
      "what": "Installs, starts, stops, and removes one bounded per-TOPS native liveness descriptor and protects the authority-owned socket lifecycle.",
      "when": "For supported production append-authority lifecycle; direct user serve remains a development or diagnostic route.",
      "where": "In per-user or system launchd or systemd domains and the per-TOPS STAV runtime socket namespace on macOS or Linux.",
      "who": "TOPS owners, target-host administrators, launchd or systemd, owner-provided equivalent supervisors, and maintainers.",
      "why": "The foundational audit service requires stable liveness independent of SSIAG so neither service becomes the other bootstrap authority."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed append-authority changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the STAV authority contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs future append-authority release and documentation truth.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns producer grants, trusted event assignment, ordering, and receipt semantics.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Owns producer authentication, permission, serialization, and trusted field assignment; ledger durability owns checksummed persistence, startup scan, and crash recovery.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.ledger-durability"
        }
      ],
      "evidence": [
        "Server, peer resolver, client, and storage tests cover exact producer grants, pair allowlists, canonical field assignment, concurrent ordering, idempotency, and committed or rejected receipts."
      ],
      "feature_id": "ssfv:symphony:stav-append-authority.serialized-append",
      "how": "Resolves Darwin or Linux kernel peer credentials to one producer grant, validates a bounded canonical request, checks the event-class and operation pair allowlist, executes one locked append path, and preserves request-ID idempotency.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements cgo-free authenticated append dispatch, producer authorization, trusted event-field assignment, serialization, idempotency, and safe receipts."
        }
      ],
      "implementation_paths": [
        "modules/stav-append-authority/client/client.go",
        "modules/stav-append-authority/internal/peerauth/resolver.go",
        "modules/stav-append-authority/internal/server/server.go",
        "modules/stav-append-authority/internal/server/server_test.go",
        "modules/stav-append-authority/internal/storage/ledger.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not permit raw qxctl append, direct ledger editing, remote transport, arbitrary producer identity, or ungranted event tuples."
      ],
      "owner_contract": "modules/stav-append-authority/SPEC.md",
      "parent_feature_id": "ssfv:symphony:stav-append-authority",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "A safe committed receipt requires the durability subsystem to finish the checksummed append and synchronization.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.ledger-durability",
          "type": "depends_on"
        },
        {
          "rationale": "All submitted candidates and emitted events and receipts use the canonical protocol kernel.",
          "target_feature_id": "ssfv:symphony:stav-protocol-kernel",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/stav-append-authority",
      "status": "experimental",
      "title": "Authenticated serialized STAV commitment",
      "what": "Authenticates a local producer, enforces its exact event-class and operation allowlist, serializes concurrent submissions, assigns canonical producer, event, time, sequence, and predecessor fields, and returns one safe committed or rejected receipt.",
      "when": "For each local STAV candidate submitted to one running per-TOPS authority.",
      "where": "At the append operation of the protected per-TOPS Unix-socket service and ledger writer.",
      "who": "Exactly granted local producers, SSIAG, the ledger subsystem, and maintainers.",
      "why": "Producers must not assign trusted ledger identity or ordering, race one another, or gain file mutation merely because they can construct valid candidates."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed append-authority changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the STAV authority contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs future append-authority release and documentation truth.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns per-TOPS authority identity, grants, storage, and enrollment semantics.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Binary installation copies executable bytes; enrollment establishes one trusted audit domain and grants no producer or reader by default.",
          "target_feature_id": "ssfv:symphony:qxctl.lifecycle-convergence"
        }
      ],
      "evidence": [
        "Enrollment, configuration, lifecycle, and path tests cover exact authority identity, empty grants, protected user/system roots, preservation, active-listener refusal, and explicit purge."
      ],
      "feature_id": "ssfv:symphony:stav-append-authority.tops-enrollment",
      "how": "Requires an installed binary, validates exact user or system authority identity, creates protected scope-specific paths, writes bounded configuration and an enrollment marker atomically, refuses active-service purge, and removes data only on explicit purge.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements cgo-free per-TOPS enrollment, strict authority configuration, protected path creation, preservation, active-service checks, and explicit purge."
        }
      ],
      "implementation_paths": [
        "modules/stav-append-authority/cmd/symphony-stav-append-authority/main.go",
        "modules/stav-append-authority/internal/config/config.go",
        "modules/stav-append-authority/internal/foundation/engine.go",
        "modules/stav-append-authority/internal/foundation/engine_test.go",
        "modules/stav-append-authority/internal/foundation/transaction.go",
        "modules/stav-append-authority/internal/lifecycle/enrollment.go",
        "modules/stav-append-authority/internal/lifecycle/enrollment_test.go",
        "modules/stav-append-authority/internal/paths/paths.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not create a system principal, grant append or read permission, create a ledger event, expose purge through machine/qxctl v1, or claim ordinary audit/reconciliation success without a real STAV receipt."
      ],
      "owner_contract": "modules/stav-append-authority/SPEC.md",
      "parent_feature_id": "ssfv:symphony:stav-append-authority",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Module enrollment establishes the TOPS-local serialization domain while qxctl lifecycle convergence coordinates exact installed component state.",
          "target_feature_id": "ssfv:symphony:qxctl.lifecycle-convergence",
          "type": "distinguished_from"
        }
      ],
      "source_scope": "modules/stav-append-authority",
      "status": "experimental",
      "title": "Per-TOPS STAV serialization-domain enrollment",
      "what": "Establishes or removes one isolated audit serialization domain with an immutable TOPS UUID, explicit authority UID and GID, strict endpoint trust, empty-by-default producer and reader grants, private ledger, recovery, and runtime roots, and preserved-by-default data.",
      "when": "Before serving, supervision, granting producers or readers, or creating the first event for a TOPS.",
      "where": "In the selected per-TOPS STAV configuration, state, ledger, recovery, and runtime namespaces.",
      "who": "TOPS owners, target-host administrators, installation tooling, and maintainers.",
      "why": "Every TOPS requires an independent audit ordering and integrity domain, even when several instances share one host."
    }
  ],
  "source_scope": "modules/stav-append-authority"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
