# STAV v1 Conformance Fixture Collection

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
