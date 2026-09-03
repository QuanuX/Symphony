# STAV v1 Conformance Fixture Collection

## Canonical Surfaces

- `knowledge/stav/fixtures/v1/MANIFEST.md`
- `knowledge/stav/fixtures/v1/invalid/candidate-duplicate-key.json`
- `knowledge/stav/fixtures/v1/invalid/candidate-null.json`
- `knowledge/stav/fixtures/v1/invalid/local-request-multiple-payloads.json`
- `knowledge/stav/fixtures/v1/invalid/local-response-wrong-payload.json`
- `knowledge/stav/fixtures/v1/invalid/query-float.json`
- `knowledge/stav/fixtures/v1/invalid/query-unknown-field.json`
- `knowledge/stav/fixtures/v1/invalid/query-unsafe-integer.json`
- `knowledge/stav/fixtures/v1/valid/append-authority-config.json`
- `knowledge/stav/fixtures/v1/valid/append-authority-status.json`
- `knowledge/stav/fixtures/v1/valid/candidate.json`
- `knowledge/stav/fixtures/v1/valid/event.json`
- `knowledge/stav/fixtures/v1/valid/local-request-status.json`
- `knowledge/stav/fixtures/v1/valid/local-response-status.json`
- `knowledge/stav/fixtures/v1/valid/query-page.json`
- `knowledge/stav/fixtures/v1/valid/query.json`
- `knowledge/stav/fixtures/v1/valid/receipt-rejected.json`
- `knowledge/stav/fixtures/v1/valid/verification.json`

## Authority

These fixtures are canonical interoperability evidence for the STAV v1 profile. Valid files must round-trip to identical canonical bytes. Invalid files must fail for the named strict-profile or typed-shape violation.

## Valid

- `valid/candidate.json`
- `valid/event.json`
- `valid/receipt-rejected.json`
- `valid/query.json`
- `valid/query-page.json`
- `valid/verification.json`

## Invalid

- `invalid/candidate-duplicate-key.json`
- `invalid/candidate-null.json`
- `invalid/query-float.json`
- `invalid/query-unsafe-integer.json`
- `invalid/query-unknown-field.json`

Invalid UTF-8, unpaired surrogate, noncharacter, excessive-depth, partial-frame, and truncated-input seeds are constructed in the pure-Go kernel tests because canonical Markdown/JSON files cannot safely represent every byte sequence.
