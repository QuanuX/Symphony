# SACV Mintlify Publication Profile

## Authority Boundary

SACV governs canonical API contracts. SODV governs publication. Mintlify is a derived publication surface and never canonical authority.

## Preconditions

No API contract may be projected into Mintlify until:

1. its owner contract and SACV registry entry are ratified;
2. SKVI indexes the canonical entry document;
3. validation and compatibility evidence passes;
4. SODV approves the intended audience and exposed operations;
5. the selected Mintlify release demonstrates lossless OpenAPI 3.2.0 consumption;
6. live requests, SDK examples, and MCP exposure each have an explicit decision.

## Default-Deny Publication Controls

For internal and administrative APIs, a publication projection defaults to:

- no production `servers` target;
- no interactive request sending;
- no credential collection in the documentation UI;
- no public SDK examples or package publication;
- no MCP tool exposure;
- no inclusion of secret-bearing examples, payloads, logs, or native errors.

## Derived Vendor Configuration

Mintlify navigation, MDX, vendor extensions, code samples, playground settings, authentication UI, and MCP settings are generated or maintained as SODV-governed projections. They MUST identify the canonical SACV registry entry and owner contract from which they derive.

If Mintlify or an SDK integration requires a combined specification, that combined document is generated outside canonical owner paths and MUST NOT be edited as source truth.

## SDK Boundary

SDK generation and publication are separate approvals. Generated SDKs MUST NOT embed credentials, default to an unratified server, bypass SSIAG, or expand any caller's effective authority. SDK examples are documentation projections, not protocol truth.
