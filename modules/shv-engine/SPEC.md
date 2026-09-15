# SHV kernel specification 0.3.0-dev

## Exact operations

Eight operations use `symphony.knowledge.engine-process.v1`: inspect, coverage_default, coverage_plan, catalogue_build, catalogue_query, evaluate, graph_project and graph_validate. The descriptor.v2 lists implemented operation IDs, feature ownership, exact input/output protocols and limits. All are explicit bounded read-only operations, with no mutation, network listener, active alias or source discovery. Canonical JSON self-seals identify artifacts; they are not publisher authentication.

## Coverage

The default is a sealed caller-visible profile with model-introduction interval 2018-01-01 through explicit as_of. No year is hardcoded in admission: callers supply a differently sealed profile with all, date, ids, class, manufacturer, and, or and not expressions. This release supports model_introduction only; unit manufacture/revision-date filtering requires future explicit semantics and fails instead of substituting launch time. A date fact is a valid inclusive calendar interval or null. Containment includes, disjointness excludes, missing/straddling stays unresolved. AND exclusion dominates, OR inclusion dominates, NOT preserves unresolved. Profiles impose no deletion or hardware-use policy. Counts describe only the caller's supplied finite inventory, not all hardware since 2018.

## Sources and catalogue

Build accepts explicit absolute source_root and root-relative source manifests. Up to eight files, one MiB per file and four MiB total; sizes/digests must match actual bytes read through no-follow helpers. Root is omitted from portable artifacts. Source formats html and opaque; opaque can be retained but cannot yield mapped assertions. Unsupported inputs are not interpreted by AI or executed.

The legacy untagged HTML mapping is deliberately finite: caller-selected distinct heading_section and field_section exact `tag#id` selectors, each unique and closed; one exact product H1 equal to the selected model within heading_section, paired DT/DD fields within field_section, bounded field text, no duplicate terms or unpaired definitions. Tag names match exactly, so custom elements cannot impersonate H1/DT/DD. Script/style closing names require a token boundary. Script/style blocks, comments and out-of-section fields are ignored. Named entities amp/lt/gt/quot/apos/nbsp/reg/trade/copy/ndash/mdash/times/micro and valid Unicode numeric scalar entities are decoded; unsupported entities in captured fields reject. ASCII whitespace is normalized. Mapping label and next_label must be adjacent DT terms. No browser repair, JavaScript or general PDF parser is present. The input is UTF-8 and malformed output cannot be serialized as valid process JSON.

Mappings explicitly bind predicate, adjacent labels, type and qualifier. Types are string, interoperable integer, reviewed MM/DD/YYYY date, or slash-separated finite tokens. Token arrays are sorted and duplicate-free. In the legacy mapping, only a date-typed model_introduction predicate supplies the introduction interval. Metadata is caller-declared, anchored to the exact document heading; manufacturer/class are not inferred from a brand model name. At most32 subjects,16 assertions each,256 total. Identifiers use [A-Za-z0-9._-]{1,128}; qualifiers are explicit exact strings.

The catalogue retains manifests and complete mappings, sorted subjects/assertions and exact source IDs. Consumers replay the whole build from the explicit source_root before query, evaluation, projection or graph validation. Missing/changed files, altered source-derived assertions or an altered graph fail even if resealed. Replay verifies correspondence to selected bytes/mappings; it does not establish publisher truth or the appropriateness of a caller's mapping.

## Explicit scoped tables in 0.3.0-dev

New subject mapping exact fields: `id`, `manufacturer`, `model`, `hardware_class`, `source_id`, `heading_section`, `interpretation_profile`, `fields`. `interpretation_profile` is exactly `scoped_tables.v1`. The identity and heading semantics and 16-field/32-subject/256-assertion bounds remain as before. Each field selects its own distinct-from-heading exact `tag#id` section containing exactly one closed, nonnested table. H1 is uniquely selected within the heading section and equals the model. Text normalization/entities and ignored script/style/comment semantics match the existing finite parser. No browser repair, execution or remote acquisition.

Scalar field exact fields: `predicate`, `section`, `label`, `next_label`, `value_type`, `qualifier`. Each table row must be exactly TH then TD; duplicate labels reject. A non-null next_label must be the adjacent row label; an explicit null requires the selected row to be the last row. Omission is invalid. Types: `string`, `integer`, `date`, `tokens` (existing semantics), or `quarter_20yy`. The latter accepts exactly `Q[1-4]'[0-9][0-9]`, explicitly meaning 2000–2099, and emits `{precision:"quarter",source_text:"Q1'23",from:"2023-01-01",through:"2023-03-31"}`. It never guesses another century. The model_introduction predicate requires date or quarter_20yy; introduced retains only the inclusive from/through interval, while the assertion retains precision and original normalized text.

