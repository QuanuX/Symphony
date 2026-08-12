# Knowledge Session Coordinator Specification

## Status

Implemented reconciliation, authenticated-session, report-only lifecycle planning/journaling, and separately authorized apply-capable lifecycle coordination development slices, version `0.1.0-dev`. Not a published release, host action executor, or canonical mutation manager.

## Process Contract

The executable implements the exact common request/response envelope under `knowledge/schemas/v1/`. It reads one request from standard input and emits one compact response on standard output. `--help`, `--version`, and `--descriptor` are direct diagnostic modes and do not accept process input.

Stable exit statuses are:

| Status | Meaning |
|---:|---|
| `0` | successful operation |
| `2` | malformed, excessive, or invalid request/argument |
| `3` | protocol, target, or deadline mismatch |
| `4` | unsupported operation or invalid operation payload |
| `5` | bounded engine/path/internal failure |

Every process-mode error still attempts one safe protocol response. If response serialization itself fails, the executable exits `5` without emitting unbounded fallback text.

## `inspect`

Payload: exact empty object.

The result returns the descriptor, `authenticated_session_foundation` readiness, implemented reconciliation and authenticated-session capability declarations, an explicit false value for canonical apply, and an explicit true value for external Maestro docking coordination. The coordinator never writes Maestro state itself.

## `check`

Payload fields:

- `paths`: one to 1,024 unique safe relative regular-file paths;
- `expected_snapshot_digest`: `null` or an exact `sha256:` digest.

The operation roots access at the current working directory, rejects symlinks and special files, reads at most 4 MiB per file, sorts paths, and returns only path, size, content digest, aggregate snapshot digest, and an expected-state comparison. It never returns file content or writes state.

The direct process checks its deadline before and between file-read chunks. qxctl independently terminates the child at the same deadline plus a bounded process-reaping allowance.

## Reconciliation Operations

`compatibility`, `begin`, `status`, `checkpoint`, `close`, and `recover` use the exact command and result schemas under `knowledge/schemas/v1/`. qxctl supplies the protected absolute user state root, its own protocol/version/capability declaration, and operation-specific expected state. The process roots repository reads at its canonical current working directory.

`begin` requires a stable caller operation ID, one to 1,024 unique safe relative regular-file paths, and either `absent` or the exact digest of a closed prior journal. It creates a new open context and initial checkpoint. `checkpoint` and `close` require the exact current journal digest and stable operation ID. Replaying the same completed operation ID is an idempotent no-op; reusing it for different state fails closed. `status` performs no repair. `recover` requires an exact digest during ordinary recovery or an explicit `discover` state when the head cannot be trusted.

Each canonical worktree path maps to one protected context directory under:

```text
<state-root>/symphony/knowledge-session-coordinator/reconciliation/v1/contexts/<worktree-key>/
```

The directory contains `journal.lock`, `journal.0.json`, `journal.1.json`, and `head.json`. Reads and mutations serialize through a nonblocking no-follow lock. State roots and files must be owned by the effective user, managed directories are mode `0700`, managed files are mode `0600`, and group/other-writable or non-regular objects fail closed.

A write synchronizes the inactive journal slot, atomically replaces the head, then synchronizes the containing directory. Recovery validates all available digests and continuity, removes only private coordinator-named atomic-head temporaries left by interrupted commits, and reports every repair action. A unique valid slot one generation ahead of the head with a matching previous digest may be adopted. A damaged or stale head may be rebuilt from a valid slot. Otherwise recovery rolls forward from the highest unique valid slot and records the abandoned head digest. Divergent equally ranked slots, unknown critical extensions, stale expected state, unsupported versions, and ambiguous evidence require administrator review and remain unmodified.

## Compatibility Contract

The coordinator advertises process protocol v1, journal read/write version 1, and named capabilities for dual-slot durability, expected-state compare-and-swap, content snapshots, idempotent operation replay, extension preservation, and recovery. qxctl and the coordinator intersect declared versions and capabilities before reconciliation.

Unknown noncritical extension values and their declared payload digests are preserved exactly as parsed by every write. An implementation that does not recognize a critical extension or stored protocol version preserves the files and blocks both mutation and automated recovery; it cannot fall back to an older slot and call that repair. Newer implementations must continue writing an existing supported format until an explicit stepwise migration commits. No implementation may infer compatibility from semantic-version ordering, silently downgrade, drop unknown state, or perform a lossy conversion.

