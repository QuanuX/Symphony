# SSIAG Provider Binding Lifecycle

## Status

Architect-ratified Phase 10A canonical contract. This contract enables exact provider-installation inventory and protected metadata-binding administration. It does not enable Keychain access, credential operations, leases, provider payloads, or secret delivery.

## Authority and Ownership

`knowledge/ssiag/` owns provider-binding semantics and schemas. The Go SSIAG foundation owns protected per-TOPS state, observation, planning, compare-and-swap, audit, commit, and recovery. A provider adapter owns only its platform implementation and declared adapter operations. qxctl is the preferred headless administrative client and does not inspect receipts, select packages, execute an adapter, or own these schemas. STAV owns committed runtime audit evidence. SCLV records only completed repository change truth. SSFV records implemented application capability and administration coverage.

This is an SSIAG domain lifecycle, not a generic vector-engine lifecycle, a foundational enrollment/supervision lifecycle, or a Maestro docking lifecycle. Reusing another lifecycle's authority would duplicate ownership and permit unrelated components to mutate SSIAG security state.

## Stable Administrative Identities

The six qxctl identities are:

- `qxcmd:symphony:ssiag.provider.installations` for `qxctl ssiag provider installations <provider-name>`;
- `qxcmd:symphony:ssiag.provider.binding.status`;
- `qxcmd:symphony:ssiag.provider.binding.plan`;
- `qxcmd:symphony:ssiag.provider.binding.apply`;
- `qxcmd:symphony:ssiag.provider.binding.apply-status`; and
- `qxcmd:symphony:ssiag.provider.binding.recover`.

The six independently owned foundation-operation identities are:

- `engop:symphony:ssiag.provider.installations.list`;
- `engop:symphony:ssiag.provider.binding.observe`;
- `engop:symphony:ssiag.provider.binding.plan`;
- `engop:symphony:ssiag.provider.binding.apply`;
- `engop:symphony:ssiag.provider.binding.apply-status`; and
- `engop:symphony:ssiag.provider.binding.recover`.

Status binds `observe`; command and backend identities remain distinct. Mutation permission uses the caller operations `symphony.ssiag.provider.binding.plan`, `.apply`, `.apply-status`, and `.recover` over the exact resource `symphony.ssiag.provider-binding:<tops-id>:<provider-name>`, audience `ssiag`, and scope `tops:<tops-id>`. Inventory and status are safe inspection; any policy decision required by a deployment remains SSIAG-owned.

The frozen HTTP surface is:

- `GET /v1/provider-installations/{provider}` for exact bounded inventory;
- `GET /v1/provider-bindings/{provider}` for protected binding status;
- `POST /v1/provider-bindings/{provider}/plans` with exactly `installation_id`, `expected_state_digest`, and `reason`;
- `POST /v1/provider-bindings/{provider}/apply` with exactly `plan_digest` and `expected_state_digest`;
- `GET /v1/provider-bindings/{provider}/attempts/{operation_id}` for apply status; and
- `POST /v1/provider-bindings/{provider}/recover` with exactly `expected_state_digest` and `reason`.

Request objects have no compatibility envelope and reject all unknown members. `absent` is the compare-and-swap bootstrap sentinel. `not_applicable` is the explicit absence representation for an installation, previous installation, operation, or attempt digest; these protocols use no implicit omission or null absence. The only nullable members are the three attempt-stage timestamps, which are explicitly null until their corresponding stages complete. Administrative reasons are one line of 1 through 1024 UTF-8 characters and exclude NUL, carriage return, and line feed.

### Offline receipt-bound recovery

Normal inventory and binding administration remains kernel-authenticated over the enrolled SSIAG Unix socket. The only offline lane is the installed foundation command:

`symphony-ssiag provider-binding-recover --scope user|system --tops-id UUID --provider NAME --expected-state-digest DIGEST|absent --reason TEXT [--json]`

