# Symphony Knowledge Vector Common Schemas v2

## Canonical Surfaces

- `knowledge/schemas/v2/MANIFEST.md`
- `knowledge/schemas/v2/engine-binding-registry.schema.json`
- `knowledge/schemas/v2/engine-descriptor.schema.json`
- `knowledge/schemas/v2/install-receipt.schema.json`
- `knowledge/schemas/v2/invariant-ownership-registry.schema.json`
- `knowledge/schemas/v2/lifecycle-boot-head.schema.json`
- `knowledge/schemas/v2/lifecycle-boot-journal.schema.json`

## Authority

These exact JSON Schema files are canonical common lifecycle contract truth owned by the `knowledge/` umbrella. Version 2 does not replace or rewrite version 1 evidence; implementations must negotiate and dual-read every supported version explicitly.

## Schemas

- `engine-descriptor.schema.json`: side-by-side engine descriptor v2 with stable engine-operation IDs and administration semantics; installation state remains receipt and observation truth.
- `engine-binding-registry.schema.json`: side-by-side user-default exact-version selection with bounded extensible role identities, explicit predecessor-protocol evidence, and exact identity mappings for the eight roles implemented by the current qxctl.
- `install-receipt.schema.json`: immutable content-addressed package ownership, entry-point, capability, receptor-compatibility, and platform-requirement evidence. Root-level `.symphony-*` control files and the complete `share/symphony/receipts/` namespace are reserved and cannot be package-owned. Activation, docking, selected-version, and mutable installation state are deliberately excluded.
- `lifecycle-boot-journal.schema.json`: apply-capable dual-slot transaction evidence that links one exact report-only source journal to prepared and finalized action attempts, dynamic plan revisions, recovery, and content-addressed applied-state commitment.
- `lifecycle-boot-head.schema.json`: atomic selector for the active member of the apply-capable dual-slot journal without replacing or rewriting report-only v1 evidence.

## Compatibility Boundary

Engine descriptor v1 remains valid exact evidence. Descriptor v2 is a separately negotiated protocol and never infers or rewrites v1 fields. Receipt v1 remains valid historical evidence and is read through its exact adapter. Receipt v2 is a separately identified protocol. A reader must preserve unsupported receipts, must not synthesize absent v2 fields from v1, and must not execute a discovered entry point merely because it appears in a receipt.

Engine-binding registry v1 remains the exact closed six-role protocol defined by its schema. Registry v2 is a distinct write protocol with a 256-binding bound and token-shaped role identities. The schema fixes the current eight role-to-module/engine mappings without treating an otherwise valid unknown role as understood by an older implementation. Current qxctl may preserve and display such v2 state but fails operational use and mutation until a compatible implementation understands every role. Migration from a valid v1 generation, or from the specifically recognized historical qxctl representation that placed SAV or SEV in a v1 envelope, is an explicit exact-digest compare-and-swap into a new v2 generation. Migration copies the existing binding values, records the predecessor digest and protocol, and performs no installed-version discovery or recency inference.

Lifecycle journal v1 remains the immutable report-only transaction and source-authorization record. Journal v2 is a side-by-side apply protocol with its own lock, slots, head, compatibility capabilities, and recovery chain. It may read and reference v1 evidence but never rewrites or upgrades a v1 stream in place. Older readers preserve v2 state and remain read-only; version recency alone never proves compatibility.

All schemas use JSON Schema Draft 2020-12, close every common-governed object with `additionalProperties: false`, and carry no secrets.

## Invariant Registry v2

`invariant-ownership-registry.schema.json` adds explicitly declared generic exact-receipt C++ engine adapters through adapter format 2. Legacy adapter format 1 keeps its existing closed identity/protocol pairs. Registry v1 remains a distinct supported read format; v2 never silently widens v1. Each generic entry binds its component, actual entrypoint, module owner, operation namespace, finite operation set and real-process evidence. Presence is traceability, not permission to execute.
