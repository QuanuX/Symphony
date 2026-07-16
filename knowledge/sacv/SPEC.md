# Symphony API Contract Vector Specification

## Status and Normative Terms

Owner-ratified API-contract governance. MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative. This specification creates no HTTP endpoint by itself.

## Purpose

Define the ownership, placement, OpenAPI version, compatibility, validation, derivation, security, and publication boundaries for Symphony HTTP API contracts.

## OpenAPI Version Profile

Canonical Symphony OpenAPI descriptions MUST declare OpenAPI Specification `3.2.0`. The exact permitted and prohibited feature profile is defined by `knowledge/sacv/profiles/openapi-3.2.md`.

A consumer that cannot parse or preserve the canonical 3.2.0 contract MUST fail its compatibility gate or defer processing. It MUST NOT rewrite or publish a downgraded description as if it were canonical.

## Ownership Model

SACV owns API-contract governance and the registry. Endpoint semantics remain with the surface that owns the behavior:

- SSIAG HTTP semantics belong to `knowledge/ssiag/`;
- STAV HTTP query semantics belong to `knowledge/stav/`;
- module-specific HTTP semantics belong to that module unless a knowledge vector owns the protocol;
- a cross-module gateway belongs to SACV only when the gateway's composition contract is explicitly assigned to SACV.

Canonical descriptions MUST NOT be copied into SACV for aggregation. The registry points to the single owner document.

## API-First Contract

For a new HTTP operation, the canonical contract MUST be reviewed before implementation is enabled. Implementations MUST be generated from or checked against the accepted contract. Generated bindings and routers MUST identify the source contract digest and generator/tool version when generation is used.

Handwritten implementations MUST pass request, response, error, security-profile, and unknown-field conformance tests. An implementation annotation or reflection output MUST NOT overwrite the canonical description.

## Contract Quad Relationship

The owning `INTENT.md`, `MANIFEST.md`, `INSTALL.md` or `SPEC.md` as applicable, and `SKILL.md` establish the human-readable authority and lifecycle boundaries. The OpenAPI document is a conditional typed artifact subordinate to those declared boundaries. It does not become a universal fifth required surface.

## Registry Contract

Every canonical OpenAPI entry document MUST have one registry entry containing:

- stable API identifier;
- semantic owner;
- canonical repository path;
- OpenAPI version;
- API contract version;
- audience and exposure classification;
- transport profile;
- authentication/authorization profile reference;
- publication eligibility and SODV state;
- SDK eligibility state;
- lifecycle status.

An empty registry is valid. A placeholder endpoint document is not required to prove that SACV exists.

## Versioning and Compatibility

The OpenAPI Specification version and the Symphony API contract version are independent. The former identifies the description language. The latter identifies the API behavior.

Within one API major version, an owner MUST NOT remove operations, remove or narrow accepted values, make optional input required, reinterpret existing fields, weaken security, change error semantics incompatibly, or change identifier meaning. Deprecation MUST precede removal and MUST identify the replacement or terminal reason.

Compatibility checks are evidence. Human review and the owning contracts decide whether a change is accepted.

## Transport Scope

SACV governs HTTP messages. A local HTTP server carried over a Unix socket MAY be described only after its transport and caller-authentication profile is ratified. OpenAPI does not itself authenticate Unix peers and MUST NOT be presented as the complete local security contract.

OpenAPI MUST NOT govern:

- qxctl command grammar;
- SSIAG provider control or secret channels;
- STAV producer-to-append-authority ingestion;
- NATS subjects or payload schemas;
- binary IPC or trading paths.

## Security Contract

Protected operations MUST reference a ratified authentication and authorization profile. A security scheme MUST NOT be invented merely because documentation or a generator requires one. In particular, no generic “SSIAG token” exists unless SSIAG separately ratifies its format, issuer, audience, lifecycle, and threat model.

Caller identity claimed in a request body is untrusted. Authorization is an implementation obligation bound to the authenticated channel and SSIAG policy.

Canonical descriptions and examples MUST NOT contain credentials, tokens, assertions, proofs, private keys, provider payloads, secret-bearing native errors, production hostnames, or realistic secret-shaped fixtures.

## Publication and Vendor Boundary

SODV alone authorizes public documentation and Mintlify configuration. The canonical OpenAPI document SHOULD remain vendor-neutral. Mintlify-specific navigation, `x-mint` behavior, live playground settings, code samples, and MCP exposure belong to an SODV-governed derived publication overlay unless separately ratified as portable contract metadata.

Internal and administrative descriptions default to:

- no public server target;
- no live request execution;
- no public SDK publication;
- no MCP tool exposure.

Publication MUST wait until the consumer proves OpenAPI 3.2.0 compatibility without semantic loss.

## Derivation Boundary

The following are derived and disposable:

- bundled or dereferenced OpenAPI documents;
- generated Go/C++ types, clients, and routers;
- SDKs and SDK examples;
- Mintlify MDX and navigation;
- API playground configuration;
- MCP tool projections;
- lint, diff, and conformance reports.

Derived artifacts MUST identify their canonical inputs and MUST NOT be edited as a competing source of truth.

## Non-Authorization Statement

This specification authorizes no endpoint, listener, remote gateway, provider operation, STAV append path, generated binding, SDK publication, Mintlify configuration, public documentation, live playground, or MCP tool.
