# Symphony Foundational Service Lifecycle

## Authority

This Architect-ratified common contract governs the cold/freezing-path administration envelope shared by the independently installed SSIAG and STAV Go foundations. Common `knowledge/` owns the envelope, recovery invariants, identifiers, and temporal rules. `knowledge/ssiag/` and `knowledge/stav/` retain all domain semantics. qxctl implements the preferred headless client but is neither required for direct module administration nor permitted to reimplement module behavior.

The four governed lifecycle surfaces are:

- SSIAG TOPS enrollment;
- SSIAG native supervision;
- STAV TOPS enrollment; and
- STAV native supervision.

This contract is a narrow reviewed exception to the generic lifecycle prohibition on arbitrary entry-point execution and live service activation. It does not widen the generic adapter registry. Only the exact SSIAG and STAV module-owned lifecycle adapters, proven by their installation evidence, may be invoked.

## Stable Identities

Each qxctl family exposes `status`, `plan`, `apply`, `apply-status`, and `recover`. Every leaf requires an exact installation `--prefix`; `--version` selects one immutable package when that prefix contains more than one compatible version. Omitting the version is allowed only when exactly one compatible package exists. qxctl never chooses the greatest, newest, or most recently modified version. Executable leaves use `qxcmd:symphony:<component>.<surface>.<leaf>`. Module-owned adapter operations use `engop:symphony:<component>.<surface>.<operation>`. Feature identities remain independent `ssfv:` records. Grammar, command identity, backend-operation identity, caller operation identity, and feature identity are never interchangeable.

The four qxctl families are:

```text
qxctl ssiag enrollment status|plan|apply|apply-status|recover
qxctl ssiag supervisor status|plan|apply|apply-status|recover
qxctl stav enrollment status|plan|apply|apply-status|recover
qxctl stav supervisor status|plan|apply|apply-status|recover
```

## Ownership Boundary

The installed module adapter owns canonical paths, configuration interpretation, enrollment behavior, descriptor rendering, native-manager interaction, socket lifecycle safety, and module-specific readiness. qxctl owns Cobra/Viper grammar, exact executable selection, bounded process invocation, request/result validation, feature bindings, and presentation. qxctl MUST NOT import module-internal packages, parse human output, invoke `serve`, discover arbitrary executables, render service descriptors, or substitute its own filesystem mutation.

The human module CLI and machine adapter MUST call the same transaction implementation. Direct administration without qxctl remains supported and cannot bypass attempt, expected-state, purge-safety, or recovery invariants.

## Invariant Ownership and Evidence

`knowledge/INVARIANTS.md` defines the common lowest-authoritative-layer rule, and `knowledge/INVARIANT-OWNERSHIP.json` assigns stable IDs to this contract's adapter provenance, bounded process framing, stable observation digests, plan/attempt compare-and-swap, evidence-driven recovery, audit closure, and shared transaction-core invariants. Those IDs are protocol identifiers; neither the Go modules, qxctl, the C++ engine, nor AI derives them from display names.

For each registered invariant, the owning module MUST keep a producer regression beside its implementation and every consuming trust boundary MUST retain a negative rejection test. Because the machine adapter crosses stdin/stdout IPC, each allowed adapter MUST also be exercised as a real receipt-backed child process. An in-process decoder or engine test alone cannot prove executable selection, argument forwarding, clean standard output, environment independence, framing, or exit propagation.

## Machine Adapter

The adapter is invoked from the exact installed module executable using protected standard input and standard output. The invoking client supplies a bounded JSON command, an empty or contract-defined environment, a hard monotonic deadline, no repository working-directory dependency, and bounded output. The adapter emits exactly one bounded JSON result and no human diagnostics on standard output.

The descriptor publishes component/build identity, exact executable and installation-evidence digests, supported operations, scopes, native managers, limits, configuration/runtime/persistent-state compatibility, and rollback readability. Capability compatibility, not semantic-version recency, controls invocation.

An old adapter that does not publish this protocol remains preserved. A newer client reports an exact compatibility blocker and never scrapes its legacy human output. The historical fixed-path install-v1 commands remain a qxctl-free compatibility surface and are not silently promoted into exact package evidence. A compatible staged adapter may observe supported legacy state and migrate forward. New adapters preserve existing human CLI grammar so older clients and qxctl-free installations retain their established administration path; a recognized legacy flag may fail closed when its former behavior cannot satisfy the journaled transaction contract.

## Desired and Observed State

Enrollment v1 desired states are `enrolled` and `unenrolled_preserved`. Purge is deliberately absent from qxctl v1. A module-native purge remains explicit and MUST first satisfy the stronger socket-lock, manager, journal, and recovery rules owned by that module.

Supervisor v1 desired states are `native_running`, `native_installed_stopped`, and `absent_stopped`. `externally_managed` is observable but not silently mutated. Observation separates descriptor presence, manager availability, enablement, process state, endpoint readiness, activated package identity, and activation generation. `manager_unavailable`, `degraded`, `failed`, `externally_managed`, and `recovery_required` are explicit states rather than aliases for success.

