# qxctl Intent

qxctl is the Go-based local administrative spine for Symphony.

## Purpose
qxctl is a repository and module inspection/control surface. It is a deterministic local status/inventory/digest tool designed to read and report Symphony repository state and to speak safe administrative commands to independently installed modules.

## Scope
qxctl operates as a local utility to verify modules, aggregate contracts, digest runtime inventory, and use safe local administrative APIs. It uses Go 1.26.5 as the current scripted baseline and targets Go 1.27 after general availability and differential conformance. A toolchain migration cannot change protocol bytes or command grammar.

The command tree and flag grammar use Cobra. Viper is a constrained command-configuration mapper: each command configuration that maps an environment value receives a private instance, all keys and environment variables are bound explicitly, and no automatic environment discovery, remote provider, configuration-file discovery, watch/reload, write-back, or secret value is permitted. Dedicated SSIAG and STAV clients retain exclusive responsibility for trusted configuration loading and endpoint authentication.

The secure-identity-access-governance integration is local and provider-read-only. qxctl reads SSIAG health and safe provider descriptors and requests exact audited authorization decisions for knowledge-session operations over a kernel-authenticated Unix domain socket. Decision/capability objects are safe non-transferable evidence, not credentials; qxctl does not receive or persist credential values.

Every SSIAG query is scoped by immutable TOPS ID. `knowledge/ssiag/` owns SSIAG protocol truth; qxctl only implements its administrative/query projection. Future administrative change separates deterministic `propose` from permission-backed local `apply`. Authorization depends on target-host ownership or granted permission, the requested operation and resource, expected state, and owner-configured safeguards; qxctl does not request or evaluate caller type.

The Architect-ratified `qxctl stav status|verify|query|doctor` grammar is operational. It loads the selected per-TOPS STAV contract, authenticates the authority endpoint from kernel credentials, submits strict local envelopes, and displays only classification-authorized projections. qxctl has no `stav append`, does not edit STAV ledgers, and does not own `knowledge/stav/` schemas. qxctl grammar is not governed by OpenAPI.

The SKVI, SCLV, SACV, SODV, and SSFV vector-engine grammars are operational as `qxctl skvi inspect|check|propose|project`, `qxctl sclv inspect|check|propose|recover|project`, `qxctl sacv inspect|check|diff|propose|project`, `qxctl sodv inspect|check|verify|propose|recover|project`, and `qxctl ssfv inspect|check|diff|propose|graph`. qxctl requires an explicit installation prefix, resolves the exact version from its inactive undocked receipt, validates every receipt-owned path, invokes the independently installed C++ engine through bounded standard I/O with a hard deadline and empty environment, and verifies response identity, digest, and operation-specific safety assertions. SSFV baselines and operation inputs are bounded no-follow JSON files; a supplied baseline defaults check freshness to `report`, while `require` fails unresolved stale semantics.

`qxctl validate scan|debug`, `profile list|show|set|remove`, and `baseline create|show|remove` are operational under `knowledge/VALIDATION.md`. qxctl validates one exact independently installed validator receipt, executes the complete C++ check with an empty environment and hard bounds, verifies stable finding/evidence/result digests, and only then applies a protected per-TOPS warning profile and compatible baseline. Debug filters affect display after the full scan. Violations cannot be downgraded; optional warnings may be recorded, reviewed, or required without changing raw evidence.

`qxctl knowledge engines list|inspect|doctor|bind|unbind` is operational for one protected user-scope `default` profile. It records exact receipt and executable digests, uses a persistent no-follow lock, requires an exact expected prior registry state, and writes the noncanonical registry atomically and durably. It never selects the newest installation implicitly.

`qxctl knowledge reconcile compatibility|begin|status|checkpoint|close|recover` is operational through the exact bound coordinator. Each call snapshots and revalidates every bound installation, supplies the complete role-sorted engine inventory plus binding-registry digest, and negotiates process/journal/capability overlap. Mutations use stable operation IDs and exact expected journal state; explicit discovery recovery is permitted only when local evidence selects one unambiguous forward state. This is noncanonical worktree coordination, not install-receipt activation, vector invocation, apply, or Maestro docking.

`qxctl knowledge session begin|status|checkpoint|close|recover` is also operational. Before every call qxctl reads one protected binding snapshot, resolves and revalidates the exact coordinator installation, authenticates the selected TOPS SSIAG endpoint, and requests the exact operation/resource/audience/scope authorization. It rejects denial, target drift, expiry, caller-class use, transferability, canonical-apply claims, or inconsistent capability evidence before invoking the coordinator. Session operations use stable operation IDs, exact prior state, linked epochs, and explicit unambiguous discovery recovery over protected noncanonical state. They do not record the reconciliation engine inventory. Repository-specific, system, and TOPS engine-binding profiles, broader safeguard administration, and `knowledge apply` remain unavailable. The separate validator warning-profile surface is implemented and grants no session or apply authority.

