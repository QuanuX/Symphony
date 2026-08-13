# Symphony Validation Evidence and Policy Contract

## Status and Authority

Architect-ratified common cross-vector contract for deterministic repository evidence, protected validation profiles, baselines, actionable subject-aware warning lifecycle state, and qxctl administration. This contract is not a knowledge vector and creates no semantic engine, daemon, network service, canonical mutation path, or CI requirement.

Canonical vector and module contracts own the conditions being checked. `symphony-validator` implements deterministic evidence production but does not own those truths. qxctl owns administrative grammar, protected profile persistence, baseline management, filtering, presentation, and policy evaluation; it does not alter a detector's result.

## Purpose

Keep one reproducible evidence stream while allowing a target-host administrator to decide how warning families are displayed, reviewed, enforced, accepted for a bounded period, or related to a stable successor. Ordinary output contains actionable open warnings rather than replaying every historical occurrence. Complete occurrence and digest history remains inspectable, and a debug scan can expose exact rule, subject, record, path, classification, and resolution evidence without weakening production checks.

## Protocols

- `symphony.validation.result.v1`: deterministic structured validator evidence plus an optional derived qxctl policy evaluation;
- `symphony.validation.policy.v1`: protected noncanonical warning disposition and presentation state;
- `symphony.validation.baseline.v1`: protected noncanonical content-addressed warning inventory for one repository identity and validator version.
- `symphony.validation.warning-state.v1`: protected noncanonical subject-aware lifecycle state with occurrence history and a digest-linked transition chain.

The exact schemas are:

- `knowledge/schemas/v1/validation-result.schema.json`;
- `knowledge/schemas/v1/validation-policy.schema.json`;
- `knowledge/schemas/v1/validation-baseline.schema.json`.
- `knowledge/schemas/v1/validation-warning-state.schema.json`.

The warning-state protocol is side-by-side rather than a widening of result, policy, or baseline v1. Exact existing clients therefore remain valid and are never required to accept unknown fields in a digest-bound v1 object.

## Immutable Detection Boundary

For the same repository snapshot and validator version, raw evidence MUST be independent of qxctl profile, baseline, caller description, output filter, or presentation mode. A profile MUST NOT:

- remove a finding from raw evidence;
- change a rule's category, matcher, phrase distance, path scope, resource bound, or historical boundary;
- downgrade a violation, malformed canonical record, integrity failure, unreadable required surface, or unsupported state;
- weaken no-follow access, ownership/mode checks, bounded parsing, atomic replacement, expected-state validation, or secret exclusion;
- cause the validator or qxctl to rewrite SKVI, SCLV, SSFV, or another canonical surface.

Candidate/debug traces MAY expose additional explicitly non-gating evidence after a separately reviewed rule contract exists. Candidate evidence never changes canonical scan success.

## Finding Identity

Every structured finding carries:

- the exact category and stable rule ID;
- deterministic detail and bounded scalar attributes;
- `scope`: `active`, `historical`, or `system`;
- an `occurrence_id` over category, rule ID, scope, and exact detail;
- a `subject_id` over the stable condition being discussed.

For `sclv.affected_surface.unindexed`, occurrence identity includes the SCLV record and path, while subject identity groups the same unindexed path across historical records. The lifecycle can consequently distinguish an entirely new subject from a new occurrence of a known subject. This preserves all append-only evidence while allowing a path-centric summary.

The validator result contains no collection timestamp. Its evidence and result digests are deterministic. qxctl may add a policy evaluation without changing the embedded evidence digest.

## Warning Dispositions

A validation policy assigns a default warning disposition and may override exact rule IDs:

- `record`: retain evidence without making the qxctl evaluation unsuccessful;
- `review`: return `review_required` when a matching warning is new relative to the selected baseline;
- `require`: return `failed` when a matching warning is new relative to the selected baseline.

Warnings without a baseline are treated as new. Violations always produce `failed` regardless of policy. `record` is the supported way to disable an optional warning gate; detection remains intact.

Presentation is independent of disposition:

- `full`: show each selected occurrence;
- `summary`: group by rule and subject while preserving counts and query access;
- `count`: show only bounded category/rule totals in ordinary output.

The default policy is `record`, historical presentation `summary`, and new presentation `full`.

## Baselines and Delta Semantics

A baseline binds exact warning occurrence and subject IDs to:

- one target-host TOPS identity;
- one local repository identity digest;
- one validator identifier and exact version;
- one raw evidence digest;
- one STSC whole-second UTC creation timestamp.

