# Symphony Knowledge Vector Engine Foundation Specification

## Status and Normative Terms

Architect-ratified cross-vector architecture with the explicitly bounded `0.1.0-dev` foundation/coordinator read-only slice and SKVI/SCLV proposal/projection slices implemented. MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative when the related implementation exists. No later session-mutation, other vector-engine, lifecycle, apply, or docking capability may be inferred from these slices.

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
| `symphony.knowledge.session-journal.v1` | authenticated-session/worktree recovery journal |
| `symphony.knowledge.provider-evidence.v1` | normalized provider evidence |
| `symphony.knowledge.install-receipt.v1` | installed-file and lifecycle receipt |
| `symphony.maestro.knowledge-engine-docking.v1` | Maestro docking projection |

The initial exact schemas are:

- `knowledge/schemas/v1/engine-process-request.schema.json`;
- `knowledge/schemas/v1/engine-process-response.schema.json`;
- `knowledge/schemas/v1/engine-descriptor.schema.json`;
- `knowledge/schemas/v1/install-receipt.schema.json`.

The process request limit is 1 MiB and the response limit is 4 MiB. JSON depth is at most 64, parsed values/events at most 16,384, one string or key at most 65,536 bytes, integers remain within `[-9007199254740991, 9007199254740991]`, and a request deadline is at most 300 seconds ahead. Unknown fields, duplicate names, invalid UTF-8, trailing data, floating-point values, out-of-range integers, unsupported versions, excessive input, unsafe paths, expired deadlines, and target mismatch fail closed. Standard output is reserved for the single protocol response; bounded diagnostics use standard error. Arguments and environment variables MUST NOT carry secrets or arbitrary executable instructions.

An engine checks the deadline before and between bounded work units and file-read chunks. The invoking process MUST independently enforce the same deadline on child-process lifetime so a blocked operating-system or filesystem call cannot outlive the request. The direct coordinator slice provides cooperative checks; the implemented qxctl SKVI/SCLV client adds a hard child-process timeout around each request deadline.

`response_digest` is the tagged SHA-256 of the compact key-sorted response object before that member is inserted. Operation-specific payload/result schemas remain owned by the applicable coordinator or vector contract.

## Authenticated Session Model

The default knowledge session is an authenticated authority epoch. It begins after successful authentication and ends at logout, session or credential expiry, revocation, or a boundary requiring re-authentication. Re-authentication creates a new authority epoch.

An administrator MAY select another supported session-lifecycle policy through qxctl. Configuration MUST NOT extend authority beyond logout, expiry, revocation, or required re-authentication. Product windows, shells, assistant interactions, and forge requests may bind to a session but do not independently create authority.

## Worktree Reconciliation Context

One authenticated session may contain multiple worktree-scoped reconciliation contexts. Each context owns its repository/worktree identity, initial and current content digests, vector-contract digests, engine inventory, journal, observer hints, and writer/reconciliation lock.

Separate worktrees MUST NOT share a mutable journal or writer lock. VCS merge is their cross-worktree reconciliation boundary. Absolute paths may appear only in protected local state and never in canonical records or portable proposals.

Correctness comes from content-addressed snapshots at begin, checkpoint, close, pre-apply, post-apply stabilization, and next-authenticated-session recovery. Filesystem notifications, IDE callbacks, forge events, and Git hooks are optional latency hints. A missed event MUST self-heal through the next bounded digest comparison.

Hooks are explicit, removable, and never installed silently. A disposable observer MAY run only for an active reconciliation context and MUST terminate at close. Neither is required for correctness.

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

## Machine-Managed and Semantic Content

An engine MAY compute deterministic facts and propose bounded machine-managed fields or sections when the owning vector contract defines their markers and formatting. Semantic claims remain caller-declared or caller-proposed until a caller holding the required permission ratifies them.

An engine MUST preserve unknown and owner-controlled content. Ambiguous ownership markers, overlapping writes, unexpected prior digests, or unstable repeated output fail closed. Generation method is not an authority category: computed content may become canonical only through the same ratified path, while manually written content is not correct merely because a caller typed it.

## Provider-Neutral Evidence

No engine may require GitHub, GitLab, Mintlify, NotebookLM, or another external provider to preserve canonical truth. Adapters are separately discoverable processes using bounded protocol requests. Local and air-gapped evidence is first-class. Git is the first repository substrate, not a universal identity model; revisions declare their scheme and are never universally assumed to be 40-character SHA-1 values.

