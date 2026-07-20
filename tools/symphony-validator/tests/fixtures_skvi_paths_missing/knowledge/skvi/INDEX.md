# Symphony Knowledge Vector Index

## Status
Status:
  Canonical declarative SKVI index.

## Purpose
A repository-maintained declarative knowledge routing table.

## Scope
SKVI indexes source truth.
SKVI does not create source truth.
SKVI maps canonical knowledge surfaces.
SKVI maps what each surface means.
SKVI maps ownership boundaries.
SKVI maps truth roles.
SKVI maps relationships.
SKVI maps future derived projection eligibility.

## What This Index Is
A repository-maintained declarative knowledge routing table.

## What This Index Is Not
SKVI is not a generated database.
SKVI is not a generated index.
SKVI is not a search engine.
SKVI is not NotebookLM.
SKVI is not Mintlify.
SKVI is not qxctl.
SKVI is not a validator implementation.
SKVI is not a publication pipeline.

## Source-Truth Doctrine
Canonical repository knowledge files are source truth.
SKVI indexes source truth.
SCLV records change truth.
SODV governs publication truth.
Published documentation is a derived public projection.
MANIFEST.md is declared contract truth.
Code is implementation truth.
Generated JSON is a derived projection.
SSCG state is the compatibility interpretation.
NotebookLM is a corpus alignment and context tool, not canonical authority.
Mintlify is a publication surface, not canonical authority.

## Projection Doctrine
Canonical Markdown is source truth.
JSON / JSONL is portable derived evidence.
DuckDB is the preferred future local analytical projection store.
HDF5 is the preferred future dense quantitative / vector / compatibility substrate.
Graph views are visual relationship projections.
All projections are derived, disposable, and rebuildable.
No projection is canonical authority.

This index authorizes no generated projection.

## Graph Projection Doctrine
SKVI INDEX.md declares nodes and relationships.
A future graph projection may visualize those relationships.
The graph does not create relationships.
The graph is not canonical authority.
The graph must be rebuildable from canonical Markdown.
Obsidian-like graph behavior is an inspiration for visual navigation, not a source-truth model.

## Future Tool Boundary
Markdown declares.
C++ detects.
C++ checks.
C++ projects.
Permission holders ratify.
Authority-free tools and callers assist.

Future C++ tooling may read, check, and project SKVI entries.
Future C++ tooling must not autonomously author canonical truth.
Future C++ tooling may identify missing or stale entries as evidence.
Future C++ tooling must not decide architectural truth.
Future qxctl may query derived SKVI projections.
Future validator checks may verify SKVI structure.

## Entry Model
- **path**: (Required) The relative path to the canonical surface. Expected to be a string.
- **title**: (Required) The title of the surface. Expected to be a human-readable string.
- **surface_type**: (Required) The structural type of the surface (e.g., markdown, directory).
- **truth_role**: (Required) The authority role (e.g., governance, seed, module manifest).
- **owner**: (Required) The responsible party or entity.
- **scope**: (Required) The operational or declarative scope.
- **relationships**: (Optional) Declared relationships to other surfaces.
- **consumers**: (Optional) Known or planned consumers of this surface.
- **deferred_projections**: (Optional) Planned future projections derived from this surface.
- **status**: (Required) The current ratification status.
- **notes**: (Optional) Additional context.