`qxctl knowledge session transition` composes those exact operations for an explicit stable `login`, `refresh`, or `logout` host event. Retry is evidence-based and idempotent, refresh rotates authority only when reauthentication is required, and optional recovery is confined to damaged local head/journal evidence. qxctl installs no hook, watcher, login-manager integration, or boot service.

`qxctl knowledge session features begin|status|checkpoint|close|recover` administers a separate persistent SSFV maintenance stream. Mutation requires an open authenticated session and exact coordinator/SSFV bindings. qxctl collects a validated semantic snapshot, computes an SSFV v2 diff against the preserved baseline for checkpoint/close, optionally obtains a complete authenticated Maestro inventory, and binds every input to fresh SSIAG authorization and exact journal compare-and-swap. `review_required` remains noncanonical evidence; qxctl never decides feature-worthiness, writes `FEATURES.md`, or applies an SSFV proposal through this circuit.

`qxctl maestro inventory` exposes the authenticated TOPS-wide derived receptor inventory used by the maintenance circuit. Its stable inventory digest excludes observation time; the separately digested outer observation remains timestamped. A damaged, busy, unsafe, or ambiguous receptor stream fails the whole request instead of becoming an incomplete inventory.

`qxctl knowledge lifecycle profile list|show|set|remove`, `ownership status|reconcile|adopt|release`, `observe`, `report`, `boot`, `status`, `recover`, `apply`, `apply-status`, and `apply-recover` implement the current generic lifecycle administration boundary governed by `knowledge/LIFECYCLE.md`. Profiles are protected per TOPS beneath the selected state root, generated and linked by qxctl, and mutated through exact compare-and-swap. Root-local ownership registries serialize cross-profile claims, while an exact receipt-layout compatibility fence makes older lifecycle clients preserve and block rather than bypass state they cannot understand. Observation scans only fixed receipt layouts under administrator-selected roots and, when an exact Maestro receptor is supplied, overlays authenticated presence without executing discovered code. Explicit apply requires an `apply-compatible` profile plus exact source-report, apply-journal, and applied-state compare-and-swap values. It obtains fresh SSIAG authorization for every phase, asks the C++ coordinator to durably prepare one dependency-ready action, reconciles ownership before the external action, executes only reviewed receipt-v2, runtime-state, established-binding, or Maestro-presence adapters, re-observes all roots, and finalizes only from verified evidence. Established bindings remain the closed six-role v1 registry, but each lifecycle selection is an exact receipt-v2 compare-and-swap. Coordinator upgrade and rollback require the candidate to reproduce the prepared journal before the registry switch; qxctl keeps the invoking coordinator for finalization and resumes through the selected coordinator after a crash or successful handoff. `qxctl ssiag grants lifecycle` generates deterministic exact grants; the separate protected `qxctl ssiag policy` circuit can propose, apply, and recover the operational overlay without rewriting enrolled config or canonical knowledge. `qxctl maestro inspect|status|recover|inventory` administers safe inspection, recovery, and complete read-only derived receptor inventory directly; dock and undock have no direct leaf command and remain lifecycle-only. Package download, receipt-v1 mutation, arbitrary entry-point execution, live service or Maestro engine activation, unconstrained rebinding, in-place coordinator self-replacement, host boot-hook installation, and canonical apply remain unavailable.

## Non-goals
- qxctl does not execute hotpath-runtime workloads.
- qxctl does not make bus traversal mandatory.
- qxctl does not require Python.
- qxctl does not perform remote execution.
- qxctl does not manage NATS directly.
- qxctl does not own hotpath-runtime execution.
- qxctl does not replace node-troll.
- qxctl does not replace bus-troll.
- qxctl does not replace hotpath-runtime.
- qxctl does not choose infrastructure.
- qxctl does not assume Docker/Kubernetes/cloud.
- qxctl does not assume trading, market-data, strategy, provider, or plugin ABI behavior.
- qxctl does not directly write generated SKVI/SCLV/SACV/SODV/SSFV records; it may request noncanonical proposals from ratified engines.
- qxctl does not enforce runtime behavior.
- qxctl does not implement identity-provider, keyring, or secret-provider SDK behavior.
- qxctl does not accept or print secret values through SSIAG commands.
- qxctl does not grant target-host authority or make caller-class policy.

## Relationship
qxctl reads and reports Symphony repository state and administers independently installed modules and vector engines. It relates to node-troll, bus-troll, hotpath-runtime, secure-identity-access-governance, STAV, and SKV engines as an administrative command and inspection surface, not as an owner of their workloads, schemas, semantics, or security state.
