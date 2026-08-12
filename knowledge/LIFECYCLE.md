# Symphony Cross-Vector Lifecycle and First-Boot Plan

## Status

This document records the Architect-ratified topology for explicit authenticated session transitions and the cross-vector desired-state lifecycle. The session-transition surface described below is implemented by qxctl over the existing SSIAG-authorized coordinator operations. Canonical profile-input, protected profile, desired-state, observation, plan-command, dependency-driven plan, applied-state, runtime-state, report-journal v1, apply-journal v2, host-integration, command/result, immutable receipt-v2, and Maestro receptor/presence schemas are present. qxctl implements SSIAG-authorized desired-profile administration, fixed-layout configured-root observation, fresh report invocation, durable report-journal administration, a Linux systemd report-only boot receptor, and an explicit `apply-compatible` convergence loop through one exact bound C++ coordinator. The coordinator implements deterministic dependency planning, protected side-by-side report and apply journals, exact compare-and-swap progression, timestamp-stable no-op detection, linked replanning, action-attempt serialization, content-addressed applied-state commitment, and evidence-based recovery. Generic staged receipt-v2 install/uninstall, protected selection/activation, established-role binding switches, verified coordinator handoff, and authenticated Maestro docking-presence convergence are implemented on Linux and the macOS development path. The C++ foundation, coordinator, five vector engines, Maestro, and Symphony Validator now generate immutable receipt-v2 packages at install time; qxctl continues to read legacy receipt v1 for compatibility but never mutates it. Downloads, arbitrary entry-point execution, live service activation, and engine execution by Maestro remain unavailable.

## Purpose

Symphony installations are rolling and modular. A valid installation may add, remove, upgrade, roll back, dock, undock, or retain multiple versions of independently installable modules and vector engines. Upgrade order is not assumed. The lifecycle system must therefore converge from observed evidence instead of treating one release image, one fixed module count, or newest-version selection as truth.

The lifecycle boundary has two distinct jobs:

1. converge an authenticated knowledge authority epoch from explicit host login, refresh, and logout events;
2. compare an administrator-owned desired installation profile with content-addressed observed installation state at first boot after a meaningful system change.

Neither job runs on a hot or warm path. Neither job mutates canonical knowledge.

## Ownership

- `knowledge/` owns cross-vector desired-state, observation, plan, journal, compatibility, and first-boot protocol truth.
- individual vectors own their semantic contracts and vector-specific lifecycle consequences;
- immutable package receipts own installed-file identity and removal boundaries;
- qxctl owns Cobra administration, explicit host-event entry points, protected profile and runtime-state selection, evidence collection, SSIAG exchange, bounded coordinator invocation, and the reviewed external package/runtime action adapters;
- the C++ knowledge-session coordinator owns freezing-path transition planning, separate report/apply lifecycle journals, action-attempt serialization, applied-state commit selection, compare-and-swap progression, compatibility negotiation, and evidence-based recovery;
- SSIAG decides whether the kernel-derived caller may perform each lifecycle operation and emits safe capability evidence only after the corresponding STAV decision event commits;
- STAV records safe security-relevant decision outcomes and never becomes the lifecycle state store;
- the independently installed C++ Maestro presence authority persists per-TOPS/per-receptor docking presence, but does not own vector semantics, desired-state policy, package truth, or engine execution;

## Namespace Allocation Evidence

Before the receptor contracts or implementation were authored, repository, QuanuX organization, and public namespace searches on 2026-08-12 found no conflicting use of `symphony.knowledge.lifecycle-host-integration.v1`, its result/boot-result family, `qxctl knowledge lifecycle host`, or the `symphony-qxctl-lifecycle@...service` unit prefix. Those names are allocated here to the cross-vector lifecycle contract and qxctl implementation. This availability proof does not claim a new independently published module, package, binary, tag, or external namespace registration.

## Explicit Session Transitions

`qxctl knowledge session transition` accepts one explicit host event and a stable event identifier:

- `login` closes any stale open epoch through the ordinary audited close operation, then begins one linked epoch;
- `refresh` checkpoints a valid epoch and rotates it through close plus begin when expiry, policy change, or configuration change requires reauthentication;
- `logout` closes an open or expired epoch and is a stable no-op when the stream is already absent or closed.

Every underlying status, recover, begin, checkpoint, and close call obtains a fresh exact SSIAG decision. A transition never reuses capability evidence. Derived operation identifiers bind each step to the host event, and the coordinator's semantic operation fingerprints make retry safe. If the completion step already appears in the current journal, the transition reports `already_applied` instead of repeating it.

Discovery recovery is opt-in through `--recover`. It is attempted only for the bounded damaged-head or damaged-journal error family. Authentication failures, incompatible critical state, ambiguous successors, unavailable installations, and permission denial are not converted into recovery attempts.

No login manager, shell hook, PAM module, systemd unit, launchd job, watcher, or background daemon is installed by this surface. A host administrator may explicitly call qxctl from an appropriate host lifecycle integration after separately reviewing that integration.

## Why Binding Registry v1 Is Not Expanded

`symphony.knowledge.engine-binding-registry.v1` intentionally closes six currently implemented roles. It remains an immutable compatibility surface for those exact coordinator and vector-engine identities. Adding arbitrary future modules to its enum or silently changing its maximum cardinality would cause older qxctl and coordinator versions to reinterpret the same protocol differently.

The generic desired-state protocol therefore sits alongside binding registry v1 rather than replacing it in place. A future compatibility adapter will project valid v1 bindings into generic observed and desired component identities. Future components will not need a synthetic legacy role.

## Generic Component Identity

The generic desired-state contract identifies a component with bounded data rather than executable discovery:

- stable component ID;
- component kind such as coordinator, vector engine, module, adapter, or UI;
- module ID and optional vector and engine IDs;
- exact selected package identity or explicit desired absence;
- install scope and administrator-selected installation root;
- required or optional presence;
- desired dock/activation disposition when those gates exist;
- selected receptor or explicit no receptor;
- compatibility requirements and critical/noncritical extensions;
- content and profile digests.

Component kind is descriptive routing, not authority. The model must not classify the caller as human, AI, service, or another actor type.

## Receipt Migration

Receipt v1 packages remain valid and are inspected through their existing exact per-module adapters. Generic discovery and mutation use the separately versioned immutable receipt-v2 contract:

- every owned path carries a content digest and file kind;
- executable entry points and descriptors are explicit;
- package identity and receipt digest are stable;
- v2 readers validate the bounded receipt-declared ownership set rather than freezing a v1-era file inventory, so a compatible newer package may add content-addressed owned files without requiring qxctl to upgrade first;
- every CMake-produced v2 package runs an existing-receipt preflight before any owned-file install rule, so a committed exact version cannot be overwritten in place and must instead be explicitly uninstalled or installed side by side under another version;
- its build-local uninstaller first requires the configured path set to match receipt-v2 ownership, validates every remaining regular file against the receipt, preserves the receipt until final removal, and treats a fully absent package or individually missing owned files as idempotent retry evidence;
- lifecycle presence is not rewritten into the immutable package receipt;
- docking and activation live in protected desired/observed state, not package ownership evidence;
- unknown executables are never run merely because a receipt was discovered.

New qxctl versions must dual-read supported v1 and v2 evidence. They must not rewrite a v1 receipt into v2 or infer missing v2 facts. An established binding still requires the exact component/module/engine/vector identity, component kind, process entry point, receptor compatibility, and critical host platform declared for that role; forward file-set extensibility never weakens those checks. Generic package mutation is limited to an exact receipt-v2 package copied from an explicit trusted staged root. A package may own neither a root-level `.symphony-*` control path nor any path beneath `share/symphony/receipts/`; those namespaces are reserved for serialization, ownership, compatibility, and immutable receipt commit evidence. No package is downloaded. Install publishes the immutable receipt only after every owned file is durable; uninstall requires a separate exact staged rollback proof, validates every remaining owned file before deletion, and removes the receipt last. Missing files during retry are treated only as resumable evidence, while any conflicting administrator file or digest mismatch fails closed. Receipt v1 remains observation-only until an exact per-package mutation adapter is separately reviewed.

