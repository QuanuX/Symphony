# Symphony Feature Administration Assurance Contract

## Status and Purpose

Architect-ratified cross-vector contract for detecting whether registered Symphony features and independently developed modules have complete, headless administrative coverage. It is an umbrella contract, not a new vector, engine, authority service, or installation format.

The default rule is: every administrator-facing feature interaction requires a stable qxctl command route. A missing route is visible evidence. It never silently becomes not applicable.

## Identity Families

- feature: `ssfv:<registered-namespace>:<stable-key>`;
- engine operation: `engop:<registered-namespace>:<stable-key>`;
- qxctl executable leaf: `qxcmd:<registered-namespace>:<stable-key>`;
- invocation: caller-supplied operation or event identity under the applicable execution contract.

These identities are not interchangeable. Stable keys are semantic alphanumeric identifiers, not random database identifiers or command spellings. Titles, paths, and compatible grammar may change without changing identity. Incompatible semantics require a new identity. Deprecated and retired identities remain tombstoned and MUST NOT be reused. A retired qxctl tombstone advertises null grammar and may retain only a hidden, fail-closed compatibility receptor that returns its stable retirement and replacement diagnostic; it cannot satisfy feature coverage.

Every public or hidden executable qxctl leaf has exactly one command ID. Command groups do not satisfy coverage. Aliases share the leaf's ID. One command may serve several feature interactions, and one interaction may compose several commands and operations.

Namespaces require explicit owner-contract allocation, collision review, and SKVI routing. Third-party developers use their own registered prefix, for example `ssfv:acme:...`, `engop:acme:...`, and `qxcmd:acme:...`; a repository, package, organization, hostname, or caller identity does not allocate a namespace.

## Single-Source Descriptors

The C++ dispatch table is the source for the observed engine descriptor. The Go command specification used to construct a Cobra leaf is the source for the observed qxctl manifest. Generated help text and prose scraping are not evidence.

Descriptor language is bounded implementation evidence, never identity or authority. Symphony-authored engines default to C++ where suitable, while justified Go or native-language engines and third-party modules remain admissible under the same operation, protocol, thermal, and administration checks.

`symphony.knowledge.engine-descriptor.v2` is side-by-side with descriptor v1. V1 is not widened or rewritten. V2 adds stable operation identities and administration semantics and deliberately omits installation state. Installation remains immutable receipt and live observation truth; integration/admission is a separate assessment.

The canonical expected qxctl registry is maintained at `tools/qxctl/COMMANDS.json`. It is a checked-in, digest-bound projection of the same Go `CommandSpec` and Cobra tree that construct the executable grammar, and qxctl verifies exact parity rather than maintaining a second handwritten command inventory. An exact binary may emit an observed registry. Absence of qxctl leaves design evaluation possible and yields `qxctl_absent` only on the live axis. An observed manifest reports literal behavior, including `json_output: false`; headless policy may then report the gap.

## Administration Expectations

The interaction vocabulary is `discover`, `inspect`, `query`, `validate`, `configure`, `propose`, `invoke`, `apply`, `lifecycle`, and `recover`. Each registered feature records applicable expectations as `required`, `optional`, `prohibited`, or `not_applicable` and a delivery form of `direct`, `composed`, `delegated`, `lifecycle_only`, `observation_only`, `runtime_only`, `system_orchestrated`, `none`, or `unreviewed`.

Exceptions are explicit and evidence-backed. `prohibited` and `not_applicable` require `none`, no command or operation mapping, and a rationale. In report-only bootstrap, an explicitly present feature with an empty expectation array is feature-level unreviewed debt and invents no interaction; `unreviewed` delivery represents a known interaction whose route is unreviewed. Neither form is an exception or passes an enforcement gate. Parent inheritance is permitted only for the same interaction and compatible requirement; every reference must resolve within the profile and the inheritance graph must be finite and acyclic. A materially distinct child interaction requires its own expectation.

For a composed interaction, every declared command must bind the exact feature and interaction, and the union of those commands' backend-operation bindings must cover every declared operation. A composed surface does not require every command to target every operation. Missing commands and missing operation coverage remain separate findings so a multi-step workflow can identify the exact broken edge without rejecting valid many-to-many composition.

Permission-backed mutation normally requires inspect/status, proposal or plan, apply, apply-status, and recovery coverage plus expected-state, stable invocation identity, SSIAG binding, result validation, and required STAV evidence. An evidence-backed contract may state why one family member is not applicable.

The foundational SSIAG/STAV exception demonstrates complete lifecycle coverage without collapsing identities: each of its four administrator-facing feature expectations is satisfied by five `qxcmd:` leaves and five separately registered `engop:` operations. `status` binds backend `observe`; plan, apply, apply-status, and recover preserve their semantic names. The checked-in administration profile lists all ten identities per feature, while `knowledge/FOUNDATIONAL-LIFECYCLE.md` owns their exact machine, authority, installation, attempt, audit, and recovery contract.

## Engine-First Evaluation

The engine is the only component whose presence may be assumed. An installed SSFV engine evaluates caller-supplied, bounded, digest-bound inputs: an SSFV semantic snapshot, the administration profile, the canonical expected command registry, zero or more engine descriptor v2 objects, and an optional observed qxctl registry. It MUST NOT require a repository checkout or qxctl executable to be locally present, scrape help, discover ambient configuration, prompt, or use caller class as authority.

Direct engine invocation uses bounded stdin/stdout JSON, exact protocols, deterministic errors and digests, explicit deadlines, and no terminal-dependent behavior. The engine detects gaps and emits proposal-only remediation constraints. It does not invent canonical English names, command grammar, feature-worthiness, exemptions, or ratification. Name and grammar proposals are caller-supplied against the exact input digests, and deterministic tooling validates syntax, collisions, completeness, compatibility, and staleness. Future AI assistance remains ordinary caller assistance under the same authenticated-subject and permission rules.

