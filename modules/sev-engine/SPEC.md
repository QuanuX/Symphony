# SEV Engine Specification

The C++26 process implements every v1 operation in `knowledge/sev/SPEC.md` over exact bounded JSON. Its exact descriptor-v2 document binds stable operation IDs and SSFV feature IDs, publishes the shared process limits and `user`/`tops` scopes, and carries an omit-self SHA-256 descriptor digest. SCSEV profile identity, receipt-owned installation state, and Maestro bindings are deliberately absent from descriptor v2. Case, action, impact, plan, verification, SCSEV, graph, recovery, closure, and compatibility outputs are content addressed.

Disposition planning validates all action self-digests, rejects absent or duplicate identities, rejects dependency cycles, and computes a deterministic ready set. A blocker removes its action and transitive dependents while leaving independent zero-indegree actions ready. Transition verification evaluates only the SAV closed rule algebra against a complete caller-supplied successor CURRENT snapshot.

SCSEV requires the fourteen canonical consequence families in contract order. It reports missing or prohibited consequences and never invents a final command identity, grammar, exemption, or patch.

Successor recalculation derives readiness from the complete supplied plan after every completed or failed action and propagates blockers only through dependents. Watch and novelty checks enforce freezing-path, offline/private, disclosure, redaction, and prohibited-secret boundaries. Session binding ties an exact case and source CURRENT to the established lifecycle profile/report-journal stream without persistence or apply authority.

Novelty, watch, trigger-coalescing, and lifecycle-binding operations each publish a dedicated Draft 2020-12 input envelope, and every newly hardened operation publishes an exact result contract. Descriptor v2 and qxctl expose those same protocols, preserving deterministic fail-closed behavior across direct IPC, side-by-side versions, and interrupted upgrade order.

Capacity review begins at 512 KiB request, 2 MiB response, 8,192 parsed JSON events, 512 affected/action/blocker/finding entries, 2,048 graph nodes, 512 inverse edge pairs, 512 watch events, or 512 novelty items. These thresholds never relax the shared hard limits or the smaller operation-local bounds.

The compatibility matrix is exact and symmetric about supported overlap:

| Reader evidence | Writer | Result |
|---|---|---|
| `v1` | `v1` | compatible: `exact_v1_overlap` |
| `v0` only | `v1` | incompatible: `no_supported_overlap`, with unknown critical state preserved |
| `v1` | `v2` | incompatible: `no_supported_overlap` until v2 is ratified |
| side-by-side exact versions during interruption | explicitly selected version | the matching v1 process remains usable; lifecycle binding and recovery never auto-fallback and remain qxctl/coordinator responsibilities |

All mutation, persistence, host discovery, authorization, audit submission, coordinator state changes, network export, and hot/warm execution are external and unimplemented by this process.
