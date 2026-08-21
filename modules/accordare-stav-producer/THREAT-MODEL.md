# Accordare STAV Producer Threat Model

## Protected Assets

The protected assets are the meaning of the closed Accordare vocabulary, the authenticated actor binding, exact before/after digests, durable pending evidence, STAV grant configuration, and the uniqueness of the STAV ledger authority.

## Defenses

- kernel-supplied peer UID/GID on both Unix sockets;
- exact configured subject mapping and exact SSIAG decision/capability recomputation;
- duplicate-key rejection, bounded frames/files, strict field sets, and result self-digest validation;
- mechanical subject-kind normalization rather than caller-class policy;
- producer-derived event tuple, target kind, outcome, reasons, and stable candidate IDs;
- private no-symlink configuration/intent/outbox/socket locks;
- fsync, atomic rename, directory sync, request collision refusal, and idempotent replay;
- exact STAV grant permissions, stopped-authority mutation, expected-state CAS, and recovery markers.

## Residual Risk

A host administrator can replace processes or configuration because host authority is the platform gate. STAV v1 is tamper-evident rather than non-repudiable. The two-phase intent closes evidence loss across qxctl/producer interruption but cannot infer whether an interrupted coordinator mutated; exact command retry is therefore required. Typed failed/unavailable events are implemented, but ambiguous process errors remain pending rather than being misclassified. Native supervision owns liveness only and cannot create identity, permission, or a STAV grant.
