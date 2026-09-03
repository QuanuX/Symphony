# Symphony Cluster Identity Vector Specification

## Cluster Condition

Two or more Nodes form a cluster when they are connected to one another through one or more buses. Nodes that have no bus connection to one another are not a cluster.

Multiple bus fabrics connecting the same participating Nodes may belong to one cluster. They do not automatically create separate clusters. Distinct clusters exist when the user establishes distinct cluster identities and their applicable connected membership.

## Relationships

SCIV may represent `node-in-cluster` and a distinguishable alternate-bus relationship such as `node-in-cluster-alt-bus`. These descriptive terms do not yet allocate stable identifiers or define a machine schema.

Physical Node identity, Node incarnation, cluster identity, bus identity, connection observation, and user-assigned names remain separate facts.

## Disconnection

Disconnection does not destroy or re-identify the physical Node. Exact rules for an interrupted connection, retained cluster membership, removal, and later reconnection remain deferred to the SCIV record lifecycle rather than inferred here.

## Non-Authorization Statement

SCIV cannot configure transport, declare a desired link present, route messages, or require any hot-path message to traverse a cluster bus.
