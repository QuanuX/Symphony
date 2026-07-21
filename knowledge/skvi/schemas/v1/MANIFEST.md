# SKVI Schemas v1

## Authority

These JSON Schema Draft 2020-12 artifacts are canonical SKVI operation and result truth. The common immutable proposal envelope remains owned by `knowledge/schemas/v1/proposal.schema.json`.

## Schemas

- `entry.schema.json`: normalized projected SKVI entry.
- `operation-payload.schema.json`: exact caller-declared proposal input.
- `check-result.schema.json`: deterministic structural check result and evidence.
- `projection.schema.json`: digest-bound disposable structural projection.

Every governed object is closed. These schemas authorize no inferred membership, ratification, canonical write, active-version selection, or Maestro docking.
