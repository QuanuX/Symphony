# Symphony Secure Identity and Access Governance Manifest

## Canonical Target

`knowledge/ssiag/`

## Classification

- independent Symphony Knowledge Vector contract surface governed by the Architect;
- canonical SSIAG vocabulary and relationship authority;
- declarative protocol truth;
- not runtime state, a credential store, or a graph database.

## Owned Truth

SSIAG owns the canonical semantics for identities, authentication results, authorization decisions, capabilities, credential references, leases, provider operations, safe outcomes, provider compatibility, configuration extensions, and their allowed relationships.

The vector owns the authorization/capability schemas, deterministic lifecycle-grant planning, protected local policy-administration contracts, exact provider installation/binding lifecycle, three-layer provider-readiness contract, and the closed provider trust/control/channel/readiness schemas under `schemas/v1/`. Policy and provider-binding administration change separate operational per-TOPS state; neither edits canonical knowledge or the enrolled configuration file.

## Authority Split

- `knowledge/ssiag/` owns protocol and relationship truth.
- `modules/secure-identity-access-governance/` implements the Go foundation.
- separately installed provider modules implement reviewed platform boundaries.
- per-TOPS configuration declares local instances and permitted extensions.
- qxctl is the administrative and query interface, not schema authority.
- STAV records safe security outcomes, not SSIAG execution context.
- SKVI indexes these canonical surfaces.

## Language Boundary

All Symphony-authored SSIAG foundation source is Go and cgo is prohibited. A platform adapter may use another language only as a separately built and installed process behind a versioned, protected IPC boundary. It may never be dynamically linked into the Go foundation.

## Identity Boundary

Immutable opaque IDs and mutable display names are separate fields. Paths, sockets, policies, service identities, and event sequences use IDs only. Display-name changes never relocate state or change security identity.

## Ratified Architecture Versus Enabled Capability

Local peer-credential authentication, foundational supervision, exact deny-by-default authorization, proposal/apply separation, provider mutual executable trust, protected one-shot secret delivery, and per-user macOS Keychain operation are ratified architectural directions. The Go foundation implements kernel credential extraction, exact UID/GID-to-subject mapping, exact-grant policy decisions, bounded non-transferable capabilities, and one metadata-only mutual executable-trust path for accepted Darwin/Linux connections. These decisions enable only explicitly granted noncanonical administrative operations and do not enable canonical apply, Keychain operations, or secret delivery.

Kernel peer authentication, endpoint trust, native per-TOPS supervision/runtime ownership, exact caller-neutral policy evaluation, audited authorization decisions, protected local policy proposal/apply/recovery, the typed mutually authenticated SSIAG-to-STAV producer, the provider v1 metadata trust runtime, exact provider installation/binding administration, and the Phase 10B readiness foundation are implemented. SSIAG validates legacy bare and exact multi-file bundle receipts, privately stages every bundle-owned byte, and keeps structural validation, native signing-policy match, and operational eligibility distinct. The separately installed Swift adapter independently verifies the invoking installed foundation during metadata trust and produces safe native signing/session readiness observations. Credential release, operational Keychain behavior, the synthetic secret channel, general safeguard administration, and canonical knowledge apply remain disabled until their implementation gates pass. Provider fallback, network listeners, graph-database deployment, and secret-bearing general administrative surfaces remain unauthorized.

## Status

Architect-ratified architecture with local peer identity, foundation supervision, deny-by-default authorization decisions, non-transferable capability evidence, safe STAV production, exact metadata-only provider trust, and protected provider-binding lifecycle implemented. Protected noncanonical knowledge-session mutation is the first consumer. Remaining canonical mutation, safeguard, credential, secret-delivery, and operational-provider capabilities are gated by their own implementation evidence.
