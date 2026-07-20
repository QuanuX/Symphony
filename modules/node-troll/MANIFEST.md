# node-troll Manifest

## Module Identity
- **Name**: node-troll
- **Type**: Foundation module

## Troll Doctrine
trolls are the local residents.
A troll is a bounded local resident of a Symphony runtime domain.
A troll is a runtime-residency role, not a caller identity or authorization class.
A troll is not a global coordinator.
A troll is not the domain itself.

## Purpose
Acts as a node-local daemon and edge supervisor responsible for machine-local presence.

## Scope
node identity
local configuration interpretation
heartbeat/state reporting
local capability declaration
local compatibility mediation
supervision of declared local runtime relationships

## Non-Scope
global coordination
platform truth authority
native hot-path ownership
market-data semantics
order-flow semantics
strategy semantics
caller classification or autonomous architectural decision-making

## Installability
- **module install status**: PROPOSED
- **install scope**: Node-local supervision and daemon operations respecting the strict Configuration Authority paths.
- **supported installation modes**: development, production, source-evidence.
- **required runtime assumptions**: Bare-metal-first capability.
- **optional capabilities**: None inherently required platform-wide.
- **non-required dependencies**: Kubernetes, containers, specific cloud providers.
- **prohibited platform-wide dependencies**: Python must not be required for the administrative spine.
- **install verification expectations**: Daemon starts and heartbeats without external orchestration, adhering strictly to configuration precedence.
- **current ratification status**: DRAFT
