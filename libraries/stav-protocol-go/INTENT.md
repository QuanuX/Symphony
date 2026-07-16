# STAV Protocol Kernel Intent

## Purpose

Provide one pure-Go implementation of the STAV v1 canonical JSON, typed message, identifier, digest, and bounded local-frame contracts owned by `knowledge/stav/`.

## Scope

- Strict I-JSON parsing and RFC 8785 canonicalization for the ratified STAV profile.
- Immutable-by-validation Go message types for candidate, event, receipt, query, query-page, and verification documents.
- UUID, TOPS ID, timestamp, registered-identifier, safe-reference, and digest validation.
- Domain-separated SHA-256 candidate, event, and genesis digests.
- Four-byte big-endian bounded frame encoding and decoding over caller-provided streams.

## Non-Scope

- Canonical ownership or schema generation.
- Networking, socket creation, peer authentication, authorization, supervision, persistence, ledger operations, recovery, repair, retention, rotation, or export.
- Configuration, status, or local request/response semantics.
- Issuing a committed receipt or assigning authoritative event fields.
