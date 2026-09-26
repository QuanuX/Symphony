# Symphony Quantitative Flow Vector Manifest

## Canonical Target

`knowledge/sqv/sqfv/`

## Declared Contract Truth Role

SQFV owns the transfer, lifetime, ordering-context, and resource-bound contracts for selected SQV research-data movement.

## Canonical Surfaces

- `knowledge/sqv/sqfv/INTENT.md`
- `knowledge/sqv/sqfv/MANIFEST.md`
- `knowledge/sqv/sqfv/SPEC.md`
- `knowledge/sqv/sqfv/SKILL.md`

## Implementation Status

The architecture owner now admits the narrow `sqfv-batch-cpp` `0.1.0-dev` source module for trusted same-process immutable batches, ordered per-port admission, explicit leases, and a bounded local frame. Its exact library contract and test extent belong to `modules/sqfv-batch-cpp/`. The earlier offline prototype remains development evidence rather than an installed component. This admission creates no provider, process service, IPC or network transport, qxctl operation, throughput guarantee, or publication. No `sqfv:` identity family is allocated by this Quad.

## Language Boundary

First-party research-data batch, queue, and transport implementation on the data plane is native C++. The first module uses C++20 internally and an exact v1 C ABI and byte-defined local frame. Its tested toolchain is recorded with its implementation evidence. A C++ ABI, cross-process boundary, and broader platform matrix remain unadmitted.

## Non-Authorization Statement

This manifest authorizes no network listener, provider connection, credential use, private-data disclosure, durable commit, or live strategy execution.
