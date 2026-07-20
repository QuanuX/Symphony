# Symphony Knowledge Vector Index Manifest

## Canonical Target
`knowledge/skvi/MANIFEST.md`

## Purpose
To declare the identity and scope of the Symphony Knowledge Vector Index (SKVI) contract surface.

## SKVI Identity
SKVI maps the locations, scopes, descriptors, ownership boundaries, and relationships of all canonical knowledge-vector files.
SKVI makes the knowledge layer discoverable without requiring agents or humans to infer structure.
SKVI is autonomous and discoverable as a peer vector surface.
SKVI is not hidden inside SKV.
SKVI is not a generated database yet.
SKVI is not a generated index yet.

## Classification
- knowledge-vector contract surface
- declarative index contract
- not generated index
- not generated database

## Declared Contract Truth Role
`MANIFEST.md` establishes the definitive boundaries of the SKVI surface. Implementations may check or project SKVI only within separately ratified contracts and remain subordinate to this declared knowledge truth.

## Installability Considerations
SKVI has no executable install surface. The checked-in validator reads SKVI directly from a repository checkout; qxctl integration, generated indexes, schemas, templates, and publication pipelines remain deferred.

## Scope
SKVI encompasses the mapping of canonical knowledge files and contract descriptors across the Symphony repository.

## Non-Scope
SKVI is not a generated database yet.
SKVI is not a generated index yet.
SKVI is not a search engine.
SKVI is not NotebookLM.
SKVI is not Mintlify.
SKVI is not a docs site.
SKVI is not qxctl.
SKVI is not symphony-validator.
SKVI does not replace module contracts.
SKVI does not replace SCLV.
SKVI does not replace SODV.
SKVI does not replace SSCG.
SKVI does not create runtime behavior.
SKVI does not enforce runtime behavior.

## Inputs SKVI Maps
Module contract boundaries, root governance, and other Knowledge Vector files, including SCLV and SODV.

## Outputs SKVI Describes
Repository-maintained paths, roles, ownership boundaries, relationships, consumers, status, and deferred-projection eligibility. Machine-generated projections remain deferred.

## Relationship to SKV
SKVI is an autonomous peer index defining the structural map of the SKV framework.

## Relationship to SCLV
SCLV records change truth; SKVI indexes where that change truth lives.

## Relationship to SODV
SODV governs publication truth; it uses SKVI's structural map to project to the public.

## Relationship to SSCG
SSCG interprets compatibility. SKVI defines where module boundaries and SSCG rules are documented.

## Relationship to symphony-validator
The checked-in `tools/symphony-validator/` implementation checks SKVI entry shape, required coverage, relative-path safety, path existence, uniqueness, and SCLV references. It is read-only and does not decide index membership.

## Relationship to qxctl
qxctl may later consume SKVI, but qxctl integration is not authorized here.

## Relationship to NotebookLM
NotebookLM aligns corpus context.
NotebookLM is not canonical authority.

## Relationship to Mintlify
Mintlify publishes derived official documentation.
Mintlify is not canonical authority.

## Non-Authorization Statement
This canonical surface authorizes no generated indexes, generated reports, new implementation or source files, schemas, templates, CI files, documentation publication configuration, Mintlify configuration, qxctl integration, validator capability beyond the separately bounded `tools/symphony-validator/` contract, NotebookLM automation, publication pipeline, database files, service files, runtime processes, deployment scripts, installer scripts, binary assets, or binary renames.
