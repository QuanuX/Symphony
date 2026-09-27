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

The six research-data children have Contract Quads under `knowledge/sqv/sqav/`, `knowledge/sqv/sqmv/`, `knowledge/sqv/sqfv/`, `knowledge/sqv/sqtv/`, `knowledge/sqv/sqpv/`, and `knowledge/sqv/sqdv/`. SQMV, SQFV and SQPV own the bounded native libraries described below; SQAV, SQTV and SQDV currently define architecture ownership. Root knowledge-manifest and SKVI routes make these separate owners discoverable. SOOV's separate FIX Quad remains under `knowledge/sqv/soov/`.

## Implementation Status

SQFV's `modules/sqfv-batch-cpp/` owns the independently installable `0.2.0-dev` C++26 trusted-process batch library, with immutable payloads, explicit leases, independent port credits/cursors, and a local frame codec. SQMV's `modules/sqmv-metadata-cpp/` `0.1.0-dev` resolves bounded immutable manifests into exact metadata bindings. SQPV's `modules/sqpv-local-store-cpp/` `0.1.0-dev` retains one exact bound stream in a finite local filesystem store with explicit commit/recovery behavior. Each module's own contract and focused evidence define its limited capability. SQAV, SQTV, SQDV and SOOV remain architecture owners without implemented modules. Provider acquisition, domain interpretation, transport adapters, richer retention policies and consumer integrations remain later increments. The older `prototypes/sqv-research-data/` fixture is historical development evidence. None of these Quads allocates a colon identity family or SQV qxctl command.

## Non-Authorization Statement

This manifest does not authorize Symphony to inspect, rewrite, constrain, execute, deploy, or publish user strategy logic.
