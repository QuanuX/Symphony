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
- `cmd/qxctl/ssfv.go`
- `internal/knowledgeengine/client.go`
- `internal/knowledgeengine/open_relative_unix.go`
- `internal/knowledgeengine/open_relative_unsupported.go`
- `internal/knowledgebinding/registry.go`
- `internal/knowledgebinding/state_unix.go`
- `internal/knowledgebinding/state_unsupported.go`

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
- `qxctl knowledge engines list [--state-root PATH] [--json]`
- `qxctl knowledge engines inspect ROLE [--state-root PATH] [--json]`
- `qxctl knowledge engines doctor [--state-root PATH] [--json]`
- `qxctl knowledge engines bind ROLE --prefix PATH [--version VERSION] --expected-registry-digest absent|DIGEST [--state-root PATH] [--json]`
- `qxctl knowledge engines unbind ROLE --expected-registry-digest DIGEST [--state-root PATH] [--json]`
- `qxctl knowledge reconcile compatibility [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge reconcile begin --operation-id ID --expected-journal-digest absent|DIGEST --path FILE... [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge reconcile status [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge reconcile checkpoint --operation-id ID --expected-journal-digest DIGEST [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge reconcile close --operation-id ID --expected-journal-digest DIGEST [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge reconcile recover --operation-id ID (--expected-journal-digest DIGEST|--discover) [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge session begin --tops-id UUID --operation-id ID --expected-journal-digest absent|DIGEST [--context-ref REF...] [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl knowledge session status --tops-id UUID [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl knowledge session checkpoint --tops-id UUID --operation-id ID --expected-journal-digest DIGEST [--context-ref REF...] [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl knowledge session close --tops-id UUID --operation-id ID --expected-journal-digest DIGEST [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl knowledge session recover --tops-id UUID --operation-id ID (--expected-journal-digest DIGEST|--discover) [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl knowledge session transition --tops-id UUID --event login|refresh|logout --event-id ID [--context-ref REF...] [--recover] [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl skvi inspect --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl skvi check --prefix PATH [--version VERSION] [--repo PATH] [--expected-index-digest DIGEST] [--json]`
- `qxctl skvi propose --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl skvi project --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl sclv inspect --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl sclv check --prefix PATH [--version VERSION] [--repo PATH] [--expected-ledger-digest DIGEST] [--json]`
- `qxctl sclv propose --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl sclv recover --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl sclv project --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl sacv inspect --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl sacv check --prefix PATH [--version VERSION] [--repo PATH] [--expected-registry-digest DIGEST] [--json]`
- `qxctl sacv diff --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl sacv propose --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl sacv project --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl sodv inspect --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl sodv check --prefix PATH [--version VERSION] [--repo PATH] [--expected-ledger-digest DIGEST] [--json]`
- `qxctl sodv verify --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl sodv propose --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl sodv recover --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl sodv project --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl ssfv inspect --prefix PATH [--version VERSION] [--repo PATH] [--json]`
- `qxctl ssfv check --prefix PATH [--version VERSION] [--repo PATH] [--expected-namespace-digest DIGEST] [--expected-registry-digest DIGEST] [--baseline FILE] [--freshness disabled|report|require] [--json]`
- `qxctl ssfv diff --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl ssfv propose --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl ssfv graph --prefix PATH [--version VERSION] [--repo PATH] [--json]`

## Ratified Knowledge Grammar, Not Yet Implemented

- `qxctl knowledge proposals list|show|verify`
- `qxctl knowledge apply ...` is namespace-reserved but unavailable until the common apply gate passes

The qxctl lifecycle administrator is also ratified for future implementation: desired-profile administration, report-only boot convergence, install, upgrade, rollback, receipt inspection, dock, undock, activate, and uninstall. The canonical common lifecycle and receipt-v2 contracts now bound that work; exact leaf grammar is added only with its reviewed runtime artifact-verification and authorization contracts. No current `module` or `knowledge lifecycle` command should imply these operations already exist.

## Installability Posture
qxctl is installable via standard `go build` or executable directly via `go run` using the Go standard toolchain. It does not require remote runtimes, providers, Docker, Kubernetes, or cloud infrastructure.

Cobra owns the command tree and flag grammar. Viper is restricted to a new private instance for each command configuration: keys and environment variables are bound explicitly, and automatic environment discovery, remote providers, file discovery, watch/reload, write-back, and secret values are prohibited. Viper does not load SSIAG or STAV trust configuration. Endpoint configuration, filesystem trust, and kernel peer verification remain in their dedicated clients.

The SSIAG command group is a cgo-free client for a local Unix domain socket. It loads scope-exact per-TOPS endpoint trust, rejects unsafe configuration/socket metadata, and verifies the connected service through native kernel peer credentials before HTTP exchange. Provider implementations remain inside the independently installed SSIAG module.

`knowledge/ssiag/` owns SSIAG protocol truth and `knowledge/stav/` owns STAV protocol truth. qxctl implements administrative and query interfaces; it does not own either schema, edit ledgers, or hold runtime security state.

The STAV commands use canonical local envelopes and a mutually authenticated Unix-socket client. The append authority enforces reader identity and classifications before projection. Raw `qxctl stav append` is prohibited.

