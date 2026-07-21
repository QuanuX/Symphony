# Symphony API Contract Vector Intent

## Purpose

SACV defines Symphony's canonical governance for declarative API contracts. It establishes how HTTP interfaces are described, owned, versioned, validated, registered, and made eligible for derived documentation or SDK publication.

## Source-Truth Boundary

`knowledge/sacv/` owns cross-cutting API-contract policy, the OpenAPI profile, compatibility rules, and the API-contract registry. It does not automatically own the endpoint semantics of every Symphony module or knowledge vector.

The surface that owns an API's domain behavior owns that API's canonical OpenAPI document. SACV registers and governs that document. A generated aggregate, code binding, SDK, documentation page, playground, or MCP tool is a derived projection and never becomes source truth.

## API-First Intent

HTTP contracts are authored before implementation. Go routers, C++ handlers, request and response types, clients, conformance tests, public documentation, and SDKs are generated from or strictly validated against the canonical contract.

## Initial Standards Target

SACV targets OpenAPI Specification 3.2.0. Symphony does not silently downgrade a canonical API contract for a lagging consumer. Publication or generation waits until the selected toolchain passes its declared compatibility gate.

## Scope

SACV governs HTTP API contracts, including administrative HTTP, remote orchestration HTTP, and webhooks when those surfaces are independently ratified.

SACV does not govern:

- qxctl command grammar;
- Unix-socket peer authentication itself;
- SSIAG provider IPC or secret delivery;
- STAV append ingestion;
- NATS or other bus payloads;
- trading hot paths or binary IPC;
- public documentation approval.

SODV governs publication approval. SKVI indexes canonical SACV and API-owner surfaces. SCLV records reviewed change truth only after real review and merge evidence exist.

## Security Intent

- API descriptions never invent an authentication scheme to fill a design gap.
- Protected operations bind to a separately ratified SSIAG security profile.
- Internal and administrative APIs are non-public by default.
- Live server targets, interactive execution, SDK publication, and MCP tool exposure are independently gated.
- Secret values, proofs, assertions, provider payloads, and credentials never appear in specifications, examples, generated fixtures, documentation, or SDK defaults.

## Non-Authorization Statement

This vector authorizes API-contract governance, registration, and a proposal-only C++ SACV engine at `modules/sacv-engine/` after the common contract transition merges. The engine may validate, diff, propose, and project registered owner contracts. It does not authorize a network listener, remote SSIAG access, an endpoint schema, canonical registry mutation, Mintlify configuration, public documentation, an SDK release, a live playground, MCP exposure, or generated server/client code.
