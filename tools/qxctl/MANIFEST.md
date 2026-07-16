# qxctl Manifest

## Identity
- Declared tool name: qxctl
- Path: tools/qxctl
- Language/Runtime: Go 1.26.5 (standard library plus first-party pure-Go STAV protocol kernel; no third-party dependency)

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
- `qxctl stav status --tops-id UUID [--scope user|system] [--json]` (reserved, fail-closed)
- `qxctl stav verify --tops-id UUID [--scope user|system] [--json]` (reserved, fail-closed)
- `qxctl stav query --tops-id UUID [--scope user|system] [bounded filters] [--json]` (reserved, fail-closed after validation)
- `qxctl stav doctor --tops-id UUID [--scope user|system]` (reserved, fail-closed)

## Installability Posture
qxctl is installable via standard `go build` or executable directly via `go run` using the Go standard toolchain. It does not require remote runtimes, providers, Docker, Kubernetes, or cloud infrastructure.

The SSIAG command group is a standard-library-only client for a local Unix domain socket. Provider implementations and dependencies remain inside the independently installed SSIAG module.

`knowledge/ssiag/` owns SSIAG protocol truth and `knowledge/stav/` owns STAV protocol truth. qxctl implements administrative and query interfaces; it does not own either schema, edit ledgers, or hold runtime security state.

The STAV query/query-page/verification schemas and bounded query grammar are ratified. The current implementation uses the first-party kernel to validate TOPS identity and query content, resolves no connection, and returns an explicit runtime gate error for all four commands. Local envelope/status content and authenticated reader transport remain deferred. Raw `qxctl stav append` is prohibited.

Future mutation support must expose distinct proposal and apply paths. Proposal is deterministic and non-mutating. Apply is local-only, peer-authenticated, SSIAG-authorized, replay/idempotency/expected-state bound, STAV-gated, and unavailable to AI agents. No mutation command is implemented by the present read-only SSIAG client.

`knowledge/sacv/` governs HTTP API contracts. It does not govern qxctl CLI grammar, and qxctl does not own or generate canonical OpenAPI descriptions.

## Non-authorizations
qxctl is not authorized to write generated artifacts. It is not authorized to introduce third-party Go dependencies, Cobra, or Python. First-party Symphony libraries remain subordinate to their canonical knowledge vectors.
qxctl is not authorized to accept, store, or print secret values.
qxctl is not authorized to grant apply authority to an AI agent or to bypass STAV availability for a security/configuration mutation.
