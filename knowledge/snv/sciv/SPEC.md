# Symphony Cluster Identity Vector Specification

## Cluster Condition

Two or more Nodes form a cluster when they are connected to one another through one or more buses. Nodes that have no bus connection to one another are not a cluster.

Multiple bus fabrics connecting the same participating Nodes may belong to one cluster. They do not automatically create separate clusters. Distinct clusters exist when the user establishes distinct cluster identities and their applicable connected membership.

## Relationships

SCIV may represent `node-in-cluster` and a distinguishable alternate-bus relationship such as `node-in-cluster-alt-bus`. These descriptive terms do not yet allocate stable identifiers or define a machine schema.

Physical Node identity, Node incarnation, cluster identity, bus identity, connection observation, and user-assigned names remain separate facts.

## Disconnection

Disconnection does not destroy or re-identify the physical Node. Temporary connection loss changes observations and preserves its supplied participation incarnation. Explicit end/rejoin belongs to SNIV's episode lifecycle. SCIV retains attributed membership/relationship history; removal requires explicit evidence and is never inferred from missing connectivity.

## Bounded Native Contract

`modules/sciv-engine` 0.1.0-dev supplies independently installable C++26 validation and typed candidate transitions. The strict v1 wire contract has explicit system, cluster, Node, incarnation, bus, source, time and profile fields. Unknown/conflicting evidence remains visible. Two or more joined, identified Nodes and supported fresh links are required for a positive supplied-cluster finding.

The first profile explicitly selects directedness, transitivity and maximum age. Buses are evaluated independently; unspecified cross-fabric bridges are not inferred. Multiple fabrics may support the same cluster. Negative findings require complete fresh pair evidence within the supplied population. These exact module rules allow separately versioned future profiles.

See `modules/sciv-engine/SPEC.md` and installed schemas/templates for deterministic correspondence, provenance, limits and transitions. Parent retention and protected selection remain separate operations.

## Non-Authorization Statement

SCIV cannot configure transport, declare a desired link present, route messages, or require any hot-path message to traverse a cluster bus.

All numeric times and limits use the exact nonnegative interoperable JSON integer range `0..9007199254740991`. The directly linked SDK enforces the same range as the bounded process and packaged schemas. This range does not require floating-point rounding or a wider implicit time representation.
