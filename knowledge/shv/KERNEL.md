# SHV initial kernel and generic graph integration

Status: implemented bounded kernel 0.2.0-dev contract; the separate generic adapter and source lifecycle engine remain 0.1.0-dev. qxctl retains explicit legacy kernel 0.1.0-dev selection. The broader Hardware Capability Atlas inventory remains a roadmap. This increment follows the September 13 SHV review and explicit user authorization for generic graph adapters.

## Ownership

`modules/shv-engine/SPEC.md` owns source-backed hardware semantics. `modules/shv-graph-adapter/SPEC.md` owns generic transport and structural query. `tools/qxctl/MANIFEST.md` owns exact receipt selection and independent consumer checks. No graph database vendor, hardware selection, deployment, active source head or topology is chosen for users.

The default core coverage is model introduction from 2018-01-01 through explicit as_of. Caller-authored profiles can include older hardware, narrow manufacturers/classes/IDs, compose filters or select all. Missing dates and intervals crossing a boundary stay unresolved. Individual manufacture dates are not synthesized from model introduction. Coverage describes the supplied inventory, not completeness of all hardware ever released.

## Agentic interface

Every invocation selects `--prefix` and `--version`, or `--adapter-prefix` and `--adapter-version` for adapter leaves. All delivered native operations are bounded, read-only process calls. Input payloads use `--input`; `--json` exposes machine output.

- `qxctl shv inspect`
- `qxctl shv coverage default|plan`
- `qxctl shv catalogue build|query`
- `qxctl shv evaluate`
- `qxctl shv graph project|validate`
- `qxctl shv graph adapter inspect|roundtrip|query`
- `qxctl shv schema` and `qxctl shv template`

Discovery reads receipt-owned versioned schemas/templates, distinguishes CLI-owned discovery envelopes, and returns unanswered templates without fabricating source paths, dates, hardware or requirements. CLI consumer checks verify result correspondence rather than accepting a self-seal alone.

## Rebuildable evidence

Catalogue build consumes exact manifest sizes/digests through bounded no-follow file reads. The caller supplies literal mappings and exact distinct product-heading/specification `tag#id` selectors. This finite HTML profile consumes paired DT/DD fields and ignores script/style/comments and unrelated navigation. The legacy profile supports typed integer/string/date/finite tokens. The explicit scoped_tables.v1 profile adds per-field table sections, quarter_20yy dates with inclusive quarter intervals, and table_rows values that retain complete operating-profile rows, header order and blank cells. Missing units are not inferred. Scalar evaluation cannot flatten structured tables; query and graph transport retain them unchanged. Opaque files can be retained without deriving claims. Query, evaluation and graph operations rebuild from the retained files and mappings; resealing a changed assertion is insufficient.

The reviewed AMD EPYC 7313/7313P specimen distinguishes the documented socket-mode sets. A dual-socket processor finding cannot establish motherboard, BIOS, operating-system or active-node compatibility. Those require their own evidence; the kernel reports unresolved when no matching predicate/qualifier exists. Caller mappings are declarations, not authenticated publisher interpretations.

## Generic adapter extension

The public C++ `symphony/graph/adapter.hpp` defines `Adapter` and `PortableReference`, plus shared exchange validation. Implementations accept the exact owner-bound graph and return sealed structural results. The reference adapter preserves source/subject nodes, attributable assertion edges and arbitrary canonical properties, and queries exact IDs with missing IDs reported separately. It is independently installable and in-memory. The installed CMake target `Symphony::ShvGraphAdapter` supplies the linkable static implementation and exact foundation dependency; `modules/shv-graph-adapter/INSTALL.md` describes standalone C++ consumer setup.

A future driver must declare its storage behavior, exact installation/descriptor/schema, operation ownership, limits, conformance and recovery semantics. It must preserve complete owner artifacts and graph IDs/properties and leave source semantic acceptance with SHV. Generic exchange does not certify a vendor backend, persistent transaction, query planner or database migration. No dynamic driver loader is shipped in this increment.

## Remaining atlas work

Partitioning and resumable corpus lifecycle; mutable approved source-location registries and refresh history; broader manufacturer/form-factor sources; generic interpretation profiles beyond this finite HTML grammar; conflict/freshness/evidence policies; hardware relationship inference with explicit scope; measurements and active observations through their proper owners; vendor database drivers and durable indexes; and Composer consumption remain future increments. Existing SCV contracts provide reuse candidates without transferring ownership implicitly.

The finite section selector is lowercase HTML tag followed by `#` and a literal nonempty ID, with exactly one `#`; this is not a CSS interpreter. Uniqueness binds the complete tag and ID pair. A different tag with the same ID does not match.

## Component source lifecycle

The independent `modules/shv-source-engine/` implements `SOURCES.md`: stable source identity, explicit relocation/authority-change plans, pure transition and finite history validation, actual-byte capture binding/comparison, and source-replayed provenance graphs through qxctl. These read-only operations do not activate a source registry or authenticate publisher authority. Core curation prioritizes original component/product identity and evidenced variants; user-selected source coverage remains extensible. The retained Intel Xeon Gold 5418Y specification is mapped with quarter precision, component fields and complete SST performance-profile rows. Whole-system catalogue permutations, further component mappings, protected activation and durable vendor storage remain separate increments.
