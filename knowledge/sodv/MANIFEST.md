# Symphony Official Documentation Vector Manifest

## Canonical Target
`knowledge/sodv/MANIFEST.md`

## Purpose
To declare the identity, classification, and bounds of the Symphony Official Documentation Vector (SODV) as the authoritative canonical publication-governance surface.

## SODV Identity
SODV is the Symphony Official Documentation Vector.

## Classification
- canonical knowledge-vector publication-governance surface
- declarative publication-governance contract
- not public documentation
- not docs site
- not Mintlify configuration
- not publication pipeline

## Declared Contract Truth Role
SODV governs official documentation publication, including how internal canonical knowledge becomes public-facing documentation. SODV supports future Mintlify publication without making Mintlify the source of truth. SODV is not the public documentation itself. SODV does not authorize a documentation publication pipeline. SODV does not authorize Mintlify configuration.

`RELEASES.md` is the append-only module-publication ledger. It separates merged authorization from completed publication so a local package cache or planned checksum can never be mistaken for a public release.

## Installability Considerations
SODV has no executable install surface. Its module-release protocol is operational as a permission-backed, append-only repository transaction, while public-documentation generation, Mintlify configuration, qxctl integration, and a general publication pipeline remain deferred.

## Scope
SODV governs publication truth. It manages the boundaries between internal declarative truth and external public presentation or distribution.

## Module Release Invariants

- authorization binds module path and version to an immutable source commit before tag publication;
- completion is a separate forward-only record after public, clean-cache resolution;
- canonical release records never use a mutable pending state;
- existing tags are never moved or replaced;
- local module caches and temporary proxies are preparation evidence, not publication evidence;
- an interrupted transaction is reconciled against actual Git and proxy state and then completed forward or failed closed.

## Non-Scope
SODV is not public documentation. SODV is not a docs site. SODV is not Mintlify. SODV is not NotebookLM. SODV is not a publication pipeline. SODV is not a generated documentation system yet. SODV is not a generated index yet. SODV is not a documentation template system yet. SODV is not a schema system. SODV is not qxctl. SODV is not symphony-validator. SODV is not SKVI. SODV is not SCLV. SODV is not SSCG. SODV does not replace canonical repository knowledge files. SODV does not replace module contracts. SODV does not replace tool contracts. SODV does not replace PR review. SODV does not create runtime behavior. SODV does not enforce runtime behavior.

## SODV Governance Scope Summary
Layer 0 canonical publication sources, Layer 1 publication relationships, Layer 2 publication evidence. Layer 3 future publication projections are strictly deferred.

## Relationship to SKV
SODV is the publication-governance sub-vector of the broader SKV framework.

## Relationship to SKVI
SKVI indexes source truth. SODV governs publication truth.

## Relationship to SACV
SACV governs canonical API-contract policy and registration. SODV governs publication eligibility and derived vendor configuration. SODV MUST NOT rewrite an owner OpenAPI document or treat a Mintlify bundle as source truth.

## Relationship to SCLV
SCLV records change truth. SODV governs publication truth.

## Relationship to SSCG
SSCG interprets compatibility. SODV governs the publication of those compatibility constraints.

## Relationship to Canonical Repository Knowledge Files
Canonical repository knowledge files are source truth. SODV dictates how they become public.

## Relationship to Published Documentation
Published documentation is a derived public projection.

## Relationship to Mintlify
Mintlify is a publication surface, not canonical authority.

## Relationship to NotebookLM
NotebookLM is a corpus alignment and context tool, not canonical authority.

## Relationship to symphony-validator
The checked-in validator checks required SODV contract anchors and SKVI-indexed path presence. It does not currently validate `RELEASES.md` transaction semantics, remote tag state, public-proxy propagation, or checksums.

## Relationship to qxctl
qxctl may later consume SODV, but qxctl integration is not authorized here.

## Relationship to Git History
Git history is version-control evidence.

## Relationship to PR History
PR history is review and merge evidence.

## Non-Authorization Statement
This canonical surface authorizes no public documentation files, docs directory, `mint.json`, Mintlify configuration, documentation publication configuration, generated documentation, generated documentation indexes, generated changelogs, generated indexes, generated reports, new implementation or source files, schemas, templates, CI files, qxctl integration, validator capability beyond the separately bounded `tools/symphony-validator/` contract, NotebookLM automation, general publication pipeline, database files, service files, runtime processes, deployment scripts, installer scripts, binary assets, or binary renames.
