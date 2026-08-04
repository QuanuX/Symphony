# Symphony Knowledge Vector Engine Foundation Specification

## Status and Normative Terms

Architect-ratified cross-vector architecture with the explicitly bounded `0.1.0-dev` foundation/coordinator, SKVI/SCLV/SACV/SODV/SSFV proposal/projection slices, exact three-record SSFV partial bootstrap, protected user-default engine binding registry, user-scope reconciliation journals, SSIAG-authorized noncanonical session journals, explicit qxctl session transitions, implemented report-only coordinator lifecycle planning, and canonical desired/observed/plan/applied/boot-journal plus receipt-v2 lifecycle contracts in `knowledge/LIFECYCLE.md`. MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative when the related implementation exists. No additional feature record, complete-catalog claim, lifecycle persistence/observation/apply, canonical apply, endpoint document, publication, repository-specific binding, system/TOPS binding, observer, or docking capability may be inferred from these contract slices.

## Purpose

Define the common process, authority, lifecycle, proposal, projection, installation, recovery, and thermal boundaries for independently installed SKV vector engines.

## Ownership Model

Canonical Markdown and typed artifacts owned by a vector remain source truth. A vector engine implements that vector's declared behavior but does not own the contract. qxctl owns command grammar and presentation, not vector semantics. Provider adapters supply bounded evidence, not canonical authority. Maestro records deployment presence and docking, not vector truth.

Shared C++ mechanics MUST remain domain-neutral and authority-free. They MAY implement bounded parsing, snapshots, digests, path safety, protocol framing, journals, proposal assembly, transaction staging, and install-receipt mechanics. They MUST NOT decide feature-worthiness, architectural purpose, compatibility acceptance, publication approval, legal capacity, or ratification.

## Engine Topology

Each vector engine is a separate executable and independently installed module. The coordinator is a separate executable. In-process dynamic plugins and a shared C++ ABI are not the default architecture. Shared mechanics are statically linked so an engine can be versioned, diagnosed, rolled back, and removed independently.

An engine MUST declare:

- stable module, engine, and vector identifiers;
- engine and supported contract versions;
- protocol compatibility range;
- owned read paths, proposed write paths, and operation vocabulary;
- maximum input, output, path, file, count, depth, and execution bounds;
- supported user, system, and TOPS scopes;
- dependency and build provenance;
- install receipt and Maestro docking compatibility;
- deterministic conformance fixtures.

## Protocol Identifiers

The v1 identifier family is:

| Identifier | Role |
|---|---|
| `symphony.knowledge.engine-process.v1` | process request/response envelope |
| `symphony.knowledge.proposal.v1` | immutable proposal |
| `symphony.knowledge.engine-descriptor.v1` | engine identity and capability descriptor |
| `symphony.knowledge.session-journal.v1` | authenticated authority-epoch journal |
| `symphony.knowledge.session-head.v1` | atomic dual-slot session-journal head |
| `symphony.knowledge.session-command.v1` | qxctl-to-coordinator authenticated-session command |
| `symphony.knowledge.session-result.v1` | coordinator authenticated-session result |
| `symphony.knowledge.session-transition-result.v1` | qxctl explicit host-event convergence result |
| `symphony.knowledge.reconciliation-journal.v1` | noncanonical worktree coordination journal |
| `symphony.knowledge.reconciliation-head.v1` | atomic dual-slot journal head |
| `symphony.knowledge.reconciliation-command.v1` | qxctl-to-coordinator reconciliation command |
| `symphony.knowledge.reconciliation-result.v1` | coordinator reconciliation result |
| `symphony.knowledge.provider-evidence.v1` | normalized provider evidence |
| `symphony.knowledge.install-receipt.v1` | installed-file and lifecycle receipt |
| `symphony.knowledge.lifecycle-desired-state.v1` | protected noncanonical exact component intent |
| `symphony.knowledge.lifecycle-observation.v1` | bounded disposable installation and platform evidence |
| `symphony.knowledge.lifecycle-plan-command.v1` | exact report-only planner request and compatibility declaration |
| `symphony.knowledge.lifecycle-plan.v1` | dependency-ready-set forward/inverse convergence plan |
| `symphony.knowledge.lifecycle-applied-state.v1` | last verified lifecycle convergence evidence |
| `symphony.knowledge.lifecycle-boot-journal.v1` | durable boot transaction, replan, blocker, and recovery evidence |
| `symphony.knowledge.lifecycle-boot-head.v1` | atomic dual-slot boot-journal head |
| `symphony.knowledge.engine-binding-registry.v1` | protected noncanonical user-scope exact-version selection |
| `symphony.knowledge.install-receipt.v2` | immutable content-addressed package ownership and capability evidence |
| `symphony.maestro.knowledge-engine-docking.v1` | Maestro docking projection |