## Relationship Model
- **indexes**: Points to a collection or registry this surface organizes. Acceptable sources: SKVI/SCLV structures. Acceptable targets: Any. Canonical relation. May become graph edge.
- **declares**: Points to a capability, doctrine, or state established by this surface. Acceptable sources: INTENT/MANIFEST. Acceptable targets: Any. Canonical relation. May become graph edge.
- **records**: Points to a historical or audit surface. Acceptable sources: SCLV. Acceptable targets: Audit records. Canonical relation. May become graph edge.
- **governs**: Points to a surface or module constrained by this policy. Acceptable sources: Governance/INTENT. Acceptable targets: Any. Canonical relation. May become graph edge.
- **derives_from**: Points to the canonical authority this surface projects from. Acceptable sources: Projections. Acceptable targets: Canonical markdown. Canonical relation. May become graph edge.
- **may_publish**: Points to a deferred projection or publication pipeline. Acceptable sources: SODV. Acceptable targets: Any projection. Deferred relation. May become graph edge.
- **may_check**: Points to a validator or tool that will inspect this surface. Acceptable sources: Any. Acceptable targets: Validator. Deferred relation. May become graph edge.
- **may_consume**: Points to an external tool or projection pipeline. Acceptable sources: Any. Acceptable targets: Tools/qxctl. Deferred relation. May become graph edge.
- **depends_on**: Points to a required upstream canonical surface. Acceptable sources: Any. Acceptable targets: Canonical markdown. Canonical relation. May become graph edge.
- **interprets**: Points to a capability that reads this surface for logic. Acceptable sources: Tools/Runtime. Acceptable targets: Canonical markdown. Canonical relation. May become graph edge.
- **supersedes**: Points to a deprecated or replaced surface. Acceptable sources: Any. Acceptable targets: Legacy surface. Canonical relation. May become graph edge.
- **renames**: Points to a prior name for a surface. Acceptable sources: Any. Acceptable targets: Legacy surface. Canonical relation. May become graph edge.
- **deprecates**: Points to a surface planned for removal. Acceptable sources: Any. Acceptable targets: Legacy surface. Canonical relation. May become graph edge.

## Indexed Canonical Surfaces

### Root Governance

#### README.md
- path: `README.md`
- title: Root README
- surface_type: root governance overview
- truth_role: project orientation and governance summary
- owner: Symphony root governance
- scope: Introduces repository purpose, boundaries, and top-level navigation expectations.
- relationships:
  - declares -> `INTENT.md`
  - may_consume -> future SODV public documentation projection
- consumers:
  - humans
  - reviewers
  - agentic tools
  - future qxctl
  - future validators
- deferred_projections:
  - JSON / JSONL portable evidence
  - DuckDB analytical projection
  - graph relationship projection
- status: canonical
- notes: Public-facing only after SODV-authorized publication.

#### INTENT.md
- path: `INTENT.md`
- title: Root Intent
- surface_type: root governance declaration
- truth_role: defines Symphony platform purpose and boundaries
- owner: Symphony root governance
- scope: Top-level intent and doctrine.
- relationships:
  - governs -> Symphony platform
- consumers:
  - humans
  - reviewers
  - agentic tools
  - future validators
- deferred_projections:
  - JSON / JSONL portable evidence
  - DuckDB analytical projection
  - graph relationship projection
- status: canonical
- notes: None.

### Runtime Module Contract Seeds

#### node-troll
##### INTENT.md
- path: `modules/node-troll/INTENT.md`
- title: node-troll Intent
- surface_type: module intent seed
- truth_role: intent and purpose for node-troll
- owner: node-troll maintainer
- scope: node-troll represents the node.
- relationships:
  - declares -> `modules/node-troll/MANIFEST.md`
- consumers: humans, future validators
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### MANIFEST.md
- path: `modules/node-troll/MANIFEST.md`
- title: node-troll Manifest
- surface_type: module contract truth
- truth_role: declared contract truth for node-troll installability
- owner: node-troll maintainer
- scope: Contractual dependencies, assumptions, capabilities. Python must not be required for remote native hot-path execution or the administrative spine. Optional isolated Python habitats may exist only when explicitly declared by a module or tool.
- relationships:
  - depends_on -> `modules/node-troll/INTENT.md`
- consumers: humans, future validators, future qxctl
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### INSTALL.md
- path: `modules/node-troll/INSTALL.md`
- title: node-troll Install
- surface_type: module install guidance
- truth_role: installability / deployment boundary guidance
- owner: node-troll maintainer
- scope: Instructions and constraints for deployment.
- consumers: humans, future tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SKILL.md
- path: `modules/node-troll/SKILL.md`
- title: node-troll Skill
- surface_type: module skill guidance
- truth_role: operational skill guidance
- owner: node-troll maintainer
- scope: Tools and skills for operating the node.
- consumers: humans, agentic tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

#### bus-troll
##### INTENT.md
- path: `modules/bus-troll/INTENT.md`
- title: bus-troll Intent
- surface_type: module intent seed
- truth_role: intent and purpose for bus-troll
- owner: bus-troll maintainer
- scope: bus-troll manages bus residency and bus compatibility. bus-troll is required only for deployments that use a managed bus boundary. Bus bypass remains valid when declared by deployment constraints. The existence of bus-troll does not make bus traversal mandatory.
- relationships:
  - declares -> `modules/bus-troll/MANIFEST.md`
