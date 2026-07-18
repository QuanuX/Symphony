# qxctl Manifest

## Identity
- Declared tool name: qxctl
- Path: tools/qxctl
- Language/Runtime: Go 1.26.5 (first-party STAV protocol/authority client plus cgo-free `golang.org/x/sys` peer credentials)

## Expected Files
- `INTENT.md`
- `MANIFEST.md`
- `INSTALL.md`
- `SKILL.md`
- `README.md`
- `cmd/qxctl/main.go`

## Supported Commands
- `qxctl doctor`
- `qxctl contracts`
- `qxctl modules`
- `qxctl module inspect <module-name>`
- `qxctl module check <module-name>`
- `qxctl modules check`
- `qxctl module metadata <module-name>`
- `qxctl modules metadata`
- `qxctl inventory`
- `qxctl inventory digest`
- `qxctl status`
- `qxctl ssiag status --tops-id UUID [--json] [--scope user|system]`
- `qxctl ssiag providers --tops-id UUID [--json] [--scope user|system]`
- `qxctl ssiag doctor --tops-id UUID [--scope user|system]`
- `qxctl stav status --tops-id UUID [--scope user|system] [--json]`
- `qxctl stav verify --tops-id UUID [--scope user|system] [--json]`
- `qxctl stav query --tops-id UUID [--scope user|system] [bounded filters] [--json]`
- `qxctl stav doctor --tops-id UUID [--scope user|system]`

## Installability Posture
qxctl is installable via standard `go build` or executable directly via `go run` using the Go standard toolchain. It does not require remote runtimes, providers, Docker, Kubernetes, or cloud infrastructure.

The SSIAG command group is a cgo-free client for a local Unix domain socket. It loads scope-exact per-TOPS endpoint trust, rejects unsafe configuration/socket metadata, and verifies the connected service through native kernel peer credentials before HTTP exchange. Provider implementations remain inside the independently installed SSIAG module.

`knowledge/ssiag/` owns SSIAG protocol truth and `knowledge/stav/` owns STAV protocol truth. qxctl implements administrative and query interfaces; it does not own either schema, edit ledgers, or hold runtime security state.

The STAV commands use canonical local envelopes and a mutually authenticated Unix-socket client. The append authority enforces reader identity and classifications before projection. Raw `qxctl stav append` is prohibited.

Future mutation support must expose distinct proposal and apply paths. Proposal is deterministic and non-mutating. Apply is local-only, peer-authenticated, SSIAG-authorized, replay/idempotency/expected-state bound, STAV-gated, and unavailable to AI agents. No mutation command is implemented by the present read-only SSIAG client.

`knowledge/sacv/` governs HTTP API contracts. It does not govern qxctl CLI grammar, and qxctl does not own or generate canonical OpenAPI descriptions.

## Non-authorizations
qxctl is not authorized to write generated artifacts. It is not authorized to introduce Cobra, Python, or unrelated third-party dependencies. The cgo-free peer-credential dependency is inherited from the ratified STAV local client. First-party Symphony libraries remain subordinate to their canonical knowledge vectors.
qxctl is not authorized to accept, store, or print secret values.
qxctl is not authorized to grant apply authority to an AI agent or to bypass STAV availability for a security/configuration mutation.
