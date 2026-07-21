# Symphony API Contract Registry

## Status

Canonical SACV registry. No API entry document is registered at this time.

## Purpose

Map each canonical Symphony HTTP API contract to its semantic owner, location, compatibility identity, security profile, exposure class, and publication state without centralizing endpoint truth.

## Entry Model

Each entry MUST provide:

- `api_id`: stable lowercase identifier;
- `title`: human-readable API name;
- `owner`: canonical knowledge vector or module;
- `path`: repository-relative canonical OpenAPI entry document;
- `openapi`: `3.2.0`;
- `api_version`: owner-defined semantic contract version;
- `audience`: `local_internal`, `remote_administrative`, `partner`, or `public`;
- `transport_profile`: ratified HTTP transport identifier;
- `security_profile`: repository-relative canonical SSIAG or public-access profile path, indexed by SKVI;
- `publication_state`: `internal_only`, `candidate`, or `sodv_approved`;
- `sdk_state`: `not_eligible`, `candidate`, or `approved`;
- `status`: `draft`, `ratified`, `deprecated`, or `retired`;
- `notes`: safe human-readable context.

## Canonical Markdown Grammar

Each entry is one contiguous ordered block using the exact field order above. Every field is written as a Markdown list item in `- field: value` form. `owner` and `path` SHOULD be backtick-delimited for readability; parsers normalize only that outer delimiter. Duplicate, unknown, missing, empty, or reordered identity-start fields fail validation rather than being inferred.

`api_id` begins a new entry. API identifiers and paths MUST be unique. `owner` is exactly one `knowledge/<vector>` or `modules/<module>` root, and `path` MUST be a descendant of that owner ending in `.openapi.json` or `.openapi.yaml`. Every registered document MUST also be indexed by SKVI.

The literal `None.` beneath `## Canonical Entries` is the only valid empty-registry representation. It is removed atomically with the first ratified entry; it is never retained beside entry blocks.

## Canonical Entries

None.

The existing SSIAG metadata-only Unix-socket routes are not registered yet. Their caller-authentication and transport description must be updated atomically with any future canonical OpenAPI entry document.

## Prohibited Entries

Do not register:

- qxctl command grammar;
- SSIAG provider IPC or secret channels;
- STAV append-authority ingestion;
- NATS or trading-path protocols;
- generated bundles, SDKs, MDX, playground configuration, or MCP projections;
- an API without an identified semantic owner and ratified security profile.
