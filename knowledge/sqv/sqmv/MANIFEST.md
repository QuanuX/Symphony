# Symphony Quantitative Metadata Vector Manifest

## Canonical Target

`knowledge/sqv/sqmv/`

## Declared Contract Truth Role

SQMV owns the meaning of research dataset descriptions beneath SQV. Source observations, access grants, movement, and retention remain with their respective owners.

## Canonical Surfaces

- `knowledge/sqv/sqmv/INTENT.md`
- `knowledge/sqv/sqmv/MANIFEST.md`
- `knowledge/sqv/sqmv/SPEC.md`
- `knowledge/sqv/sqmv/SKILL.md`

## Implementation Status

The narrow `modules/sqmv-metadata-cpp/` `0.1.0-dev` C++26 library implements immutable metadata reference resolution and exact SQFV binding verification under its own versioned module contract. It records attributable evidence references without a resident catalogue, provider connection, access authority or SQV qxctl operation. No `sqmv:` identity family is allocated by this Quad; the module's content-reference grammar belongs to its exact SPEC.

## Language Boundary

First-party research-data implementation on the data plane uses C++26. The first library pins its package dependencies and compatible toolchain; callers may retain a resolved manifest and binding without repeated lookup on the data path.

## Non-Authorization Statement

This manifest authorizes no source collection, credential use, private-data disclosure, dataset mutation, storage action, or strategy execution.
