# Accordare STAV Producer Manifest

## Identity

- module: `modules/accordare-stav-producer/`
- Go module: `github.com/QuanuX/Symphony/modules/accordare-stav-producer`
- executable: `symphony-accordare-stav-producer`
- source version: `0.1.0-dev`
- implementation: Go, cgo-free
- thermal path: freezing

## Implemented Capabilities

- immutable prefix-based receipt-v2 install and replay plus receipt-verified uninstall that refuses retained per-TOPS enrollment references;
- per-TOPS user/system enrollment with exact service and submitter UID/GID/SSIAG subject binding;
- bounded local framed protocol with mutual kernel endpoint authentication;
- strict duplicate-detecting evidence JSON that preserves required explicit-null fields;
- closed verification of SSIAG decision/capability and coordinator result self-digest;
- deterministic safe-candidate construction for prepare, seal, alias, and recover succeeded, failed, and unavailable outcomes;
- private fsync-before-visible pre-mutation intent storage, semantic retry, and conflict refusal;
- private fsync-before-visible outbox, exact replay, collision refusal, and reconciliation;
- bounded socket concurrency, socket lifecycle lease, stale-socket recovery, and graceful shutdown;
- qxctl two-phase Named Version preparation/completion plus committed/intent-pending/append-pending presentation;
- qxctl authenticated producer status, exact stable-candidate reconciliation, and receipt-bound native supervisor install/uninstall;
- qxctl exact-grant install/remove with SSIAG authorization, stopped-authority rule, config CAS, durable attempt marker, and two-sided recovery.

## Deliberately Absent

- arbitrary append, read, ledger edit, repair, HTTP, TCP, NATS, or remote transport;
- Named Version bodies, SSIAG proofs, capabilities, paths, aliases, or raw errors in the outbox or STAV candidate;
- automatic grants from package presence, enrollment, vocabulary validity, or socket access;
- inferred terminal outcomes or raw error text; failed/unavailable production requires the closed typed evidence contract;
- Windows-native implementation; Windows users require WSL or a remote TOPS node;
- hot-path, warm-path, trading-runtime, CUDA, or accelerator involvement.
