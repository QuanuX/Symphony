# Knowledge Session Coordinator Specification

## Status

Implemented reconciliation, authenticated-session, and report-only lifecycle-planning development slices, version `0.1.0-dev`. Not a published release and not a canonical mutation manager.

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

The result returns the descriptor, `authenticated_session_foundation` readiness, implemented reconciliation and authenticated-session capability declarations, and explicit false values for canonical apply and Maestro docking.

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

## Report-Only Lifecycle Planning

`lifecycle_plan` accepts the exact `symphony.knowledge.lifecycle-plan-command.v1` payload. The caller supplies complete digest-bound desired-state and observation documents, an optional prior applied-state digest, and an explicit declaration of readable process, desired, observation, plan, applied-state, and receipt versions plus named capabilities. The coordinator performs no filesystem discovery and reads no profile, receipt, registry, lifecycle journal, or Maestro state while handling this operation.

Parsing closes every governed object, bounds components to 4,096, packages and dependencies to 256 per component, capabilities to 128, and extensions to 64. It validates normalized document, platform, component, and extension digests; safe absolute and relative paths; exact component identities; selected receipt integrity; receipt v1/v2 protocol identity; and explicit receptor identity for every docked observation. Unknown critical extensions, unknown packages, ambiguous selected state, identity drift, invalid entry-point evidence, and unsupported receipt protocols fail closed or produce explicit fatal blockers without actions.

Compatibility is capability-based rather than release-order based. A v1-only caller remains fully operational when all supplied evidence uses receipt v1 and it advertises the v1 adapter. Receipt v2 becomes required only when desired or observed evidence actually contains v2. Missing common protocol, schema-reader, receipt-reader, or planner capabilities returns a compatibility-blocked plan with no actions; it never silently downgrades or infers compatibility from semantic versions.

The planner emits a deterministic dependency-ready-set graph. Action IDs derive from component/action semantics, exact pre-state, target-state digest, target receptor, artifacts, evidence, and prerequisite semantics—not input array position. Unsatisfied critical dependencies block only their dependent actions; unsatisfied noncritical dependencies produce explicit non-blocking advisories. A critical dependency that contradicts the target's desired disposition remains blocked instead of being misrepresented as a healable ordering edge. Missing required component capabilities are localized compatibility blockers. Strongly connected critical-dependency cycles are isolated while unrelated acyclic actions remain eligible. Repeating the operation with changed verified observations recomputes readiness, permitting a previously blocked component to heal without changing the fixed safety-phase order.

Package selection while active or docked is sequenced as `undock`, `deactivate`, `select`, restore desired activation, and `dock`. Receptor replacement is sequenced as exact old-receptor undock followed by exact desired-receptor dock. Dock actions carry a required `target_receptor_id`; every action carries a digest-bound target state and explicit inverse identity when an inverse exists. Desired upgrade and rollback are equally ordinary forward convergence toward the exact selected receipt; “newest” is never inferred.

The result always declares `apply_authorized: false`. It is disposable noncanonical planning evidence: no lock is acquired, no boot journal or applied state is written, no authorization is requested by this process, no package is installed or removed, no component is activated, and no receptor is contacted. qxctl now owns configured-root collection, fresh SSIAG authorization, exact invocation, and result validation. Persistence, per-action authorization/compare-and-swap, execution, verification, and audit remain separate gates.

The observation document digest includes its normalized collection timestamp and therefore identifies exact evidence. The coordinator separately derives stable inventory from the validated observation with `observed_at` and `observation_digest` removed. Transaction, observation-key, and semantic action identities use that stable inventory digest, so repeated collection of unchanged state does not create a false transaction while real inventory, binding, platform, capability, or provider-availability changes still replan.

## Descriptor Truth

`inspect`, `check`, reconciliation `compatibility|begin|status|checkpoint|close|recover`, authenticated-session `session_begin|session_status|session_checkpoint|session_close|session_recover`, and report-only `lifecycle_plan` are implemented. `apply` is disabled. The descriptor declares user-scope process invocation, C++26, freezing-path placement, `installed_undocked`, no default receptor, and no network listener.

## Install and Uninstall

Installation uses module-and-version-specific paths and creates no active alias. The receipt uses `prefix_mode: installation_prefix`, lists all owned relative files, and carries no host-specific secret or timestamp. qxctl may bind the exact validated receipt and executable digests in its separate user-default registry; that does not alter this receipt's inactive-undocked state. The generated uninstall script removes those files only and refuses directory removal.

## Non-Authorization

This implementation does not authenticate the operating-system caller, decide permission, mutate canonical repository content, run a watcher, install a hook, discover lifecycle state, administer desired profiles, persist or apply a lifecycle plan, invoke a vector engine, directly call SSIAG/STAV, activate an install receipt, or dock with Maestro. SSIAG authenticates and decides; qxctl obtains and validates exact lifecycle permission before invoking this authority-free report operation and transports exact safe evidence for implemented protected session operations; the coordinator validates session evidence and mutates only protected noncanonical reconciliation or authenticated-session journals. `lifecycle_plan` performs no mutation.