Matrix field exact fields: `predicate`, `section`, `columns`, `value_type`, `qualifier`. value_type is `table_rows`. Columns is 1–16 distinct nonempty strings of at most 256 UTF-8 bytes. The first row is exactly those TH headers, subsequent rows are TD cells of the same width. There must be 1–32 data rows. The first column is the declared row identity and must be nonempty and unique; other cells may be empty. Output value is `{columns:[...],rows:[[string,...],...]}` preserving full row and column order. No numeric/unit conversion, mode mixing, independent maxima or unit inference. The source headings may omit units: those remain unresolved. Table cells are at most 4096 normalized UTF-8 bytes, raw selected field buffers at most 65536; at most 256 rows per parsed table. Nested tables, nested cells/rows, unpaired structure and any rowspan/colspan attribute on selected cells reject. Optional thead/tbody/tfoot groups must be nonnested and explicitly closed. Outside cells within a selected table, only ASCII whitespace, ignored script/style/comments and the declared table structure are accepted; misplaced text or other elements reject. Closing tags within tables may contain only the name and whitespace. Every raw-text or tag-separator buffer append enforces the 65536-byte bound.

Catalogue outer protocol remains symphony.shv.catalogue.v1 with explicit tagged mappings. Query, graph projection, and graph validation retain complete structured values and replay original bytes and mapping. The existing scalar requirement language cannot compare structured quarter/table values: matching such a value with a scalar requirement rejects as an unsupported type, not hardware contradiction. Coverage already consumes inclusive intervals and preserves partial overlap as unresolved.

qxctl independently parses retained tables and reconstructs assertions/intervals. The invocation boundary binds its validator to the exact selected version; `0.1.0-dev` rejects tagged table mappings even if a false old engine emits them. Graph owner version equals the exact selected kernel, and graph_validate requires that version. The adapter remains `0.1.0-dev` and carries the new structured graph unchanged using its existing arbitrary JSON contract. Source lifecycle engine remains `0.1.0-dev` and independently selected.


## Query and evaluation

Empty subject_ids selects the complete bounded catalogue; an explicit list preserves caller order and returns missing IDs separately. Requirements preserve caller order and exact predicate/operator/value/qualifier. Supported operators: eq (matching value types), gte (interoperable integers only), contains (string membership in a finite_supported_set token array). Unsupported/mismatched value semantics reject as input errors, never as hardware incompatibility.

Exact predicate and qualifier match supplies evidence: comparison true is supported, false contradicted, absent match unresolved. A finite_supported_set is an explicitly selected mapping meaning; absence in arbitrary prose cannot support contradiction. Only one assertion per subject/predicate is supported: duplicate/conflicting mapping entries reject visibly. Multi-source conflict aggregation, arbitrary relationship inference, independent evidence policies and freshness evaluation remain future contracts. A processor-mode match is not whole-system or active-state proof. Changing a requirement creates a different request; proceeding with a mismatch does not rewrite its finding.

## Generic graph boundary

Graph exchange protocol symphony.graph.exchange.v1 preserves exact owner engine/version/catalogue digest and full owner_artifact. Subject/source nodes retain full values; each assertion becomes an attributable edge. Identity encodings and sorted rows are deterministic. SHV graph_validate replays sources and reconstructs every node/edge before equality; a generic storage adapter validates only its declared structure and transport. The generic portable reference adapter is a separate installation; no vendor graph database or persistent backend is selected by this kernel.

## Bounds and nonclaims

The shared foundation keeps one MiB requests, four MiB responses, 65,536-byte JSON strings, depth64 and parser-value bounds. Source bytes travel by verified references. Selectors cap depth16/nodes128; input inventory128; requirements32; fields256 bytes for labels and4,096 for values. Bounded exhaustion rejects; no clipping or partial success. This is a useful specimen kernel, not the broad core atlas, partition protocol, firmware/Node observation owner, performance guarantee or provisioning action. qxctl must expose every delivered operation and exact installed schema/template.

The finite section selector is lowercase HTML tag followed by `#` and a literal nonempty ID, with exactly one `#`; this is not a CSS interpreter. Uniqueness binds the complete tag and ID pair. A different tag with the same ID does not match.

The consolidated SHV gate adds provenance-bearing PDF ingestion (kernel 0.3.0-dev) and explicit kernel dependency admission (partition 0.2.0-dev), described in knowledge/shv/DOCUMENT-INGESTION.md. Earlier installations remain explicitly selectable.

## Mechanical interface release 0.4.0-dev

OWNER-INTERFACE.json declares exact metadata. INTERFACE-GENERATOR.json selects generation paths and frozen history; the shared registration-driven generator produces src/interface.generated.hpp, Go admission and CMake inventory. qxctl verifies the installed declaration through receipt-v2 and compiled admission. Existing command identities and defaults remain unchanged. Old packages remain independently selectable; artifacts and retained writers keep their exact original identity. Metadata generation is separate from native semantic handlers and independent Go result replay.

This inventory is not exhaustive. New owners can use `tools/shv-interface-codegen/EXTENDING.md` without extending a fixed generator whitelist. They must supply their own semantic contracts and qxctl integration.

The kernel retains its exact embedded PDF 0.2 reader. PDF 0.3 is independently selectable for PDF operations; it is not substituted into kernel provenance.
