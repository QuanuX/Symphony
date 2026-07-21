# Symphony Knowledge Vector Index Specification

## Canonical Target
`knowledge/skvi/SPEC.md`

## Specification Status
- canonical declarative source truth
- proposal/projection engine authorized after contract merge
- programmatic canonical apply disabled
- interpreted independently by the separately bounded validator implementation

## Purpose
To define the behavioral and structural specification for how SKVI maps the Symphony repository.

## SKVI Behavioral Model
SKVI maps files, boundaries, and relationships. The canonical Markdown surface remains declarative. The subordinate engine may inspect it, validate it, compute drift evidence, propose bounded entry changes, and build disposable projections. It does not enforce application runtime behavior or decide membership without permission-backed ratification.

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

## Layer 3 Derived Projections
The SKVI engine may build digest-bound JSON/JSONL, search, analytical, and graph projections whose inputs and engine version are recorded. A projection is disposable, rebuildable, and noncanonical. Publication remains SODV-gated.

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
The checked-in `tools/symphony-validator/` implementation parses this repository-maintained index and checks entry shape, required-surface coverage, indexed paths, uniqueness, and SCLV references. Its evidence does not create canonical membership, manufacture ratification, or authorize remediation.

## SKVI Engine Operations

The initial `symphony-skvi` operation set is:

- `inspect`: report canonical surface and engine compatibility;
- `check`: produce deterministic structural and drift evidence;
- `propose`: produce an immutable `symphony.knowledge.proposal.v1` without writing canonical files;
- `project`: build an authorized disposable projection.

`qxctl skvi ...` invokes these operations through `symphony.knowledge.engine-process.v1`. Direct invocation remains available for diagnostics. Apply is reserved and disabled under `knowledge/SPEC.md`.

## Relationship to NotebookLM
NotebookLM aligns corpus context.

## Relationship to Mintlify
Mintlify publishes derived official documentation.

## Deferred Surfaces
- Publication pipelines
- programmatic canonical apply
- autonomous semantic membership decisions

## Non-Authorization Statement
This specification authorizes the bounded SKVI proposal/projection engine, its independent installation contract, and qxctl invocation after merge. It does not authorize programmatic canonical mutation, generated ratification, NotebookLM automation, public publication, or replacement of repository-maintained `INDEX.md`.
