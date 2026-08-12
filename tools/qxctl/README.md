# qxctl

Symphony's Go-based administrative spine.

This tool is seeded with `Go 1.26.5` as the deterministic scripted baseline. Its administrative command grammar uses Cobra and its explicitly bound command configuration uses private Viper instances. SSIAG/STAV trust loading remains outside Viper in dedicated cgo-free clients.

**Future Posture:** `qxctl` will migrate to Go 1.27 after general availability and the differential conformance/cross-build gate passes. It does not currently require or use unreleased features, and the migration cannot alter STAV wire bytes or CLI grammar.

## Usage

```bash
# Print help
go run ./cmd/qxctl --help

# Perform local repository checks
go run ./cmd/qxctl doctor

# Verify the first runtime-set module contract surfaces
go run ./cmd/qxctl contracts

# List canonical runtime modules
go run ./cmd/qxctl modules

# Verify contract shape for all modules
go run ./cmd/qxctl modules check

# Inspect a specific runtime module
go run ./cmd/qxctl module inspect <module-name>

# Verify contract shape for a specific module
go run ./cmd/qxctl module check <module-name>

# Extract contract metadata for all modules
go run ./cmd/qxctl modules metadata [--json]

# Extract contract metadata for a specific module
go run ./cmd/qxctl module metadata <module-name> [--json]

# Emit deterministic runtime inventory snapshot
go run ./cmd/qxctl inventory [--json]

# Emit deterministic runtime inventory SHA-256 digest
go run ./cmd/qxctl inventory digest [--json]

# Report consolidated administrative status
go run ./cmd/qxctl status [--json]

# Read safe local SSIAG status
go run ./cmd/qxctl ssiag status --tops-id UUID [--json] [--scope user|system]

# List safe provider metadata
go run ./cmd/qxctl ssiag providers --tops-id UUID [--json] [--scope user|system]

# Verify local SSIAG availability
go run ./cmd/qxctl ssiag doctor --tops-id UUID [--scope user|system]

# Generate deterministic proposal-only lifecycle grants for one configured subject
go run ./cmd/qxctl ssiag grants lifecycle --tops-id UUID --subject-id ID [--profile-id ID] [--authority-basis host_owner|granted_permission] --json

go run ./cmd/qxctl ssiag policy status --tops-id UUID [--scope user|system] [--json]

go run ./cmd/qxctl ssiag policy propose --tops-id UUID --operation-id ID --expected-policy-digest sha256:... (--input policy.json|--reset) [--authority-basis host_owner|granted_permission] [--ttl 5m] --json

go run ./cmd/qxctl ssiag policy apply --tops-id UUID --input proposal.json [--scope user|system] [--json]

go run ./cmd/qxctl ssiag policy recover --tops-id UUID --operation-id ID (--expected-attempt-digest sha256:...|--discover) [--scope user|system] [--json]

# Query the authenticated local STAV append authority
go run ./cmd/qxctl stav status --tops-id UUID [--scope user|system] [--json]
go run ./cmd/qxctl stav verify --tops-id UUID [--scope user|system] [--json]
go run ./cmd/qxctl stav query --tops-id UUID [--scope user|system] [--after-sequence N] [--through-sequence N] [--from-time UTC] [--through-time UTC] [--event-class ID]... [--outcome VALUE]... [--correlation-id UUID] [--request-id UUID] [--limit 1..1000] [--json]
go run ./cmd/qxctl stav doctor --tops-id UUID [--scope user|system]

# Manage the protected user-default exact engine bindings
go run ./cmd/qxctl knowledge engines list [--state-root /chosen/state/root] [--json]
go run ./cmd/qxctl knowledge engines inspect skvi [--json]
go run ./cmd/qxctl knowledge engines bind skvi --prefix /chosen/prefix [--version 0.1.0-dev] --expected-registry-digest absent [--json]
go run ./cmd/qxctl knowledge engines doctor [--json]
go run ./cmd/qxctl knowledge engines unbind skvi --expected-registry-digest sha256:... [--json]

# Administer durable worktree reconciliation through the exact bound coordinator
go run ./cmd/qxctl knowledge reconcile compatibility [--repo /repository] [--json]
go run ./cmd/qxctl knowledge reconcile begin --operation-id ID --expected-journal-digest absent --path INTENT.md --path knowledge/INTENT.md [--json]
go run ./cmd/qxctl knowledge reconcile status [--repo /repository] [--json]
go run ./cmd/qxctl knowledge reconcile checkpoint --operation-id ID --expected-journal-digest sha256:... [--json]
go run ./cmd/qxctl knowledge reconcile close --operation-id ID --expected-journal-digest sha256:... [--json]
go run ./cmd/qxctl knowledge reconcile recover --operation-id ID --expected-journal-digest sha256:... [--json]
go run ./cmd/qxctl knowledge reconcile recover --operation-id ID --discover [--json]

# Invoke an exact independently installed SKVI engine
go run ./cmd/qxctl skvi inspect --prefix /chosen/prefix [--version 0.1.0-dev] [--json]
go run ./cmd/qxctl skvi check --prefix /chosen/prefix [--expected-index-digest sha256:...] [--json]
go run ./cmd/qxctl skvi propose --prefix /chosen/prefix --input proposal-input.json [--json]
go run ./cmd/qxctl skvi project --prefix /chosen/prefix [--json]

# Invoke an exact independently installed SCLV engine
go run ./cmd/qxctl sclv inspect --prefix /chosen/prefix [--version 0.1.0-dev] [--json]
go run ./cmd/qxctl sclv check --prefix /chosen/prefix [--expected-ledger-digest sha256:...] [--json]
go run ./cmd/qxctl sclv propose --prefix /chosen/prefix --input proposal-input.json [--json]
go run ./cmd/qxctl sclv recover --prefix /chosen/prefix --input recovery-input.json [--json]
go run ./cmd/qxctl sclv project --prefix /chosen/prefix [--json]

# Invoke an exact independently installed SACV engine
go run ./cmd/qxctl sacv inspect --prefix /chosen/prefix [--version 0.1.0-dev] [--json]
go run ./cmd/qxctl sacv check --prefix /chosen/prefix [--expected-registry-digest sha256:...] [--json]
go run ./cmd/qxctl sacv diff --prefix /chosen/prefix --input diff-input.json [--json]
go run ./cmd/qxctl sacv propose --prefix /chosen/prefix --input proposal-input.json [--json]
go run ./cmd/qxctl sacv project --prefix /chosen/prefix [--json]

# Invoke an exact independently installed SODV engine
go run ./cmd/qxctl sodv inspect --prefix /chosen/prefix [--version 0.1.0-dev] [--json]
go run ./cmd/qxctl sodv check --prefix /chosen/prefix [--expected-ledger-digest sha256:...] [--json]
go run ./cmd/qxctl sodv verify --prefix /chosen/prefix --input observed-state.json [--json]
go run ./cmd/qxctl sodv propose --prefix /chosen/prefix --input proposal-input.json [--json]
go run ./cmd/qxctl sodv recover --prefix /chosen/prefix --input recovery-input.json [--json]
go run ./cmd/qxctl sodv project --prefix /chosen/prefix [--json]

# Invoke an exact independently installed SSFV engine
go run ./cmd/qxctl ssfv inspect --prefix /chosen/prefix [--version 0.1.0-dev] [--json]
go run ./cmd/qxctl ssfv check --prefix /chosen/prefix [--expected-namespace-digest sha256:...] [--expected-registry-digest sha256:...] [--baseline semantic-snapshot.json] [--freshness disabled|report|require] [--json]
go run ./cmd/qxctl ssfv diff --prefix /chosen/prefix --input diff-input.json [--json]
go run ./cmd/qxctl ssfv propose --prefix /chosen/prefix --input proposal-input.json [--json]
go run ./cmd/qxctl ssfv graph --prefix /chosen/prefix [--json]

# Validate through one exact independently installed validator
go run ./cmd/qxctl validate profile set --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --profile-id default --warning record --historical summary --new full --expected-policy-digest absent
go run ./cmd/qxctl validate baseline create --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --prefix /chosen/prefix --baseline-id default --expected-baseline-digest absent --repo /absolute/path/to/Symphony
go run ./cmd/qxctl validate scan --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --prefix /chosen/prefix --profile-id default --baseline-id default --repo /absolute/path/to/Symphony [--json]
go run ./cmd/qxctl validate debug --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --prefix /chosen/prefix --baseline-id default --rule sclv.affected_surface.unindexed
```

