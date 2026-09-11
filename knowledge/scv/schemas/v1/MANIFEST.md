# SCV Source-Knowledge Schemas v1 Manifest

## Canonical Surfaces

- `knowledge/scv/schemas/v1/MANIFEST.md`
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
