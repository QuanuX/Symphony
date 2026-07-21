# Symphony Secure Identity and Access Governance Specification

## Status and Normative Terms

Architect-ratified v1 architecture and relationship contract. MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are normative when the related capability is implemented. A ratified architecture does not become operational merely because its contract is described.

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

qxctl MAY query status and safe provider descriptors and MAY submit authenticated administrative proposals. Administrative change MUST separate `propose` from `apply`. Apply is local-only in v1 and MUST require an authenticated canonical subject, effective target-host permission, request and correlation identifiers, TOPS binding, operation and resource, intent, expiry, replay protection, idempotency, proposal digest, expected prior-state digest, and the applicable audit contract. No apply route is implemented by the present foundation.

Authorization MUST NOT request, infer, store, or evaluate whether a caller is human, AI, agentic, a service, a workload, an organization, or another actor type. Callers with the same effective host permission and operation context receive the same supported authorization result. The target-host administrator MAY inspect, explain, enable, disable, reset, export, import, or replace caller-neutral configurable safeguards through qxctl when that surface is implemented; SSIAG MUST NOT establish a separate enrollment authority superior to host administration.

Confirmations, quorum, delay, maintenance windows, budgets, step-up assurance, executable trust, workload attestation, and similar interlocks are configurable safeguards. Path safety, bounded parsing, atomic writes, expected-state validation, ledger framing, and secret exclusion are protocol integrity and MUST NOT be exposed as optional safeguard toggles. A direct safeguard profile MAY remove optional governance interlocks but MUST preserve protocol integrity.

Ordinary audited mutation MUST fail closed when its required STAV append is unavailable. A target-host administrator MUST have an explicit audit-deferred recovery operation once that contract is implemented. The recovery operation MUST remain permission-backed, bind the same operation and expected state, preserve protocol integrity, write durable local recovery evidence before completing, mark the outcome audit-deferred, and reconcile that evidence into STAV when service returns. It MUST NOT become a silent fallback or a secondary ledger writer.

The default administrative authority session begins after successful authentication and ends at logout, credential/session expiry, revocation, or required re-authentication. Re-authentication begins a new authority epoch. An administrator MAY select another supported lifecycle policy through qxctl, but SSIAG MUST NOT project authority beyond an invalidated authentication boundary. Repository worktree reconciliation contexts may bind to one authority epoch without becoming identity or permission evidence themselves.

SSIAG authorization for vector engines, qxctl administration, audit-deferred recovery, and later reconciliation is administrative cold/freezing-path work. It MUST NOT execute inline with a hot or warm path, acquire locks shared with hot/warm execution, make hot/warm progress synchronously depend on SSIAG/STAV availability, or otherwise add blocking, jitter, or latency there.

qxctl MUST NOT own SSIAG schemas, provider SDKs, runtime policy, credential material, provider execution, or STAV ledger files.

## Local Caller Authentication

Local v1 MUST authenticate Unix-socket peers using kernel-attested credentials captured from the accepted connection. macOS uses the effective peer identity exposed by its Unix-socket credential mechanism; Linux/WSL use the corresponding kernel peer-credential mechanism. Socket paths, ownership, groups, and modes are defense in depth and MUST NOT replace peer authentication.

The runtime MUST map the peer's operating-system identity to a canonical SSIAG subject through explicit per-TOPS configuration. A caller-supplied subject field is never identity evidence. Ambiguous user/group/service mappings fail closed. Workloads that share one operating-system identity require an additional ratified capability or separate service identity before they can receive distinct authority.

The local v1 reference contract maps an exact effective UID/GID pair to one canonical subject ID and kind. A configured UID/GID pair MUST map to no more than one subject, and one subject ID MUST NOT have multiple local mappings in v1. The peer PID is connection-lifetime evidence and MUST NOT be used as the stable subject key. An authenticated but unmapped peer MAY use explicitly allowlisted read-only metadata routes; it MUST fail subject resolution and therefore cannot use mutation, provider, lease, credential, or apply operations.

