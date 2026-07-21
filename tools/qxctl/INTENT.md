# qxctl Intent

qxctl is the Go-based local administrative spine for Symphony.

## Purpose
qxctl is a repository and module inspection/control surface. It is a deterministic local status/inventory/digest tool designed to read and report Symphony repository state and to speak safe administrative commands to independently installed modules.

## Scope
qxctl operates as a local utility to verify modules, aggregate contracts, digest runtime inventory, and query metadata-only module control APIs. It uses Go 1.26.5 as the current scripted baseline and targets Go 1.27 after general availability and differential conformance. A toolchain migration cannot change protocol bytes or command grammar.

The command tree and flag grammar use Cobra. Viper is a constrained command-configuration mapper: each command configuration that maps an environment value receives a private instance, all keys and environment variables are bound explicitly, and no automatic environment discovery, remote provider, configuration-file discovery, watch/reload, write-back, or secret value is permitted. Dedicated SSIAG and STAV clients retain exclusive responsibility for trusted configuration loading and endpoint authentication.

The initial secure-identity-access-governance integration is local and read-only. qxctl reads SSIAG health and safe provider descriptors over a Unix domain socket. qxctl does not receive or persist credential values.

Every SSIAG query is scoped by immutable TOPS ID. `knowledge/ssiag/` owns SSIAG protocol truth; qxctl only implements its administrative/query projection. Future administrative change separates deterministic `propose` from permission-backed local `apply`. Authorization depends on target-host ownership or granted permission, the requested operation and resource, expected state, and owner-configured safeguards; qxctl does not request or evaluate caller type.

The Architect-ratified `qxctl stav status|verify|query|doctor` grammar is operational. It loads the selected per-TOPS STAV contract, authenticates the authority endpoint from kernel credentials, submits strict local envelopes, and displays only classification-authorized projections. qxctl has no `stav append`, does not edit STAV ledgers, and does not own `knowledge/stav/` schemas. qxctl grammar is not governed by OpenAPI.

The vector-engine grammar is ratified for implementation after its canonical contract transition merges. `qxctl knowledge ...` is the cross-vector umbrella; `qxctl skvi|sclv|sacv|sodv|ssfv ...` owns vector-specific commands. qxctl invokes exact independently installed C++ engines through the bounded standard-I/O process protocol. Initial implementation is read/query/check/propose/project only. `knowledge apply` and SSFV commands remain reserved and disabled until their separate gates pass.

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
- qxctl does not directly write generated SKVI/SCLV records; it may request noncanonical proposals from ratified engines.
- qxctl does not enforce runtime behavior.
- qxctl does not implement identity-provider, keyring, or secret-provider SDK behavior.
- qxctl does not accept or print secret values through SSIAG commands.
- qxctl does not grant target-host authority or make caller-class policy.

## Relationship
qxctl reads and reports Symphony repository state and administers independently installed modules and vector engines. It relates to node-troll, bus-troll, hotpath-runtime, secure-identity-access-governance, STAV, and SKV engines as an administrative command and inspection surface, not as an owner of their workloads, schemas, semantics, or security state.
