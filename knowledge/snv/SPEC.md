# Symphony Node Vector Specification

## Status

Architect-ratified composition and ownership boundary. Exact record schemas and executable behavior remain deferred.

## Composition

SNIV, SNRV, SCIV, and SCNV remain separately defined semantic owners beneath SNV. SNV may project their relationships but must not collapse physical identity, resources, cluster membership, or names into one interchangeable field.

## Physical Node Continuity

A Node is a physical resource. Software reflash, operating-system replacement, ordinary restart, disconnection, and reconnection do not alone create a new physical Node. Destruction, physical replacement, or a material change to local hardware creates a new physical-resource identity, including when an external provider identifier remains unchanged.

A remote or virtual resource attached to a Node is not silently incorporated into its local physical identity. SNRV distinguishes that resource and its attachment.

## Node Incarnation

A Node incarnation begins when a physical Node is attached to a bus and established as a participant in a trading system, research cluster, or another Symphony-built system. Incarnation, physical identity, boot identity, Habitat identity, and cluster membership are distinct facts. Their exact transition and cardinality rules require their future record contracts.

## Cluster Condition

Multiple Nodes form a cluster only when connected to one another through one or more buses. Multiple buses do not create multiple clusters unless the user establishes distinct cluster identities. A Node-in-cluster relationship may distinguish an alternate bus without re-identifying the physical Node.

## Naming Condition

SNV records provider names, resource identifiers, infrastructure-domain names, Symphony Node identities, Node incarnations, cluster names, user nicknames, short names, and other ratified aliases as distinguishable facts. SCNV makes applicable names resolvable to the same SNV subject within their declared scopes.

No name is proof of physical identity by itself. If a proposed resource has no identity in any applicable owner domain, it remains a theory or unresolved candidate rather than a Node record.

## Scope and Reuse

Names registered directly within a TOPS or TROG follow that declared scope. Where multiple named clusters exist, the same Node short name may identify different Nodes in different clusters, while two active Nodes in the same cluster require unambiguous identifiers. Reuse after retirement is permitted under the exact future naming record contract.

## Non-Authorization Statement

SNV records and resolves supplied or observed facts. It does not create hardware, connect buses, declare a strategy deployment valid, issue a name, or grant operational authority.