## Capacity and Growth Posture

The administration envelope's JSON-value count is parser workload, not a count of stored feature records. The 17,000-value smoke fixture is deliberately synthetic boundary evidence; it does not describe the canonical catalog. The current source truth remains bounded Markdown and JSON under review, and no database is required for canonical feature administration.

SSFV alone admits at most 65,536 parsed JSON values for one complete administration request while retaining the common one-MiB request ceiling. Ordinary administration envelopes reaching 32,768 values require an explicit capacity review; reaching 49,152 values requires a ratified paging or digest-addressed chunking migration before further growth. Those thresholds preserve at least 50% and 25% structural headroom respectively. A future Maestro, DuckDB, or graph projection may cache or index rebuildable evidence, but it must not replace canonical repository truth, relax complete-envelope validation, or silently raise the parser bound.

## Three Independent Axes

- design: `satisfied`, `uncovered`, `exempt`, `prohibited`, `stale`, or `unresolved`;
- live: `not_evaluated`, `ready`, `qxctl_absent`, `not_installed`, `not_bound`, `incompatible`, `unreachable`, `disabled`, or `unknown`;
- authorization: `not_evaluated`, `allowed`, `denied`, `expired`, or `indeterminate`.

Permission denial and missing installation do not change design coverage. A command mapping without handler, backend operation, result validation, required machine output, tests, or recovery evidence is not complete coverage.

## Independent Module Admission

`invariant:symphony:module-administration.declaration` in `knowledge/INVARIANT-OWNERSHIP.json` makes this admission boundary mechanically traceable to the SSFV producer and validator regressions. It applies when a new installable module, adapter entry point, backend operation, or administrator-facing interaction first enters review; forgetting to request qxctl work is the exact case it is designed to expose.

An independently developed engine may ship with empty `feature_ids` or `unreviewed` administration disposition so the platform can report what the developer omitted. Omission of descriptor fields is invalid; omission of semantic registration is `semantic_registration_required`.

Integration states are `unassessed`, `descriptor_invalid`, `semantic_registration_required`, `administration_unintegrated`, `integration_ready`, `blocked_incompatible`, and `retired`. Each result records the exact descriptor digest when available and an explicit `docking_ready` projection, which is true if and only if integration state is `integration_ready`. These are derived assessment states and MUST NOT be stored as, inferred from, or substituted for receipt installation, activation, or docking state; docking-ready evidence is not docking presence or authority.

Repository admission independently enumerates bounded immediate `modules/` children with `CMakeLists.txt`, `go.mod`, `Package.swift`, or actual `src`, `cmd`, or `Sources` implementation markers. Every such source module MUST have an exact module-root `FEATURES.md`, SSFV registry route, and administration-profile entry. A documentation-only Contract Quad seed remains excluded until an implementation marker appears. This check discovers omission only; it does not decide feature names, create records, or make the SSFV engine repository-dependent.

A module may be built, tested, packaged, installed inactive, and inspected while unintegrated. Required safe administration gaps block the `integration_ready` claim and may block docking or activation under the applicable lifecycle contract. Runtime-only, system-orchestrated, prohibited, and not-applicable operations remain valid only with explicit evidence.

## Digests and Closure

Every self-digest is the tagged SHA-256 of the compact UTF-8 JSON object after recursive lexicographic key sorting and before the self-digest member is inserted; the self-digest member is omitted from the preimage. Results bind the SSFV semantic snapshot, profile, canonical expected registry, optional observed registry, and every engine descriptor digest.

Checks operate feature-to-command, command-to-feature, command-to-engine, engine-to-command, and implementation-to-evidence. Duplicate IDs, ID reuse, unresolved mappings, stale digests, inheritance cycles, descriptor/dispatch drift, expected/observed registry drift, unvalidated structured results, and missing recovery fail closed or produce explicit uncovered evidence according to the forward gate.

## Bootstrap and Forward Gate

The current SSFV registry contains exactly eighty-seven experimental records and is explicitly partial. Its report-only bootstrap covers 170 reviewed expectations: 159 required, ten prohibited, and one not applicable. The profile is `enforce_new_records`, so every newly ratified feature must include its administration disposition in the same reviewed change. Existing expected routes that lack exact feature bindings or backend-operation identity remain uncovered implementation work; gate advancement does not deem them satisfied. The stable qxctl registry contains 191 command leaves. Root-summary assurance has one exact read-only command identity, while invariant assurance exposes read-only status, list, show, and exact-validator-check identities over the current thirteen-record incremental invariant registry; apply is explicitly prohibited. Provider-trust assurance adds exact inspect and permission-backed fresh-verification routes. Provider-readiness assurance adds one permission-backed headless observation route bound to the foundation and macOS adapter operations while every operational flag remains false. Provider-binding assurance adds distinct installation discovery, binding inspection, plan, apply, apply-status, and recovery identities while keeping operational Keychain and secret delivery disabled. SAV and SEV add exact bounded read-only/proposal command identities without transferring canonical authority or host mutation to either engine. `enforce_all_records` requires separate evidence that the registered catalog is fully profiled. None of these states claims repository-wide feature, legacy-invariant, or installed-package completeness.

## Non-Authorization

This contract authorizes read-only coverage evidence and proposal-only remediation recipes. It does not authorize canonical mutation, semantic name invention, dynamic executable injection, receipt rewriting, installation, docking, activation, arbitrary module entry-point execution, SSIAG permission, STAV append, AI-specific authority, or qxctl presence.