| Observed combination | Operational behavior |
|---|---|
| older/newer qxctl and coordinator share process v1, journal v1, and every required capability | full read/write operation; executable version order is irrelevant |
| journal v1 is readable but qxctl lacks its write version or any required capability | status/compatibility remain read-only; mutations fail before state changes |
| an engine binding is added, removed, upgraded, or rolled back while a context is open | the next exact-state checkpoint remains valid and records the new binding-registry and engine-inventory digests |
| a synchronized inactive slot is exactly one linked generation ahead of the head | ordinary or discovery recovery adopts it and commits a new recovery checkpoint |
| the head or one slot is damaged while another unique valid slot remains | explicit discovery recovery rebuilds the head and records its disposition |
| an inactive slot is equally ranked, unexpectedly ahead, newer-format, or contains an unknown critical extension | normal writes and automated downgrade stop; all evidence remains in place |
| a future implementation needs a new journal format | it first dual-reads the prior format, preserves that write format while old participants remain bound, then performs a separately contracted idempotent migration |

## Authenticated-Session Operations

`session_begin`, `session_status`, `session_checkpoint`, `session_close`, and `session_recover` use the exact session command and result schemas under `knowledge/schemas/v1/`. Their operation names are distinct from reconciliation operations. qxctl authenticates SSIAG through the configured per-TOPS Unix socket and requests a fresh exact authorization decision before every operation. The coordinator does not call SSIAG itself and does not decide authority.

The command carries the complete safe authorization decision and capability. The coordinator requires an allow decision with `caller_class_used: false` and `canonical_apply: false`; a capability with `transferable: false` and `canonical_apply: false`; an unexpired UTC interval; one exact subject, TOPS ID, operation, opaque canonical-repository-root resource, `qxctl` audience, TOPS scope, request/correlation pair, authority basis, grant ID, policy digest, configuration digest, and binding digest; and exact agreement between decision, capability, and command. The repository resource and binding digest are recomputed before use. Possession of this evidence outside the protected qxctl-to-coordinator invocation is not bearer authority and cannot authorize canonical apply.

One canonical TOPS/subject/repository tuple maps to one protected session directory under:

```text
<state-root>/symphony/knowledge-session-coordinator/sessions/v1/epochs/<session-key>/
```

The directory contains `journal.lock`, `journal.0.json`, `journal.1.json`, and `head.json`. Its ownership, modes, no-follow traversal, write ordering, synchronization, expected-state compare-and-swap, idempotent replay, extension preservation, and unambiguous forward-recovery requirements match the reconciliation durability boundary. Each checkpoint binds an operation fingerprint over the operation, stable ID, expected state, canonical repository, and normalized context-reference set; reuse of an operation ID with different semantics fails closed even though a retry obtains fresh SSIAG evidence. The session journal is separate from every worktree reconciliation journal. It may attach bounded reconciliation-context references; those references do not become authentication or permission evidence.

`session_begin` requires `absent` or the exact closed predecessor journal digest and begins a new linked authority epoch. `session_checkpoint` requires an open unexpired epoch, the exact current journal digest, and unchanged policy/configuration digests. It may attach previously unattached context references. `session_close` records `logout`, `expired`, `policy_changed`, or `config_changed` as applicable. `session_status` is read-only and reports the effective open, closed, or expired state. `session_recover` requires full write compatibility, uses exact state or explicit discovery, and repairs only a unique verifiable forward state: two slots must be identical or form one adjacent digest-linked chain. Exact recovery against an absent stream fails its compare-and-swap; explicit discovery may report an absent no-op. An inactive linked successor requires explicit recovery. Same-generation divergence, an unlinked generation jump, an unknown critical extension, unsupported newer state, invalid evidence, stale expected state, or an expired mutation capability remains visible and unmodified.

Two-way procedural compatibility is determined independently for reconciliation and authenticated-session journals through explicit process protocols, journal read/write versions, and named required capabilities. Upgrade order is not authority or compatibility evidence. A participant lacking a required write capability may inspect supported state but cannot mutate it. A writer preserves supported older formats and all noncritical extension values until an explicit idempotent migration contract is installed. Recovery never deletes or fabricates authority evidence to make versions appear compatible.

The qxctl `knowledge session transition` surface composes these existing operations for one explicit stable host event ID. It is not a new coordinator operation. The coordinator continues to validate and durably commit each status, recover, begin, checkpoint, or close request independently, so qxctl/coordinator upgrade order cannot bypass expected-state, capability, or idempotency checks.

## Persistent SSFV Session Maintenance

