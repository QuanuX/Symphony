# SCLV Version 3 Schema Manifest

## Authority

These schemas are canonical protocol truth for prospective provider-neutral SCLV v3 records and the proposal-only SCLV engine. They are not generated projections.

## Exact Files

- `record.schema.json` defines one normalized canonical v3 record.
- `proposal-input.schema.json` defines caller-declared proposal input plus bounded provider evidence.
- `recovery-input.schema.json` defines non-mutating ephemeral-journal reconciliation input.
- `check-result.schema.json` defines deterministic ledger diagnostics.
- `projection.schema.json` defines the disposable provider-neutral ledger projection.

The common normalized adapter envelope is `knowledge/schemas/v1/provider-evidence.schema.json`.

## Boundary

Schema validity does not grant repository permission, ratification, apply authority, journal deletion, canonical append, release publication, or provider trust.