The initial exact schemas are:

- `knowledge/schemas/v1/engine-process-request.schema.json`;
- `knowledge/schemas/v1/engine-process-response.schema.json`;
- `knowledge/schemas/v1/engine-descriptor.schema.json`;
- `knowledge/schemas/v1/install-receipt.schema.json`;
- `knowledge/schemas/v1/engine-binding-registry.schema.json`;
- `knowledge/schemas/v1/reconciliation-journal.schema.json`;
- `knowledge/schemas/v1/reconciliation-head.schema.json`;
- `knowledge/schemas/v1/reconciliation-command.schema.json`;
- `knowledge/schemas/v1/reconciliation-result.schema.json`;
- `knowledge/schemas/v1/session-journal.schema.json`;
- `knowledge/schemas/v1/session-head.schema.json`;
- `knowledge/schemas/v1/session-command.schema.json`;
- `knowledge/schemas/v1/session-result.schema.json`;
- `knowledge/schemas/v1/session-transition-result.schema.json`;
- `knowledge/schemas/v1/lifecycle-desired-state.schema.json`;
- `knowledge/schemas/v1/lifecycle-observation.schema.json`;
- `knowledge/schemas/v1/lifecycle-plan-command.schema.json`;
- `knowledge/schemas/v1/lifecycle-plan.schema.json`;
- `knowledge/schemas/v1/lifecycle-applied-state.schema.json`;
- `knowledge/schemas/v1/lifecycle-boot-journal.schema.json`;
- `knowledge/schemas/v1/lifecycle-boot-head.schema.json`;
- `knowledge/schemas/v2/install-receipt.schema.json`.

The process request limit is 1 MiB and the response limit is 4 MiB. JSON depth is at most 64, parsed values/events at most 16,384, one string or key at most 65,536 bytes, integers remain within `[-9007199254740991, 9007199254740991]`, and a request deadline is at most 300 seconds ahead. Unknown fields, duplicate names, invalid UTF-8, trailing data, floating-point values, out-of-range integers, unsupported versions, excessive input, unsafe paths, expired deadlines, and target mismatch fail closed. Standard output is reserved for the single protocol response; bounded diagnostics use standard error. Arguments and environment variables MUST NOT carry secrets or arbitrary executable instructions.

An engine checks the deadline before and between bounded work units and file-read chunks. The invoking process MUST independently enforce the same deadline on child-process lifetime so a blocked operating-system or filesystem call cannot outlive the request. The direct coordinator slice provides cooperative checks; the implemented qxctl SKVI/SCLV/SACV/SODV/SSFV client adds a hard child-process timeout around each request deadline.

`response_digest` is the tagged SHA-256 of the compact key-sorted response object before that member is inserted. Operation-specific payload/result schemas remain owned by the applicable coordinator or vector contract.

## Authenticated Session Model

The default knowledge session is an authenticated authority epoch. It begins after successful authentication and ends at logout, session or credential expiry, revocation, or a boundary requiring re-authentication. Re-authentication creates a new authority epoch.

An administrator MAY select another supported session-lifecycle policy through qxctl. Configuration MUST NOT extend authority beyond logout, expiry, revocation, or required re-authentication. Product windows, shells, assistant interactions, and forge requests may bind to a session but do not independently create authority.

The implemented session slice is local, user-scope, and noncanonical. Before every `session_begin|status|checkpoint|close|recover`, qxctl authenticates the selected TOPS SSIAG Unix-socket endpoint and requests an exact authorization decision for that operation, an opaque digest of the canonical repository root, the `qxctl` audience, and TOPS session scope. SSIAG derives the subject from kernel peer credentials, evaluates explicit subject grants with a deny default, and returns only safe, expiring, non-transferable capability evidence after the corresponding STAV policy-decision event commits. qxctl and the coordinator independently recompute or compare the repository resource and reject caller-class-dependent evidence, canonical-apply claims, target drift, stale evidence, and binding-digest mismatch.

