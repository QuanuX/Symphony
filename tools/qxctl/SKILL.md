# qxctl Skill

## How Callers Should Use qxctl
Any caller operating within its effective target-host permission should use `qxctl` as the primary local administrative spine to verify repository status, module integrity, and runtime inventory. Caller type does not expand or reduce authority.

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
- `go run ./cmd/qxctl skvi check --prefix /chosen/prefix`
- `go run ./cmd/qxctl skvi project --prefix /chosen/prefix --json`
- `go run ./cmd/qxctl skvi propose --prefix /chosen/prefix --input proposal-input.json`

## Constraints
- Use the Go standard library, ratified first-party Go libraries, and only their approved cgo-free platform dependencies.
- Treat Cobra as the command grammar and Viper only as a private, explicitly bound command-configuration mapper. Do not enable `AutomaticEnv`, configuration-file discovery, remote providers, watch/reload, or write-back.
- Keep SSIAG/STAV trust configuration and endpoint authentication outside Viper in their dedicated clients.
- Run commands synchronously in the active execution session.
- SSIAG commands may read safe metadata only. Never pass secret values through qxctl arguments, input, output, logs, or fixtures.
- The current implementation is read-only for every caller. When proposal and apply support exists, use only operations permitted by the target host and satisfy the configured safeguards; never emulate, manufacture, or bypass host authority.
- STAV commands require an enrolled, running authority and an explicit reader grant. Never bypass endpoint authentication, reader classification, or add raw append behavior.
- For implemented SKVI commands, invoke only an explicit installation prefix and exact version. Treat proposal and projection output as noncanonical; `qxctl skvi propose` does not apply its result.
- Keep SKVI proposal input to the exact nonsecret operation schema. Never place credentials, proofs, raw tokens, provider payloads, environment data, or executable instructions in its semantic fields.
- Treat the default knowledge session as a login/authentication-to-logout/expiry/revocation authority epoch containing separate worktree reconciliation contexts. Never extend authority across a required re-authentication boundary.
- Keep vector administration, recovery, and audit reconciliation away from hot and warm paths.

## Do-Not-Use-For List
- Do not use qxctl for managing NATS directly.
- Do not use qxctl for deploying to cloud/Docker/Kubernetes.
- Do not use qxctl to replace `node-troll`, `bus-troll`, or `hotpath-runtime`.
- Do not use qxctl to write generated SKVI/SCLV records directly; use ratified proposal operations and the separately gated apply path when available.
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
