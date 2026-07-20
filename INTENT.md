# QuanuX Symphony Intent

## Purpose
The purpose of the root repository is to establish the open-source platform boundary, guarantee modular sovereignty, and enforce platform-wide invariants such as individual installability and domain neutrality.

## Scope
- Platform invariants and doctrine definitions.
- Monorepo structural governance.
- Contract seed enforcement.

## Non-Scope
- Enforcing platform-wide domain dogma (market-data, order-flow).
- Mandating global infrastructure assumptions (Kubernetes, specific cloud providers).
- Supplanting individual module autonomy.

## Relationship to Modules
Modules retain sovereignty over their own physical bounds, capability declarations, and dependencies.

## Relationships
- **qxctl**: speaks commands.
- **Maestro**: coordinates.
- **SKV**: preserves knowledge.
- **SSCG**: interprets compatibility.
- **SCLV**: records change.
- **SKVI**: maps knowledge.
- **SACV**: governs declarative API contracts and their registry from `knowledge/sacv/`.
- **SSIAG**: implements secure identity and access governance from `knowledge/ssiag/`.
- **STAV**: defines per-TOPS audit protocol truth in `knowledge/stav/`.

## Relationship to First Runtime Set
The first runtime set (`node-troll`, `bus-troll`, `hotpath-runtime`) represents the most fundamental physical constraints of the system. Root governance ensures these modules remain strictly bounded and individually installable.

## Installability Expectations
Every module is expected to be individually installable without assuming a monolithic platform deployment.

Shared implementation may live under `libraries/` only when it has no runtime identity or authority and does not weaken consumer installability. Canonical knowledge vectors, not libraries, own protocol truth.

## Caller-Class Neutrality and Host Authority
Symphony does not classify a caller as human, AI, agent, service, workload, organization, or another actor type when deciding authority. Supported authorization decisions use the target host's ownership or granted permissions, the requested operation and resource, expected state, and owner-configured safeguards.

The target host's administrator is sovereign over configurable Symphony governance. SSIAG may verify and project effective host authority, but it does not create a superior registration authority or permanently veto that administrator. Enhanced identity assurance and governance interlocks are caller-neutral, owner-configured safeguards. Protocol-integrity requirements remain mandatory within supported tooling.

Symphony does not decide whether an actor may own property, open an account, sign a contract, assume liability, or act for another entity. Those facts belong to the relevant owner, provider, counterparty, and applicable law.

## Owner Ratification Boundaries
Any structural or doctrinal shift at the root level requires explicit ratification by a caller holding the required transition-owner permission.