The root-local `symphony.knowledge.lifecycle-root-ownership.v1` registry is the serialized cross-control-domain package-mutation authority for receipt-v2 packages. Each participating TOPS/profile/state-root control domain contributes content-addressed `retained` or `retiring` claims beneath the same package-root lock. An install requires an enforced registry, exact fence, and the calling profile's exact retained claim. An uninstall requires an enforced registry, exact fence, the calling profile's exact retiring claim or exact legacy release, and every other claim for that receipt to be retiring. Any retained or legacy-preserve claim blocks reclamation. When all profile claims are retiring, one serialized uninstall removes the package and retires every satisfied claim, so independently ordered releases cannot deadlock. The registry is operational evidence, not canonical knowledge or permission; every operation still requires exact SSIAG authorization and the prepared lifecycle action.

A newly created empty root enters enforced state immediately. Before the first registry commit, qxctl durably publishes the static `symphony.knowledge.lifecycle-root-ownership-fence.v1` document at `share/symphony/receipts/symphony-root-ownership/1/install-receipt.json`. Ownership-aware observation consumes only its exact bytes; ownership-unaware lifecycle clients see an unsupported preserved receipt, and the existing coordinator therefore emits a global critical blocker without actions. This turns their already-ratified unknown-package behavior into a two-way compatibility fence rather than assuming that an older executable can interpret new hidden state. If the first registry commit is interrupted, status reports the orphaned fence and explicit reconciliation reconstructs the registry without removing the fence.

A pre-existing root without a registry enters `adoption_required` and receives conservative `legacy_preserve` claims for every observed receipt. Adoption proves that current profile claims have been reconciled but never silently releases unmatched legacy packages; those require an explicit digest-bound `ownership release`. Adoption is also the explicit operational drain barrier: the administrator must allow any mutation already prepared by an older client from a pre-fence observation to finish or be abandoned before adopting. A file-level fence cannot retroactively cancel such an in-flight prepared action. If one nevertheless completes late, the next complete observation exposes the changed inventory; retained desired presence remains a missing-package obligation, while any introduced unclaimed package restores a legacy-preserve claim and reopens adoption. If package removal becomes durable before the matching registry commit, that observation prunes only now-absent `retiring` and `legacy_preserve` claims; a `retained` claim survives absence so its owner still sees the missing desired package. This makes mixed-version behavior fail-closed and recoverable in both upgrade orders without claiming impossible retroactive control over an older process. Profile updates cannot remove or relocate a claimed component/root mapping, and profile removal remains blocked while the profile owns or retires a package. The administrator first keeps the component mapped with desired absence, converges it, then removes or relocates the now-unclaimed definition. The enforced lock order is profile store then installation root: profile mutation evaluates root claims while holding the exclusive profile lock, and profile-derived reconciliation plus every apply action holds a shared profile lease through its root or adapter mutation. A profile therefore cannot move, disappear, or change generation between ownership reconciliation and package mutation.

Build-local receipt-v2 installers and uninstallers remain valid for installation roots that have never entered qxctl ownership administration. When either the root-local registry or its receipt-layout compatibility fence exists, direct installers refuse before writing content and uninstallers refuse before mutating present content, directing the administrator to qxctl so an independently invoked package-local command cannot race or bypass active cross-profile claims. qxctl may still install from a private staged root because the staged package is validated before the serialized final mutation. A fully absent package remains an idempotent uninstall success before that gate.

Ownership status, adoption, and legacy release acquire the same shared profile lease and revalidate the exact configured root before reading or mutating root state.

## Desired, Observed, and Applied State

Four state surfaces remain separate:

- **profile input**: bounded declarative caller intent accepted by qxctl without caller-manufactured generations or digests;
- **desired state**: protected noncanonical administrator intent managed through qxctl;
- **observed state**: a disposable content-addressed inventory of validated receipts, files, bindings, and authenticated Maestro presence;
- **applied state**: durable noncanonical evidence of the last exact lifecycle plan successfully committed.