SSIAG commands require an immutable TOPS UUID through `--tops-id` or `SYMPHONY_SSIAG_TOPS_ID`. They use `SYMPHONY_SSIAG_SOCKET` only as an explicit override; otherwise the selected scope and TOPS ID determine the isolated socket. They never accept or print credential values.

The ratified administrative model separates non-mutating proposal from permission-backed local apply. Implemented knowledge-session and lifecycle authorization uses exact target-host ownership or granted-permission grants and never requests or evaluates caller type. Protected validator warning-profile and baseline administration is implemented; broader safeguards and canonical knowledge apply remain future gates. Canonical knowledge and STAV remain read-only through qxctl, and qxctl never writes STAV ledger files directly.

Validator warning sensitivity is administered without changing the detector: `record` retains a warning, `review` signals new matching warnings, and `require` fails on new matching warnings. Presentation can be `full`, `summary`, or `count`; baselines classify new, unchanged, and resolved occurrences. Violations, parser bounds, path safety, atomic writes, expected-state validation, ledger framing, and secret exclusion are never optional. Broader safeguard management, canonical apply, and audit-deferred recovery remain unavailable.

Validation state is protected beneath `<state-root>/symphony/<tops-id>/qxctl/validation/`. Mutations require `absent` or the exact current digest and use caller-neutral owner-only, no-follow, synchronized atomic replacement. A baseline acknowledges observed evidence; it does not ratify, delete, or resolve a warning. A repository-identity or validator-version mismatch fails closed.

