# Symphony Secure Identity and Access Governance Architecture

## Architectural Outcome

SSIAG is a node-local warm-path security foundation in the Symphony monorepo. It implements one complete decision system while keeping each concept distinct:

```text
proof -> subject -> policy decision -> capability
      -> reference or lease -> provider operation -> safe STAV outcome
```

The monorepo is intentional: agents and humans can inspect doctrine, runtime modules, qxctl, adapters, SKV relationships, and validation evidence together. Deployment remains modular. The Go foundation and every native adapter are independently buildable and installable.

## Authority Map

| Surface | Authority | Explicitly not |
|---|---|---|
| `knowledge/ssiag/` | SSIAG protocol, vocabulary, relationships, extensions | runtime state or executable schema owner |
| Go SSIAG module | foundation implementation and per-TOPS metadata API | native framework host or canonical source truth |
| provider module | reviewed platform operation implementation | policy authority or fallback selector |
| `knowledge/stav/` | audit envelope and append protocol truth | operational ledger |
| append authority | event validation, sequencing, durable append | schema author or agent endpoint |
| qxctl | administrative/query interface | provider, schema owner, ledger editor, secret holder |
| SKVI | relationship index | source-truth creator |

## Host and TOPS Topology

One installation is shared; instances are not:

```text
TOPS host
├── symphony-ssiag                         one installed Go binary
├── TOPS 018f...0002
│   ├── config/<id>/ssiag/config.json
│   ├── state/<id>/ssiag/
│   ├── runtime/<id>/ssiag.sock
│   └── state/<id>/stav/                  future independent ledger
├── TOPS 018f...0003
│   ├── config/<id>/ssiag/config.json
│   ├── state/<id>/ssiag/
│   ├── runtime/<id>/ssiag.sock
│   └── state/<id>/stav/
└── symphony-ssiag-provider-macos-keychain optional shared adapter binary
```

IDs are immutable opaque UUIDs. Names are mutable display metadata. Concatenated `<uuid>-<name>` values are unsuitable for canonical identity because a rename would either change identity or require parsing ambiguity. SSIAG stores the fields separately and uses the UUID only in paths and security decisions.

## Install Versus Enrollment

Host installation owns only the SSIAG executable and install manifest. TOPS enrollment owns one configuration, state directory, enrollment manifest, and runtime socket namespace. Therefore:

- installing again updates one binary, not every configuration;
- adding another TOPS does not duplicate the binary;
- host uninstall cannot destroy TOPS state;
- per-TOPS purge cannot target a sibling instance;
- display-name updates do not move files.

SSIAG and the STAV append authority occupy a foundational bootstrap stratum. A native OS supervisor or explicit owner-provided equivalent anchors them in production; direct-run remains a separate development mode. Supervision owns liveness only. It does not confer policy, provider, apply, producer, or ledger authority, and node-troll does not inherit authority by supervising another component.

## Process Components

### Command Entrypoint

`cmd/symphony-ssiag` parses lifecycle and metadata commands. It requires a TOPS ID for every instance operation and accepts an environment fallback only through `SYMPHONY_SSIAG_TOPS_ID`.

### Paths and Lifecycle

`internal/paths` validates scopes and canonical UUIDs and resolves host versus instance layouts. `internal/lifecycle` uses digest-bearing manifests, atomic replacement, restrictive permissions, non-regular-file rejection, explicit force, and explicit per-TOPS purge.

### Configuration

`internal/config` reads bounded strict JSON and rejects unknown fields. The topology object contains separate `id` and `name`. The optional compatibility-safe `authentication` block fixes the mechanism to `unix_peer_credentials` and maps exact UID/GID pairs to unique canonical subject IDs and kinds. New enrollments write the block with an explicit empty subject array. Provider entries are safe declarations, never credentials.

### Metadata API

`internal/server` exposes `GET /v1/status` and `GET /v1/providers` over a local Unix socket. Before dispatch it captures kernel credentials from the accepted connection and rejects requests without authenticated peer context. It returns schema, runtime version, mode, TOPS identity/display name, readiness, and allowlisted descriptors. TCP and mutation endpoints are absent.

`internal/peerauth` uses build-tagged, cgo-free wrappers: Darwin `LOCAL_PEERCRED` plus `LOCAL_PEERPID`, and Linux/WSL `SO_PEERCRED`. Mapping uses the exact effective UID/GID pair. PID is retained only as connection evidence. An unmapped authenticated peer may query the current safe metadata routes but `SubjectFromContext` fails closed, preserving the mutation gate.