`ssfv_maintenance_begin`, `ssfv_maintenance_status`, `ssfv_maintenance_checkpoint`, `ssfv_maintenance_close`, and `ssfv_maintenance_recover` use the exact common command, journal, head, and result schemas. qxctl administers them as `knowledge session features ...`. The stream is keyed independently by canonical TOPS, SSIAG subject, and repository identities and lives beneath:

```text
<state-root>/symphony/knowledge-session-coordinator/ssfv-maintenance/v1/contexts/<context-key>/
```

Each mutation requires a live authenticated-session journal digest, a fresh SSIAG authorization bound to operation, expected state, session, snapshot, and inventory evidence, and a stable operation ID. Begin also requires the exact binding-registry identity, exact receipt/executable identity of the bound SSFV engine, one validated read-only semantic snapshot, and explicit Maestro inventory evidence. The Maestro evidence is either a complete authenticated derived inventory or `not_configured` with a nonempty reason; absence is never silently omitted. Checkpoint and close additionally require an SSFV v2 diff from the immutable baseline to the exact live snapshot.

The journal stores the complete initial semantic snapshot and its `baseline_engine` separately from `current_engine`. A compatible SSFV upgrade or rollback can therefore produce a later checkpoint without reinterpreting or invalidating the baseline. Package version order is never compatibility evidence. qxctl and the coordinator negotiate process protocol, journal read/write version, and named capabilities; status stays read-only where a safe read overlap exists, while mutation requires full overlap. Unknown noncritical extensions survive every write. Unsupported formats, unknown critical extensions, stale compare-and-swap state, ambiguous slots, or a partial Maestro inventory fail closed.

Durability uses a private nonblocking no-follow lock, two synchronized journal slots, an atomic synchronized head, linked generations, exact operation fingerprints, and explicit forward recovery. Status creates no missing state and performs no repair. Recovery may select only one valid adjacent digest-linked chain and publishes a new generation recording the recovered predecessor; it never rewrites the baseline, fabricates feature truth, or discards ambiguous evidence. A result of `review_required` reports semantic drift for caller review. It does not decide feature-worthiness, ratify semantics, create or edit `FEATURES.md`, apply a proposal, or make Maestro inventory canonical.

## Report-Only Lifecycle Planning

`lifecycle_plan` accepts the exact `symphony.knowledge.lifecycle-plan-command.v1` payload. The caller supplies complete digest-bound desired-state and observation documents, an optional prior applied-state digest, and an explicit declaration of readable process, desired, observation, plan, applied-state, and receipt versions plus named capabilities. The coordinator performs no filesystem discovery and reads no profile, receipt, registry, lifecycle journal, or Maestro state while handling this operation.

Parsing closes every governed object, bounds components to 4,096, packages and dependencies to 256 per component, capabilities to 128, and extensions to 64. It validates normalized document, platform, component, and extension digests; safe absolute and relative paths; exact component identities; selected receipt integrity; receipt v1/v2 protocol identity; and explicit receptor identity for every docked observation. Unknown critical extensions, unknown packages, ambiguous selected state, identity drift, invalid entry-point evidence, and unsupported receipt protocols fail closed or produce explicit fatal blockers without actions.

Compatibility is capability-based rather than release-order based. A v1-only caller remains fully operational when all supplied evidence uses receipt v1 and it advertises the v1 adapter. Receipt v2 becomes required only when desired or observed evidence actually contains v2. Missing common protocol, schema-reader, receipt-reader, or planner capabilities returns a compatibility-blocked plan with no actions; it never silently downgrades or infers compatibility from semantic versions.

The planner emits a deterministic dependency-ready-set graph. Action IDs derive from component/action semantics, exact pre-state, target-state digest, target receptor, artifacts, evidence, and prerequisite semantics—not input array position. Unsatisfied critical dependencies block only their dependent actions; unsatisfied noncritical dependencies produce explicit non-blocking advisories. A critical dependency that contradicts the target's desired disposition remains blocked instead of being misrepresented as a healable ordering edge. Missing required component capabilities are localized compatibility blockers. Strongly connected critical-dependency cycles are isolated while unrelated acyclic actions remain eligible. Repeating the operation with changed verified observations recomputes readiness, permitting a previously blocked component to heal without changing the fixed safety-phase order.

Package selection while active or docked is sequenced as `undock`, `deactivate`, `select`, restore desired activation, and `dock`. Receptor replacement is sequenced as exact old-receptor undock followed by exact desired-receptor dock. Dock actions carry a required `target_receptor_id`; every action carries a digest-bound target state and explicit inverse identity when an inverse exists. Desired upgrade and rollback are equally ordinary forward convergence toward the exact selected receipt; “newest” is never inferred.

