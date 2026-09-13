# SHV kernel specification 0.1.0-dev

## Exact operations

Eight operations use `symphony.knowledge.engine-process.v1`: inspect, coverage_default, coverage_plan, catalogue_build, catalogue_query, evaluate, graph_project and graph_validate. The descriptor.v2 lists implemented operation IDs, feature ownership, exact input/output protocols and limits. All are explicit bounded read-only operations, with no mutation, network listener, active alias or source discovery. Canonical JSON self-seals identify artifacts; they are not publisher authentication.

## Coverage

The default is a sealed caller-visible profile with model-introduction interval 2018-01-01 through explicit as_of. No year is hardcoded in admission: callers supply a differently sealed profile with all, date, ids, class, manufacturer, and, or and not expressions. This release supports model_introduction only; unit manufacture/revision-date filtering requires future explicit semantics and fails instead of substituting launch time. A date fact is a valid inclusive calendar interval or null. Containment includes, disjointness excludes, missing/straddling stays unresolved. AND exclusion dominates, OR inclusion dominates, NOT preserves unresolved. Profiles impose no deletion or hardware-use policy. Counts describe only the caller's supplied finite inventory, not all hardware since 2018.

## Sources and catalogue

Build accepts explicit absolute source_root and root-relative source manifests. Up to eight files, one MiB per file and four MiB total; sizes/digests must match actual bytes read through no-follow helpers. Root is omitted from portable artifacts. Source formats html and opaque; opaque can be retained but cannot yield mapped assertions. Unsupported inputs are not interpreted by AI or executed.

The HTML mapping is deliberately finite: caller-selected distinct heading_section and field_section exact `tag#id` selectors, each unique and closed; one exact product H1 equal to the selected model within heading_section, paired DT/DD fields within field_section, bounded field text, no duplicate terms or unpaired definitions. Tag names match exactly, so custom elements cannot impersonate H1/DT/DD. Script/style closing names require a token boundary. Script/style blocks, comments and out-of-section fields are ignored. Named entities amp/lt/gt/quot/apos/nbsp/reg/trade/copy/ndash/mdash/times/micro and valid Unicode numeric scalar entities are decoded; unsupported entities in captured fields reject. ASCII whitespace is normalized. Mapping label and next_label must be adjacent DT terms. No browser repair, JavaScript or general PDF parser is present. The input is UTF-8 and malformed output cannot be serialized as valid process JSON.

Mappings explicitly bind predicate, adjacent labels, type and qualifier. Types are string, interoperable integer, reviewed MM/DD/YYYY date, or slash-separated finite tokens. Token arrays are sorted and duplicate-free. Only a date-typed model_introduction predicate supplies the introduction interval. Metadata is caller-declared, anchored to the exact document heading; manufacturer/class are not inferred from a brand model name. At most32 subjects,16 assertions each,256 total. Identifiers use [A-Za-z0-9._-]{1,128}; qualifiers are explicit exact strings.

The catalogue retains manifests and complete mappings, sorted subjects/assertions and exact source IDs. Consumers replay the whole build from the explicit source_root before query, evaluation, projection or graph validation. Missing/changed files, altered source-derived assertions or an altered graph fail even if resealed. Replay verifies correspondence to selected bytes/mappings; it does not establish publisher truth or the appropriateness of a caller's mapping.

## Query and evaluation

Empty subject_ids selects the complete bounded catalogue; an explicit list preserves caller order and returns missing IDs separately. Requirements preserve caller order and exact predicate/operator/value/qualifier. Supported operators: eq (matching value types), gte (interoperable integers only), contains (string membership in a finite_supported_set token array). Unsupported/mismatched value semantics reject as input errors, never as hardware incompatibility.

Exact predicate and qualifier match supplies evidence: comparison true is supported, false contradicted, absent match unresolved. A finite_supported_set is an explicitly selected mapping meaning; absence in arbitrary prose cannot support contradiction. Only one assertion per subject/predicate is supported: duplicate/conflicting mapping entries reject visibly. Multi-source conflict aggregation, arbitrary relationship inference, independent evidence policies and freshness evaluation remain future contracts. A processor-mode match is not whole-system or active-state proof. Changing a requirement creates a different request; proceeding with a mismatch does not rewrite its finding.

## Generic graph boundary

Graph exchange protocol symphony.graph.exchange.v1 preserves exact owner engine/version/catalogue digest and full owner_artifact. Subject/source nodes retain full values; each assertion becomes an attributable edge. Identity encodings and sorted rows are deterministic. SHV graph_validate replays sources and reconstructs every node/edge before equality; a generic storage adapter validates only its declared structure and transport. The generic portable reference adapter is a separate installation; no vendor graph database or persistent backend is selected by this kernel.

## Bounds and nonclaims

The shared foundation keeps one MiB requests, four MiB responses, 65,536-byte JSON strings, depth64 and parser-value bounds. Source bytes travel by verified references. Selectors cap depth16/nodes128; input inventory128; requirements32; fields256 bytes for labels and4,096 for values. Bounded exhaustion rejects; no clipping or partial success. This is a useful specimen kernel, not the broad core atlas, partition protocol, firmware/Node observation owner, performance guarantee or provisioning action. qxctl must expose every delivered operation and exact installed schema/template.

The finite section selector is lowercase HTML tag followed by `#` and a literal nonempty ID, with exactly one `#`; this is not a CSS interpreter. Uniqueness binds the complete tag and ID pair. A different tag with the same ID does not match.