The four STAV commands use mutually authenticated, TOPS-scoped Unix-socket IPC to the local append authority. `status`, `verify`, and bounded `query` return only classification-authorized read projections; `doctor` composes client-side availability and verification checks. qxctl verifies the configured authority identity before sending application bytes and never opens the ledger file. `qxctl stav append` is intentionally absent.

The SKVI, SCLV, SACV, SODV, and SSFV commands are cold/freezing-path local process operations. qxctl validates the exact inactive-undocked receipt and all package-owned files, requiring the prefix and installed files to be owned by the effective user or root and not writable by group or other. It invokes only the versioned engine path with an empty environment, enforces a hard deadline, and verifies response identity, digest, and safety assertions. Secure local receipt traversal is implemented on Linux and the macOS development path; other native operating systems fail closed rather than substituting a weaker file-open routine. Proposal, diff, verification, recovery, and SSFV baseline input comes from bounded no-follow JSON files. SKVI cannot decide membership. SCLV cannot grant permission, ratify, append, commit, mutate or delete journals, or treat a projection as canonical. SACV cannot decide semantic ownership, create endpoints, publish, generate bindings, or treat compatibility evidence or a projection as canonical. SODV cannot create or move tags, query external providers, declare publication complete, append records, mutate recovery journals, or treat a projection as canonical. SSFV cannot decide feature-worthiness or semantic truth, ratify or apply proposals, persist graphs, or create a feature record.

`knowledge engines` manages one protected user-scope `default` profile. A binding records the exact receipt and executable digests for an inactive-undocked coordinator or vector engine. Registry mutations require the exact prior digest, serialize through a persistent no-follow lock, and commit atomically. Multiple versions may remain installed, but one exact version may be bound per role.

`knowledge reconcile` snapshots and revalidates every binding, invokes only the exact bound coordinator, and records the registry digest plus role-sorted module/engine/version/receipt/executable evidence in each checkpoint. Its capability negotiation supports compatible newer or older executables without inferring safety from version recency: supported formats remain writable, missing write capabilities become read-only, noncritical extensions survive writes, and unknown critical or newer state is preserved without downgrade. Mutations use stable operation IDs and exact prior journal digests. Explicit discovery recovery may repair a corrupt/missing atomic head, adopt one uniquely linked successor left by an interrupted commit, and remove only safe stale head temporaries. Ambiguity remains visible and unmodified. This surface does not itself authenticate a session, invoke vector semantics, write canonical knowledge, or dock with Maestro.

`knowledge session begin|status|checkpoint|close|recover` establishes and maintains a separate protected noncanonical authority-epoch journal. Every call requires `--tops-id`, reads one protected binding snapshot to resolve and revalidate the exact coordinator installation, authenticates the TOPS-scoped SSIAG endpoint, and requests an exact audited authorization decision for that operation and canonical repository resource. qxctl rejects denial, expiry, target/configuration drift, caller-class use, transferability, canonical-apply claims, and malformed capability bindings. The coordinator uses stable operation IDs, exact prior digests, dual-slot durability, linked epochs, and unambiguous forward recovery; `--context-ref` attaches reconciliation contexts without turning them into identity or engine-inventory evidence. This surface does not invoke vector semantics, write canonical knowledge, administer safeguards, activate receipts, or dock with Maestro. Canonical `knowledge apply` remains unavailable.

