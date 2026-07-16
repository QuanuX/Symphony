# Symphony Secure Identity and Access Governance Intent

## Identity

- **name**: secure-identity-access-governance
- **acronym**: SSIAG
- **type**: security foundation module
- **implementation language**: Go only
- **linking constraint**: cgo prohibited

## Purpose

Implement the canonical `knowledge/ssiag/` relationship and provider contracts for the complete decision chain:

```text
identity proof -> authenticated subject -> policy decision
  -> bounded capability -> credential reference or lease
  -> provider operation -> safe STAV outcome
```

## Monorepo and Independence

The module lives in the Symphony monorepo intentionally so humans and agents can reason across contracts, implementation, qxctl, providers, STAV, SKVI, and validator evidence in one workspace. Monorepo visibility does not grant runtime authority. SSIAG remains independently buildable, installable, enrollable, startable, stoppable, and removable.

One installed host binary may serve several TOPS instances. Every TOPS configuration, state root, socket, policy namespace, and audit relationship is isolated by immutable opaque ID. Mutable display names never determine security paths.

## Scope

- identity and proof-summary models;
- deny-by-default authorization;
- bounded capabilities, opaque references, and short-lived leases;
- provider discovery and future safe operation dispatch;
- provider-neutral, per-TOPS local administration;
- safe STAV security outcomes;
- independent platform-provider process boundaries.

## Non-Scope

- a password manager or general-purpose vault;
- plaintext, environment, or implicit fallback providers;
- qxctl schema ownership or provider SDK behavior;
- runtime events in SKV or SCLV;
- direct agent ledger edits or arbitrary appends;
- hot-path calls;
- mandatory bus, container, cloud, Python, vendor, or telemetry infrastructure;
- operational credential access in the present scaffold.

## Ratified Architecture and Current Gates

Local peer-credential authentication, foundational supervision, proposal/apply separation, a dedicated per-TOPS Go STAV append authority, provider mutual executable trust, protected one-shot secret delivery, and per-user macOS Keychain operation are ratified architectural directions.

Their operational code remains disabled until the exact schemas, platform identities, lifecycle rules, and negative-test gates in `REQUIREMENTS.md` pass. Remote SSIAG, agent apply authority, implicit fallback, and network listeners remain unauthorized.
