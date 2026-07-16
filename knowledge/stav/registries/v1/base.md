# STAV v1 Base Registries

## Status

Owner-ratified closed protocol registry. Producer-specific values remain gated.

## Generic Outcomes

- `allowed`
- `denied`
- `failed`
- `succeeded`
- `unavailable`

## Redaction Classifications

- `administrative_metadata`
- `restricted_metadata`

## Protocol Reason Codes

- `symphony.stav.configuration.not-applicable`
- `symphony.stav.authentication.not-applicable`
- `symphony.stav.trog.not-applicable`
- `symphony.stav.receipt.rejected`
- `symphony.stav.verification.digest-mismatch`
- `symphony.stav.verification.sequence-mismatch`

## Conformance-Only Identifiers

The following values exist only in canonical fixtures and tests. They grant no producer or runtime authorization:

- `symphony.stav.fixture.authentication`
- `symphony.stav.fixture.principal`
- `symphony.stav.fixture.producer`
- `symphony.stav.fixture.target`
- `symphony.stav.fixture.event`
- `symphony.stav.fixture.operation`
- `symphony.stav.fixture.intent`
- `symphony.stav.fixture.allowed`

## Registry Rules

All registered identifiers are lowercase dotted ASCII identifiers. An identifier has two or more non-empty segments; each segment begins with a lowercase letter and continues with lowercase letters, digits, or hyphens. An unknown value fails closed.

Event classes, operation identifiers, authentication methods, reference kinds, producer kinds, and producer-specific reason codes are not assigned here. They require a producer-integration contract that states which authenticated producer can emit each tuple. Per-installation extensions may add names only through a future owner-controlled configuration contract and cannot alter the canonical meanings above.
