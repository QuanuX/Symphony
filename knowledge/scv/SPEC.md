# Symphony Cloud Vector Specification

## Status

Architect-ratified domain specification. The bounded source-knowledge process increment is assigned to `knowledge/scv/SOURCE-KNOWLEDGE.md` and its v1 schemas. A network API, full provider corpus, general graph database integration and operational provider adapters remain deferred.

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

This list remains a broader design obligation; the admitted source-knowledge schema realizes only its explicitly declared bounded subset.

SCV owns the catalog meaning of providers, offerings, regions, infrastructure domains, and provider-defined identifier systems. When a particular resource is established as a Node, SNV records that Node's exact provider-resource identifier, selected offering reference, infrastructure-domain association, and related identities. Referencing SCV knowledge from an SNV record does not transfer the instantiated Node association to SCV or the provider catalog semantics to SNV.

## Evidence and Choice

Provider marketing, APIs, account observations, user declarations, and local probes are evidence with distinct authority and freshness. An unavailable or conflicting fact remains unresolved. SCV must not make one provider's abstraction the universal vocabulary for every other provider.

SCV may expose exact differences and possibilities. The user decides whether a constraint is mandatory, whether a hybrid design is desirable, and whether a deployment should proceed.

## Account Information

Account information is not presumed durable SCV state. Where a private user permits it, purpose-built observation adapters may supply bounded current evidence. Historical account reconstruction and the anticipated separate account-truth domain remain deferred.

## Graph and Engine Boundary

The independently installed C++ engines own deterministic domain operations defined by SCV contracts; it does not acquire provider authority by computing a graph. The graph is a rebuildable installation-local knowledge representation and must identify the canonical and observed evidence from which it was produced.

AI access, a private API, or another query surface may consume SCV projections only after its own version, compatibility, security, and authority contract is ratified.

## SOV Relationship

SCV is operationally submissive to SOV: SOV may request or use supported SCV evidence while planning and executing an authorized operation. SOV cannot rewrite SCV semantics, manufacture provider facts, or treat a query result as permission.

## Non-Authorization Statement

No SCV record authorizes a provider operation, purchase, deployment, account access, credential release, or substitution of one resource for another.

## Family Delegation

SCV delegates cloud-hosting/hyperscaler knowledge to `knowledge/scv/schv/SPEC.md` and edge knowledge to `knowledge/scv/scev/SPEC.md`. SCHV delegates native AWS, Azure, DigitalOcean and Google Cloud knowledge to its aws, azure, do and gcp Contract Quads; SCEV delegates Cloudflare knowledge to its cf Contract Quad. These are domain scopes, not rankings or restrictions on cross-provider or user-defined composition.

Agent-first profile preparation, exact installed schemas and qxctl retained-run coordination are specified in `knowledge/scv/AGENT-WORKFLOWS.md`. That companion adds no provider, deployment or selected-head authority.

The `0.5.0-dev` coverage operation in `knowledge/scv/COVERAGE.md` makes selected source and interpretation gaps explicit through qxctl. Coverage is scoped accounting, not complete vendor expertise or runtime compatibility.

The additive `0.6.0-dev` contract is specified by `knowledge/scv/OWNER-INTERFACE.md`, `knowledge/scv/PROVIDER-PACKS.md` and `knowledge/scv/COMPOSITION.md`. A versioned owner declaration produces mechanical interfaces while independent consumers check meaning. Portable authored provider packages retain detached semantic expectations and exact source mappings. Bounded exploration combines caller-permitted recipes against caller requirements, reports evidence and implementation obligations, and exposes changed-input reassessment. The user retains provider, policy, requirement, recipe and execution decisions. Structured JSON interpretation is an exact authored pointer slice, not general comprehension of vendor architectures or referenced schemas.
