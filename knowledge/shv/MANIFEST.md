# Symphony Hardware Vector Manifest

## Canonical Target

`knowledge/shv/`

## Declared Contract Truth Role

SHV is the owner of the Hardware Capability Atlas and hardware matrix concept. Those are one semantic vector, though future implementation may contain multiple cooperating components.

## Canonical Surfaces

- `knowledge/shv/INTENT.md`
- `knowledge/shv/MANIFEST.md`
- `knowledge/shv/SPEC.md`
- `knowledge/shv/SKILL.md`
- `knowledge/shv/KERNEL.md`
- `modules/shv-engine/SPEC.md`
- `modules/shv-graph-adapter/SPEC.md`

## Implemented Initial Kernel

`modules/shv-engine/` supplies the independently installable C++ 0.1.0-dev kernel: explicit coverage profiles, retained-source catalogue build/replay, exact requirement evaluation, deterministic graph projection and source-backed graph validation. `modules/shv-graph-adapter/` supplies a separately installable generic C++ graph port and working in-memory portable reference adapter for lossless exchange and structural queries. `qxctl shv` administers every delivered native operation and discovers exact installed schemas and unanswered templates.

These bounded v1 contracts are implemented; a broad hardware atlas, automated source acquisition, multi-source conflict aggregation, persistence, vendor database drivers, custom plugin loading and Composer integration remain later work. No dedicated graph database is selected. The existing C++ DuckDB SQL default is unchanged. Read `KERNEL.md` and both module SPEC files for exact ownership and limits.

## Reproducibility

An installation must be able to generate its own graph through the same defined process and data points used to produce the reference graph. A prebuilt opaque database alone cannot satisfy the vector contract.

## Non-Authorization Statement

This manifest authorizes no hardware probe, benchmark, purchase, provider action, graph technology, network API, AI operation, or performance claim.

## Component source lifecycle

The independent `modules/shv-source-engine/` implements `SOURCES.md`: stable source identity, explicit relocation/authority-change plans, pure transition and finite history validation, actual-byte capture binding/comparison, and source-replayed provenance graphs through qxctl. These read-only operations do not activate a source registry or authenticate publisher authority. Core curation prioritizes original component/product identity and evidenced variants; user-selected source coverage remains extensible. Whole-system catalogue permutations, new component mappings, protected activation and durable vendor storage remain separate increments.

## Caller-owned coverage accounting

`INVENTORY.md` defines qxctl inventory snapshots and replayed comparison. Native owners retain source and coverage semantics.

## Explicit component dossiers

`DOSSIERS.md` defines caller-owned associations, exact citations and generic graph exchange through qxctl, with no inferred physical identity.
