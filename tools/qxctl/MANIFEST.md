# qxctl Manifest

## Identity
- Declared tool name: qxctl
- Path: tools/qxctl
- Language/Runtime: Go 1.26.5 with Cobra command grammar, constrained Viper configuration mapping, first-party STAV protocol/authority clients, and cgo-free platform dependencies

## Expected Files
- `INTENT.md`
- `MANIFEST.md`
- `INSTALL.md`
- `SKILL.md`
- `README.md`
- `cmd/qxctl/main.go`
- `cmd/qxctl/commands.go`
- `internal/knowledgeengine/client.go`
- `internal/knowledgeengine/open_relative_unix.go`
- `internal/knowledgeengine/open_relative_unsupported.go`

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
- `qxctl skvi inspect --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl skvi check --prefix PATH [--version VERSION] [--repo PATH] [--expected-index-digest DIGEST] [--json]`
- `qxctl skvi propose --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl skvi project --prefix PATH [--version VERSION] [--repo PATH] [--json]`

## Ratified Vector-Engine Grammar, Not Yet Implemented

- `qxctl knowledge engines list|inspect|doctor`
- `qxctl knowledge session begin|status|checkpoint|close|recover`
- `qxctl knowledge proposals list|show|verify`
- `qxctl sclv inspect|check|propose|recover|project`
- `qxctl sacv inspect|check|diff|propose|project`
- `qxctl sodv inspect|check|propose|verify|recover|project`
- `qxctl ssfv ...` is namespace-reserved but unavailable until the SSFV Contract Quad gate passes
- `qxctl knowledge apply ...` is namespace-reserved but unavailable until the common apply gate passes

The qxctl lifecycle administrator is also ratified for future implementation: install, upgrade, rollback, receipt inspection, dock, undock, activate, and uninstall. Exact leaf grammar is added only with its reviewed artifact-verification and receipt contract. No current `module` command should imply these operations already exist.

## Installability Posture
qxctl is installable via standard `go build` or executable directly via `go run` using the Go standard toolchain. It does not require remote runtimes, providers, Docker, Kubernetes, or cloud infrastructure.

Cobra owns the command tree and flag grammar. Viper is restricted to a new private instance for each command configuration: keys and environment variables are bound explicitly, and automatic environment discovery, remote providers, file discovery, watch/reload, write-back, and secret values are prohibited. Viper does not load SSIAG or STAV trust configuration. Endpoint configuration, filesystem trust, and kernel peer verification remain in their dedicated clients.

The SSIAG command group is a cgo-free client for a local Unix domain socket. It loads scope-exact per-TOPS endpoint trust, rejects unsafe configuration/socket metadata, and verifies the connected service through native kernel peer credentials before HTTP exchange. Provider implementations remain inside the independently installed SSIAG module.

`knowledge/ssiag/` owns SSIAG protocol truth and `knowledge/stav/` owns STAV protocol truth. qxctl implements administrative and query interfaces; it does not own either schema, edit ledgers, or hold runtime security state.

The STAV commands use canonical local envelopes and a mutually authenticated Unix-socket client. The append authority enforces reader identity and classifications before projection. Raw `qxctl stav append` is prohibited.

Future mutation support must expose distinct proposal and apply paths. Proposal is deterministic and non-mutating. Apply is local-only, peer-authenticated, permission-backed through SSIAG, replay/idempotency/expected-state bound, and governed by the applicable STAV availability or explicit audit-deferred recovery contract. Authorization evaluates target-host ownership or granted permission and owner-configured safeguards, never caller type. No mutation command or audit-deferred recovery path is implemented by the present read-only SSIAG client.

Future safeguard administration must provide the same supported inspection and control surface to every caller holding target-host administrator permission. A conservative default profile may enable confirmations, quorum, delays, budgets, step-up assurance, or similar governance interlocks. The administrator may disable or replace those optional controls, including selecting a direct profile. Path safety, bounded parsing, atomic writes, expected-state validation, ledger framing, and secret exclusion are protocol integrity rather than optional safeguards.

`knowledge/sacv/` governs HTTP API contracts. It does not govern qxctl CLI grammar, and qxctl does not own or generate canonical OpenAPI descriptions.

`knowledge/SPEC.md` governs the cross-vector process, authenticated-session, worktree-reconciliation, proposal, projection, install-receipt, and docking boundaries. Vector engines are independent C++ processes; qxctl remains Go and does not dynamically link them or absorb their domain logic.

The SKVI client is the first implemented process client. It requires an explicit prefix and exact version, accepts proposal content only from a bounded no-follow regular file, validates the exact inactive-undocked receipt and its nine owned files, provides an empty child environment, enforces the process deadline independently, and validates response identity and digest. It does not select an active version, install, uninstall, dock, or apply.

## Non-authorizations
qxctl is not authorized to write canonical generated artifacts. It may invoke ratified engines to create noncanonical proposals and disposable projections. The Architect-ratified Cobra and Viper libraries and their required cgo-free Go dependencies are authorized only for command grammar and constrained configuration mapping; Python, C bindings, remote configuration backends, in-process vector execution engines, and unrelated third-party dependencies remain prohibited. First-party Symphony libraries remain subordinate to their canonical knowledge vectors.
qxctl is not authorized to accept, store, or print secret values.
qxctl is not authorized to grant host permission, classify callers, silently bypass STAV, or present protocol-integrity requirements as optional safeguards. Any future audit-deferred administrator recovery path requires its own explicit contract, durable local recovery evidence, and later STAV reconciliation.
qxctl and every administrative recovery it coordinates are prohibited from executing inline with, sharing locks with, or adding synchronous dependencies, jitter, or latency to hot or warm paths.
