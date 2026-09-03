# Symphony Consolidated Naming Vector Specification

## Recording Rule

Every name remains a distinguishable field with its source, subject, scope, and applicable lifecycle. Provider-resource identifiers, offering names, infrastructure-domain names, Symphony Node identities, cluster names, user nicknames, and short names are not collapsed merely because they resolve to the same Node.

All applicable names may be used to identify the same subject when the requested scope and time make the result unambiguous.

## Scope Rules

- A name declared directly in a TOPS or TROG follows that declared scope.
- Each cluster is identified before cluster-local Node short names are resolved.
- A short name such as `annex` may be used in two different clusters.
- Two active Nodes in the same cluster may not both resolve from the same unqualified short name.
- A retired name may be reused when the future lifecycle contract makes the prior retirement and new subject unambiguous.

These rules record user naming. They do not require a user to adopt one naming style.

## Universal Namespace Boundary

SCNV consumes applicable namespace-family and collision doctrine from `knowledge/NAMESPACES.md` but does not own that universal doctrine. It cannot allocate feature, command, operation, protocol, vector, API, provider, broker, or package namespaces.

## Non-Authorization Statement

Name resolution is not physical-identity proof, cluster-connectivity proof, provider authority, or permission to operate upon the resolved subject.
