# STAV v1 Base Registries

## Status

Architect-ratified closed protocol registry. Runtime producer grants may use only identifiers explicitly configured for that authenticated producer.

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
- `symphony.stav.receipt.committed`
- `symphony.stav.receipt.idempotency-conflict`
- `symphony.stav.receipt.event-class-denied`
- `symphony.stav.receipt.operation-denied`
- `symphony.stav.receipt.tops-mismatch`
- `symphony.stav.receipt.ledger-full`
- `symphony.stav.receipt.ledger-unavailable`
- `symphony.stav.response.succeeded`
- `symphony.stav.response.invalid-request`
- `symphony.stav.response.unauthorized-peer`
- `symphony.stav.response.operation-denied`
- `symphony.stav.response.ledger-unavailable`
- `symphony.stav.response.ledger-full`
- `symphony.stav.response.internal-failure`
- `symphony.stav.verification.digest-mismatch`
- `symphony.stav.verification.sequence-mismatch`
- `symphony.stav.verification.tops-mismatch`
- `symphony.stav.verification.frame-corrupt`

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

Event classes, operation identifiers, authentication methods, reference kinds, producer kinds, and producer-specific reason codes are not assigned here. Each producer integration MUST document its meanings, and each installation MUST explicitly grant the exact authenticated producer `(event_class, operation_id)` tuples it may emit. Configuration can select registered integration identifiers but cannot alter their canonical meanings.
