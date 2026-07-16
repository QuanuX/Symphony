# Symphony TOPS Audit Vector Manifest

## Canonical Target

`knowledge/stav/`

## Owned Truth

STAV owns its event envelope, presence rules, integrity rules, append-authority contract, producer authority, query/proposal permissions, storage boundaries, redaction classifications, and projection doctrine.

## Authority Split

- `knowledge/stav/`: canonical protocol and schema truth;
- `libraries/stav-protocol-go/`: pure-Go, build-time implementation of serialization, validation, digest, identifier, and frame mechanics with no runtime authority;
- `modules/stav-append-authority/`: independently installable Go implementation of the dedicated per-TOPS append-authority role;
- SSIAG and node-troll: initially authorized producer classes;
- qxctl: canonical administrative and query interface implementing the protocol;
- agents: query and proposal authority only.

qxctl does not own the schema. Producers do not choose sequence numbers or edit ledger files. Agents do not append directly.

## Operational Storage

- user: `${XDG_STATE_HOME}/symphony/<tops-id>/stav/`
- system: `/var/lib/symphony/<tops-id>/stav/`

Operational state is never stored under `knowledge/stav/`. DuckDB, HDF5, JSONL exports, graphs, and embeddings are derived, disposable projections and never canonical records.

## Status

Owner-ratified append-authority architecture, canonical candidate/event/receipt/read schemas, strict I-JSON/JCS profile, digest domains, local frame mechanics, protocol-kernel namespace, and bounded read-only qxctl grammar. No operational listener or ledger writer is enabled until local envelopes, authentication/authorization, durability, recovery, retention, rotation, repair, and producer/reader contracts pass.
