# SAV Engine Specification

## Protocol implementation

The executable implements the operations and v1 contracts named in `knowledge/sav/SPEC.md`. Its exact descriptor-v2 document binds every canonical `engop:` identity to its feature, protocol, mutability, idempotency, recovery, invocation, authorization, and thermal metadata; publishes the shared process limits and `user`/`tops` scopes; and carries an omit-self SHA-256 descriptor digest. Receipt-owned installation state and Maestro bindings are deliberately absent from descriptor v2.

Inputs use exact field sets, sorted unique semantic identities, fixed collection bounds, tagged SHA-256 digests, and STSC whole-second UTC. Source `content_digest` binds compact recursively ordered JSON payload bytes. Accord Reference `record_digest` binds the record with that field omitted.

CURRENT is derived only from supplied source projections. Its stable digest excludes collection timestamps; its observation digest includes them. Evaluation executes only the closed SAV v1 rule algebra and returns independent resolution, accord, and readiness axes. Missing evidence is never interpreted as success or contradiction.

Named Version validation enforces the `savver:` namespace, immutable self-digest, lineage, exact requirements, and freezing-only posture. Extension Capsule validation enforces `savcapsule:` identity and reports semantic, qxctl, operation, and receptor gaps without inventing missing identities. Installation Blueprint planning enforces `savblueprint:` identity, exact inverse graphs, acyclicity, component/receptor/capsule references, localized blocker propagation, and deterministic forward or reverse readiness.

Every extended operation has a dedicated Draft 2020-12 input wrapper and result schema. Engine descriptor v2 and qxctl must advertise those operation envelopes—not the nested artifact alone—so old readers, new writers, direct-process clients, and interrupted upgrade sequences fail closed at the same explicit protocol boundary.

Capacity review begins at 512 KiB request, 2 MiB response, 8,192 parsed JSON events, 128 source/component entries, 512 reference/finding or repeated-identity entries, 2,048 graph nodes, or 512 inverse edge pairs. These thresholds never relax the shared hard limits or the smaller operation-local bounds.

The compatibility matrix is exact and symmetric about supported overlap:

| Reader evidence | Writer | Result |
|---|---|---|
| `v1` | `v1` | compatible: `exact_v1_overlap` |
| `v0` only | `v1` | incompatible: `no_supported_overlap` |
| `v1` | `v2` | incompatible: `no_supported_overlap` until v2 is ratified |
| side-by-side exact versions during interruption | explicitly selected version | the matching v1 process remains usable; selection and recovery never auto-fallback and remain qxctl/coordinator responsibilities |

## Failure and non-authorizations

Malformed or inconsistent inputs fail with a structured process error. Unsupported operations return `operation.unsupported`. Apply, ambient discovery, arbitrary expression execution, filesystem mutation, network listening, authority decisions, and persistence are not implemented.
