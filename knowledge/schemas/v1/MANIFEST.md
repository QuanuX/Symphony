# Symphony Knowledge Vector Common Schemas v1

## Authority

These exact JSON Schema files are canonical common process and lifecycle contract truth owned by the `knowledge/` umbrella. Implementations remain subordinate to them.

## Schemas

- `engine-process-request.schema.json`: one bounded local process request envelope.
- `engine-process-response.schema.json`: one bounded local process response envelope.
- `engine-descriptor.schema.json`: installed engine/coordinator identity and capability truth.
- `qxctl-command-registry.schema.json`: canonical expected or exact observed qxctl executable-leaf identity, grammar, feature, backend, machine-output, and trust evidence.
- `feature-administration-profile.schema.json`: registered-feature interaction requirements, explicit dispositions, finite inheritance, and bootstrap forward-gate truth.
- `invariant-ownership-registry.schema.json`: lowest-authoritative-layer invariant IDs, owner contracts/components, producer and consumer regressions, real-process IPC evidence, and finite allowed versioned adapters.
- `invariant-query-result.schema.json`: bounded digest-bearing qxctl status, list, and show projections over the canonical invariant-ownership registry, with semantic validity explicitly reserved for the complete validator check.
- `administration-coverage-input.schema.json`: bounded repository-independent SSFV administration-check input with optional observed qxctl evidence.
- `administration-coverage-result.schema.json`: digest-bound design, live, authorization, uncovered-surface, remediation, and module-integration evidence.
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
- `ssfv-maintenance-command.schema.json`: exact authenticated command for persistent SSFV baseline, checkpoint, close, status, and forward-recovery operations.
- `ssfv-maintenance-journal.schema.json`: protected per-TOPS, per-subject, per-repository SSFV baseline and checkpoint lineage with separate baseline/current engine identities for upgrade-order tolerance.
- `ssfv-maintenance-head.schema.json`: atomic selector for the active member of a dual-slot SSFV maintenance journal.
- `ssfv-maintenance-result.schema.json`: bounded compatibility, review state, mutation, and repair evidence for the persistent SSFV maintenance stream.
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
- `lifecycle-root-ownership.schema.json`: protected root-local multi-profile package claims, conservative legacy preservation, explicit release evidence, enforcement state, and digest-linked generations for shared installation roots.
- `lifecycle-root-ownership-fence.schema.json`: static noncanonical receipt-layout compatibility evidence that makes ownership-unaware lifecycle clients preserve the root and fail closed before mutation.
- `lifecycle-root-ownership-result.schema.json`: bounded status, reconcile, adopt, and legacy-release result for one exact shared installation root.
- `lifecycle-root-ownership-reconciliation.schema.json`: deterministic per-profile collection of root ownership results used before reviewed lifecycle actions.
- `lifecycle-boot-journal.schema.json`: dual-read-compatible lifecycle transaction with authorization-bound profile digest, replan, attempt, blocker, checkpoint, compatibility, and recovery evidence.
- `lifecycle-boot-head.schema.json`: atomic selector for the active member of a dual-slot lifecycle boot journal.
- `lifecycle-boot-command.schema.json`: exact SSIAG-authorized boot, status, and recovery invocation contract, including expected-state, stable-inventory, and two-way client capabilities.
- `lifecycle-boot-result.schema.json`: strict report-only persisted-journal, optional plan, compatibility, mutation, recovery, and repair-evidence response.
- `lifecycle-apply-command.schema.json`: exact prepare, finalize, close, status, and recovery request contract for the separately authorized apply-capable v2 journal, including source-report, journal, applied-state, and artifact compare-and-swap evidence.
- `lifecycle-apply-result.schema.json`: bounded compatibility, active-action, plan, applied-state, mutation, and recovery result for the v2 lifecycle apply circuit.
- `lifecycle-host-integration.schema.json`: protected Linux systemd receptor descriptor binding exact roots, unit/executor digests, bounded accepted fallbacks, enablement/recovery intent, generation, predecessor, and STSC time.
- `lifecycle-host-integration-result.schema.json`: bounded install/update/status/reconcile/enable/disable/uninstall result with explicit drift, repair, recovery, and apply-disabled evidence.
- `lifecycle-host-boot-result.schema.json`: idempotent kernel-boot dispatch evidence binding the Linux boot UUID to one exact report-journal digest without apply authority.
- `maestro-receptor-descriptor.schema.json`: exact freezing-path receptor identity, limits, compatibility capabilities, and explicit no-execution/no-listener boundary.
- `maestro-docking-command.schema.json`: exact qxctl-to-Maestro inspect, authenticated status, lifecycle dock/undock, and recovery request contract.
- `maestro-docking-presence.schema.json`: one SSIAG-capability-bound component docking or undocking presence record.
- `maestro-docking-presence-registry.schema.json`: per-TOPS, per-receptor, digest-linked component presence generation.
- `maestro-docking-presence-head.schema.json`: atomic selector for the active member of a dual-slot presence registry.
- `maestro-docking-result.schema.json`: bounded compatibility, descriptor, presence, mutation, and forward-recovery result.
- `maestro-receptor-inventory-command.schema.json`: exact authenticated read-only request for a TOPS-wide derived receptor inventory.
- `maestro-receptor-inventory-result.schema.json`: deterministic stable receptor/component inventory plus a separately timestamped observation envelope.
- `temporal.schema.json`: reusable structural definitions for canonical STSC civil-date, whole-second UTC, and exact-nine-digit nanosecond UTC encodings; real Gregorian validation remains a required implementation conformance check.
- `validation-result.schema.json`: deterministic raw repository evidence and optional separately digested qxctl policy evaluation.
- `validation-policy.schema.json`: protected noncanonical warning disposition and presentation profile.
- `validation-baseline.schema.json`: protected noncanonical repository/version-bound warning identity inventory for delta evaluation.
- `validation-warning-state.schema.json`: protected noncanonical subject-aware warning lifecycle, occurrence/evidence history, administrative classifications, presentation-only mute state, and digest-linked transition chain.
- `foundation-lifecycle-adapter.schema.json`: exact installed SSIAG/STAV adapter identity, capabilities, compatibility, limits, and operation IDs.
- `foundation-lifecycle-command.schema.json`: bounded observe/plan/apply/apply-status/recover process command with caller intent and exact state.
- `foundation-lifecycle-observation.schema.json`: safe installation, enrollment, native-manager, endpoint, activation, and recovery evidence.
- `foundation-lifecycle-plan.schema.json`: immutable desired-state, identity, audit-mode, expiry, and expected-state proposal.
- `foundation-lifecycle-attempt.schema.json`: protected pre-mutation phase, predecessor, audit, and recovery evidence.
- `foundation-lifecycle-result.schema.json`: validated observation/plan/mutation/replay/recovery and audit disposition.

