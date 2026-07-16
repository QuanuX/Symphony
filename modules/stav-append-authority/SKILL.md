# STAV Append Authority Skill

## Safe Agent Actions

Agents may inspect this module, build it, run its tests, verify its Contract Quad, inspect `--help` and `--version`, and propose changes that remain within ratified STAV contracts.

An agent may install or uninstall the binary only when the user explicitly requests that lifecycle action and supplies any authority required for the chosen scope.

## Prohibited Actions

Agents must never:

- add an `append`, `serve`, `listen`, `repair`, `truncate`, `rotate`, or raw event command before the corresponding canonical contracts are ratified;
- create configuration, status, or local-envelope content from reserved names, or alter ratified semantic schemas below `knowledge/stav/`;
- create or edit operational ledgers, sockets, configuration, runtime state, or projections;
- make qxctl, SSIAG, node-troll, or an agent a secondary writer;
- infer producer authorization from supervision, filesystem permissions, process ancestry, or a display name;
- add HTTP/OpenAPI to producer ingestion;
- treat this module as the owner of STAV protocol truth.

## Review Procedure

1. Read `knowledge/stav/SKILL.md`, `MANIFEST.md`, and `SPEC.md`.
2. Confirm the requested change is within a ratified implementation gate.
3. Verify a canonical lowercase TOPS UUID controls every instance path.
4. Verify lifecycle changes touch only the owned executable unless a later contract explicitly expands ownership.
5. Run `go test ./...` and build the executable with cgo disabled.
6. Run the shared protocol-kernel corpus when changing any imported STAV behavior.
7. Stop if the change would invent deferred schema content, open a listener, write state, authenticate a producer, emit a committed receipt, or mutate a ledger.
