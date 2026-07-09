# qxctl Skill

## How Agents Should Use qxctl
Agents should use `qxctl` as the primary local administrative spine to verify repository status, module integrity, and runtime inventory.

## Command Examples
- `go run ./cmd/qxctl status`
- `go run ./cmd/qxctl status --json`
- `go run ./cmd/qxctl inventory digest`
- `go run ./cmd/qxctl modules check`

## Constraints
- Use Go standard library only.
- Run commands synchronously in the active execution session.

## Do-Not-Use-For List
- Do not use qxctl for managing NATS directly.
- Do not use qxctl for deploying to cloud/Docker/Kubernetes.
- Do not use qxctl to replace `node-troll`, `bus-troll`, or `hotpath-runtime`.
- Do not use qxctl to write generated SKVI/SCLV records.
- Do not use qxctl to enforce runtime behavior or execute hotpath workloads.

## Preferred Verification Sequence
1. `go run ./cmd/qxctl doctor`
2. `go run ./cmd/qxctl contracts`
3. `go run ./cmd/qxctl modules check`
4. `go run ./cmd/qxctl inventory digest`
5. `go run ./cmd/qxctl status`
