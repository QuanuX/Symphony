# Symphony Validator Intent

****

## Purpose
Define the exact authority boundaries and deterministic, non-autonomous behavior of the implemented C++ validator.

## Scope
Defines the tool contract and behavioral specification for the Symphony validator.

## Non-scope
The implemented boundary contains a C++26 command-line parser/checker, a CMake build contract, and smoke fixtures. It does not contain a runtime service, deployment system, CI integration, qxctl mediation, executable schema generator, JSON/Markdown projector, auto-remediation path, or publication pipeline.

## Role
The validator is deterministic, explainable, and non-autonomous. It produces structured evidence for every caller, CI systems, qxctl, and agentic tools, but it does not perform interpretation, remediation, or architectural decision-making.

## Deterministic, Non-Autonomous Doctrine
The validator does not infer intent.
The validator does not rewrite files.
The validator does not choose remedies.
The validator does not make architecture decisions.
The validator does not replace permission-backed ratification or semantic review.

## Relationship to qxctl
The validator is currently invoked directly. Any `qxctl`-mediated invocation requires a separate contract and implementation increment.

## Relationship to CI / PR gates
The validator provides deterministic exit codes and line-oriented evidence suitable for local preflight. CI and PR-gate wiring remain separate integration work.

## Relationship to SKV / SKVI / SCLV / SODV
The validator does not replace SKV / SKVI / SCLV / SODV records. It provides evidence to support them.

## Relationship to Module Sovereignty
The validator sits outside module logic and respects module sovereignty by only observing declarative boundaries.

## Relationship to Python Doctrine
Python must not be required for remote native hot-path execution or the administrative spine.
Optional isolated Python habitats may exist only when explicitly declared by a module or tool.
Choosing C++ for the validator does not ban optional isolated Python habitats.
It prevents Python from becoming required validator infrastructure for the administrative spine.

## Non-authorization Statement
This contract authorizes the checked-in deterministic C++26 parser/checker, its CMake build, and smoke fixtures. It does not authorize executable schema generation, JSON/Markdown projection, qxctl integration, CI mutation, publication, or remediation.


## Troll Doctrine
trolls are the local residents.
A troll is a bounded local resident of a Symphony runtime domain.
A troll is a runtime-residency role, not a caller identity or authorization class.

node-troll represents the node.
bus-troll manages bus residency and bus compatibility.
hotpath-runtime owns the native hot path.
hotpath-runtime is not a troll; it is the native hot-path runtime substrate.


## Bus-Troll Optionality
bus-troll is first-class and individually installable.
bus-troll is required only for deployments that use a managed bus boundary.
Bus bypass remains valid when declared by deployment constraints.
The existence of bus-troll does not make bus traversal mandatory.
