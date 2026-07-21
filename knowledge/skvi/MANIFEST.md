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
SKVI canonical truth is the repository-maintained Markdown index; executable outputs remain derived.

## Classification
- knowledge-vector contract surface
- declarative index contract
- source truth for index membership and relationships
- independently implementable through a subordinate proposal/projection engine

## Declared Contract Truth Role
`MANIFEST.md` establishes the definitive boundaries of the SKVI surface. Implementations may check or project SKVI only within separately ratified contracts and remain subordinate to this declared knowledge truth.

## Installability Considerations
The SKVI engine is an independently installable C++ module at `modules/skvi-engine/` with executable `symphony-skvi`. Its initial contract is inspect, check, propose, and project only through the bounded `knowledge/SPEC.md` process protocol. It may install without Maestro as `installed_undocked`. The checked-in validator remains an independent read-only checker.

## Scope
SKVI encompasses the mapping of canonical knowledge files and contract descriptors across the Symphony repository.

## Non-Scope
SKVI is not a generated database.
SKVI projections are not canonical indexes.
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
SKVI does not execute application workloads or enforce runtime behavior.

## Inputs SKVI Maps
Module contract boundaries, root governance, and other Knowledge Vector files, including SCLV and SODV.

## Outputs SKVI Describes
Repository-maintained paths, roles, ownership boundaries, relationships, consumers, status, and projection eligibility. Authorized machine projections are disposable, digest-bound, and rebuildable.

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
qxctl may invoke implemented SKVI inspect, check, propose, and project operations. qxctl is presentation and administration, never membership authority.

## Relationship to NotebookLM
NotebookLM aligns corpus context.
NotebookLM is not canonical authority.

## Relationship to Mintlify
Mintlify publishes derived official documentation.
Mintlify is not canonical authority.

## Non-Authorization Statement
This manifest authorizes the bounded proposal/projection engine and qxctl surface described above after contract merge. It does not authorize canonical mutation, self-ratification, autonomous membership decisions, public publication, NotebookLM automation, or a projection that competes with `INDEX.md`.
