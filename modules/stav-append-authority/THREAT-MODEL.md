# STAV Append Authority Threat Model

## Protected Assets

- per-TOPS event ordering, canonical bytes, digest chain, and durable receipts;
- authority to assign trusted identity/order/integrity fields;
- producer and reader authorization;
- endpoint authenticity and executable/state separation;
- recovery evidence.

## Principal Threats and Controls

- **Socket impersonation**: clients verify the connected authority UID/GID; the server verifies its own configured identity.
- **Caller spoofing**: the server uses kernel peer credentials, never request-supplied identity or process ancestry.
- **Over-broad producer access**: grants contain exact event-class/operation pairs and an assigned producer identity.
- **Unauthorized reads**: query omits events outside the reader's classification grant.
- **Multiple writers**: one process holds a non-blocking exclusive ledger lock; no direct mutation API exists.
- **Partial or reordered records**: each record has bounded framing/checksum and each event has sequence/predecessor validation.
- **False acknowledgement**: the receipt follows a successful full-frame file sync.
- **Crash ambiguity**: startup preserves an incomplete tail, truncates only after evidence sync, and rebuilds idempotency.
- **Silent repair**: complete corruption and chain mismatch prevent startup; no resynchronization or middle salvage exists.
- **Secret leakage**: schemas carry only safe references and closed metadata; SSIAG exposes no secret-bearing producer field.
- **Agent escalation**: qxctl and agents have no append or ledger-file surface.

## Residual Risk

Kernel UID/GID identity intentionally cannot distinguish processes sharing the same OS identity. Production deployments should use distinct service identities and restrictive filesystem policy. v1 is tamper-evident, not non-repudiating; privileged host compromise and signing/checkpoint threats require a later threat model.
