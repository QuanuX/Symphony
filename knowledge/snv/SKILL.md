# Symphony Node Vector Skill

## Purpose

Guide agents through Node identity, resource, cluster, and naming questions without allowing one subvector to absorb another.

## Reading Order

1. `knowledge/ARCHITECTURE.md`
2. `knowledge/SLANG.md`
3. `knowledge/NAMESPACES.md`
4. the SNV Contract Quad
5. the relevant SNIV, SNRV, SCIV, and SCNV Contract Quads
6. applicable SCV, SOV, SHV, SKV lifecycle, SSIAG, STAV, and SODV contracts

## Procedure

1. Determine whether the subject is physical identity, resource inventory, incarnation, cluster relationship, or name resolution.
2. Preserve provider-native facts separately from Symphony and user-assigned facts.
3. Distinguish local physical changes from software changes and remote-resource attachments.
4. Require evidenced bus connectivity before calling multiple Nodes a cluster.
5. Resolve names only within their declared scope and time.
6. Surface ambiguity or absent identity instead of manufacturing a record.

## Stop Conditions

Stop for Architect review before defining an identifier encoding, `::` grammar, name-allocation service, discovery mechanism, incarnation transition, cardinality, graph database, engine operation, or qxctl command.
