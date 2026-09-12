# SCV Source-Knowledge Schemas v1 Manifest

## Canonical Surfaces

- `knowledge/scv/schemas/v1/MANIFEST.md`
- `knowledge/scv/schemas/v1/bundle.schema.json`
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

## Portable Packages and Composition

The exact `.6` catalog adds ten protocol entries, totaling 84 across 23 schema documents. Eight owner companions and a receipt-owned owner declaration support discovery without a checkout. Its new closed schemas are:

- `knowledge/scv/schemas/v1/owner-interface.schema.json`, owned by `knowledge/scv/OWNER-INTERFACE.md`;
- `knowledge/scv/schemas/v1/provider-pack.schema.json`, owned by `knowledge/scv/PROVIDER-PACKS.md`;
- `knowledge/scv/schemas/v1/composition.schema.json`, owned by `knowledge/scv/COMPOSITION.md`.

Artifact admission, installation versions and schema discovery remain exact. Bound maxima do not promise that every nested maximum fits simultaneously in the shared process envelope. Schemas describe shape; native replay and independent consumers verify provenance and interpretation.

## Maintained Composition Workflow

`knowledge/scv/COMPOSITION-WORKFLOWS.md` owns `knowledge/scv/schemas/v1/scv-composition-workflow.schema.json`. Exact `.7` discovery adds five protocol entries for run/status/recover inputs, the retained run and result. The catalogue contains 89 entries across 24 schemas with nine owner companions. Artifact records admit exact `.7` owners while preserving earlier release admissions; the native operation inventory remains 26.

## Precise Obligation Follow-up

`knowledge/scv/OBLIGATIONS.md` owns `knowledge/scv/schemas/v1/obligation.schema.json` and `knowledge/scv/schemas/v1/scv-obligation-link.schema.json`. The exact `.8` catalog contains 97 protocols in 26 schemas. Native replay, precise target identity, caller criterion equality and retained-record correspondence are separate runtime checks.

## Exact Evidence Bundles

`knowledge/scv/BUNDLES.md` owns `knowledge/scv/schemas/v1/bundle.schema.json`. The exact `.9` catalog contains 103 protocols in 27 schemas. Structural validity is supplemented by independent closure, digest, owner, expansion budget and native semantic checks. Existing artifact records admit the two new operation kinds without changing legacy input limits.
