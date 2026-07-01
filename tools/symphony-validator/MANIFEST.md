# Symphony Validator Manifest

****

## Tool Identity
Symphony Validator

## Path canonical seed
`tools/symphony-validator/`

## Classification
- repository/tooling concern
- not a runtime module resident

## Planned Language
C++

## Contract Files
- `INTENT.md`
- `MANIFEST.md`
- `INSTALL.md`
- `SKILL.md`
- `SPEC.md`

## Installability
The validator is planned as an individually installable native tool.
No implementation, build system, or install command is authorized by this manifest.
Python must not be required for validator execution as part of the administrative spine.
Optional isolated Python habitats may exist only when explicitly declared by a module or tool.

## Planned Invocation Surfaces
- Direct binary invocation
- `qxctl`-mediated invocation
- CI-mediated invocation
- Local preflight invocation

## Planned Output Surfaces
- JSON structured evidence
- Markdown agent/human ingestion projection

## Dependencies
None. (C++ implementation not yet authorized).

## Non-goals
The validator must not choose infrastructure for users, assume Docker/Kubernetes/cloud providers, impose market-data/order-flow/trading doctrine, require Python for hot-path or administrative spine, ban optional isolated Python habitats, treat contract seeds as runtime implementation, convert monorepo modularity into microservices doctrine, absorb module sovereignty into root-level logic, become a runtime daemon, become a hidden coordinator, replace qxctl, replace Maestro, replace SKV / SKVI / SCLV / SODV records, become agentic, infer intent, auto-remediate files, or make architecture decisions.
It does not contain any forbidden terms (such as c-o-r-e).

## Non-authorization Statement
This canonical seed does not authorize canonical repository mutation, C++ validator implementation, or executable schema generation.
