# Symphony Semantic Feature File Format

## Purpose

Define the deterministic Markdown container used by each registered distributed `FEATURES.md` owner file.

## Ownership

A feature file belongs to one exact source scope and one owner contract. Several feature records may share the same file. A feature identity appears in exactly one canonical owner file.

The repository-root source scope is the exact literal `.` and owns `FEATURES.md`. Any other source scope is a normalized repository-relative directory and owns `<source_scope>/FEATURES.md`.

## Exact Managed Region

Every registered feature file contains exactly one managed region:

````markdown
# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/example/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [],
  "source_scope": "modules/example"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
````

The example is protocol illustration only. A canonical registered feature file MUST contain at least one ratified record; this document does not authorize an empty owner file or a feature bootstrap.

The opening marker MUST be followed by one newline, the exact fence ` ```json`, one newline, one JSON object, the exact closing fence, one newline, and the closing marker. Missing, duplicate, nested, reordered, or ambiguous markers fail validation.

## Canonical Content

The embedded JSON object is the canonical semantic record container. It conforms to `knowledge/ssfv/schemas/v1/feature-file.schema.json` and contains records conforming to the SSFV feature-record v2 schema.

Text outside the managed region is owner-controlled explanatory Markdown. An engine MUST preserve that text byte-for-byte when rendering a proposal. Outside text MUST NOT duplicate or override the embedded semantic records.

## Deterministic Rendering

An SSFV engine renders:

- UTF-8 without a byte-order mark;
- lexicographically ordered JSON object keys;
- two-space indentation;
- one trailing newline inside the JSON fence;
- records ordered lexicographically by stable feature ID;
- deterministic ordering for set-like record collections;
- the exact managed-region markers and fences in this document.

The registry record digest binds the compact normalized record object, not the surrounding Markdown or feature-file envelope.

## Routing Cardinality

- `feature_id` is globally unique.
- One source scope maps to exactly one `feature_file` and `owner_contract` tuple.
- Many feature IDs may share that routing tuple.
- One feature ID has exactly one canonical owner file.
- A move preserves the stable feature ID and updates old routing, new routing, the registry, and SKVI in one reviewed change.

## Non-Authorization Statement

This format defines a canonical container. It does not create a `FEATURES.md`, declare a feature implemented, authorize engine mutation, or make generated explanatory text canonical.
