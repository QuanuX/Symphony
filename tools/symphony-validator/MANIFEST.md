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
The validator is an implemented native tool with a local CMake build and direct invocation surface.
Portable installation packaging is not yet implemented.
Python must not be required for validator execution as part of the administrative spine.
Optional isolated Python habitats may exist only when explicitly declared by a module or tool.

## Invocation Surfaces
- Direct binary invocation
- Local preflight invocation

Deferred integration surfaces are `qxctl` mediation and CI/PR-gate invocation.

## Output Surfaces
- Deterministic line-oriented evidence
- Deterministic summary and exit status (including `21` for caller-authority regression, `22` for SACV registry failure, and `23` for SODV release-ledger failure)

Structured JSON and Markdown projections remain deferred. Runtime source/AST caller-authority analysis and auto-remediation are strictly deferred and unauthorized.

## Canonical JSON Boundary

The artifact checker recognizes exactly 57 canonical JSON paths: 28 STAV v1 schemas/fixtures, six common SKV process/descriptor/receipt/proposal/provider-evidence schemas, four SKVI operation/result schemas, five SCLV v3 operation/result schemas, six SACV v1 operation/result schemas, and eight SODV operational schemas. It does not authorize a directory prefix, generated projection, or new JSON artifact by extension.

The SACV registry checker independently validates the empty marker or exact thirteen-field entry grammar, identity/path uniqueness, owner-path containment, OpenAPI/profile enums, SKVI coverage, and no-follow document presence. OpenAPI semantic validation remains in the independently installed SACV engine rather than being duplicated through an unsafe partial parser.

The SODV release checker independently validates bounded no-follow v1/v2 records, identity and time order, type/status coupling, immutable authorization relationships, publication-unit shape, and one completion per authorization. Provider observation, network resolution, proposal generation, and recovery recommendations remain in the independently installed SODV engine.

## Caller-Authority Capability
The implemented checker reads active Markdown from the bounded repository surfaces defined in `SPEC.md`. It emits lexical-path evidence for configured caller-class authority constructions and for fail-visible discovery, stream, symlink, and resource-limit conditions. It does not follow symlink targets or modify scanned content.

## Dependencies
A conforming C++26 compiler, CMake 3.25 or newer, and the C++ standard library. Runtime validation has no third-party library dependency.

The caller-authority checker depends only on the validator evidence formatter and the C++ standard library. Its direct input is a repository path; its implemented outputs are line-oriented evidence, one summary line, and the process status.

## Non-goals
The validator must not choose infrastructure for users, assume Docker/Kubernetes/cloud providers, impose market-data/order-flow/trading doctrine, require Python for hot-path or administrative spine, ban optional isolated Python habitats, treat contract seeds as runtime implementation, convert monorepo modularity into microservices doctrine, absorb module sovereignty into root-level logic, become a runtime daemon, become a hidden coordinator, replace qxctl, replace Maestro, replace SKV / SKVI / SCLV / SODV records, perform autonomous semantic decisions, infer intent, auto-remediate files, or make architecture decisions.
It does not contain any forbidden terms (such as c-o-r-e).

## Non-authorization Statement
This manifest authorizes the checked-in deterministic C++26 parser/checker, CMake build, and smoke fixtures. It does not authorize executable schema generation, structured projectors, qxctl integration, CI configuration, runtime residency, publication, or remediation.
