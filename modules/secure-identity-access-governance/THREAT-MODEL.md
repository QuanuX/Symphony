# Symphony Secure Identity and Access Governance Threat Model

## Status and Method
Draft threat model for the scaffold and planned provider phases. It uses assets, actors, trust boundaries, abuse cases, and required controls. Each operational provider requires an additional provider-specific review.

## Protected Assets
- Provider authentication sessions.
- Passwords, tokens, private keys, certificates, recovery data, and decrypted values.
- Non-exportable signing/decryption capability.
- Subject identity and assurance state.
- Authorization policy and decisions.
- Credential references and leases.
- SSIAG configuration and provider manifests.
- Audit integrity.
- TOPS identity and cross-instance isolation.
- Availability of legitimate credential operations.

## Actors
- Authorized local operator.
- Authorized TOPS workload.
- qxctl process.
- SSIAG service account.
- Provider adapter.
- node-troll or Maestro when explicitly integrated.
- Local unprivileged attacker.
- Compromised authorized process.
- Malicious or compromised provider.
- Remote network attacker.
- Supply-chain attacker.
- Misconfigured administrator.

## Trust Boundaries
1. Operator or agentic tool to qxctl.
2. qxctl to local SSIAG socket.
3. SSIAG identity plane to policy plane.
4. Policy plane to credential/provider plane.
5. SSIAG to external provider.
6. SSIAG to workload delivery channel.
7. SSIAG candidate-event producer to the STAV append authority.
8. Repository source truth to build/release artifacts.

## Primary Threats and Controls

### Secret Leakage Through CLI or Logs
**Threat:** Values appear in arguments, terminal output, process listings, errors, crash reports, tests, telemetry, or audit events.

**Controls:**
- no secret-valued flags;
- structured safe error categories;
- explicit redaction tests;
- no request-body logging;
- opaque references in qxctl and SKV;
- provider-specific error wrapping;
- file-descriptor delivery preference;
- crash-dump policy documented for production.

### Socket Impersonation or Unauthorized Local Access
**Threat:** An attacker creates a fake socket, connects without authority, or replaces the SSIAG endpoint.

**Controls:**
- absolute canonical socket path;
- reject non-socket path collisions;
- restrictive parent-directory and socket permissions;
- peer-credential authentication before mutation APIs;
- service-account ownership checks;
- qxctl status displays SSIAG schema/version;
- installation manifest digest checks.

Socket permissions alone are not final caller authentication. Mutation remains disabled until the selected peer-authentication design passes review.

### Cross-TOPS Identity Confusion
**Threat:** A request, configuration, socket, provider operation, or event for one TOPS is accepted under another TOPS on the same host.

**Controls:**
- canonical lowercase UUID validation before path construction;
- display names excluded from security paths;
- one configuration/state/socket namespace per TOPS ID;
- status and qxctl identity comparison;
- future policy, lease, provider request, and STAV event binding to the same TOPS ID;
- adversarial tests with sibling TOPS instances.

### Confused Deputy
**Threat:** A legitimate qxctl or workload identity causes the SSIAG to exercise a credential for an unintended provider, audience, tenant, or operation.

**Controls:**
- bind decisions to subject, provider, reference, operation, audience, scope, and expiry;
- deny by default;
- no wildcard provider fallback;
- explicit tenant and audience validation;
- request and lease identifiers;
- provider capability checks.

### Replay
**Threat:** A captured assertion, lease, or request is reused.

**Controls:**
- short expiries;
- nonce or request identifier;
- lease single-use option;
- issuer/audience checks;
- replay cache for mutation APIs;
- clock-skew bounds;
- provider-native replay protection where available.

### Provider Downgrade
**Threat:** The SSIAG silently substitutes an exportable or weaker provider for a non-exportable/user-presence requirement.

**Controls:**
- capabilities are explicit;
- policy states minimum assurance and operation semantics;
- no silent fallback;
- downgrade attempts are denied and audited;
- compatibility changes require SCLV review.

### Malicious Provider or Adapter
**Threat:** An adapter exfiltrates material, lies about capabilities, or returns crafted errors/results.

**Controls:**
- adapter identity and digest verification;
- minimal operating-system permissions;
- explicit provider allowlist with visible evidence;
- versioned protocol;
- response limits and validation;
- adapter-specific sandboxing where available;
- independent provider security review.

### Supply-Chain Compromise
**Threat:** A Go dependency, build worker, release artifact, or provider binary is replaced.

**Controls:**
- minimal dependencies;
- module checksum and dependency review;
- reproducible-build target;
- signed release artifacts and provenance plan;
- installed binary digest manifest;
- SBOM and vulnerability scan before production release.

