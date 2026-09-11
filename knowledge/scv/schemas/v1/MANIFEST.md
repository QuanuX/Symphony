# SCV Source-Knowledge Schemas v1 Manifest

## Canonical Surfaces

- `knowledge/scv/schemas/v1/MANIFEST.md`
- `knowledge/scv/schemas/v1/provider-coverage.schema.json`
- `knowledge/scv/schemas/v1/interpretation-profile.schema.json`
- `knowledge/scv/schemas/v1/provider-interpretation.schema.json`
- `knowledge/scv/schemas/v1/connection-operation.schema.json`
- `knowledge/scv/schemas/v1/capture.schema.json`
- `knowledge/scv/schemas/v1/capture-index.schema.json`
- `knowledge/scv/schemas/v1/corpus.schema.json`
- `knowledge/scv/schemas/v1/corpus-operation.schema.json`
- `knowledge/scv/schemas/v1/graph-operation.schema.json`
- `knowledge/scv/schemas/v1/interpretation-input.schema.json`
- `knowledge/scv/schemas/v1/knowledge-graph.schema.json`
- `knowledge/scv/schemas/v1/source-operation.schema.json`
- `knowledge/scv/schemas/v1/source.schema.json`

## Authority

`knowledge/scv/SOURCE-KNOWLEDGE.md` owns these closed local process data shapes. Structural validation is necessary but does not replace byte limits, strict STSC dates, digest verification, domain/lineage checks or evidence interpretation. These schemas do not define a network API, provider credential, durable commit or canonical mutation. Unknown authority roles and selectors remain opaque declarations.

`knowledge/scv/CORPUS.md` owns the additive capture-index, corpus and corpus-operation schemas introduced with engine `0.2.0-dev`. The original source, capture, source-operation, interpretation-input, knowledge-graph and graph-operation schemas remain unchanged. A corpus index references separately retained exact captures; schema validity does not prove those bytes exist.

`knowledge/scv/INTERPRETATION.md` owns the additive interpretation-profile, provider-interpretation and connection-operation schemas introduced with engine `0.3.0-dev`. The previous nine schemas remain unchanged. Profile replay, exact evidence qualification and connection-result reassessment supplement structural validation without proving empirical truth or selecting a graph head.

## Agent Operating Increment

`knowledge/scv/AGENT-WORKFLOWS.md` governs the following additive exact `.4` package resources:

- `knowledge/scv/schemas/v1/administration.schema.json`
- `knowledge/scv/schemas/v1/engine-descriptor-v2.schema.json`
- `knowledge/scv/schemas/v1/profile-preparation.schema.json`
- `knowledge/scv/schemas/v1/qxctl-error.schema.json`
- `knowledge/scv/schemas/v1/schema-catalog.json`
- `knowledge/scv/schemas/v1/schema-discovery.schema.json`
- `knowledge/scv/schemas/v1/scv-artifact.schema.json`
- `knowledge/scv/schemas/v1/scv-workflow.schema.json`

The catalog maps exact protocol IDs to closed schema definitions and explicit authoring templates. It is packaged with receipt ownership; the original source/knowledge/corpus/interpretation schemas remain unchanged. Templates are authoring aids, not evidence.

## Maintained Provider Coverage

`knowledge/scv/COVERAGE.md` owns the additive provider-coverage schema and `.5` catalog input/result entries. The `.5` installed catalog covers 74 protocols in 20 schema documents. Additive provider/coverage retention kinds use the existing provenance record shape. Earlier installed catalogs remain immutable; the twelve original source, corpus and interpretation schema files remain unchanged.
