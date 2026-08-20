# Symphony Evolution Vector Specification

## Status and Normative Terms

Architect-ratified v1 contract. MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative. The first engine implementation is report-only and proposal-only; durable mutation remains external.

## Purpose

SEV governs how Symphony reasons about a planned change or encountered novelty, derives a deterministic impact and disposition plan, recalculates after evidence changes, and verifies a transition without becoming the authority that performs it.

## Stable Identities

SEV uses:

```text
sevcase:<namespace>:<stable.dotted-key>
sevdisp:<namespace>:<stable.dotted-key>
```

They MUST match:

```text
^(sevcase|sevdisp):[a-z][a-z0-9-]{0,62}:[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$
```

The first-party namespace is `symphony`. Case identity is caller supplied and remains distinct from the semantic fingerprint that provides idempotency. IDs are not permission grants or database keys.

## Evolution Case

`symphony.sev.evolution-case.v1` is an append-forward noncanonical case envelope. It binds:

- case ID and semantic fingerprint;
- exact source CURRENT snapshot and digest;
- case kind: `planned_change` or `encountered_novelty`;
- caller-declared target or novelty evidence;
- affected Accord References, SSFV features, qxctl commands, engine operations, packages, bindings, receptors, contracts, and traits;
- current disposition state;
- dependency graph, hard safety phases, ready set, and blockers;
- apply and recovery requirements;
- required target reobservation;
- state generation, predecessor digest, and case digest.

The engine does not persist the case. A durable coordinator stream MAY preserve exact case evidence under the separately authorized lifecycle contract.

## Case Lifecycle

Case states are:

- `open`;
- `assessed`;
- `planned`;
- `ready`;
- `awaiting_external_action`;
- `reobservation_required`;
- `recalculation_required`;
- `attunement_required`;
- `converged`;
- `blocked`;
- `abandoned`;
- `closed`.

No state is inferred solely from a process exit code or wall-clock time. Completed, failed, blocked, superseded, and recovered attempts remain immutable evidence.

## Impact Assessment

`impact_assess` compares the requested target or novelty with one exact CURRENT snapshot and applicable Accord References. It returns sorted affected surfaces, unresolved ownership, missing evidence, definite conflicts, compatibility observations, and a deterministic impact digest.

The engine MUST NOT claim complete impact when the source CURRENT coverage is partial/unknown or a required consequence family is unresolved.

## Dependency and Ready-Set Rules

The plan is a directed acyclic graph unless it reports an explicit dependency-cycle blocker. Edges are either:

- `hard_safety`: immutable order that cannot be bypassed or reordered;
- `semantic_dependency`: action requires evidence produced by its predecessor;
- `independent`: no ordering edge; actions may appear together in the ready set.

A localized blocker removes only its dependent descendants from readiness. Independent compatible actions may proceed in deterministic stable-ID order. After every completed or failed external action, SEV recomputes the complete ready set from new observed evidence. It never edits the prior plan to make history appear linear.

## Disposition Planning

`disposition_plan` uses only dispositions in `DISPOSITIONS.md`. Every action declares:

- stable semantic operation identity;
- disposition;
- target and exact expected-state digest;
- required authorization and audit evidence;
- hard prerequisites;
- closed success-observation predicate using the SAV v1 rule algebra;
- recovery operation identity;
- execution class: report-only, proposal-only, or external apply;
- thermal restriction;
- action digest.

SEV executes no prose, shell fragment, arbitrary expression, plugin, or model output as a success predicate. The predicate is the same closed, versioned rule object used by SAV and is evaluated only against the newly supplied CURRENT source projections. Adding a predicate kind therefore requires compatible SAV/SEV contract evolution rather than an implementation-only shortcut.

Engine plans are noncanonical and non-authoritative.

## Verification and Recalculation

`transition_verify` requires the exact case, plan, attempted action, execution-evidence digest, and complete new SAV CURRENT snapshot. It marks success only when observation proves the declared predicate. Ambiguity, partial delivery, stale evidence, or incompatible state is indeterminate or failed, never successful.

`case_recalculate` preserves all prior evidence and derives a successor generation. `case_recover` returns only deterministic advice for one exact supplied case stream; the engine never mutates coordinator state. `case_close` may propose closure only when converged or explicitly abandoned with reason and authority evidence.

## SCSEV

`command_surface_assess` is governed by `profiles/qxctl-command-surface.md`. It consumes existing SSFV administration and qxctl registry protocols. It validates every consequence family and returns:

- `complete`;
- `semantic_registration_required`;
- `administration_unintegrated`;
- `blocked_incompatible`;
- `retirement_incomplete`;
- `unresolved`.

It returns proposal constraints with null final identity/grammar when caller input is absent. It does not create a duplicate registry or an executable command.

## Engine Process

