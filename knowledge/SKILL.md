# Symphony Knowledge Vector Engine Skill

## Purpose

Guide every caller in safely inspecting, proposing, implementing, installing, and reviewing SKV vector-engine changes without displacing vector ownership or manufacturing authority.

## Required Reading Order

1. `knowledge/INTENT.md`
2. `knowledge/MANIFEST.md`
3. `knowledge/SPEC.md`
4. `knowledge/INVARIANTS.md` and `knowledge/INVARIANT-OWNERSHIP.json` when a rule crosses an implementation or process boundary
5. `knowledge/TIME.md` when a field, deadline, freshness rule, journal, or durable event involves time
6. `knowledge/FOUNDATIONAL-LIFECYCLE.md` before SSIAG/STAV enrollment or supervision administration
7. `knowledge/VALIDATION.md` when validator evidence, warning policy, baselines, or debug filtering is involved
8. the affected vector's Contract Quad
9. `tools/qxctl/` contracts for administrative grammar
10. `knowledge/ssiag/SPEC.md` before any apply or safeguard work
11. `knowledge/stav/SPEC.md` before any audited outcome or recovery work
12. `knowledge/sodv/SPEC.md` before release or publication

## Safe Initial Operations

After this contract transition is merged, authorized implementation work may:

- build the authority-free shared C++ foundation and coordinator read path;
- inspect bounded repository and provider inputs;
- bind one exact inactive-undocked coordinator or vector-engine installation per role in the protected qxctl user-default profile;
- begin, inspect, checkpoint, close, and recover a protected noncanonical worktree reconciliation context through the exact bound coordinator;
- compute content digests and deterministic validation evidence;
- maintain noncanonical authenticated-session, worktree, report-only lifecycle, and separately authorized apply-capable lifecycle journals;
- maintain a separate noncanonical SSFV semantic baseline/checkpoint stream during an open authenticated session and attach explicit complete or not-configured Maestro inventory evidence;
- create immutable proposals;
- produce vector-authorized disposable projections;
- expose implemented proposal/read operations through qxctl;
- prove independent install/uninstall without silently docking or mutating canonical files.

The implemented `0.1.0-dev` foundation supports direct coordinator `inspect`, explicit-path read-only `check`, durable reconciliation, SSIAG-authorized noncanonical authenticated sessions, report-only `lifecycle_plan`, protected report-only `lifecycle_boot|lifecycle_boot_status|lifecycle_boot_recover`, and apply coordination `lifecycle_apply_prepare|lifecycle_apply_finalize|lifecycle_apply_close|lifecycle_apply_status|lifecycle_apply_recover` over fully supplied desired/observed evidence. The independently installed SKVI, SCLV, SACV, SODV, and SSFV engines retain their ratified bounded check/proposal/projection operations. The independently installed Maestro process implements receptor inspection plus authenticated status, presence mutation, and bounded recovery. qxctl validates exact receipts, owned paths, process identities, deadlines, and response digests; it also implements protected profiles/runtime state, fixed-layout receipt plus Maestro-presence observation, fresh SSIAG authorization, report-only planning, staged receipt-v2 actions, lifecycle-only dock/undock, and durable journal administration. Apply capability is noncanonical and operation-bound; it grants no vector ratification, canonical endpoint publication, canonical knowledge mutation, or live process activation.

Use `qxctl knowledge session transition --event login|refresh|logout --event-id ID` only when an explicit host lifecycle integration supplies a stable event identity. Safe retries reuse the same event ID. Add `--recover` only when discovery recovery from damaged local session evidence is intended; it does not recover denial, incompatible critical state, or ambiguity. Symphony does not install a login hook, watcher, or boot unit through this command.

For modular installation planning, read `knowledge/LIFECYCLE.md` with this Contract Quad and the exact common lifecycle schemas. Preserve binding registry v1, receipt v1, and report-journal v1 evidence exactly. Treat profile input, desired, observed, runtime, planned, report-journal, apply-journal, and applied state as separate noncanonical evidence. A new or missing module is a plan input, not implicit permission to execute, remove, upgrade, downgrade, bind, or dock it. Use qxctl to generate protected state and observe configured receipt roots; do not hand-edit it. The coordinator consumes complete digest-bound desired/observed evidence and derives component action order only from the explicit dependency ready set. Preserve blockers, replan only after verified evidence changes, bind dock actions to exact receptor identities, and never reorder the enclosing safety phases.

For SSIAG or STAV enrollment and native supervision, use the separate `knowledge/FOUNDATIONAL-LIFECYCLE.md` envelope. Invoke only the exact installation-proven module adapter; never import module internals, parse its human output, render descriptors in qxctl, invoke `serve`, select an implicit newest version, or bypass a protected attempt. Use status before plan, apply only an exact unexpired plan, use apply-status for its attempt, and recover only a unique digest-linked transition. Audit-deferred bootstrap must be explicit and remains reconciliation-required until the closed SSIAG producer binds a STAV receipt.