The result always declares `apply_authorized: false`. It is disposable noncanonical planning evidence: no lock is acquired, no boot journal or applied state is written, no authorization is requested by this process, no package is installed or removed, no component is activated, and no receptor is contacted. qxctl owns configured-root collection, fresh SSIAG authorization, exact invocation, and result validation. Apply uses separate operations, schemas, authorization resources, state directories, and journals; planner output alone never grants mutation.

The observation document digest includes its normalized collection timestamp and therefore identifies exact evidence. The coordinator separately derives stable inventory from the validated observation with `observed_at` and `observation_digest` removed. Transaction, observation-key, and semantic action identities use that stable inventory digest, so repeated collection of unchanged state does not create a false transaction while real inventory, binding, platform, capability, or provider-availability changes still replan.

## Durable Report-Only Lifecycle Journal

`lifecycle_boot`, `lifecycle_boot_status`, and `lifecycle_boot_recover` use the exact lifecycle boot command/result contracts under `knowledge/schemas/v1/`. qxctl supplies a protected state root, exact TOPS/profile identity, one complete SSIAG authorization decision, and explicit journal client versions/capabilities. Boot also supplies its profile digest, complete desired and observed documents, independently computed stable-inventory digest, optional prior-applied-state digest, and planner client declaration. The coordinator recomputes stable inventory and the exact authorization resource before any state access.

Each TOPS/profile pair maps through SHA-256 path keys to one private stream under `<state-root>/symphony/knowledge-session-coordinator/lifecycle/v1/tops/<tops-key>/profiles/<profile-key>/`. A persistent no-follow `0600` lock serializes readers and writers. Journal slots and the atomic head are regular `0600` files owned by the effective user. A mutation synchronizes the inactive slot, synchronizes the directory, atomically replaces and synchronizes the head, and never edits the active slot in place.

Boot requires `absent` or the exact current journal digest. Stable operation-ID replay is idempotent; reuse with different evidence fails closed. A timestamp-only observation refresh has a different document digest but the same stable-inventory digest and therefore performs no journal write. A real profile, desired, stable-inventory, compatibility, provider, binding, receipt, mode, or prior-applied evidence change commits a linked generation and plan revision while retaining the open transaction identity. The journal persists the authorization-bound profile digest, plan identity, blockers, ready-set checkpoints, compatibility and recovery evidence, but action attempts remain empty and both `apply_authorized` and `canonical` remain false.

Status is strictly read-only and does not create an absent stream. Recovery requires full write compatibility and either an exact selected digest or explicit discovery. It accepts only one valid slot, equal identical slots, or one adjacent predecessor/successor digest chain. A damaged head or uniquely linked synchronized successor is repaired through a new forward journal/checkpoint generation. Divergent equal generations, unlinked jumps, unsafe filesystem objects, unknown critical extensions, unsupported formats, stale expected state, or ambiguous evidence remain preserved and fail closed. Unknown noncritical extensions are digest-validated and preserved exactly across writes.

## Apply-Compatible Lifecycle Coordination

`lifecycle_apply_prepare`, `lifecycle_apply_finalize`, `lifecycle_apply_close`, `lifecycle_apply_status`, and `lifecycle_apply_recover` use `symphony.knowledge.lifecycle-apply-command.v1` and `symphony.knowledge.lifecycle-apply-result.v1`. Their mutable stream uses the separately identified `symphony.knowledge.lifecycle-boot-journal.v2` and `symphony.knowledge.lifecycle-boot-head.v2` contracts. It lives beside, and never overwrites, the report-only v1 stream. Every prepare and close names the exact current report-journal digest; the coordinator reopens and validates that source journal and requires matching TOPS, profile, profile digest, desired state, stable inventory, and `apply-compatible` mode.

Every status, prepare, finalize, close, and recovery call carries a fresh exact SSIAG decision for its own operation and digest-bound resource. Status is read-only and reports `apply_authorized: false`; mutation results report that the exact invocation was authorized but remain `canonical: false`. Capability availability in the descriptor is never caller authority.

The apply stream uses its own persistent no-follow lock, dual journal slots, atomic head, file/directory synchronization, exact compare-and-swap journal state, exact compare-and-swap applied state, idempotent semantic operation fingerprints, and explicit read/write capability negotiation. Journal v2 dual-reads v1 only as immutable source evidence and writes only v2. An incompatible or newer critical state remains preserved and read-only. Recovery repairs only one unique adjacent digest-linked chain and preserves any active prepared action.

