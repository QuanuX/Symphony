# STAV v1 Schema Collection

## Authority

This collection is canonical protocol truth owned by `knowledge/stav/`. JSON Schema Draft 2020-12 describes the ratified STAV v1 data model; Go types and generated documentation are subordinate implementations or projections.

## Files

- `common.schema.json`: shared scalar and tagged-value definitions.
- `candidate.schema.json`: untrusted producer-proposed candidate content.
- `event.schema.json`: canonical ten-group event content.
- `receipt.schema.json`: rejected and durably committed receipt representation.
- `query.schema.json`: bounded forward-only query parameters.
- `query-page.schema.json`: redacted verification-aware result page.
- `verification.schema.json`: bounded chain-verification result.
- `append-authority-config.schema.json`: per-TOPS storage, IPC, and peer-grant contract.
- `append-authority-status.schema.json`: safe operational status projection.
- `local-request.schema.json`: authenticated local operation envelope.
- `local-response.schema.json`: authenticated local result envelope.

All v1 operational schemas listed above are Architect-ratified. Signed checkpoints and remote transport remain outside v1.
