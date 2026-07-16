# STAV v1 Schema Collection

## Authority

This collection is canonical protocol truth owned by `knowledge/stav/`. JSON Schema Draft 2020-12 describes the ratified STAV v1 data model; Go types and generated documentation are subordinate implementations or projections.

## Files

- `common.schema.json`: shared scalar and tagged-value definitions.
- `candidate.schema.json`: untrusted producer-proposed candidate content.
- `event.schema.json`: canonical ten-group event content.
- `receipt.schema.json`: rejected and future committed receipt representation.
- `query.schema.json`: bounded forward-only query parameters.
- `query-page.schema.json`: redacted verification-aware result page.
- `verification.schema.json`: bounded chain-verification result.

Configuration, status, local request, and local response schemas are intentionally absent and remain owner gates.
