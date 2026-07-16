# qxctl Skill

## How Agents Should Use qxctl
Agents should use `qxctl` as the primary local administrative spine to verify repository status, module integrity, and runtime inventory.

## Command Examples
- `go run ./cmd/qxctl status`
- `go run ./cmd/qxctl status --json`
- `go run ./cmd/qxctl inventory digest`
- `go run ./cmd/qxctl modules check`
- `go run ./cmd/qxctl ssiag doctor --tops-id UUID`
- `go run ./cmd/qxctl ssiag status --tops-id UUID --json`
- `go run ./cmd/qxctl ssiag providers --tops-id UUID --json`
- `go run ./cmd/qxctl stav status --tops-id UUID`
- `go run ./cmd/qxctl stav verify --tops-id UUID`
- `go run ./cmd/qxctl stav query --tops-id UUID --limit 100`

## Constraints
- Use the Go standard library, ratified first-party Go libraries, and only their approved cgo-free platform dependencies.
- Run commands synchronously in the active execution session.
- SSIAG commands may read safe metadata only. Never pass secret values through qxctl arguments, input, output, logs, or fixtures.
- When proposal support exists, agents may create and inspect proposals only. Never invoke, emulate, or bypass apply authority.
- STAV commands require an enrolled, running authority and an explicit reader grant. Never bypass endpoint authentication, reader classification, or add raw append behavior.

## Do-Not-Use-For List
- Do not use qxctl for managing NATS directly.
- Do not use qxctl for deploying to cloud/Docker/Kubernetes.
- Do not use qxctl to replace `node-troll`, `bus-troll`, or `hotpath-runtime`.
- Do not use qxctl to write generated SKVI/SCLV records.
- Do not use qxctl to enforce runtime behavior or execute hotpath workloads.
- Do not use qxctl to implement provider SDK behavior or bypass SSIAG policy.
- Do not use qxctl to append STAV events or edit ledger files.

## Preferred Verification Sequence
1. `go run ./cmd/qxctl doctor`
2. `go run ./cmd/qxctl contracts`
3. `go run ./cmd/qxctl modules check`
4. `go run ./cmd/qxctl inventory digest`
5. `go run ./cmd/qxctl status`
6. `go run ./cmd/qxctl ssiag doctor --tops-id UUID` when the selected SSIAG enrollment is running
7. `go run ./cmd/qxctl stav doctor --tops-id UUID` when the selected STAV enrollment is running
