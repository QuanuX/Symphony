# Accordare STAV Producer Specification

## Inputs

The v1 local protocol separates `prepare` and `complete`. Prepare contains the original qxctl Named Version command, operation, TOPS ID, and coordinator component/version/receipt/executable digests with explicit null terminal fields. Complete repeats those exact identities and adds either the original successful coordinator result or one closed failed/unavailable outcome without a payload. Each local envelope binds an independent request UUID used only for IPC response matching.

The command must contain an unexpired SSIAG allow decision and nontransferable capability for the exact qxctl operation, normalized Named Version resource, TOPS scope, qxctl audience, and kernel-authenticated submitter subject. Caller-class-dependent or canonical-apply authority is prohibited.

## Derivation

Only `named_version_prepare`, `named_version_seal`, `named_version_alias`, and `named_version_recover` terminal outcomes are accepted. Successful completion requires and recomputes the result digest. Failed and unavailable completion accepts no result or raw error and derives unchanged digest metadata. Every outcome maps mechanically to its canonical `symphony.sav.named-version.*` tuple. RFC 9562 v4-form request/correlation UUIDs derive deterministically from TOPS, operation, and stable operation ID.

SSIAG compact subject kinds such as `owner` are mechanically projected to the STAV namespace as `symphony.identity.owner`; already namespaced kinds remain exact. The opaque subject ID never changes.

## Durability and Recovery

Before coordinator mutation, the authenticated intent is written under `intents-v1` with mode `0600`, file sync, atomic rename, and directory sync. Its UUID is deterministic from TOPS, operation, and stable operation ID. An exact retry is idempotent even when authorization timestamps differ; any command, coordinator, peer, operation, or TOPS conflict fails closed. Intent preparation time is an STSC whole second used for capability validity, never operation ordering.

The exact safe candidate and its digest are written with mode `0600`, file sync, atomic rename, and directory sync before any append attempt. At most 10,000 entries may exist. Symlinks, unexpected filenames, duplicate request IDs with different evidence, invalid candidates, or mismatched digests fail closed.

After STAV returns an exact committed receipt, the outbox entry and then the bound intent are removed with directory synchronization. A transport or cleanup failure remains pending. Status reports `intent_pending` and `append_pending` separately. Startup and explicit `reconcile` replay candidates in deterministic request-ID order; unresolved intents require exact command retry. STAV and coordinator idempotency make identical replay safe.

## Grant Contract

The installation grant uses producer reference `accordare-stav-producer` / `symphony.stav.producer`, the enrolled producer UID/GID and service subject, and exactly four `(event_class, operation_id)` permissions. qxctl may mutate the STAV configuration only while its socket is absent, after an exact SSIAG grant-administration decision, and when `--expected-config-digest` matches the observed bounded regular file. A durable old/new digest attempt marker allows the next invocation to close a crash that occurred on either side of atomic replacement.

## Bounds

- local frame: 1 MiB;
- command evidence: 1 MiB;
- result evidence: 1 MiB;
- configuration: 1 MiB;
- intent entries: 10,000;
- outbox entries: 10,000;
- concurrent connections: 64;
- connection deadline: 5 seconds;
- supported hosts: macOS and Linux.