Protected qxctl runtime state remains a fifth administrative surface. It records only the exact selected receipt plus generic `active`/`inactive` eligibility for a component, uses its own linked compare-and-swap generations, and remains `undocked`. It is not an immutable package receipt, an engine binding, a Maestro presence record, or proof that arbitrary component code ran.

A difference is a plan input, not an error and not permission to mutate. Version precedence is never inferred from semantic-version recency. Upgrade and rollback are both exact selected-identity changes.

The planner uses these dispositions:

- desired and observed exact match: converged;
- desired present, selected package missing: pending or blocked when required;
- desired absent, package present: retirement candidate, preserved until authenticated apply;
- observed package not mentioned by desired state: unmanaged and preserved;
- selected version changes while both versions exist: exact switch candidate;
- binding references a temporarily missing package: degraded binding retained for diagnosis, never silently deleted;
- unknown noncritical extension: preserved through compatible writes;
- unknown critical extension, ambiguous package identity, or unsupported newer state: preserved and mutation blocked.

## Dependency-Driven Two-Way Convergence

Lifecycle action order is derived from explicit dependencies and verified state, not from vector name, directory order, discovery order, release recency, or one hard-coded sequence. Every plan carries a dependency graph and a deterministic ready set. Stable action identifiers derive from action semantics, prerequisites, and expected evidence rather than ordinal position.

The v1 scheduler contract is `dependency_ready_set_v1`:

1. select all actions whose prerequisites are satisfied by the current verified observation;
2. deterministically choose from that ready set by lexicographic action ID;
3. preserve a blocked action and continue unrelated ready actions;
4. re-observe after each committed or already-applied action;
5. recompute the ready set only when verified evidence changes;
6. emit a new linked plan revision when readiness, compatibility, or required direction changes;
7. resume a previously blocked action when its prerequisites become satisfied;
8. verify the whole resulting observation before applied state can advance.

Two-way means both upgrade orders and both action directions are first-class. A new qxctl may drive an older compatible coordinator, and a new coordinator may preserve the older command surface. Where rollback is supported, forward actions carry an explicit inverse relationship; neither direction is privileged as “newest.” The scheduler may change component-action order to converge, but it may not reorder the enclosing safety phases: lock, observe, authorize, compare-and-swap, act, verify, and audit remain ordered invariants.

Blockers are typed as `dependency_wait`, `observation_retryable`, `compatibility_blocked`, `authorization_denied`, `integrity_fatal`, `critical_state_unknown`, or `cycle_detected`. An unsatisfied dependency marked `critical: true` is a hard localized gate. An unsatisfied dependency marked `critical: false` is emitted as an explicit advisory and does not stall convergence. Missing declared component capabilities are localized compatibility blockers. Dependency waits and retryable observation errors may become ready after new evidence. Authorization denial, integrity failure, and unknown critical state do not become permission to try a different order. A dependency cycle blocks that cyclic component set while unrelated acyclic components remain eligible; the scheduler never guesses a cycle-breaking edge.

A single plan is bounded to 4,096 actions, one transaction to 256 plan revisions, and one action to eight attempts. Exceeding a bound is explicit blocked evidence, not an invitation to discard history or silently start another transaction.

The implemented report-only planner additionally binds each dock action to one exact receptor and every action to a target-state digest. A receptor change becomes an ordered undock/dock pair. Changing a selected package while the component is active or docked becomes `undock`, `deactivate`, `select`, restore desired activation, then `dock`. This local safety order is represented through prerequisites inside the action graph; it does not serialize unrelated ready components. Reinvocation with changed verified observation evidence recomputes the graph and can release a prior dependency wait without editing the earlier report.

## First Boot After Change

