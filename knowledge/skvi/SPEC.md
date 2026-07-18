# Symphony Knowledge Vector Index Specification

## Canonical Target
`knowledge/skvi/SPEC.md`

## Specification Status
- declarative only
- non-executable
- not a generated index
- not a JSON schema
- not a Markdown template
- not CI configuration
- not qxctl integration
- interpreted by the separately bounded validator implementation

## Purpose
To define the behavioral and structural specification for how SKVI maps the Symphony repository.

## SKVI Behavioral Model
SKVI is a passive index surface. It does not enforce runtime behavior. It maps files, boundaries, and relationships. 

## Initial SKVI Map Scope
The scope of the structural map spans four layers.

## Layer 0 Canonical Surfaces
- `README.md`
- `INTENT.md`
- `go.work`
- `modules/node-troll/`
- `modules/bus-troll/`
- `modules/hotpath-runtime/`
- `modules/secure-identity-access-governance/`
- `modules/ssiag-provider-macos-keychain/`
- `libraries/`
- `libraries/stav-protocol-go/`
- `tools/symphony-validator/`
- `knowledge/`
- `knowledge/skvi/`
- `knowledge/sacv/`
- `knowledge/ssiag/`
- `knowledge/stav/`
- `knowledge/sclv/`
- `knowledge/sodv/`

## Layer 1 Contract Files
Where present:
- `INTENT.md`
- `MANIFEST.md`
- `INSTALL.md`
- `SKILL.md`
- `SPEC.md`
- `REQUIREMENTS.md`
- `IMPLEMENTATION.md`
- `THREAT-MODEL.md`

## Layer 2 Relationship Descriptors
Relationship descriptors among:
- root governance
- module contracts
- tool contracts
- knowledge-vector contracts
- SCLV records
- SODV publication governance
- SACV API-contract governance and registry
- validator evidence
- future qxctl consumption
- NotebookLM corpus alignment
- Mintlify publication projection

## Layer 3 Future Generated Projections
Future generated projections are not authorized by this canonical seed.
Future generated indexes are not authorized by this canonical seed.
Future JSON schemas are not authorized by this canonical seed.
Future Markdown templates are not authorized by this canonical seed.

## Source Truth Versus Projection Boundaries
Canonical repository knowledge files are source truth.
SKVI indexes source truth.
SCLV records change truth.
SODV governs publication truth.
Published documentation is a derived public projection.

MANIFEST.md is declared contract truth.
Code is implementation truth.
Generated JSON is a derived projection.
SSCG state is the compatibility interpretation.

## What Future SKVI Indexes May Claim
They may claim to accurately reflect the directory layout and inter-document descriptors (like "Module X depends on Tool Y").

## What Future SKVI Indexes Must Not Claim
They must not claim to replace Git history, module logic, or PR review.

## Relationship to SCLV
SCLV records change truth. SKVI indexes where these SCLV change records reside.

## Relationship to SODV
SODV governs publication truth based on the index mapped by SKVI.

## Relationship to SSCG
SSCG interprets compatibility across the structural contracts that SKVI indexes.

## Relationship to symphony-validator Evidence
The checked-in `tools/symphony-validator/` implementation parses this human-authored index and checks entry shape, required-surface coverage, indexed paths, uniqueness, and SCLV references. Its evidence does not create canonical membership or authorize remediation.

## Relationship to qxctl
qxctl may later consume SKVI, but qxctl integration is not authorized here.

## Relationship to NotebookLM
NotebookLM aligns corpus context.

## Relationship to Mintlify
Mintlify publishes derived official documentation.

## Deferred Surfaces
- Generated indexes
- Publication pipelines

## Non-Authorization Statement
This canonical surface authorizes no generated indexes, generated reports, new implementation or source files, schemas, templates, CI files, documentation publication configuration, Mintlify configuration, qxctl integration, validator capability beyond the separately bounded `tools/symphony-validator/` contract, NotebookLM automation, publication pipeline, database files, service files, runtime processes, deployment scripts, installer scripts, binary assets, or binary renames.