The exact engine ID is `symphony-sev`, module ID `sev-engine`, and vector ID `sev`. It uses `symphony.knowledge.engine-process.v1`.

### v1 Operations

- `inspect`;
- `case_open`;
- `impact_assess`;
- `disposition_plan`;
- `transition_verify`;
- `case_recalculate`;
- `case_status`;
- `case_recover`;
- `case_close`;
- `command_surface_assess`;
- `novelty_bundle_check`;
- `watch_policy_check`;
- `trigger_coalesce`;
- `evolution_session_bind`;
- `project_graph`;
- `compatibility`.

Operation identities are:

- `engop:symphony:sev.inspect`;
- `engop:symphony:sev.case.open`;
- `engop:symphony:sev.impact.assess`;
- `engop:symphony:sev.disposition.plan`;
- `engop:symphony:sev.transition.verify`;
- `engop:symphony:sev.case.recalculate`;
- `engop:symphony:sev.case.status`;
- `engop:symphony:sev.case.recover`;
- `engop:symphony:sev.case.close`;
- `engop:symphony:sev.command-surface.assess`;
- `engop:symphony:sev.novelty-bundle.check`;
- `engop:symphony:sev.watch-policy.check`;
- `engop:symphony:sev.trigger.coalesce`;
- `engop:symphony:sev.session.bind`;
- `engop:symphony:sev.graph.project`;
- `engop:symphony:sev.compatibility`.

## Bounds and Digests

V1 admits at most 1 MiB request, 4 MiB response, 1,024 affected surfaces, 1,024 actions, 4,096 graph nodes, 8,192 edges, 1,024 blockers/findings, and common JSON/path/time limits. Self-digests use recursively key-sorted compact JSON with the self field omitted. Semantic-set arrays sort by stable identity. Bound changes require review.

## Durable Coordination

The knowledge-session coordinator, not SEV, owns durable mutation streams. `evolution_session_bind` content-addresses the exact case, source CURRENT, target, lifecycle profile, source report journal, desired state, direction, and STSC creation time that enter that existing stream. It creates no second journal and grants no apply authority. Accordare evolution uses existing no-follow locks, dual journal slots, atomic heads, file/directory synchronization, exact compare-and-swap, immutable applied evidence, idempotent operation fingerprints, and unique adjacent-chain recovery.

qxctl performs a reviewed external action only after fresh SSIAG authorization. It fully reobserves configured roots and Maestro receptors, obtains a successor CURRENT snapshot, and invokes verification/finalization. A stopped or restarted session recovers from durable evidence, not from timestamps or file order.

## Installation, Selection, and Maestro

The independently installed engine uses receipt v2, exact inactive-undocked version paths, and explicit qxctl binding. Exact versions coexist. The recommended receptor is `receptor:symphony:knowledge.sev`. Installation, selection, activation, docking, execution, and authorization remain separate.

## Watch and Session Foundation

Any watcher or session trigger is opt-in and qxctl-administered. The default conceptual session begins at successful authentication and ends at logout or mandatory reauthentication, while an administrator may select another bounded policy. `watch_policy_check` validates the bounded policy and `trigger_coalesce` deterministically binds an ordered event set into one case candidate. Neither persists a policy, opens a case, installs a watcher, runs on a hot/warm path, silently applies, or makes AI mandatory.

## Novelty Export

Novelty remains local by default. Voluntary export requires an inspectable bounded bundle, explicit SSIAG permission, schema-aware redaction, safe STAV outcome evidence, and indeterminate partial-delivery handling. Payloads, local overlays, secrets, and raw host inventories do not enter STAV. No network export is implemented by the v1 engine.

The v1 operation contract separates nested artifacts from operation envelopes. Novelty Bundle check, Watch Policy check, trigger coalescing, and evolution-session binding each have a strict input schema, and the three check/coalescing operations have strict result schemas under `knowledge/sev/schemas/v1/`; the binding result remains the existing immutable binding schema. Engine descriptors and qxctl advertise those exact protocols so unsupported old/new combinations fail closed without losing caller evidence.

## Time

STSC applies. Durable timestamps use target-host whole-second UTC. Monotonic clocks govern deadlines. Generations, fingerprints, expected-state digests, and predecessor links govern causal order.

## Upgrade Compatibility

Schemas are immutable. Readers may implement explicit bounded legacy adapters. Unknown critical versions remain preserved and read-only. Forward, reverse, foundation-first, client-first, engine-first, and interrupted sequences converge through explicit evidence and dynamic replanning without newest-version inference.

## Non-Authorization Statement

SEV cannot authenticate, authorize, ratify, mutate canonical knowledge, apply its plan, invent IDs or grammar, rewrite history, bypass hard safety edges, select versions, mutate Maestro, publish novelty, require a database/AI service, or participate in hot/warm execution.
