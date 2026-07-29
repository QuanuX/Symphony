# Symphony Semantic Feature Vector Registry

## Status

Canonical SSFV feature-routing registry. No feature record or distributed `FEATURES.md` file is registered at this time.

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

None.

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

This empty registry is complete for the current contract state. It does not authorize feature bootstrap, distributed file creation, engine implementation, or canonical mutation.