### Memory Exposure
**Threat:** Plaintext exists in heap copies, swap, dumps, or post-use pages.

**Controls:**
- prefer non-exportable operations;
- minimize material lifetime and copies;
- bounded buffers;
- no ordinary strings for long-lived secret material in provider implementations;
- process dump restrictions;
- locked memory where supported and reviewed;
- accurately document that perfect erasure cannot be guaranteed.

### Audit Tampering or Sensitive Audit Content
**Threat:** Events are changed, removed, reordered, or contain secret values.

**Controls:**
- one serialized append authority per TOPS sequence;
- required event identifiers and sequence/preceding-digest chaining;
- safe field allowlist;
- separate runtime audit vector from SCLV;
- restricted permissions and retention;
- fail-closed policy for high-risk operations when audit integrity is unavailable.

STAV v1 detects tampering but does not provide non-repudiation. Agents, qxctl, and event producers never edit ledger files directly.

### Denial of Service
**Threat:** Requests exhaust goroutines, file descriptors, provider sessions, interaction prompts, or rate limits.

**Controls:**
- server timeouts;
- request and response size limits;
- concurrency quotas per subject/provider;
- provider circuit breakers;
- interactive prompt rate limits;
- bounded queues;
- health separates SSIAG liveness from provider readiness.

### Unsafe Uninstall or Purge
**Threat:** Uninstall deletes unrelated files or silently destroys configuration needed for recovery.

**Controls:**
- manifest-owned absolute paths;
- expected-layout validation;
- digest comparison;
- host uninstall always preserves per-TOPS configuration/state;
- explicit one-TOPS unenroll purge and independent binary force flags;
- refuse path traversal or root-directory targets.

### Adapter Protocol Injection or Exfiltration
**Threat:** A native adapter accepts undeclared fields, receives secrets through an unsafe channel, emits extra output, lies about readiness, or is replaced on disk.

**Controls:**
- independent executable and manifest digest;
- exact allowlisted operations and strict schemas;
- bounded messages, timeouts, cancellation, and sanitized environment;
- reject extra output and protocol-major mismatch;
- no secret-valued arguments or environment variables;
- operational access remains disabled until the ratified mutual executable trust and channel-separation architecture is fully specified, implemented, and verified.

## Provider-Specific Concerns

### Native Keyrings
- Locked session behavior.
- Headless Linux without Secret Service.
- Access-control prompts and user presence.
- Platform-specific listing and deletion semantics.

### OAuth/OIDC and IAM
- Issuer, audience, tenant, redirect, and nonce validation.
- Token exchange scope escalation.
- Refresh-token storage.
- Browser/device flow phishing and session confusion.

### Passkeys and FIDO2
- Origin and relying-party binding.
- User-verification requirements.
- Synced versus device-bound credentials.
- Attestation privacy and trust policy.

### YubiKey PIV and SSH
- PIN retry exhaustion.
- Touch-policy enforcement.
- Slot and certificate lifecycle.
- Agent forwarding and remote-socket exposure.

### Remote Secret Providers
- Provider TLS identity.
- Lease revocation and renewal.
- Namespace/tenant confusion.
- Excessive provider SDK permissions.

## Implemented and Unimplemented Controls
The module does not expose credential mutation/use endpoints. Kernel peer authentication is implemented on accepted Darwin/Linux connections, including exact UID/GID subject resolution and fail-closed ambiguous mapping. SSIAG's typed STAV producer authenticates the authority endpoint, exposes no secret-bearing field, and requires a committed receipt. Foundational SSIAG supervision, qxctl-to-SSIAG endpoint authentication, provider mutual executable trust, per-user Keychain access, control/secret channel separation, and proposal/apply mutation remain ratified but unimplemented controls.

Configured production subjects, SSIAG service identity, signing requirements, Keychain item policy, secret buffers, lease replay, and provider channel framing remain release blockers for those later capabilities, not accepted residual risks.

## Security Test Gates
- Verify no secret-shaped fixture value appears in stdout, stderr, JSON, logs, or audit output.
- Verify socket path collision with a regular file fails closed.
- Verify requests without accepted-connection credential context fail closed.
- Verify exact UID/GID mappings resolve once and duplicate/ambiguous mappings are rejected.
- Verify methods other than declared read-only endpoints are rejected.
- Verify provider duplicates and invalid names fail configuration validation.
- Verify install/uninstall path and digest protections.
- Verify two TOPS identities cannot collide or cross-query.
- Verify host uninstall preserves all TOPS state and purge affects one UUID only.
- Verify the Swift adapter rejects secret-shaped unknown fields and all credential operations.
- Fuzz configuration and API decoders with size limits.
- Run race detection and static analysis.
- Execute provider-specific negative tests before enabling each provider.
