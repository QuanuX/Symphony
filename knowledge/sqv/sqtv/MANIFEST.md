# Symphony Quantitative Transformation Vector Manifest

## Canonical Target

`knowledge/sqv/sqtv/`

## Declared Contract Truth Role

SQTV owns conversion and research-transformation semantics beneath SQV, including declared loss and derived lineage.

## Canonical Surfaces

- `knowledge/sqv/sqtv/INTENT.md`
- `knowledge/sqv/sqtv/MANIFEST.md`
- `knowledge/sqv/sqtv/SPEC.md`
- `knowledge/sqv/sqtv/SKILL.md`

## Implementation Status

The first implemented scope is `modules/sqtv-integer-conversion-cpp/` `0.1.0-dev`: an independently installed C++26 static library for exact dense integer width, signedness and byte-order conversion. Its SPEC admits exact schemas, operation identity, finite limits and lineage; richer transformations remain open. No `sqtv:` colon identity family or SQV qxctl surface is allocated.

## Language Boundary

First-party research-data conversion and transformation on the data plane are native C++. The first module declares its exact C++26 API, operation and dependencies. Additional operations and platforms require independent admission and evidence.

## Non-Authorization Statement

This manifest authorizes no data access, provider collection, information-reducing conversion, strategy calculation, or publication.