“First boot” is evidence-based, not a mutable boolean, wall-clock value, or version-string comparison. Its timestamps follow `knowledge/TIME.md`; the target TOPS supplies durable commit time, while the Linux kernel boot UUID, generations, operation IDs, content digests, and predecessor links establish identity and causal position. The implemented `qxctl knowledge lifecycle host` surface may install one explicit system-scoped systemd receptor per TOPS/profile. On each Linux boot it invokes the existing report-only boot circuit, records or replays `host-boot:<kernel-boot-id>`, and exits. It never invokes lifecycle apply, downloads a package, activates component code, or becomes a watcher or daemon.

### Linux systemd host receptor

`qxctl knowledge lifecycle host install|update|status|reconcile|enable|disable|uninstall|run` administers or dispatches the receptor. Every operation is caller-neutral, requires its own exact SSIAG permission, and binds the stable TOPS/profile lifecycle resource. Installation and descriptor mutations use exact compare-and-swap through `--expected-host-digest`. The integration root is administrator-selected at install; changing it requires explicit uninstall followed by install so marker-owned state cannot be orphaned by an in-place move. System scope and administrator privilege are mandatory for mutation; native Windows and launchd receptors are not implemented. Windows administrators use WSL or an explicitly connected remote Linux TOPS node.

The stable unit namespace is `symphony-qxctl-lifecycle@<tops-id>-<profile-hash>.service`. The unit loosely orders after and wants the independently supervised per-TOPS SSIAG and STAV units, but does not alter their deliberate independence from each other. It is a hardened oneshot with bounded restart on failure. The descriptor contains no shell, environment lookup, executable discovery, repository choice, apply flag, or version-recency selection.

The qxctl executable copied into the receptor is immutable and content-addressed beneath the administrator-selected integration root. The durable host descriptor identifies the active executable and up to eight explicitly ordered fallback executors, the exact repository/state/integration roots, enablement intent, systemd unit digest, recovery mode, generation, predecessor digest, and STSC UTC update time. An update installs the new slot side by side and retains the predecessor set. A cached old unit and a new descriptor therefore remain procedurally compatible: either accepted executable can read v1 state, while an unrecognized executable fails closed. Reconciliation repairs the unit from the descriptor, restores enablement, and promotes the first digest-valid fallback when the selected slot is unavailable. No semantic-version ordering is inferred.

`strict` recovery mode reports damaged lifecycle boot state without mutation. `discover` mode asks the existing bounded coordinator recovery operation to select only one uniquely linked local successor, then repeats status. Authentication denial, unavailable exact installations, ambiguous evidence, unsupported critical state, and permission failure still fail closed. A repeated invocation during the same Linux boot recognizes the boot UUID already stored in the current journal and returns replay evidence without creating a new revision.

Uninstall first commits `retiring` with desired enablement false, then disables the unit, removes only the exact regular unit and marker-owned executor tree, reloads systemd, and finally removes the protected host descriptor. If interruption occurs after any step, `host reconcile` resumes the retiring cleanup. Unknown objects in the integration root prevent deletion. The ordinary explicit `knowledge lifecycle boot|status|recover` grammar remains available whether or not this receptor is installed, so host integration is removable and is not required for lifecycle correctness.

SSIAG policy targets for this lifecycle are stable per exact TOPS and profile, while permissions remain exact per operation. The profile-list operation uses a domain-separated stable per-TOPS profile-catalog resource because it enumerates several profiles; no valid profile ID can collide with that catalog identity, and catalog read authority does not inherit mutation permission over any profile. Receipt digests, desired/observed/applied evidence, journal digests, and compare-and-swap values remain bound and independently verified by qxctl and the coordinator; they are not embedded in the SSIAG resource name. This prevents policy churn on every version or content change without turning a grant into wildcard artifact authority. `qxctl ssiag grants lifecycle` emits the complete deterministic caller-neutral grant input for one configured subject and authority basis, including the exact profile and profile-catalog resources. It is proposal-only evidence with `apply_enabled=false`; applying or replacing SSIAG policy remains a separate reviewed surface.

A stable lifecycle observation key binds at least:

