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
```

SSIAG commands require an immutable TOPS UUID through `--tops-id` or `SYMPHONY_SSIAG_TOPS_ID`. They use `SYMPHONY_SSIAG_SOCKET` only as an explicit override; otherwise the selected scope and TOPS ID determine the isolated socket. They never accept or print credential values.

The ratified administrative model separates non-mutating proposal from permission-backed local apply. Implemented knowledge-session authorization uses exact target-host ownership or granted-permission grants and never requests or evaluates caller type. Owner-configurable safeguard administration and canonical apply remain future gates. Canonical knowledge and STAV remain read-only through qxctl. The protected user-scope engine-binding registry, reconciliation journals, and authenticated-session journals are the current mutable administrative surfaces, and qxctl never writes STAV ledger files directly.

Future safeguard administration will let a target-host administrator inspect and change optional governance interlocks through supported qxctl commands, including selecting a direct profile. That future surface does not make parser bounds, path safety, atomic writes, expected-state validation, ledger framing, or secret exclusion optional. No safeguard-management, apply, or audit-deferred recovery command is implemented today.

The four STAV commands use mutually authenticated, TOPS-scoped Unix-socket IPC to the local append authority. `status`, `verify`, and bounded `query` return only classification-authorized read projections; `doctor` composes client-side availability and verification checks. qxctl verifies the configured authority identity before sending application bytes and never opens the ledger file. `qxctl stav append` is intentionally absent.

The SKVI, SCLV, SACV, SODV, and SSFV commands are cold/freezing-path local process operations. qxctl validates the exact inactive-undocked receipt and all package-owned files, requiring the prefix and installed files to be owned by the effective user or root and not writable by group or other. It invokes only the versioned engine path with an empty environment, enforces a hard deadline, and verifies response identity, digest, and safety assertions. Secure local receipt traversal is implemented on Linux and the macOS development path; other native operating systems fail closed rather than substituting a weaker file-open routine. Proposal, diff, verification, recovery, and SSFV baseline input comes from bounded no-follow JSON files. SKVI cannot decide membership. SCLV cannot grant permission, ratify, append, commit, mutate or delete journals, or treat a projection as canonical. SACV cannot decide semantic ownership, create endpoints, publish, generate bindings, or treat compatibility evidence or a projection as canonical. SODV cannot create or move tags, query external providers, declare publication complete, append records, mutate recovery journals, or treat a projection as canonical. SSFV cannot decide feature-worthiness or semantic truth, ratify or apply proposals, persist graphs, or create a feature record.

`knowledge engines` manages one protected user-scope `default` profile. A binding records the exact receipt and executable digests for an inactive-undocked coordinator or vector engine. Registry mutations require the exact prior digest, serialize through a persistent no-follow lock, and commit atomically. Multiple versions may remain installed, but one exact version may be bound per role.

`knowledge reconcile` snapshots and revalidates every binding, invokes only the exact bound coordinator, and records the registry digest plus role-sorted module/engine/version/receipt/executable evidence in each checkpoint. Its capability negotiation supports compatible newer or older executables without inferring safety from version recency: supported formats remain writable, missing write capabilities become read-only, noncritical extensions survive writes, and unknown critical or newer state is preserved without downgrade. Mutations use stable operation IDs and exact prior journal digests. Explicit discovery recovery may repair a corrupt/missing atomic head, adopt one uniquely linked successor left by an interrupted commit, and remove only safe stale head temporaries. Ambiguity remains visible and unmodified. This surface does not itself authenticate a session, invoke vector semantics, write canonical knowledge, or dock with Maestro.

`knowledge session begin|status|checkpoint|close|recover` establishes and maintains a separate protected noncanonical authority-epoch journal. Every call requires `--tops-id`, reads one protected binding snapshot to resolve and revalidate the exact coordinator installation, authenticates the TOPS-scoped SSIAG endpoint, and requests an exact audited authorization decision for that operation and canonical repository resource. qxctl rejects denial, expiry, target/configuration drift, caller-class use, transferability, canonical-apply claims, and malformed capability bindings. The coordinator uses stable operation IDs, exact prior digests, dual-slot durability, linked epochs, and unambiguous forward recovery; `--context-ref` attaches reconciliation contexts without turning them into identity or engine-inventory evidence. This surface does not invoke vector semantics, write canonical knowledge, administer safeguards, activate receipts, or dock with Maestro. `knowledge apply` remains unavailable.

`knowledge session transition` provides an explicit idempotent adapter for a host lifecycle integration. A stable event ID composes freshly authorized status, optional bounded discovery recovery, close, begin, and checkpoint operations for `login`, `refresh`, or `logout`. Retry with the same event ID observes journal operation evidence and does not repeat a completed transition. qxctl installs no PAM module, login-manager hook, watcher, service, or boot unit.

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
```

The selected SSIAG configuration must map the invoking effective UID/GID and contain an exact grant for each requested `symphony.knowledge.session.*` operation, the opaque `symphony.knowledge.repository:<sha256>` resource derived from the canonical repository root, audience `qxctl`, and scope `tops:<tops-id>`. With no exact grant—or if STAV is unavailable—the operation fails closed without coordinator mutation.

Generic module and vector additions, removals, upgrades, rollbacks, and first-boot convergence are contractually specified in `knowledge/LIFECYCLE.md` and the common lifecycle schemas. Binding registry v1 remains fixed to its six existing roles. qxctl now persists protected per-TOPS desired profiles, collects fixed-layout configured-root observations, and invokes the exact bound C++ coordinator for a fresh report-only dependency plan. It preserves unmanaged and unsupported packages, isolates localized blockers, binds exact receptor targets, and never authorizes apply. The stable inventory key excludes observation time, so merely running the report later does not restart an unchanged transaction; real content or compatibility changes cause dynamic replanning.

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
```

Each operation requires a matching exact `symphony.knowledge.lifecycle.*` SSIAG grant and committed STAV decision. Profile updates use the digest returned by `show` or `list` as the next expected state. Plans, applied state, and boot journals are not yet persisted; installation, uninstall, activation, docking, and lifecycle apply remain unavailable.
