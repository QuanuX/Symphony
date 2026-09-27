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

The six research-data children have Contract Quads under `knowledge/sqv/sqav/`, `knowledge/sqv/sqmv/`, `knowledge/sqv/sqfv/`, `knowledge/sqv/sqtv/`, `knowledge/sqv/sqpv/`, and `knowledge/sqv/sqdv/`. SQFV owns the first admitted library described below; the other five currently define architecture ownership. Root knowledge-manifest and SKVI routes make these separate owners discoverable. SOOV's separate FIX Quad remains under `knowledge/sqv/soov/`.

## Implementation Status

SQFV's `modules/sqfv-batch-cpp/` owns the independently installable `0.2.0-dev` C++26 trusted-process batch library, with immutable payloads, explicit leases, independent port credits/cursors, and a local frame codec. Its exact module contract and focused evidence define that limited capability. The other research-data children and SOOV remain architecture owners without their own implemented engines. Provider acquisition, retention/recovery, transport adapters, and consumer integrations remain later increments. The older `prototypes/sqv-research-data/` fixture is historical development evidence. None of these Quads allocates a colon identity family or SQV qxctl command.

## Non-Authorization Statement

This manifest does not authorize Symphony to inspect, rewrite, constrain, execute, deploy, or publish user strategy logic.
