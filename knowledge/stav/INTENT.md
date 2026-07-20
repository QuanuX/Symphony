# Symphony TOPS Audit Vector Intent

## Purpose

STAV defines the canonical, system-facing protocol for safe, per-TOPS administrative and security audit events. It is directly usable by authorized human, AI, service, workload, and other caller types through the same governed interfaces.

## Canonical and Operational Separation

`knowledge/stav/` owns protocol truth only. Runtime ledgers live in per-installation state outside the repository. No runtime event may be written into a Git working tree.

## Integrity Posture

STAV v1 is tamper-evident through a monotonic sequence and preceding-event digest chain. It is not non-repudiable. Signed checkpoints are deferred until a future threat model requires them.

## Authority Intent

One dedicated Go append-authority process per TOPS serialization domain validates and serializes events over authenticated local IPC. Authorized runtime components submit candidate events. qxctl administers and queries. Caller type does not determine authority: direct ledger mutation and arbitrary append are unsupported for every caller, while configured producer and reader grants determine protocol access.

The independently installable implementation lives at `modules/stav-append-authority/` and implements this vector; it does not own or redefine STAV protocol truth.

Shared pure-Go protocol mechanics live at `libraries/stav-protocol-go/`. The library is a build-time implementation without a resident, installer, state, transport, authorization, or ledger authority.

The authority is part of the foundational bootstrap stratum. Supervision owns liveness only and never transfers ledger or producer authority.

## Privacy Intent

STAV records allowlisted outcomes and administrative metadata, never security proofs, assertions, tokens, credentials, provider payloads, secret-bearing errors, or routine heartbeats. STAV has no default remote aggregation or phone-home behavior.