- desired-state digest;
- stable observed-inventory digest, computed from the validated observation while excluding the document-only `observed_at` and `observation_digest` fields;
- binding-registry digest when present;
- supported lifecycle protocol/capability set;
- TOPS and profile identity;
- normalized platform-compatibility digest covering the operating-system/kernel ABI, architecture, qxctl/coordinator identities, and configured provider availability without host secrets or volatile boot identity.

The SSIAG boot resource separately binds the exact profile digest, desired-state digest, selected boot mode, stable-inventory digest, TOPS, and profile identity. Authorization for one evidence tuple cannot be replayed for a different desired plan or mode.

The complete observation document remains content-addressed, so refreshed collection time changes its evidence digest. Time alone does not change the stable inventory digest, transaction identity, ready set, or semantic action identities. Applied state stores the last successfully stabilized observation key. The prior applied-state digest is the transaction's compare-and-swap anchor, not an input to the stable observation key; otherwise every successful applied-state write would invalidate its own next boot. A transaction identity binds the new observation key and the exact prior applied-state digest. An observation key equal to the last stabilized key is an idempotent no-op. A changed key starts or resumes one durable boot transaction. The sequence is:

1. acquire the protected lifecycle lock without sharing any hot/warm lock;
2. read the desired profile and prior journal without following links;
3. inventory only configured roots and validate bounded receipts without executing discovered code;
4. negotiate common read/write protocols and capabilities;
5. build a deterministic plan and plan digest;
6. report, block, or request authenticated apply according to the qxctl-managed policy;
7. apply only actions whose exact expected state and artifacts still match;
8. checkpoint every completed action and preserve incomplete actions as forward-resumable evidence;
9. verify the resulting observation before closing the transaction;
10. publish only safe lifecycle/audit metadata through the applicable SSIAG/STAV path.

Steps 1 through 9 are implemented for the exact local `apply-compatible` scope. The report-only v1 journal remains unchanged and becomes the required source authorization for a separate apply-capable v2 stream. Before qxctl changes host state, the coordinator durably records the selected action as active. An exact staged receipt may resolve only the single package-absence blocker on an install with no graph prerequisites or additional blockers; it never converts an ordered or multiply blocked action into a ready action. qxctl then executes only a reviewed generic adapter, re-observes all configured roots, and asks the coordinator to finalize the attempt. The coordinator accepts success only when the new observation directly proves the prepared transition, then writes content-addressed applied evidence and advances the v2 journal/head commit point. A crash before host mutation replays the prepared action; a crash after host mutation observes `already_applied`; a crash after applied-evidence publication but before head publication leaves an unselected immutable file and recovery continues from the selected journal chain. When the report is already converged, an explicit close operation commits applied evidence without manufacturing an action.

Step 10 is satisfied only for the existing SSIAG authorization-decision audit path: each status, prepare, finalize, close, and recovery request obtains fresh exact permission evidence whose safe security decision is committed through SSIAG/STAV before release. qxctl and the coordinator do not append a second direct lifecycle-action event, and transient attempt details never become mutable STAV ledger state. A distinct lifecycle event family, if desired beyond the existing decisions, remains a separate protocol and producer-integration gate.

The administrator-selectable boot mode is `report` or `apply-compatible`. `report` remains the default and never executes an action. `apply-compatible` only makes the profile eligible for the separate explicit `qxctl knowledge lifecycle apply` command; `boot` itself remains report-only. Apply requires the exact report-journal digest, exact current apply-journal state, exact applied-state state, a stable operation ID, and explicit trusted staged roots. Disabling application never disables parsing bounds, ownership checks, receipt integrity, expected-state compare-and-swap, critical-extension blocking, or secret exclusion.

Desired profiles are protected per TOPS beneath `${XDG_STATE_HOME:-~/.local/state}/symphony/<tops-id>/qxctl/knowledge/lifecycle/profiles/`, or beneath an explicit qxctl-selected state root. Mutations use an exact expected profile digest, semantic retry is a stable no-op, generations and predecessor digests are qxctl-generated, and durable writes use a persistent no-follow lock plus atomic replacement and directory synchronization. Profile roots may be changed through qxctl; an absent future installation root is retained as empty observed evidence rather than created or treated as a scan failure. Existing roots are no-follow trusted directories, and observation scans only `<root>/share/symphony/receipts/<module>/<version>/install-receipt.json`, never arbitrary executable discovery. Known receipt v1 packages use exact existing adapters. Receipt v2 packages are checked against their content-addressed owned files, entry points, capabilities, receptors, and platform requirements. Unsupported, unreadable, and ambiguous packages remain explicit preserved unknown evidence.

