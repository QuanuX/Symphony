# Symphony Secure Identity and Access Governance Requirements

## Status
Owner-ratified architecture with phased implementation requirements. “Must” statements remain release gates for the phase that implements them; ratification alone does not enable runtime behavior.

## Product Goals
- Provide one safe command surface for identity proof, authorization, and credential use across TOPS nodes.
- Preserve the Symphony monorepo for whole-repository agent context.
- Preserve independent module installation and runtime authority.
- Prefer external secure stores and non-exportable operations over a new local vault.
- Keep qxctl and SKV free of secret material.

## Non-Goals
- Password-manager user interface.
- Cross-device vault synchronization.
- Universal identity-provider abstraction that erases provider semantics.
- Hot-path authorization calls.
- Mandatory bus, cloud, container, Python, or vendor dependency.

## Functional Requirements

### Lifecycle
- **SSIAG-F-001**: The module must build as a native Go executable from its own Go module.
- **SSIAG-F-002**: The executable must support user and system installation scopes.
- **SSIAG-F-003**: Installation must be idempotent for identical binaries.
- **SSIAG-F-004**: Installation must record the exact host binary path and installed binary digest.
- **SSIAG-F-005**: Host uninstall must always preserve every per-TOPS configuration and state root.
- **SSIAG-F-006**: Uninstall must refuse to delete a changed binary unless force is explicit.
- **SSIAG-F-007**: The module must be runnable without installing any other Symphony runtime module.
- **SSIAG-F-008**: Enrollment must require a canonical lowercase TOPS UUID and a separate non-empty display name.
- **SSIAG-F-009**: Unenrollment must preserve instance data unless per-TOPS purge is explicit.

### Topology Isolation
- **SSIAG-F-050**: Several TOPS instances must be able to share one host binary.
- **SSIAG-F-051**: Configuration, state, runtime socket, policy namespace, service identity, and STAV scope must be isolated by immutable TOPS ID.
- **SSIAG-F-052**: A display-name change must not change security identity or relocate state.
- **SSIAG-F-053**: A purge operation must validate and affect exactly one TOPS UUID.

### Control API
- **SSIAG-F-010**: The initial server must listen only on a local Unix domain socket.
- **SSIAG-F-011**: The server must reject an existing non-socket object at the configured socket path.
- **SSIAG-F-012**: The server must expose versioned status and provider-discovery endpoints.
- **SSIAG-F-013**: The server must use bounded header, read, write, and idle timeouts.
- **SSIAG-F-014**: The server must shut down gracefully on process termination signals.

### qxctl
- **SSIAG-F-020**: qxctl must discover the SSIAG socket from an explicit environment override or the selected scope and TOPS ID.
- **SSIAG-F-021**: qxctl must provide status, provider, and doctor commands.
- **SSIAG-F-022**: qxctl JSON output must be stable and versioned.
- **SSIAG-F-023**: qxctl must not import provider-specific dependencies.
- **SSIAG-F-024**: qxctl must return a nonzero exit code when the SSIAG is unavailable or unhealthy.
- **SSIAG-F-025**: qxctl status and doctor must reject a response whose TOPS ID differs from the requested ID.

### Identity and Authorization
- **SSIAG-F-030**: Every future mutation request must resolve to a subject.
- **SSIAG-F-031**: Every identity proof must state proof type, issuer, audience, issue time, expiry, and assurance attributes when applicable.
- **SSIAG-F-032**: Authorization must default to deny.
- **SSIAG-F-033**: Authorization must bind subject, operation, provider, credential reference, scope, and time.
- **SSIAG-F-034**: User-presence and interaction requirements must remain visible in capability metadata.
- **SSIAG-F-035**: Local v1 must derive operating-system caller identity from kernel-attested Unix-socket peer credentials and map it to a canonical per-TOPS subject.
- **SSIAG-F-036**: Caller-supplied subject identifiers must not be accepted as identity evidence.
- **SSIAG-F-037**: Administrative change must separate non-mutating proposal from authorized apply; AI agents must never receive apply authority.
- **SSIAG-F-038**: Apply requests must bind TOPS, subject, operation, target, request/correlation identifiers, intent, expiry, idempotency, and expected prior-state digest.

