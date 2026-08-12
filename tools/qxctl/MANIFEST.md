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
- `cmd/qxctl/lifecycle.go`
- `cmd/qxctl/lifecycle_apply.go`
- `cmd/qxctl/ssfv.go`
- `internal/knowledgeengine/client.go`
- `internal/knowledgeengine/open_relative_unix.go`
- `internal/knowledgeengine/open_relative_unsupported.go`
- `internal/knowledgebinding/registry.go`
- `internal/knowledgebinding/state_unix.go`
- `internal/knowledgebinding/state_unsupported.go`
- `internal/knowledgelifecycle/profile.go`
- `internal/knowledgelifecycle/observation.go`
- `internal/knowledgelifecycle/executor.go`
- `internal/knowledgelifecycle/install_unix.go`
- `internal/knowledgelifecycle/install_unsupported.go`
- `internal/knowledgelifecycle/runtime.go`
- `internal/knowledgelifecycle/runtime_unix.go`
- `internal/knowledgelifecycle/runtime_unsupported.go`
- `internal/knowledgelifecycle/scan_unix.go`
- `internal/knowledgelifecycle/scan_unsupported.go`
- `internal/knowledgelifecycle/state_unix.go`
- `internal/knowledgelifecycle/state_unsupported.go`

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
- `qxctl knowledge session features begin --tops-id UUID --operation-id ID --expected-journal-digest absent|DIGEST [--maestro-prefix PATH] [--maestro-version VERSION] [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl knowledge session features status --tops-id UUID [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl knowledge session features checkpoint|close --tops-id UUID --operation-id ID --expected-journal-digest DIGEST [--maestro-prefix PATH] [--maestro-version VERSION] [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl knowledge session features recover --tops-id UUID --operation-id ID (--expected-journal-digest DIGEST|--discover) [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
- `qxctl knowledge lifecycle profile list --tops-id UUID [--profile-id ID] [--scope user|system] [--state-root PATH] [--json]`
- `qxctl knowledge lifecycle profile show --tops-id UUID [--profile-id ID] [--scope user|system] [--state-root PATH] [--json]`
- `qxctl knowledge lifecycle profile set --tops-id UUID --input FILE --expected-profile-digest absent|DIGEST [--profile-id ID] [--scope user|system] [--state-root PATH] [--json]`
- `qxctl knowledge lifecycle profile remove --tops-id UUID --expected-profile-digest DIGEST [--profile-id ID] [--scope user|system] [--state-root PATH] [--json]`
- `qxctl knowledge lifecycle observe --tops-id UUID [--profile-id ID | --root PATH...] [--scope user|system] [--state-root PATH] [--json]`
- `qxctl knowledge lifecycle report --tops-id UUID [--profile-id ID] [--prior-applied-state-digest DIGEST] [--scope user|system] [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge lifecycle boot --tops-id UUID --operation-id ID --expected-journal-digest absent|DIGEST [--profile-id ID] [--prior-applied-state-digest DIGEST] [--scope user|system] [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge lifecycle status --tops-id UUID [--profile-id ID] [--scope user|system] [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge lifecycle recover --tops-id UUID --operation-id ID (--expected-journal-digest DIGEST|--discover) [--profile-id ID] [--scope user|system] [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge lifecycle apply --tops-id UUID --operation-id ID --source-journal-digest DIGEST --expected-apply-journal-digest absent|DIGEST --expected-applied-state-digest absent|DIGEST [--source-root PATH...] [--max-actions 1..4096] [--profile-id ID] [--scope user|system] [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge lifecycle apply-status --tops-id UUID [--profile-id ID] [--scope user|system] [--state-root PATH] [--repo PATH] [--json]`
- `qxctl knowledge lifecycle apply-recover --tops-id UUID --operation-id ID (--expected-apply-journal-digest DIGEST|--discover) [--profile-id ID] [--scope user|system] [--state-root PATH] [--repo PATH] [--json]`
- `qxctl maestro inventory --prefix PATH --tops-id UUID [--version VERSION] [--scope user|system] [--state-root PATH] [--repo PATH] [--ttl DURATION] [--json]`
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
- `qxctl validate scan --tops-id UUID --prefix PATH [--version VERSION] [--repo PATH] [--state-root PATH] [--profile-id ID] [--baseline-id ID] [--json]`
- `qxctl validate debug --tops-id UUID --prefix PATH [--version VERSION] [--repo PATH] [--state-root PATH] [--profile-id ID] [--baseline-id ID] [--rule ID] [--record ID] [--path PATH] [--delta new|unchanged|resolved] [--json]`
- `qxctl validate profile list|show|set|remove --tops-id UUID [protected policy flags] [--json]`
- `qxctl validate baseline create|show|remove --tops-id UUID [exact validator/baseline flags] [--json]`

## Ratified Knowledge Grammar, Not Yet Implemented

- `qxctl knowledge proposals list|show|verify`
- `qxctl knowledge apply ...` remains namespace-reserved for future canonical knowledge proposal application; it is distinct from implemented noncanonical `knowledge lifecycle apply`

Individual lifecycle action leaves remain intentionally absent. The implemented `knowledge lifecycle apply` command derives dependency-ready order from verified evidence and never accepts a caller-selected action kind. It can install/uninstall exact receipt-v2 packages from explicit trusted staged roots, update protected generic selection/activation state, and commit exact authenticated Maestro docking presence when the caller supplies an exact Maestro installation plus an exhaustive receptor set. It cannot download, mutate receipt v1, run arbitrary entry points, alter the six-role engine binding registry, replace the active coordinator, execute a docked engine, or write canonical knowledge.

## Installability Posture
qxctl is installable via standard `go build` or executable directly via `go run` using the Go standard toolchain. It does not require remote runtimes, providers, Docker, Kubernetes, or cloud infrastructure.

Cobra owns the command tree and flag grammar. Viper is restricted to a new private instance for each command configuration: keys and environment variables are bound explicitly, and automatic environment discovery, remote providers, file discovery, watch/reload, write-back, and secret values are prohibited. Viper does not load SSIAG or STAV trust configuration. Endpoint configuration, filesystem trust, and kernel peer verification remain in their dedicated clients.

The SSIAG command group is a cgo-free client for a local Unix domain socket. It loads scope-exact per-TOPS endpoint trust, rejects unsafe configuration/socket metadata, and verifies the connected service through native kernel peer credentials before HTTP exchange. Provider implementations remain inside the independently installed SSIAG module.

`knowledge/ssiag/` owns SSIAG protocol truth and `knowledge/stav/` owns STAV protocol truth. qxctl implements administrative and query interfaces; it does not own either schema, edit ledgers, or hold runtime security state.

The STAV commands use canonical local envelopes and a mutually authenticated Unix-socket client. The append authority enforces reader identity and classifications before projection. Raw `qxctl stav append` is prohibited.

Future canonical mutation support must expose distinct proposal and apply paths. Proposal is deterministic and non-mutating. Canonical apply is local-only, peer-authenticated, permission-backed through SSIAG, replay/idempotency/expected-state bound, and governed by the applicable STAV availability or explicit audit-deferred recovery contract. Authorization evaluates target-host ownership or granted permission and owner-configured safeguards, never caller type. The current SSIAG client also requests safe decisions for protected noncanonical session and lifecycle apply operations; no canonical mutation command or audit-deferred recovery path is implemented.

Validator warning administration provides the same supported inspection and control surface to every caller holding applicable target-host permission. It controls only `record|review|require` disposition and `full|summary|count` presentation after immutable raw detection. Broader future safeguards may add confirmations, quorum, delays, budgets, or step-up assurance. Path safety, bounded parsing, atomic writes, expected-state validation, ledger framing, violation detection, and secret exclusion are protocol integrity rather than optional safeguards.

`knowledge/sacv/` governs HTTP API contracts. It does not govern qxctl CLI grammar, and qxctl does not own or generate canonical OpenAPI descriptions.

`knowledge/SPEC.md` governs the cross-vector process, engine-binding, authenticated-session, worktree-reconciliation, proposal, projection, install-receipt, and docking boundaries. Vector engines are independent C++ processes; qxctl remains Go and does not dynamically link them or absorb their domain logic.

The shared knowledge-engine process client has six implemented vector/coordinator invocation consumers and validates six binding roles including the reconciliation coordinator. A separate exact-receipt path validates the independently installed nine-file Symphony Validator without adding it to the engine-binding registry. qxctl invokes its deterministic JSON mode with an empty environment and hard deadline, then validates repository/version identity, finding identities, summaries, and nested digests before policy evaluation. Filters never narrow detector execution.

Validation profiles and baselines live below `<state-root>/symphony/<tops-id>/qxctl/validation/`. Their owner-only no-follow locks/files, exact compare-and-swap, semantic retry, STSC whole-second UTC timestamps, synchronized temporary files, atomic replacement, and directory synchronization match the common validation contract. A baseline is noncanonical acknowledgement evidence, not warning deletion, resolution, or ratification. Repository or validator-version mismatch fails closed.

The user-scope binding client records one exact inactive-undocked installation per role in a protected `default` profile under `${XDG_STATE_HOME:-~/.local/state}/symphony/qxctl/knowledge/engine-bindings/`. Reads and mutations use a persistent no-follow lock file. Mutations require `absent` or the exact current registry digest, validate the full receipt and executable before binding, increment the generation only when state changes, emit the STSC whole-second UTC profile, and commit a mode-`0600` registry with file and directory durability. The v1 reader preserves compatibility with previously valid UTC fractional-second state; the next ordinary compare-and-swap generation normalizes its new timestamp without rewriting history. `doctor` revalidates stored receipt and executable digests. Bindings contain no secrets and are noncanonical. A binding alone does not invoke engines, select a repository profile, change an install receipt's inactive state, install, uninstall, dock, establish an authenticated session, or apply.

The reconciliation command layer revalidates one immutable binding snapshot before every invocation and refuses a missing, replaced, or content-mismatched coordinator or vector-engine installation. It records only role, module/engine identity, exact version, and receipt/executable digests; prefixes remain in the protected registry. The coordinator owns journal durability and recovery. qxctl owns neither the canonical schemas nor compatibility by version recency, and cannot convert an unsupported or critical future state into a downgrade.

The session command layer reads one immutable binding snapshot to resolve and revalidate the exact coordinator installation, then authenticates SSIAG through the per-TOPS trust configuration and requests one fresh exact decision per operation. It does not attach or record the reconciliation engine inventory. qxctl validates the complete decision/capability boundary before passing it to the coordinator. The coordinator owns journal durability, compatibility, and recovery; SSIAG owns policy decisions; qxctl owns neither. No component may convert safe decision evidence into transferable bearer authority or canonical apply permission.

The session transition layer performs only an explicit idempotent composition of status, bounded discovery recovery, close, begin, and checkpoint. It derives stable step identities from one host event ID, checks current journal checkpoint evidence before retry, obtains a new SSIAG decision for every step, and emits `symphony.knowledge.session-transition-result.v1`. It installs no host integration.

The lifecycle layer stores protected profiles under `<state-root>/symphony/<tops-id>/qxctl/knowledge/lifecycle/profiles/` and protected generic selection/activation state under the same TOPS-scoped qxctl state boundary. It uses caller-neutral exact SSIAG operations and resources, persistent no-follow locks, effective-user-owned mode-`0600` files, linked content digests, semantic retry, and durable replacement. Observation scans only fixed receipt layouts; authenticated Maestro presence may be overlaid from an exhaustive caller-supplied receptor set. `apply` records one attempt before external mutation, supports reviewed receipt-v2/runtime/Maestro-presence adapters, re-observes, and closes only on whole-profile convergence. Host boot-hook installation, receipt-v1 mutation, downloads, live process activation, and Maestro engine execution remain unimplemented.

## Non-authorizations
qxctl is not authorized to write canonical generated artifacts. It may invoke ratified engines to create noncanonical proposals and disposable projections. The Architect-ratified Cobra and Viper libraries and their required cgo-free Go dependencies are authorized only for command grammar and constrained configuration mapping; Python, C bindings, remote configuration backends, in-process vector execution engines, and unrelated third-party dependencies remain prohibited. First-party Symphony libraries remain subordinate to their canonical knowledge vectors.
qxctl is not authorized to accept, store, or print secret values.
qxctl is not authorized to grant host permission, classify callers, silently bypass STAV, or present protocol-integrity requirements as optional safeguards. Any future audit-deferred administrator recovery path requires its own explicit contract, durable local recovery evidence, and later STAV reconciliation.
qxctl and every administrative recovery it coordinates are prohibited from executing inline with, sharing locks with, or adding synchronous dependencies, jitter, or latency to hot or warm paths.
