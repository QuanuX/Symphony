# Symphony TOPS Audit Vector Manifest

## Canonical Target

`knowledge/stav/`

## Owned Truth

STAV owns its event envelope, presence rules, integrity rules, append-authority contract, producer authority, query/proposal permissions, storage boundaries, redaction classifications, and projection doctrine.

## Authority Split

- `knowledge/stav/`: canonical protocol and schema truth;
- `libraries/stav-protocol-go/`: pure-Go, build-time implementation of serialization, validation, digest, identifier, and frame mechanics with no runtime authority;
- `modules/stav-append-authority/`: independently installable Go implementation of the dedicated per-TOPS append-authority role;
- SSIAG: first implemented producer class through an explicit per-installation grant;
- node-troll: future producer class requiring separate review and grant;
- qxctl: canonical administrative and query interface implementing the protocol;
- agents: query and proposal authority only.

qxctl does not own the schema. Producers do not choose sequence numbers or edit ledger files. Agents do not append directly.

## Operational Storage

- user: `${XDG_STATE_HOME}/symphony/<tops-id>/stav/`
- system: `/var/lib/symphony/<tops-id>/stav/`

Operational state is never stored under `knowledge/stav/`. DuckDB, HDF5, JSONL exports, graphs, and embeddings are derived, disposable projections and never canonical records.

## Status

Architect-ratified operational v1. Canonical schemas, protocol kernel, per-TOPS append authority, mutual Unix peer authentication, exact producer/reader grants, fsync-before-receipt ledger, restart recovery/idempotency, native per-TOPS supervision/runtime ownership, qxctl read interface, and SSIAG producer are implemented. Signed checkpoints, remote export, non-repudiation, automatic rotation, general repair, and node-troll producer authority remain deferred.