- consumers: humans, future validators
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### MANIFEST.md
- path: `modules/bus-troll/MANIFEST.md`
- title: bus-troll Manifest
- surface_type: module contract truth
- truth_role: declared contract truth for bus-troll
- owner: bus-troll maintainer
- scope: Contractual dependencies and capability boundaries.
- consumers: humans, future validators, future qxctl
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### INSTALL.md
- path: `modules/bus-troll/INSTALL.md`
- title: bus-troll Install
- surface_type: module install guidance
- truth_role: installability / deployment boundary guidance
- owner: bus-troll maintainer
- scope: Instructions and constraints for deployment.
- consumers: humans, future tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SKILL.md
- path: `modules/bus-troll/SKILL.md`
- title: bus-troll Skill
- surface_type: module skill guidance
- truth_role: operational skill guidance
- owner: bus-troll maintainer
- scope: Tools and skills for operating the bus-troll.
- consumers: humans, agentic tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

#### hotpath-runtime
##### INTENT.md
- path: `modules/hotpath-runtime/INTENT.md`
- title: hotpath-runtime Intent
- surface_type: module intent seed
- truth_role: intent and purpose for hotpath-runtime
- owner: hotpath-runtime maintainer
- scope: hotpath-runtime owns the native hot path. hotpath-runtime is not a troll; it is the native hot-path runtime substrate.
- relationships:
  - declares -> `modules/hotpath-runtime/MANIFEST.md`
- consumers: humans, future validators
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### MANIFEST.md
- path: `modules/hotpath-runtime/MANIFEST.md`
- title: hotpath-runtime Manifest
- surface_type: module contract truth
- truth_role: declared contract truth for hotpath-runtime
- owner: hotpath-runtime maintainer
- scope: Contractual dependencies and hot-path execution boundaries.
- consumers: humans, future validators, future qxctl
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### INSTALL.md
- path: `modules/hotpath-runtime/INSTALL.md`
- title: hotpath-runtime Install
- surface_type: module install guidance
- truth_role: installability / deployment boundary guidance
- owner: hotpath-runtime maintainer
- scope: Instructions and constraints for deployment.
- consumers: humans, future tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SKILL.md
- path: `modules/hotpath-runtime/SKILL.md`
- title: hotpath-runtime Skill
- surface_type: module skill guidance
- truth_role: operational skill guidance
- owner: hotpath-runtime maintainer
- scope: Tools and skills for operating the hotpath-runtime.
- consumers: humans, agentic tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

### Validator Declarative Tool Contract Seed
##### INTENT.md
- path: `tools/symphony-validator/INTENT.md`
- title: Validator Intent
- surface_type: tool intent seed
- truth_role: intent and purpose for symphony-validator
- owner: validator maintainer
- scope: Define validator boundaries. The validator is deterministic, explainable, and non-autonomous. It produces structured evidence for every caller, CI systems, qxctl, and agentic tools. It does not perform interpretation, remediation, or architectural decision-making. Evidence model is truth. JSON is the structured evidence projection. Markdown is the caller-ingestion projection.
- relationships:
  - declares -> `tools/symphony-validator/MANIFEST.md`
  - declares -> `tools/symphony-validator/SPEC.md`
- consumers: humans, future implementations
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### MANIFEST.md
- path: `tools/symphony-validator/MANIFEST.md`
- title: Validator Manifest
- surface_type: tool contract truth
- truth_role: declared contract truth for symphony-validator
- owner: validator maintainer
- scope: Contractual definitions.
- consumers: humans, future qxctl
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### INSTALL.md
- path: `tools/symphony-validator/INSTALL.md`
- title: Validator Install
- surface_type: tool install guidance
- truth_role: installability / deployment boundary guidance
- owner: validator maintainer
- scope: Instructions and constraints for installation.
- consumers: humans
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SKILL.md
- path: `tools/symphony-validator/SKILL.md`
- title: Validator Skill
- surface_type: tool skill guidance
- truth_role: operational skill guidance
- owner: validator maintainer
- scope: Usage and operation.
- consumers: humans, agentic tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SPEC.md
- path: `tools/symphony-validator/SPEC.md`
- title: Validator Specification
- surface_type: tool specification
- truth_role: declarative specification behavior
- owner: validator maintainer
- scope: Deterministic validation rules.
- consumers: humans, future implementations
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### CMakeLists.txt
- path: `tools/symphony-validator/CMakeLists.txt`
- title: symphony-validator Build Contract
- surface_type: tool_build_contract
- truth_role: implementation_build_contract
- owner: symphony-validator
- scope: Declares the local C++26 build contract for the validator implementation.
- relationships: supports tools/symphony-validator/INSTALL.md; builds validator source surfaces
- consumers: maintainers; local build tooling; validator developers
- deferred_projections: none
- status: canonical
- notes: C++26 build contract surface; not a generated projection.