## qxctl Grammar

The cross-vector groups are:

```text
qxctl knowledge engines ...
qxctl knowledge session ...
qxctl knowledge proposals ...
qxctl knowledge apply ...        # reserved; disabled until the apply gate passes
```

Vector-specific groups are:

```text
qxctl skvi ...
qxctl sclv ...
qxctl sacv ...
qxctl sodv ...
qxctl ssfv ...                   # reserved until the SSFV Contract Quad gate passes
```

qxctl MUST resolve exact installed engine identities and protocol compatibility from trusted receipts. Direct engine invocation remains available for diagnostics and conformance. qxctl MUST NOT absorb vector semantics, classify callers, accept secret-bearing engine input, or present a reserved command as operational.

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

Every engine and the coordinator MUST support independent install, upgrade, rollback, and uninstall. Installation never silently changes repository hooks, canonical files, active engine bindings, or Maestro state. Uninstall removes only files owned by the selected receipt and preserves canonical knowledge and session/recovery evidence.

Every package declares a default receptor that an administrator may change through qxctl. Installation without Maestro is valid and reports `installed_undocked`. Multiple compatible versions may coexist, dock, undock, and activate under explicit administrator selection. A newer installation MUST NOT silently replace the active version.

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
6. SSFV only after its separate Contract Quad gate.

Scaffolding every engine in advance is prohibited. Each slice must pass its contract, conformance, receipt, and uninstall gates before the next vector claims implementation.

## SSFV Gate

The SSFV namespace is reserved, but no SSFV engine or `FEATURES.md` generation is authorized until the Architect ratifies its Contract Quad, stable feature identifiers, feature-worthiness criteria, hierarchy, distributed-file ownership, relationship vocabulary, and graph-projection contract.

## Historical and Validator Boundary

Append-only SCLV and SODV records remain immutable. A contract transition changes prospective behavior and never rewrites earlier evidence. `symphony-validator` remains an independent, read-only checker. Shared authority-free mechanics may be extracted for static reuse only when the validator's Contract Quad, direct invocation, evidence semantics, and absence of remediation remain intact.

## Implemented Foundation, SKVI, and SCLV Slices

`libraries/knowledge-vector-engine-cpp/` implements the authority-free bounded parser, framing, digest, no-follow path, file-read, snapshot, versioned CMake package, receipt, and uninstall mechanics. nlohmann/json `v3.12.0` is pinned and vendored with its official release checksum and MIT license; it is not a runtime download and is not linked into `symphony-validator`.

`modules/knowledge-session-coordinator/` implements process `inspect` and read-only snapshot `check` only. It does not yet establish an authenticated session, persist a journal, acquire a reconciliation lock, invoke a vector engine, mutate a worktree, integrate qxctl/SSIAG/STAV, activate an installed version, or dock with Maestro. Descriptor-visible lifecycle operations are reserved and apply is disabled.

`modules/skvi-engine/` implements deterministic `inspect`, structural `check`, caller-declared `propose`, and disposable JSON `project`. It parses repository-maintained `knowledge/skvi/INDEX.md`, rejects unsafe or ambiguous state, binds proposals and projections to canonical digests, installs under an exact versioned prefix, and exposes no canonical write path. `qxctl skvi ...` validates that exact inactive undocked installation and invokes it out of process with empty child environment, bounded input/output, a hard deadline, and response-digest/identity checks. qxctl lifecycle selection and activation remain deferred.

`modules/sclv-engine/` implements deterministic `inspect`, append-only v1/v2/v3 `check`, provider-neutral v3 `propose`, non-mutating ephemeral-journal `recover`, and disposable JSON `project`. Its separately discoverable local-Git and air-gapped adapter processes normalize bounded evidence but do not grant permission or ratify. The module installs three exact executables under one inactive-undocked eleven-file receipt. `qxctl sclv ...` validates that installation and applies the same empty-environment, deadline, response-identity, digest, and result-safety gates. The engine never appends, commits, deletes journals, edits provider state, or activates itself.

## Non-Authorization Statement

This specification does not claim implementation beyond the explicitly identified foundation/coordinator, SKVI, and SCLV slices, enable canonical apply, authorize an external package coordinate, create an HTTP surface, publish a release artifact, permit direct ledger mutation, activate Maestro, or authorize SSFV semantics.
