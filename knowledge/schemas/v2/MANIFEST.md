# Symphony Knowledge Vector Common Schemas v2

## Authority

These exact JSON Schema files are canonical common lifecycle contract truth owned by the `knowledge/` umbrella. Version 2 does not replace or rewrite version 1 evidence; implementations must negotiate and dual-read every supported version explicitly.

## Schemas

- `engine-descriptor.schema.json`: side-by-side engine descriptor v2 with stable engine-operation IDs and administration semantics; installation state remains receipt and observation truth.
- `install-receipt.schema.json`: immutable content-addressed package ownership, entry-point, capability, receptor-compatibility, and platform-requirement evidence. Root-level `.symphony-*` control files and the complete `share/symphony/receipts/` namespace are reserved and cannot be package-owned. Activation, docking, selected-version, and mutable installation state are deliberately excluded.
- `lifecycle-boot-journal.schema.json`: apply-capable dual-slot transaction evidence that links one exact report-only source journal to prepared and finalized action attempts, dynamic plan revisions, recovery, and content-addressed applied-state commitment.
- `lifecycle-boot-head.schema.json`: atomic selector for the active member of the apply-capable dual-slot journal without replacing or rewriting report-only v1 evidence.

## Compatibility Boundary

Engine descriptor v1 remains valid exact evidence. Descriptor v2 is a separately negotiated protocol and never infers or rewrites v1 fields. Receipt v1 remains valid historical evidence and is read through its exact adapter. Receipt v2 is a separately identified protocol. A reader must preserve unsupported receipts, must not synthesize absent v2 fields from v1, and must not execute a discovered entry point merely because it appears in a receipt.

Lifecycle journal v1 remains the immutable report-only transaction and source-authorization record. Journal v2 is a side-by-side apply protocol with its own lock, slots, head, compatibility capabilities, and recovery chain. It may read and reference v1 evidence but never rewrites or upgrades a v1 stream in place. Older readers preserve v2 state and remain read-only; version recency alone never proves compatibility.

All schemas use JSON Schema Draft 2020-12, close every common-governed object with `additionalProperties: false`, and carry no secrets.