Session state is keyed by exact TOPS, subject, and repository identities. It uses a protected no-follow lock, two journal slots, an atomic head, compare-and-swap mutations, stable operation IDs, linked authority epochs, and unambiguous forward recovery. Context references may attach one or more separate reconciliation journals to an epoch without turning those contexts into identity or permission evidence. The capability is not a transferable bearer credential; its protected use authorizes only the named noncanonical coordinator operation. Programmatic canonical apply remains disabled.

The implemented `qxctl knowledge session transition` surface composes those primitives for explicit `login`, `refresh`, and `logout` host events. One stable event ID derives distinct recover/close/begin/checkpoint operation IDs. A retry that observes its completed step reports `already_applied`. Login closes a stale open epoch before beginning a linked epoch; refresh checkpoints a valid epoch or rotates it when the coordinator proves reauthentication is required; logout is a stable no-op for absent/closed state. Every composed step obtains fresh SSIAG evidence. `--recover` is explicit and limited to the damaged local-head/journal family; it never converts denial, incompatibility, or ambiguity into a repair attempt. qxctl installs no host hook or watcher.

## Worktree Reconciliation Context

One authenticated session may contain multiple worktree-scoped reconciliation contexts. Each context owns its repository/worktree identity, initial and current content digests, vector-contract digests, engine inventory, journal, observer hints, and writer/reconciliation lock. A reconciliation context is noncanonical coordination state, not authentication or permission evidence, and may be inspected or repaired independently of canonical apply.

Separate worktrees MUST NOT share a mutable journal or writer lock. VCS merge is their cross-worktree reconciliation boundary. Absolute paths may appear only in protected local state and never in canonical records or portable proposals.

Correctness comes from content-addressed snapshots at begin, checkpoint, close, pre-apply, post-apply stabilization, and next-authenticated-session recovery. Filesystem notifications, IDE callbacks, forge events, and Git hooks are optional latency hints. A missed event MUST self-heal through the next bounded digest comparison.

Hooks are explicit, removable, and never installed silently. A disposable observer MAY run only for an active reconciliation context and MUST terminate at close. Neither is required for correctness.

### Reconciliation Durability and Compatibility

The reconciliation journal is deliberately separate from the authenticated-session journal. It contains no proof of identity, authority, permission, or ratification. An authenticated authority epoch may bind one or more reconciliation context identifiers without merging their expected-state or lock domains.

One protected context directory contains a persistent no-follow lock, two journal slots, and an atomic head. A mutation writes and synchronizes the inactive slot before atomically replacing and synchronizing the head. Recovery validates the head and both slots. It may adopt only one uniquely linked durable successor, repair a head that references an intact slot, or roll forward from the highest valid slot while recording the discarded head digest. Ambiguous divergent successors, invalid ownership/modes, unknown critical extensions, unsafe paths, stale expected state, or unsupported formats fail closed with bounded repair guidance.

Compatibility is negotiated from process protocols, journal read/write versions, operation availability, and named capabilities. Package recency is not compatibility evidence. Readers preserve unknown noncritical opaque extensions through their parsed JSON value and reject unknown critical extensions. Writers retain the existing supported journal format unless a separately reviewed compare-and-swap migration succeeds. Format transitions are stepwise, idempotent, content-addressed, and recover through the same dual-slot procedure. A newer component MUST preserve the preceding supported format while it claims two-way compatibility; an older component remains fully operational only while the journal and required capability set remain within its declared range.

Unsupported or lossy combinations never delete an installation, rewrite a journal, fabricate a downgrade, or silently reduce safeguards. They preserve evidence and return deterministic upgrade, downgrade, rebind, or explicit-migration guidance. Self-healing means recovery from verifiable local evidence; it never manufactures authority or guesses across incompatible state.

## Proposal Boundary

A proposal is immutable, content-addressed, noncanonical, deterministic for the same declared inputs, and safe to inspect without granting mutation authority. It binds:

- proposal, engine, vector, and contract identities and versions;
- repository, revision scheme/value, worktree, and tree digest;
- authenticated-session reference and worktree-context reference when applicable;
- bounded read set and input digests;
- vector-owned prospective write set and expected prior digests;
- normalized evidence with provenance;
- typed operations and desired-change digest;
- deterministic validation results;
- creation and expiry semantics.

