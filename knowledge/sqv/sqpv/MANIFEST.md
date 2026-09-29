# Symphony Quantitative Persistence Vector Manifest

## Canonical Target

`knowledge/sqv/sqpv/`

## Declared Contract Truth Role

SQPV owns selected research-data storage, retained positions, commit and read guarantees, integrity, retention, and recovery beneath SQV.

## Canonical Surfaces

- `knowledge/sqv/sqpv/INTENT.md`
- `knowledge/sqv/sqpv/MANIFEST.md`
- `knowledge/sqv/sqpv/SPEC.md`
- `knowledge/sqv/sqpv/SKILL.md`

## Implementation Status

The narrow `modules/sqpv-local-store-cpp/` `0.1.0-dev` C++26 library supplies one explicit local filesystem retention path with bounded capacity, exact SQMV/SQFV binding, serialized append, verified commit records and process-crash recovery. Its module SPEC owns exact file layout, synchronization, retry and read behavior. Other backends, pruning, replication and delivery profiles remain independent future work. No `sqpv:` identity family, service or SQV qxctl operation is allocated by this Quad.

## Language Boundary

First-party research-data storage bridges on the data plane use C++26. The first library pins exact native dependencies and states its tested platform and guarantee; later adapters require their own evidence.

## Non-Authorization Statement

This manifest authorizes no storage write, credential use, artifact deletion, private-data disclosure, remote commit, or live strategy execution.

## Non-live administration

`modules/sqpv-inspection-engine/MANIFEST.md` and `modules/sqpv-inspection-engine/SPEC.md` declare the independently installed C++26 owner adapter. Operations: `store_inspect`. Shared qxctl schema/template and source discovery supply the resource surface. Live runtime activation remains deferred.