It uses the same `engop:symphony:ssiag.provider.binding.recover` backend operation, `symphony.ssiag.provider.binding.recover` permission, provider-binding resource, manager, recovery request, and `provider-binding-result.v1` result with `operation: recover`. It creates no qxcmd identity and is not a qxctl bypass. Before state access it requires the exact running foundation to be receipt-v2 installed, target-host ownership, entry into the enrolled SSIAG service UID/GID for the target TOPS, exclusive ownership of the same persistent socket-lifecycle lease used by the service, and the live SSIAG socket to be absent while that lease remains held. It must then satisfy the ordinary exact expected-state, candidate-verification, and committed STAV gates. A running service, missing or mismatched receipt, mismatched identity, ambiguous attempt, changed evidence, unavailable audit, or failed compare-and-swap fails closed without modifying state.

## Canonical Protocol Inventory

Phase 10A owns the following strict JSON protocols:

| Protocol | Canonical schema |
|---|---|
| `symphony.ssiag.provider-installation-inventory.v1` | `schemas/v1/provider-installation-inventory.schema.json` |
| `symphony.ssiag.provider-binding-status.v1` | `schemas/v1/provider-binding-status.schema.json` |
| `symphony.ssiag.provider-binding-plan-request.v1` | `schemas/v1/provider-binding-plan-request.schema.json` |
| `symphony.ssiag.provider-binding-plan.v1` | `schemas/v1/provider-binding-plan.schema.json` |
| `symphony.ssiag.provider-binding-apply-request.v1` | `schemas/v1/provider-binding-apply-request.schema.json` |
| `symphony.ssiag.provider-binding-recovery-request.v1` | `schemas/v1/provider-binding-recovery-request.schema.json` |
| `symphony.ssiag.provider-binding-result.v1` | `schemas/v1/provider-binding-result.schema.json` |
| `symphony.ssiag.provider-binding-state.v1` | `schemas/v1/provider-binding-state.schema.json` |
| `symphony.ssiag.provider-binding-attempt.v1` | `schemas/v1/provider-binding-attempt.schema.json` |

Every member is explicit. Unknown, duplicate, missing, trailing, oversized, noncanonical, symlinked, or unsafe evidence fails closed. Omit-self digests use compact UTF-8 JSON with recursive lexical object-key order and array order preserved. Arrays documented as sorted and unique must already be canonical before hashing.

## Installation Inventory

Inventory is bounded observation, never selection authority. It may inspect only the scope's admitted installation root and an exact enrolled legacy declaration's containing installation prefix. The inventory request supplies no root or path. It must not scan arbitrary filesystem locations, infer authority from directory order, or select the newest semantic version.

Each candidate binds a complete `provider-executable-trust.v1` declaration to observed receipt and compatibility evidence. Candidate order is lexical by declaration digest. The 128-entry inventory bound is cumulative across every admitted root. Inventory reports incompatibility and ambiguity rather than silently dropping evidence. All operational, provider-operation, and secret-channel flags remain false.

## Binding State

The protected state is separate from the Phase 9 single-declaration compatibility file. The inventory contains the finite set of exact declaration pairs. State names one explicit active installation or `not_applicable`, one retained previous installation or `not_applicable`, a generation, previous-state digest, and state digest. An installation binds both the exact foundation executable and exact adapter executable through its verified receipt declarations; compatibility never follows version recency.

Lifecycle-capable foundations use the protected state. A Phase 9 binary continues to read only its exact single-declaration file and therefore remains metadata-only and deny-by-default. New state must not be written into the old file in a shape an old binary could partially accept. A legacy declaration may be imported only through an explicit plan and remains compatibility evidence until separately retired.

At most one installation may be active. The active binding digest must name it exactly. The retained previous binding is rollback evidence, not an automatic fallback. Inventory entries are sorted lexically by opaque installation ID and unique. If more than one exact binding could satisfy the executing foundation, selection is ambiguous and fails closed.

## Two-Way Upgrade and Recovery

Multiple immutable foundation and adapter versions may coexist. A plan is a dependency graph, not a timestamp-ordered script. It may move forward or backward only across exact pairs already present in the request and proven compatible by receipt, protocol, capability, platform, and executable evidence. Timestamps, modification time, semantic newest, and filesystem enumeration never choose direction.

The planner dynamically chooses a valid action order from observed evidence. Installing the adapter first, installing the foundation first, or staging both before activation must converge to the same exact desired pair when the evidence graph permits it. A localized incompatible pair remains a blocker; it must not force unrelated state mutation or a weaker fallback.