All schemas use JSON Schema Draft 2020-12, close every common-governed object with `additionalProperties: false`, and carry no secrets. The proposal operation's bounded `data` object is deliberately governed by the applicable vector schema; operation-specific payload/result schemas remain owned by that engine Contract Quad.

## Boundary

The host-integration schemas authorize only an explicit Linux systemd receptor for report-only boot planning; they grant no lifecycle apply, component execution, login/session hook, native Windows receptor, or hidden watcher. The separate foundational-lifecycle schemas authorize only exact installation-proven SSIAG/STAV module adapters and do not widen generic executable or service activation.

The binding schema authorizes only explicit user-scope selection among exact validated local installations. Session artifacts preserve SSIAG decision evidence but are not transferable bearer credentials and grant no canonical write authority by possession. qxctl implements protected desired-profile persistence, fixed-layout configured-root observation, report-only planner invocation, SSIAG-authorized lifecycle administration, and an explicit `apply-compatible` convergence surface for generic receipt-v2 packages, protected runtime state, and Maestro docking presence. The coordinator serializes attempts and commits applied evidence only after verified re-observation. These artifacts do not authorize canonical knowledge mutation, network package acquisition, receipt-v1 package mutation, arbitrary entry-point execution, system/TOPS engine-binding changes, repository-specific overrides, or vector-specific semantic decisions. The SSFV maintenance stream records exact read-only engine evidence and noncanonical review state; it never decides feature-worthiness or applies `FEATURES.md`. Maestro records authenticated durable presence for exact compatible receipt-v2 vector engines and derives inventory from that state; it does not start, supervise, schedule, or invoke them.
