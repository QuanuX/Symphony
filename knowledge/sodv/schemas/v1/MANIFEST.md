# SODV Engine Schema Manifest

## Canonical Surfaces

- `knowledge/sodv/schemas/v1/MANIFEST.md`
- `knowledge/sodv/schemas/v1/check-result.schema.json`
- `knowledge/sodv/schemas/v1/observed-state.schema.json`
- `knowledge/sodv/schemas/v1/projection.schema.json`
- `knowledge/sodv/schemas/v1/proposal-input.schema.json`
- `knowledge/sodv/schemas/v1/recovery-input.schema.json`
- `knowledge/sodv/schemas/v1/recovery-result.schema.json`
- `knowledge/sodv/schemas/v1/release-record-v2.schema.json`
- `knowledge/sodv/schemas/v1/verify-result.schema.json`

This directory owns the machine-readable contracts for the initial Symphony Official Documentation Vector engine.

| Schema | Purpose |
|---|---|
| `release-record-v2.schema.json` | Provider-neutral forward release record proposed for `RELEASES.md` |
| `observed-state.schema.json` | Caller-supplied immutable tag and public-artifact observations |
| `check-result.schema.json` | Read-only canonical ledger validation result |
| `verify-result.schema.json` | Noncanonical comparison of authorization and observed state |
| `proposal-input.schema.json` | Caller-declared forward-record proposal input |
| `recovery-input.schema.json` | Safe noncanonical release-transaction reconciliation input |
| `recovery-result.schema.json` | Non-mutating reconciliation result |
| `projection.schema.json` | Disposable inventory derived from the canonical release ledger |

These operational schemas do not authorize public-documentation schemas, tags, uploads, package publication, canonical apply, Mintlify configuration, or release completion.
