# bus-troll Install

## Install Status
PROPOSED

## Install Scope
Managed bus boundary operations.

## Supported Installation Modes
- **development**: Source-based build and local test deployment.
- **production**: Bare-metal service installation and lifecycle management.
- **source-evidence / proposal-only state**: Current state. Contract seeds only.

## Installability / Optionality Doctrine
bus-troll is first-class and individually installable.
bus-troll is required only for deployments that use a managed bus boundary.
Bus bypass remains valid when declared by deployment constraints.
The existence of bus-troll does not make bus traversal mandatory.

## Configuration Authority
Module-specific runtime configuration separated from the administrative spine.

## Explicit Non-Requirements
- Kubernetes, containers, specific cloud providers.
- NATS or bus messaging is not mandatory for every system design.

## Current Limitations
Proposal-only state. No implementation code or installer scripts exist.

## Ratification Status
DRAFT
