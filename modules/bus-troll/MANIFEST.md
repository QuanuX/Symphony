# bus-troll Manifest

## Module Identity
- **Name**: bus-troll
- **Type**: Foundation module

## Troll Doctrine
trolls are the local residents.
A troll is a bounded local resident of a Symphony runtime domain.
A troll is not an AI agent.
A troll is not a global coordinator.
A troll is not the domain itself.

## Purpose
Acts as the managed boundary and bridge between local runtime components and the bus fabric.

## Scope
local bus boundary residency
NATS or other declared bus connection lifecycle, where applicable
bus credential/configuration boundary mediation, where applicable
bus health/state reporting
bus compatibility interpretation
safe traversal between local runtime components and the bus fabric
optional bus bridge behavior when required by deployment constraints

## Non-Scope
forcing all deployments to use a bus
making NATS mandatory for every system design
owning node identity
owning the native hot path
owning global coordination
replacing Maestro
replacing node-troll
acting as an AI agent

## Installability
- **module install status**: PROPOSED
- **install scope**: Managed bus boundary operations.
- **supported installation modes**: development, production, source-evidence.
- **required runtime assumptions**: Bare-metal-first capability.
- **optional capabilities**: None inherently required platform-wide.
- **non-required dependencies**: Kubernetes, containers, specific cloud providers.
- **prohibited platform-wide dependencies**: None that mandate bus usage.
- **install verification expectations**: Daemon manages bus residency without global dependency.
- **current ratification status**: DRAFT

bus-troll is first-class and individually installable.
bus-troll is required only for deployments that use a managed bus boundary.
Bus bypass remains valid when declared by deployment constraints.
The existence of bus-troll does not make bus traversal mandatory.
