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
- `knowledge/shv/PROFILES.md`
- `modules/shv-engine/SPEC.md`
- `modules/shv-graph-adapter/SPEC.md`

## Implemented Initial Kernel

`modules/shv-engine/` supplies the independently installable C++ kernel (current 0.3.0-dev, with earlier exact releases retained): explicit coverage profiles, retained-source catalogue build/replay, exact requirement evaluation, deterministic graph projection and source-backed graph validation. `modules/shv-graph-adapter/` supplies a separately installable generic C++ graph port and working in-memory portable reference adapter for lossless exchange and structural queries. `qxctl shv` administers every delivered native operation and discovers exact installed schemas and unanswered templates.

These bounded v1 contracts are implemented; a broad hardware atlas, automated source acquisition, multi-source conflict aggregation, additional vendor-specific database drivers, custom plugin loading and Composer integration remain later work. Durable structural storage is implemented under `GRAPH-STORE.md`. No dedicated graph database is selected. The existing C++ DuckDB SQL default is unchanged. Read `KERNEL.md` and both module SPEC files for exact ownership and limits.

## Reproducibility

An installation must be able to generate its own graph through the same defined process and data points used to produce the reference graph. A prebuilt opaque database alone cannot satisfy the vector contract.

## Non-Authorization Statement

This manifest authorizes no hardware probe, benchmark, purchase, provider action, graph technology, network API, AI operation, or performance claim.

## Component source lifecycle

The independent `modules/shv-source-engine/` implements `SOURCES.md`: stable source identity, explicit relocation/authority-change plans, pure transition and finite history validation, actual-byte capture binding/comparison, and source-replayed provenance graphs through qxctl. These read-only operations do not activate a source registry or authenticate publisher authority. Core curation prioritizes original component/product identity and evidenced variants; user-selected source coverage remains extensible. Protected source activation is implemented separately under `ACTIVATION.md`; the pure source engine does not itself grant authority. Whole-system catalogue permutations and expanded component mappings remain caller-selected scope.

## Caller-owned coverage accounting

`INVENTORY.md` defines qxctl inventory snapshots and replayed comparison. Native owners retain source and coverage semantics.

## Explicit component dossiers

`DOSSIERS.md` defines caller-owned associations, exact citations and generic graph exchange through qxctl, with no inferred physical identity.

## Documented identifier profile

`IDENTIFIERS.md` and `AMD-PRODUCT-IDENTIFIERS.v1.md` define a versioned, source-specific mapping composed through existing C++ catalogue/evaluation operations. No new native operation or automatic loader is added.

## PDF evidence boundary

`PDF-EVIDENCE.md` distinguishes implemented opaque capture/replay from prospective document decoding and native PDF hardware interpretation.

`PDF-ADAPTER.md` now defines the independently installable C++ adapter and qxctl extraction/replay of one explicit AMD PDF table. General PDF coverage and namespace equivalence remain separate. Versioned documentary graph projection and kernel ingestion are implemented as described below.

PDF adapter 0.2.0 now projects qualified documentary assertion edges and revalidates them from original bytes after generic graph exchange. These remain PDF-owned artifacts; kernel 0.3.0-dev admits them through the explicit `DOCUMENT-INGESTION.md` contract.

`DOCUMENT-INGESTION.md` now defines implemented kernel ingestion, caller-owned class field evolution and command-owner discipline. Durable storage and protected local catalogue publication are implemented under their separate contracts below.

`GRAPH-STORE.md` defines the independent durable DuckDB adapter and qxctl control family. It stores generic graph evidence; protected local catalogue head publication is implemented under `PUBLICATION.md`; `CATALOGUE-PUBLICATION-PLAN.md` preserves the earlier plan.

`PUBLICATION.md` defines protected caller catalogue heads, distinct SSIAG publication authority, original-source and durable-snapshot replay, and recovery through the single qxctl publication family.

## Caller profiles and portable universes

`PROFILES.md` defines the independent `shv-profile-engine`, declaration conformance,
portable recipes and exact local source binding. The existing coverage and catalogue
owners retain their controls; profiles do not impose a universal hardware census.
