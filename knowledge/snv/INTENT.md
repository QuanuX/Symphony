# Symphony Node Vector Intent

## Purpose

The Symphony Node Vector (SNV) records and relates the identities, resources, cluster memberships, and user-assigned names of physical Nodes used by Symphony systems.

## Scope

SNV is composed of:

- SNIV for physical Node identity and its exact provider-resource, selected-offering, and infrastructure-domain associations;
- SNRV for local and explicitly qualified remote resource truth;
- SCIV for cluster identity, bus connectivity, and Node-in-cluster relationships;
- SCNV for SNV-bounded consolidation and resolution of names already assigned by users or providers.

## Recording Boundary

SNV records facts and their lineage. It does not choose, issue, dictate, reserve on a user's behalf, or automatically rename a Node, cluster, provider resource, TOPS, or TROG.

## Relationships

- SCV supplies provider/offsite resource knowledge.
- SOV establishes and changes infrastructure through qxctl.
- SHV may supply detailed hardware capability evidence.
- SKV governs SNV's contracts, indexing, change, evolution, and publication relationships.
