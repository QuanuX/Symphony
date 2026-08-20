# SAV Engine Specification

## Protocol implementation

The executable implements the operations and v1 contracts named in `knowledge/sav/SPEC.md`. Descriptor v2 binds every `engop:` identity to its feature, protocol, mutability, idempotency, recovery, invocation, authorization, and thermal metadata.

Inputs use exact field sets, sorted unique semantic identities, fixed collection bounds, tagged SHA-256 digests, and STSC whole-second UTC. Source `content_digest` binds compact recursively ordered JSON payload bytes. Accord Reference `record_digest` binds the record with that field omitted.

CURRENT is derived only from supplied source projections. Its stable digest excludes collection timestamps; its observation digest includes them. Evaluation executes only the closed SAV v1 rule algebra and returns independent resolution, accord, and readiness axes. Missing evidence is never interpreted as success or contradiction.

Named Version validation enforces the `savver:` namespace, immutable self-digest, lineage, exact requirements, and freezing-only posture. Extension Capsule validation enforces `savcapsule:` identity and reports semantic, qxctl, operation, and receptor gaps without inventing missing identities. Installation Blueprint planning enforces `savblueprint:` identity, exact inverse graphs, acyclicity, component/receptor/capsule references, localized blocker propagation, and deterministic forward or reverse readiness.

## Failure and non-authorizations

Malformed or inconsistent inputs fail with a structured process error. Unsupported operations return `operation.unsupported`. Apply, ambient discovery, arbitrary expression execution, filesystem mutation, network listening, authority decisions, and persistence are not implemented.
