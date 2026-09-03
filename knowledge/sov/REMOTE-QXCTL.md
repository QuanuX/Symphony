# Remote qxctl Architecture Research Contract

## Status

Phase 2 research obligation. This surface preserves supported possibilities and required questions without selecting an exact transport or controller/target work split. The Architect-ratified preference for an applicable IPC family, with scoped remote CLI or shell-mediated fallback, remains a research constraint rather than a selected implementation.

## Supported Arrangements

The architecture must be capable of supporting:

1. controller-side qxctl coordinating the remote operation;
2. qxctl installed on an enabled target Node;
3. qxctl staged for one bounded target-side operation.

The cluster creator chooses the arrangement for the applicable Node or operation. A Symphony-wide default may be offered but cannot erase explicit per-Node configuration.

## Research Questions

Each arrangement must define:

- how qxctl or the bounded operation reaches the target;
- bootstrap before an executable Habitat and command transport afterward;
- enablement and disablement;
- authentication and authorization;
- exact command scoping and argument bounds;
- controller/target version negotiation;
- expected state, idempotency, timeout, interruption, and retry;
- result evidence, safe audit projection, and recovery;
- secret exclusion and provider credential boundaries;
- interaction with Habitat conditioning, Nest delivery, bus-adapter setup, Prima Parte, Maestro, SNV, SCV, and SODV;
- provider, hardware, transport, and bus neutrality;
- absence of unrequested hot/warm residency.

## Transport Posture

IPC is preferred where it can satisfy the selected remote topology. A scoped remote CLI or shell-mediated invocation is an available fallback. A general remote shell, permanent qxctl service, hidden listener, agent daemon, or universal bus consumer is not authorized by this research contract.

## Output

Research produces evidence, compatibility findings, and candidate contracts. It does not make a transport operational or grant remote execution authority.
