# SACV OpenAPI 3.2 Profile

## Target

Symphony canonical HTTP API descriptions target OpenAPI Specification 3.2.0.

## Required Posture

- YAML is the preferred authored representation; JSON is permitted only when the owner justifies it.
- `openapi` MUST be `3.2.0`.
- `info.version` MUST identify the Symphony API contract version, not the OpenAPI version.
- Every operation MUST have a stable `operationId` unique within the entry document.
- Requests, success responses, error responses, and content types MUST be explicit.
- Unknown-field behavior, identifier formats, bounds, and presence requirements MUST match the owning contract.
- Error bodies MUST use stable, safe reason codes and MUST NOT expose native or secret-bearing errors.
- Examples MUST be synthetic and secret-free.
- Protected operations MUST reference a ratified security profile.
- Internal or administrative documents MUST NOT declare a public production server.

## References and Bundling

References MAY split a canonical API across owner-controlled files when all referenced files share one semantic owner and are registered as part of the same contract. Cross-owner aggregation is a derived bundle, not a canonical document.

Remote references SHOULD NOT be required to validate a repository checkout. A validator MUST bound reference depth and size and reject cycles or traversal outside the approved owner surface.

## Compatibility Gate

Before adopting a consumer, generator, linter, Mintlify release, or SDK pipeline, verify that it:

1. accepts OpenAPI 3.2.0;
2. preserves 3.2.0 semantics and extensions used by the owner;
3. resolves references without network-dependent authority changes;
4. retains security, error, and presence semantics;
5. fails visibly rather than downgrading or dropping unsupported fields.

## Explicit Exclusions

This profile does not define an SSIAG token, remote gateway, default server, universal JSON-only rule, provider protocol, STAV append protocol, qxctl grammar, NATS schema, or trading-path representation.
