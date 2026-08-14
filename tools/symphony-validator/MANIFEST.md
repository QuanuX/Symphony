# Symphony Validator Manifest

****

## Tool Identity
Symphony Validator

## Canonical Path
`tools/symphony-validator/`

## Classification
- repository/tooling concern
- not a runtime module resident

## Language
C++26

## Contract Files
- `INTENT.md`
- `MANIFEST.md`
- `INSTALL.md`
- `SKILL.md`
- `SPEC.md`

## Installability
The validator is an implemented native tool with a local CMake build, exact versioned installation receipt, receipt-owned uninstall target, direct invocation surface, and qxctl mediation.
Python must not be required for validator execution as part of the administrative spine.
Optional isolated Python habitats may exist only when explicitly declared by a module or tool.

## Invocation Surfaces
- Direct binary invocation
- Local preflight invocation
- Exact receipt-validated `qxctl validate scan|debug|root-summary` and `qxctl knowledge invariant check` invocation

CI/PR-gate invocation remains a separate integration surface.

## Output Surfaces
- Deterministic line-oriented evidence
- Deterministic summary and exit status (including `21` for caller-authority regression, `22` for SACV registry failure, `23` for SODV release-ledger failure, `24` for feature-administration contract failure, `25` for root-summary assurance failure, and `26` for invariant-ownership assurance failure)
- Deterministic `symphony.validation.result.v1` JSON through `check --json`
- Deterministic `symphony.repository.root-summary.v1` JSON and exact bounded Markdown through `root-summary`

General documentation projection, runtime source/AST caller-authority analysis, and auto-remediation remain deferred and unauthorized.

## Canonical JSON Boundary

The artifact checker recognizes exactly 160 canonical JSON paths: 28 STAV v1 schemas/fixtures, seventy common SKV process/descriptor/receipt/binding/proposal/provider-evidence/reconciliation/session/SSFV-maintenance/generic-lifecycle/foundational-lifecycle/ownership/temporal/Maestro/validation/feature-administration/invariant schemas (sixty-six v1 and four v2), nineteen SSIAG authorization/grant-planning/policy-administration/provider-control/provider-trust schemas, four SKVI operation/result schemas, five SCLV v3 operation/result schemas, six SACV v1 operation/result schemas, eight SODV operational schemas, eighteen SSFV v1/v2 schemas, the exact feature-administration profile JSON, and the exact common invariant-ownership registry. It does not authorize a directory prefix, generated projection, or new JSON artifact by extension.

The contract-shape and canonical-surface checks require the common STSC surface and its authority/profile/implementation boundaries, the SSFV governance/engine surfaces, and the independently installed Maestro Contract Quad/build/qxctl client anchors. The validator confirms anchors, presence, SKVI coverage, and exact JSON allowlisting; it does not decide feature-worthiness, duplicate vector semantics, or inspect operational presence state.

The SACV registry checker independently validates the empty marker or exact thirteen-field entry grammar, identity/path uniqueness, owner-path containment, OpenAPI/profile enums, SKVI coverage, and no-follow document presence. OpenAPI semantic validation remains in the independently installed SACV engine rather than being duplicated through an unsafe partial parser.

The SODV release checker independently validates bounded no-follow v1/v2 records, identity and time order, type/status coupling, immutable authorization relationships, publication-unit shape, and one completion per authorization. Provider observation, network resolution, proposal generation, and recovery recommendations remain in the independently installed SODV engine.

The feature-administration checker independently validates the bounded no-follow profile, exact SSFV registry byte digest and dynamic registered set, and checked-in expected qxctl command registry. It verifies omit-self JSON digests, profile coverage, sorted unique command IDs, registered feature bindings, profile command references, inheritance closure, and report-only versus enforcement-gate treatment of unreviewed debt. It also performs a sorted no-follow census of direct `modules/` children and requires every scope with a direct CMake, Go, Swift, `src`, `cmd`, or `Sources` implementation marker to have its exact root `FEATURES.md`, SSFV route, and profile mapping; documentation-only Contract Quad seeds are not implementation. The current profile contains 139 reviewed expectations for 73 records, reports zero unreviewed entries, and uses `enforce_new_records`. The checker reads declarative data only; it does not execute qxctl, require qxctl installation, invent names or grammar, assess feature semantics, or suggest remediation.

The invariant-ownership checker independently validates the bounded no-follow common registry, exact fixed shape and stable identity grammar, recursive omit-self digest, sorted/unique invariant, adapter, operation, path, and case relations, owner and implementation path presence, named Go/C++ regression definitions, distinct producer/consumer evidence roles, adapter closure, and per-adapter real-process evidence for IPC invariants. Named test presence is source traceability, not evidence that a suite executed or passed. A failure returns exit `26`; applicable suites remain the responsibility of build and test orchestration.

The root-summary projector consumes only those validated SSFV/administration inputs plus completed SODV publication units. It emits an omit-self digest-bearing JSON object or one exact Markdown managed region, and the complete check rejects README drift only after all authoritative gates pass. It does not write the repository, convert README into canonical truth, choose current release versions, or publish documentation.

Direct `apply` invocation is rejected with stable invalid-usage status because the validator is read-only. Validation and root-summary projection do not mutate repository bytes on either pass or failure.

## Caller-Authority Capability
The implemented checker reads active Markdown from the bounded repository surfaces defined in `SPEC.md`. It emits lexical-path evidence for configured caller-class authority constructions and for fail-visible discovery, stream, symlink, and resource-limit conditions. It does not follow symlink targets or modify scanned content.

## Dependencies
A conforming C++26 compiler, CMake 3.25 or newer, the shared static knowledge-vector foundation, and its vendored nlohmann JSON dependency. The installed executable has no dynamic Symphony-library dependency.

The caller-authority checker depends only on the validator evidence formatter and the C++ standard library. Its direct input is a repository path; its implemented outputs are line-oriented evidence, one summary line, and the process status.

## Non-goals
The validator must not choose infrastructure for users, assume Docker/Kubernetes/cloud providers, impose market-data/order-flow/trading doctrine, require Python for hot-path or administrative spine, ban optional isolated Python habitats, treat contract seeds as runtime implementation, convert monorepo modularity into microservices doctrine, absorb module sovereignty into root-level logic, become a runtime daemon, become a hidden coordinator, replace qxctl, replace Maestro, replace SKV / SKVI / SCLV / SODV / SSFV records, perform autonomous semantic decisions, infer intent, auto-remediate files, or make architecture decisions.
It does not contain any forbidden terms (such as c-o-r-e).

## Non-authorization Statement
This manifest authorizes the checked-in deterministic C++26 parser/checker, structured result projector, exact installation/uninstallation surface, qxctl integration, and fixtures. It does not authorize executable schema generation, CI configuration, runtime residency, publication, or remediation.
