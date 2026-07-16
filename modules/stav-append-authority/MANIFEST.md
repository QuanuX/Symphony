# STAV Append Authority Manifest

## Identity

- module: `modules/stav-append-authority/`
- Go module: `github.com/QuanuX/Symphony/modules/stav-append-authority`
- executable: `symphony-stav-append-authority`
- implementation language: Go
- cgo: prohibited

## Canonical Dependencies

- `knowledge/stav/` for protocol, envelope, authority, redaction, storage, query, and implementation gates;
- `knowledge/tops/` for immutable TOPS identity;
- `knowledge/skvi/` for discoverability;
- `libraries/stav-protocol-go/` for the first-party pure-Go implementation of ratified protocol mechanics;
- `tools/qxctl/` for the ratified read-only administrative/query grammar.

## Implemented Capabilities

- independently build the executable;
- install or uninstall only that executable at user or system scope;
- validate canonical lowercase RFC 9562 TOPS UUIDs through the shared protocol kernel;
- resolve the approved user/system configuration, state, runtime, and socket paths without creating them.

## Deliberately Absent

- configuration, status, and local request/response schemas or files;
- candidate ingestion or event/receipt emission despite the canonical schemas and kernel types now existing;
- producer enrollment and authorization;
- socket creation, listening, IPC, HTTP, OpenAPI, NATS, or remote access;
- ledger creation, append, verification, query, recovery, rotation, repair, or export;
- supervisor definitions and service-manager labels;
- agent mutation paths.

## Status

Namespace and reversible lifecycle scaffold only. Operational enablement remains fail-closed.
