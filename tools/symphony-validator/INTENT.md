# Symphony Validator Intent

****

## Purpose
Declare the exact declarative boundaries and expected non-agentic behavior of the C++ validator before implementation begins.

## Scope
Defines the tool contract and behavioral specification for the Symphony validator.

## Non-scope
This seed does not contain C++ source files, headers, build files, CMake files, Makefiles, CI files, executable schemas, JSON schema files, Markdown template files, generated reports, implementation logic, service files, installer scripts, deployment scripts, runtime scaffolding, binary assets, or binary renames.

## Role
The validator is deterministic, explainable, and non-agentic. It produces structured evidence for humans, CI systems, qxctl, and agentic tools, but it does not perform interpretation, remediation, or architectural decision-making.

## Non-Agentic Doctrine
The validator does not infer intent.
The validator does not rewrite files.
The validator does not choose remedies.
The validator does not make architecture decisions.
The validator does not replace human review.
The validator does not replace agentic review.

## Relationship to qxctl
The validator declares the tool boundary that `qxctl` will eventually mediate.

## Relationship to CI / PR gates
The validator will provide deterministic exit codes and structured evidence for CI and PR gates.

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
This canonical seed does not authorize canonical repository mutation, C++ validator implementation, or executable schema generation.


## Troll Doctrine
trolls are the local residents.
A troll is a bounded local resident of a Symphony runtime domain.
A troll is not an AI agent.

node-troll represents the node.
bus-troll manages bus residency and bus compatibility.
hotpath-runtime owns the native hot path.
hotpath-runtime is not a troll; it is the native hot-path runtime substrate.


## Bus-Troll Optionality
bus-troll is first-class and individually installable.
bus-troll is required only for deployments that use a managed bus boundary.
Bus bypass remains valid when declared by deployment constraints.
The existence of bus-troll does not make bus traversal mandatory.
