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
- `producer-vocabulary.schema.json`: authority-free closed producer-integration vocabulary; validity does not implement a producer or grant an installation.
- `accordare-producer-config.schema.json`: exact per-TOPS producer identity, submitter, socket, vocabulary, and STAV routing contract.
- `accordare-producer-submission.schema.json`: bounded qxctl/coordinator Named Version evidence accepted by the producer.
- `accordare-producer-local-request.schema.json`: authenticated local submit, status, and reconciliation envelope.
- `accordare-producer-local-response.schema.json`: committed, durable-pending, status, and rejection result envelope.

All fifteen v1 operational schemas listed above are Architect-ratified. The producer-vocabulary schema also governs contract-only reserved integrations. Signed checkpoints and remote transport remain outside v1.
