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
- Deterministic summary and exit status

Structured JSON and Markdown projections remain deferred.

## Dependencies
A conforming C++26 compiler, CMake 3.25 or newer, and the C++ standard library. Runtime validation has no third-party library dependency.

## Non-goals
The validator must not choose infrastructure for users, assume Docker/Kubernetes/cloud providers, impose market-data/order-flow/trading doctrine, require Python for hot-path or administrative spine, ban optional isolated Python habitats, treat contract seeds as runtime implementation, convert monorepo modularity into microservices doctrine, absorb module sovereignty into root-level logic, become a runtime daemon, become a hidden coordinator, replace qxctl, replace Maestro, replace SKV / SKVI / SCLV / SODV records, become agentic, infer intent, auto-remediate files, or make architecture decisions.
It does not contain any forbidden terms (such as c-o-r-e).

## Non-authorization Statement
This manifest authorizes the checked-in deterministic C++26 parser/checker, CMake build, and smoke fixtures. It does not authorize executable schema generation, structured projectors, qxctl integration, CI configuration, runtime residency, publication, or remediation.
