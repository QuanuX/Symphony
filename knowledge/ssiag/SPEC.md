# Symphony Secure Identity and Access Governance Specification

## Status and Normative Terms

Owner-ratified v1 architecture and relationship contract. MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative when the related capability is implemented. A ratified architecture does not become operational merely because its contract is described.

## Canonical Relationship Types

SSIAG models the following distinct nodes: topology identity, identity proof summary, authenticated subject, policy, decision, capability, credential reference, lease, provider, provider operation, and safe outcome.

Allowed directional relationships are:

```text
TOPS contains subject/provider/policy
proof authenticates subject
policy evaluates subject + requested operation + target
decision grants or denies bounded capability
capability permits credential-reference or provider operation use
lease binds subject + capability + reference + expiry
provider performs declared operation
operation produces safe outcome
safe outcome may be submitted to STAV
```

Proofs, raw assertions, secret values, raw tokens, and provider payloads are execution data, not graph nodes or audit fields.

## Extension Contract

Per-TOPS configuration MAY add instances of canonical node types, safe attributes, provider descriptors, and explicitly versioned extension relationships. An extension MUST declare its namespace, version, source and target types, cardinality, validation rule, safe-output classification, and compatibility behavior. Configuration MUST NOT redefine canonical relationship semantics or enable an unknown extension silently.

Any future graph or vector representation is a disposable projection rebuilt from canonical contracts and local configuration. It is never source truth.

## Topology Identity

Canonical `tops_id`, `trog_id`, `module_id`, and `host_id` values are immutable opaque UUIDs. Their corresponding names are mutable display metadata. An implementation MUST NOT construct an identity by concatenating an ID and name. Security paths and namespaces use the ID alone.

Several TOPS instances MAY share a host. Each TOPS MUST have isolated configuration, state, socket, policy namespace, service identity, and STAV sequence.

## Foundation Boundary

The SSIAG foundation MUST be authored exclusively in Go, MUST compile with `CGO_ENABLED=0`, MUST expose no network listener in v1, and MUST NOT dynamically link platform credential frameworks. Provider interaction occurs through explicit Go interfaces or a separately installed, versioned local IPC adapter.

The initial provider implementation order is macOS Keychain, Linux Secret Service, explicit headless Linux/NVIDIA ARM hardware or workload identity, then WSL interoperability. Provider absence or incompatibility fails closed. `pass`, plaintext files, and a locally invented vault are not implicit fallbacks.

## Provider Protocol Minimum

A provider adapter MUST declare protocol version, adapter identity/version, platform, capabilities, interaction requirements, exportability, readiness, maximum request size, and supported safe operations. Handshake and response schemas MUST be validated before operational requests. Unknown major versions, unknown operations, excess messages, timeouts, malformed responses, and identity mismatches fail closed.

The transport MUST be local and protected. Standard input/output or a local socket MAY be used. Arguments, environment variables, logs, diagnostics, and qxctl MUST NOT carry secret values. The foundation MUST sanitize adapter errors and bound request time and response size.

## Administrative Surface

qxctl MAY query status and safe provider descriptors and MAY submit authenticated administrative proposals. Administrative change MUST separate `propose` from `apply`. Agents MAY query and propose but MUST NOT apply. Apply is local-only in v1 and MUST require an authenticated canonical subject, authorization, request and correlation identifiers, TOPS binding, intent, expiry, replay protection, idempotency, expected prior-state digest, and STAV availability.

qxctl MUST NOT own SSIAG schemas, provider SDKs, runtime policy, credential material, provider execution, or STAV ledger files.

## Local Caller Authentication

Local v1 MUST authenticate Unix-socket peers using kernel-attested credentials captured from the accepted connection. macOS uses the effective peer identity exposed by its Unix-socket credential mechanism; Linux/WSL use the corresponding kernel peer-credential mechanism. Socket paths, ownership, groups, and modes are defense in depth and MUST NOT replace peer authentication.

The runtime MUST map the peer's operating-system identity to a canonical SSIAG subject through explicit per-TOPS configuration. A caller-supplied subject field is never identity evidence. Ambiguous user/group/service mappings fail closed. Workloads that share one operating-system identity require an additional ratified capability or separate service identity before they can receive distinct authority.

