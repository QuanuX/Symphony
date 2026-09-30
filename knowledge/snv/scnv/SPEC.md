# Symphony Consolidated Naming Vector Specification

## Recording Rule

Every name remains a distinguishable field with its source, subject, scope, and applicable lifecycle. Provider-resource identifiers, offering names, infrastructure-domain names, Symphony Node identities, cluster names, user nicknames, and short names are not collapsed merely because they resolve to the same Node.

All applicable names may be used to identify the same subject when the requested scope and time make the result unambiguous. The bounded v1 implementation compares exact UTF-8 bytes with no normalization or case folding. Name, kind, source namespace, typed subject, scope and effective/observed/recorded times remain explicit. See `modules/scnv-engine/SPEC.md` for exact fields, bounds, immutable revision semantics and reusable SDK/process contracts.

## Scope Rules

- A name declared directly in a TOPS or TROG follows that declared scope.
- Each cluster is identified before cluster-local Node short names are resolved.
- A short name such as `annex` may be used in two different clusters.
- Two active Nodes in the same cluster may not both resolve from the same unqualified short name.
- A retired name may be reused through an explicit new association referencing the terminal retired origin, with a nonoverlapping effective interval and supplied subject. Restoration additionally preserves the prior subject and retains any retired gap. Original records remain available; correction and retirement never erase their lineage.

Exact resolution reports unique, absent, ambiguous, insufficient or unsupported scope. Multiple sources naming one subject remain distinguishable associations. Different active subjects matching one unqualified scoped name cause a collision finding; qualification may resolve a specific exact query, and no first-match policy resolves ambiguity. Partial evidence cannot establish uniqueness or absence beyond the supplied coverage. Scoped candidate pages require the original immutable evidence digest to continue.

Unique resolution binds the exact subject, owner version, matching record IDs, scope, effective time and evidence digest. Later alias reassignment cannot redirect an already bound target. A digest establishes representation agreement and conveys no physical-identity proof or operating permission.

These rules record user naming. They do not require a user to adopt one naming style.

## Universal Namespace Boundary

SCNV consumes applicable namespace-family and collision doctrine from `knowledge/NAMESPACES.md` but does not own that universal doctrine. It cannot allocate feature, command, operation, protocol, vector, API, provider, broker, or package namespaces.

## Non-Authorization Statement

Name resolution is not physical-identity proof, cluster-connectivity proof, provider authority, or permission to operate upon the resolved subject.
