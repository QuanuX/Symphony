# Symphony Quantitative Acquisition Vector Manifest

## Canonical Target

`knowledge/sqv/sqav/`

## Declared Contract Truth Role

SQAV owns research-data source collection semantics beneath SQV. It preserves source-specific authority and evidence rather than defining one universal provider interface.

## Canonical Surfaces

- `knowledge/sqv/sqav/INTENT.md`
- `knowledge/sqv/sqav/MANIFEST.md`
- `knowledge/sqv/sqav/SPEC.md`
- `knowledge/sqv/sqav/SKILL.md`

## Implementation Status

`modules/sqav-capture-cpp/` owns the first `0.1.0-dev` C++26 optional offline original-byte capture library, its exact binary format, metadata/flow bridge and package. Provider connectors, provider-specific parsing, live operations and SQV qxctl remain later work. No `sqav:` identity family is allocated by this Quad.

## Language Boundary

First-party research-data collection and parsing on the data plane are native C++. Exact adapter dependencies, ABI, target, and provider compatibility require separate admission and tests.

## Non-Authorization Statement

This manifest authorizes no provider session, credential use, account access, subscription, data purchase, live order, or redistribution.