`knowledge session transition` provides an explicit idempotent adapter for a host lifecycle integration. A stable event ID composes freshly authorized status, optional bounded discovery recovery, close, begin, and checkpoint operations for `login`, `refresh`, or `logout`. Retry with the same event ID observes journal operation evidence and does not repeat a completed transition. qxctl installs no PAM module, login-manager hook, watcher, service, or boot unit.

`knowledge session features begin|status|checkpoint|close|recover` administers a separate persistent SSFV maintenance stream during an open authenticated session. Begin captures the exact bound SSFV engine and a content-addressed semantic baseline. Checkpoint and close re-run the exact installation, compute an SSFV v2 diff against that immutable baseline, and record `current` or `review_required` without deciding semantic truth. `--maestro-prefix` adds a complete authenticated derived receptor inventory; omitting it records explicit `not_configured` evidence. Every mutation uses a stable operation ID, exact journal compare-and-swap, fresh SSIAG authorization, dual-slot durability, and evidence-preserving recovery. This circuit never creates or edits `FEATURES.md` and never applies an SSFV proposal.

`maestro inventory` returns the sorted derived inventory for every receptor in one TOPS. Its stable inventory digest excludes the outer observation timestamp. If any receptor stream is busy, unsafe, damaged, or ambiguous, the complete request fails rather than presenting a partial deployment as complete.

```bash
go run ./cmd/qxctl knowledge session begin \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 \
  --operation-id session-start-001 \
  --expected-journal-digest absent \
  --repo /absolute/path/to/Symphony

go run ./cmd/qxctl knowledge session status \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 \
  --repo /absolute/path/to/Symphony --json

go run ./cmd/qxctl knowledge session transition \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 \
  --event login \
  --event-id host-login-0001 \
  --repo /absolute/path/to/Symphony --json

go run ./cmd/qxctl knowledge session features begin \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 \
  --operation-id features-begin-0001 \
  --expected-journal-digest absent \
  --repo /absolute/path/to/Symphony --json

go run ./cmd/qxctl knowledge session features checkpoint \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 \
  --operation-id features-checkpoint-0001 \
  --expected-journal-digest sha256:... \
  --maestro-prefix /absolute/maestro/prefix \
  --repo /absolute/path/to/Symphony --json

go run ./cmd/qxctl maestro inventory \
  --prefix /absolute/maestro/prefix \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --json
```

The selected SSIAG configuration must map the invoking effective UID/GID and contain an exact grant for each requested `symphony.knowledge.session.*`, `symphony.knowledge.ssfv_maintenance_*`, or `symphony.maestro.receptor-inventory.read` operation and its exact opaque resource, audience `qxctl`, and scope `tops:<tops-id>`. With no exact grant—or if STAV is unavailable—the operation fails closed without coordinator mutation.

Generic module and vector additions, removals, upgrades, rollbacks, and first-boot convergence are contractually specified in `knowledge/LIFECYCLE.md` and the common lifecycle schemas. Binding registry v1 remains fixed to its six existing roles. qxctl persists protected per-TOPS desired profiles, collects fixed-layout configured-root observations, and invokes the exact bound C++ coordinator for disposable reports, protected durable boot journals, or the separate apply-capable v2 journal. It preserves unmanaged and unsupported packages, isolates localized blockers, binds exact receptor targets, and requires fresh exact SSIAG permission at every apply phase. The stable inventory key excludes observation time, so a later timestamp-only scan does not advance the journal; real content or compatibility changes create a linked plan revision.

```bash
# First profile generation; profile.json uses lifecycle-profile-input.v1.
go run ./cmd/qxctl knowledge lifecycle profile set \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 \
  --input profile.json --expected-profile-digest absent --json

# Disposable evidence and a fresh non-mutating dependency plan.
go run ./cmd/qxctl knowledge lifecycle observe \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --profile-id default --json
go run ./cmd/qxctl knowledge lifecycle report \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --profile-id default --json

# Durable report-only first-boot evidence. Reuse the returned digest for the next mutation.
go run ./cmd/qxctl knowledge lifecycle boot \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --profile-id default \
  --operation-id node-boot-0001 --expected-journal-digest absent --json
go run ./cmd/qxctl knowledge lifecycle status \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --profile-id default --json
go run ./cmd/qxctl knowledge lifecycle recover \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --profile-id default \
  --operation-id node-boot-recover-0001 --discover --json

# Explicit apply-compatible convergence. The profile must select apply-compatible mode.
# The source digest is the exact report journal returned by boot; both other values
# are compare-and-swap anchors returned by apply-status (or absent initially).
go run ./cmd/qxctl knowledge lifecycle apply \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --profile-id default \
  --operation-id node-apply-0001 --source-journal-digest sha256:... \
  --expected-apply-journal-digest absent --expected-applied-state-digest absent \
  --source-root /absolute/trusted/staged/package/root \
  --maestro-prefix /opt/symphony --maestro-receptor-id maestro-primary \
  --maestro-receptor-id maestro-previous --json
go run ./cmd/qxctl knowledge lifecycle apply-status \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --profile-id default --json
go run ./cmd/qxctl knowledge lifecycle apply-recover \
  --tops-id 018f0c3a-7b2d-7e11-8c12-0242ac120002 --profile-id default \
  --operation-id node-apply-recover-0001 --discover --json
```