### Knowledge Vector Surfaces

#### Knowledge Root
##### INTENT.md
- path: `knowledge/INTENT.md`
- title: Knowledge Vector Intent
- surface_type: vector intent seed
- truth_role: intent and purpose for knowledge vectors
- owner: knowledge maintainer
- scope: Root definition of knowledge vector domains.
- consumers: humans, future validators
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

#### SKVI
##### INDEX.md
- path: `knowledge/skvi/INDEX.md`
- title: Symphony Knowledge Vector Index
- surface_type: skvi_index
- truth_role: canonical knowledge routing table
- owner: SKVI
- scope: Repository-maintained declarative index of canonical Symphony knowledge-vector surfaces, their truth roles, ownership boundaries, relationships, consumers, deferred projections, and status.
- relationships:
  - indexes -> canonical repository knowledge surfaces
  - declares -> knowledge routing relationships
  - depends_on -> `knowledge/skvi/SPEC.md`
  - may_consume -> `knowledge/skvi/MANIFEST.md`
  - may_consume -> `knowledge/skvi/SKILL.md`
  - may_check -> future tools/symphony-validator/SPEC.md rules
  - interprets -> SCLV cross-reference validation
  - interprets -> SODV publication governance
- consumers:
  - maintainers
  - agentic reviewers
  - NotebookLM corpus alignment
  - future deterministic validator checks
  - future qxctl-derived evidence consumers
  - future SODV publication governance
- deferred_projections:
  - JSON / JSONL portable evidence
  - DuckDB analytical projection
  - HDF5 dense quantitative / vector / compatibility substrate
  - graph view relationship projection
  - qxctl-readable evidence projection
  - readable Markdown report
- status: canonical
- notes: Added to make SKVI self-indexing explicit rather than implicit in validator behavior. The validator must consume declared SKVI truth, not invent canonical index membership.

##### INTENT.md
- path: `knowledge/skvi/INTENT.md`
- title: SKVI Intent
- surface_type: vector intent seed
- truth_role: intent and purpose for SKVI
- owner: SKVI maintainer
- scope: Define SKVI boundaries.
- consumers: humans, future validators
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### MANIFEST.md
- path: `knowledge/skvi/MANIFEST.md`
- title: SKVI Manifest
- surface_type: vector contract truth
- truth_role: declared contract truth for SKVI
- owner: SKVI maintainer
- scope: Contractual requirements.
- consumers: humans, future validators
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SKILL.md
- path: `knowledge/skvi/SKILL.md`
- title: SKVI Skill
- surface_type: vector skill guidance
- truth_role: operational skill guidance
- owner: SKVI maintainer
- scope: Usage and interaction.
- consumers: humans, agentic tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SPEC.md
- path: `knowledge/skvi/SPEC.md`
- title: SKVI Specification
- surface_type: vector specification
- truth_role: declarative specification behavior
- owner: SKVI maintainer
- scope: Formatting and structure definitions.
- consumers: humans, future implementations
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