Proposals MUST NOT contain secrets, credentials, proofs, raw tokens, unbounded provider payloads, environment dumps, absolute portable paths, or arbitrary commands. Proposal generation never establishes permission or ratification.

Every proposal authority object states `engine_decided_domain_truth: false`. This common, vector-neutral assertion means validation and deterministic computation do not let an engine decide membership, semantic ownership, ratification, publication eligibility, feature-worthiness, or another vector-owned fact. Vector-specific validation evidence may name the exact decision boundary but MUST NOT replace or weaken the common assertion.

## Machine-Managed and Semantic Content

An engine MAY compute deterministic facts and propose bounded machine-managed fields or sections when the owning vector contract defines their markers and formatting. Semantic claims remain caller-declared or caller-proposed until a caller holding the required permission ratifies them.

An engine MUST preserve unknown and owner-controlled content. Ambiguous ownership markers, overlapping writes, unexpected prior digests, or unstable repeated output fail closed. Generation method is not an authority category: computed content may become canonical only through the same ratified path, while manually written content is not correct merely because a caller typed it.

## Provider-Neutral Evidence

No engine may require GitHub, GitLab, Mintlify, NotebookLM, or another external provider to preserve canonical truth. Adapters are separately discoverable processes using bounded protocol requests. Local and air-gapped evidence is first-class. Git is the first repository substrate, not a universal identity model; revisions declare their scheme and are never universally assumed to be 40-character SHA-1 values.

## qxctl Grammar

The cross-vector groups are:

```text
qxctl knowledge engines ...
qxctl knowledge reconcile compatibility|begin|status|checkpoint|close|recover ...
qxctl knowledge session begin|status|checkpoint|close|recover ...
qxctl knowledge session transition --event login|refresh|logout --event-id ID ...
qxctl knowledge proposals ...
qxctl knowledge apply ...        # reserved; disabled until the apply gate passes
```

Vector-specific groups are:

```text
qxctl skvi ...
qxctl sclv ...
qxctl sacv ...
qxctl sodv ...
qxctl ssfv inspect|check|diff|propose|graph ...
```

qxctl MUST resolve exact installed engine identities and protocol compatibility from trusted receipts. Direct engine invocation remains available for diagnostics and conformance. qxctl MUST NOT absorb vector semantics, classify callers, accept secret-bearing engine input, or present a reserved command as operational.

The implemented user-scope `qxctl knowledge engines list|inspect|doctor|bind|unbind` surface manages one `default` binding profile. A bind selects one exact inactive-undocked receipt for a role and records its receipt and executable digests. `registry_digest` is the tagged SHA-256 of the compact recursively key-sorted registry object with the `registry_digest` member omitted and bindings sorted by role. Binding is separate from installation and Maestro docking, never selects the newest version implicitly, uses an exact expected prior registry state, and grants no repository, session, permission, vector-semantic, or canonical-write authority. Multiple versions may remain installed while one exact version is bound per role. Repository-specific profiles and system/TOPS mutations remain unavailable.

The implemented user-scope `qxctl knowledge reconcile ...` surface resolves and revalidates the exact bound coordinator, performs an explicit compatibility handshake, and invokes it through bounded local process IPC. `begin` requires a caller-declared bounded path inventory and exact expected journal state. `checkpoint`, `close`, and ordinary `recover` require the exact current digest. An administrator may request discovery recovery only explicitly when ordinary status cannot validate the head. Reconciliation mutates protected noncanonical coordination state only; it does not authenticate a caller, invoke a vector engine, mutate a canonical repository file, install a hook, run an observer, append STAV, or dock with Maestro.

The implemented user-scope `qxctl knowledge session ...` surface revalidates the exact coordinator binding, obtains a fresh exact SSIAG decision for every underlying operation, and invokes the coordinator with the common session command. `begin` requires `absent` or the exact digest of a closed predecessor; mutations require a stable operation ID and exact current digest; explicit discovery recovery is permitted only when one unique forward state is provable. `status` is read-only. `transition` performs only the explicit idempotent composition described above and returns a digest-bound noncanonical transition result. These surfaces establish and recover noncanonical authority-epoch evidence only. They do not activate a receipt, invoke vector semantics, mutate canonical knowledge, administer policy/safeguards, use a credential provider, or dock with Maestro.

