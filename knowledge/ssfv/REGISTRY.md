# Symphony Semantic Feature Vector Registry

## Status

Canonical SSFV feature-routing registry. The first partial bootstrap covers the repository-root platform capability, the shared knowledge-vector engine foundation, and the knowledge-session coordinator foundation.

## Purpose

Map each stable SSFV feature identity to one canonical distributed owner record without centralizing or duplicating feature semantics.

## Entry Model

Each entry MUST provide, in this exact order:

- `feature_id`: stable SSFV identity;
- `feature_file`: repository-relative canonical `FEATURES.md`;
- `owner_contract`: repository-relative owning Contract Quad file;
- `source_scope`: explicit repository-relative source scope;
- `status`: `experimental`, `implemented`, `deprecated`, or `retired`;
- `parent_feature_id`: stable SSFV identity or `none`;
- `record_digest`: lowercase `sha256:` digest of the normalized feature record;
- `notes`: safe routing context.

## Canonical Markdown Grammar

Each entry is one contiguous ordered block using Markdown list items in `- field: value` form. Outer backticks are presentation delimiters only. Duplicate, unknown, missing, empty, or reordered fields fail validation.

`feature_id` MUST be globally unique. One source scope maps to exactly one `feature_file` plus `owner_contract` routing tuple, and several feature IDs MAY share that tuple. A feature identity appears in exactly one owner file. Paths MUST be normalized, repository-relative, and no-follow. The exact literal `.` represents repository-root source scope and owns root `FEATURES.md`; every other source scope owns `<source_scope>/FEATURES.md`. Every registered file MUST also be indexed by SKVI.

The literal `None.` beneath `## Canonical Entries` is the only valid empty-registry representation. It is removed atomically with the first ratified entry and MUST NOT coexist with entry blocks.

## Canonical Entries

- feature_id: `ssfv:symphony:knowledge-session-coordinator`
- feature_file: `modules/knowledge-session-coordinator/FEATURES.md`
- owner_contract: `modules/knowledge-session-coordinator/SPEC.md`
- source_scope: `modules/knowledge-session-coordinator`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:9b49266cf09d3f223f681a94323fdf185de79b32a0e3b5d3ba0770dffa480c5f`
- notes: First partial bootstrap record for durable reconciliation, SSIAG-authorized authenticated sessions, explicit idempotent qxctl host-event convergence, protected desired-profile and observation administration, and report-only dependency-driven lifecycle planning; lifecycle action persistence and apply remain disabled.

- feature_id: `ssfv:symphony:knowledge-vector-engine-foundation`
- feature_file: `libraries/knowledge-vector-engine-cpp/FEATURES.md`
- owner_contract: `libraries/knowledge-vector-engine-cpp/SPEC.md`
- source_scope: `libraries/knowledge-vector-engine-cpp`
- status: `experimental`
- parent_feature_id: `ssfv:symphony:platform`
- record_digest: `sha256:c31e519f59335ed7e145874795409bbe36a555006f7bf1e18c93afea9952b4ce`
- notes: First partial bootstrap record for the implemented authority-free shared C++ mechanics, including canonical STSC Gregorian and UTC representation validation.

- feature_id: `ssfv:symphony:platform`
- feature_file: `FEATURES.md`
- owner_contract: `INTENT.md`
- source_scope: `.`
- status: `experimental`
- parent_feature_id: `none`
- record_digest: `sha256:700f02746603ccebdb94711291414c5f840b5c28039dc525dc9b0837f041c0fd`
- notes: Repository-root capability record; bootstrap coverage is explicitly partial and does not imply production readiness or complete catalog coverage.

## Prohibited Entries

Do not register:

- proposals or planned behavior as present application truth;
- generated inventories, graph projections, summaries, documentation, or marketing pages;
- a directory merely because it exists;
- a language, source file, or symbol without the full feature-worthiness gate;
- an owner record using implicit globs or traversal;
- proposal-only modules such as node-troll, bus-troll, or hotpath-runtime before implementation exists;
- a record not covered by SKVI and an owner contract.

## Non-Authorization Statement

This three-record registry is an explicitly partial catalog. It does not authorize another feature record, another distributed file, repository-wide completeness, engine-decided feature-worthiness, or canonical mutation.