qxctl compares a new raw result with the baseline and reports `new`, `unchanged`, and `resolved` warning identities. A repository identity or validator version mismatch fails closed and requires an explicit new baseline; qxctl never silently migrates or reuses incompatible baseline evidence.

Baseline creation is an administrative acknowledgement of observed evidence, not a declaration that a warning is correct, resolved, ratified, or canonical. It does not modify historical ledgers.

## Actionable Warning Lifecycle

A warning-state stream is bound to one TOPS, repository identity, validator identifier, and exact validator version. Synchronization always consumes one complete, already digest-verified raw result. It records every observed warning occurrence, its stable subject, exact finding body, first and last STSC observation, and every distinct raw evidence digest that carried the occurrence. A subsequent complete observation classifies an absent open or accepted subject as resolved; it never deletes the occurrence or its evidence. A resolved subject that reappears becomes open and records a recurrence transition.

The subject classification is exactly one of:

- `open`: currently actionable;
- `accepted`: explicitly acknowledged with a required rationale and optional future STSC whole-second UTC expiry;
- `resolved`: absent from the latest complete detector result;
- `superseded`: explicitly related, with rationale, to a distinct known successor subject.

Acceptance expiry is evaluated as open for presentation immediately and is durably recorded on the next complete synchronization. Supersession must be acyclic. Resolution is detector-derived only: an administrator cannot declare an unobserved defect resolved, and a caller cannot reopen a resolved subject without new complete detector evidence.

`muted_presentation_only` is orthogonal to classification. It can remove a current warning from ordinary prose output, but never changes raw evidence, actionability classification, policy outcome, query visibility, occurrence history, or digests. Debug and warning-state queries expose muted subjects explicitly.

Every detected, new-occurrence, resolution, recurrence, expiry, acceptance, reopen, supersession, mute, and unmute transition is sequence-ordered and digest-linked. Administrative transitions require an exact expected state digest and a bounded rationale. Warning-state files use the same no-follow, owner-only, synchronized atomic replacement and directory durability rules as profiles and baselines.

Ordinary scan output displays violations plus actionable open, non-muted warnings. Accepted, resolved, superseded, and presentation-muted warnings remain available through debug and exact warning-state queries. JSON raw evidence remains complete; presentation never rewrites the detector result.

## qxctl Administration

The intended command group is:

```text
qxctl validate scan ...
qxctl validate debug ...
qxctl validate profile list|show|set|remove ...
qxctl validate baseline create|show|remove ...
qxctl validate warning status|list|show|sync ...
qxctl validate warning accept|reopen|supersede|mute|unmute ...
```

`scan` performs the complete raw check and applies one protected profile and optional baseline. `debug` performs the same complete check and then filters only the displayed projection by exact rule, record, path, or delta class. A filter never narrows execution.

Profiles, baselines, and warning-state streams live under:

```text
<state-root>/symphony/<tops-id>/qxctl/validation/
```

They use effective-user-owned mode-`0700` directories, mode-`0600` files, persistent no-follow locks, exact expected-state compare-and-swap, synchronized temporary files, atomic replacement, and directory synchronization. They contain no credentials, authorization proofs, secrets, canonical records, or absolute repository paths. Warning synchronization invokes the same complete detector as scan; classification and query filters are downstream projections and cannot narrow detector execution.

Every caller holding the applicable target-host authority receives the same supported controls. Authority derives from target-host identity and granted permission, never caller class.

## Installation and Invocation

The validator is an independently installable, versioned C++26 tool. Multiple versions MAY coexist under one prefix. qxctl resolves one exact version through its immutable receipt, validates every receipt-owned file and the executable under no-follow ownership/mode rules, invokes it with an empty environment and hard deadline, and verifies both structured digests before applying a profile.

Direct invocation remains available for diagnostics. qxctl does not scrape line-oriented output.

## Thermal and Execution Boundaries

Repository validation, profile administration, baseline comparison, and debug projection are freezing-path work. They MUST NOT run inline with hot or warm trading work, share progress locks with those paths, or create a synchronous trading dependency.

The validator and qxctl validation group are finite local processes. They do not become resident, dock with Maestro, schedule engines, invoke vector semantics, perform garbage collection, monitor files, or install a host/session hook.

## Non-Authorization

This contract does not authorize canonical apply, automatic remediation, persistent observers, CI mutation, PR merging, semantic ratification, warning deletion, detector-derived resolution override, generated SKVI/SCLV/SSFV records, graph persistence, remote validation APIs, Windows-native execution, or policy based on whether a caller is human, AI, agentic, automated, a service, a workload, or an organization.