Apply persists a `prepared` attempt before mutation, independently verifies every target receipt, executable, and metadata handshake before persisting `candidate_verified`, requires a committed STAV receipt before persisting `audited`, rechecks the exact static candidate evidence under the serialized binding lock, atomically replaces state under exact compare-and-swap while the attempt remains `audited`, and only then persists the `committed` post-commit marker before writing the result and removing the attempt. Recovery advances only the uniquely linked evidence-supported successor through `prepared -> candidate_verified -> audited -> committed`; it may never skip, reorder, infer, or silently repeat a stage. Any still-present attempt, including a post-state `audited` or `committed` attempt awaiting result/cleanup, remains `recovery_required` to status clients. A retained previous pair may be restored only when the plan explicitly authorized it and its exact evidence still matches. Ambiguity, changed bytes, unknown critical state, or incompatible evidence remains `recovery_required`.

The prepared attempt also binds the initiating safe actor ID, actor kind, and authentication method. These are audit-candidate identity, not authorization for recovery. Every recovering caller must still satisfy current target-host ownership or the exact current permission, but STAV submission always reconstructs the original candidate identity. A crash after the append authority commits and before SSIAG persists `audited` can therefore replay the same request ID and candidate digest idempotently even when another administrator or the receipt-bound offline lane performs recovery.

Unknown critical future state is preserved read-only and never downgraded. Known noncritical extensions may be preserved byte-for-byte by a future format, but v1 defines no extension bag. An old reader therefore fails closed rather than rewriting a newer state.

## STAV Binding Lifecycle

Provider binding mutation is not provider execution. SSIAG submits the distinct safe pair:

- event class `symphony.ssiag.provider.binding.lifecycle`;
- operation ID `symphony.ssiag.provider.binding.change`;
- intent ID `symphony.ssiag.provider.binding.change`.

Outcomes `succeeded`, `failed`, and `unavailable` map to `symphony.ssiag.provider.binding.<outcome>`. The target is only the safe provider-binding reference. Configuration carries the previous and new state digests. Paths, receipts, executable digests, policy bodies, native errors, credential references, Keychain item identifiers, provider payloads, proofs, and secret values are excluded.

STAV requires tagged previous and new digests. On the first binding, `target_state.previous_state_digest` is the literal `absent`; SSIAG deterministically substitutes the tagged SHA-256 of compact canonical JSON `{protocol:"symphony.ssiag.provider-binding-absence.v1",tops_id,scope,provider_name,provider_kind}` as the safe audit `previous_digest`. Every later transition uses the prior protected `state_digest` directly. The absence anchor is derived audit evidence only: it is not persisted state, a new wire protocol, or another schema.

The installation's SSIAG producer grant must explicitly admit this exact pair. qxctl never receives producer authority. No audit-deferred provider-binding mutation is enabled in Phase 10A.

## Operational Prohibitions

Every Phase 10A inventory, status, plan, result, state, and attempt reports:

- `operational_access_enabled: false`;
- `provider_operations_enabled: false`; and
- `secret_channel_enabled: false`.

Planning or applying a binding proves only exact metadata compatibility. It cannot create, read, update, rotate, delete, sign with, decrypt with, assert with, export, lease, or deliver Keychain material. The synthetic Phase 9 descriptor remains unopened. System and headless scope remain unavailable for the login-Keychain provider and never fall back.

## Validation and Evidence

Repository validation checks exact schema authorization, SKVI routing, SSFV and qxctl identity closure, invariant ownership, digest shape, and real implementation/test references. It does not decide which adapter is trustworthy or which pair should activate.

The implementation gate requires permutations for adapter-first, foundation-first, both-staged, rollback, missing package, changed receipt, changed bytes, incompatible protocol, ambiguous pair, stale compare-and-swap, expired issued plan, competing attempt, crash at every attempt stage, unknown state, and old-reader preservation. Real receipt-backed Go-to-native process evidence remains mandatory. Secret-marker tests must prove that control, qxctl, STAV, logs, arguments, environment, and fixtures contain no secret values.

## Deferred Gates

The provider-operation catalog, Keychain namespace and access-control matrix, non-exportable item lifecycle, signing/assertion/decryption requests, leases, operational result transport, production signing identity, entitlements/notarization, one-shot secret delivery, memory/crash policy, and audit-deferred recovery remain separate later gates.