If the desired profile is absent, unsupported, or critically extended, first boot fails closed or reports protected read-only compatibility state and preserves every installed package and binding. If an operating-system update changes only the platform-compatibility digest, the same planner reevaluates all selected components against their declared platform requirements without assuming reinstall, upgrade, or removal. The `knowledge lifecycle boot|status|recover|apply|apply-status|apply-recover` and Linux `host install|update|status|reconcile|enable|disable|uninstall|run` grammar is implemented.

## Durability and Recovery

Both lifecycle journal families use the proven durability pattern: a persistent mode-`0600` no-follow lock, private per-TOPS/profile directories, dual slots, an atomically replaced head, file and directory synchronization, linked generations, stable operation IDs, and exact compare-and-swap. Report journal v1 and apply journal v2 are side by side rather than an in-place format upgrade, and each has its own lock and head. The v2 stream names and digest-binds its exact v1 source journal. It persists action attempts before and after external mutation and selects immutable `applied.<digest>.json` evidence only through the committed journal/head. Its stable-inventory digest deliberately excludes only observation collection time and its enclosing document digest, so a timestamp-only rescan does not advance either transaction. A future writable format must dual-read its predecessor before it can migrate state.

Implemented report and apply recovery may repair a missing/damaged head or adopt one uniquely linked adjacent successor by committing a new forward recovery checkpoint. Each requires exact expected state or explicit `--discover`, full write compatibility, and a unique digest-linked local chain. Apply recovery preserves the active prepared action and its attempt history so qxctl can re-observe and resume without guessing whether the host mutation occurred. Recovery may not select between divergent equal-generation states, delete unknown state, downgrade a critical extension, manufacture a successful action, guess a dependency edge, reorder a safety phase, or erase a still-unresolved desired/observed difference.

Transient attempt errors remain noncanonical journal evidence and are reconciled forward. A failed or blocked attempt remains visible within the current transaction, while a later evidence-backed retry appends a new attempt and closes only after whole-profile verification. Canonical vectors and append-only ledgers are never poisoned with mutable boot-attempt state.

## Addition and Removal Scenarios

### Module arrives before desired-state support

The package remains installed and unmanaged. Older software does not execute or remove it. A compatible qxctl can later adopt it after validating its receipt and an administrator updates desired state. Receipt-v2 support may arrive before or after the package without changing that rule.

### Desired state arrives before the module

The plan remains pending. Existing compatible versions and bindings remain intact. A required missing component blocks only the dependent freezing-path lifecycle action, not unrelated modules or hot/warm work.

### qxctl arrives before the coordinator

qxctl reports capability mismatch and performs no lifecycle mutation. Manual session commands supported by the older coordinator remain available.

### Coordinator arrives before qxctl

The newer coordinator continues accepting the older command contract. New optional behavior remains dormant because the older client does not advertise it.

### Module is dropped from desired state

The package becomes a retirement candidate. Apply must first verify that no selected binding, open transaction, active receptor, or dependent desired component still requires it. Uninstall removes only receipt-owned files and preserves journals, audit evidence, canonical knowledge, and unrelated administrator files.

### Upgrade or rollback is interrupted

Both package versions may coexist. Protected runtime selection and the applied-state digest remain on the last committed identity until one exact forward action commits. Retry reuses the same semantic operation ID and either observes the completed step or resumes the prepared action from the last checkpoint. No semantic-version preference is inferred.

## Maestro Docking Boundary