On Darwin, the reference implementation reads `LOCAL_PEERCRED` and requires an effective group, then captures `LOCAL_PEERPID`. On Linux and WSL, it reads `SO_PEERCRED`. Extraction occurs from the accepted connection descriptor before request dispatch. An unavailable descriptor, unsupported platform, malformed credential result, missing effective group, or invalid PID fails authentication.

The same per-TOPS authentication configuration MUST name one canonical SSIAG service identity separately from caller subjects and bind it to an explicitly present effective UID/GID pair. Local v1 uses service subject ID `symphony.ssiag.service` and kind `symphony.identity.service`; configuration cannot rename either value. New user enrollment binds that service identity to the enrolling process effective UID/GID. New system enrollment requires administrator-authored exact service UID and GID values and MUST NOT infer root as a universal service identity. Re-enrollment preserves an existing service mapping and rejects a conflicting replacement.

Before changing a runtime path, removing a stale socket, or listening, the SSIAG process MUST verify that its effective UID/GID exactly matches the configured service identity. A local client MUST load endpoint trust from the immutable-TOPS configuration, reject a symbolic-link or non-regular final component, reject unsafe scope-specific ownership or write permissions, require the canonical socket path unless an explicit absolute location override is used, and verify that the pre-dial object is a Unix socket owned by the configured service UID. After dialing and before sending application bytes, it MUST verify the connection's kernel-attested peer UID/GID exactly matches the configured service UID/GID. The post-dial check is authoritative; path ownership, modes, and access groups remain defense in depth and MAY control reachability without becoming identity evidence. A socket-location override MUST NOT override endpoint identity.

User trust configuration is effective-user-owned and owner-only. System trust configuration contains no secret material, remains administrator-owned, is never writable by group or other, and MAY be readable by the separately identified service and authorized clients. Runtime-directory ownership, service-account provisioning, and supervisor labels remain explicit installation concerns rather than implied endpoint authority.

Remote caller authentication, bearer tokens, OAuth/OIDC, mTLS, and network listeners are not part of local v1. No generic SSIAG token is defined.

## Foundational Supervision

SSIAG and the STAV append authority occupy the foundational bootstrap stratum. A native OS supervisor or explicit owner-provided equivalent MAY anchor them. Direct-run development remains distinct from production supervision.

Supervision owns process liveness only. Starting, stopping, monitoring, or restarting a process MUST NOT grant the supervisor SSIAG policy authority, provider authority, administrative apply authority, or ledger file access. node-troll does not inherit authority merely because it supervises another component.

The implemented native profiles are per-TOPS launchd jobs on macOS and per-TOPS systemd units on Linux. The stable launchd label prefixes are `io.github.quanux.symphony.ssiag.` and `io.github.quanux.symphony.stav.` followed by the immutable TOPS UUID. Linux unit names are `symphony-ssiag@<tops-id>.service` and `symphony-stav@<tops-id>.service`. No SSIAG unit requires, wants, or starts STAV: each service starts independently. Future ordinary audited mutation fails closed at the operation boundary if STAV is unavailable; the separately permissioned audit-deferred administrator recovery contract above remains the only planned exception.

System service accounts are an owner or package-manager prerequisite. Symphony MUST NOT silently create an account, infer root, or treat an account name as endpoint authority. Supervisor installation consumes the numeric UID/GID already recorded in the per-TOPS trust configuration and fails when that identity is not provisioned where the native manager requires a name. Shared system parents are administrator-owned and traversable; the SSIAG and STAV runtime/state children are separately owned and `0700`. Linux runtime children live below `/run/symphony/<tops-id>/`; macOS uses its native `/var/run/symphony/<tops-id>/` root.

The Go processes own their sockets rather than using supervisor socket activation. After process-identity verification and before stale-socket inspection, each process acquires a non-blocking exclusive lock on a persistent adjacent `.lock` regular file. It refuses an active or foreign endpoint, removes only a socket whose failed connection proves the permitted stale cases, drains bounded in-flight work on SIGTERM, removes its socket, and releases the lock last. The lock file is not unlinked because inode replacement would defeat serialization.