#### SCLV
##### INTENT.md
- path: `knowledge/sclv/INTENT.md`
- title: SCLV Intent
- surface_type: vector intent seed
- truth_role: intent and purpose for SCLV
- owner: SCLV maintainer
- scope: SCLV records change truth.
- consumers: humans, future validators
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### MANIFEST.md
- path: `knowledge/sclv/MANIFEST.md`
- title: SCLV Manifest
- surface_type: vector contract truth
- truth_role: declared contract truth for SCLV
- owner: SCLV maintainer
- scope: Contractual requirements.
- consumers: humans, future validators
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SKILL.md
- path: `knowledge/sclv/SKILL.md`
- title: SCLV Skill
- surface_type: vector skill guidance
- truth_role: operational skill guidance
- owner: SCLV maintainer
- scope: Usage and interaction.
- consumers: humans, agentic tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SPEC.md
- path: `knowledge/sclv/SPEC.md`
- title: SCLV Specification
- surface_type: vector specification
- truth_role: declarative specification behavior
- owner: SCLV maintainer
- scope: Structuring change records.
- consumers: humans, future implementations
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### CHANGELOG.md
- path: `knowledge/sclv/CHANGELOG.md`
  title: `Symphony Change Log Vector Ledger`
  surface_type: `sclv_change_ledger`
  truth_role: `canonical change truth ledger`
  owner: `SCLV`
  scope: |
    Repository-maintained declarative ledger for canonical SCLV records. Records canonical change truth against SKVI-indexed surfaces and preserves evidence, relationship changes, doctrine changes, compatibility consequences, publication consequences, projection consequences, and non-authorizations.
  relationships: |
    records change truth for canonical repository surfaces
    references SKVI-indexed paths
    depends_on `knowledge/sclv/SPEC.md`
    may inform `knowledge/sodv/SPEC.md` publication governance
    may be checked by future `tools/symphony-validator/SPEC.md` rules
    may be consumed by future qxctl-derived evidence projections
    does not replace Git history
    does not replace PR review
    does not replace SSCG interpretation
  consumers:
    - `maintainers`
    - `agentic reviewers`
    - `NotebookLM corpus alignment`
    - `future deterministic validator checks`
    - `future qxctl-derived evidence consumers`
    - `future SODV publication governance`
  deferred_projections:
    - `JSON / JSONL portable evidence`
    - `DuckDB analytical projection`
    - `HDF5 dense quantitative / vector / compatibility substrate`
    - `graph view relationship projection`
    - `qxctl-readable evidence projection`
    - `readable Markdown report`
  status: `canonical`
  notes: |
    Added because knowledge/sclv/CHANGELOG.md was canonicalized after the initial SKVI declarative index. This closes expected post-bootstrap SKVI/SCLV index drift without creating generated projections or implementation.

#### SODV
##### INTENT.md
- path: `knowledge/sodv/INTENT.md`
- title: SODV Intent
- surface_type: vector intent seed
- truth_role: intent and purpose for SODV
- owner: SODV maintainer
- scope: SODV governs publication truth.
- consumers: humans, future validators
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### MANIFEST.md
- path: `knowledge/sodv/MANIFEST.md`
- title: SODV Manifest
- surface_type: vector contract truth
- truth_role: declared contract truth for SODV
- owner: SODV maintainer
- scope: Contractual requirements for publication.
- consumers: humans, future validators
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SKILL.md
- path: `knowledge/sodv/SKILL.md`
- title: SODV Skill
- surface_type: vector skill guidance
- truth_role: operational skill guidance
- owner: SODV maintainer
- scope: Usage and interaction.
- consumers: humans, agentic tools
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

##### SPEC.md
- path: `knowledge/sodv/SPEC.md`
- title: SODV Specification
- surface_type: vector specification
- truth_role: declarative specification behavior
- owner: SODV maintainer
- scope: Formatting for publication.
- consumers: humans, future implementations
- relationships: none defined
- deferred_projections: strictly deferred
- notes: none
- status: canonical

## Deferred Projections
No deferred projections, generated indexes, graphs, DuckDB, JSON, HDF5 outputs, qxctl integrations, validator implementations, or publication pipelines are canonically indexed here. They remain entirely deferred and are not canonical authority.

## Non-Authorized Artifacts
This index authorizes none of the following:
- generated index
- generated graph
- graph database
- graph visualization
- JSON / JSONL projection
- DuckDB projection
- HDF5 projection
- qxctl integration
- validator implementation
- parser implementation
- projector implementation
- schemas
- templates
- docs directory
- mint.json
- public documentation
- Mintlify configuration
- documentation publication configuration
- publication pipeline
- NotebookLM automation
- implementation files
- source files
- build files
- CI files

Note on terminology: The term `c-o-r-e` is forbidden as an active project term.

- path: does/not/exist.md
- title: Test Entry
- surface_type: test
- truth_role: test
- owner: test
- scope: test
- status: canonical
- relationships: none
- consumers: none
- deferred_projections: none
- notes: none