The local v1 reference contract maps an exact effective UID/GID pair to one canonical subject ID and kind. A configured UID/GID pair MUST map to no more than one subject, and one subject ID MUST NOT have multiple local mappings in v1. The peer PID is connection-lifetime evidence and MUST NOT be used as the stable subject key. An authenticated but unmapped peer MAY use explicitly allowlisted read-only metadata routes; it MUST fail subject resolution and therefore cannot use mutation, provider, lease, credential, or apply operations.

On Darwin, the reference implementation reads `LOCAL_PEERCRED` and requires an effective group, then captures `LOCAL_PEERPID`. On Linux and WSL, it reads `SO_PEERCRED`. Extraction occurs from the accepted connection descriptor before request dispatch. An unavailable descriptor, unsupported platform, malformed credential result, missing effective group, or invalid PID fails authentication.

Remote caller authentication, bearer tokens, OAuth/OIDC, mTLS, and network listeners are not part of local v1. No generic SSIAG token is defined.

## Foundational Supervision

SSIAG and the STAV append authority occupy the foundational bootstrap stratum. A native OS supervisor or explicit owner-provided equivalent MAY anchor them. Direct-run development remains distinct from production supervision.

Supervision owns process liveness only. Starting, stopping, monitoring, or restarting a process MUST NOT grant the supervisor SSIAG policy authority, provider authority, administrative apply authority, or ledger file access. node-troll does not inherit authority merely because it supervises another component.

The first operational macOS Keychain topology is per-user and session-aware. System or headless SSIAG MUST report the login-Keychain provider unavailable rather than silently changing scope or provider.

## Provider Trust and Secret Delivery

Before operational behavior, the Go foundation MUST verify the adapter's allowlisted path, ownership, permissions, installed digest or ratified signature identity, protocol identity, and version. The macOS Swift adapter MUST verify that its invoking SSIAG process satisfies the ratified path, ownership, and code-signing requirement before accepting operational requests. Unsigned or development identities remain metadata-only unless an explicit development policy says otherwise.

The metadata/control channel and a secret-bearing channel are distinct. General JSON control messages MAY carry safe references, operation metadata, bounded results, and safe reason codes. They MUST NOT carry secret bytes.

When policy and provider capability explicitly permit export, secret bytes MUST use a bounded one-shot inherited descriptor, socket pair, or equivalent protected local channel. The channel MUST be bound to one authorized request, have explicit size and time limits, prohibit retries/replay, close after delivery, and never surface through qxctl, arguments, environment variables, logs, STAV, OpenAPI, examples, or general diagnostics.

Non-exportable sign, assert, decrypt, or key-use operations MUST remain inside the provider and return only the bounded result authorized by policy.

## macOS Keychain Operational Profile

The first operational Keychain provider MUST be per-user and session-aware. Items MUST be scoped to the immutable TOPS ID, non-synchronizing by default, and protected by the most restrictive accessibility and user-presence policy compatible with the declared operation. System/headless processes MUST NOT implicitly use a user's login Keychain.

Where it meets the use case, non-exportable key creation and sign/assert/decrypt behavior SHOULD precede general secret export. The exact reverse-domain namespace, item classes, operation catalog, access-control matrix, signing identities, entitlements, notarization, and provisioning experience remain implementation-detail gates and MUST be recorded before operational enablement.

## STAV Projection

SSIAG MAY submit safe metadata for authentication and policy results, provider-operation lifecycle, credential rotation, enrollment, provider unavailability, and lease lifecycle. It MUST NOT submit proofs, assertions, tokens, credential values, provider payloads, secret-bearing errors, or routine heartbeat events.

## Remaining Operational Gates

Architecture is ratified for local peer authentication, foundational supervision, proposal/apply mutation, provider mutual trust, protected secret delivery, and per-user macOS Keychain use. Darwin/Linux peer extraction and exact UID/GID subject mapping are implemented for accepted SSIAG connections, with unmapped peers restricted to the existing read-only metadata surface. Runtime mutation and provider enablement still require service identities, client-side endpoint trust, request schemas, replay/idempotency limits, code-signing policy, Keychain namespace and operation policy, secret-channel framing, broader negative tests, lifecycle tests, and release evidence.

Remote SSIAG access, a network gateway, generic SSIAG tokens, implicit provider fallback, agent apply authority, and graph-database deployment remain unauthorized.
