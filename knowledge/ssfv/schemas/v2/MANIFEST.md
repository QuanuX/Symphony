# SSFV Executable Protocol Schema Manifest v2

## Purpose

Define the executable SSFV check, diff, proposal, feature-record, and registry-entry shapes completed after the contract-only v1 transition.

## Canonical Schemas

- `feature-record.schema.json`
- `registry-entry.schema.json`
- `check-input.schema.json`
- `check-result.schema.json`
- `diff-input.schema.json`
- `diff-result.schema.json`
- `proposal-input.schema.json`

The v2 operation schemas use `knowledge/ssfv/schemas/v1/semantic-snapshot.schema.json` and the embedded feature-file v1 envelope. Existing v1 schemas remain historical canonical contract evidence and are not silently reinterpreted.

## Status

Architect-ratified implementation contract. These schemas authorize bounded read, comparison, proposal, and projection behavior only.
