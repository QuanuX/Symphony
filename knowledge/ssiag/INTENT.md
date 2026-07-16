# Symphony Secure Identity and Access Governance Intent

## Purpose

SSIAG defines Symphony's canonical identity, authentication, authorization, bounded-capability, credential-reference, lease, and provider-operation relationship model.

## Source-Truth Boundary

`knowledge/ssiag/` is canonical protocol truth. Runtime code implements this contract; qxctl administers and queries it; per-TOPS configuration extends only the extension points this contract permits.

## Complete Decision Chain

```text
identity proof
  -> authenticated subject
  -> authorization policy decision
  -> bounded capability
  -> credential reference or lease
  -> provider operation
  -> safe STAV outcome
```

“Governance” includes the complete runtime decision chain. It is not limited to entitlement review or administrative reporting.

## Relationship Model

SSIAG has graph-like nodes and relationships and may later support derived graph projections. No graph database or generated identity database is authorized by this seed. Canonical Markdown and ratified configuration contracts remain authoritative.

## Security Intent

- deny by default;
- keep proof and credential material outside qxctl, SKV, STAV, logs, and projections;
- isolate every security namespace by immutable opaque TOPS identity;
- fail closed when an explicit provider is missing or incompatible;
- never select a plaintext or weaker provider fallback implicitly;
- keep SSIAG outside trading hot paths;
- emit only allowlisted audit metadata.

## Open-Source Posture

SSIAG performs no tracking, telemetry aggregation, or phone-home behavior by default. Its host and surrounding operating environment are secured by the installing owner or organization.

## Ratified Local Architecture

- Local v1 caller identity comes from kernel-attested Unix-socket peer credentials mapped to canonical SSIAG subjects.
- Foundational SSIAG and STAV services use an explicit bootstrap supervision stratum; supervision owns liveness and does not confer authorization.
- Administrative change uses separate proposal and apply authority. Agents may query and propose only.
- Provider control and secret delivery are distinct channels. Non-exportable operations remain inside the provider.
- The first operational macOS Keychain topology is per-user and session-aware; system/headless use never falls back implicitly.
- Remote SSIAG access is not part of local v1.

These choices are architecturally ratified but not operationally enabled merely by this document. Exact schemas, platform policy, implementation, and negative-test gates remain mandatory.