### Decision Packages

`internal/identity`, `internal/policy`, and `internal/credential` keep proof summaries, subjects, deny-by-default decisions, references, leases, and operations separate. Present types are scaffolding; they do not establish authenticated sessions or perform operations.

### Provider Registry

`internal/provider` validates safe declarations. A declared provider is not ready. Operational readiness requires a real adapter, protocol negotiation, executable trust, caller authentication, and provider-specific review.

## Provider Boundary

All foundation source is Go and cgo is prohibited. Provider-native frameworks never link into the foundation process. Integration options are:

1. a portable Go implementation using reviewed native protocols without cgo; or
2. a separately built and installed adapter reached over protected local IPC.

The macOS Keychain adapter uses option 2. It is Swift because Apple's native framework boundary is outside the foundation. Its present scaffold accepts only `hello`, `status`, and `capabilities`, rejects unknown fields, bounds messages, and reports operational access disabled.

The planned sequence is macOS Keychain, Linux Secret Service, an explicit headless Linux/NVIDIA ARM hardware or workload provider, then WSL interoperability. Missing providers fail closed. No plaintext, `pass`, or invented local vault fallback is allowed.

## Trust Boundaries

### qxctl to SSIAG

The socket is local and permission-restricted, and every accepted connection is authenticated using kernel-attested Unix-socket peer credentials. Explicit per-TOPS UID/GID mappings resolve canonical subjects; absent or ambiguous mappings cannot yield mutation authority. The service verifies its configured process identity before runtime mutation, while qxctl and the self-client verify the configured kernel-attested server identity before application exchange. Permissions remain defense in depth. Status and provider metadata are the only enabled operations. Future administrative change separates proposal from apply. Agents may query and propose only. Apply still waits for a mapped subject, authorization, replay protection, idempotency, expected-state binding, and audit.

### SSIAG to Adapter

The future foundation must verify the configured executable path, ownership, digest/signature, adapter identity, protocol major version, supported operation, request size, deadline, response size, and response schema. The macOS Swift adapter must verify the invoking SSIAG process under the ratified path, ownership, and code-signing policy. Secret values cannot travel in the JSON control envelope, arguments, environment variables, logs, qxctl, OpenAPI, or STAV. Explicitly exportable bytes use a request-bound bounded one-shot descriptor/channel. Non-exportable operations remain inside the provider.

### SSIAG to Workload

The preferred order is provider-executed non-exportable operation, short-lived audience-bound result/capability, protected one-shot descriptor/channel, then an exceptional explicitly reviewed materialization path. No secret delivery mechanism exists in the scaffold.

### SSIAG to STAV

SSIAG will be a candidate-event producer, never a ledger writer. One dedicated Go append-authority process per TOPS serialization domain authenticates producers and assigns ledger identity, timestamp, sequence, and preceding digest. Only safe result metadata crosses the boundary; execution context stays private.

## qxctl Integration

The command group is lowercase and TOPS-scoped:

```text
qxctl ssiag status --tops-id UUID [--scope user|system] [--json]
qxctl ssiag providers --tops-id UUID [--scope user|system] [--json]
qxctl ssiag doctor --tops-id UUID [--scope user|system]
```

qxctl has no third-party dependencies and is provider-neutral. It uses the authority-free first-party STAV protocol kernel only for STAV contract mechanics. It verifies status responses are for the requested TOPS. It must not bypass SSIAG, accept secret flags, call Keychain APIs, edit STAV files, or own schemas.

## Failure Model

- invalid identity: reject before path resolution;
- path collision or symlink target: reject rather than replace;
- changed installed binary: require explicit force;
- missing enrollment: refuse to serve;
- returned TOPS mismatch: qxctl and direct CLI fail;
- missing/incompatible provider: fail closed;
- malformed or oversized adapter message: terminate/reject safely;
- unavailable append authority: fail closed for security, provider, credential, policy, and configuration apply; never invent a secondary writer or producer-side spool.

## Compatibility

All schemas and provider protocols are versioned. Additive metadata may be introduced within a major version only when older consumers remain safe. Credential references stay opaque. Path, identity, provider-protocol, and STAV changes require migration/rollback design and real SCLV evidence after merge.