Use `qxctl knowledge lifecycle profile set --tops-id UUID --input FILE --expected-profile-digest absent|DIGEST` for exact profile compare-and-swap. Use `ownership status|reconcile` to inspect or refresh one configured shared root; use `ownership adopt` only after reviewing every conservative legacy claim, and `ownership release` only for one exact legacy receipt intentionally made reclaimable. Never delete or rewrite the root-local registry or its `symphony-root-ownership/1` receipt-layout compatibility fence by hand; older lifecycle clients must encounter that fence and stop on their existing unknown-package blocker. Use `observe` for disposable fixed-layout receipt evidence, `report` for a fresh disposable plan, and `boot` with a stable operation ID plus exact journal state for durable report-only progression. Use `status` for v1 inspection and `recover --discover` only for one unique v1 chain. For an `apply-compatible` profile, call `apply` only with the exact boot source digest, apply-journal compare-and-swap, applied-state compare-and-swap, stable operation ID, and explicit staged roots. Use `apply-status` or bounded `apply-recover`; never choose an action manually or edit protected state. A timestamp-only observation refresh must change document evidence without advancing stable semantic identities.

The implemented `qxctl knowledge engines list|inspect|doctor|bind|unbind` surface manages only the protected user-scope `default` binding profile. Supply `absent` for the first expected registry state or the exact digest reported by `list` for later mutations. A bind selects exact content for later reconciliation; it does not install, invoke, activate, dock, authenticate, authorize, or apply.

The implemented `qxctl knowledge reconcile compatibility|begin|status|checkpoint|close|recover` surface uses one exact bound coordinator. Supply a stable operation identifier for every mutation and the exact current journal digest; use discovery recovery only when ordinary status cannot validate local state. Treat two-slot recovery as evidence-based forward repair, never permission to discard an incompatible journal or unknown critical extension.

The implemented `qxctl knowledge session begin|status|checkpoint|close|recover` surface uses the same exact bound coordinator and one TOPS-scoped SSIAG endpoint. Every call requests a fresh exact grant and validates subject, target, policy/configuration, expiry, caller-class neutrality, non-transferability, canonical-apply disablement, and capability binding. Session evidence authorizes only the named protected noncanonical operation. Keep session and reconciliation journals separate and use context references only to associate them.

Use `qxctl knowledge session features begin|status|checkpoint|close|recover` for the separate persistent SSFV maintenance stream. Mutations require an open authenticated session, exact coordinator/SSFV bindings, stable operation IDs, exact compare-and-swap, and fresh SSIAG evidence. Preserve the baseline snapshot and its original engine identity across compatible upgrades or rollbacks. Optional Maestro evidence must be a complete authenticated inventory; when Maestro is intentionally absent, record the explicit not-configured reason. `review_required` is evidence for caller review, not feature-worthiness, ratification, or apply authority.

## Prohibited Initial Operations

- Do not apply a proposal programmatically.
- Do not create or rewrite canonical Markdown from an engine.
- Do not treat a generated fact or proposal as permission-backed ratification.
- Do not install Git hooks implicitly.
- Do not pass secrets through JSON, process arguments, environment variables, logs, proposals, projections, or fixtures.
- Do not add HTTP merely to avoid the standard-I/O process contract.
- Do not make GitHub, GitLab, Mintlify, NotebookLM, a package registry, or Maestro required for canonical truth.
- Do not implement the SSFV engine or bootstrap distributed feature records outside the now-ratified SSFV Contract Quad and their separate reviewed slices.
- Do not put administrative reconciliation or audit recovery on a hot or warm path.

## Session and Recovery Procedure

1. Use `qxctl knowledge session begin` to obtain exact SSIAG authorization and establish a noncanonical authority epoch when an authority-bearing operation requires one; reconciliation alone makes no authentication claim.
2. Open or resume a separately locked reconciliation context for each worktree.
3. Capture canonical contract, tree, and engine digests.
4. Treat observer and hook input only as a dirty-path hint.
5. Checkpoint or close through bounded reconciliation.
6. Emit a deterministic no-op or proposal for each affected vector.
7. On interruption, retain only safe noncanonical journal state.
8. Require fresh authentication when recovery crosses logout, expiry, revocation, or required re-authentication.

## Review Rules

- Verify every new installable module, adapter entry point, backend operation, and administrator-facing interaction has a same-change stable feature declaration plus qxctl/backend mapping or explicit evidence-backed disposition; absence is uncovered.
- Verify each affected `invariant:` record resolves to one owner, existing implementation and test paths, owner producer regression, consumer boundary rejection, and only allowed versioned adapters. For IPC, run the real receipt-backed process regression rather than accepting only an in-process codec test.
- Verify exact process and message bounds before calling a protocol operational.
- Verify path ownership, symlink handling, special-file rejection, expected-state binding, and stable output.
- Verify separate worktrees never share a writer lock or mutable journal.
- Verify every projection identifies canonical input and engine digests.
- Verify qxctl reports unimplemented reserved commands honestly.
- Verify install and uninstall use receipts and preserve unrelated versions and user-owned files.
- Verify lifecycle plans support both forward and inverse actions, continue unrelated ready work around localized blockers, isolate cycles, preserve stable semantic action IDs, and enforce the ratified action/replan/attempt bounds.
- Verify cold/freezing administration has no inline call, lock, or synchronous dependency on hot/warm execution.
- Verify canonical timestamps use the applicable STSC profile, impossible Gregorian dates fail closed, and wall-clock text is not substituted for causal sequence or identity.

## Stop Conditions

Stop for permission-backed Architect review before changing a cleared namespace, enabling apply, adding an external package coordinate, changing protocol major version, enabling a network listener, selecting a Maestro receptor contract, implementing the SSFV engine, bootstrapping SSFV feature records, weakening protocol integrity, or introducing any hot/warm dependency.
