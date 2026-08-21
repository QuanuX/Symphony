# Accordare STAV Producer Threat Model

## Protected Assets

The protected assets are the meaning of the closed Accordare vocabulary, the authenticated actor binding, exact before/after digests, durable pending evidence, STAV grant configuration, and the uniqueness of the STAV ledger authority.

## Defenses

- kernel-supplied peer UID/GID on both Unix sockets;
- exact configured subject mapping and exact SSIAG decision/capability recomputation;
- duplicate-key rejection, bounded frames/files, strict field sets, and result self-digest validation;
- mechanical subject-kind normalization rather than caller-class policy;
- producer-derived event tuple, target kind, outcome, reasons, and stable candidate IDs;
- private no-symlink configuration/outbox/socket locks;
- fsync, atomic rename, directory sync, request collision refusal, and idempotent replay;
- exact STAV grant permissions, stopped-authority mutation, expected-state CAS, and recovery markers.

## Residual Risk

A host administrator can replace processes or configuration because host authority is the platform gate. STAV v1 is tamper-evident rather than non-repudiable. A process crash after coordinator persistence but before qxctl reaches the producer is not yet closed by a pre-mutation intent protocol; the implemented circuit guarantees durability from producer acceptance onward. Failed and unavailable Named Version outcome events remain reserved but unimplemented until typed upstream failures can be proven without unsafe raw errors.