### Credential Use
- **SSIAG-F-040**: Credential references must be opaque outside the SSIAG/provider boundary.
- **SSIAG-F-041**: Leases must have an identifier, creation time, expiry, operation set, and subject binding.
- **SSIAG-F-042**: The SSIAG must prefer non-exportable provider operations.
- **SSIAG-F-043**: Provider capability absence must be explicit; the SSIAG must not emulate stronger semantics with weaker storage.
- **SSIAG-F-044**: Passkeys must be modeled as assertion capabilities, not stored passwords.
- **SSIAG-F-045**: Hardware keys must be modeled by supported operations and user-presence policy, not as retrievable private-key blobs.

## Security Requirements
- **SSIAG-S-001**: Secret values must never appear in command arguments, normal stdout/stderr, logs, qxctl JSON, SKV surfaces, provider descriptors, installation manifests, or audit records.
- **SSIAG-S-002**: Configuration files must contain references and policy only.
- **SSIAG-S-003**: The scaffold must ship with no plaintext, environment, or test credential provider enabled.
- **SSIAG-S-004**: Runtime directories and sockets must be created with least-privilege permissions.
- **SSIAG-S-005**: Mutation APIs must not be enabled until local peer authentication is implemented and tested.
- **SSIAG-S-006**: Provider failures must be translated into safe categories before logging or returning them.
- **SSIAG-S-007**: Requests must carry unique identifiers suitable for audit correlation without containing sensitive data.
- **SSIAG-S-008**: Short-lived material must be erased or released as soon as practical; no guarantee of perfect memory erasure may be claimed.
- **SSIAG-S-009**: Credential delivery through environment variables or files must require explicit policy and threat review.
- **SSIAG-S-010**: Development modes and allowlists must produce visible evidence and must never silently bypass identity or policy.
- **SSIAG-S-011**: Provider manifests and adapter executables must be authenticated before use.
- **SSIAG-S-012**: Rate limits and replay protection must precede any remotely reachable mutation API.
- **SSIAG-S-013**: Plaintext files, environment values, `pass`, and a locally invented vault must never be implicit provider fallbacks.
- **SSIAG-S-014**: AI agents must not edit STAV ledgers or submit arbitrary ledger events.
- **SSIAG-S-015**: Socket permissions must remain defense in depth and must not substitute for peer-credential authentication.
- **SSIAG-S-016**: Provider control messages must not carry secret bytes.
- **SSIAG-S-017**: An explicitly exportable secret must use a request-bound, bounded, one-shot protected local descriptor/channel and must never traverse qxctl, OpenAPI, STAV, arguments, environment variables, or logs.
- **SSIAG-S-018**: The macOS adapter must authenticate the invoking SSIAG executable under the ratified path, ownership, and code-signing policy before operational access.

## Operational Requirements
- **SSIAG-O-001**: Every provider must report declared, ready, degraded, locked, unavailable, or disabled status without revealing sensitive detail.
- **SSIAG-O-002**: Health must distinguish process liveness from provider readiness.
- **SSIAG-O-003**: Audit sinks must fail according to declared policy: fail-closed for high-risk operations and configurable for read-only inspection.
- **SSIAG-O-004**: Clock skew assumptions must be explicit for leases and federated tokens.
- **SSIAG-O-005**: Rotation must support overlapping credential versions and rollback.
- **SSIAG-O-006**: Backup guidance must exclude provider-held secret values unless the provider's own supported backup mechanism is used.
- **SSIAG-O-007**: Per-TOPS purge must be explicit and independently auditable.
- **SSIAG-O-008**: Safe SSIAG security outcomes must be submitted only through the dedicated per-TOPS Go STAV append authority after its remaining implementation gates pass.
- **SSIAG-O-009**: Security, provider, credential, policy, and configuration apply operations must fail closed when their required STAV append cannot be accepted.
- **SSIAG-O-010**: Supervision must own liveness only and must not confer SSIAG policy, provider, apply, or STAV ledger authority.

