# Symphony Quantitative Delivery Vector Manifest

## Canonical Target

`knowledge/sqv/sqdv/`

## Declared Contract Truth Role

SQDV owns selected research-data consumer views, recipient scope, delivery-state meaning at its admitted boundary, and consumer resume beneath SQV.

## Canonical Surfaces

- `knowledge/sqv/sqdv/INTENT.md`
- `knowledge/sqv/sqdv/MANIFEST.md`
- `knowledge/sqv/sqdv/SPEC.md`
- `knowledge/sqv/sqdv/SKILL.md`

## Implementation Status

`modules/sqdv-delivery-cpp/` implements the first bounded C++26 full-batch delivery library at `0.1.0-dev`: independent recipient allowances, exact metadata/view identity, explicit processing checkpoints and retained-to-live resume through SQFV and the selected SQPV backend. Provider-specific recipient adapters, access enforcement, external destinations, SQV qxctl surfaces and performance thresholds remain future work. No `sqdv:` identity family is allocated by this Quad.

## Language Boundary

First-party research-data delivery adapters on the data plane use C++26 in this effort. The first trusted same-process source interface has an exact package contract; external receiving interfaces, access enforcement and remote destination evidence require independent selection and verification.

## Non-Authorization Statement

This manifest authorizes no recipient access, external data export, credential use, private-data disclosure, remote ingestion, or live strategy execution.

## Non-live administration

`modules/sqdv-checkpoint-engine/MANIFEST.md` and `modules/sqdv-checkpoint-engine/SPEC.md` declare the independently installed C++26 owner adapter. Operations: `checkpoint_inspect`. Shared qxctl schema/template and source discovery supply the resource surface. Live runtime activation remains deferred.
