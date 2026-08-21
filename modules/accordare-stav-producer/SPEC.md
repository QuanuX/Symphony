# Accordare STAV Producer Specification

## Inputs

The v1 submit operation accepts exactly one `symphony.accordare.stav-producer.submission.v1` containing the original qxctl Named Version command, the original coordinator result, the operation name, the TOPS ID, and coordinator component/version/receipt/executable digests. The local envelope also binds an independent request UUID used only for IPC response matching.

The command must contain an unexpired SSIAG allow decision and nontransferable capability for the exact qxctl operation, normalized Named Version resource, TOPS scope, qxctl audience, and kernel-authenticated submitter subject. Caller-class-dependent or canonical-apply authority is prohibited.

## Derivation

Only `named_version_prepare`, `named_version_seal`, `named_version_alias`, and `named_version_recover` successful results are accepted. The producer recomputes the result digest and maps each operation to its canonical `symphony.sav.named-version.*` tuple. It derives the target only from validated output digests and derives RFC 9562 v4-form request/correlation UUIDs deterministically from TOPS, operation, and stable operation ID.

SSIAG compact subject kinds such as `owner` are mechanically projected to the STAV namespace as `symphony.identity.owner`; already namespaced kinds remain exact. The opaque subject ID never changes.

## Durability and Recovery

The exact safe candidate and its digest are written with mode `0600`, file sync, atomic rename, and directory sync before any append attempt. At most 10,000 entries may exist. Symlinks, unexpected filenames, duplicate request IDs with different evidence, invalid candidates, or mismatched digests fail closed.

After STAV returns an exact committed receipt, the outbox entry is removed and its directory synchronized. A transport failure or post-commit cleanup failure remains `pending`. Startup and explicit `reconcile` replay candidates in deterministic request-ID order. STAV idempotency makes identical replay safe.

## Grant Contract

The installation grant uses producer reference `accordare-stav-producer` / `symphony.stav.producer`, the enrolled producer UID/GID and service subject, and exactly four `(event_class, operation_id)` permissions. qxctl may mutate the STAV configuration only while its socket is absent, after an exact SSIAG grant-administration decision, and when `--expected-config-digest` matches the observed bounded regular file. A durable old/new digest attempt marker allows the next invocation to close a crash that occurred on either side of atomic replacement.

## Bounds

- local frame: 1 MiB;
- command evidence: 1 MiB;
- result evidence: 1 MiB;
- configuration: 1 MiB;
- outbox entries: 10,000;
- concurrent connections: 64;
- connection deadline: 5 seconds;
- supported hosts: macOS and Linux.
