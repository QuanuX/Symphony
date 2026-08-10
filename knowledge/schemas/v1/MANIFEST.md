# Symphony Knowledge Vector Common Schemas v1

## Authority

These exact JSON Schema files are canonical common process and lifecycle contract truth owned by the `knowledge/` umbrella. Implementations remain subordinate to them.

## Schemas

- `engine-process-request.schema.json`: one bounded local process request envelope.
- `engine-process-response.schema.json`: one bounded local process response envelope.
- `engine-descriptor.schema.json`: installed engine/coordinator identity and capability truth.
- `install-receipt.schema.json`: versioned, prefix-relative package ownership and docking state.
- `engine-binding-registry.schema.json`: protected, noncanonical user-default selection of exact inactive-undocked engine and coordinator installations. A binding is not installation, Maestro docking, authentication, permission, or canonical apply authority.
- `reconciliation-journal.schema.json`: protected noncanonical worktree state, content snapshots, checkpoint chain, compatibility envelope, extensions, and recovery evidence.
- `reconciliation-head.schema.json`: atomic selector for the active member of a dual-slot reconciliation journal.
- `reconciliation-command.schema.json`: exact qxctl-to-coordinator reconciliation operation payload.
- `reconciliation-result.schema.json`: bounded compatibility, state, mutation, recovery, and repair-guidance result.
- `session-command.schema.json`: exact qxctl-to-coordinator authenticated-session lifecycle request, including SSIAG authorization evidence and expected state.
- `session-journal.schema.json`: protected noncanonical authority-epoch state, capability and operation-fingerprint binding, linked epochs, attached reconciliation-context references, compatibility, extensions, and recovery evidence.
- `session-head.schema.json`: atomic selector for the active member of a dual-slot authenticated-session journal.
- `session-result.schema.json`: bounded authenticated-session compatibility, effective state, mutation, recovery, and repair-guidance result.
- `session-transition-result.schema.json`: qxctl's digest-bound noncanonical result for one explicit idempotent login, refresh, or logout convergence event.
- `proposal.schema.json`: provider-neutral immutable proposal envelope and vector-neutral authority boundary. Its explicit `engine_decided_domain_truth: false` assertion prevents any engine from converting validation into ownership, membership, ratification, publication, or other semantic authority.
- `provider-evidence.schema.json`: bounded provider-neutral revision, change-request, and ratification evidence normalized by separately discoverable adapters.
- `lifecycle-profile-input.schema.json`: bounded declarative profile intent accepted by qxctl without caller-authored generations, predecessor links, or generated digests.
- `lifecycle-profile.schema.json`: protected per-TOPS selected-root and desired-state profile with exact generation, predecessor, canonical-false, and content-digest evidence.
- `lifecycle-desired-state.schema.json`: protected noncanonical exact component intent, dependency, compatibility, activation, and docking selection.
- `lifecycle-observation.schema.json`: disposable bounded inventory of configured roots, platforms, packages, component state, integrity, capabilities, and unknown preserved receipts without executable discovery.
- `lifecycle-plan-command.schema.json`: exact report-only caller-to-coordinator desired/observation evidence and explicit protocol, receipt-reader, and planner-capability declaration.
- `lifecycle-plan.schema.json`: deterministic dependency-ready-set action graph, forward/inverse relationships, exact receptor and target-state binding, typed blockers, noncritical dependency advisories, bounded dynamic replanning, and immutable safety-phase order.
- `lifecycle-applied-state.schema.json`: durable noncanonical last-verified component state and actual execution-order evidence anchored to exact desired, observation, plan, and transaction identities.
- `lifecycle-runtime-state.schema.json`: protected qxctl-owned exact package selection and administrative activation evidence for generic lifecycle components; docking remains closed to `undocked`.
- `lifecycle-boot-journal.schema.json`: dual-read-compatible lifecycle transaction with authorization-bound profile digest, replan, attempt, blocker, checkpoint, compatibility, and recovery evidence.
- `lifecycle-boot-head.schema.json`: atomic selector for the active member of a dual-slot lifecycle boot journal.
- `lifecycle-boot-command.schema.json`: exact SSIAG-authorized boot, status, and recovery invocation contract, including expected-state, stable-inventory, and two-way client capabilities.
- `lifecycle-boot-result.schema.json`: strict report-only persisted-journal, optional plan, compatibility, mutation, recovery, and repair-evidence response.
- `lifecycle-apply-command.schema.json`: exact prepare, finalize, close, status, and recovery request contract for the separately authorized apply-capable v2 journal, including source-report, journal, applied-state, and artifact compare-and-swap evidence.
- `lifecycle-apply-result.schema.json`: bounded compatibility, active-action, plan, applied-state, mutation, and recovery result for the v2 lifecycle apply circuit.
- `temporal.schema.json`: reusable structural definitions for canonical STSC civil-date, whole-second UTC, and exact-nine-digit nanosecond UTC encodings; real Gregorian validation remains a required implementation conformance check.

All schemas use JSON Schema Draft 2020-12, close every common-governed object with `additionalProperties: false`, and carry no secrets. The proposal operation's bounded `data` object is deliberately governed by the applicable vector schema; operation-specific payload/result schemas remain owned by that engine Contract Quad.

## Boundary

The binding schema authorizes only explicit user-scope selection among exact validated local installations. Session artifacts preserve SSIAG decision evidence but are not transferable bearer credentials and grant no canonical write authority by possession. qxctl implements protected desired-profile persistence, fixed-layout configured-root observation, report-only planner invocation, SSIAG-authorized lifecycle boot/status/recovery administration, and an explicit `apply-compatible` convergence surface for generic receipt-v2 packages and protected runtime selection/activation state. The coordinator receives complete caller-supplied evidence, serializes action attempts, and commits content-addressed applied evidence only after qxctl re-observes and proves the exact action outcome. These artifacts do not authorize canonical apply, network package acquisition, receipt-v1 mutation, arbitrary entry-point execution, system/TOPS engine-binding changes, repository-specific overrides, live Maestro docking, or any vector-specific semantic decision.
