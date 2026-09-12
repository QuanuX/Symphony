# qxctl Manifest

## Canonical Surfaces

- `tools/qxctl/COMMANDS.json`
- `tools/qxctl/COMMANDS.md`
- `tools/qxctl/ERRORS.md`
- `tools/qxctl/FEATURES.md`
- `tools/qxctl/INSTALL.md`
- `tools/qxctl/INTENT.md`
- `tools/qxctl/MANIFEST.md`
- `tools/qxctl/README.md`
- `tools/qxctl/SKILL.md`

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
- `COMMANDS.md`
- `COMMANDS.json`
- `FEATURES.md`
- `cmd/qxctl/main.go`
- `cmd/qxctl/commands.go`
- `cmd/qxctl/command_manifest.go`
- `cmd/qxctl/command_specs.go`
- `cmd/qxctl/knowledge_invariant.go`
- `cmd/qxctl/foundation_lifecycle.go`
- `cmd/qxctl/lifecycle.go`
- `cmd/qxctl/lifecycle_apply.go`
- `cmd/qxctl/ssfv.go`
- `internal/commandregistry/registry.go`
- `internal/foundationlifecycle/client.go`
- `internal/knowledgeengine/client.go`
- `internal/knowledgeengine/open_relative_unix.go`
- `internal/knowledgeengine/open_relative_unsupported.go`
- `internal/invariantregistry/registry.go`
- `internal/knowledgebinding/registry.go`
- `internal/knowledgebinding/state_unix.go`
- `internal/knowledgebinding/state_unsupported.go`
- `internal/knowledgelifecycle/profile.go`
- `internal/knowledgelifecycle/observation.go`
- `internal/knowledgelifecycle/ownership.go`
- `internal/knowledgelifecycle/ownership_unix.go`
- `internal/knowledgelifecycle/ownership_unsupported.go`
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
- `qxctl commands manifest --json`
- `qxctl commands expected --json`
- `qxctl commands verify --input FILE --json`
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
- `qxctl ssiag grants lifecycle --tops-id UUID --subject-id ID [--profile-id ID] [--authority-basis host_owner|granted_permission] [--json]`
- `qxctl ssiag providers --tops-id UUID [--json] [--scope user|system]`
- `qxctl ssiag provider show PROVIDER --tops-id UUID [--json] [--scope user|system]`
- `qxctl ssiag provider verify PROVIDER --tops-id UUID [--authority-basis host_owner|granted_permission] [--json] [--scope user|system]`
- `qxctl ssiag doctor --tops-id UUID [--scope user|system]`
- `qxctl ssiag enrollment status|plan|apply|apply-status|recover --prefix PATH --tops-id UUID [--version VERSION] [operation flags] [--json]`
- `qxctl ssiag supervisor status|plan|apply|apply-status|recover --prefix PATH --tops-id UUID [--version VERSION] [operation flags] [--json]`
- `qxctl stav status --tops-id UUID [--scope user|system] [--json]`
- `qxctl stav verify --tops-id UUID [--scope user|system] [--json]`
- `qxctl stav query --tops-id UUID [--scope user|system] [bounded filters] [--json]`
- `qxctl stav doctor --tops-id UUID [--scope user|system]`
- `qxctl stav enrollment status|plan|apply|apply-status|recover --prefix PATH --tops-id UUID [--version VERSION] [operation flags] [--json]`
- `qxctl stav supervisor status|plan|apply|apply-status|recover --prefix PATH --tops-id UUID [--version VERSION] [operation flags] [--json]`
- `qxctl knowledge engines list [--state-root PATH] [--json]`
- `qxctl knowledge engines inspect ROLE [--state-root PATH] [--json]`
- `qxctl knowledge engines doctor [--state-root PATH] [--json]`
- `qxctl knowledge engines bind ROLE --prefix PATH [--version VERSION] --expected-registry-digest absent|DIGEST [--state-root PATH] [--json]`
- `qxctl knowledge engines unbind ROLE --expected-registry-digest DIGEST [--state-root PATH] [--json]`
- `qxctl knowledge engines migrate --expected-registry-digest DIGEST [--state-root PATH] [--json]`
- `qxctl knowledge invariant status [--repo PATH] [--json]`
- `qxctl knowledge invariant list [--repo PATH] [--json]`
- `qxctl knowledge invariant show --invariant-id ID [--repo PATH] [--json]`
- `qxctl knowledge invariant check --prefix PATH [--version VERSION] [--repo PATH] [--json]`
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
- `qxctl knowledge lifecycle ownership status --tops-id UUID --root PATH [--profile-id ID] [--scope user|system] [--state-root PATH] [--json]`
- `qxctl knowledge lifecycle ownership reconcile --tops-id UUID --root PATH [--profile-id ID] [--scope user|system] [--state-root PATH] [--json]`
- `qxctl knowledge lifecycle ownership adopt --tops-id UUID --root PATH --expected-ownership-registry-digest DIGEST [--profile-id ID] [--scope user|system] [--state-root PATH] [--json]`
- `qxctl knowledge lifecycle ownership release --tops-id UUID --root PATH --receipt-digest DIGEST --expected-ownership-registry-digest DIGEST [--profile-id ID] [--scope user|system] [--state-root PATH] [--json]`
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
- `qxctl sclv evidence local-git --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
- `qxctl sclv evidence airgap --prefix PATH --input FILE [--version VERSION] [--repo PATH] [--json]`
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
- `qxctl ssfv administration-check --prefix PATH --input FILE --json [--version VERSION] [--repo PATH]`
- `qxctl validate scan --tops-id UUID --prefix PATH [--version VERSION] [--repo PATH] [--state-root PATH] [--profile-id ID] [--baseline-id ID] [--json]`
- `qxctl validate debug --tops-id UUID --prefix PATH [--version VERSION] [--repo PATH] [--state-root PATH] [--profile-id ID] [--baseline-id ID] [--rule ID] [--record ID] [--path PATH] [--delta new|unchanged|resolved] [--json]`
- `qxctl validate profile list|show|set|remove --tops-id UUID [protected policy flags] [--json]`
- `qxctl validate baseline create|show|remove --tops-id UUID [exact validator/baseline flags] [--json]`
- `qxctl validate warning status|list|show|sync|accept|reopen|supersede|mute|unmute --tops-id UUID [exact lifecycle/CAS flags] [--json]`
- `qxctl validate root-summary --prefix PATH [--version VERSION] [--repo PATH] [--json]`

## Ratified Knowledge Grammar, Not Yet Implemented

- `qxctl knowledge proposals list|show|verify`
- `qxctl knowledge apply ...` remains namespace-reserved for future canonical knowledge proposal application; it is distinct from implemented noncanonical `knowledge lifecycle apply`

Individual lifecycle action leaves remain intentionally absent. The implemented `knowledge lifecycle apply` command derives dependency-ready order from verified evidence and never accepts a caller-selected action kind. It can install exact receipt-v2 packages from explicit trusted staged roots and reclaim them only after enforced root-local multi-profile claim resolution, update protected generic selection/activation state, atomically select exact receipt-v2 installations for the eight established binding roles (coordinator, SKVI, SCLV, SACV, SODV, SSFV, SAV, and SEV), perform a candidate-verified coordinator handoff, and commit exact authenticated Maestro docking presence when the caller supplies an exact Maestro installation plus an exhaustive receptor set. It cannot download, mutate receipt v1, run arbitrary entry points, add roles to the closed registry, replace a coordinator in place, execute a docked engine, or write canonical knowledge.

## Installability Posture
qxctl is installable via standard `go build` or executable directly via `go run` using the Go standard toolchain. It does not require remote runtimes, providers, Docker, Kubernetes, or cloud infrastructure.

Cobra owns the command tree and flag grammar. Viper is restricted to a new private instance for each command configuration: keys and environment variables are bound explicitly, and automatic environment discovery, remote providers, file discovery, watch/reload, write-back, and secret values are prohibited. Viper does not load SSIAG or STAV trust configuration. Endpoint configuration, filesystem trust, and kernel peer verification remain in their dedicated clients.

Every public or hidden executable Cobra leaf also owns one attached stable `CommandSpec`. Tree construction fails closed when executable grammar is unclassified, duplicated, or disguised as a structural namespace. `commands expected` and the checked-in `COMMANDS.json` provide client-independent design evidence for engine-first evaluation; `commands manifest` binds the same projection to the exact running executable; `commands verify` strictly validates a bounded no-follow expected registry against the current tree. The registry contains 227 commands, including complete SAV and SEV report/proposal surfaces, explicit engine-binding registry migration, and 29 SCV source/knowledge/corpus/projection/connection leaves. Six provider-installation and provider-binding leaves complement the two singular provider-trust leaves and one permission-backed provider-readiness leaf without changing the plural metadata-list or doctor semantics. The four foundational lifecycle features each register five stable qxctl commands and five distinct module operations. The four read-only invariant leaves bind `ssfv:symphony:qxctl.invariant-assurance`; `knowledge invariant check` additionally binds `ssfv:symphony:symphony-validator.invariant-ownership-assurance`. Hidden prohibited leaves remain explicit identity evidence but cannot satisfy required coverage. These commands do not inject grammar, infer command names, decide feature worthiness, invent backend operation identities, or grant authority.

The SSIAG command group is a cgo-free client for a local Unix domain socket. It loads scope-exact per-TOPS endpoint trust, rejects unsafe configuration/socket metadata, and verifies the connected service through native kernel peer credentials before HTTP exchange. The plural provider route remains metadata discovery. Singular show and verify routes validate one exact, closed, omit-self-digest-bound `symphony.ssiag.provider-trust-result.v1`; fresh verification uses the SSIAG-authorized `symphony.ssiag.provider.trust.verify` operation and a caller-neutral authority basis. `provider installations` consumes SSIAG-owned safe inventory and exposes only opaque exact installation identities. `provider binding status|plan|apply|apply-status|recover` validates the nine strict SSIAG lifecycle protocols, exact TOPS/scope identity, lowercase digests, safe flags, attempt stages including `candidate_verified`, and omit-self result digests. qxctl never opens a provider receipt, supplies an adapter path or raw version, chooses newest, launches an adapter, edits STAV, or transports provider payloads and secret values. SSIAG retains selection, dependency ordering, receipt and executable inspection, compare-and-swap state, STAV-before-commit, recovery, provider process launch, platform trust, and keyring access.

`knowledge/ssiag/` owns SSIAG protocol truth and `knowledge/stav/` owns STAV protocol truth. qxctl implements administrative and query interfaces; it does not own either schema, edit ledgers, or hold runtime security state.

The STAV commands use canonical local envelopes and a mutually authenticated Unix-socket client. The append authority enforces reader identity and classifications before projection. Raw `qxctl stav append` is prohibited.

The foundational `ssiag|stav enrollment|supervisor status|plan|apply|apply-status|recover` families invoke only an exact receipt-v2 adapter selected beneath the required installation prefix. Omitting `--version` succeeds only when one compatible version exists. qxctl independently verifies package ownership, executable and receipt digests, adapter capabilities, canonical observation/plan/result digests, operation shape, deadlines, compare-and-swap evidence, and audit/recovery disposition. It never imports module internals, renders native descriptors, invokes `serve`, selects a newest version, exposes purge, or treats itself as mutation authority.

Future canonical mutation support must expose distinct proposal and apply paths. Proposal is deterministic and non-mutating. Canonical apply is local-only, peer-authenticated, permission-backed through SSIAG, replay/idempotency/expected-state bound, and governed by the applicable STAV availability or explicit audit-deferred recovery contract. Authorization evaluates target-host ownership or granted permission and owner-configured safeguards, never caller type. The current SSIAG client also requests safe decisions for protected noncanonical session and lifecycle apply operations; no canonical mutation command or audit-deferred recovery path is implemented.

Validator warning administration provides the same supported inspection and control surface to every caller holding applicable target-host permission. It controls `record|review|require` disposition and `full|summary|count` presentation after immutable raw detection. Its side-by-side warning-state v1 records stable subjects, every immutable occurrence body/evidence digest, classifications `open|accepted|resolved|superseded`, optional expiring acceptance, presentation-only mute, and a digest-linked transition chain. Only a complete detector observation resolves or reopens a subject. Administrative transitions require rationale and exact CAS; mute cannot alter a policy gate. Broader future safeguards may add confirmations, quorum, delays, budgets, or step-up assurance. Path safety, bounded parsing, atomic writes, expected-state validation, ledger framing, violation detection, and secret exclusion are protocol integrity rather than optional safeguards.

`knowledge/sacv/` governs HTTP API contracts. It does not govern qxctl CLI grammar, and qxctl does not own or generate canonical OpenAPI descriptions.

`knowledge/SPEC.md` governs the cross-vector process, engine-binding, authenticated-session, worktree-reconciliation, proposal, projection, install-receipt, and docking boundaries. Vector engines are independent C++ processes; qxctl remains Go and does not dynamically link them or absorb their domain logic.

The shared knowledge-engine process client has eight implemented vector/coordinator invocation consumers and validates eight binding roles including SAV, SEV, and the reconciliation coordinator. SCLV's two evidence-normalization adapters are not new roles: their explicit commands revalidate the complete SCLV receipt-v2 package, require the exact typed adapter entry point and provider-evidence protocol, invoke it through the same bounded empty-environment process boundary, and validate the complete normalized result and digest. Normalized evidence does not give qxctl truth, permission, ratification, or canonical apply authority. A separate exact-receipt path validates the independently installed nine-file Symphony Validator without adding it to the engine-binding registry. qxctl invokes complete validation, complete invariant assurance, or the distinct root-summary projection with an empty environment and hard deadline, then strictly validates repository/version/finding or summary identity and nested digests before downstream evaluation. Invariant `check` returns the exact complete `symphony.validation.result.v1` and preserves exit 26; root-summary failure preserves exit 25 and never becomes a qxctl-authored projection. Filters and invariant presentation never narrow detector execution.

The direct `knowledge invariant status|list|show` path is a bounded, no-follow consumer of `knowledge/INVARIANT-OWNERSHIP.json`. It rejects malformed shape, identity, ordering, references, controls, or digest evidence and emits digest-bound `symphony.knowledge.invariant-query-result.v1` projections with `semantic_validity=not_asserted`. This makes invariant ownership discoverable to a headless installation but does not reimplement the owning rule or assert that producer, consumer, or real-process evidence passes. All invariant operations are local, noninteractive, read-only freezing-path commands and expose no canonical mutation or remediation authority.

Validation profiles, baselines, and warning-state streams live below `<state-root>/symphony/<tops-id>/qxctl/validation/`. Their owner-only no-follow locks/files, exact compare-and-swap, bounded history, STSC whole-second UTC timestamps, synchronized temporary files, atomic replacement, and directory synchronization match the common validation contract. A baseline is noncanonical acknowledgement evidence, not warning deletion, resolution, or ratification. A lifecycle classification never rewrites the raw result; repository or validator-version mismatch fails closed.

The user-scope binding client records one exact inactive-undocked installation per role in a protected `default` profile under `${XDG_STATE_HOME:-~/.local/state}/symphony/qxctl/knowledge/engine-bindings/`. Reads and mutations use a persistent no-follow lock file. New registries and post-migration mutations use `symphony.knowledge.engine-binding-registry.v2`; v1 remains closed to its original six roles. Mutations require `absent` or the exact current registry digest, validate the full receipt and executable before binding, increment the generation only when state changes, emit the STSC whole-second UTC profile, and commit a mode-`0600` registry with file and directory durability. The dual reader preserves valid v1 and v2 fractional-second timestamps. Canonical v1 remains operationally readable, but any v1 mutation first requires explicit `knowledge engines migrate` with its exact digest. The migration creates one linked v2 generation, records the predecessor protocol and digest, copies every exact binding, and neither inspects installations nor infers a version. A narrowly recognized historical v1 envelope containing SAV or SEV is exposed as nonconforming migration input and cannot be used operationally before that explicit migration. Structurally valid unknown v2 roles are preserved for list/inspect but stop all operational reads and mutations in this qxctl version. `doctor` reports registry compatibility and revalidates understood stored receipt and executable digests. Bindings contain no secrets and are noncanonical. A binding alone does not invoke engines, select a repository profile, change an install receipt's inactive state, install, uninstall, dock, establish an authenticated session, or apply.

The reconciliation command layer revalidates one immutable binding snapshot before every invocation and refuses a missing, replaced, or content-mismatched coordinator or vector-engine installation. It records only role, module/engine identity, exact version, and receipt/executable digests; prefixes remain in the protected registry. The coordinator owns journal durability and recovery. qxctl owns neither the canonical schemas nor compatibility by version recency, and cannot convert an unsupported or critical future state into a downgrade.

The session command layer reads one immutable binding snapshot to resolve and revalidate the exact coordinator installation, then authenticates SSIAG through the per-TOPS trust configuration and requests one fresh exact decision per operation. It does not attach or record the reconciliation engine inventory. qxctl validates the complete decision/capability boundary before passing it to the coordinator. The coordinator owns journal durability, compatibility, and recovery; SSIAG owns policy decisions; qxctl owns neither. No component may convert safe decision evidence into transferable bearer authority or canonical apply permission.

The session transition layer performs only an explicit idempotent composition of status, bounded discovery recovery, close, begin, and checkpoint. It derives stable step identities from one host event ID, checks current journal checkpoint evidence before retry, obtains a new SSIAG decision for every step, and emits `symphony.knowledge.session-transition-result.v1`. It installs no host integration.

The lifecycle layer stores protected profiles under `<state-root>/symphony/<tops-id>/qxctl/knowledge/lifecycle/profiles/` and protected generic selection/activation state under the same TOPS-scoped qxctl state boundary. It uses caller-neutral exact SSIAG operations with a stable TOPS/profile resource plus a separate read-only per-TOPS profile-catalog resource, persistent no-follow locks, effective-user-owned mode-`0600` files, linked content digests, semantic retry, and durable replacement. Evidence digests remain independently bound by lifecycle schemas and compare-and-swap instead of causing policy-grant churn. Observation scans only fixed receipt layouts; authenticated Maestro presence may be overlaid from an exhaustive caller-supplied receptor set. Receipt v1 retains exact legacy file adapters, while receipt v2 validates its bounded content-addressed owned set and stable role identity so compatible file additions do not force qxctl-first upgrade ordering. `apply` records one attempt before external mutation, supports reviewed receipt-v2/runtime/established-binding/Maestro-presence adapters, re-observes, and closes only on whole-profile convergence. `ssiag grants lifecycle` emits grant input; `ssiag policy` administers SSIAG's protected operational overlay through the server-owned schema and state machine. Login/session hook installation, native Windows host integration, hidden watchers, canonical knowledge apply, receipt-v1 mutation, downloads, live process activation, and Maestro engine execution remain unimplemented.

## SCV source and knowledge administration

`scv` selects one installed C++ domain with `--domain` (default `scv`): `scv`, `schv`, `scev`, `schv-aws`, `schv-azure`, `schv-do`, `schv-gcp`, or `scev-cf`. `--prefix` is explicit; existing leaves default exactly to `0.1.0-dev`, while the six corpus leaves described below default to `0.2.0-dev` and the three provider/connection leaves default to `0.3.0-dev`. `--version` never means latest. These engines require complete receipt-v2 verification and the existing bounded engine-process envelope, empty child environment, operation deadline, exact engine/result identity and canonical digests. They remain independently callable C++ processes. No engine-binding v1 role is added or reinterpreted.

Pure leaves are `inspect`, `provider-onboard`, `source-plan`, `source-check`, `capture-import`, `capture-compare`, `interpret`, `graph`, `query`, `diff`, `explain`, and `evaluate`. The last four invoke the owner's `graph_query`, `graph_diff`, `graph_explain`, and `graph_evaluate`. Their inputs/results follow `knowledge/scv/SOURCE-KNOWLEDGE.md`; qxctl prints the full qualified result rather than discarding incomplete evidence. `--repo` supplies an operation working directory and need not be a Git repository. Bounded no-follow `--input` files carry the selected operation document.

`scv acquire --input FILE` consumes `{source, locator_id}` (the qxctl argument contract `symphony.qxctl.scv-acquire-input.v1`; no extra protocol key). It first invokes `source_status` to validate the exact source and installation, performs a public HTTPS GET, then invokes `capture_import`. It accepts no credential/header/cookie options and reads no proxy environment. Port 443, TLS verification, public DNS/address checks pinned to the dialed address, eight redirects, a 20-second total deadline, bounded headers and 65,536 UTF-8 body bytes define this transport adapter. Exact query parameters remain on every request; fragments remain in the configured source but are removed from HTTP requests. Every followed redirect and final resolved URI is evidence, never a source configuration update. HTTP failures, partial-content responses, byte limits, interrupted reads and unsupported encoding are explicit capture qualifications. ETag or Last-Modified is retained as upstream evidence, not silently promoted to an API version. No fetched text executes. Each connection resolves at most 32 addresses once, rejects the entire set before dialing if any address is nonpublic, and alternates the validated IPv6/IPv4 candidates with at most two seconds per attempt inside the same retrieval deadline. Failure issues identify a finite transport stage without embedding raw errors or URL query values.

The separate `scv source propose|apply|status|recover` leaves administer a user-owned, noncanonical source selection under explicit `--tops-id`, `--source-id`, and optional `--state-root`. The state path is `symphony/qxctl/scv/sources-v1/<TOPS>/<hash(domain,source_id)>` beneath the owned state root. The path hash is only a filesystem key. Directory/file ownership, private permissions, no-follow descriptor traversal, hard-link rejection, an exclusive per-source lock, atomic replacement, file/directory fsync, whole-document seals and retained predecessor links protect the filesystem transaction. The source store is not an executable registry, source-truth authority, provider account store, or canonical generated knowledge file. As with existing user-scope qxctl state, this boundary protects against other host identities; the owning OS user can independently administer their own files.

`source propose` takes exactly `{operation_id, desired, reason}` (`symphony.qxctl.scv-source-proposal-input.v1`, without an extra protocol key); current state comes from the locked store. It emits the exact C++ `SourcePlan` consumed directly by `source apply --input PLAN`. Applying retains the exact plan, pure C++ transition, selected receipt/executable digests and stable operation identity before requesting SSIAG permission. SSIAG actions are `symphony.scv.source.onboard`, `.relocate`, `.authority-change`, and `.revise`. Their resource is `symphony.scv.source:` plus the hexadecimal SHA-256 of canonical JSON `{tops_id, domain, source_id}`, audience `qxctl`, scope `tops:<UUID>`. This stable source boundary permits an owner to grant the applicable actions to an authenticated human or delegated agent without granting unrelated sources. The operation ID becomes authorization correlation ID; each fresh request has its own request ID and a one-minute requested lifetime. Recovery uses the same exact intent and original installation and requests fresh permission before any unfinished source selection.

The SSIAG response must come from the existing configured, kernel-authenticated Unix endpoint and pass full decision/capability validation; caller-supplied authorization JSON is never accepted. SSIAG returns decisions only after its existing STAV policy-decision submission succeeds. The journal records that actual decision before atomically selecting the C++ successor and marking the operation committed. It does not claim this decision audit is a STAV source-write receipt. Prior authorization decisions survive interrupted recovery. Denial/unavailability leaves the source head unchanged and the exact prepared intent inspectable. `source status --operation-id ID` reports retained outcome; `source recover --operation-id ID` reuses the retained plan. Repeating an already committed exact operation reports its outcome without repeating the write; changing an operation's plan or installation fails. History is explicitly bounded to 128 source operations and 16 MiB per source, with no implicit pruning; a full history fails before the next selection and requires a separately designed archival extension.

`scv projection select|status|recover` composes graph-owner validation with a separate atomic graph selection store. Its state and authorization contract is described in `knowledge/scv/SOURCE-KNOWLEDGE.md`; source location selection and graph publication never share a head or imply one another.

## Non-authorizations
The lifecycle layer also owns the implemented Linux systemd report-only receptor administration. Its descriptor is protected beside the selected profile and binds the exact system state, repository, integration root, unit digest, content-addressed active qxctl executable, bounded predecessor candidates, desired enablement, recovery mode, generation, predecessor digest, and STSC timestamp. Install/update/enable/disable/uninstall are compare-and-swap operations; reconcile repairs unit drift, promotes only an already accepted digest-valid fallback, and resumes durable `retiring` cleanup. The kernel boot UUID supplies the idempotency identity. This grants no lifecycle-apply, component-execution, canonical, remote, hot/warm, login-hook, or hidden-daemon authority.

qxctl is not authorized to write canonical generated artifacts. It may invoke ratified engines to create noncanonical proposals and disposable projections. The Architect-ratified Cobra and Viper libraries and their required cgo-free Go dependencies are authorized only for command grammar and constrained configuration mapping; Python, C bindings, remote configuration backends, in-process vector execution engines, and unrelated third-party dependencies remain prohibited. First-party Symphony libraries remain subordinate to their canonical knowledge vectors.
qxctl is not authorized to accept, store, or print secret values.
qxctl is not authorized to grant host permission, classify callers, silently bypass STAV, or present protocol-integrity requirements as optional safeguards. Any future audit-deferred administrator recovery path requires its own explicit contract, durable local recovery evidence, and later STAV reconciliation.
qxctl and every administrative recovery it coordinates are prohibited from executing inline with, sharing locks with, or adding synchronous dependencies, jitter, or latency to hot or warm paths.

## SCV Immutable Corpus Retention

`qxctl scv corpus acquire|import|recover|inspect|export|diff` extends `ssfv:symphony:qxctl.scv-administration`. Corpus operations default exactly to engine `0.2.0-dev` and explicitly permit `0.3.0-dev`; original SCV leaves retain the `0.1.0-dev` default. The consumer checks the exact installed version, receipt, descriptor operation set and result contract, admitting thirteen operations for `0.1.0-dev`, seventeen for `0.2.0-dev` and twenty for `0.3.0-dev`. It does not switch an old invocation to a new installation or overwrite an immutable old package.

`knowledge/scv/CORPUS.md` owns capture indexing, corpus construction, explicit freshness-qualified member selection and difference semantics. C++ performs those pure operations. The Go adapter owns a private no-follow corpus root, immutable capture/index/snapshot publication, exact input and installation intent, per-member checkpoints, completed-job replay and bounded recovery. A refresh is a new operation with an explicit predecessor snapshot; concurrent jobs retain separate snapshots. No current/latest alias or selected corpus head exists.

Acquire supplies explicit source/locator members and validates them before bounded public HTTPS requests. Import supplies exact captures and claims no live fetch. A completed retry returns its original snapshot without refetching; recovery processes unfinished members under the original intent and installation. Objects are addressed by content digest, verified on reuse and published without overwrite after flushing bytes. A snapshot is exposed only after all referenced objects are durable. Interrupted work can leave reusable unreferenced artifacts, not a falsely complete snapshot.

Inspect and diff verify explicit snapshot identities and their referenced captures/indexes. Export uses exact snapshot/member selection, query time, evidence-age limit and latest-attempt or last-complete policy; it materializes at most sixteen captures within the original engine request limits. A failed refresh does not erase an earlier complete capture or make it fresh. Missing/tampered bytes fail; a metadata digest alone cannot prove replay. Catalog member bounds, job deadline and checkpoint limits are independent of the per-request frame and byte limits.

Immutable evidence storage and job bookkeeping use local filesystem permissions and create no permission grant, source selection, graph selection or audit event. Existing protected source/graph selection continues to require its actual SSIAG authorization and durable intent/audit path. Corpus content never executes as instructions. This adapter supplies bounded acquisition and retention, not a crawler, scheduler, provider account client, automatic semantic expertise, arbitrary graph synthesis or retention-deletion policy.

All six corpus leaves take `--input` JSON and explicit `--corpus-root`/`--tops-id`, plus the exact engine prefix, domain and version. Acquire input is `{operation_id,corpus_id,previous_snapshot_digest,members:[{member_id,source,locator_id}]}`; import replaces each member body with `{member_id,capture}`. Recover input is `{operation_id}`; inspect is `{snapshot_digest}`; export is `{snapshot_digest,member_ids,selection,query_time,max_age_seconds}`; diff is `{before_snapshot_digest,after_snapshot_digest}`. Null predecessor explicitly starts a new corpus lineage. No flag or JSON key implicitly selects the newest snapshot.

The adapter admits at most 128 members and schedules sequential acquisition under a 120-second member scheduling/network budget followed by one bounded capture/index/checkpoint tail. It can retain the failed deadline attempt and leave remaining members unfinished for explicit recovery. Preflight, deep reconstruction and finalization are separately bounded by the ancestry/object caps and per-owner five-second process bound; the member budget is not a total command wall-clock guarantee. A finite job completing does not assert provider completeness. Each engine request retains its own established byte/value/deadline limits.

Known imported observation times or predecessor times later than the local clock fail before intent. After member processing, finalization derives actual snapshot time and checks it against the predecessor and all observations before freezing the snapshot. A clock rollback leaves unfinished recoverable finalization; it cannot justify inventing time or publishing an invalid frozen snapshot. The pure owner operations keep their separate explicit-time contract.

The retained profile creates indexes only from acquired/imported captures through the selected domain; raw indexes and snapshots are not import inputs. A parent can explicitly import in-scope subordinate captures and create its own indexes without installing sibling/provider processes. This is narrower than pure C++ acceptance of already-produced subordinate indexes. Deep verification is bounded to 128 snapshots in the reconstruction chain, including the requested snapshot and 4,096 unique capture/index pairs. A successor reserves one ancestry level and verifies all new pairs before publication. The adapter has no implicit pruning, missing-ancestor shortcut or retention migration; exceeding these limits is an explicit failure.

Corpus acquire/import/recover return `symphony.qxctl.scv-corpus-result.v1`; inspect returns the exact `symphony.scv.corpus.v1` artifact, export returns `symphony.qxctl.scv-corpus-export.v1`, and diff returns `symphony.scv.corpus-diff.v1`. The six input metadata protocols are `symphony.qxctl.scv-corpus-<leaf>-input.v1`, with the exact argument shapes above and no extra protocol field. These are evidence-only or read-only operations with no SSIAG authority mode.

## SCV Provider Interpretation and Connection Evidence

`qxctl scv provider interpret`, `qxctl scv connection evaluate` and `qxctl scv connection reassess` extend `ssfv:symphony:qxctl.scv-administration` through the three exact C++ owner operations in `knowledge/scv/INTERPRETATION.md`. All take bounded `--input` JSON, explicit `--prefix`, domain and version options. New leaves default to `0.3.0-dev`, with no latest substitution. Interpretation is invoke/evidence-only; evaluation and reassessment are validate/evidence-only. These commands need no corpus store or protected source/graph selection and introduce no SSIAG authority mode.

Interpretation input is `{captures,profiles,bindings,selection_policy}`. Explicit reusable profiles map unique retained literal context and values to ordinary knowledge v1 while preserving exact author, profile, capture, extraction and unresolved findings. Evaluation input is `{interpretations,additional_knowledge,query_time,connections}`. The C++ owner replays each wrapper, builds the selected graph and evaluates caller-defined required/optional checks against exact claim subjects, scopes, types, units and current evidence policy. Reassessment input is `{before,after}` with exact prior evaluation artifacts; both are replayed before input axes and affected check findings are compared.

Result protocols are `symphony.scv.provider-interpretation.v1`, `symphony.scv.connection-evaluation.v1` and `symphony.scv.connection-reassessment.v1`. Input metadata protocols use the existing `symphony.scv.<hyphenated-operation>-input.v1` convention and select owner request definitions without embedding a protocol field in payloads. The consumer independently validates seals, exact request/result binding, the full selected claim inventory and policy, extraction evidence, comparison/aggregation and reassessment evidence. Its finite mechanical re-extraction checks literal/delimited tokens, unique context, admitted media and failed/partial dispositions against exact generated typed values, the complete required context/token anchor set and extraction findings. Scalar types and canonical numeric representations are validated before comparison. It does not become the semantic profile owner or replace C++ replay with a digest check.

Supported extraction is a declared finite mapping, not automated comprehension of all native provider documentation. A satisfied connection finding establishes only the selected documented checks; live identity, account permissions, route, payload contract, installed adapter and runtime performance remain explicitly separate requirements. Missing or expired evidence remains unresolved, and recommendations or user assertions retain their conditional character. Endpoint labels and required/optional choices belong to the caller. No provider action, graph selection, remediation, source instruction execution, profile publication or durable canonical mutation occurs.

## SCV Agent Operating Workflow

`knowledge/scv/AGENT-WORKFLOWS.md` governs exact installed schema discovery, native profile preparation, artifact retention, corpus metadata query and recoverable qxctl runs. `ERRORS.md` defines the additive SCV JSON error contract. New wrapper operations retain exact engine invocation and consumer validation; no source checkout, provider account, model call or hidden background loop is required.

Provider coverage administration and additive provider/coverage artifact retention follow `knowledge/scv/COVERAGE.md`, preserving exact source selection, declared gaps and owner replay.

SCV `.6` adds `scv provider pack prepare|evaluate`, `scv composition explore|reassess` and `scv interface show`. The new leaves select exact `.6` by default; existing leaf defaults and `.1`–`.5` installations retain their admitted contracts. Provider packs, detached fixtures, requirements, recipes and counterfactual changes are caller-authored inputs. qxctl validates installed native results and can retain all four new owner artifact kinds through `scv artifact import|show|list`; it does not choose provider policy or execute a recipe. Read `knowledge/scv/PROVIDER-PACKS.md`, `COMPOSITION.md` and `OWNER-INTERFACE.md` for the exact semantics and finite bounds.

## Maintained SCV composition workflow (`0.7.0-dev`)

`qxctl scv composition workflow run|status|recover` coordinates retained provider-package evaluation, finite composition and optional reassessment. The exact request and result contract is `knowledge/scv/COMPOSITION-WORKFLOWS.md`, discoverable in the selected `.7` installation. Run/recovery selects an exact `.7` composition owner; each package stage retains and invokes the original pack owner, including preserved `.6` installations. Status requires only the explicit workflow root, TOPS and operation ID and reports sealed-checkpoint inspection without native replay.

The new journal pins exact caller requirements, policies, query time, record references, root/TOPS and installation. Completed stages retain immutable owner-validated records; retries and recovery reconstruct their exact inputs before replay. This adds no native C++ operation, implicit source refresh, provider choice, graph selection or obligation execution. Completed workflow progress is separate from failed fixtures, unresolved findings and unimplemented recipes. Earlier command defaults and workflow protocols remain unchanged.

## Precise Obligation Follow-up

Exact `0.8.0-dev` exposes 28 native operations, including replayed obligation inventory and subsequent-evidence comparison under `knowledge/scv/OBLIGATIONS.md`. qxctl exposes direct operations and immutable original-owner relationship retention/show. Native check state remains separate from supplied reference provenance, causal attribution and runtime verification. Ten owner companions and 26 schemas expose 97 catalog protocols. Earlier exact packages, protocols and command defaults remain preserved.
