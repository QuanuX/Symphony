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
- `lifecycle-desired-state.schema.json`: protected noncanonical exact component intent, dependency, compatibility, activation, and docking selection.
- `lifecycle-observation.schema.json`: disposable bounded inventory of configured roots, platforms, packages, component state, integrity, capabilities, and unknown preserved receipts without executable discovery.
- `lifecycle-plan.schema.json`: deterministic dependency-ready-set action graph, forward/inverse relationships, typed blockers, bounded dynamic replanning, and immutable safety-phase order.
- `lifecycle-applied-state.schema.json`: durable noncanonical last-verified component state and actual execution-order evidence anchored to exact desired, observation, plan, and transaction identities.
- `lifecycle-boot-journal.schema.json`: dual-read-compatible lifecycle transaction, replan, attempt, blocker, checkpoint, compatibility, and recovery evidence.
- `lifecycle-boot-head.schema.json`: atomic selector for the active member of a dual-slot lifecycle boot journal.

All schemas use JSON Schema Draft 2020-12, close every common-governed object with `additionalProperties: false`, and carry no secrets. The proposal operation's bounded `data` object is deliberately governed by the applicable vector schema; operation-specific payload/result schemas remain owned by that engine Contract Quad.

## Boundary

The binding schema authorizes only explicit user-scope selection among exact validated local installations. Session artifacts preserve SSIAG decision evidence but are not transferable bearer credentials and grant no canonical write authority by possession. Lifecycle schemas establish contract truth but do not implement persistence, observation, planning, recovery, or apply. These artifacts do not authorize canonical apply, network access, system/TOPS binding changes, repository-specific overrides, live Maestro docking, or any vector-specific semantic decision.
