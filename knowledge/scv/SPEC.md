# Symphony Cloud Vector Specification

## Status

Architect-ratified domain specification. Detailed data, graph, engine, adapter, API, and operation contracts remain deferred.

## Knowledge Model Obligations

Future SCV contracts must be able to distinguish, where relevant and evidenced:

- provider identity plus the provider-defined semantics of its resource identifiers;
- offering or product catalog identity and versioned capabilities;
- region, zone, locality, infrastructure-domain semantics, and applicable availability;
- physical, dedicated, shared, remote-attached, and virtual resource properties without collapsing them;
- account- or organization-declared provider and region constraints;
- price, balance, quota, permission, or availability observations when a private installation explicitly supplies them;
- compatibility consequences for Node, Habitat, Nest, bus, storage, training, research, and other user-selected requirements;
- hybrid relationships spanning providers, regions, and owned hardware;
- evidence source, observation time, version, lineage, uncertainty, conflict, and staleness.

This list is a design obligation, not a ratified graph schema.

SCV owns the catalog meaning of providers, offerings, regions, infrastructure domains, and provider-defined identifier systems. When a particular resource is established as a Node, SNV records that Node's exact provider-resource identifier, selected offering reference, infrastructure-domain association, and related identities. Referencing SCV knowledge from an SNV record does not transfer the instantiated Node association to SCV or the provider catalog semantics to SNV.

## Evidence and Choice

Provider marketing, APIs, account observations, user declarations, and local probes are evidence with distinct authority and freshness. An unavailable or conflicting fact remains unresolved. SCV must not make one provider's abstraction the universal vocabulary for every other provider.

SCV may expose exact differences and possibilities. The user decides whether a constraint is mandatory, whether a hybrid design is desirable, and whether a deployment should proceed.

## Account Information

Account information is not presumed durable SCV state. Where a private user permits it, purpose-built observation adapters may supply bounded current evidence. Historical account reconstruction and the anticipated separate account-truth domain remain deferred.

## Graph and Engine Boundary

The future C++ engine owns deterministic domain operations defined by SCV contracts; it does not acquire provider authority by computing a graph. The graph is a rebuildable installation-local knowledge representation and must identify the canonical and observed evidence from which it was produced.

AI access, a private API, or another query surface may consume SCV projections only after its own version, compatibility, security, and authority contract is ratified.

## SOV Relationship

SCV is operationally submissive to SOV: SOV may request or use supported SCV evidence while planning and executing an authorized operation. SOV cannot rewrite SCV semantics, manufacture provider facts, or treat a query result as permission.

## Non-Authorization Statement

No SCV record authorizes a provider operation, purchase, deployment, account access, credential release, or substitution of one resource for another.
