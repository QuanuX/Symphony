# Prima Parte Node-Local State Witness

## Status

Architect-ratified Phase 3 concept. This document records its narrow SOV boundary; no implementation is claimed.

## Purpose

Prima Parte is an optional, extremely lightweight C++ executable placed on a remote Node administered through qxctl. It is the Node-local state witness and qxctl-addressable receptor surface for its own Node. Here, `receptor surface` describes an explicit qxctl invocation target; it is not a Phase 1 Maestro receptor, receives no `receptor:` identity, and acquires no docking or presence semantics by implication. Its orchestral name suggests a section leader at the deployment point, but it is not a smaller running Maestro and inherits none of Maestro's broader responsibilities.

## Runtime Character

Prima Parte has zero resident runtime when it is not explicitly invoked:

- no daemon or running loop;
- no watcher, timer, or polling;
- no persistent listener or bus connection;
- no periodic diagnostics;
- no spontaneous activation.

When invoked, it necessarily consumes bounded resources and then exits. The promise is absence of residency and unsolicited work, not zero physical impact while executing.

A dormant executable cannot receive unsolicited bus commands. qxctl must reach and invoke it through the explicitly enabled transport. A selected bus or qxctl relay may carry its bounded egress, but a permanent listener would be a separate optional component.

## State Vocabulary

The local durable vocabulary is:

- `current`;
- `previous`;
- `lastTransmitted`.

Its bounded crash-safe state may also identify a local generation or equivalent causal marker, predecessor and current digests, Node/TOPS and incarnation identity, Prima Parte version, associated qxctl operation identity, boot identity, exact OS/Habitat/Nest identities and digests, bounded diagnostic state or digest, Node observation time, transmission time, and pending transmission condition.

Node time is evidence, not sole ordering authority. Generation and digest lineage establish causal continuity; Maestro may independently record receipt time.

`lastTransmitted` means the state record was sent by the remote Node after execution of the associated command. It does not mean Maestro or another receiver confirmed receipt.

## Bounded Transaction

1. qxctl or its enabled remote transport performs or coordinates a material operation.
2. qxctl explicitly invokes Prima Parte.
3. Prima Parte observes its complete bounded local state and checks for a durably pending transmission.
4. It compares the observation with durable `current` and `previous` state.
5. If unchanged and no transmission is pending, it returns without transmission.
6. If changed, it crash-safely commits the next local generation and pending state before transmission.
7. If a transmission is pending, it sends one bounded change observation through the selected adapter or relay.
8. It updates `lastTransmitted` according to send execution and exits.

If transmission fails, the locally committed transition remains pending and identifiable for retry on the next explicit invocation even when the next observation otherwise matches `current`. There is no background retry loop. Exact ordering when a newly observed transition and an older pending transmission coexist remains a Phase 3 contract question; it must preserve bounded causal evidence rather than discard either transition.

## Maestro Relationship

Maestro may record the last observation it received from each enabled Prima Parte. That record is not an assertion of continuously verified present truth. Prima Parte records only its own Node and never aggregates the system.

## Bootstrap

Hardware with no executable environment cannot run Prima Parte. The selected provisioning route establishes an executable Habitat first, then installs or stages the exact Prima Parte executable. Reimage continuity is not fabricated; a replacement or reset is represented according to SNV identity and incarnation truth.

## Non-Scope

Prima Parte does not supervise, schedule, mutate, interpret strategy logic, decide compatibility, define Habitat/Nest semantics, archive evidence, collect bulk telemetry, or act as a bus gateway. Any future mutation authority requires a separate contract and ratification.