Plan binds the exact observation digest and desired state. Apply binds the immutable plan digest and exact expected attempt state. No qxctl route exposes `--force`, `--no-stop`, arbitrary service-manager arguments, or an implicit newest-version selector.

## Target-Host Authority

Mutation and recovery use `target_host_permission`. User scope derives authority from the invoking kernel UID/GID and the protected user-scope lifecycle boundary. System scope requires effective UID zero and exact explicit service or authority UID/GID wherever the domain enrollment requires them. Caller class, supplied subject, AI or human status, repository ownership, source-control identity, and command possession confer no authority.

SSIAG cannot be a mandatory prerequisite for its own initial enrollment or activation. STAV cannot be a mandatory prerequisite for its own initial enrollment or activation. If SSIAG evidence is available, an owning contract may require it as an additional caller-neutral safeguard for a non-bootstrap operation; it never replaces the foundational target-host rule.

## Attempt and Recovery

Before external mutation, the module transaction engine writes a protected digest-linked attempt outside any module subtree that the operation may remove. The attempt binds component, surface, scope, TOPS UUID, stable operation/request/correlation IDs, exact executable and installation evidence, prior observation, desired state, immutable plan, completed phase, predecessor, audit state, and STSC timestamps.

`stable_state_digest` is the tagged SHA-256 of the compact lexicographically key-sorted observation object with `observed_at`, `stable_state_digest`, and `observation_digest` omitted. `observation_digest` is the tagged SHA-256 of the same canonical object with only `observation_digest` omitted, and therefore binds the stable-state digest plus documentary observation time. Implementations and qxctl independently recompute both values.

The implementation re-observes after every external action and closes only from exact evidence:

- if the desired state is already proven, close as an idempotent replay;
- if the expected predecessor is still proven, retry the prepared action;
- if one digest-linked successor is proven, recover forward;
- if state is partial, divergent, tampered, ambiguous, or unsupported, remain `recovery_required` and fail closed.

Timestamp, semantic version, filesystem modification time, and process age never select recovery direction. A retry reuses the stable operation identity. Reusing an operation identity with different evidence is an error.

## Installation and Version Compatibility

SSIAG and STAV current packages use immutable receipt-v2 ownership beneath versioned `libexec/symphony/<module>/<version>/` paths and may coexist side by side within one installation prefix. Their receipt-owned `ssiag.foundation-lifecycle` and `stav.foundation-lifecycle` adapter entry points bind the command protocol and exact executable. A supervisor descriptor pins an immutable executable path, digest, receipt, and activation generation. No service descriptor targets a mutable `current` binary. Superseded packages remain installed by default for rollback and bespoke topology.

Readers preserve supported legacy SSIAG install-v1 and STAV pre-receipt evidence without rewriting it on read. New writes use current evidence. Package removal refuses while any TOPS descriptor, running service, retained lifecycle claim, or unresolved attempt references that package. Candidate activation is refused before stopping the prior service when configuration, runtime API, state, policy-overlay, or ledger compatibility cannot prove rollback safety. STAV ledger bytes and digest history are never rewritten by a lifecycle transition.

## Audit and Deferred Recovery

Every foundational mutation first records protected local attempt evidence. Ordinary audited mutations use the closed SSIAG-to-STAV producer path and release success only with the required receipt. qxctl never receives a raw STAV producer grant, constructs arbitrary candidates, or edits ledger files.

Bootstrap, self-impacting teardown, or recovery may use `audit_deferred` only when the caller explicitly selects that path and the target-host permission and expected-state checks succeed. It is never an automatic downgrade. The result remains `reconciliation_required` until the closed producer appends the later lifecycle outcome and the attempt binds the exact receipt. Reconciliation records its real later timestamp and original operation/correlation identities; it never backdates the append or impersonates the original event time.

An unreconciled attempt cannot be discarded, relabeled as committed, or used as proof of completed permanent retirement. Operations that would make reconciliation permanently impossible remain blocked pending a separately ratified retirement contract.

## Temporal and Thermal Boundary

All durable instants follow `knowledge/TIME.md`. The target TOPS owns durable commit time. Canonical machine timestamps use UTC; local time is presentation only. Live deadlines and elapsed measurements use a monotonic clock. Wall-clock text is evidence, not identity or causal order.

This entire surface is cold/freezing path. It MUST NOT execute inline with hot or warm work, acquire locks shared with hot/warm execution, or make hot/warm progress synchronously depend on lifecycle, SSIAG, or STAV availability.

## Platform Boundary

Native managers are launchd on supported macOS development hosts and systemd on supported Linux hosts. Missing user managers, absent user buses, launchd-domain unavailability, WSL without supported systemd, and owner-managed processes produce explicit unavailable or external states. No native Windows engine, service manager, or fake portable supervisor is authorized. Windows users operate through WSL or a remote TOPS node under a future transport contract.

The current qxctl command target is `local`. A future remote qxctl transport may execute the same stable operation on a target TOPS node; it must not change the command identity, allow the client host to dictate durable time, or treat remote possession as target-host permission.