Native profiles restart only on failure, use a five-second systemd restart delay with five attempts per minute, or the launchd ten-second throttle, and allow ten seconds for SIGTERM shutdown. Direct user-scope `serve` is a development/diagnostic mode. System scope requires the installed supervisor or an explicit owner-controlled equivalent to assert `--supervised`; this flag records launch intent and is not an authorization credential. `--no-start` and `--no-stop` allow an owner-provided equivalent to consume or remove the generated descriptor without making systemd universal.

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

The SSIAG v1 producer vocabulary is closed to the following event and operation pairs:

| Meaning | Event class | Operation ID | Intent ID |
|---|---|---|---|
| authentication decision | `symphony.ssiag.authentication.decision` | `symphony.ssiag.authenticate` | `symphony.ssiag.authentication.evaluate` |
| policy decision | `symphony.ssiag.policy.decision` | `symphony.ssiag.authorize` | `symphony.ssiag.policy.evaluate` |
| provider operation | `symphony.ssiag.provider.operation` | `symphony.ssiag.provider.execute` | `symphony.ssiag.provider.execute` |
| credential rotation | `symphony.ssiag.credential.rotation` | `symphony.ssiag.credential.rotate` | `symphony.ssiag.credential.rotate` |
| enrollment lifecycle | `symphony.ssiag.enrollment.lifecycle` | `symphony.ssiag.enrollment.change` | `symphony.ssiag.enrollment.change` |
| lease issuance | `symphony.ssiag.lease.lifecycle` | `symphony.ssiag.lease.issue` | `symphony.ssiag.lease.issue` |
| lease revocation | `symphony.ssiag.lease.lifecycle` | `symphony.ssiag.lease.revoke` | `symphony.ssiag.lease.revoke` |

The corresponding v1 outcome-to-reason mappings are also closed:

- authentication: `allowed`, `denied`, `failed`, or `unavailable` maps to `symphony.ssiag.authentication.<outcome>`;
- policy: `allowed`, `denied`, `failed`, or `unavailable` maps to `symphony.ssiag.policy.<outcome>`;
- provider: `succeeded`, `failed`, or `unavailable` maps to `symphony.ssiag.provider.<outcome>`;
- credential rotation: `succeeded`, `failed`, or `unavailable` maps to `symphony.ssiag.credential.rotation.<outcome>`;
- enrollment: `succeeded` or `failed` maps to `symphony.ssiag.enrollment.<outcome>`;
- lease issuance: `succeeded` maps to `symphony.ssiag.lease.issued`, while `failed` maps to `symphony.ssiag.lease.failed`;
- lease revocation: `succeeded` maps to `symphony.ssiag.lease.revoked`, while `failed` maps to `symphony.ssiag.lease.failed`.

The SSIAG producer constructs the candidate from typed safe references and closed outcome/reason mappings, never accepts arbitrary event class or operation values, and never assigns producer identity, event identity, ordering, or integrity. The STAV append authority authenticates the SSIAG operating-system identity, assigns the configured producer identity, enforces the exact pair allowlist, and returns a durable receipt. A caller requiring audit MUST treat a transport failure, endpoint-identity mismatch, local rejection, or non-committed receipt as failure; it MUST NOT spool or write the ledger directly.

## Remaining Operational Gates

Architecture is ratified for local peer authentication, foundational supervision, proposal/apply mutation, provider mutual trust, protected secret delivery, and per-user macOS Keychain use. Darwin/Linux peer extraction, exact UID/GID subject mapping, endpoint trust, native launchd/systemd supervision, runtime ownership, bounded restart/shutdown, and serialized stale-socket recovery are implemented. Unmapped peers remain restricted to the existing read-only metadata surface. Mutually authenticated STAV submission and the closed SSIAG producer vocabulary are implemented as the audit dependency for future operations. Runtime mutation and provider enablement still require their request schemas, policy execution, replay limits, code-signing policy, Keychain namespace and operation policy, secret-channel framing, broader mutation/provider negative tests, lifecycle tests, and release evidence.

Remote SSIAG access, a network gateway, generic SSIAG tokens, implicit provider fallback, non-permission-backed apply authority, and graph-database deployment remain unauthorized.
