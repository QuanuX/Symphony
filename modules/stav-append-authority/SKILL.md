# STAV Append Authority Skill

## Safe Agent Actions

Agents may inspect, build, test, verify configuration shape, and use qxctl read-only commands when the Architect or operator authorizes access. Agents may propose administrative configuration changes but must not apply them without explicit authority.

Install, enroll, serve, unenroll, and purge are state-changing operator actions and require explicit user direction. Purge is destructive even though it is scoped to one TOPS.

## Prohibited Actions

Agents must never:

- edit, append, truncate, rotate, replace, or repair a ledger file;
- invoke or create a raw append surface;
- grant themselves producer/reader permissions or treat socket access as authorization;
- make qxctl, SSIAG, node-troll, a supervisor, or an agent a secondary writer;
- record proofs, assertions, tokens, credentials, provider payloads, or secret-bearing errors;
- bypass endpoint/caller authentication, fsync-before-receipt, or evidence preservation;
- introduce HTTP/OpenAPI or remote producer ingestion;
- treat module code as the owner of STAV protocol truth.

## Review Procedure

1. Read `knowledge/stav/SPEC.md`, registry, schemas, and this module's Contract Quad.
2. Confirm the change preserves one TOPS, one lock, one writer, and exact peer grants.
3. Run protocol fixtures, storage/recovery tests, authenticated-socket tests, race tests, and cgo-disabled builds.
4. Verify qxctl remains read-only and SSIAG emits only its closed vocabulary.
5. Stop on any new recovery behavior, writer, transport, secret field, or authority expansion without Architect ratification.