## Authority and Apply Gate

Initial releases are inspect, query, check, validate, diff, project, and propose only as permitted by each vector. Programmatic canonical apply is disabled until all of the following are implemented and verified together:

1. SSIAG authenticates the caller and projects effective target-host ownership or granted permission for the exact operation and resource.
2. The proposal, repository, revision/tree, contract digest, read set, write set, and expected prior state are bound and fresh.
3. One coordinator serializes vector-owned operations under the worktree lock, stages privately, validates, and commits atomically or leaves canonical files unchanged.
4. Replay protection, idempotency, crash recovery, and bounded stabilization pass their negative tests.
5. qxctl exposes owner-configurable caller-neutral safeguards without weakening protocol integrity.
6. Required STAV event classes and producer grants are ratified and operational, including an explicit audit-deferred administrator recovery contract where applicable.

Caller type is never requested, inferred, stored, or evaluated for authority. External providers, counterparties, owners, and applicable law determine legal and financial capacity; Symphony represents effective permissions and evidence.

## Managed Freshness Gate

Structural protocol-integrity violations always fail governed validation. Proposal freshness at commit, merge, or release is an owner-configurable caller-neutral safeguard administered through qxctl. A guarded profile MAY require every affected semantic proposal to be ratified, explicitly deferred with reason, or proven irrelevant. An administrator MAY disable or replace that optional freshness gate, including through a direct profile, without disabling path safety, bounded parsing, expected-state validation, atomicity, ledger framing, or secret exclusion.

Ordinary authoring MAY carry a visible semantic proposal awaiting a caller with the required review permission. Disabling the freshness safeguard does not convert an unratified proposal into canonical truth.

## Audit-Deferred Recovery and Thermal Isolation

Ordinary audited mutation MAY fail closed while its required audit service is unavailable. A future target-host-administrator recovery route exposed through qxctl MUST record durable local evidence before completion, bind permission and expected state, mark the result audit-deferred, and reconcile forward without editing or impersonating STAV history.

Vector engines, the coordinator, qxctl recovery, SSIAG/STAV coordination, observers, projections, and deferred-audit reconciliation are administrative cold/freezing-path work. They MUST NOT execute inline with a hot or warm path, acquire locks shared with hot/warm execution, create a synchronous hot/warm dependency, or introduce blocking, jitter, or latency there. This is a bounded isolation invariant, not a complete trading-node thermal doctrine.

## Projection Boundary

JSON/JSONL, search, graph, database, documentation, SDK, and analytical outputs are derived, disposable projections. Each projection MUST bind canonical input digests and engine versions and MUST be rebuildable. A projection never becomes a competing source of truth. Each vector separately authorizes which projection classes may be implemented.

## Installation, Receipts, and Maestro

Every engine and the coordinator MUST support independent install, upgrade, rollback, and uninstall. Installation never silently changes repository hooks, canonical files, engine bindings, or Maestro state. Uninstall removes only files owned by the selected receipt and preserves canonical knowledge, binding diagnostics, and session/recovery evidence.

Every package declares compatible receptors; protected desired state carries any administrator-selected receptor. Installation without Maestro is valid and reports `installed_undocked`. Multiple compatible versions may coexist, dock, undock, and activate under explicit administrator selection. A newer installation MUST NOT silently replace the active version.

Binding registry v1 remains closed to its six ratified roles and MUST NOT be expanded in place to simulate generic lifecycle support. Canonical cross-vector desired-state, observed-state, dependency-driven plan, applied-state, generic receipt v2, and first-boot transaction contracts are defined in `knowledge/LIFECYCLE.md` and their exact schemas. Additions, removals, upgrades, and rollbacks are digest differences to be planned, not implicit permission to mutate. Unmanaged packages and temporarily missing selected packages are preserved; no first-boot path may infer newest-version preference, delete a degraded binding, execute unknown discovered code, or rewrite a v1 receipt. First-boot `report` mode precedes any separately gated authenticated `apply-compatible` implementation.

