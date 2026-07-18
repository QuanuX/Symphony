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

Local peer-credential authentication, foundational supervision, proposal/apply separation, provider mutual executable trust, protected one-shot secret delivery, and per-user macOS Keychain operation are ratified architectural directions. The Go foundation implements kernel credential extraction and exact UID/GID-to-subject mapping for accepted Darwin/Linux connections; this does not enable mutation or provider operations.

Kernel peer authentication, endpoint trust, native per-TOPS supervision/runtime ownership, and the typed mutually authenticated SSIAG-to-STAV producer are implemented. Credential release, provider mutation, and operational Keychain behavior remain disabled until their exact contracts and verification gates pass. Provider fallback, network listeners, graph-database deployment, and agent access to secret-bearing operations remain unauthorized.

## Status

Architect-ratified architecture with local peer identity, foundation supervision, and safe STAV producer foundations implemented. Remaining mutation and provider capabilities are gated by their own implementation evidence.