Each operation requires a matching exact `symphony.knowledge.lifecycle.*` SSIAG grant and committed STAV decision. The grant resource is stable for the exact TOPS/profile pair; operation names remain exact, while artifact and state digests remain mandatory independently verified lifecycle evidence. `qxctl ssiag grants lifecycle --tops-id UUID --subject-id ID [--profile-id ID] --json` emits the deterministic 18-grant input for a configured caller-neutral subject. The grant plan itself remains proposal-only. The separate `qxctl ssiag policy propose|apply|recover` circuit can now install those or other exact grants into SSIAG's operational overlay when authorized by target-host ownership or exact current policy; it never edits canonical knowledge or enrolled config. Profile updates use the digest returned by `show` or `list` as the next expected state. `boot` persists report-only plan/checkpoint identity through an exact expected journal digest; `status` is read-only and `recover` repairs only uniquely linked v1 evidence. `apply` uses that exact report journal as immutable source authority, prepares each action durably before execution, reconciles root-local ownership claims, re-observes after it, and commits content-addressed applied evidence only after full convergence. A trusted staged receipt resolves only the isolated package-absence blocker for an install with no graph prerequisite or additional blocker; staged availability cannot bypass dependency order. `apply-status` is read-only and `apply-recover` repairs only uniquely linked v2 evidence while preserving an active action. Generic package mutation is restricted to exact receipt-v2 packages in explicit trusted staged roots. The six established engine roles also adapt selection to binding-registry v1 without extending its role set. A coordinator replacement must be installed side by side and must reproduce the exact prepared journal through its own validated process before the binding changes. The old coordinator finalizes the no-crash path; durable active-action replay lets the selected coordinator self-heal a crash after the switch. Old versions remain installed by default for rollback or bespoke multi-version topology. Exact Maestro docking presence requires `--maestro-prefix` plus one or more repeatable `--maestro-receptor-id` values, a compatible v2 vector-engine receipt, and fresh `symphony.maestro.docking.*` SSIAG capability evidence. The supplied receptor list is exhaustive for that lifecycle invocation: it must include every receptor that could currently hold a component plus every desired target receptor. This permits safe old-receptor discovery, inverse undock, and forward docking when upgrades or receptor changes occur out of order; ambiguity fails closed instead of fabricating an undocked observation. Direct Maestro administration exposes inspect, status, bounded recovery, and complete read-only derived inventory; lifecycle apply owns dock/undock. Package download, receipt-v1 mutation, arbitrary entry-point execution, live service or Maestro engine activation, in-place coordinator replacement, unconstrained rebinding, canonical knowledge apply, and host boot-hook installation remain unavailable.

Package mutation and ownership mutation share one persistent lock at the exact target root. Profile mutation holds the exclusive profile lock while checking claims, while ownership reconciliation and action execution hold a shared profile lease before acquiring the root lock. A new empty root is enforced immediately; a pre-existing root is conservatively adopted with legacy-preserve claims. Before the registry commit, qxctl publishes a static receipt-layout compatibility fence: ownership-aware clients consume it, while older lifecycle clients preserve it as unsupported evidence and produce their existing global critical blocker. `knowledge lifecycle ownership status|reconcile|adopt|release` provides explicit administration. Before `adopt`, the administrator drains any old mutation already prepared from a pre-fence observation; no file marker can retroactively cancel such an action. A late completion is exposed by the next observation as missing retained state or new legacy evidence. Profile updates cannot abandon a claimed component/root mapping: keep it mapped with desired absence, converge, and only then remove or relocate it. Profile removal is likewise refused while that profile retains or retires a receipt, and uninstall is refused while any retained or legacy claim remains.
