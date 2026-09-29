# SKVI Schemas v2

## Canonical Surfaces

- `knowledge/skvi/schemas/v2/MANIFEST.md`
- `knowledge/skvi/schemas/v2/check-result.schema.json`
- `knowledge/skvi/schemas/v2/projection.schema.json`

## Authority

These JSON Schema Draft 2020-12 artifacts govern the SKVI `0.2.0-dev` check and projection results. They increase the result entry ceiling to 2,048 while preserving the v1 normalized entry and operation-payload schemas. The v1 check and projection schemas remain unchanged for exact `0.1.0-dev` compatibility.

## Schemas

- `check-result.schema.json`: deterministic structural check result with at most 2,048 entries checked.
- `projection.schema.json`: digest-bound disposable structural projection with at most 2,048 entries.

The snapshot-file ceiling remains 1,024 and the evidence-item ceiling remains 1,024. These schemas authorize no inferred membership, ratification, canonical write, active-version selection, or Maestro docking.
