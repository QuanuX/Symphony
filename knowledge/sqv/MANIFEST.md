# Symphony Quantitative Vector Manifest

## Canonical Target

`knowledge/sqv/`

## Declared Contract Truth Role

SQV owns Symphony-authored quantitative framework semantics while preserving complete user ownership of strategy logic.

## Canonical Surfaces

- `knowledge/sqv/INTENT.md`
- `knowledge/sqv/MANIFEST.md`
- `knowledge/sqv/SPEC.md`
- `knowledge/sqv/SKILL.md`
- `knowledge/sqv/RESEARCH-DATA.md`

## Research-Data Child Posture

The six research-data children have Contract Quads under `knowledge/sqv/sqav/`, `knowledge/sqv/sqmv/`, `knowledge/sqv/sqfv/`, `knowledge/sqv/sqtv/`, `knowledge/sqv/sqpv/`, and `knowledge/sqv/sqdv/`. All six research-data owners have bounded first native libraries described below. Root knowledge-manifest and SKVI routes make these separate owners discoverable. SOOV's separate FIX Quad remains under `knowledge/sqv/soov/`.

## Implementation Status

SQFV's `modules/sqfv-batch-cpp/` owns the independently installable `0.2.0-dev` C++26 trusted-process batch library, with immutable payloads, explicit leases, independent port credits/cursors, and a local frame codec. SQMV's `modules/sqmv-metadata-cpp/` `0.1.0-dev` resolves bounded immutable manifests into exact metadata bindings. SQPV's `modules/sqpv-local-store-cpp/` `0.1.0-dev` retains one exact bound stream in a finite local filesystem store with explicit commit/recovery behavior. SQDV's `modules/sqdv-delivery-cpp/` `0.1.0-dev` supplies trusted same-process full-batch delivery, exact retained/live resume and independent processing acknowledgements. Each module's own contract and focused evidence define its limited capability. SQAV's `modules/sqav-capture-cpp/` `0.1.0-dev` preserves bounded original bytes and attributable acquisition evidence with an exact metadata/flow bridge. SQTV's `modules/sqtv-integer-conversion-cpp/` `0.1.0-dev` performs exact dense integer representation conversion with derived lineage. SOOV remains a separate architecture owner without an implemented module. Provider acquisition, domain interpretation, transport adapters, richer retention policies and consumer integrations remain later increments. The older `prototypes/sqv-research-data/` fixture is historical development evidence. None of these Quads allocates a colon identity family or SQV qxctl command.

## Non-Authorization Statement

This manifest does not authorize Symphony to inspect, rewrite, constrain, execute, deploy, or publish user strategy logic.