Lifecycle plans MUST derive component action order from explicit dependency readiness. They MUST continue unrelated ready actions around a localized blocker and MAY emit a linked plan revision only after verified evidence changes. Forward and inverse actions are equally explicit so supported upgrade and rollback orders do not depend on release recency. Action reordering MUST NOT reorder lock, observation, authorization, compare-and-swap, action, verification, or audit phases. Cycles block only their cyclic component set; denial, integrity failure, and unknown critical state remain stop conditions rather than alternate-order hints. The canonical scheduler bounds are 4,096 actions, 256 plan revisions per transaction, and eight attempts per action.

Docking descriptors contain no secrets, shell fragments, arbitrary arguments, or executable policy. Maestro persists deployment presence; it does not own vector semantics.

## Dependency and Platform Policy

The standard library and first-party code are preferred. A narrowly scoped C/C++ dependency MAY be used for mature JSON, YAML, Unicode, OpenAPI, or platform behavior when it is exactly pinned, audited, licensed, checksummed, reproducibly/offline buildable, and bounded to the consumers that require it. Runtime dependency downloads and unbounded plugin discovery are prohibited.

The engine foundation is Linux-first. Native Windows engines are not built. WSL uses the Linux path; remote Windows administration uses qxctl against a supported node. Broader remote administration and cross-platform UI contracts remain separate future work.

## Implementation Order

Implementation proceeds as tested vertical slices:

1. authority-free shared C++ foundation and authenticated-session/worktree coordinator;
2. SKVI inspect/check/propose/project plus qxctl integration and independent install/uninstall proof;
3. SCLV engine, provider-neutral v3 format/validator activation, and local/air-gapped evidence adapters;
4. SACV OpenAPI 3.2.0 engine;
5. SODV release/publication reconciliation engine;
6. SSFV Contract Quad, namespace, initially empty registry, and payload contracts;
7. SSFV engine and qxctl client;
8. first distributed SSFV feature bootstrap after source review and feature-worthiness ratification.

Scaffolding every engine in advance is prohibited. Each slice must pass its contract, conformance, receipt, and uninstall gates before the next vector claims implementation.

## SSFV Gate

The Architect has ratified `knowledge/ssfv/` with stable identifiers, feature-worthiness criteria, hierarchy, sparse distributed-file ownership, typed relationships, lifecycle, 5W1H semantics, content-addressed freshness, and portable JSON graph contracts. This completes the semantic contract gate.

The implementation gate and first partial-bootstrap gate are complete. Exactly three ratified records cover the repository-root platform capability, shared engine foundation, and knowledge-session coordinator foundation, whose existing record now includes durable reconciliation. That bounded result does not authorize another distributed `FEATURES.md`, another canonical feature entry, a complete-catalog claim, a persistent graph store, or a Maestro receptor.

## Historical and Validator Boundary

Append-only SCLV and SODV records remain immutable. A contract transition changes prospective behavior and never rewrites earlier evidence. `symphony-validator` remains an independent, read-only checker. Shared authority-free mechanics may be extracted for static reuse only when the validator's Contract Quad, direct invocation, evidence semantics, and absence of remediation remain intact.

## Implemented Foundation and Vector Slices

`libraries/knowledge-vector-engine-cpp/` implements the authority-free bounded parser, framing, digest, no-follow path, file-read, snapshot, versioned CMake package, receipt, and uninstall mechanics. nlohmann/json `v3.12.0` is pinned and vendored with its official release checksum and MIT license; it is not a runtime download and is not linked into `symphony-validator`.

`modules/knowledge-session-coordinator/` implements process `inspect`, read-only snapshot `check`, compatibility negotiation, durable user-scope reconciliation `begin`, `status`, `checkpoint`, `close`, and `recover`, plus authenticated-session `session_begin`, `session_status`, `session_checkpoint`, `session_close`, and `session_recover`. Reconciliation uses per-worktree dual-slot journals; session coordination uses separate per-TOPS/subject/repository dual-slot authority-epoch journals. Both use expected-state generations, persistent no-follow locks, atomic head replacement, two-way procedural compatibility, extension preservation, idempotent operations, and bounded evidence-based recovery. qxctl authenticates SSIAG and supplies one fresh exact audited decision per session operation; the coordinator validates that evidence without independently deciding permission. It does not invoke a vector engine, mutate a worktree or canonical knowledge, directly call SSIAG/STAV, install a hook, run an observer, or dock with Maestro. Canonical apply remains disabled.

