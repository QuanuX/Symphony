# qxctl Intent

qxctl is the Go-based local administrative spine for Symphony.

## Purpose
qxctl is a repository and module inspection/control surface. It is a deterministic local status/inventory/digest tool designed to read and report Symphony repository state and to speak safe administrative commands to independently installed modules.

## Scope
qxctl operates as a local utility to verify modules, aggregate contracts, digest runtime inventory, and query metadata-only module control APIs. It uses Go 1.26.5 as the current scripted baseline and targets Go 1.27 after general availability and differential conformance. A toolchain migration cannot change protocol bytes or command grammar.

The initial secure-identity-access-governance integration is local and read-only. qxctl reads SSIAG health and safe provider descriptors over a Unix domain socket. qxctl does not receive or persist credential values.

Every SSIAG query is scoped by immutable TOPS ID. `knowledge/ssiag/` owns SSIAG protocol truth; qxctl only implements its administrative/query projection. Future administrative change separates deterministic `propose` from authorized local `apply`; AI agents may never apply.

The Architect-ratified `qxctl stav status|verify|query|doctor` grammar is operational. It loads the selected per-TOPS STAV contract, authenticates the authority endpoint from kernel credentials, submits strict local envelopes, and displays only classification-authorized projections. qxctl has no `stav append`, does not edit STAV ledgers, and does not own `knowledge/stav/` schemas. qxctl grammar is not governed by OpenAPI.

## Non-goals
- qxctl does not execute hotpath-runtime workloads.
- qxctl does not make bus traversal mandatory.
- qxctl does not require Python.
- qxctl does not perform remote execution.
- qxctl does not manage NATS directly.
- qxctl does not own hotpath-runtime execution.
- qxctl does not replace node-troll.
- qxctl does not replace bus-troll.
- qxctl does not replace hotpath-runtime.
- qxctl does not choose infrastructure.
- qxctl does not assume Docker/Kubernetes/cloud.
- qxctl does not assume trading, market-data, strategy, provider, or plugin ABI behavior.
- qxctl does not write generated SKVI/SCLV records.
- qxctl does not enforce runtime behavior.
- qxctl does not implement identity-provider, keyring, or secret-provider SDK behavior.
- qxctl does not accept or print secret values through SSIAG commands.

## Relationship
qxctl reads and reports Symphony repository state. It relates to node-troll, bus-troll, hotpath-runtime, and secure-identity-access-governance as an administrative command and inspection surface, not as an owner of their workloads or security state.
