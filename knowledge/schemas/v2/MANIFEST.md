# Symphony Knowledge Vector Common Schemas v2

## Authority

These exact JSON Schema files are canonical common lifecycle contract truth owned by the `knowledge/` umbrella. Version 2 does not replace or rewrite version 1 evidence; implementations must negotiate and dual-read every supported version explicitly.

## Schemas

- `install-receipt.schema.json`: immutable content-addressed package ownership, entry-point, capability, receptor-compatibility, and platform-requirement evidence. Activation, docking, selected-version, and mutable installation state are deliberately excluded.

## Compatibility Boundary

Receipt v1 remains valid historical evidence and is read through its exact adapter. Receipt v2 is a separately identified protocol. A reader must preserve unsupported receipts, must not synthesize absent v2 fields from v1, and must not execute a discovered entry point merely because it appears in a receipt.

All schemas use JSON Schema Draft 2020-12, close every common-governed object with `additionalProperties: false`, and carry no secrets.
