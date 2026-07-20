# STAV Append Authority Skill

## Safe Caller Actions

A caller may inspect, build, test, verify configuration shape, use qxctl read-only commands, propose administrative configuration changes, or perform an available state-changing operation only when its target-host permissions authorize that operation. Caller type is not evaluated. The current qxctl STAV commands remain read-only for every caller.

Install, enroll, supervisor install/uninstall, serve, unenroll, and purge are state-changing operations. The caller must hold the applicable target-host permission and explicitly invoke the exact operation. Purge is destructive even though it is scoped to one TOPS.

## Prohibited Actions

No supported caller operation may:

- edit, append, truncate, rotate, replace, or repair a ledger file;
- invoke or create a raw append surface;
- manufacture producer/reader permissions or treat socket access as authorization;
- make qxctl, SSIAG, node-troll, a supervisor, or any other component a secondary writer;
- record proofs, assertions, tokens, credentials, provider payloads, or secret-bearing errors;
- bypass endpoint/caller authentication, fsync-before-receipt, or evidence preservation;
- introduce HTTP/OpenAPI or remote producer ingestion;
- treat module code as the owner of STAV protocol truth.

## Review Procedure

1. Read `knowledge/stav/SPEC.md`, registry, schemas, and this module's Contract Quad.
2. Confirm the change preserves one TOPS, separate socket/ledger locks, one writer, exact peer grants, and liveness-only supervision.
3. Run protocol fixtures, storage/recovery tests, authenticated-socket tests, race tests, and cgo-disabled builds.
4. Verify qxctl remains read-only and SSIAG emits only its closed vocabulary.
5. Stop on any new recovery behavior, writer, transport, secret field, or authority expansion without Architect ratification.
