# STAV Append Authority Manifest

## Identity

- module: `modules/stav-append-authority/`
- Go module: `github.com/QuanuX/Symphony/modules/stav-append-authority`
- executable: `symphony-stav-append-authority`
- implementation: Go, `CGO_ENABLED=0`

## Canonical Dependencies

- `knowledge/stav/`: protocol, schemas, storage, authorization, projection, and recovery truth;
- `libraries/stav-protocol-go/`: strict canonical codecs, digests, identifiers, and framing;
- `knowledge/tops/`: immutable TOPS identity;
- `tools/qxctl/`: read-only administrative client;
- SSIAG: first governed producer through its closed STAV vocabulary.

## Implemented Capabilities

- atomic user/system executable install and conservative uninstall;
- per-TOPS enroll, preserve-by-default unenroll, and explicit one-TOPS purge;
- strict configuration/status/local-envelope contracts;
- mutually authenticated local IPC using Darwin/Linux kernel peer credentials;
- authority endpoint identity verification in both server and client;
- exact producer tuple and reader-classification grants;
- exclusive locked append-only ledger with frame checksums and digest-chain verification;
- fsync-before-receipt, restart idempotency reconstruction, and bounded query projection;
- incomplete-final-frame evidence preservation and fail-closed complete corruption;
- `qxctl stav status|verify|query|doctor` integration;
- typed SSIAG producer integration.

## Deliberately Absent

- qxctl or agent append authority;
- HTTP, OpenAPI, NATS, TCP, or remote STAV transport;
- automatic retention deletion, rotation, middle-frame salvage, or general repair;
- signed checkpoints, non-repudiation, and remote export;
- secret values, assertions, tokens, provider payloads, or routine telemetry;
- service-manager definitions or inherited supervisor authority.

Host uninstall always preserves per-TOPS configuration and ledgers. Purge requires the selected TOPS ID and refuses an active listener.