Prepare recomputes the plan from the complete supplied desired/observed evidence and normally requires the requested action to be in its deterministic ready set. The sole evidence-resolution exception is an `install` blocked only because its exact desired package is absent: one matching staged receipt may satisfy that single blocker only when the action has no graph prerequisite and no additional blocker. A staged artifact can never bypass dependency order. Prepare rejects non-mutating report/preserve/verify actions, records the exact ready install/uninstall/selection/activation/docking action and a `started` attempt, and durably commits before returning. A dock action must bind one exact receptor; an undock action discovers its exact live receptor through exhaustive authenticated qxctl observation. The coordinator never performs the host or Maestro mutation. qxctl executes a reviewed adapter outside the process, then re-observes all configured roots and calls finalize with the same semantic operation identity and bounded execution-evidence digest.

Finalize requires the exact active action, journal, applied state, source report, desired state, profile, stable inventory, artifact set, and operation fingerprint. A committed or already-applied result is accepted only when the new observation proves the action target. The coordinator appends the completed attempt, replans dynamically from that verified evidence, and either preserves localized blockers, prepares the next ready set, or commits applied evidence when the whole profile converges. Blocked and failed attempts remain durable evidence and never masquerade as completion.

Applied state is an immutable content-addressed `applied.<digest>.json` document. It is synchronized before a journal references it; the journal and head are the selection/commit point. An orphan content-addressed file left by an interruption is harmless and is never selected by discovery alone. `lifecycle_apply_close` handles an already-converged report by committing applied evidence without inventing an action. A closed journal requires a non-null applied-state digest and verified close time.

The current external action vocabulary is intentionally narrower than the planner vocabulary. qxctl may install or uninstall only exact immutable receipt-v2 packages from explicit trusted staged roots, update only its protected generic selection/activation state, and commit exact authenticated Maestro presence through an explicitly configured exhaustive receptor set. The coordinator serializes and verifies dock/undock actions but never writes Maestro state or replaces its own running installation. Receipt-v1 mutation, download, arbitrary receipt entry-point execution, live service/process activation, engine-binding rewrite, Maestro engine execution, and canonical writes are absent.

## Descriptor Truth

`inspect`, `check`, reconciliation `compatibility|begin|status|checkpoint|close|recover`, authenticated-session `session_begin|session_status|session_checkpoint|session_close|session_recover`, persistent `ssfv_maintenance_begin|ssfv_maintenance_status|ssfv_maintenance_checkpoint|ssfv_maintenance_close|ssfv_maintenance_recover`, report-only `lifecycle_plan`, durable report-only `lifecycle_boot|lifecycle_boot_status|lifecycle_boot_recover`, and apply coordination `lifecycle_apply_prepare|lifecycle_apply_finalize|lifecycle_apply_close|lifecycle_apply_status|lifecycle_apply_recover` are implemented. Canonical `apply` remains disabled. The descriptor declares user-scope process invocation, C++26, freezing-path placement, `installed_undocked`, no default receptor, and no network listener. Its apply capability flags describe protocol availability only and never caller authority.

## Install and Uninstall

Installation uses module-and-version-specific paths and creates no active alias. The receipt uses `prefix_mode: installation_prefix`, lists all owned relative files, and carries no host-specific secret or timestamp. qxctl may bind the exact validated receipt and executable digests in its separate user-default registry; that does not alter this receipt's inactive-undocked state. The generated uninstall script removes those files only and refuses directory removal.

## Non-Authorization

This implementation does not authenticate the operating-system caller, decide permission, mutate canonical repository content, run a watcher, install a hook, discover receipts, administer desired profiles, execute a host lifecycle action, invoke a vector engine, directly call SSIAG/STAV, activate an install receipt, replace itself, or dock with Maestro. SSIAG authenticates and decides; qxctl obtains exact permission, invokes the selected SSFV and optional Maestro processes separately, transports their validated read-only evidence, and owns the reviewed external adapters. The coordinator independently validates stateful session, SSFV-maintenance, and lifecycle-journal evidence and mutates only protected noncanonical reconciliation, authenticated-session, SSFV-maintenance, lifecycle-journal, or applied-state files. Neither semantic review evidence nor `lifecycle_plan`/`lifecycle_boot` grants apply; only exact separately authorized apply operations advance the v2 coordination stream, and none grants canonical mutation.
