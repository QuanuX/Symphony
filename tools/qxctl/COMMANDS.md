# qxctl Command Registry

`COMMANDS.json` is the canonical machine projection of qxctl's expected command registry. It is repository-owned evidence for engine-first administration evaluation and remains usable when no qxctl executable is installed.

The source is not a second hand-maintained command list. Each public or hidden executable Cobra leaf carries one attached `CommandSpec`; one full-tree parity walk derives grammar and JSON support from Cobra, combines them with the stable semantic fields, sorts the records, and computes the registry self-digest. Structural namespaces and internal non-executable plumbing are explicit roles. Hidden prohibited leaves remain registered but cannot satisfy required coverage. A retired ID moves through the dedicated tombstone factory: its old path returns only a deterministic fail-closed diagnostic, its manifest grammar becomes `null`, and its stable ID remains unavailable for reuse.

Each executable command retains its qxctl-owned wrapper binding and may add reviewed backend feature/interaction bindings from the single table in `cmd/qxctl/command_specs.go`. The current registry contains 144 executable commands. The four SSIAG/STAV supervision and TOPS-enrollment features each bind five explicit qxctl leaves to five distinct module operations, closing all previously uncovered administration surfaces without collapsing feature, command, engine-operation, or caller-operation identity. Nine protected warning-lifecycle leaves and the exact receipt-backed root-summary route expose the new validation administration without altering raw detector evidence. The registry records actual grammar and dispatch; it never fabricates coverage for an unavailable route.

Generate and verify from `tools/qxctl`:

```text
go run ./cmd/qxctl commands expected --json > COMMANDS.json
go run ./cmd/qxctl commands verify --input COMMANDS.json --json
```

An installed client can emit executable-bound evidence with:

```text
qxctl commands manifest --json
```

Expected evidence has null client version, executable digest, and receipt digest. Observed evidence binds the exact client version and executable digest, plus a receipt digest when one is available. Both forms use `symphony.qxctl.command-registry.v1` and cover every public or hidden executable leaf, including executable parents with subcommands.

The registry records identities and reviewed evidence; it does not inject commands, allocate names, infer feature-worthiness, or grant authority. Any proposal generator must know the registry protocol, the target feature and interaction, stable engine operation and I/O protocols, mutability and authority boundary, recovery requirements, noninteractive/JSON behavior, and required tests. Only reviewed source changes can ratify a new `qxcmd` identity, regardless of caller class.
