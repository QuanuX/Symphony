# Symphony API Contract Vector Skill

## Purpose

Guide humans and agents in creating, reviewing, registering, validating, and publishing Symphony HTTP API contracts without displacing domain ownership or security authority.

## Required Reading Order

1. `knowledge/sacv/INTENT.md`
2. `knowledge/sacv/MANIFEST.md`
3. `knowledge/sacv/SPEC.md`
4. `knowledge/sacv/profiles/openapi-3.2.md`
5. `knowledge/sacv/REGISTRY.md`
6. the owning domain vector or module contracts
7. `knowledge/sodv/SPEC.md` before proposing publication

## Agent Authority

Agents may inspect API governance, compare implementations with canonical contracts, propose contract changes, and produce non-authoritative validation evidence.

Agents must not:

- invent an endpoint, security scheme, server URL, scope, or token format;
- register an API without an identified semantic owner;
- enable remote access, a live playground, or MCP tools;
- publish internal or administrative endpoints;
- place secrets, proofs, assertions, credentials, or realistic secret-shaped examples in a contract;
- treat generated code, a bundled specification, Mintlify, or an SDK as canonical truth;
- use OpenAPI to describe provider secret delivery or STAV append ingestion.

## Change Procedure

1. Identify the semantic owner and affected Contract Quad.
2. Confirm that HTTP is the correct transport.
3. Ratify caller authentication and authorization before declaring a protected operation.
4. Author or update the canonical OpenAPI 3.2.0 entry document at the owner path.
5. Register the entry document in `knowledge/sacv/REGISTRY.md` and SKVI.
6. Validate syntax, references, compatibility, safe examples, and implementation conformance.
7. Keep bundles, generated types, routers, clients, SDKs, MDX, and vendor overlays derived.
8. Obtain SODV approval before creating publication configuration or externally visible projections.
9. Create SCLV evidence only after actual review and merge evidence exists.

## Stop Conditions

Stop and obtain owner ratification before selecting a new API transport, adding a remote SSIAG gateway, weakening security, publishing an administrative API, enabling live requests or MCP exposure, changing the SACV OpenAPI target, or moving endpoint truth between owners.

## Non-Authorization Statement

This skill is procedural guidance. It authorizes no endpoint, listener, schema generation, SDK publication, Mintlify configuration, public documentation, or agent mutation path.