## Portability Requirements
- **SSIAG-P-001**: All Symphony-authored SSIAG foundation source must be Go and must compile without cgo.
- **SSIAG-P-002**: The first operational provider must be the independently installed macOS Keychain adapter after its security gate passes.
- **SSIAG-P-003**: Provider-specific native dependencies must be isolated and optional.
- **SSIAG-P-004**: Containers and Kubernetes must not be required.
- **SSIAG-P-005**: Bus traversal must remain optional.
- **SSIAG-P-006**: Foundational SSIAG/STAV supervision must use an explicit native OS or owner-provided bootstrap adapter; the module library must not assume one universal service manager.
- **SSIAG-P-007**: A Symphony-authored non-Go provider adapter must remain a separate executable behind explicit protected IPC.
- **SSIAG-P-008**: Provider implementation order must be macOS Keychain, Linux Secret Service, explicit headless Linux/NVIDIA ARM hardware or workload provider, then WSL interoperability.
- **SSIAG-P-009**: The first operational macOS Keychain topology must be per-user and session-aware; system/headless mode must report the provider unavailable without fallback.

## SKV Requirements
- **SSIAG-K-001**: SKVI must index every canonical SSIAG contract and design Markdown surface.
- **SSIAG-K-002**: SCLV must not record a merge before actual PR and merge evidence exists.
- **SSIAG-K-003**: SCLV must list compatibility, publication, and rollback consequences for provider/protocol changes.
- **SSIAG-K-004**: SODV must remain the publication authority.
- **SSIAG-K-005**: NotebookLM responses must be treated as context and checked against repository source truth.
- **SSIAG-K-006**: Runtime audit events must not be written into SCLV.
- **SSIAG-K-007**: `knowledge/ssiag/` must remain canonical authority for SSIAG protocol and relationship truth.
- **SSIAG-K-008**: `knowledge/stav/` must remain canonical authority for STAV schema and append protocol truth.
- **SSIAG-K-009**: `knowledge/sacv/` must govern HTTP API-contract policy and registry; no SSIAG endpoint document may be created without ratified transport and security ownership.

## Acceptance Criteria for This Scaffold
- All Go packages compile and tests pass under the declared Go baseline.
- The local server exposes only status and provider metadata.
- No operational credential provider exists.
- User install is idempotent, uninstall is digest-safe, and multiple TOPS enrollments remain isolated.
- qxctl can report SSIAG status and providers through the local socket.
- Every accepted Darwin/Linux connection must carry kernel peer credentials before request dispatch.
- Exact UID/GID subject mappings must validate as one-to-one; an unmapped peer cannot resolve a mutation subject.
- Existing qxctl inventory/status tests account for the new module.
- symphony-validator reports no new violation.
- Documentation contains the architecture, requirements, threat model, and phased implementation procedure.

## Remaining Detail Gates Before Operational Provider Work
1. Define launchd/service-manager identities, ownership, restart bounds, direct-run bootstrap behavior, and qxctl-to-server endpoint trust.
2. Scaffold the ratified dedicated Go STAV append-authority namespace, then separately ratify schema content, serialization, durability, recovery, retention, rotation, and repair before enabling it.
3. Freeze mutation proposal/apply schemas, replay/idempotency bounds, expiry/skew, and expected-state conflict behavior.
4. Freeze provider mutual executable trust, signing requirements, secret-channel framing, memory handling, and crash-dump policy.
5. Define the macOS Keychain item namespace, item/operation catalog, access-control matrix, entitlements, notarization, and provider-owned provisioning experience.