Desired state may carry a preferred receptor, but it cannot claim docking without observed Maestro evidence. Observation and applied-state evidence carry the exact receptor identity whenever they report `docked`; a dock action carries the exact target receptor. A lifecycle invocation that enables Maestro supplies an exhaustive, duplicate-free set of receptor identities covering both every possible current receptor and every desired target. qxctl queries each under a fresh exact authorization, treats duplicate live presence as critical ambiguity, discovers the old receptor before inverse undock, and rejects desired targets outside the supplied set. This lets a later invocation safely reverse or complete an out-of-order receptor change without converting an unobserved receptor into fabricated absence. The planner may report receptor incompatibility without treating package installation as failed. Docking is a separate authenticated qxctl action with its own expected registry state, exact receipt/executable evidence, and verified re-observation. Maestro commits presence only and never invokes the recorded engine.

## Thermal and Platform Boundary

Lifecycle observation, planning, recovery, SSIAG/STAV exchange, and Maestro coordination are freezing-path work. They do not execute inline with hot or warm trading work, share progress locks with it, or create synchronous trading dependencies.

The engine implementation remains Linux-first with macOS development support. Native Windows lifecycle engines are not planned. WSL uses Linux lifecycle behavior; a later cross-platform UI may administer a WSL or remote supported node through qxctl contracts.

## Implementation Sequence

1. Implement explicit idempotent qxctl login/refresh/logout transition composition over the existing authenticated coordinator operations. **Completed.**
2. Add the generic desired-state, observation, dependency-driven plan, applied-state, boot-journal/head, and immutable content-addressed receipt v2 schemas while retaining strict v1 adapters. **Completed.**
3. Implement the C++ coordinator dependency scheduler, two-way compatibility negotiation, and deterministic report-only planner over caller-supplied evidence. **Completed.** Configured-root observation remains in step 4.
4. Implement qxctl desired-profile administration and configured-root inventory with caller-neutral SSIAG authorization. **Completed.** qxctl also performs fresh observation and exact bound-coordinator invocation for report-only planning; no plan is persisted.
5. Implement durable C++ boot journaling, report mode, bounded replanning recovery, and installation-change diagnosis. **Completed for report-only operation, including the explicit Linux systemd receptor.** Applied-state writes and action attempts were excluded from this v1 step and are implemented only through the separate v2 boundary in step 6.
6. Implement separately gated `apply-compatible`, exact package lifecycle actions, and forward/inverse rollback proof. **Completed for explicit local receipt-v2 install/uninstall, protected selection/activation, six-role binding adaptation, coordinator upgrade/rollback handoff, and the report-only Linux host receptor.** Live service/process activation, receipt-v1 mutation adapters, downloads, automatic old-version reclamation, login/session hooks, and hidden host hooks remain excluded.
7. Add Maestro docking only after its receptor and presence contracts are ratified. **Completed for authenticated durable presence only; engine invocation, supervision, and scheduling remain excluded.**

Each step must preserve older supported operation surfaces and pass upgrade-order matrices in both directions.

## Current Non-Authorizations

The explicit Linux systemd receptor is freezing-path dispatch to report-only qxctl administration and grants no apply, component-execution, remote, hot-path, or canonical authority.

These contracts do not authorize implicit or unattended apply, receipt-v1 package mutation, implicit latest-version selection, package download, arbitrary executable discovery or entry-point execution, receipt rewriting, live process/service activation, in-place coordinator self-replacement, unconstrained engine-binding rewrite, Maestro engine invocation or supervision, remote lifecycle APIs, canonical knowledge mutation, hot/warm participation, native Windows engines, or hidden host integration. Established-role binding changes occur only as prepared lifecycle actions with exact receipt-v2 evidence and binding-registry compare-and-swap. Coordinator selection additionally requires the candidate executable to reopen and reproduce the exact prepared journal before the old coordinator yields the binding; the old coordinator finalizes the attempt, and a crash retry through the new binding observes the same durable active action. Superseded packages remain installed by default so rollback and bespoke multi-version designs remain possible. The implemented generic activation state is protected administrative eligibility only, and Maestro docking is presence rather than execution. A canonical schema is not evidence that any broader adapter or runtime behavior exists.