qxctl implements a protected user-scope `default` engine binding registry beneath its state root. It verifies exact inactive-undocked coordinator and vector-engine receipts and package files, records receipt and executable digests, serializes registry access with a persistent no-follow lock file, requires exact expected prior state, and commits replacements through a durable same-directory atomic rename. `list` and `inspect` report binding state; `doctor` revalidates the exact installation. `knowledge reconcile` revalidates one immutable binding snapshot, records its full engine inventory in protected noncanonical reconciliation state, and invokes only the exact bound coordinator. `knowledge session` reads one protected snapshot to resolve and revalidate the exact coordinator, but its separate authority-epoch journal does not record the reconciliation engine inventory. The session path additionally authenticates the configured SSIAG endpoint and validates one fresh exact decision. Binding itself does not invoke an engine, create a repository profile, establish authority, activate an install receipt, dock with Maestro, or enable apply.

`modules/skvi-engine/` implements deterministic `inspect`, structural `check`, caller-declared `propose`, and disposable JSON `project`. It parses repository-maintained `knowledge/skvi/INDEX.md`, rejects unsafe or ambiguous state, binds proposals and projections to canonical digests, installs under an exact versioned prefix, and exposes no canonical write path. `qxctl skvi ...` validates that exact inactive undocked installation and invokes it out of process with empty child environment, bounded input/output, a hard deadline, and response-digest/identity checks. qxctl lifecycle selection and activation remain deferred.

`modules/sclv-engine/` implements deterministic `inspect`, append-only v1/v2/v3 `check`, provider-neutral v3 `propose`, non-mutating ephemeral-journal `recover`, and disposable JSON `project`. Its separately discoverable local-Git and air-gapped adapter processes normalize bounded evidence but do not grant permission or ratify. The module installs three exact executables under one inactive-undocked eleven-file receipt. `qxctl sclv ...` validates that installation and applies the same empty-environment, deadline, response-identity, digest, and result-safety gates. The engine never appends, commits, deletes journals, edits provider state, or activates itself.

`modules/sacv-engine/` implements deterministic API-registry inspection/checks, bounded OpenAPI 3.2.0 JSON compatibility diffs, caller-declared registry proposals, and disposable inventories. YAML entry documents fail closed until a separate parser gate. `qxctl sacv ...` validates its exact inactive-undocked nine-file installation and invokes it under the common process-safety gates. The engine does not decide semantic ownership, expose endpoints, publish, generate bindings, or apply canonical changes.

`modules/sodv-engine/` implements deterministic append-only v1/v2 release-ledger checks, caller-supplied external-state verification, provider-neutral v2 release-record proposals, non-mutating interrupted-session recovery, and disposable release-transaction inventories. `qxctl sodv ...` validates its exact inactive-undocked nine-file installation and invokes it under the common process-safety gates. The engine has no network access and never creates or moves tags, contacts package providers, declares completion, mutates recovery journals, appends records, publishes, or applies canonical changes.

`modules/ssfv-engine/` implements deterministic `inspect`, structural and freshness-aware `check`, baseline-versus-live `diff`, caller-declared `propose`, and disposable JSON `graph`. It validates exact managed regions, namespaces, registry routing, record normalization, hierarchy, evidence paths, and SKVI coverage without deciding semantic truth. `qxctl ssfv ...` validates its exact inactive-undocked nine-file installation and adds no-follow baseline/input handling plus operation-specific authority and projection safety checks. The first three canonical records were authored and ratified through ordinary reviewed source changes; the engine and client never create feature records, apply proposals, persist graphs, activate a version, or dock with Maestro.

## Non-Authorization Statement

This specification does not claim implementation beyond the explicitly identified foundation/coordinator reconciliation and authenticated-session slices, explicit qxctl session transitions, SKVI/SCLV/SACV/SODV/SSFV slices, exact three-record SSFV partial bootstrap, user-default binding registry, and canonical lifecycle/receipt schemas; claim lifecycle persistence, observation, planning, recovery, or apply runtime; enable desired-state or first-boot lifecycle apply or canonical apply; authorize another feature record or complete-catalog claim; authorize an external package coordinate; create an HTTP surface; publish a release artifact; permit direct ledger mutation; or activate Maestro.