Future canonical mutation support must expose distinct proposal and apply paths. Proposal is deterministic and non-mutating. Apply is local-only, peer-authenticated, permission-backed through SSIAG, replay/idempotency/expected-state bound, and governed by the applicable STAV availability or explicit audit-deferred recovery contract. Authorization evaluates target-host ownership or granted permission and owner-configured safeguards, never caller type. The current SSIAG client requests safe decisions for protected noncanonical session operations only; no canonical mutation command or audit-deferred recovery path is implemented.

Future safeguard administration must provide the same supported inspection and control surface to every caller holding target-host administrator permission. A conservative default profile may enable confirmations, quorum, delays, budgets, step-up assurance, or similar governance interlocks. The administrator may disable or replace those optional controls, including selecting a direct profile. Path safety, bounded parsing, atomic writes, expected-state validation, ledger framing, and secret exclusion are protocol integrity rather than optional safeguards.

`knowledge/sacv/` governs HTTP API contracts. It does not govern qxctl CLI grammar, and qxctl does not own or generate canonical OpenAPI descriptions.

`knowledge/SPEC.md` governs the cross-vector process, engine-binding, authenticated-session, worktree-reconciliation, proposal, projection, install-receipt, and docking boundaries. Vector engines are independent C++ processes; qxctl remains Go and does not dynamically link them or absorb their domain logic.

The shared knowledge-engine process client has six implemented invocation consumers and validates six binding roles including the reconciliation coordinator. SKVI, SACV, SODV, SSFV, and the coordinator each validate an exact inactive-undocked nine-file receipt; SCLV validates an exact inactive-undocked eleven-file receipt containing its engine and two provider-evidence adapters. All require an explicit prefix and exact version. The prefix and receipt-owned files must be owned by the effective user or root and not writable by group or other. Clients accept proposal/diff/verification/recovery/baseline content only from a bounded no-follow regular file, provide an empty child environment, enforce the process deadline independently, and validate response identity and digest. The vector command layers additionally reject self-ratification, ownership or membership escalation, engine-declared completion, journal mutation, canonical projection/diff/graph status, listener enablement, or apply. SSFV additionally validates freshness mode/baseline coupling and exact vector-owned proposal targets.

The user-scope binding client records one exact inactive-undocked installation per role in a protected `default` profile under `${XDG_STATE_HOME:-~/.local/state}/symphony/qxctl/knowledge/engine-bindings/`. Reads and mutations use a persistent no-follow lock file. Mutations require `absent` or the exact current registry digest, validate the full receipt and executable before binding, increment the generation only when state changes, and commit a mode-`0600` registry with file and directory durability. `doctor` revalidates stored receipt and executable digests. Bindings contain no secrets and are noncanonical. A binding alone does not invoke engines, select a repository profile, change an install receipt's inactive state, install, uninstall, dock, establish an authenticated session, or apply.

The reconciliation command layer revalidates one immutable binding snapshot before every invocation and refuses a missing, replaced, or content-mismatched coordinator or vector-engine installation. It records only role, module/engine identity, exact version, and receipt/executable digests; prefixes remain in the protected registry. The coordinator owns journal durability and recovery. qxctl owns neither the canonical schemas nor compatibility by version recency, and cannot convert an unsupported or critical future state into a downgrade.

The session command layer reads one immutable binding snapshot to resolve and revalidate the exact coordinator installation, then authenticates SSIAG through the per-TOPS trust configuration and requests one fresh exact decision per operation. It does not attach or record the reconciliation engine inventory. qxctl validates the complete decision/capability boundary before passing it to the coordinator. The coordinator owns journal durability, compatibility, and recovery; SSIAG owns policy decisions; qxctl owns neither. No component may convert safe decision evidence into transferable bearer authority or canonical apply permission.

The session transition layer performs only an explicit idempotent composition of status, bounded discovery recovery, close, begin, and checkpoint. It derives stable step identities from one host event ID, checks current journal checkpoint evidence before retry, obtains a new SSIAG decision for every step, and emits `symphony.knowledge.session-transition-result.v1`. It installs no host integration. Cross-vector desired-state and first-boot contracts are canonical under `knowledge/LIFECYCLE.md`, but their persistence, observer, dependency-driven runtime planner, recovery, and apply leaves remain unimplemented.

## Non-authorizations
qxctl is not authorized to write canonical generated artifacts. It may invoke ratified engines to create noncanonical proposals and disposable projections. The Architect-ratified Cobra and Viper libraries and their required cgo-free Go dependencies are authorized only for command grammar and constrained configuration mapping; Python, C bindings, remote configuration backends, in-process vector execution engines, and unrelated third-party dependencies remain prohibited. First-party Symphony libraries remain subordinate to their canonical knowledge vectors.
qxctl is not authorized to accept, store, or print secret values.
qxctl is not authorized to grant host permission, classify callers, silently bypass STAV, or present protocol-integrity requirements as optional safeguards. Any future audit-deferred administrator recovery path requires its own explicit contract, durable local recovery evidence, and later STAV reconciliation.
qxctl and every administrative recovery it coordinates are prohibited from executing inline with, sharing locks with, or adding synchronous dependencies, jitter, or latency to hot or warm paths.
