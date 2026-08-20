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

## Tool Boundary
Markdown declares.
C++ detects.
C++ checks.
C++ projects.
Permission holders ratify.
Authority-free tools and callers assist.

The checked-in C++26 validator reads and checks SKVI entries and produces deterministic, read-only evidence. Separately installed C++ vector engines may produce bounded proposals and vector-authorized disposable projections under `knowledge/SPEC.md`.
C++ tooling must not autonomously author canonical truth.
C++ tooling may identify missing or stale entries as evidence.
C++ tooling must not decide architectural truth.
qxctl may invoke implemented vector-engine proposal/read operations; canonical apply remains disabled until its separate gate passes.
Future validator increments may add separately ratified deterministic checks without changing SKVI ownership.

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
- **checked_by**: Points to a validator that currently checks some declared property of this surface. Acceptable sources: Any canonical surface. Acceptable targets: Validator. Canonical relation. The target remains evidence-only and gains no authorship authority.
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
  - symphony-validator and future validator extensions
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
  - symphony-validator and future validator extensions
- deferred_projections:
  - JSON / JSONL portable evidence
  - DuckDB analytical projection
  - graph relationship projection
- status: canonical
- notes: None.

#### Root FEATURES.md
- path: `FEATURES.md`
- title: Root Symphony Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic capability truth for the repository-root platform boundary
- owner: Symphony root governance
- scope: Owns the experimental `ssfv:symphony:platform` capability record at exact source scope `.`.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`; declares -> `ssfv:symphony:platform`
- consumers: symphony-ssfv, qxctl, reviewers, agentic tools, future SODV-governed documentation projections
- deferred_projections: portable SSFV graph, module catalog, encyclopedic reference, reviewed public documentation
- notes: First partial SSFV bootstrap record; it does not claim complete repository coverage or production readiness.
- status: canonical

#### go.work
- path: `go.work`
- title: Symphony Go Workspace
- surface_type: monorepo development composition
- truth_role: Go module workspace implementation truth
- owner: Symphony root governance
- scope: Composes SSIAG, the STAV protocol kernel, the STAV append authority, and qxctl at the production Go 1.26.5 pin without creating runtime coupling.
- relationships: depends_on -> `libraries/stav-protocol-go/MANIFEST.md`; depends_on -> `modules/stav-append-authority/MANIFEST.md`; depends_on -> `tools/qxctl/MANIFEST.md`
- consumers: Go tooling, maintainers, CI, agentic tools
- deferred_projections: Go 1.27 dual-toolchain conformance evidence
- notes: Before an independent consumer release, the protocol kernel receives a real tag and the consumer records that version; the workspace is not a runtime dependency.
- status: canonical

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
- consumers: humans, symphony-validator and future validator extensions
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
- consumers: humans, symphony-validator and future validator extensions, future qxctl
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
- consumers: humans, symphony-validator and future validator extensions
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
- consumers: humans, symphony-validator and future validator extensions, future qxctl
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
- consumers: humans, symphony-validator and future validator extensions
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
- consumers: humans, symphony-validator and future validator extensions, future qxctl
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

#### secure-identity-access-governance
##### INTENT.md
- path: `modules/secure-identity-access-governance/INTENT.md`
- title: Symphony Secure Identity and Access Governance Intent
- surface_type: module intent
- truth_role: intent and authority boundaries for the node-local SSIAG foundation
- owner: secure-identity-access-governance maintainer
- scope: Defines purpose, monorepo relationship, security scope, non-scope, and owner-ratification boundaries.
- relationships:
  - declares -> `modules/secure-identity-access-governance/MANIFEST.md`
  - depends_on -> `INTENT.md`
- consumers: humans, reviewers, agentic tools, symphony-validator and future validator extensions, qxctl maintainers
- deferred_projections: JSON / JSONL portable evidence, graph relationship projection
- notes: Monorepo visibility does not grant runtime authority.
- status: canonical

##### MANIFEST.md
- path: `modules/secure-identity-access-governance/MANIFEST.md`
- title: Symphony Secure Identity and Access Governance Manifest
- surface_type: module contract truth
- truth_role: declared capabilities, contamination boundaries, dependencies, and installability
- owner: secure-identity-access-governance maintainer
- scope: Declares Go runtime surfaces, qxctl surfaces, provider posture, and independent lifecycle.
- relationships:
  - depends_on -> `modules/secure-identity-access-governance/INTENT.md`
  - declares -> `modules/secure-identity-access-governance/INSTALL.md`
  - declares -> `modules/secure-identity-access-governance/SPEC.md`
- consumers: humans, reviewers, qxctl, agentic tools, symphony-validator and future validator extensions
- deferred_projections: JSON / JSONL portable evidence, graph relationship projection
- notes: No credential-releasing provider is enabled by the scaffold.
- status: canonical

##### INSTALL.md
- path: `modules/secure-identity-access-governance/INSTALL.md`
- title: Symphony Secure Identity and Access Governance Install
- surface_type: module install guidance
- truth_role: command-line installation and uninstallation contract
- owner: secure-identity-access-governance maintainer
- scope: Defines user/system paths, build, verification, uninstall, purge, and configuration precedence.
- relationships:
  - depends_on -> `modules/secure-identity-access-governance/MANIFEST.md`
- consumers: humans, TOPS operators, qxctl maintainers, agentic tools
- deferred_projections: derived installation runbook
- notes: Per-TOPS launchd/systemd supervision and descriptor-only owner-provided integration are implemented.
- status: canonical

##### SKILL.md
- path: `modules/secure-identity-access-governance/SKILL.md`
- title: Symphony Secure Identity and Access Governance Skill
- surface_type: module skill guidance
- truth_role: safe operational and agentic usage guidance
- owner: secure-identity-access-governance maintainer
- scope: Defines safe-use rules, prohibited uses, and verification sequence.
- relationships:
  - depends_on -> `modules/secure-identity-access-governance/MANIFEST.md`
  - interprets -> `modules/secure-identity-access-governance/THREAT-MODEL.md`
- consumers: humans, maintainers, agentic tools
- deferred_projections: none
- notes: qxctl and Knowledge Vector surfaces must remain secret-free.
- status: canonical

##### SPEC.md
- path: `modules/secure-identity-access-governance/SPEC.md`
- title: Symphony Secure Identity and Access Governance Specification
- surface_type: module specification
- truth_role: normative SSIAG behavior and protocol boundaries
- owner: secure-identity-access-governance maintainer
- scope: Defines invariants, domain models, kernel-authenticated local metadata API, typed STAV producer, provider contract, installation, observability, and compatibility.
- relationships:
  - depends_on -> `modules/secure-identity-access-governance/MANIFEST.md`
  - depends_on -> `modules/secure-identity-access-governance/REQUIREMENTS.md`
- consumers: implementers, reviewers, agentic tools, symphony-validator and future validator extensions
- deferred_projections: protocol schema, conformance evidence
- notes: Mutation endpoints remain disabled pending security gates.
- status: canonical

##### ARCHITECTURE.md
- path: `modules/secure-identity-access-governance/ARCHITECTURE.md`
- title: Symphony Secure Identity and Access Governance Architecture
- surface_type: module architecture
- truth_role: component, trust-boundary, provider, qxctl, and SKV design
- owner: secure-identity-access-governance maintainer
- scope: Preserves monorepo-wide caller context and module-bounded install/runtime authority.
- relationships:
  - depends_on -> `modules/secure-identity-access-governance/INTENT.md`
  - interprets -> `knowledge/INTENT.md`
  - interprets -> `tools/qxctl/INTENT.md`
- consumers: humans, implementers, reviewers, agentic tools
- deferred_projections: architecture diagram, graph relationship projection
- notes: Identity/authorization and credential-use planes remain distinct.
- status: canonical

##### REQUIREMENTS.md
- path: `modules/secure-identity-access-governance/REQUIREMENTS.md`
- title: Symphony Secure Identity and Access Governance Requirements
- surface_type: module requirements
- truth_role: traceable functional, security, operational, portability, and SKV requirements
- owner: secure-identity-access-governance maintainer
- scope: Defines numbered release gates and owner decisions.
- relationships:
  - depends_on -> `modules/secure-identity-access-governance/ARCHITECTURE.md`
  - depends_on -> `modules/secure-identity-access-governance/THREAT-MODEL.md`
- consumers: implementers, reviewers, testers, agentic tools, symphony-validator and future validator extensions
- deferred_projections: requirements traceability matrix, conformance evidence
- notes: Requirements apply when their related capability is implemented.
- status: canonical

##### THREAT-MODEL.md
- path: `modules/secure-identity-access-governance/THREAT-MODEL.md`
- title: Symphony Secure Identity and Access Governance Threat Model
- surface_type: module security analysis
- truth_role: assets, actors, trust boundaries, threats, and required controls
- owner: secure-identity-access-governance maintainer
- scope: Covers SSIAG and planned provider risks without storing secret examples.
- relationships:
  - governs -> `modules/secure-identity-access-governance/SPEC.md`
  - governs -> `modules/secure-identity-access-governance/IMPLEMENTATION.md`
- consumers: security reviewers, implementers, operators, agentic tools
- deferred_projections: provider-specific threat reviews, security test evidence
- notes: Each operational provider requires an additional review.
- status: canonical

##### IMPLEMENTATION.md
- path: `modules/secure-identity-access-governance/IMPLEMENTATION.md`
- title: Symphony Secure Identity and Access Governance Procedural Implementation Guide
- surface_type: module implementation guide
- truth_role: phased implementation, verification, rollback, and release procedure
- owner: secure-identity-access-governance maintainer
- scope: Defines ordered phases from ratification through providers, TOPS integration, SCLV, and publication.
- relationships:
  - depends_on -> `modules/secure-identity-access-governance/REQUIREMENTS.md`
  - depends_on -> `modules/secure-identity-access-governance/THREAT-MODEL.md`
  - may_check -> `tools/symphony-validator/SPEC.md`
- consumers: implementers, reviewers, operators, agentic tools
- deferred_projections: release checklist, requirements traceability evidence
- notes: SCLV recording waits for real PR and merge evidence.
- status: canonical

##### README.md
- path: `modules/secure-identity-access-governance/README.md`
- title: Symphony Secure Identity and Access Governance README
- surface_type: module orientation
- truth_role: concise implementation status and contributor entrypoint
- owner: secure-identity-access-governance maintainer
- scope: Summarizes scaffold behavior and directs readers to canonical design surfaces.
- relationships:
  - derives_from -> `modules/secure-identity-access-governance/MANIFEST.md`
  - derives_from -> `modules/secure-identity-access-governance/ARCHITECTURE.md`
- consumers: humans, contributors, agentic tools
- deferred_projections: SODV-governed public module page
- notes: Repository source truth; not an independently authorized publication pipeline.
- status: canonical

#### stav-append-authority
##### INTENT.md
- path: `modules/stav-append-authority/INTENT.md`
- title: STAV Append Authority Intent
- surface_type: module intent
- truth_role: implementation purpose and canonical-authority boundary
- owner: STAV append-authority maintainer
- scope: Defines the independently installable Go append-authority role and canonical-authority boundary.
- relationships:
  - depends_on -> `knowledge/stav/INTENT.md`
  - declares -> `modules/stav-append-authority/MANIFEST.md`
- consumers: humans, reviewers, agentic tools, qxctl maintainers
- deferred_projections: operational append service
- notes: The module implements STAV and never owns STAV protocol truth.
- status: canonical

##### MANIFEST.md
- path: `modules/stav-append-authority/MANIFEST.md`
- title: STAV Append Authority Manifest
- surface_type: module contract truth
- truth_role: capability, dependency, contamination, and absent-surface declaration
- owner: STAV append-authority maintainer
- scope: Declares the Go/cgo boundary, authenticated append authority, reversible lifecycle, native supervision, and closed operational gates.
- relationships:
  - depends_on -> `modules/stav-append-authority/INTENT.md`
  - depends_on -> `knowledge/stav/MANIFEST.md`
  - declares -> `modules/stav-append-authority/SPEC.md`
- consumers: humans, reviewers, qxctl, agentic tools, symphony-validator and future validator extensions
- deferred_projections: release and conformance evidence
- notes: Operational listener, durability, read projection, SSIAG producer, and native supervision are implemented.
- status: canonical

##### INSTALL.md
- path: `modules/stav-append-authority/INSTALL.md`
- title: STAV Append Authority Installation
- surface_type: module install guidance
- truth_role: independent executable build, installation, and uninstallation contract
- owner: STAV append-authority maintainer
- scope: Defines user/system binary and per-TOPS supervisor lifecycle with explicit state preservation.
- relationships: depends_on -> `modules/stav-append-authority/MANIFEST.md`
- consumers: humans, TOPS operators, release tooling, agentic tools
- deferred_projections: release packaging artifacts
- notes: Host installation remains separate from TOPS enrollment and supervisor installation.
- status: canonical

##### SKILL.md
- path: `modules/stav-append-authority/SKILL.md`
- title: STAV Append Authority Skill
- surface_type: module skill guidance
- truth_role: safe caller interaction and implementation stop conditions
- owner: STAV append-authority maintainer
- scope: Permits inspection and verification while prohibiting schema invention and unauthorized ledger mutation.
- relationships:
  - depends_on -> `knowledge/stav/SKILL.md`
  - interprets -> `modules/stav-append-authority/THREAT-MODEL.md`
- consumers: humans, reviewers, agentic tools
- deferred_projections: none
- notes: Lifecycle actions require the applicable target-host permission; caller type is not evaluated.
- status: canonical

##### SPEC.md
- path: `modules/stav-append-authority/SPEC.md`
- title: STAV Append Authority Specification
- surface_type: module specification
- truth_role: operational append-authority, supervision, path, and lifecycle behavior
- owner: STAV append-authority maintainer
- scope: Defines install targets, per-TOPS path resolution, authenticated append behavior, native supervision, and fail-closed commands.
- relationships:
  - depends_on -> `modules/stav-append-authority/MANIFEST.md`
  - implements -> `knowledge/stav/SPEC.md`
- consumers: implementers, reviewers, qxctl, agentic tools, symphony-validator and future validator extensions
- deferred_projections: release conformance evidence
- notes: Canonical protocol content remains owned by knowledge/stav.
- status: canonical

##### ARCHITECTURE.md
- path: `modules/stav-append-authority/ARCHITECTURE.md`
- title: STAV Append Authority Architecture
- surface_type: module architecture
- truth_role: source-truth direction, current boundary, future process shape, and TOPS isolation
- owner: STAV append-authority maintainer
- scope: Describes the implemented authenticated single-writer, durability, query, and liveness boundaries.
- relationships:
  - depends_on -> `modules/stav-append-authority/INTENT.md`
  - interprets -> `knowledge/stav/SPEC.md`
- consumers: humans, implementers, reviewers, agentic tools
- deferred_projections: rendered runtime trust-boundary diagram
- notes: Supervision owns liveness only and grants no ledger or producer authority.
- status: canonical

##### REQUIREMENTS.md
- path: `modules/stav-append-authority/REQUIREMENTS.md`
- title: STAV Append Authority Requirements
- surface_type: module requirements
- truth_role: traceable namespace, lifecycle, path, and closed-gate requirements
- owner: STAV append-authority maintainer
- scope: Defines numbered requirements for the ratified increment and future stop conditions.
- relationships:
  - depends_on -> `modules/stav-append-authority/ARCHITECTURE.md`
  - depends_on -> `modules/stav-append-authority/THREAT-MODEL.md`
- consumers: implementers, reviewers, testers, agentic tools, symphony-validator and future validator extensions
- deferred_projections: requirements traceability evidence
- notes: Operational v1 and supervision requirements are active; deferred capabilities retain explicit gates.
- status: canonical

##### THREAT-MODEL.md
- path: `modules/stav-append-authority/THREAT-MODEL.md`
- title: STAV Append Authority Threat Model
- surface_type: module security analysis
- truth_role: current lifecycle controls and future operational threat gates
- owner: STAV append-authority maintainer
- scope: Covers executable lifecycle, TOPS isolation, future producer, ordering, durability, redaction, and repair threats.
- relationships:
  - governs -> `modules/stav-append-authority/SPEC.md`
  - governs -> `modules/stav-append-authority/IMPLEMENTATION.md`
- consumers: security reviewers, implementers, operators, agentic tools
- deferred_projections: producer-specific and storage-specific security reviews
- notes: Operational controls may not be invented below the canonical vector.
- status: canonical

##### IMPLEMENTATION.md
- path: `modules/stav-append-authority/IMPLEMENTATION.md`
- title: STAV Append Authority Implementation Guide
- surface_type: module implementation guide
- truth_role: phased procedure from namespace scaffold through operational SSIAG producer integration
- owner: STAV append-authority maintainer
- scope: Records completed canonical content, durability, IPC, native supervision, qxctl, and SSIAG producer phases plus deferred node-troll and Go 1.27 work.
- relationships:
  - depends_on -> `modules/stav-append-authority/REQUIREMENTS.md`
  - depends_on -> `modules/stav-append-authority/THREAT-MODEL.md`
  - may_check -> `tools/symphony-validator/SPEC.md`
- consumers: implementers, reviewers, operators, agentic tools
- deferred_projections: release checklist and conformance evidence
- notes: SCLV recording waits for real PR and merge evidence.
- status: canonical

##### README.md
- path: `modules/stav-append-authority/README.md`
- title: STAV Append Authority
- surface_type: module orientation
- truth_role: concise operational status and contributor entrypoint
- owner: STAV append-authority maintainer
- scope: Directs readers to the Contract Quad and summarizes the operational single-writer boundary.
- relationships:
  - derives_from -> `modules/stav-append-authority/MANIFEST.md`
  - derives_from -> `modules/stav-append-authority/ARCHITECTURE.md`
- consumers: humans, contributors, agentic tools
- deferred_projections: SODV-governed public module page
- notes: Repository source truth; no public publication is authorized here.
- status: canonical

### First-Party Shared Libraries

#### Libraries README
- path: `libraries/README.md`
- title: Symphony First-Party Libraries
- surface_type: shared-library topology doctrine
- truth_role: implementation placement and runtime-authority boundary
- owner: Symphony root governance
- scope: Defines build-time shared code and versioned native development packages as distinct from independently installed resident runtime modules.
- relationships: depends_on -> `INTENT.md`; governs -> `libraries/stav-protocol-go/MANIFEST.md`; governs -> `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- consumers: implementers, reviewers, agentic tools, symphony-validator and future validator extensions
- deferred_projections: dependency graph and release evidence
- notes: Libraries own no canonical protocol truth or resident operational identity; a native development package may still be independently installed and removed through a receipt.
- status: canonical

#### STAV Protocol Kernel
- path: `libraries/stav-protocol-go/MANIFEST.md`
- title: STAV Protocol Kernel for Go
- surface_type: first-party shared Go implementation library
- truth_role: implementation truth for ratified STAV v1 protocol mechanics
- owner: STAV protocol-kernel maintainer
- scope: Implements strict I-JSON/JCS, typed semantic/operational envelopes, identifiers, SHA-256 domains, and bounded local framing without runtime authority.
- relationships: depends_on -> `knowledge/stav/SPEC.md`; may_check -> `tools/symphony-validator/SPEC.md`
- consumers: `modules/stav-append-authority/`, `modules/secure-identity-access-governance/`, `tools/qxctl/`, implementers, reviewers, agentic tools
- deferred_projections: versioned library release and conformance evidence
- notes: No binary, installer, resident, socket, state, authentication, authorization, or ledger behavior.
- status: canonical

#### STAV Protocol Kernel Go 1.27 Migration
- path: `libraries/stav-protocol-go/GO_1_27_MIGRATION.md`
- title: Go 1.27 Confirmed-Release Migration
- surface_type: toolchain migration procedure
- truth_role: compatibility and conformance gate
- owner: STAV protocol-kernel maintainer
- scope: Keeps Go 1.26.5 in production until Go 1.27 GA and byte-identical differential validation pass.
- relationships: depends_on -> `knowledge/stav/SPEC.md`; governs -> `libraries/stav-protocol-go/`
- consumers: maintainers, release engineers, reviewers, agentic tools
- deferred_projections: dual-toolchain CI evidence and release record
- notes: Toolchain adoption cannot change STAV wire bytes, digests, public APIs, or authority boundaries.
- status: canonical

#### Knowledge Vector Engine C++ Foundation INTENT.md
- path: `libraries/knowledge-vector-engine-cpp/INTENT.md`
- title: Knowledge Vector Engine C++ Foundation Intent
- surface_type: first-party shared-library intent
- truth_role: implemented authority-free foundation purpose and boundary
- owner: SKV foundation maintainers
- scope: Defines bounded JSON, digest, path, snapshot, process, and STSC temporal-validation mechanics without semantic authority.
- relationships: depends_on -> `knowledge/SPEC.md`; declares -> `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- consumers: coordinator and future vector-engine implementers, reviewers, agentic tools
- deferred_projections: dependency and conformance evidence
- notes: No executable or canonical mutation authority belongs to the library.
- status: canonical

#### Knowledge Vector Engine C++ Foundation MANIFEST.md
- path: `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- title: Knowledge Vector Engine C++ Foundation Manifest
- surface_type: first-party native library manifest
- truth_role: implemented component, dependency, installability, and authority boundary
- owner: SKV foundation maintainers
- scope: Declares the C++26 static target, `0.1.0-dev` components including canonical temporal validation, pinned JSON dependency, and versioned install paths.
- relationships: depends_on -> `libraries/knowledge-vector-engine-cpp/INTENT.md`; implements -> `knowledge/SPEC.md`
- consumers: coordinator and future vector engines, packagers, reviewers, agentic tools
- deferred_projections: package inventory and SBOM evidence
- notes: nlohmann/json is vendored and has no runtime download or validator linkage.
- status: canonical

#### Knowledge Vector Engine C++ Foundation INSTALL.md
- path: `libraries/knowledge-vector-engine-cpp/INSTALL.md`
- title: Knowledge Vector Engine C++ Foundation Installation
- surface_type: native library installation contract
- truth_role: versioned build, test, install, consumer, and uninstall procedure
- owner: SKV foundation maintainers
- scope: Defines CMake build and receipt-owned prefix lifecycle without runtime activation.
- relationships: depends_on -> `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- consumers: implementers, packagers, reviewers, agentic tools
- deferred_projections: qxctl lifecycle evidence
- notes: The development package is not a resident module or shared runtime dependency.
- status: canonical

#### Knowledge Vector Engine C++ Foundation SKILL.md
- path: `libraries/knowledge-vector-engine-cpp/SKILL.md`
- title: Knowledge Vector Engine C++ Foundation Skill
- surface_type: native foundation skill guidance
- truth_role: safe implementation and review procedure
- owner: SKV foundation maintainers
- scope: Guides limits, strict parsing, path safety, STSC temporal validation, response framing, and authority separation.
- relationships: depends_on -> `libraries/knowledge-vector-engine-cpp/SPEC.md`; depends_on -> `knowledge/SKILL.md`
- consumers: C++ implementers, reviewers, agentic tools
- deferred_projections: conformance checklist
- notes: Vector semantics and host permissions never belong in the shared library.
- status: canonical

#### Knowledge Vector Engine C++ Foundation SPEC.md
- path: `libraries/knowledge-vector-engine-cpp/SPEC.md`
- title: Knowledge Vector Engine C++ Foundation Specification
- surface_type: native foundation specification
- truth_role: exact implemented limits, digest, path, snapshot, and dependency contract
- owner: SKV foundation maintainers
- scope: Defines `0.1.0-dev` mechanics, the strict shared parser defaults and explicit per-engine finite value-count override, canonical Gregorian/UTC profiles, and adversarial rejection requirements.
- relationships: depends_on -> `knowledge/SPEC.md`; governs -> `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`
- consumers: coordinator and future vector engines, testers, reviewers
- deferred_projections: protocol conformance report
- notes: The version is developmental and not published.
- status: canonical

#### Knowledge Vector Engine C++ Foundation FEATURES.md
- path: `libraries/knowledge-vector-engine-cpp/FEATURES.md`
- title: Knowledge Vector Engine C++ Foundation Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the authority-free shared engine foundation
- owner: SKV foundation maintainers
- scope: Owns the experimental `ssfv:symphony:knowledge-vector-engine-foundation` record for exact source scope `libraries/knowledge-vector-engine-cpp`.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/SPEC.md`; declares -> `ssfv:symphony:knowledge-vector-engine-foundation`
- consumers: symphony-ssfv, qxctl, foundation and vector-engine maintainers, reviewers, agentic tools
- deferred_projections: portable SSFV graph, capability catalog, conformance documentation
- notes: Records implemented mechanics only; no runtime identity, semantic authority, or published package is claimed.
- status: canonical

#### Knowledge Vector Engine C++ Foundation CMakeLists.txt
- path: `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`
- title: Knowledge Vector Engine C++ Foundation Build Contract
- surface_type: native build and install contract
- truth_role: implementation build, static-link, test, package, receipt, and uninstall truth
- owner: SKV foundation maintainers
- scope: Builds and installs the versioned `Symphony::KnowledgeVectorEngine` CMake package.
- relationships: implements -> `libraries/knowledge-vector-engine-cpp/SPEC.md`; consumed_by -> `modules/knowledge-session-coordinator/CMakeLists.txt`
- consumers: CMake, implementers, packagers, reviewers
- deferred_projections: reproducible build provenance
- notes: No runtime dependency download is permitted.
- status: canonical

#### Knowledge Vector Engine C++ Dependency Record
- path: `libraries/knowledge-vector-engine-cpp/third_party/README.md`
- title: Knowledge Vector Engine Third-Party Source Record
- surface_type: dependency provenance record
- truth_role: pinned upstream, checksum, license, and linkage evidence
- owner: SKV foundation maintainers
- scope: Records nlohmann/json `v3.12.0` and its official release checksum.
- relationships: depends_on -> `libraries/knowledge-vector-engine-cpp/SPEC.md`; informs -> `knowledge/sodv/SPEC.md`
- consumers: dependency reviewers, packagers, SODV maintainers, agentic tools
- deferred_projections: SBOM and license report
- notes: Upgrades require a new reviewed dependency and release-evidence increment.
- status: canonical

#### Knowledge Vector Engine Foundation Conformance Tests
- path: `libraries/knowledge-vector-engine-cpp/tests/foundation_test.cpp`
- title: Knowledge Vector Engine Foundation Conformance Tests
- surface_type: C++26 conformance-test implementation
- truth_role: digest, bounded protocol, path, snapshot, temporal, schema identity, and lifecycle-invariant proof
- owner: SKV foundation maintainers
- scope: Verifies authority-free mechanics, canonical Gregorian/UTC conformance, and the exact common schema identities consumed across installed engines and the coordinator.
- relationships: verifies -> `libraries/knowledge-vector-engine-cpp/SPEC.md`; verifies -> `knowledge/schemas/v1/MANIFEST.md`; verifies -> `knowledge/schemas/v2/MANIFEST.md`
- consumers: foundation/coordinator/vector-engine maintainers, reviewers, release gates
- deferred_projections: portable conformance evidence
- notes: Schema checks establish identity and invariant anchors, not domain authority or runtime activation.
- status: canonical

#### Knowledge Vector Engine Temporal API
- path: `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp`
- title: Knowledge Vector Engine Temporal Validation API
- surface_type: C++26 public foundation header
- truth_role: authority-free common temporal validation interface
- owner: SKV foundation maintainers
- scope: Declares canonical civil-date, whole-second UTC, and exact-nanosecond UTC representation validators.
- relationships: implements -> `knowledge/TIME.md`; governed_by -> `libraries/knowledge-vector-engine-cpp/SPEC.md`
- consumers: coordinator and vector-engine implementations, conformance tests, reviewers
- deferred_projections: generated native API reference
- notes: The interface validates text only and owns no clock, timezone, freshness, sequence, or timestamp authority.
- status: canonical

#### Knowledge Vector Engine Temporal Implementation
- path: `libraries/knowledge-vector-engine-cpp/src/temporal.cpp`
- title: Knowledge Vector Engine Temporal Validation Implementation
- surface_type: C++26 foundation implementation
- truth_role: real-Gregorian and exact-UTC implementation proof
- owner: SKV foundation maintainers
- scope: Rejects impossible dates, year zero, invalid time fields, offsets, leap-second text, and noncanonical precision.
- relationships: implements -> `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp`; conforms_to -> `knowledge/TIME.md`
- consumers: statically linked coordinator and vector engines, conformance tests, reviewers
- deferred_projections: portable conformance evidence
- notes: Domain order and freshness remain outside the foundation.
- status: canonical

### SACV Canonical Knowledge Vector

#### SACV INTENT.md
- path: `knowledge/sacv/INTENT.md`
- title: Symphony API Contract Vector Intent
- surface_type: knowledge-vector intent
- truth_role: canonical API-contract governance intent
- owner: SACV maintainer
- scope: Defines API-first source truth, OpenAPI 3.2.0 targeting, distributed semantic ownership, and security/publication boundaries.
- relationships: declares -> `knowledge/sacv/MANIFEST.md`; depends_on -> `knowledge/sodv/SPEC.md`; depends_on -> `knowledge/SPEC.md`
- consumers: humans, reviewers, API owners, agentic tools, `symphony-sacv`, qxctl, symphony-validator and future validator extensions
- deferred_projections: OpenAPI validation evidence, documentation, SDK, and graph projections
- notes: Authorizes governance and proposal-engine behavior, not canonical apply, an endpoint, or a remote listener.
- status: canonical

#### SACV MANIFEST.md
- path: `knowledge/sacv/MANIFEST.md`
- title: Symphony API Contract Vector Manifest
- surface_type: knowledge-vector manifest
- truth_role: canonical API ownership and placement contract
- owner: SACV maintainer
- scope: Declares SACV-owned policy, registry truth, and independent proposal-engine installability while retaining endpoint semantics with domain owners.
- relationships: depends_on -> `knowledge/sacv/INTENT.md`; depends_on -> `knowledge/SPEC.md`; declares -> `knowledge/sacv/SPEC.md`; declares -> `knowledge/sacv/REGISTRY.md`
- consumers: humans, reviewers, module and vector owners, SKVI, SODV
- deferred_projections: validator and publication evidence
- notes: OpenAPI is a conditional typed artifact, not a mandatory fifth Contract Quad file.
- status: canonical

#### SACV SKILL.md
- path: `knowledge/sacv/SKILL.md`
- title: Symphony API Contract Vector Skill
- surface_type: knowledge-vector skill guidance
- truth_role: safe API-contract authoring and review procedure
- owner: SACV maintainer
- scope: Guides all callers and proposal-engine users without authorizing canonical apply, endpoints, publication, live requests, or MCP exposure.
- relationships: depends_on -> `knowledge/sacv/SPEC.md`; depends_on -> `knowledge/SPEC.md`; interprets -> `knowledge/sodv/SPEC.md`
- consumers: humans, API maintainers, security reviewers, agentic tools
- deferred_projections: conformance workflow automation
- notes: Security schemes and server URLs may not be invented.
- status: canonical

#### SACV SPEC.md
- path: `knowledge/sacv/SPEC.md`
- title: Symphony API Contract Vector Specification
- surface_type: knowledge-vector specification
- truth_role: normative API-contract governance
- owner: SACV maintainer
- scope: Defines OpenAPI versioning, ownership, registry, compatibility, security, proposal-engine operations, derivation, and publication boundaries.
- relationships: depends_on -> `knowledge/sacv/MANIFEST.md`; depends_on -> `knowledge/SPEC.md`; governs -> future owner-controlled OpenAPI descriptions; depends_on -> `knowledge/sodv/SPEC.md`
- consumers: API owners, implementers, reviewers, `symphony-sacv`, qxctl, symphony-validator and future validator/generator extensions
- deferred_projections: generated bindings, SDKs, Mintlify documentation, MCP tools
- notes: Canonical descriptions target OpenAPI 3.2.0; none are registered and programmatic apply remains disabled.
- status: canonical

#### SACV REGISTRY.md
- path: `knowledge/sacv/REGISTRY.md`
- title: Symphony API Contract Registry
- surface_type: canonical API-contract registry
- truth_role: routing and ownership map for HTTP API entry documents
- owner: SACV maintainer
- scope: Registers owner, path, versions, audience, transport, security, publication, SDK, and lifecycle state.
- relationships: depends_on -> `knowledge/sacv/SPEC.md`; indexes -> future canonical owner API descriptions
- consumers: humans, SKVI, SODV, symphony-validator and future validator/generator extensions
- deferred_projections: machine-readable registry evidence
- notes: The empty registry is intentional; placeholder endpoint documents are prohibited.
- status: canonical

#### SACV OpenAPI 3.2 Profile
- path: `knowledge/sacv/profiles/openapi-3.2.md`
- title: SACV OpenAPI 3.2 Profile
- surface_type: API-contract standards profile
- truth_role: normative OpenAPI 3.2.0 authoring and compatibility policy
- owner: SACV maintainer
- scope: Defines required posture, reference handling, compatibility gates, and exclusions for canonical descriptions.
- relationships: depends_on -> `knowledge/sacv/SPEC.md`; governs -> future canonical owner OpenAPI descriptions
- consumers: API owners, implementers, reviewers, symphony-validator and future validator/generator extensions
- deferred_projections: lint and compatibility evidence
- notes: A lagging consumer must defer or fail, not silently downgrade.
- status: canonical

#### SACV Mintlify Publication Profile
- path: `knowledge/sacv/profiles/mintlify-publication.md`
- title: SACV Mintlify Publication Profile
- surface_type: API publication profile
- truth_role: SACV-to-SODV publication boundary
- owner: SACV and SODV maintainers
- scope: Defines preconditions and default-deny controls for Mintlify, SDK examples, live requests, and MCP projections.
- relationships: depends_on -> `knowledge/sacv/SPEC.md`; depends_on -> `knowledge/sodv/SPEC.md`
- consumers: documentation maintainers, API owners, security reviewers, agentic tools
- deferred_projections: Mintlify configuration, MDX, SDK examples, MCP tools
- notes: Vendor configuration is derived and currently unauthorized.
- status: canonical

#### SACV v1 Schema Manifest
- path: `knowledge/sacv/schemas/v1/MANIFEST.md`
- title: SACV v1 Schema Manifest
- surface_type: vector-specific protocol schema manifest
- truth_role: canonical inventory for exact SACV engine payload and result schemas
- owner: SACV maintainers
- scope: Declares the registry-entry, check, diff, proposal-input, and projection schema family.
- relationships: depends_on -> `knowledge/sacv/SPEC.md`; implemented_by -> `modules/sacv-engine/SPEC.md`
- consumers: SACV engine, qxctl, symphony-validator, conformance tests, reviewers
- deferred_projections: rendered protocol reference
- notes: Schemas govern engine data and create no HTTP endpoint.
- status: canonical

#### SACV Registry Entry Schema
- path: `knowledge/sacv/schemas/v1/registry-entry.schema.json`
- title: SACV Registry Entry v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical normalized API registry entry shape
- owner: SACV maintainers
- scope: Closes the exact thirteen-field API identity, ownership, profile, publication, SDK, and lifecycle record.
- relationships: depends_on -> `knowledge/sacv/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sacv-engine/SPEC.md`
- consumers: SACV engine, qxctl proposal callers, validator, tests, reviewers
- deferred_projections: registry forms and reference documentation
- notes: An entry routes to owner truth and does not centralize endpoint semantics.
- status: canonical

#### SACV Check Result Schema
- path: `knowledge/sacv/schemas/v1/check-result.schema.json`
- title: SACV Check Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical registry and API conformance evidence shape
- owner: SACV maintainers
- scope: Closes registry/contract digests, counts, findings, valid state, read-only status, and disabled apply.
- relationships: depends_on -> `knowledge/sacv/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sacv-engine/SPEC.md`
- consumers: qxctl, SACV engine, validator, tests, reviewers
- deferred_projections: conformance reports
- notes: Invalid evidence never authorizes repair.
- status: canonical

#### SACV Diff Input Schema
- path: `knowledge/sacv/schemas/v1/diff-input.schema.json`
- title: SACV Diff Input v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical bounded comparison request shape
- owner: SACV maintainers
- scope: Binds baseline and candidate owner-document paths to exact tagged digests.
- relationships: depends_on -> `knowledge/sacv/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sacv-engine/SPEC.md`
- consumers: qxctl diff callers, SACV engine, tests, reviewers
- deferred_projections: compatibility review forms
- notes: Diff input is evidence-only and contains no remote reference.
- status: canonical

#### SACV Diff Result Schema
- path: `knowledge/sacv/schemas/v1/diff-result.schema.json`
- title: SACV Diff Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic compatibility-evidence shape
- owner: SACV maintainers
- scope: Classifies identical, additive, breaking, and review-required operation changes without accepting them.
- relationships: depends_on -> `knowledge/sacv/schemas/v1/diff-input.schema.json`; implemented_by -> `modules/sacv-engine/SPEC.md`
- consumers: qxctl, SACV engine, API reviewers, tests
- deferred_projections: compatibility reports
- notes: Compatibility evidence is noncanonical and cannot ratify a change.
- status: canonical

#### SACV Proposal Input Schema
- path: `knowledge/sacv/schemas/v1/proposal-input.schema.json`
- title: SACV Proposal Input v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical caller-declared registry proposal input shape
- owner: SACV maintainers
- scope: Binds provider-neutral repository/session context and one register or replace operation to expected state.
- relationships: depends_on -> `knowledge/sacv/schemas/v1/registry-entry.schema.json`; depends_on -> `knowledge/schemas/v1/proposal.schema.json`; implemented_by -> `modules/sacv-engine/SPEC.md`
- consumers: qxctl proposal callers, SACV engine, tests, reviewers
- deferred_projections: proposal forms
- notes: Caller declares semantic ownership; the engine validates but does not decide it.
- status: canonical

#### SACV Projection Schema
- path: `knowledge/sacv/schemas/v1/projection.schema.json`
- title: SACV Registry Inventory Projection v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical disposable projection-result shape
- owner: SACV maintainers
- scope: Closes normalized registry inventory, contract digests, operation counts, and noncanonical rebuildable status.
- relationships: depends_on -> `knowledge/sacv/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sacv-engine/SPEC.md`
- consumers: qxctl, SACV engine, tests, graph/search planners
- deferred_projections: analytical and graph inventory projections
- notes: It contains no raw bundle, runtime binding, SDK, or publication artifact.
- status: canonical

### SSIAG Canonical Knowledge Vector

#### SSIAG INTENT.md
- path: `knowledge/ssiag/INTENT.md`
- title: Symphony Secure Identity and Access Governance Intent
- surface_type: knowledge-vector intent
- truth_role: canonical SSIAG purpose and complete decision-chain authority
- owner: SSIAG knowledge maintainer
- scope: Defines SSIAG source truth, relationship intent, security boundaries, and deferred decisions.
- relationships: declares -> `knowledge/ssiag/MANIFEST.md`; governs -> `modules/secure-identity-access-governance/INTENT.md`
- consumers: humans, reviewers, agentic tools, qxctl and SSIAG implementers
- deferred_projections: graph relationship projection
- notes: Graph-like semantics do not authorize a graph database.
- status: canonical

#### SSIAG MANIFEST.md
- path: `knowledge/ssiag/MANIFEST.md`
- title: Symphony Secure Identity and Access Governance Manifest
- surface_type: knowledge-vector manifest
- truth_role: canonical SSIAG authority and language-boundary declaration
- owner: SSIAG knowledge maintainer
- scope: Declares protocol ownership, identity separation, provider boundaries, and non-authorizations.
- relationships: depends_on -> `knowledge/ssiag/INTENT.md`; declares -> `knowledge/ssiag/SPEC.md`
- consumers: humans, reviewers, provider and foundation implementers, SKVI
- deferred_projections: JSON / JSONL portable evidence
- notes: The indexed surface is canonical source truth while its internal protocol status remains draft pending ratification.
- status: canonical

#### SSIAG SKILL.md
- path: `knowledge/ssiag/SKILL.md`
- title: Symphony Secure Identity and Access Governance Skill
- surface_type: knowledge-vector skill guidance
- truth_role: safe caller procedure for SSIAG changes
- owner: SSIAG knowledge maintainer
- scope: Defines reading order, caller-neutral host authority, change procedure, and stop conditions.
- relationships: depends_on -> `knowledge/ssiag/SPEC.md`; interprets -> `knowledge/stav/SPEC.md`
- consumers: humans, maintainers, reviewers, agentic tools
- deferred_projections: none
- notes: Callers may query, propose, or use a future apply operation only within effective host permissions; no supported operation may bypass policy or expose credentials.
- status: canonical

#### SSIAG SPEC.md
- path: `knowledge/ssiag/SPEC.md`
- title: Symphony Secure Identity and Access Governance Specification
- surface_type: knowledge-vector specification
- truth_role: canonical SSIAG vocabulary, relationship, extension, and provider protocol truth
- owner: SSIAG knowledge maintainer
- scope: Defines graph-like nodes/edges, immutable IDs, Go-only foundation, exact deny-by-default authorization, non-transferable capabilities, provider boundary, qxctl, and STAV projection.
- relationships: depends_on -> `knowledge/ssiag/MANIFEST.md`; governs -> `knowledge/ssiag/schemas/v1/MANIFEST.md`; governs -> `modules/secure-identity-access-governance/SPEC.md`; governs -> `modules/ssiag-provider-macos-keychain/SPEC.md`
- consumers: implementers, reviewers, qxctl, provider modules, agentic tools
- deferred_projections: graph view and rendered authorization schema documentation
- notes: Caller authentication, endpoint trust, native supervision/runtime ownership, exact authorization decisions, non-transferable capability evidence, and STAV producer integration are implemented; canonical apply and credential delivery remain gated.
- status: canonical

#### SSIAG Provider Binding Lifecycle
- path: `knowledge/ssiag/PROVIDER-LIFECYCLE.md`
- title: SSIAG Provider Binding Lifecycle
- surface_type: canonical provider installation and protected binding contract
- truth_role: exact bounded inventory, binding state, plan/apply, audit, and recovery truth
- owner: SSIAG knowledge maintainer
- scope: Defines six stable command/backend identities, six local routes, one host-owner-only receipt-bound offline recovery receptor, nine strict protocols, no-newest selection, compare-and-swap, and four-stage crash recovery while all operational provider flags remain false.
- relationships: depends_on -> `knowledge/ssiag/SPEC.md`; depends_on -> `knowledge/stav/SPEC.md`; governs -> `modules/secure-identity-access-governance/internal/provider/binding.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/provider_binding.go`
- consumers: SSIAG foundation, qxctl, STAV, validators, conformance tests, reviewers
- deferred_projections: provider installation compatibility and binding history views
- notes: Binding administration changes safe metadata state only; it never invokes a provider or transports secret material.
- status: canonical

#### SSIAG Provider Readiness Contract
- path: `knowledge/ssiag/PROVIDER-READINESS.md`
- title: SSIAG Provider Readiness Contract
- surface_type: canonical provider bundle, signing-policy, session, and readiness contract
- truth_role: structural validation, native policy match, operational eligibility, packaging, and safe administration truth
- owner: SSIAG knowledge maintainer
- scope: Defines the exact multi-file bundle, protected native requirement, three-layer readiness model, permission-backed qxctl route, compatibility, recovery, and Phase 10B operational prohibition.
- relationships: depends_on -> `knowledge/ssiag/SPEC.md`; governs -> `modules/secure-identity-access-governance/internal/provider/readiness.go`; governs -> `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/SignedBundleReadiness.swift`; consumed_by -> `tools/qxctl/internal/ssiagclient/provider_readiness.go`
- consumers: SSIAG foundation, macOS provider, qxctl, validators, release tooling, conformance tests, reviewers
- deferred_projections: production signing and notarization release evidence
- notes: Structural or native-policy success never implies operational eligibility; all provider operations and Keychain access remain disabled.
- status: canonical

#### SSIAG v1 Schema Manifest
- path: `knowledge/ssiag/schemas/v1/MANIFEST.md`
- title: SSIAG Canonical Schemas v1
- surface_type: canonical protocol schema manifest
- truth_role: inventory and boundary for exact authorization, policy, provider-control, and provider-trust schemas
- owner: SSIAG knowledge maintainer
- scope: Declares thirty-two bounded caller-neutral SSIAG v1 contracts, including metadata-only provider trust, protected exact-installation binding, and three-layer readiness boundaries.
- relationships: depends_on -> `knowledge/ssiag/SPEC.md`; governs -> `modules/secure-identity-access-governance/internal/policy/policy.go`; governs -> `modules/secure-identity-access-governance/internal/provider/trust.go`; governs -> `modules/knowledge-session-coordinator/src/authority_session.cpp`
- consumers: SSIAG, provider adapters, qxctl, coordinator, validator, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: These schemas define safe evidence and explicitly disabled secret-channel descriptors, not credentials, bearer tokens, grants by possession, or canonical apply authority.
- status: canonical

#### SSIAG macOS Signing Policy Schema
- path: `knowledge/ssiag/schemas/v1/macos-signing-policy.schema.json`
- title: SSIAG macOS Signing Policy v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical receipt-owned native code-requirement policy shape
- owner: SSIAG knowledge maintainer
- scope: Closes the protocol and bounded adapter requirement text evaluated natively inside the staged Swift bundle.
- relationships: depends_on -> `knowledge/ssiag/PROVIDER-READINESS.md`; implemented_by -> `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/SignedBundleReadiness.swift`
- consumers: macOS provider builder, adapter, validators, conformance tests, reviewers
- deferred_projections: safe compiled-requirement digest evidence
- notes: Requirement text is protected package input and never qxctl or readiness output.
- status: canonical

#### SSIAG Provider Readiness Observation Request Schema
- path: `knowledge/ssiag/schemas/v1/provider-readiness-observation-request.schema.json`
- title: SSIAG Provider Readiness Observation Request v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical permission-backed readiness request shape
- owner: SSIAG knowledge maintainer
- scope: Closes request/correlation identity and caller-neutral host-owner or granted-permission basis without path or signing-policy inputs.
- relationships: depends_on -> `knowledge/ssiag/PROVIDER-READINESS.md`; consumed_by -> `tools/qxctl/internal/ssiagclient/provider_readiness.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: none
- notes: The request is non-mutating and carries no provider payload or secret.
- status: canonical

#### SSIAG Provider Readiness Observation Schema
- path: `knowledge/ssiag/schemas/v1/provider-readiness-observation.schema.json`
- title: SSIAG Safe Provider Readiness Observation v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical native safe-observation shape
- owner: SSIAG knowledge maintainer
- scope: Separates structural validation, native policy match, disabled operational eligibility, safe signing digests, and bounded security-session capability flags.
- relationships: depends_on -> `knowledge/ssiag/PROVIDER-READINESS.md`; implemented_by -> `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/SignedBundleReadiness.swift`
- consumers: SSIAG provider launcher, qxctl, validators, conformance tests, reviewers
- deferred_projections: platform-readiness diagnostics
- notes: Certificates, profile/entitlement payloads, requirements, native errors, session IDs, operations, and secrets are excluded.
- status: canonical

#### SSIAG Provider Readiness Result Schema
- path: `knowledge/ssiag/schemas/v1/provider-readiness-result.schema.json`
- title: SSIAG Provider Readiness Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical TOPS/provider/installation-bound safe readiness result
- owner: SSIAG knowledge maintainer
- scope: Closes the exact provider identity, safe observation, whole-second UTC time, disabled operation flags, and omit-self result digest.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/provider-readiness-observation.schema.json`; implemented_by -> `modules/secure-identity-access-governance/internal/provider/readiness.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/provider_readiness.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: provider readiness history views
- notes: `readiness_proven_operations_disabled` is evidence, never permission or provider selection authority.
- status: canonical

#### SSIAG Authorization Request Schema
- path: `knowledge/ssiag/schemas/v1/authorization-request.schema.json`
- title: SSIAG Authorization Request v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical bounded exact authorization-request truth
- owner: SSIAG knowledge maintainer
- scope: Closes request/correlation IDs, exact operation/resource/audience/scope, and fresh UTC request/expiry intent without a caller-supplied subject or class.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/MANIFEST.md`; implemented_by -> `modules/secure-identity-access-governance/internal/policy/policy.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: SSIAG, qxctl, validator, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: Kernel peer authentication supplies subject identity outside this payload.
- status: canonical

#### SSIAG Capability Schema
- path: `knowledge/ssiag/schemas/v1/capability.schema.json`
- title: SSIAG Non-Transferable Capability v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical safe decision-evidence binding truth
- owner: SSIAG knowledge maintainer
- scope: Closes subject, TOPS, exact target, authority basis, grant, request/correlation IDs, issue/expiry, policy/configuration digests, binding digest, non-transferability, and disabled canonical apply.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/MANIFEST.md`; emitted_by -> `modules/secure-identity-access-governance/internal/policy/policy.go`; validated_by -> `tools/qxctl/cmd/qxctl/main.go`; validated_by -> `modules/knowledge-session-coordinator/src/authority_session.cpp`
- consumers: SSIAG, qxctl, coordinator, validator, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: It is not a generic token, secret, reusable bearer authority, or canonical apply permission.
- status: canonical

#### SSIAG Authorization Decision Schema
- path: `knowledge/ssiag/schemas/v1/authorization-decision.schema.json`
- title: SSIAG Authorization Decision v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical audited allow/deny decision truth
- owner: SSIAG knowledge maintainer
- scope: Closes exact subject/target, effect/reason, optional allow capability, policy/configuration digests, decision time/expiry, caller-class neutrality, and disabled canonical apply.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/capability.schema.json`; emitted_by -> `modules/secure-identity-access-governance/internal/policy/policy.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/client.go`; consumed_by -> `modules/knowledge-session-coordinator/src/authority_session.cpp`
- consumers: SSIAG, qxctl, coordinator, validator, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: SSIAG releases the decision only after its safe STAV policy-decision append commits.
- status: canonical

#### SSIAG Lifecycle Grant Plan Schema
- path: `knowledge/ssiag/schemas/v1/lifecycle-grant-plan.schema.json`
- title: SSIAG Lifecycle Grant Plan v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic proposal-only lifecycle grant input truth
- owner: SSIAG knowledge maintainer
- scope: Closes one exact caller-neutral subject, authority basis, TOPS/profile resource, domain-separated and separately permissioned per-TOPS profile-catalog read resource, audience, scope, complete lifecycle operation set, and disabled apply boundary.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/MANIFEST.md`; generated_by -> `tools/qxctl/cmd/qxctl/main.go`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: SSIAG administrators, qxctl, validators, conformance tests, reviewers
- deferred_projections: rendered grant-install guidance
- notes: Generation does not edit SSIAG configuration, grant authority, or authorize lifecycle execution.
- status: canonical

#### SSIAG Authorization Policy Schema
- path: `knowledge/ssiag/schemas/v1/authorization-policy.schema.json`
- title: SSIAG Local Authorization Policy v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deny-by-default operational policy truth
- owner: SSIAG knowledge maintainer
- scope: Closes bounded capability lifetime and exact caller-neutral grants for configured subjects.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/MANIFEST.md`; implemented_by -> `modules/secure-identity-access-governance/internal/policyadmin/manager.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: It contains policy but no credentials, proofs, provider payloads, or caller class.
- status: canonical

#### SSIAG Policy Proposal Request Schema
- path: `knowledge/ssiag/schemas/v1/policy-proposal-request.schema.json`
- title: SSIAG Policy Proposal Request v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical subject-free local policy intent truth
- owner: SSIAG knowledge maintainer
- scope: Closes operation/request/correlation IDs, authority basis, expected digest, replace/reset intent, desired policy, and UTC lifetime.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/authorization-policy.schema.json`; consumed_by -> `modules/secure-identity-access-governance/internal/policyadmin/manager.go`; emitted_by -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: Subject identity is intentionally absent and is derived by SSIAG from kernel peer evidence.
- status: canonical

#### SSIAG Policy Proposal Schema
- path: `knowledge/ssiag/schemas/v1/policy-proposal.schema.json`
- title: SSIAG Policy Proposal v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical subject-bound digest proposal truth
- owner: SSIAG knowledge maintainer
- scope: Binds target-host authority, TOPS, config/current/desired digests, complete intent, UTC lifetime, and proposal digest.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/policy-proposal-request.schema.json`; emitted_by -> `modules/secure-identity-access-governance/internal/policyadmin/manager.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: Always reports caller-class unused, canonical false, and applied false.
- status: canonical

#### SSIAG Policy Apply Request Schema
- path: `knowledge/ssiag/schemas/v1/policy-apply-request.schema.json`
- title: SSIAG Policy Apply Request v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical exact-proposal replay truth
- owner: SSIAG knowledge maintainer
- scope: Carries one complete digest-bound proposal to the local audit-before-commit apply circuit.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/policy-proposal.schema.json`; implemented_by -> `modules/secure-identity-access-governance/internal/server/server.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: The request grants no authority by possession; SSIAG re-authenticates and revalidates it.
- status: canonical

#### SSIAG Policy Recovery Request Schema
- path: `knowledge/ssiag/schemas/v1/policy-recovery-request.schema.json`
- title: SSIAG Policy Recovery Request v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical explicit attempt-recovery truth
- owner: SSIAG knowledge maintainer
- scope: Requires an exact operation and either expected attempt digest or explicit discovery.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/MANIFEST.md`; implemented_by -> `modules/secure-identity-access-governance/internal/policyadmin/manager.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: It rolls uniquely linked evidence forward and does not bypass STAV.
- status: canonical

#### SSIAG Policy Result Schema
- path: `knowledge/ssiag/schemas/v1/policy-result.schema.json`
- title: SSIAG Policy Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical metadata-only policy operation result truth
- owner: SSIAG knowledge maintainer
- scope: Reports source, generation, policy/state/attempt digests, recovery need, change/recovery state, and UTC observation.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/MANIFEST.md`; emitted_by -> `modules/secure-identity-access-governance/internal/policyadmin/manager.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: Contains no policy body and always reports caller-class unused and canonical false.
- status: canonical

#### SSIAG Policy State Schema
- path: `knowledge/ssiag/schemas/v1/policy-state.schema.json`
- title: SSIAG Protected Policy State v1
- surface_type: JSON Schema Draft 2020-12 persistence contract
- truth_role: canonical protected local generation-state truth
- owner: SSIAG knowledge maintainer
- scope: Closes config/overlay source, prior/current/config/state digests, monotonic generation, policy presence, and UTC commit time.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/authorization-policy.schema.json`; persisted_by -> `modules/secure-identity-access-governance/internal/policyadmin/storage_unix.go`
- consumers: SSIAG, validators, recovery logic, reviewers
- deferred_projections: safe metadata status
- notes: Operational per-TOPS state; never repository source truth or canonical knowledge apply.
- status: canonical

#### SSIAG Policy Attempt Schema
- path: `knowledge/ssiag/schemas/v1/policy-attempt.schema.json`
- title: SSIAG Durable Policy Attempt v1
- surface_type: JSON Schema Draft 2020-12 persistence contract
- truth_role: canonical crash-recovery stage truth
- owner: SSIAG knowledge maintainer
- scope: Closes prepared/audited stages, proposal, preparation/audit time, committed receipt digest, and attempt digest.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/policy-proposal.schema.json`; persisted_by -> `modules/secure-identity-access-governance/internal/policyadmin/storage_unix.go`
- consumers: SSIAG, validators, recovery logic, reviewers
- deferred_projections: safe recovery status
- notes: A pending attempt blocks competing mutation; recovery is explicit and audit-required.
- status: canonical

#### SSIAG Provider Executable Trust Schema
- path: `knowledge/ssiag/schemas/v1/provider-executable-trust.schema.json`
- title: SSIAG Provider Executable Trust v1
- surface_type: JSON Schema Draft 2020-12 protected-state contract
- truth_role: canonical exact adapter and foundation executable provenance truth
- owner: SSIAG knowledge maintainer
- scope: Binds one TOPS provider declaration to exact package, executable, ownership, foundation, signing, disabled-operation, and self-digest evidence.
- relationships: depends_on -> `knowledge/ssiag/SPEC.md`; implemented_by -> `modules/secure-identity-access-governance/internal/provider/trust.go`
- consumers: SSIAG, qxctl, provider installers, validators, security reviewers
- deferred_projections: safe provider-trust status
- notes: A declaration cannot prove its own signing identity or authorize provider operations.
- status: canonical

#### SSIAG Provider Control Request Schema
- path: `knowledge/ssiag/schemas/v1/provider-control-request.schema.json`
- title: SSIAG Provider Control Request v1
- surface_type: JSON Schema Draft 2020-12 process-message contract
- truth_role: canonical secret-free provider metadata request truth
- owner: SSIAG knowledge maintainer
- scope: Binds exact request, correlation, TOPS, provider, adapter, deadline, foundation, disabled-operation, and request-digest evidence.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/provider-executable-trust.schema.json`; emitted_by -> `modules/secure-identity-access-governance/internal/provider/trust.go`; consumed_by -> `modules/ssiag-provider-macos-keychain/Sources/SymphonySSIAGMacOSKeychain/main.swift`
- consumers: SSIAG, provider adapters, validators, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: It contains no credential, proof, assertion, token, provider payload, or secret value.
- status: canonical

#### SSIAG Provider Handshake Schema
- path: `knowledge/ssiag/schemas/v1/provider-handshake.schema.json`
- title: SSIAG Provider Handshake v1
- surface_type: JSON Schema Draft 2020-12 process-message contract
- truth_role: canonical mutually verified metadata capability truth
- owner: SSIAG knowledge maintainer
- scope: Closes exact protocol identity, platform, limits, safe operations, foundation trust, disabled operational flags, and handshake digest.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/provider-control-request.schema.json`; emitted_by -> `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/Protocol.swift`; consumed_by -> `modules/secure-identity-access-governance/internal/provider/trust.go`
- consumers: SSIAG, provider adapters, qxctl, validators, conformance tests, reviewers
- deferred_projections: provider compatibility evidence
- notes: Metadata capability is not operational readiness or permission.
- status: canonical

#### SSIAG Provider Control Response Schema
- path: `knowledge/ssiag/schemas/v1/provider-control-response.schema.json`
- title: SSIAG Provider Control Response v1
- surface_type: JSON Schema Draft 2020-12 process-message contract
- truth_role: canonical bounded provider metadata response truth
- owner: SSIAG knowledge maintainer
- scope: Binds one response to its request, exact handshake or safe error, disabled-operation flags, completion time, and response digest.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/provider-handshake.schema.json`; emitted_by -> `modules/ssiag-provider-macos-keychain/Sources/SymphonySSIAGMacOSKeychain/main.swift`; consumed_by -> `modules/secure-identity-access-governance/internal/provider/launcher.go`
- consumers: SSIAG, provider adapters, validators, conformance tests, reviewers
- deferred_projections: rendered protocol documentation
- notes: Native error details and secret material are explicitly excluded.
- status: canonical

#### SSIAG Provider One-Shot Channel Schema
- path: `knowledge/ssiag/schemas/v1/provider-one-shot-channel.schema.json`
- title: SSIAG Provider One-Shot Channel v1
- surface_type: JSON Schema Draft 2020-12 channel contract
- truth_role: canonical one-process one-request one-response and disabled-secret-channel truth
- owner: SSIAG knowledge maintainer
- scope: Defines bounded standard I/O metadata framing and the synthetic not-applicable secret descriptor retained for future compatibility.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/provider-control-response.schema.json`; implemented_by -> `modules/secure-identity-access-governance/internal/provider/launcher.go`
- consumers: SSIAG, provider adapters, validators, security reviewers
- deferred_projections: future separately ratified secret-channel contract
- notes: The Phase 9 secret descriptor has descriptor -1, zero capacity, and cannot carry bytes.
- status: canonical

#### SSIAG Provider Trust Verification Request Schema
- path: `knowledge/ssiag/schemas/v1/provider-trust-verification-request.schema.json`
- title: SSIAG Provider Trust Verification Request v1
- surface_type: JSON Schema Draft 2020-12 administrative request contract
- truth_role: canonical caller-neutral fresh provider verification intent truth
- owner: SSIAG knowledge maintainer
- scope: Closes request and correlation UUIDs plus host-owner or granted-permission authority basis without caller-supplied subject identity.
- relationships: depends_on -> `knowledge/ssiag/SPEC.md`; emitted_by -> `tools/qxctl/internal/ssiagclient/client.go`; consumed_by -> `modules/secure-identity-access-governance/internal/server/server.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: rendered administrative documentation
- notes: Kernel peer evidence supplies the subject and SSIAG independently decides permission.
- status: canonical

#### SSIAG Provider Trust Result Schema
- path: `knowledge/ssiag/schemas/v1/provider-trust-result.schema.json`
- title: SSIAG Provider Trust Result v1
- surface_type: JSON Schema Draft 2020-12 administrative result contract
- truth_role: canonical safe provider declaration and fresh-verification evidence truth
- owner: SSIAG knowledge maintainer
- scope: Reports exact state, mode, adapter evidence, deterministic checks, mutual trust, disabled-operation flags, UTC observation, and result digest.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/provider-control-response.schema.json`; emitted_by -> `modules/secure-identity-access-governance/internal/provider/trust.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: provider diagnostic and compatibility views
- notes: Always read-only, noncanonical, caller-class neutral, and free of credential material.
- status: canonical

#### SSIAG Provider Installation Inventory Schema
- path: `knowledge/ssiag/schemas/v1/provider-installation-inventory.schema.json`
- title: SSIAG Exact Provider Installation Inventory v1
- surface_type: JSON Schema Draft 2020-12 observation contract
- truth_role: canonical bounded exact-installation compatibility inventory truth
- owner: SSIAG knowledge maintainer
- scope: Closes sorted opaque installation identities, adapter/foundation receipt and executable digests, compatibility state, disabled operational flags, and inventory digest.
- relationships: depends_on -> `knowledge/ssiag/PROVIDER-LIFECYCLE.md`; emitted_by -> `modules/secure-identity-access-governance/internal/provider/binding.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/provider_binding.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: provider installation compatibility view
- notes: Inventory is observation only and never selects a newest version.
- status: canonical

#### SSIAG Provider Binding Status Schema
- path: `knowledge/ssiag/schemas/v1/provider-binding-status.schema.json`
- title: SSIAG Provider Binding Status v1
- surface_type: JSON Schema Draft 2020-12 observation contract
- truth_role: canonical safe protected-binding and pending-attempt status truth
- owner: SSIAG knowledge maintainer
- scope: Closes current/previous opaque installation IDs, generation, state and attempt digests, recovery state, disabled operational flags, and result digest.
- relationships: depends_on -> `knowledge/ssiag/PROVIDER-LIFECYCLE.md`; emitted_by -> `modules/secure-identity-access-governance/internal/provider/binding.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/provider_binding.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: provider binding status view
- notes: Attempt state explicitly includes none, prepared, candidate-verified, audited, and committed.
- status: canonical

#### SSIAG Provider Binding Plan Request Schema
- path: `knowledge/ssiag/schemas/v1/provider-binding-plan-request.schema.json`
- title: SSIAG Provider Binding Plan Request v1
- surface_type: JSON Schema Draft 2020-12 administrative request contract
- truth_role: canonical caller-neutral desired-binding intent truth
- owner: SSIAG knowledge maintainer
- scope: Closes exactly installation ID, expected state digest, and one bounded administrative reason without a caller-supplied subject or envelope.
- relationships: depends_on -> `knowledge/ssiag/PROVIDER-LIFECYCLE.md`; emitted_by -> `tools/qxctl/internal/ssiagclient/provider_binding.go`; consumed_by -> `modules/secure-identity-access-governance/internal/provider/binding.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: binding-plan forms
- notes: Literal `not_applicable` requests an unbound-preserved target; `absent` is the bootstrap CAS sentinel.
- status: canonical

#### SSIAG Provider Binding Plan Schema
- path: `knowledge/ssiag/schemas/v1/provider-binding-plan.schema.json`
- title: SSIAG Provider Binding Plan v1
- surface_type: JSON Schema Draft 2020-12 administrative plan contract
- truth_role: canonical exact evidence-bound dependency-plan truth
- owner: SSIAG knowledge maintainer
- scope: Closes desired/current/inventory digests, dependency-ordered forward or reverse actions, applicability, expiry, disabled operational flags, and plan digest.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/provider-binding-plan-request.schema.json`; emitted_by -> `modules/secure-identity-access-governance/internal/provider/binding.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/provider_binding.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: plan graph view
- notes: Direction follows exact compatible evidence rather than timestamps or semantic-newest order.
- status: canonical

#### SSIAG Provider Binding Apply Request Schema
- path: `knowledge/ssiag/schemas/v1/provider-binding-apply-request.schema.json`
- title: SSIAG Provider Binding Apply Request v1
- surface_type: JSON Schema Draft 2020-12 administrative request contract
- truth_role: canonical digest-only provider-binding apply intent truth
- owner: SSIAG knowledge maintainer
- scope: Closes exactly the previously issued plan digest and expected protected-state digest.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/provider-binding-plan.schema.json`; emitted_by -> `tools/qxctl/internal/ssiagclient/provider_binding.go`; consumed_by -> `modules/secure-identity-access-governance/internal/provider/binding.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: none
- notes: Apply cannot manufacture, widen, or silently refresh a stale plan.
- status: canonical

#### SSIAG Provider Binding Recovery Request Schema
- path: `knowledge/ssiag/schemas/v1/provider-binding-recovery-request.schema.json`
- title: SSIAG Provider Binding Recovery Request v1
- surface_type: JSON Schema Draft 2020-12 administrative request contract
- truth_role: canonical expected-state-bound recovery intent truth
- owner: SSIAG knowledge maintainer
- scope: Closes exactly the expected protected-state digest and one bounded administrative reason.
- relationships: depends_on -> `knowledge/ssiag/PROVIDER-LIFECYCLE.md`; emitted_by -> `tools/qxctl/internal/ssiagclient/provider_binding.go`; consumed_by -> `modules/secure-identity-access-governance/internal/provider/binding.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: recovery guidance
- notes: Recovery advances only one uniquely linked evidence-supported stage.
- status: canonical

#### SSIAG Provider Binding Result Schema
- path: `knowledge/ssiag/schemas/v1/provider-binding-result.schema.json`
- title: SSIAG Provider Binding Operation Result v1
- surface_type: JSON Schema Draft 2020-12 administrative result contract
- truth_role: canonical safe provider-binding apply/status/recovery result truth
- owner: SSIAG knowledge maintainer
- scope: Closes operation identity, current/previous binding, state/attempt/receipt digests, recovery disposition, disabled operational flags, and result digest.
- relationships: depends_on -> `knowledge/ssiag/PROVIDER-LIFECYCLE.md`; emitted_by -> `modules/secure-identity-access-governance/internal/provider/binding.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/provider_binding.go`
- consumers: SSIAG, qxctl, validators, conformance tests, reviewers
- deferred_projections: binding operation history view
- notes: Results contain safe metadata only and remain noncanonical.
- status: canonical

#### SSIAG Provider Binding State Schema
- path: `knowledge/ssiag/schemas/v1/provider-binding-state.schema.json`
- title: SSIAG Protected Provider Binding State v1
- surface_type: JSON Schema Draft 2020-12 operational state contract
- truth_role: canonical protected current/previous exact-binding persistence truth
- owner: SSIAG knowledge maintainer
- scope: Closes generation, bound/unbound state, current/previous installation IDs, prior-state digest, timestamp, disabled operational flags, and state digest.
- relationships: depends_on -> `knowledge/ssiag/PROVIDER-LIFECYCLE.md`; persisted_by -> `modules/secure-identity-access-governance/internal/provider/binding_storage_unix.go`
- consumers: SSIAG binding manager, recovery, validators, conformance tests, reviewers
- deferred_projections: binding history view
- notes: The state is separate from the Phase 9 single-declaration compatibility file so older readers fail closed.
- status: canonical

#### SSIAG Provider Binding Attempt Schema
- path: `knowledge/ssiag/schemas/v1/provider-binding-attempt.schema.json`
- title: SSIAG Durable Provider Binding Attempt v1
- surface_type: JSON Schema Draft 2020-12 operational attempt contract
- truth_role: canonical crash-recovery stage and target-state persistence truth
- owner: SSIAG knowledge maintainer
- scope: Closes the exact plan, target state, receipt, stage-specific UTC timestamps, disabled operational flags, and attempt digest.
- relationships: depends_on -> `knowledge/ssiag/schemas/v1/provider-binding-plan.schema.json`; depends_on -> `knowledge/ssiag/schemas/v1/provider-binding-state.schema.json`; persisted_by -> `modules/secure-identity-access-governance/internal/provider/binding_storage_unix.go`
- consumers: SSIAG binding manager, recovery, validators, conformance tests, reviewers
- deferred_projections: recovery-stage diagnostics
- notes: Conditionals require explicit `prepared -> candidate_verified -> audited -> committed` timestamp and receipt presence.
- status: canonical

### STAV Canonical Knowledge Vector

#### STAV INTENT.md
- path: `knowledge/stav/INTENT.md`
- title: Symphony TOPS Audit Vector Intent
- surface_type: knowledge-vector intent
- truth_role: canonical audit-vector purpose and privacy posture
- owner: STAV knowledge maintainer
- scope: Separates protocol truth from per-TOPS operational ledgers and establishes tamper-evident intent.
- relationships: declares -> `knowledge/stav/MANIFEST.md`; depends_on -> `knowledge/ssiag/INTENT.md`
- consumers: humans, reviewers, SSIAG, node-troll, qxctl, agentic tools
- deferred_projections: redacted query projection
- notes: V1 is tamper-evident, not non-repudiable.
- status: canonical

#### STAV MANIFEST.md
- path: `knowledge/stav/MANIFEST.md`
- title: Symphony TOPS Audit Vector Manifest
- surface_type: knowledge-vector manifest
- truth_role: canonical STAV authority, writer, storage, and projection boundaries
- owner: STAV knowledge maintainer
- scope: Declares schema authority, append authority, producers, qxctl, callers, and operational storage.
- relationships: depends_on -> `knowledge/stav/INTENT.md`; declares -> `knowledge/stav/SPEC.md`; governs -> `modules/stav-append-authority/MANIFEST.md`
- consumers: humans, reviewers, append-authority and producer implementers, SKVI
- deferred_projections: JSONL, DuckDB, HDF5, redacted graph projections
- notes: No operational ledger belongs in the repository.
- status: canonical

#### STAV SKILL.md
- path: `knowledge/stav/SKILL.md`
- title: Symphony TOPS Audit Vector Skill
- surface_type: knowledge-vector skill guidance
- truth_role: safe agent and implementation procedure for audit events
- owner: STAV knowledge maintainer
- scope: Defines allowed queries/proposals, prohibited direct writes, review procedure, and stop conditions.
- relationships: depends_on -> `knowledge/stav/SPEC.md`; governs -> `tools/qxctl/cmd/qxctl/main.go`; governs -> `modules/stav-append-authority/SKILL.md`
- consumers: humans, reviewers, operators, agentic tools
- deferred_projections: none
- notes: No supported caller interface edits, repairs, reorders, or directly appends ledger files.
- status: canonical

#### STAV SPEC.md
- path: `knowledge/stav/SPEC.md`
- title: Symphony TOPS Audit Vector Specification
- surface_type: knowledge-vector specification
- truth_role: canonical ten-group envelope, append protocol, integrity, and redaction truth
- owner: STAV knowledge maintainer
- scope: Defines per-TOPS sequence isolation, field presence, serialized append authority, SSIAG outcome classes, and exclusions.
- relationships: depends_on -> `knowledge/stav/MANIFEST.md`; governs -> `modules/stav-append-authority/SPEC.md`; interprets -> `knowledge/ssiag/SPEC.md`
- consumers: SSIAG, node-troll, qxctl, append-authority implementers, reviewers, agents
- deferred_projections: signed checkpoints, verifier evidence, query stores
- notes: Canonical semantic/operational schemas, strict JCS, durability, authenticated listener, read projection, and native supervision are implemented.
- status: canonical

#### STAV v1 Schemas
- path: `knowledge/stav/schemas/v1/MANIFEST.md`
- title: STAV v1 Canonical JSON Schemas
- surface_type: JSON Schema Draft 2020-12 contract directory
- truth_role: canonical STAV semantic and read-message structure truth
- owner: STAV knowledge maintainer
- scope: Defines all eleven ratified v1 structures: common values, candidate, event, receipt, query, query-page, verification, append-authority configuration and status, and bounded local request and response messages.
- relationships: depends_on -> `knowledge/stav/SPEC.md`; governs -> `libraries/stav-protocol-go/`
- consumers: protocol-kernel, append-authority, qxctl, producer implementers, reviewers, symphony-validator and future validator extensions
- deferred_projections: generated documentation and conformance reports
- notes: Signed checkpoints and remote transport remain deferred; the implemented configuration, status, and local request/response schemas are canonical here.
- status: canonical

#### STAV v1 Registries
- path: `knowledge/stav/registries/v1/base.md`
- title: STAV v1 Closed Registries
- surface_type: canonical protocol registry directory
- truth_role: generic outcome, redaction, and protocol-reason truth
- owner: STAV knowledge maintainer
- scope: Defines closed generic values and reserves producer-specific assignments for producer integration.
- relationships: depends_on -> `knowledge/stav/SPEC.md`; governs -> `libraries/stav-protocol-go/`
- consumers: protocol-kernel, producer integrators, reviewers, agentic tools
- deferred_projections: machine-readable registry evidence
- notes: SSIAG event-class assignments are not guessed here.
- status: canonical

#### STAV v1 Fixtures
- path: `knowledge/stav/fixtures/v1/MANIFEST.md`
- title: STAV v1 Conformance Fixtures
- surface_type: valid and invalid protocol corpus
- truth_role: canonical interoperability and rejection evidence
- owner: STAV knowledge maintainer
- scope: Exercises canonical documents, duplicate/null/number/unknown-field rejection, and stable digest inputs.
- relationships: depends_on -> `knowledge/stav/schemas/v1/`; governs -> `libraries/stav-protocol-go/GO_1_27_MIGRATION.md`
- consumers: protocol-kernel tests, toolchain migration, symphony-validator, future validator extensions, and language implementations
- deferred_projections: cross-language conformance reports
- notes: Invalid UTF-8 and partial-input cases are constructed in kernel tests where text files cannot safely represent them.
- status: canonical

### SSIAG macOS Keychain Provider Module

#### macOS Provider INTENT.md
- path: `modules/ssiag-provider-macos-keychain/INTENT.md`
- title: SSIAG macOS Keychain Provider Intent
- surface_type: provider module intent
- truth_role: independent Swift adapter purpose and process boundary
- owner: SSIAG macOS provider maintainer
- scope: Defines optional Apple Keychain boundary and metadata-only scaffold status.
- relationships: depends_on -> `knowledge/ssiag/SPEC.md`; declares -> `modules/ssiag-provider-macos-keychain/MANIFEST.md`
- consumers: humans, reviewers, provider implementers, agentic tools
- deferred_projections: platform integration evidence
- notes: No operational Keychain access is enabled.
- status: canonical

#### macOS Provider MANIFEST.md
- path: `modules/ssiag-provider-macos-keychain/MANIFEST.md`
- title: SSIAG macOS Keychain Provider Manifest
- surface_type: provider module contract truth
- truth_role: language, binary, protocol, capability, and lifecycle declaration
- owner: SSIAG macOS provider maintainer
- scope: Declares Swift executable identity, metadata IPC, independent installability, and prohibited claims.
- relationships: depends_on -> `modules/ssiag-provider-macos-keychain/INTENT.md`; declares -> `modules/ssiag-provider-macos-keychain/SPEC.md`
- consumers: humans, reviewers, SSIAG implementers, symphony-validator and future validator extensions
- deferred_projections: compatibility evidence
- notes: Native Swift code remains outside the Go-only foundation.
- status: canonical

#### macOS Provider INSTALL.md
- path: `modules/ssiag-provider-macos-keychain/INSTALL.md`
- title: SSIAG macOS Keychain Provider Installation
- surface_type: provider module installation guidance
- truth_role: independent build, test, install, upgrade, verify, and uninstall procedure
- owner: SSIAG macOS provider maintainer
- scope: Defines macOS prerequisites and digest-safe user/system lifecycle.
- relationships: depends_on -> `modules/ssiag-provider-macos-keychain/MANIFEST.md`
- consumers: TOPS operators, maintainers, reviewers, agentic tools
- deferred_projections: release runbook
- notes: Uninstall never deletes Keychain items or TOPS state.
- status: canonical

#### macOS Provider SKILL.md
- path: `modules/ssiag-provider-macos-keychain/SKILL.md`
- title: SSIAG macOS Keychain Provider Skill
- surface_type: provider module skill guidance
- truth_role: safe build, test, install, and operational-gate procedure
- owner: SSIAG macOS provider maintainer
- scope: Defines the caller-neutral capability boundary and ratification required before Apple Security access.
- relationships: depends_on -> `modules/ssiag-provider-macos-keychain/SPEC.md`; interprets -> `knowledge/stav/SPEC.md`
- consumers: humans, maintainers, security reviewers, agentic tools
- deferred_projections: provider security review checklist
- notes: The `security` CLI may not become a hidden fallback.
- status: canonical

#### macOS Provider SPEC.md
- path: `modules/ssiag-provider-macos-keychain/SPEC.md`
- title: SSIAG macOS Keychain Provider Specification
- surface_type: provider module specification
- truth_role: normative metadata IPC and independent lifecycle behavior
- owner: SSIAG macOS provider maintainer
- scope: Defines bounded JSON-lines metadata operations, descriptor truth, installation, and future operational gate.
- relationships: depends_on -> `knowledge/ssiag/SPEC.md`; governs -> `modules/ssiag-provider-macos-keychain/README.md`
- consumers: implementers, reviewers, SSIAG foundation, symphony-validator and future validator extensions
- deferred_projections: operational provider protocol conformance
- notes: Operational access must report false until a separate review enables it.
- status: canonical

#### macOS Provider README.md
- path: `modules/ssiag-provider-macos-keychain/README.md`
- title: SSIAG macOS Keychain Provider README
- surface_type: provider module orientation
- truth_role: concise contributor entrypoint
- owner: SSIAG macOS provider maintainer
- scope: Directs readers to contracts and states metadata-only status.
- relationships: derives_from -> `modules/ssiag-provider-macos-keychain/MANIFEST.md`
- consumers: humans, contributors, agentic tools
- deferred_projections: SODV-governed public module page
- notes: Repository source truth; not a publication pipeline.
- status: canonical

### Symphony Validator Tool Contract and Implementation Boundary
##### INTENT.md
- path: `tools/symphony-validator/INTENT.md`
- title: Validator Intent
- surface_type: tool intent
- truth_role: intent and purpose for symphony-validator
- owner: validator maintainer
- scope: Defines the implemented deterministic, explainable, non-autonomous C++26 parser/checker boundary. Current output is line-oriented evidence plus a summary and exit status; structured projectors and integration surfaces remain deferred.
- relationships:
  - declares -> `tools/symphony-validator/MANIFEST.md`
  - declares -> `tools/symphony-validator/SPEC.md`
- consumers: humans, validator maintainers, local preflight automation
- deferred_projections: JSON/Markdown evidence, qxctl mediation, CI/PR-gate integration
- notes: Implementation authority is bounded by the validator Contract Quad and CMake build contract.
- status: canonical

##### MANIFEST.md
- path: `tools/symphony-validator/MANIFEST.md`
- title: Validator Manifest
- surface_type: tool contract truth
- truth_role: declared contract truth for symphony-validator
- owner: validator maintainer
- scope: Contractual definitions.
- consumers: humans, validator maintainers, future qxctl integration
- relationships: governs -> `tools/symphony-validator/CMakeLists.txt`
- deferred_projections: Markdown caller-ingestion projection, JSON/JSONL portable evidence, installation packaging, qxctl mediation, and CI/PR-gate integration. The implemented bounded active-Markdown caller-authority check does not authorize runtime/source/AST analysis, remediation, or general semantic analysis.
- notes: The checked-in parser/checker is authorized; runtime residency is not.
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
- deferred_projections: portable installation packaging
- notes: Documents the current local C++26 build and direct invocation path.
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
- deferred_projections: qxctl/CI invocation and structured evidence projectors
- notes: Current use is direct, deterministic, and read-only.
- status: canonical

##### SPEC.md
- path: `tools/symphony-validator/SPEC.md`
- title: Validator Specification
- surface_type: tool specification
- truth_role: normative parser/checker behavior and authority boundary
- owner: validator maintainer
- scope: Deterministic validation rules, including caller-authority regression checking (exit code 21).
- consumers: humans, validator implementation, reviewers
- relationships: governs -> `tools/symphony-validator/src/`; constrains -> future projectors and integrations
- deferred_projections: JSON/Markdown, graph, analytical, and qxctl-readable projections
- notes: Authorizes the checked-in deterministic parser/checker but not projectors, qxctl/CI integration, publication, or remediation.
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

##### Canonical JSON Artifact Allowlist Implementation
- path: `tools/symphony-validator/src/artifacts.cpp`
- title: symphony-validator Exact Canonical JSON Artifact Allowlist
- surface_type: C++26 validator implementation surface
- truth_role: exact path-scoped canonical JSON authorization evidence implementation truth
- owner: validator maintainer
- scope: Recognizes only Architect-ratified JSON schema/fixture paths and rejects prefix- or extension-wide projection authorization.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; governed_by -> `knowledge/SPEC.md`; governed_by -> vector Contract Quads
- consumers: symphony-validator, validator tests, reviewers
- deferred_projections: structured artifact evidence
- notes: Adding a schema requires an explicit contract, path entry, count update, and regression verification.
- status: canonical

##### Runtime Contract Validator Implementation
- path: `tools/symphony-validator/src/runtime_contracts.cpp`
- title: symphony-validator Runtime Contract Checks
- surface_type: C++26 read-only validator implementation
- truth_role: installed-module and runtime-contract conformance evidence
- owner: validator maintainer
- scope: Checks bounded runtime contracts, including the Maestro receptor-presence module surfaces, without invoking engines or mutating operational state.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; checks -> `modules/maestro/MANIFEST.md`; checks -> `modules/maestro/SPEC.md`
- consumers: symphony-validator, repository reviewers, release gates
- deferred_projections: runtime remediation, package activation, receptor docking
- notes: Contract validation is evidence only and grants no lifecycle or canonical authority.
- status: canonical

##### SKVI Coverage Validator Implementation
- path: `tools/symphony-validator/src/skvi_coverage.cpp`
- title: symphony-validator SKVI Exact-Surface Coverage Checks
- surface_type: C++26 read-only validator implementation
- truth_role: canonical surface-index coverage and historical drift evidence
- owner: validator maintainer
- scope: Compares bounded repository surfaces and SCLV affected-surface references with exact SKVI path records, including Maestro module coverage.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; checks -> `knowledge/skvi/INDEX.md`; reads -> `knowledge/sclv/CHANGELOG.md`
- consumers: symphony-validator, SKVI maintainers, repository reviewers
- deferred_projections: automatic index mutation or warning remediation
- notes: Missing coverage is reported deterministically; the validator never creates or rewrites SKVI records.
- status: canonical

##### symphony-validator Smoke Regression Suite
- path: `tools/symphony-validator/tests/smoke.sh`
- title: symphony-validator Repository and Fixture Smoke Suite
- surface_type: shell conformance-test implementation
- truth_role: current-repository, exact-artifact-count, negative-fixture, and exit-behavior proof
- owner: validator maintainer
- scope: Executes deterministic valid/invalid repository matrices, exact canonical JSON counts, caller-authority checks, and validator build-integrity failures.
- relationships: verifies -> `tools/symphony-validator/SPEC.md`; executes -> `tools/symphony-validator/src/artifacts.cpp`
- consumers: validator maintainers, reviewers, release gates
- deferred_projections: CI/PR-gate integration
- notes: The suite validates behavior and never remediates repository state.
- status: canonical

### qxctl Tool Contract

#### qxctl INTENT.md
- path: `tools/qxctl/INTENT.md`
- title: qxctl Intent
- surface_type: tool intent
- truth_role: canonical administrative-spine purpose and authority boundary
- owner: qxctl maintainer
- scope: Defines Go/Cobra/Viper administration, local module clients, vector-engine grammar, and non-ownership of module/vector semantics.
- relationships: declares -> `tools/qxctl/MANIFEST.md`; depends_on -> `knowledge/SPEC.md`; interprets -> vector and module contracts
- consumers: administrators, implementers, reviewers, agentic tools
- deferred_projections: command reference and completion metadata
- notes: Implemented SSIAG/STAV, vector-engine, user-default binding, reconciliation, authenticated-session, report lifecycle, and explicit noncanonical lifecycle-apply commands are distinguished from reserved canonical proposal/apply and safeguard grammar.
- status: canonical

#### qxctl MANIFEST.md
- path: `tools/qxctl/MANIFEST.md`
- title: qxctl Manifest
- surface_type: tool manifest
- truth_role: command, dependency, installation, and non-authorization contract
- owner: qxctl maintainer
- scope: Enumerates operational commands, user-default engine bindings, bound reconciliation/session grammar, protected lifecycle profile/observation/report/boot/apply grammar, reserved canonical knowledge grammar, constrained dependencies, and lifecycle boundaries.
- relationships: depends_on -> `tools/qxctl/INTENT.md`; depends_on -> `knowledge/SPEC.md`; governs -> `tools/qxctl/cmd/qxctl/`
- consumers: qxctl implementers, module/vector maintainers, reviewers, agentic tools
- deferred_projections: command registry and module lifecycle evidence
- notes: Reserved commands must not be presented as implemented.
- status: canonical

#### qxctl INSTALL.md
- path: `tools/qxctl/INSTALL.md`
- title: qxctl Install
- surface_type: tool installation guidance
- truth_role: qxctl build and installation procedure
- owner: qxctl maintainer
- scope: Defines supported build/install invocation for the Go administrative spine.
- relationships: depends_on -> `tools/qxctl/MANIFEST.md`
- consumers: administrators, contributors, packaging maintainers
- deferred_projections: packaged installation runbook
- notes: qxctl installation does not install every independently managed module.
- status: canonical

#### qxctl SKILL.md
- path: `tools/qxctl/SKILL.md`
- title: qxctl Skill
- surface_type: tool skill guidance
- truth_role: safe caller command and verification procedure
- owner: qxctl maintainer
- scope: Guides caller-neutral administration, trust handling, proposal boundaries, sessions, and hot/warm isolation.
- relationships: depends_on -> `tools/qxctl/MANIFEST.md`; depends_on -> `knowledge/SKILL.md`
- consumers: administrators, reviewers, agentic tools
- deferred_projections: command procedure documentation
- notes: qxctl does not grant authority or directly mutate canonical vector files.
- status: canonical

#### qxctl README.md
- path: `tools/qxctl/README.md`
- title: qxctl README
- surface_type: tool orientation
- truth_role: implemented command and contributor overview
- owner: qxctl maintainer
- scope: Describes current qxctl use and operational integrations.
- relationships: depends_on -> `tools/qxctl/MANIFEST.md`
- consumers: users, contributors, reviewers, agentic tools
- deferred_projections: SODV-governed public command documentation
- notes: Implementation claims must remain synchronized with the manifest.
- status: canonical

#### qxctl Command Grammar Implementation
- path: `tools/qxctl/cmd/qxctl/main.go`
- title: qxctl Administrative Operation Implementation
- surface_type: administrative CLI implementation surface
- truth_role: local operation dispatch, process-client invocation, and presentation implementation truth
- owner: qxctl maintainers
- scope: Implements current repository, module, SSIAG, STAV, knowledge-engine binding/reconciliation/authenticated-session/session-transition/lifecycle, SKVI, SCLV including typed provider-evidence result validation, SACV, SODV, and shared SSFV administrative operation handlers.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; emits -> `knowledge/schemas/v1/session-transition-result.schema.json`; invokes -> `tools/qxctl/cmd/qxctl/lifecycle.go`; invokes -> `tools/qxctl/internal/knowledgeengine/client.go`; invokes -> `tools/qxctl/internal/knowledgebinding/registry.go`; invokes -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: qxctl executable, tests, maintainers, reviewers
- deferred_projections: generated CLI reference and operation evidence
- notes: Presentation does not own vector semantics or authorize mutation.
- status: canonical

#### qxctl Cobra Command Grammar
- path: `tools/qxctl/cmd/qxctl/commands.go`
- title: qxctl Cobra Command Grammar
- surface_type: administrative CLI implementation surface
- truth_role: implemented command tree, flag grammar, and failure routing
- owner: qxctl maintainers
- scope: Implements current repository, SSIAG, STAV, user-default engine binding, bound-coordinator reconciliation/authenticated-session and explicit session-transition, lifecycle profile/observation/report/boot/apply, exact-installation SKVI/SCLV/SACV/SODV/SSFV grammar, and the nested `sclv evidence local-git|airgap` adapter leaves without owning domain semantics.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; invokes -> `tools/qxctl/cmd/qxctl/lifecycle.go`; invokes -> `tools/qxctl/internal/knowledgeengine/client.go`; invokes -> `tools/qxctl/internal/knowledgebinding/registry.go`
- consumers: qxctl executable, compatibility tests, maintainers, reviewers
- deferred_projections: generated CLI reference documentation
- notes: Cobra grammar exposes reviewed noncanonical lifecycle apply but does not authorize arbitrary actions, live process activation, or canonical apply.
- status: canonical

#### qxctl Knowledge Engine Process Client
- path: `tools/qxctl/internal/knowledgeengine/client.go`
- title: qxctl Knowledge Engine Process Client
- surface_type: bounded Go process-client implementation
- truth_role: trusted receipt resolution, child-process bounds, and response verification implementation truth
- owner: qxctl maintainers
- scope: Resolves exact installed coordinator, SKVI, SCLV, SACV, SODV, and SSFV versions for binding and lifecycle evidence; requires exact receipt-v2 typed SCLV adapter entry points before invoking local-Git or air-gap evidence normalization; invokes the bound coordinator and five vector engines with an empty environment and hard deadline; and verifies response identity and digest.
- relationships: implements -> `knowledge/SPEC.md`; implements -> `knowledge/schemas/v1/reconciliation-command.schema.json`; implements -> `knowledge/schemas/v1/session-command.schema.json`; implements -> `knowledge/schemas/v1/provider-evidence.schema.json`; implements -> `knowledge/skvi/SPEC.md`; implements -> `knowledge/sclv/SPEC.md`; implements -> `knowledge/sacv/SPEC.md`; implements -> `knowledge/sodv/SPEC.md`; implements -> `knowledge/ssfv/SPEC.md`; called_by -> `tools/qxctl/cmd/qxctl/commands.go`
- consumers: qxctl knowledge-reconciliation/session/lifecycle/SKVI/SCLV/SACV/SODV/SSFV commands, tests, reviewers, future compatible vector clients
- deferred_projections: additional compatible vector clients
- notes: It does not install, alter receipt activation, dock, infer membership, grant permission, ratify, mutate canonical knowledge, or apply; coordinator journal mutation is governed separately.
- status: canonical

#### qxctl Lifecycle Operation Layer
- path: `tools/qxctl/cmd/qxctl/lifecycle.go`
- title: qxctl Cross-Vector Lifecycle Administration
- surface_type: administrative CLI implementation surface
- truth_role: protected profile, SSIAG authorization, observation, coordinator invocation, plan validation, and presentation implementation truth
- owner: qxctl maintainers
- scope: Implements claim-continuous profile list/show/set/remove, shared-root ownership status/reconcile/adopt/release, fixed-layout observe, dynamic planning, report-only boot/status/recovery invocation, shared lifecycle evidence validation, and presentation.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; implements -> `knowledge/LIFECYCLE.md`; invokes -> `tools/qxctl/internal/knowledgelifecycle/profile.go`; invokes -> `tools/qxctl/internal/knowledgelifecycle/observation.go`; invokes -> `tools/qxctl/internal/knowledgelifecycle/ownership.go`; invokes -> `tools/qxctl/internal/knowledgeengine/client.go`; invokes -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: qxctl executable, lifecycle tests, administrators, reviewers
- deferred_projections: host boot integration and generated lifecycle command documentation
- notes: Boot remains report-only; apply-compatible execution is implemented separately in lifecycle_apply.go.
- status: canonical

#### qxctl Lifecycle Apply Operation Layer
- path: `tools/qxctl/cmd/qxctl/lifecycle_apply.go`
- title: qxctl Apply-Compatible Lifecycle Administration
- surface_type: administrative CLI implementation surface
- truth_role: exact source/CAS binding, dynamic ready-set execution, per-phase authorization, re-observation, and result-validation implementation truth
- owner: qxctl maintainers
- scope: Implements apply, apply-status, and apply-recover by binding one report-journal source, preparing one coordinator action, reconciling shared-root claims, executing only a reviewed adapter, re-observing complete state, finalizing evidence, dynamically replanning, and explicitly closing convergence.
- relationships: implements -> `knowledge/LIFECYCLE.md`; implements -> `knowledge/schemas/v1/lifecycle-apply-command.schema.json`; implements -> `knowledge/schemas/v1/lifecycle-apply-result.schema.json`; invokes -> `tools/qxctl/internal/knowledgelifecycle/executor.go`; invokes -> `tools/qxctl/internal/knowledgeengine/client.go`; invokes -> `tools/qxctl/internal/ssiagclient/client.go`
- consumers: qxctl executable, lifecycle tests, administrators, reviewers
- deferred_projections: additional separately reviewed lifecycle action adapters
- notes: Callers cannot select action order directly; package download, receipt-v1 mutation, live service activation, in-place coordinator replacement, unconstrained/new-role rebinding, Maestro engine execution, and canonical apply are absent. Exact established-role binding, candidate-verified coordinator handoff, and Maestro presence docking are implemented through reviewed external adapters.
- status: canonical

#### qxctl Lifecycle Profile Model
- path: `tools/qxctl/internal/knowledgelifecycle/profile.go`
- title: qxctl Protected Lifecycle Profile Model
- surface_type: bounded Go lifecycle-state implementation
- truth_role: profile validation, normalization, generation, continuity, compare-and-swap, and digest implementation truth
- owner: qxctl maintainers
- scope: Implements canonical-schema-conformant profile input and protected noncanonical desired-profile state with semantic retry.
- relationships: implements -> `knowledge/schemas/v1/lifecycle-profile-input.schema.json`; implements -> `knowledge/schemas/v1/lifecycle-profile.schema.json`; emits -> `knowledge/schemas/v1/lifecycle-desired-state.schema.json`; called_by -> `tools/qxctl/cmd/qxctl/lifecycle.go`
- consumers: qxctl lifecycle commands, tests, administrators, reviewers
- deferred_projections: policy profiles and system-scope deployment service integration
- notes: qxctl generates generations and content digests; callers supply intent and exact expected state.
- status: canonical

#### qxctl Lifecycle Observation Model
- path: `tools/qxctl/internal/knowledgelifecycle/observation.go`
- title: qxctl Lifecycle Observation Model
- surface_type: bounded Go observation implementation
- truth_role: normalized platform, component, package, binding, stable-inventory, and document-digest implementation truth
- owner: qxctl maintainers
- scope: Builds exact disposable lifecycle-observation v1 evidence, consumes only the exact ownership compatibility fence, and separates stable inventory identity from collection-time document evidence.
- relationships: implements -> `knowledge/schemas/v1/lifecycle-observation.schema.json`; consumes -> `knowledge/schemas/v1/lifecycle-root-ownership-fence.schema.json`; called_by -> `tools/qxctl/cmd/qxctl/lifecycle.go`; populated_by -> `tools/qxctl/internal/knowledgelifecycle/scan_unix.go`
- consumers: qxctl lifecycle commands, coordinator planner, tests, reviewers
- deferred_projections: Maestro presence observation
- notes: Excluding observed_at from stable inventory prevents timestamp-only false transactions without weakening document evidence.
- status: canonical

#### qxctl Lifecycle Action Executor
- path: `tools/qxctl/internal/knowledgelifecycle/executor.go`
- title: qxctl Reviewed Lifecycle Action Executor
- surface_type: bounded Go lifecycle adapter dispatcher
- truth_role: deterministic action-to-adapter routing, staged-artifact, target-state, and execution-evidence implementation truth
- owner: qxctl maintainers
- scope: Validates the coordinator-selected action against desired/observed evidence, routes receipt-v2 install/claim-governed uninstall or protected select/deselect/activate/deactivate, reports idempotent outcomes, and rejects dock/undock and non-mutating planner actions.
- relationships: implements -> `knowledge/LIFECYCLE.md`; called_by -> `tools/qxctl/cmd/qxctl/lifecycle_apply.go`; invokes -> `tools/qxctl/internal/knowledgelifecycle/install_unix.go`; invokes -> `tools/qxctl/internal/knowledgelifecycle/ownership.go`; invokes -> `tools/qxctl/internal/knowledgelifecycle/runtime.go`
- consumers: qxctl lifecycle apply, tests, reviewers
- deferred_projections: live service/process, receipt-v1, coordinator-handoff, and Maestro adapters
- notes: It uses no network, shell, receipt entry point, implicit version choice, or arbitrary command.
- status: canonical

#### qxctl Unix Receipt-v2 Lifecycle Adapter
- path: `tools/qxctl/internal/knowledgelifecycle/install_unix.go`
- title: qxctl Unix Receipt-v2 Install and Uninstall Adapter
- surface_type: platform-specific secure package mutation implementation
- truth_role: no-follow staged copy, receipt-last commit, rollback-proof, owned removal, and retry implementation truth
- owner: qxctl maintainers
- scope: Installs exact receipt-v2 files from a separate trusted staged root only under an exact retained claim and uninstalls only after a separate staged rollback proof validates every remaining owned path and the root-local registry proves unanimous release; missing paths permit forward retry while conflicts fail closed.
- relationships: implements -> `knowledge/schemas/v2/install-receipt.schema.json`; implements -> `knowledge/schemas/v1/lifecycle-root-ownership.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; called_by -> `tools/qxctl/internal/knowledgelifecycle/executor.go`
- consumers: qxctl lifecycle apply, tests, packagers, reviewers
- deferred_projections: package-manager-specific staging adapters
- notes: Installation publishes the receipt last; uninstall removes it last and commits satisfied retiring claims under the same root lock; no download or entry-point execution occurs.
- status: canonical

#### qxctl Unsupported Native Receipt-v2 Lifecycle Adapter
- path: `tools/qxctl/internal/knowledgelifecycle/install_unsupported.go`
- title: qxctl Unsupported Native Lifecycle Package Adapter
- surface_type: platform fail-closed implementation
- truth_role: unsupported-native-operating-system package-mutation rejection truth
- owner: qxctl maintainers
- scope: Refuses generic receipt-v2 package mutation where the Linux/macOS no-follow contract is unavailable.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgelifecycle/executor.go`
- consumers: qxctl lifecycle apply, tests, reviewers
- deferred_projections: WSL and remote-node administration documentation
- notes: It introduces no native Windows lifecycle engine or weaker fallback.
- status: canonical

#### qxctl Lifecycle Runtime-State Model
- path: `tools/qxctl/internal/knowledgelifecycle/runtime.go`
- title: qxctl Protected Lifecycle Runtime-State Model
- surface_type: bounded Go lifecycle-state implementation
- truth_role: generic selection/activation validation, generation, compare-and-swap, idempotency, and digest implementation truth
- owner: qxctl maintainers
- scope: Implements exact per-component receipt selection plus administrative active/inactive state while holding docking to undocked and rejecting deselection of active components.
- relationships: implements -> `knowledge/schemas/v1/lifecycle-runtime-state.schema.json`; called_by -> `tools/qxctl/internal/knowledgelifecycle/executor.go`; persisted_by -> `tools/qxctl/internal/knowledgelifecycle/runtime_unix.go`
- consumers: qxctl lifecycle observation/apply, tests, reviewers
- deferred_projections: live process/service and Maestro presence state
- notes: Active is eligibility state, not evidence that code or a service ran.
- status: canonical

#### qxctl Unix Lifecycle Runtime State
- path: `tools/qxctl/internal/knowledgelifecycle/runtime_unix.go`
- title: qxctl Unix Protected Lifecycle Runtime State
- surface_type: platform-specific protected local-state implementation
- truth_role: per-TOPS no-follow lock, ownership, atomic replacement, and durability implementation truth
- owner: qxctl maintainers
- scope: Implements Linux/macOS protected runtime-state directories, nonblocking locking, mode-0600 state, same-directory atomic replacement, and file/directory durability.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgelifecycle/runtime.go`
- consumers: qxctl lifecycle apply/observation, tests, reviewers
- deferred_projections: supervised system-scope state administration
- notes: Runtime state is noncanonical and contains no credentials or arbitrary executable instructions.
- status: canonical

#### qxctl Unsupported Native Lifecycle Runtime State
- path: `tools/qxctl/internal/knowledgelifecycle/runtime_unsupported.go`
- title: qxctl Unsupported Native Lifecycle Runtime State
- surface_type: platform fail-closed implementation
- truth_role: unsupported-native-operating-system runtime-state rejection truth
- owner: qxctl maintainers
- scope: Refuses generic lifecycle runtime-state mutation where the Linux/macOS protected-state contract is unavailable.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgelifecycle/runtime.go`
- consumers: qxctl lifecycle commands, tests, reviewers
- deferred_projections: WSL and remote-node administration documentation
- notes: It does not introduce a native Windows lifecycle engine or weaker persistence fallback.
- status: canonical

#### qxctl Unix Lifecycle Receipt Scanner
- path: `tools/qxctl/internal/knowledgelifecycle/scan_unix.go`
- title: qxctl Unix Fixed-Layout Lifecycle Receipt Scanner
- surface_type: platform-specific secure observation implementation
- truth_role: no-follow receipt discovery, v1 adaptation, v2 content-addressed integrity, and unknown-preservation implementation truth
- owner: qxctl maintainers
- scope: Scans only configured Linux/macOS receipt layouts, executes nothing discovered, validates bounded package evidence, and preserves unsupported state.
- relationships: implements -> `knowledge/LIFECYCLE.md`; reads -> `knowledge/schemas/v1/install-receipt.schema.json`; reads -> `knowledge/schemas/v2/install-receipt.schema.json`; called_by -> `tools/qxctl/internal/knowledgelifecycle/observation.go`
- consumers: qxctl lifecycle observation/report, tests, administrators, reviewers
- deferred_projections: additional ratified receipt adapters
- notes: Unknown, invalid, unreadable, or ambiguous packages remain visible and are never silently removed.
- status: canonical

#### qxctl Unsupported Native Lifecycle Scanner
- path: `tools/qxctl/internal/knowledgelifecycle/scan_unsupported.go`
- title: qxctl Unsupported Native Lifecycle Scanner
- surface_type: platform fail-closed implementation
- truth_role: unsupported-native-operating-system rejection truth
- owner: qxctl maintainers
- scope: Refuses lifecycle receipt observation where the Linux/macOS no-follow scanner contract is unavailable.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgelifecycle/observation.go`
- consumers: qxctl lifecycle commands, tests, reviewers
- deferred_projections: WSL and remote-node administration documentation
- notes: It does not introduce a native Windows lifecycle engine or weaker traversal fallback.
- status: canonical

#### qxctl Unix Lifecycle Profile State
- path: `tools/qxctl/internal/knowledgelifecycle/state_unix.go`
- title: qxctl Unix Protected Lifecycle Profile State
- surface_type: platform-specific protected local-state implementation
- truth_role: per-TOPS no-follow lock, ownership, atomic replacement, removal, and durability implementation truth
- owner: qxctl maintainers
- scope: Implements Linux/macOS protected profile directories, nonblocking shared/exclusive locking, mode-0600 state, and file/directory durability.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgelifecycle/profile.go`
- consumers: qxctl lifecycle profile commands, tests, reviewers
- deferred_projections: supervised system-scope state administration
- notes: Stored state is noncanonical and contains no credentials or provider payloads.
- status: canonical

#### qxctl Unsupported Native Lifecycle Profile State
- path: `tools/qxctl/internal/knowledgelifecycle/state_unsupported.go`
- title: qxctl Unsupported Native Lifecycle Profile State
- surface_type: platform fail-closed implementation
- truth_role: unsupported-native-operating-system rejection truth
- owner: qxctl maintainers
- scope: Refuses lifecycle profile persistence where the Linux/macOS protected-state contract is unavailable.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgelifecycle/profile.go`
- consumers: qxctl lifecycle commands, tests, reviewers
- deferred_projections: WSL and remote-node administration documentation
- notes: It does not introduce a native Windows lifecycle state implementation.
- status: canonical

#### qxctl Lifecycle Conformance Tests
- path: `tools/qxctl/cmd/qxctl/lifecycle_test.go`
- title: qxctl Lifecycle Plan and Grammar Conformance Tests
- surface_type: Go conformance-test implementation
- truth_role: report/apply result validation, dynamic-scheduler, exact-CAS, and command-registration proof
- owner: qxctl maintainers
- scope: Verifies exact plan and v1/v2 journal/result field/digest/set validation, scheduler invariants, invalid evidence rejection, and lifecycle Cobra grammar.
- relationships: verifies -> `tools/qxctl/cmd/qxctl/lifecycle.go`; verifies -> `tools/qxctl/cmd/qxctl/lifecycle_apply.go`; verifies -> `knowledge/schemas/v1/lifecycle-plan.schema.json`; verifies -> `knowledge/schemas/v1/lifecycle-apply-result.schema.json`
- consumers: qxctl maintainers, reviewers, release gates
- deferred_projections: live adapter integration tests
- notes: Tests prove bounded noncanonical handling and grant no canonical lifecycle authority.
- status: canonical

#### qxctl Linux Lifecycle Host Command
- path: `tools/qxctl/cmd/qxctl/lifecycle_host.go`
- title: qxctl Linux Report-Only Lifecycle Host Command
- surface_type: Go Cobra lifecycle-host administration implementation
- truth_role: caller-neutral systemd receptor administration and kernel-boot dispatch implementation truth
- owner: qxctl maintainers
- scope: Implements exact-SSIAG install/update/status/reconcile/enable/disable/uninstall/run grammar, descriptor authority, accepted-executor verification, kernel boot UUID idempotency, optional bounded discovery recovery, and report-only boot invocation.
- relationships: implements -> `knowledge/LIFECYCLE.md`; implements -> `knowledge/schemas/v1/lifecycle-host-integration-result.schema.json`; emits -> `knowledge/schemas/v1/lifecycle-host-boot-result.schema.json`; calls -> `tools/qxctl/internal/knowledgelifecycle/host.go`; calls -> `tools/qxctl/internal/knowledgelifecycle/host_admin_unix.go`
- consumers: Linux TOPS administrators, systemd, qxctl tests, reviewers
- deferred_projections: remote Linux host administration and cross-platform UI
- notes: Native Windows and launchd receptors, lifecycle apply, component execution, login/session hooks, and hidden watchers remain excluded.
- status: canonical

#### qxctl Lifecycle Host Descriptor Model
- path: `tools/qxctl/internal/knowledgelifecycle/host.go`
- title: qxctl Protected Lifecycle Host Descriptor Model
- surface_type: bounded Go lifecycle host-state implementation
- truth_role: descriptor validation, digest continuity, compare-and-swap, executor compatibility-set, and deterministic unit-rendering truth
- owner: qxctl maintainers
- scope: Stores one protected v1 host descriptor per TOPS/profile beside lifecycle profiles and binds system roots, unit identity/digest, active and fallback executors, recovery mode, desired enablement, generation, predecessor, and STSC time.
- relationships: implements -> `knowledge/schemas/v1/lifecycle-host-integration.schema.json`; implements -> `knowledge/LIFECYCLE.md`; persisted_by -> `tools/qxctl/internal/knowledgelifecycle/state_unix.go`; called_by -> `tools/qxctl/cmd/qxctl/lifecycle_host.go`
- consumers: qxctl host administration, systemd unit rendering, tests, reviewers
- deferred_projections: descriptor v2 after explicit dual-read ratification
- notes: The model infers no version recency and accepts no caller-class authority.
- status: canonical

#### qxctl Unix Lifecycle Host Administrator
- path: `tools/qxctl/internal/knowledgelifecycle/host_admin_unix.go`
- title: qxctl Unix Lifecycle Host Administrator
- surface_type: platform-specific protected systemd integration implementation
- truth_role: content-addressed executor provisioning, systemd reconciliation, fallback promotion, and resumable uninstall truth
- owner: qxctl maintainers
- scope: On Linux, copies exact qxctl bytes to protected digest slots, atomically writes the stable unit, reconciles desired enablement, promotes an accepted valid fallback, and removes only marker-owned integration state after durable retiring intent.
- relationships: implements -> `knowledge/LIFECYCLE.md`; implements -> `knowledge/schemas/v1/lifecycle-host-integration-result.schema.json`; called_by -> `tools/qxctl/cmd/qxctl/lifecycle_host.go`; verifies -> `knowledge/schemas/v1/lifecycle-host-integration.schema.json`
- consumers: qxctl Linux host administration, tests, reviewers
- deferred_projections: package-manager adapters and remote-node transport
- notes: Darwin compiles the testable implementation but production preflight refuses it; Linux system scope and administrator privilege are mandatory.
- status: canonical

#### qxctl Unsupported Lifecycle Host Administrator
- path: `tools/qxctl/internal/knowledgelifecycle/host_admin_unsupported.go`
- title: qxctl Unsupported Native Lifecycle Host Administrator
- surface_type: platform fail-closed implementation
- truth_role: unsupported-native-platform host-integration rejection truth
- owner: qxctl maintainers
- scope: Refuses native lifecycle host integration outside the Unix implementation boundary while preserving qxctl command compilation.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/cmd/qxctl/lifecycle_host.go`
- consumers: qxctl builds, tests, reviewers
- deferred_projections: none; Windows uses WSL or a remote Linux TOPS node
- notes: It introduces no native Windows lifecycle engine or weaker persistence fallback.
- status: canonical

#### qxctl Lifecycle Host Conformance Tests
- path: `tools/qxctl/internal/knowledgelifecycle/host_test.go`
- title: qxctl Lifecycle Host Conformance Tests
- surface_type: Go platform conformance-test implementation
- truth_role: descriptor CAS, report-only unit, fallback recovery, and resumable uninstall proof
- owner: qxctl maintainers
- scope: Verifies digest-linked descriptor idempotency, stable systemd grammar, absence of apply/shell authority, fallback promotion, and complete marker-owned uninstall.
- relationships: verifies -> `tools/qxctl/internal/knowledgelifecycle/host.go`; verifies -> `tools/qxctl/internal/knowledgelifecycle/host_admin_unix.go`; conforms_to -> `knowledge/LIFECYCLE.md`
- consumers: qxctl maintainers, reviewers, release gates
- deferred_projections: live disposable-systemd integration matrix
- notes: Tests use private temporary roots and a deterministic fake systemctl runner.
- status: canonical

#### qxctl Lifecycle Executor Tests
- path: `tools/qxctl/internal/knowledgelifecycle/executor_test.go`
- title: qxctl Lifecycle Executor Conformance Tests
- surface_type: Go platform conformance-test implementation
- truth_role: runtime compare-and-swap, receipt-v2 install/uninstall, interruption replay, rollback proof, and conflict-protection proof
- owner: qxctl maintainers
- scope: Verifies selection/activation idempotency and active deselection refusal; staged package install/uninstall and safe retry; separate rollback proof; and preservation of conflicting administrator files.
- relationships: verifies -> `tools/qxctl/internal/knowledgelifecycle/executor.go`; verifies -> `tools/qxctl/internal/knowledgelifecycle/install_unix.go`; verifies -> `tools/qxctl/internal/knowledgelifecycle/runtime.go`
- consumers: qxctl maintainers, reviewers, release gates
- deferred_projections: crash-injection and package-manager matrix evidence
- notes: Fixtures are local, nonsecret, and execute no installed content.
- status: canonical

#### qxctl Lifecycle State and Observation Tests
- path: `tools/qxctl/internal/knowledgelifecycle/profile_test.go`
- title: qxctl Lifecycle Profile and Observation Conformance Tests
- surface_type: Go conformance-test implementation
- truth_role: profile durability/continuity and fixed-layout observation proof
- owner: qxctl maintainers
- scope: Verifies compare-and-swap, semantic retry, identity drift rejection, required-array shape, v1/v2 receipts, content drift, unknown preservation, future absent roots, and timestamp-neutral stable inventory.
- relationships: verifies -> `tools/qxctl/internal/knowledgelifecycle/profile.go`; verifies -> `tools/qxctl/internal/knowledgelifecycle/observation.go`; verifies -> `tools/qxctl/internal/knowledgelifecycle/scan_unix.go`
- consumers: qxctl maintainers, reviewers, release gates
- deferred_projections: crash-injection and system-scope supervision evidence
- notes: Test fixtures contain no credentials and execute no discovered package.
- status: canonical

#### qxctl CLI Compatibility Tests
- path: `tools/qxctl/cmd/qxctl/cli_compat_test.go`
- title: qxctl Command Compatibility Tests
- surface_type: Go command-compatibility test implementation
- truth_role: stable usage, failure routing, flag precedence, and forbidden-Viper capability proof
- owner: qxctl maintainers
- scope: Verifies exact CLI output and error grammar including lifecycle leaves while preserving constrained configuration behavior.
- relationships: verifies -> `tools/qxctl/cmd/qxctl/commands.go`; verifies -> `tools/qxctl/cmd/qxctl/testdata/help.golden`
- consumers: qxctl maintainers, reviewers, release gates
- deferred_projections: generated shell completions
- notes: Compatibility proof does not convert reserved operations into implemented behavior.
- status: canonical

#### qxctl Help Golden
- path: `tools/qxctl/cmd/qxctl/testdata/help.golden`
- title: qxctl Exact Help Compatibility Golden
- surface_type: command-output test fixture
- truth_role: exact implemented command-help expectation
- owner: qxctl maintainers
- scope: Records the stable concise usage text verified by the CLI compatibility suite.
- relationships: generated_by -> `tools/qxctl/cmd/qxctl/main.go`; verified_by -> `tools/qxctl/cmd/qxctl/cli_compat_test.go`
- consumers: qxctl tests, maintainers, reviewers
- deferred_projections: public command reference
- notes: The fixture is test evidence and not an independent command authority.
- status: canonical

#### qxctl Knowledge Engine Binding Registry
- path: `tools/qxctl/internal/knowledgebinding/registry.go`
- title: qxctl Knowledge Engine Binding Registry
- surface_type: protected local lifecycle-state implementation
- truth_role: exact user-default engine selection, expected-state, digest, and doctor implementation truth
- owner: qxctl maintainers
- scope: Records one exact inactive-undocked installation per coordinator/vector role in a noncanonical user-default profile, emits STSC whole-second UTC, preserves legacy fractional-second read compatibility, and revalidates bound receipt and executable digests.
- relationships: implements -> `knowledge/SPEC.md`; conforms_to -> `knowledge/TIME.md`; implements -> `knowledge/schemas/v1/engine-binding-registry.schema.json`; called_by -> `tools/qxctl/cmd/qxctl/main.go`
- consumers: qxctl knowledge commands, reconciliation coordinator integration, tests
- deferred_projections: repository/system/TOPS profiles and Maestro docking
- notes: Binding is not installation, engine invocation, authentication, permission, repository activation, or docking.
- status: canonical

#### qxctl Knowledge Engine Binding Temporal Tests
- path: `tools/qxctl/internal/knowledgebinding/registry_test.go`
- title: qxctl Knowledge Engine Binding Temporal Tests
- surface_type: Go conformance-test implementation
- truth_role: STSC write normalization and legacy read-compatibility proof
- owner: qxctl maintainers
- scope: Proves new binding generations emit whole-second UTC and prior fractional-second v1 state remains readable across upgrade order.
- relationships: verifies -> `tools/qxctl/internal/knowledgebinding/registry.go`; conforms_to -> `knowledge/TIME.md`
- consumers: Go test, qxctl maintainers, reviewers
- deferred_projections: cross-language temporal conformance report
- notes: Compatibility acceptance does not make fractional seconds a new canonical write profile.
- status: canonical

#### qxctl Unix Knowledge Binding State
- path: `tools/qxctl/internal/knowledgebinding/state_unix.go`
- title: qxctl Unix Knowledge Binding State
- surface_type: platform-specific protected local-state implementation
- truth_role: no-follow lock, atomic replacement, and durability implementation truth
- owner: qxctl maintainers
- scope: Implements Linux/macOS owner-controlled state directories, shared/exclusive no-follow registry locking, mode-0600 files, same-directory atomic replacement, and directory durability.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgebinding/registry.go`
- consumers: qxctl knowledge commands, tests, reviewers
- deferred_projections: system/TOPS protected state
- notes: The persistent lock file is never unlinked during normal operation.
- status: canonical

#### qxctl Unsupported Native Knowledge Binding State
- path: `tools/qxctl/internal/knowledgebinding/state_unsupported.go`
- title: qxctl Unsupported Native Knowledge Binding State
- surface_type: platform fail-closed implementation
- truth_role: unsupported-native-operating-system rejection truth
- owner: qxctl maintainers
- scope: Refuses binding-registry access where the Linux/macOS protected local-state contract is unavailable.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgebinding/registry.go`
- consumers: qxctl knowledge commands, tests, reviewers
- deferred_projections: WSL and remote administration documentation
- notes: It does not introduce a native Windows state implementation.
- status: canonical

#### qxctl Unix No-Follow Knowledge-Engine Access
- path: `tools/qxctl/internal/knowledgeengine/open_relative_unix.go`
- title: qxctl Unix No-Follow Knowledge-Engine Access
- surface_type: platform-specific secure file-access implementation
- truth_role: Linux and macOS descriptor traversal plus installation trust-check implementation truth
- owner: qxctl maintainers
- scope: Opens exact relative receipt paths without following symlinks and rejects group/world-writable or foreign-owned engine installations.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: qxctl vector-engine clients, tests, reviewers
- deferred_projections: none
- notes: Effective-user and root ownership are accepted without classifying the caller.
- status: canonical

#### qxctl Unsupported Native Knowledge-Engine Access
- path: `tools/qxctl/internal/knowledgeengine/open_relative_unsupported.go`
- title: qxctl Unsupported Native Knowledge-Engine Access
- surface_type: platform fail-closed implementation
- truth_role: unsupported-native-operating-system rejection truth
- owner: qxctl maintainers
- scope: Refuses knowledge-engine receipt traversal where the Linux/macOS no-follow contract is unavailable.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; called_by -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: qxctl vector-engine clients, tests, reviewers
- deferred_projections: WSL and remote administration documentation
- notes: It does not introduce a weaker native Windows fallback.
- status: canonical

#### qxctl SSFV Operation Layer
- path: `tools/qxctl/cmd/qxctl/ssfv.go`
- title: qxctl SSFV Operation Layer
- surface_type: administrative CLI implementation surface
- truth_role: SSFV payload assembly, result-safety validation, and presentation implementation truth
- owner: qxctl maintainers
- scope: Implements no-follow baselines/inputs, freshness coupling, proposal-target checks, and disposable graph validation.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; implements -> `knowledge/ssfv/SPEC.md`; invokes -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: qxctl executable, tests, administrators, reviewers
- deferred_projections: generated SSFV CLI reference
- notes: The command layer does not parse feature semantics, ratify, apply, or persist graphs.
- status: canonical

### Knowledge Session Coordinator Module

#### Knowledge Session Coordinator INTENT.md
- path: `modules/knowledge-session-coordinator/INTENT.md`
- title: Knowledge Session Coordinator Intent
- surface_type: coordinator module intent
- truth_role: domain-neutral session/reconciliation purpose and implemented boundary
- owner: SKV coordinator maintainers
- scope: Declares the `0.1.0-dev` read/check, durable reconciliation, SSIAG-authorized noncanonical authenticated-session lifecycles, report-only lifecycle administration, and separate apply-capable attempt/applied-state coordination.
- relationships: depends_on -> `knowledge/SPEC.md`; declares -> `modules/knowledge-session-coordinator/MANIFEST.md`
- consumers: qxctl and vector-engine implementers, reviewers, administrators, agentic tools
- deferred_projections: session and worktree reconciliation operator evidence
- notes: Successful inspect/check does not establish authority; each session operation requires a fresh exact SSIAG decision.
- status: canonical

#### Knowledge Session Coordinator MANIFEST.md
- path: `modules/knowledge-session-coordinator/MANIFEST.md`
- title: Knowledge Session Coordinator Manifest
- surface_type: independently installable coordinator manifest
- truth_role: executable, protocol, operation, dependency, and lifecycle truth
- owner: SKV coordinator maintainers
- scope: Declares implemented inspect/check/reconciliation/authenticated-session/lifecycle-plan/lifecycle-boot/lifecycle-apply coordination operations, disabled canonical apply, and installed-undocked state.
- relationships: depends_on -> `modules/knowledge-session-coordinator/INTENT.md`; implements -> `knowledge/SPEC.md`; statically_links -> `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- consumers: qxctl planners, packagers, implementers, reviewers, agentic tools
- deferred_projections: installed-engine inventory and Maestro presence evidence
- notes: No default receptor or unversioned active alias is selected.
- status: canonical

#### Knowledge Session Coordinator INSTALL.md
- path: `modules/knowledge-session-coordinator/INSTALL.md`
- title: Knowledge Session Coordinator Installation
- surface_type: module installation contract
- truth_role: build, test, versioned install, and receipt-owned uninstall procedure
- owner: SKV coordinator maintainers
- scope: Defines monorepo and installed-foundation builds plus isolated prefix lifecycle.
- relationships: depends_on -> `modules/knowledge-session-coordinator/MANIFEST.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/INSTALL.md`
- consumers: implementers, administrators, packagers, reviewers
- deferred_projections: qxctl install/rollback/uninstall evidence
- notes: Installation leaves the coordinator undocked and inactive.
- status: canonical

#### Knowledge Session Coordinator SKILL.md
- path: `modules/knowledge-session-coordinator/SKILL.md`
- title: Knowledge Session Coordinator Skill
- surface_type: coordinator skill guidance
- truth_role: safe direct diagnostics and process invocation procedure
- owner: SKV coordinator maintainers
- scope: Guides descriptor, inspect, check, compatibility, durable reconciliation/session lifecycles, report-only lifecycle planning, durable boot/status/recovery, apply prepare/finalize/close/status/recovery, SSIAG evidence, deadlines, stdout, and stop-condition handling.
- relationships: depends_on -> `modules/knowledge-session-coordinator/SPEC.md`; depends_on -> `knowledge/SKILL.md`
- consumers: administrators, implementers, reviewers, agentic tools
- deferred_projections: additional external action-adapter procedures
- notes: All journals and applied evidence are noncanonical; host action execution remains outside the coordinator and canonical apply remains disabled.
- status: canonical

#### Knowledge Session Coordinator SPEC.md
- path: `modules/knowledge-session-coordinator/SPEC.md`
- title: Knowledge Session Coordinator Specification
- surface_type: coordinator module specification
- truth_role: exact read/check/reconciliation/session/report/apply lifecycle operation, compatibility, durability, recovery, exit, descriptor, install, and non-authorization contract
- owner: SKV coordinator maintainers
- scope: Defines process inspect/check, durable reconciliation and authority epochs, persistent SSFV maintenance, deterministic planning, report-only lifecycle boot/status/recovery, and separately authorized v2 prepare/finalize/close/status/recovery with verified applied-state commitment; canonical apply remains disabled.
- relationships: depends_on -> `knowledge/SPEC.md`; depends_on -> `knowledge/ssiag/SPEC.md`; implements -> `knowledge/schemas/v1/engine-process-request.schema.json`; implements -> `knowledge/schemas/v1/engine-process-response.schema.json`; implements -> `knowledge/schemas/v1/reconciliation-command.schema.json`; implements -> `knowledge/schemas/v1/reconciliation-result.schema.json`; persists -> `knowledge/schemas/v1/reconciliation-journal.schema.json`; persists -> `knowledge/schemas/v1/reconciliation-head.schema.json`; implements -> `knowledge/schemas/v1/session-command.schema.json`; implements -> `knowledge/schemas/v1/session-result.schema.json`; persists -> `knowledge/schemas/v1/session-journal.schema.json`; persists -> `knowledge/schemas/v1/session-head.schema.json`; implements -> `knowledge/schemas/v1/ssfv-maintenance-command.schema.json`; emits -> `knowledge/schemas/v1/ssfv-maintenance-result.schema.json`; persists -> `knowledge/schemas/v1/ssfv-maintenance-journal.schema.json`; persists -> `knowledge/schemas/v1/ssfv-maintenance-head.schema.json`; implements -> `knowledge/schemas/v1/lifecycle-plan-command.schema.json`; emits -> `knowledge/schemas/v1/lifecycle-plan.schema.json`; implements -> `knowledge/schemas/v1/lifecycle-boot-command.schema.json`; emits -> `knowledge/schemas/v1/lifecycle-boot-result.schema.json`; persists -> `knowledge/schemas/v1/lifecycle-boot-journal.schema.json`; persists -> `knowledge/schemas/v1/lifecycle-boot-head.schema.json`; implements -> `knowledge/schemas/v1/lifecycle-apply-command.schema.json`; emits -> `knowledge/schemas/v1/lifecycle-apply-result.schema.json`; persists -> `knowledge/schemas/v2/lifecycle-boot-journal.schema.json`; persists -> `knowledge/schemas/v2/lifecycle-boot-head.schema.json`; persists -> `knowledge/schemas/v1/lifecycle-applied-state.schema.json`
- consumers: C++ implementers, qxctl, testers, reviewers
- deferred_projections: live component service/process adapters and direct coordinator host integration
- notes: qxctl owns desired/runtime administration, read-only vector/Maestro evidence collection, SSIAG exchange, and the explicit Linux report-only host receptor; the coordinator owns protected journals, attempt serialization, verification, applied-state selection, and recovery. Direct SSIAG/STAV calls, vector invocation, canonical apply, and Maestro state writes remain unimplemented.
- status: canonical

#### Knowledge Session Coordinator FEATURES.md
- path: `modules/knowledge-session-coordinator/FEATURES.md`
- title: Knowledge Session Coordinator Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for durable reconciliation, authenticated-session, persistent SSFV maintenance, report-only lifecycle, and apply coordination
- owner: SKV coordinator maintainers
- scope: Owns the experimental coordinator feature plus five ratified F1 subfeatures for reconciliation, authority epochs, semantic maintenance, lifecycle planning, and lifecycle apply coordination in exact source scope `modules/knowledge-session-coordinator`.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`; depends_on -> `modules/knowledge-session-coordinator/SPEC.md`; declares -> `ssfv:symphony:knowledge-session-coordinator`; declares -> five `ssfv:symphony:knowledge-session-coordinator.*` F1 subfeatures
- consumers: symphony-ssfv, qxctl, coordinator maintainers, reviewers, administrators, agentic tools
- deferred_projections: portable SSFV graph, authenticated-session capability lineage, operator documentation
- notes: Records the parent coordinator feature and five separately identified implemented subsystems; canonical apply and Maestro engine execution remain unimplemented.
- status: canonical

#### Knowledge Session Coordinator CMakeLists.txt
- path: `modules/knowledge-session-coordinator/CMakeLists.txt`
- title: Knowledge Session Coordinator Build Contract
- surface_type: module build and install contract
- truth_role: static-link, test, package receipt, and uninstall implementation truth
- owner: SKV coordinator maintainers
- scope: Builds the exact versioned executable and supports source or installed foundation consumption.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`
- consumers: CMake, implementers, packagers, reviewers
- deferred_projections: reproducible build and receipt evidence
- notes: No global executable alias or active binding is installed.
- status: canonical

#### Knowledge Session Coordinator Reconciliation Header
- path: `modules/knowledge-session-coordinator/src/reconciliation.hpp`
- title: Knowledge Session Coordinator Reconciliation Interface
- surface_type: C++ coordinator implementation surface
- truth_role: reconciliation capability and request-handler boundary
- owner: SKV coordinator maintainers
- scope: Declares the internal compatibility description and bounded reconciliation dispatcher.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implemented_by -> `modules/knowledge-session-coordinator/src/reconciliation.cpp`
- consumers: coordinator dispatcher, tests, reviewers
- deferred_projections: authenticated-session integration
- notes: The interface grants no canonical or authentication authority.
- status: canonical

#### Knowledge Session Coordinator Dispatcher
- path: `modules/knowledge-session-coordinator/src/coordinator.cpp`
- title: Knowledge Session Coordinator Operation Dispatcher
- surface_type: C++ coordinator implementation surface
- truth_role: descriptor, inspect, operation routing, deadline, and canonical-non-apply boundary implementation truth
- owner: SKV coordinator maintainers
- scope: Routes bounded process requests across inspect/check, reconciliation, authenticated-session, persistent SSFV maintenance, lifecycle-plan, report boot/status/recovery, and apply prepare/finalize/close/status/recovery while keeping canonical apply disabled.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; dispatches_to -> `modules/knowledge-session-coordinator/src/reconciliation.cpp`; dispatches_to -> `modules/knowledge-session-coordinator/src/authority_session.cpp`; dispatches_to -> `modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp`; dispatches_to -> `modules/knowledge-session-coordinator/src/lifecycle.cpp`; dispatches_to -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`
- consumers: symphony-knowledge-session, qxctl, process tests, reviewers
- deferred_projections: vector invocation, coordinator-owned host actions, canonical apply
- notes: Operation availability is reported literally; capability availability grants no caller authority and report-only results never become applied state.
- status: canonical

#### Knowledge Session Coordinator Lifecycle Header
- path: `modules/knowledge-session-coordinator/src/lifecycle.hpp`
- title: Knowledge Session Coordinator Lifecycle Planner Interface
- surface_type: C++ coordinator implementation surface
- truth_role: report-only lifecycle capability and request-handler boundary
- owner: SKV coordinator maintainers
- scope: Declares the planner capability description, reusable deterministic plan builder, stable-inventory digest, and bounded lifecycle-plan dispatcher.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implemented_by -> `modules/knowledge-session-coordinator/src/lifecycle.cpp`
- consumers: coordinator dispatcher, tests, reviewers
- deferred_projections: lifecycle action execution
- notes: The interface grants no lifecycle or canonical mutation authority.
- status: canonical

#### Knowledge Session Coordinator Lifecycle Implementation
- path: `modules/knowledge-session-coordinator/src/lifecycle.cpp`
- title: Knowledge Session Coordinator Report-Only Lifecycle Planner
- surface_type: C++ freezing-path planning implementation
- truth_role: strict evidence parsing, capability negotiation, dependency-ready-set, deterministic action, receptor-target, and blocker-isolation implementation truth
- owner: SKV coordinator maintainers
- scope: Validates supplied desired/observed lifecycle evidence and emits deterministic forward/inverse plans with exact target identities, localized blockers, cycle isolation, and disabled apply.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implements -> `knowledge/schemas/v1/lifecycle-plan-command.schema.json`; reads -> `knowledge/schemas/v1/lifecycle-desired-state.schema.json`; reads -> `knowledge/schemas/v1/lifecycle-observation.schema.json`; emits -> `knowledge/schemas/v1/lifecycle-plan.schema.json`
- consumers: symphony-knowledge-session, qxctl lifecycle client, tests, reviewers
- deferred_projections: action execution, applied-state persistence, Maestro exchange
- notes: No filesystem discovery, persistence, authorization, action execution, or receptor contact occurs.
- status: canonical

#### Knowledge Session Coordinator Lifecycle Journal Header
- path: `modules/knowledge-session-coordinator/src/lifecycle_journal.hpp`
- title: Knowledge Session Coordinator Lifecycle Journal Interface
- surface_type: C++ coordinator implementation surface
- truth_role: report-only v1 and apply-capable v2 lifecycle capability and request-handler boundary
- owner: SKV coordinator maintainers
- scope: Declares report-journal compatibility plus bounded boot/status/recovery and apply prepare/finalize/close/status/recovery dispatchers.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implemented_by -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`
- consumers: coordinator dispatcher, tests, reviewers
- deferred_projections: direct host action and Maestro adapters
- notes: The interface coordinates attempts and applied evidence but grants no package, docking, or canonical authority by itself.
- status: canonical

#### Knowledge Session Coordinator Lifecycle Journal Implementation
- path: `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`
- title: Knowledge Session Coordinator Durable Lifecycle Journals
- surface_type: C++ freezing-path state implementation
- truth_role: SSIAG binding, source-journal linkage, compare-and-swap, prepared attempts, applied-state commitment, dynamic replanning, dual-slot durability, and explicit recovery implementation truth
- owner: SKV coordinator maintainers
- scope: Implements separate private report/apply locks and streams, exact authorization/stable-inventory/source binding, linked plans/checkpoints, timestamp-neutral rescans, prepared/completed attempts, content-addressed applied evidence, atomic heads, extension preservation, fail-closed critical state, and forward recovery.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implements -> `knowledge/schemas/v1/lifecycle-boot-command.schema.json`; implements -> `knowledge/schemas/v1/lifecycle-apply-command.schema.json`; implements -> `knowledge/ssiag/schemas/v1/authorization-decision.schema.json`; reads -> `knowledge/schemas/v1/lifecycle-desired-state.schema.json`; reads -> `knowledge/schemas/v1/lifecycle-observation.schema.json`; persists -> `knowledge/schemas/v1/lifecycle-boot-journal.schema.json`; persists -> `knowledge/schemas/v1/lifecycle-boot-head.schema.json`; persists -> `knowledge/schemas/v2/lifecycle-boot-journal.schema.json`; persists -> `knowledge/schemas/v2/lifecycle-boot-head.schema.json`; persists -> `knowledge/schemas/v1/lifecycle-applied-state.schema.json`; emits -> `knowledge/schemas/v1/lifecycle-boot-result.schema.json`; emits -> `knowledge/schemas/v1/lifecycle-apply-result.schema.json`
- consumers: symphony-knowledge-session, qxctl lifecycle administration, tests, reviewers
- deferred_projections: live component service/process adapters and Maestro docking
- notes: Report v1 remains apply-false and can be dispatched by qxctl's explicit Linux host receptor; apply v2 records operation-bound noncanonical authority and attempts. Neither grants canonical apply.
- status: canonical

#### Knowledge Session Coordinator Lifecycle Planner Tests
- path: `modules/knowledge-session-coordinator/tests/lifecycle_test.cpp`
- title: Knowledge Session Coordinator Lifecycle Planner Conformance Tests
- surface_type: C++26 conformance-test implementation
- truth_role: deterministic two-way dependency planner, compatibility, blocker, receptor, scale, stable-inventory, durability, SSIAG binding, and recovery proof
- owner: SKV coordinator maintainers
- scope: Verifies forward/inverse selection, dynamic readiness healing, cycle isolation, safety ordering, bounded scale, integrity/compatibility failures, timestamp-neutral identity, report/apply source linkage, prepared attempts, replay, applied closure, compare-and-swap, active-action recovery, SSIAG/inventory binding, and explicit recovery.
- relationships: verifies -> `modules/knowledge-session-coordinator/src/lifecycle.cpp`; verifies -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`; verifies -> `knowledge/schemas/v1/lifecycle-plan.schema.json`; verifies -> `knowledge/schemas/v1/lifecycle-boot-journal.schema.json`; verifies -> `knowledge/schemas/v2/lifecycle-boot-journal.schema.json`; verifies -> `knowledge/schemas/v1/lifecycle-applied-state.schema.json`
- consumers: coordinator/qxctl maintainers, reviewers, release gates
- deferred_projections: direct host-adapter and Maestro conformance
- notes: Tests exercise planning, report/apply durability, attempt verification, and applied evidence but never perform package, canonical, or Maestro mutation.
- status: canonical

#### Knowledge Session Coordinator Dispatcher Tests
- path: `modules/knowledge-session-coordinator/tests/coordinator_test.cpp`
- title: Knowledge Session Coordinator Dispatcher Conformance Tests
- surface_type: C++26 bounded process-dispatch conformance test
- truth_role: Maestro-aware coordinator descriptor, routing, compatibility, and authority-boundary proof
- owner: SKV coordinator maintainers
- scope: Verifies coordinator operation dispatch and descriptor behavior after Maestro receptor-presence integration while preserving the coordinator's non-writer boundary.
- relationships: verifies -> `modules/knowledge-session-coordinator/src/coordinator.cpp`; verifies -> `modules/knowledge-session-coordinator/SPEC.md`; observes -> `modules/maestro/SPEC.md`
- consumers: coordinator maintainers, qxctl maintainers, reviewers, release gates
- deferred_projections: Maestro state mutation, engine invocation, scheduling, or supervision
- notes: Test evidence does not make the coordinator a Maestro writer or lifecycle action executor.
- status: canonical

#### Knowledge Session Coordinator Process Smoke Tests
- path: `modules/knowledge-session-coordinator/tests/process_smoke.sh`
- title: Knowledge Session Coordinator Process Conformance Smoke Tests
- surface_type: bounded process-level conformance test
- truth_role: executable identity, descriptor, deterministic response, operation availability, disabled-apply, and error-envelope proof
- owner: SKV coordinator maintainers
- scope: Verifies the direct executable surface, including durable lifecycle operation descriptors and persistence/non-apply inspection claims.
- relationships: verifies -> `modules/knowledge-session-coordinator/src/coordinator.cpp`; verifies -> `modules/knowledge-session-coordinator/SPEC.md`; verifies -> `knowledge/schemas/v1/engine-process-response.schema.json`
- consumers: coordinator maintainers, packagers, reviewers, release gates
- deferred_projections: installed host-boot-hook and lifecycle-action conformance
- notes: The smoke test invokes only read/report surfaces and unsupported-operation failure; it performs no package, docking, or canonical mutation.
- status: canonical

#### Knowledge Session Coordinator Authority-Session Header
- path: `modules/knowledge-session-coordinator/src/authority_session.hpp`
- title: Knowledge Session Coordinator Authority-Session Interface
- surface_type: C++ coordinator implementation surface
- truth_role: authenticated-session capability and request-handler boundary
- owner: SKV coordinator maintainers
- scope: Declares the internal session compatibility description and bounded SSIAG-evidence-aware session dispatcher.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implemented_by -> `modules/knowledge-session-coordinator/src/authority_session.cpp`
- consumers: coordinator dispatcher, tests, reviewers
- deferred_projections: authenticated-session conformance evidence
- notes: The interface validates decision evidence but grants no canonical or vector-semantic authority.
- status: canonical

#### Knowledge Session Coordinator Authority-Session Implementation
- path: `modules/knowledge-session-coordinator/src/authority_session.cpp`
- title: Knowledge Session Coordinator Durable Authority Epochs
- surface_type: C++ freezing-path state implementation
- truth_role: SSIAG evidence validation, session durability, compare-and-swap, replay, linked-epoch, and recovery implementation truth
- owner: SKV coordinator maintainers
- scope: Implements private user-scope per-TOPS/subject/repository locks, dual session-journal slots, atomic heads, exact decision/capability binding, context attachment, extension preservation, and explicit evidence-based repair.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implements -> `knowledge/schemas/v1/session-command.schema.json`; implements -> `knowledge/ssiag/schemas/v1/authorization-decision.schema.json`; implements -> `knowledge/ssiag/schemas/v1/capability.schema.json`; persists -> `knowledge/schemas/v1/session-journal.schema.json`; persists -> `knowledge/schemas/v1/session-head.schema.json`; emits -> `knowledge/schemas/v1/session-result.schema.json`
- consumers: symphony-knowledge-session, qxctl, tests, reviewers
- deferred_projections: vector invocation, observers, canonical apply, Maestro docking
- notes: Evidence is non-transferable and canonical apply remains false; ambiguous, expired, critical, or incompatible state is preserved and blocks mutation.
- status: canonical

#### Knowledge Session Coordinator Reconciliation Implementation
- path: `modules/knowledge-session-coordinator/src/reconciliation.cpp`
- title: Knowledge Session Coordinator Durable Reconciliation
- surface_type: C++ freezing-path state implementation
- truth_role: capability negotiation, journal durability, compare-and-swap, replay, and recovery implementation truth
- owner: SKV coordinator maintainers
- scope: Implements private user-scope per-worktree locks, dual journal slots, atomic heads, content and engine-inventory checkpoints, extension preservation, and explicit evidence-based repair.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implements -> `knowledge/schemas/v1/reconciliation-command.schema.json`; persists -> `knowledge/schemas/v1/reconciliation-journal.schema.json`; persists -> `knowledge/schemas/v1/reconciliation-head.schema.json`; emits -> `knowledge/schemas/v1/reconciliation-result.schema.json`
- consumers: symphony-knowledge-session, qxctl, tests, reviewers
- deferred_projections: authenticated-session journal, vector invocation, observers, canonical apply, Maestro docking
- notes: Unknown critical/newer state and ambiguous evidence are preserved and block automated downgrade.
- status: canonical

### SKVI Engine Module

#### SKVI Engine INTENT.md
- path: `modules/skvi-engine/INTENT.md`
- title: SKVI Engine Intent
- surface_type: vector-engine module intent
- truth_role: subordinate structural-check, proposal, and projection purpose
- owner: SKVI engine maintainers
- scope: Declares deterministic inspect/check/propose/project behavior without membership or mutation authority.
- relationships: depends_on -> `knowledge/skvi/INTENT.md`; declares -> `modules/skvi-engine/MANIFEST.md`
- consumers: qxctl, implementers, administrators, reviewers, agentic tools
- deferred_projections: installed-engine inventory and conformance evidence
- notes: The engine implements SKVI truth but does not own it.
- status: canonical

#### SKVI Engine MANIFEST.md
- path: `modules/skvi-engine/MANIFEST.md`
- title: SKVI Engine Manifest
- surface_type: independently installable vector-engine manifest
- truth_role: executable, operation, protocol, dependency, and lifecycle truth
- owner: SKVI engine maintainers
- scope: Declares the C++26 executable, implemented operation set, disabled apply, and inactive installed-undocked state.
- relationships: depends_on -> `modules/skvi-engine/INTENT.md`; implements -> `knowledge/skvi/SPEC.md`; statically_links -> `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- consumers: qxctl, packagers, implementers, reviewers, agentic tools
- deferred_projections: engine inventory and Maestro presence evidence
- notes: No default receptor, active alias, or canonical write route exists.
- status: canonical

#### SKVI Engine INSTALL.md
- path: `modules/skvi-engine/INSTALL.md`
- title: SKVI Engine Installation
- surface_type: module installation contract
- truth_role: build, test, versioned install, qxctl invocation, and receipt-owned uninstall procedure
- owner: SKVI engine maintainers
- scope: Defines monorepo and installed-foundation builds plus exact prefix installation and removal.
- relationships: depends_on -> `modules/skvi-engine/MANIFEST.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/INSTALL.md`; consumed_by -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: implementers, administrators, packagers, qxctl, reviewers
- deferred_projections: lifecycle administration evidence
- notes: Installation leaves the version inactive and undocked.
- status: canonical

#### SKVI Engine SKILL.md
- path: `modules/skvi-engine/SKILL.md`
- title: SKVI Engine Skill
- surface_type: vector-engine skill guidance
- truth_role: safe direct and qxctl operation procedure
- owner: SKVI engine maintainers
- scope: Guides check, projection, caller-declared proposal review, and stop conditions.
- relationships: depends_on -> `modules/skvi-engine/SPEC.md`; depends_on -> `knowledge/skvi/SKILL.md`
- consumers: administrators, implementers, reviewers, agentic tools
- deferred_projections: qxctl lifecycle procedure
- notes: Proposal and projection output remain noncanonical.
- status: canonical

#### SKVI Engine SPEC.md
- path: `modules/skvi-engine/SPEC.md`
- title: SKVI Engine Specification
- surface_type: vector-engine module specification
- truth_role: exact operation, bound, exit, install, and non-authorization contract
- owner: SKVI engine maintainers
- scope: Defines inspect, structural check, caller-declared add/replace/remove proposals, disposable JSON projection, and disabled apply.
- relationships: depends_on -> `knowledge/SPEC.md`; implements -> `knowledge/skvi/SPEC.md`; implements -> `knowledge/schemas/v1/proposal.schema.json`; implements -> `knowledge/skvi/schemas/v1/MANIFEST.md`
- consumers: C++ implementers, qxctl, testers, reviewers
- deferred_projections: expanded SKVI-authorized projection formats
- notes: It has no session, authentication, network, SSIAG/STAV, lifecycle, or Maestro authority.
- status: canonical

#### SKVI Engine CMakeLists.txt
- path: `modules/skvi-engine/CMakeLists.txt`
- title: SKVI Engine Build Contract
- surface_type: module build and install contract
- truth_role: static-link, test, package receipt, and uninstall implementation truth
- owner: SKVI engine maintainers
- scope: Builds and tests the exact versioned executable and supports source or installed foundation consumption.
- relationships: implements -> `modules/skvi-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`
- consumers: CMake, implementers, packagers, reviewers
- deferred_projections: reproducible build, install, and receipt evidence
- notes: No global executable alias or active binding is installed.
- status: canonical

### SCLV Engine Module

#### SCLV Engine INTENT.md
- path: `modules/sclv-engine/INTENT.md`
- title: SCLV Engine Intent
- surface_type: vector-engine module intent
- truth_role: subordinate provider-neutral change-check, proposal, recovery, and projection purpose
- owner: SCLV engine maintainers
- scope: Declares deterministic inspect/check/propose/recover/project behavior and bounded evidence normalization without ratification or mutation authority.
- relationships: depends_on -> `knowledge/sclv/INTENT.md`; declares -> `modules/sclv-engine/MANIFEST.md`
- consumers: qxctl, implementers, administrators, reviewers, agentic tools
- deferred_projections: installed-engine inventory and conformance evidence
- notes: The engine implements SCLV truth but does not own or append it.
- status: canonical

#### SCLV Engine MANIFEST.md
- path: `modules/sclv-engine/MANIFEST.md`
- title: SCLV Engine Manifest
- surface_type: independently installable vector-engine manifest
- truth_role: executable, adapter, operation, protocol, dependency, and lifecycle truth
- owner: SCLV engine maintainers
- scope: Declares the C++26 engine, local-Git and air-gapped adapters, eleven-file package, disabled apply, and inactive installed-undocked state.
- relationships: depends_on -> `modules/sclv-engine/INTENT.md`; implements -> `knowledge/sclv/SPEC.md`; statically_links -> `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- consumers: qxctl, packagers, implementers, reviewers, agentic tools
- deferred_projections: engine inventory and Maestro presence evidence
- notes: No default receptor, active alias, canonical append, or journal mutation route exists.
- status: canonical

#### SCLV Engine INSTALL.md
- path: `modules/sclv-engine/INSTALL.md`
- title: SCLV Engine Installation
- surface_type: module installation contract
- truth_role: build, test, versioned install, qxctl invocation, and receipt-owned uninstall procedure
- owner: SCLV engine maintainers
- scope: Defines monorepo and installed-foundation builds plus exact three-executable prefix installation and removal.
- relationships: depends_on -> `modules/sclv-engine/MANIFEST.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/INSTALL.md`; consumed_by -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: implementers, administrators, packagers, qxctl, reviewers
- deferred_projections: lifecycle administration evidence
- notes: Installation leaves the version inactive and undocked and preserves canonical knowledge.
- status: canonical

#### SCLV Engine SKILL.md
- path: `modules/sclv-engine/SKILL.md`
- title: SCLV Engine Skill
- surface_type: vector-engine skill guidance
- truth_role: safe direct and qxctl operation procedure
- owner: SCLV engine maintainers
- scope: Guides ledger checks, provider-evidence normalization, proposal/recovery review, projection use, and stop conditions.
- relationships: depends_on -> `modules/sclv-engine/SPEC.md`; depends_on -> `knowledge/sclv/SKILL.md`
- consumers: administrators, implementers, reviewers, agentic tools
- deferred_projections: qxctl lifecycle procedure
- notes: Provider evidence, proposals, recovery results, and projections remain non-authorizing.
- status: canonical

#### SCLV Engine SPEC.md
- path: `modules/sclv-engine/SPEC.md`
- title: SCLV Engine Specification
- surface_type: vector-engine module specification
- truth_role: exact operation, evidence-adapter, bound, exit, install, and non-authorization contract
- owner: SCLV engine maintainers
- scope: Defines v1/v2/v3 ledger checks, provider-neutral proposals, non-mutating recovery, derived projections, and separate local/air-gapped adapters.
- relationships: depends_on -> `knowledge/SPEC.md`; implements -> `knowledge/sclv/SPEC.md`; implements -> `knowledge/schemas/v1/provider-evidence.schema.json`; implements -> `knowledge/sclv/schemas/v3/MANIFEST.md`
- consumers: C++ implementers, qxctl, testers, reviewers
- deferred_projections: expanded SCLV-authorized projection and evidence formats
- notes: It has no authentication, network, SSIAG/STAV, lifecycle, ratification, append, commit, journal-mutation, or Maestro authority.
- status: canonical

#### SCLV Engine CMakeLists.txt
- path: `modules/sclv-engine/CMakeLists.txt`
- title: SCLV Engine Build Contract
- surface_type: module build and install contract
- truth_role: static-link, multi-executable test, package receipt, and uninstall implementation truth
- owner: SCLV engine maintainers
- scope: Builds and tests the exact engine and adapters and supports source or installed foundation consumption.
- relationships: implements -> `modules/sclv-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`
- consumers: CMake, implementers, packagers, reviewers
- deferred_projections: reproducible build, install, and receipt evidence
- notes: No global executable alias or active binding is installed.
- status: canonical

### SACV Engine Module

#### SACV Engine INTENT.md
- path: `modules/sacv-engine/INTENT.md`
- title: SACV Engine Intent
- surface_type: vector-engine module intent
- truth_role: subordinate API contract check, diff, proposal, and projection purpose
- owner: SACV engine maintainers
- scope: Declares deterministic OpenAPI 3.2.0 governance operations without endpoint, ownership, publication, or mutation authority.
- relationships: depends_on -> `knowledge/sacv/INTENT.md`; declares -> `modules/sacv-engine/MANIFEST.md`
- consumers: qxctl, implementers, administrators, reviewers, agentic tools
- deferred_projections: installed-engine inventory and conformance evidence
- notes: The engine implements SACV truth but does not own it.
- status: canonical

#### SACV Engine MANIFEST.md
- path: `modules/sacv-engine/MANIFEST.md`
- title: SACV Engine Manifest
- surface_type: independently installable vector-engine manifest
- truth_role: executable, operation, protocol, dependency, and lifecycle truth
- owner: SACV engine maintainers
- scope: Declares the C++26 executable, five read/proposal operations, exact package, disabled apply, and inactive installed-undocked state.
- relationships: depends_on -> `modules/sacv-engine/INTENT.md`; implements -> `knowledge/sacv/SPEC.md`; statically_links -> `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- consumers: qxctl, packagers, implementers, reviewers, agentic tools
- deferred_projections: engine inventory and Maestro presence evidence
- notes: No endpoint, default receptor, active alias, canonical write, generator, or publication route exists.
- status: canonical

#### SACV Engine INSTALL.md
- path: `modules/sacv-engine/INSTALL.md`
- title: SACV Engine Installation
- surface_type: module installation contract
- truth_role: build, test, versioned install, qxctl invocation, and receipt-owned uninstall procedure
- owner: SACV engine maintainers
- scope: Defines exact prefix installation of one binary, contracts, receipt, and licenses plus receipt-owned removal.
- relationships: depends_on -> `modules/sacv-engine/MANIFEST.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/INSTALL.md`; consumed_by -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: implementers, administrators, packagers, qxctl, reviewers
- deferred_projections: lifecycle administration evidence
- notes: Installation leaves the version inactive and undocked.
- status: canonical

#### SACV Engine SKILL.md
- path: `modules/sacv-engine/SKILL.md`
- title: SACV Engine Skill
- surface_type: vector-engine skill guidance
- truth_role: safe direct and qxctl operation procedure
- owner: SACV engine maintainers
- scope: Guides JSON checks, YAML fail-closed interpretation, compatibility evidence, proposals, and projections.
- relationships: depends_on -> `modules/sacv-engine/SPEC.md`; depends_on -> `knowledge/sacv/SKILL.md`
- consumers: administrators, implementers, API owners, reviewers, agentic tools
- deferred_projections: qxctl lifecycle procedure
- notes: YAML remains canonical-capable; only this development parser is unavailable.
- status: canonical

#### SACV Engine SPEC.md
- path: `modules/sacv-engine/SPEC.md`
- title: SACV Engine Specification
- surface_type: vector-engine module specification
- truth_role: exact operation, parser, bound, exit, install, and non-authorization contract
- owner: SACV engine maintainers
- scope: Defines inspect/check/diff/propose/project, bounded JSON OpenAPI conformance, YAML fail-closed behavior, and disabled apply.
- relationships: depends_on -> `knowledge/SPEC.md`; implements -> `knowledge/sacv/SPEC.md`; implements -> `knowledge/sacv/schemas/v1/MANIFEST.md`
- consumers: C++ implementers, qxctl, tests, validators, API reviewers
- deferred_projections: independently gated YAML parser and expanded SACV projections
- notes: It has no session, authentication, network, SSIAG/STAV, lifecycle, ownership, endpoint, publication, generator, or Maestro authority.
- status: canonical

#### SACV Engine CMakeLists.txt
- path: `modules/sacv-engine/CMakeLists.txt`
- title: SACV Engine Build Contract
- surface_type: module build and install contract
- truth_role: static-link, test, package receipt, and uninstall implementation truth
- owner: SACV engine maintainers
- scope: Builds and tests the exact versioned executable and supports source or installed foundation consumption.
- relationships: implements -> `modules/sacv-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`
- consumers: CMake, implementers, packagers, reviewers
- deferred_projections: reproducible build, install, and receipt evidence
- notes: No global executable alias or active binding is installed.
- status: canonical

### Knowledge Vector Surfaces

#### Knowledge Root
##### INTENT.md
- path: `knowledge/INTENT.md`
- title: Knowledge Vector Intent
- surface_type: vector intent seed
- truth_role: intent and purpose for knowledge vectors
- owner: knowledge maintainer
- scope: Root definition of SKVI, SCLV, SODV, SACV, SSFV, SSIAG, and STAV vector domains.
- consumers: humans, reviewers, agentic tools, symphony-validator and future validator extensions
- relationships: declares -> `knowledge/MANIFEST.md`; declares -> `knowledge/sacv/INTENT.md`; declares -> `knowledge/ssfv/INTENT.md`; declares -> `knowledge/ssiag/INTENT.md`; declares -> `knowledge/stav/INTENT.md`; checked_by -> `tools/symphony-validator/SPEC.md`
- deferred_projections: vector-authorized JSON/JSONL, search, graph, analytical, and documentation evidence
- notes: Owns the cross-vector engine foundation contract without owning vector-specific semantics.
- status: canonical

##### MANIFEST.md
- path: `knowledge/MANIFEST.md`
- title: Symphony Knowledge Vector Manifest
- surface_type: SKV umbrella manifest
- truth_role: common vector-engine identity, namespace, installability, and authority boundary
- owner: Symphony Knowledge Vector maintainers
- scope: Declares independently installed C++ engines, the implemented shared mechanics/reconciliation/authenticated-session coordinator/SKVI/SCLV/SACV/SODV/SSFV slices, explicit qxctl session transitions and report/apply lifecycle administration, common invariant ownership, generic desired-state convergence, the first partial SSFV catalog, Linux-first delivery, Maestro readiness, and proposal-only canonical-write state.
- relationships: depends_on -> `knowledge/INTENT.md`; declares -> `knowledge/SPEC.md`; declares -> `knowledge/INVARIANTS.md`; governs -> `libraries/knowledge-vector-engine-cpp/`; governs -> `modules/knowledge-session-coordinator/`; governs -> `modules/skvi-engine/`; governs -> `modules/sclv-engine/`; governs -> future cleared vector-engine module paths
- consumers: vector maintainers, engine implementers, qxctl, Maestro planners, reviewers, agentic tools
- deferred_projections: engine inventory, install receipts, Maestro presence graph
- notes: Foundation/coordinator and all five vector-engine `0.1.0-dev` slices plus Maestro presence exist; SSFV has exactly seventy-four experimental records, explicit top-level owner-scope coverage, fifty-nine ratified nested subfeatures, reviewed F2 and F3 non-feature dispositions, and incomplete remaining nested review.
- status: canonical

##### SPEC.md
- path: `knowledge/SPEC.md`
- title: Symphony Knowledge Vector Engine Foundation Specification
- surface_type: SKV umbrella specification
- truth_role: normative cross-vector engine, session, proposal, projection, installation, and isolation contract
- owner: Symphony Knowledge Vector maintainers
- scope: Defines process identifiers, authenticated authority epochs and explicit transitions, worktree reconciliation, canonical desired/observed/plan/applied/boot lifecycle contracts, proposal/apply separation, provider neutrality, qxctl grammar, install receipts, Maestro docking readiness, and hot/warm isolation.
- relationships: depends_on -> `knowledge/MANIFEST.md`; governs -> `knowledge/schemas/v1/MANIFEST.md`; governs -> `knowledge/schemas/v2/MANIFEST.md`; governs -> `libraries/knowledge-vector-engine-cpp/SPEC.md`; governs -> `modules/knowledge-session-coordinator/SPEC.md`; depends_on -> `knowledge/ssiag/SPEC.md`; depends_on -> `knowledge/stav/SPEC.md`
- consumers: C++ engine and coordinator implementers, qxctl, SSIAG/STAV integrators, reviewers, agentic tools
- deferred_projections: apply/provider/docking schemas, conformance evidence, engine inventory, docking graph
- notes: Sixty-six common v1 schemas and four common v2 schemas are canonical; lifecycle profile/runtime persistence, observation, planning, report/apply journal recovery, feature-administration assurance, common invariant ownership, exact staged receipt-v2/runtime/Maestro-presence actions, shared-root ownership fencing, the explicit Linux report-only host receptor, and applied-state commitment are implemented, the seventy-four-record SSFV catalog is partial, and canonical programmatic apply is disabled.
- status: canonical

##### SKILL.md
- path: `knowledge/SKILL.md`
- title: Symphony Knowledge Vector Engine Skill
- surface_type: SKV umbrella skill guidance
- truth_role: safe engine implementation, review, session, and recovery procedure
- owner: Symphony Knowledge Vector maintainers
- scope: Guides proposal-only implementation and records stop conditions for apply, namespaces, external packages, networking, SSFV engine/bootstrap work, and hot/warm isolation.
- relationships: depends_on -> `knowledge/SPEC.md`; governs -> future vector-engine implementation procedure
- consumers: implementers, maintainers, reviewers, qxctl contributors, agentic tools
- deferred_projections: conformance checklist and requirements traceability evidence
- notes: Does not authorize canonical mutation or self-ratification.
- status: canonical

##### LIFECYCLE.md
- path: `knowledge/LIFECYCLE.md`
- title: Symphony Cross-Vector Lifecycle and First-Boot Plan
- surface_type: cross-vector lifecycle architecture contract
- truth_role: explicit session-transition and generic desired-state/first-boot topology truth
- owner: Symphony Knowledge Vector maintainers
- scope: Defines stable host-event convergence, immutable binding-v1 compatibility, generic component identity, receipt-v2 migration, desired/observed/plan/applied separation, dependency-ready-set two-way convergence, evidence-based first boot, durable recovery, module addition/removal behavior, the explicit Linux report-only systemd receptor, and the Maestro boundary.
- relationships: depends_on -> `knowledge/SPEC.md`; depends_on -> `knowledge/TIME.md`; governs -> `knowledge/schemas/v1/lifecycle-profile-input.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-profile.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-desired-state.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-observation.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-plan-command.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-plan.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-applied-state.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-boot-journal.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-boot-head.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-boot-command.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-boot-result.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-host-integration.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-host-integration-result.schema.json`; governs -> `knowledge/schemas/v1/lifecycle-host-boot-result.schema.json`; governs -> `knowledge/schemas/v2/install-receipt.schema.json`; governs -> `tools/qxctl/`; governs -> `modules/knowledge-session-coordinator/`; defers_to -> `knowledge/ssiag/SPEC.md`; defers_to -> `knowledge/stav/SPEC.md`
- consumers: qxctl and coordinator implementers, module packagers, Maestro planners, administrators, reviewers, agentic tools
- deferred_projections: live component process/service adapters, login/session hooks, hidden watchers, native Windows integration, and additional docking operations
- notes: The lifecycle schema family, protected qxctl profile/runtime persistence, fixed-layout observation, report planning, durable report/apply recovery, staged receipt-v2/runtime actions, the explicit Linux report-only systemd receptor, and applied-state commitment are implemented; canonical apply remains disabled.
- status: canonical

##### TIME.md
- path: `knowledge/TIME.md`
- title: Symphony Temporal Semantics Contract
- surface_type: cross-vector common contract
- truth_role: canonical UTC, civil-date, duration, elapsed-time, ordering, and clock-failure semantics
- owner: Symphony Knowledge Vector maintainers
- scope: Defines canonical seconds/nanoseconds profiles, target-host timestamp authority, local-time presentation, monotonic elapsed time, and the separation of wall-clock evidence from causal identity.
- relationships: depends_on -> `knowledge/SPEC.md`; governs -> `knowledge/schemas/v1/temporal.schema.json`; governs -> `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp`; informs -> `knowledge/LIFECYCLE.md`; informs -> `knowledge/ssiag/SPEC.md`; informs -> `knowledge/stav/SPEC.md`
- consumers: all Symphony modules, vector engines, qxctl, validators, reviewers, agentic tools
- deferred_projections: clock-quality attestation, timezone-database governance, and cross-node temporal analysis if separately ratified
- notes: STSC is not a vector and creates no engine, runtime, time service, synchronization authority, or Maestro receptor.
- status: canonical

##### FOUNDATIONAL-LIFECYCLE.md
- path: `knowledge/FOUNDATIONAL-LIFECYCLE.md`
- title: Symphony Foundational Service Lifecycle
- surface_type: cross-vector common lifecycle contract
- truth_role: canonical SSIAG/STAV enrollment, native-supervision, bootstrap-authority, compatibility, attempt, and audit-recovery envelope
- owner: Symphony Knowledge Vector maintainers
- scope: Governs the four exact module-owned SSIAG/STAV lifecycle families without widening generic executable or live-service activation.
- relationships: depends_on -> `knowledge/SPEC.md`; depends_on -> `knowledge/TIME.md`; defers_to -> `knowledge/ssiag/SPEC.md`; defers_to -> `knowledge/stav/SPEC.md`; governs -> `knowledge/schemas/v1/foundation-lifecycle-adapter.schema.json`; governs -> `knowledge/schemas/v1/foundation-lifecycle-command.schema.json`; governs -> `knowledge/schemas/v1/foundation-lifecycle-observation.schema.json`; governs -> `knowledge/schemas/v1/foundation-lifecycle-plan.schema.json`; governs -> `knowledge/schemas/v1/foundation-lifecycle-attempt.schema.json`; governs -> `knowledge/schemas/v1/foundation-lifecycle-result.schema.json`
- consumers: SSIAG and STAV module adapters, qxctl, SSFV administration assurance, reviewers, agentic tools
- deferred_projections: remote target-host transport and separately ratified permanent-retirement handling
- notes: qxctl never becomes the module implementation or a raw STAV producer.
- status: canonical

##### INVARIANTS.md
- path: `knowledge/INVARIANTS.md`
- title: Symphony Common Invariant Ownership
- surface_type: cross-component common ownership contract
- truth_role: canonical lowest-authoritative-layer, stable invariant identity, regression, module-admission, and allowed-adapter truth
- owner: Symphony Knowledge Vector maintainers
- scope: Assigns common invariants to one owner and requires producer regression, consumer boundary rejection, real-process IPC evidence, and explicit new-module administration declarations.
- relationships: depends_on -> `knowledge/SPEC.md`; governs -> `knowledge/INVARIANT-OWNERSHIP.json`; governs -> `knowledge/schemas/v1/invariant-ownership-registry.schema.json`; informs -> `knowledge/FOUNDATIONAL-LIFECYCLE.md`; informs -> `knowledge/FEATURE-ADMINISTRATION.md`
- consumers: module owners, vector engines, qxctl, validators, reviewers, independent developers, agentic tools
- deferred_projections: generated traceability reports, candidate ID suggestions, and command scaffolding
- notes: Stable IDs are ratified identifiers; C++, qxctl, validators, and AI do not invent invariant or feature semantics.
- status: canonical

##### Invariant Ownership Registry
- path: `knowledge/INVARIANT-OWNERSHIP.json`
- title: Symphony Common Invariant Ownership Registry v1
- surface_type: canonical machine-readable JSON registry
- truth_role: exact invariant-to-owner, implementation, producer-test, consumer-test, real-process-test, and adapter routing truth
- owner: Symphony Knowledge Vector maintainers
- scope: Registers an incremental forward-gated set of module-administration admission and foundational lifecycle invariants with exact evidence paths and the two allowed receipt-v2 lifecycle adapters without claiming complete legacy invariant coverage.
- relationships: governed_by -> `knowledge/INVARIANTS.md`; conforms_to -> `knowledge/schemas/v1/invariant-ownership-registry.schema.json`; references -> `knowledge/FOUNDATIONAL-LIFECYCLE.md`; references -> `knowledge/FEATURE-ADMINISTRATION.md`
- consumers: module owners, validators, CI, reviewers, agentic tools
- deferred_projections: generated requirements traceability and execution reports
- notes: Digest validity proves canonical registry identity but not test execution; every referenced path and named case must also resolve and pass.
- status: canonical

##### Common v1 Schema Manifest
- path: `knowledge/schemas/v1/MANIFEST.md`
- title: Symphony Knowledge Vector Common Schemas v1
- surface_type: common protocol schema manifest
- truth_role: canonical inventory and boundary for exact common JSON schemas
- owner: Symphony Knowledge Vector maintainers
- scope: Declares sixty-six exact process, descriptor, install-receipt-v1, engine-binding, proposal, provider-evidence, reconciliation/session, profile-input/profile, generic and foundational lifecycle, feature-administration, invariant-ownership/query, qxctl-registry, temporal, validation warning-state, and Maestro presence schemas.
- relationships: depends_on -> `knowledge/SPEC.md`; governs -> `libraries/knowledge-vector-engine-cpp/SPEC.md`; governs -> `modules/knowledge-session-coordinator/SPEC.md`; governs -> `modules/skvi-engine/SPEC.md`; governs -> `modules/sclv-engine/SPEC.md`
- consumers: C++ foundation and engine implementers, qxctl planners, validator, reviewers
- deferred_projections: generated schema documentation and conformance evidence
- notes: Operation-specific payload/result schemas remain with the applicable coordinator or vector.
- status: canonical

##### Invariant Ownership Registry Schema
- path: `knowledge/schemas/v1/invariant-ownership-registry.schema.json`
- title: Symphony Common Invariant Ownership Registry v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical closed machine shape for lowest-authoritative-layer invariant ownership and regression routing
- owner: Symphony Knowledge Vector maintainers
- scope: Closes invariant and adapter identifiers, owner links, implementation paths, producer and consumer regression references, IPC classification, and mandatory real-process IPC evidence.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/INVARIANTS.md`; governs -> `knowledge/INVARIANT-OWNERSHIP.json`
- consumers: module owners, validators, CI, reviewers, agentic tools
- deferred_projections: referential and test-execution result schemas
- notes: JSON Schema enforces structure; repository validation must additionally enforce ordering, digest, referential integrity, path existence, and named-test execution.
- status: canonical

##### Invariant Query Result Schema
- path: `knowledge/schemas/v1/invariant-query-result.schema.json`
- title: Symphony Invariant Ownership Query Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical closed status, list, and show result shape for read-only invariant administration
- owner: Symphony Knowledge Vector maintainers
- scope: Closes digest-bearing qxctl projections over the canonical registry while explicitly reserving semantic validity for complete validator assurance.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/INVARIANTS.md`; consumes -> `knowledge/INVARIANT-OWNERSHIP.json`
- consumers: qxctl, headless administrators, reviewers, agentic tools
- deferred_projections: remote invariant inventory transport
- notes: A valid query result proves bounded consumer-side identity, shape, ordering, and digest checks; it does not claim complete semantic validation or test execution.
- status: canonical

##### Common v2 Schema Manifest
- path: `knowledge/schemas/v2/MANIFEST.md`
- title: Symphony Knowledge Vector Common Schemas v2
- surface_type: common protocol schema manifest
- truth_role: canonical inventory and compatibility boundary for exact common v2 JSON schemas
- owner: Symphony Knowledge Vector maintainers
- scope: Declares immutable content-addressed install receipt v2 plus side-by-side apply-journal/head v2 while preserving exact receipt and report-journal v1 evidence and adapters.
- relationships: depends_on -> `knowledge/SPEC.md`; governs -> `knowledge/schemas/v2/install-receipt.schema.json`; governs -> `knowledge/schemas/v2/lifecycle-boot-journal.schema.json`; governs -> `knowledge/schemas/v2/lifecycle-boot-head.schema.json`
- consumers: lifecycle implementers, qxctl planners, packagers, validator, reviewers
- deferred_projections: generated schema documentation and conformance evidence
- notes: Version 2 never authorizes rewriting or guessing fields for a version 1 receipt.
- status: canonical

##### Foundational Lifecycle Adapter Schema
- path: `knowledge/schemas/v1/foundation-lifecycle-adapter.schema.json`
- title: Foundational Lifecycle Adapter Descriptor v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: exact installed SSIAG/STAV adapter identity, capability, compatibility, and limit truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes the fixed module-owned adapter descriptor without authorizing arbitrary executables.
- relationships: governed_by -> `knowledge/FOUNDATIONAL-LIFECYCLE.md`; depends_on -> `knowledge/schemas/v1/MANIFEST.md`
- consumers: SSIAG, STAV, qxctl, SSFV administration assurance
- deferred_projections: separately ratified remote transport capabilities
- notes: The descriptor recognizes only receipt-owned SSIAG and STAV adapters and grants no mutation authority.
- status: canonical

##### Foundational Lifecycle Command Schema
- path: `knowledge/schemas/v1/foundation-lifecycle-command.schema.json`
- title: Foundational Lifecycle Command v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: exact bounded observe, plan, apply, apply-status, and recovery input truth
- owner: Symphony Knowledge Vector maintainers
- scope: Binds caller intent, exact expected state, immutable plan, attempt state, and deadline.
- relationships: governed_by -> `knowledge/FOUNDATIONAL-LIFECYCLE.md`; depends_on -> `knowledge/schemas/v1/foundation-lifecycle-plan.schema.json`
- consumers: SSIAG and STAV adapters, qxctl
- deferred_projections: remote target-host transport envelopes
- notes: The operation-specific shape closes omitted and surplus mutation evidence before adapter execution.
- status: canonical

##### Foundational Lifecycle Observation Schema
- path: `knowledge/schemas/v1/foundation-lifecycle-observation.schema.json`
- title: Foundational Lifecycle Observation v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: safe installed, enrollment, supervisor, endpoint, activation, and recovery evidence
- owner: Symphony Knowledge Vector maintainers
- scope: Separates stable state identity from timestamped observation evidence.
- relationships: governed_by -> `knowledge/FOUNDATIONAL-LIFECYCLE.md`; conforms_to -> `knowledge/TIME.md`
- consumers: SSIAG and STAV adapters, qxctl, recovery tooling
- deferred_projections: cross-node observation aggregation
- notes: Stable-state identity excludes only documentary time and digest fields while retaining recovery evidence.
- status: canonical

##### Foundational Lifecycle Plan Schema
- path: `knowledge/schemas/v1/foundation-lifecycle-plan.schema.json`
- title: Foundational Lifecycle Plan v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: immutable expected-state and desired-state proposal truth
- owner: Symphony Knowledge Vector maintainers
- scope: Carries domain identity inputs, explicit audit mode, expiry, and self-digest.
- relationships: governed_by -> `knowledge/FOUNDATIONAL-LIFECYCLE.md`; conforms_to -> `knowledge/TIME.md`
- consumers: SSIAG and STAV adapters, qxctl
- deferred_projections: future domain intents introduced through a compatible schema version
- notes: Expiry and wall-clock evidence never select package versions or recovery direction.
- status: canonical

##### Foundational Lifecycle Attempt Schema
- path: `knowledge/schemas/v1/foundation-lifecycle-attempt.schema.json`
- title: Foundational Lifecycle Attempt v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: protected pre-mutation phase, predecessor, audit, and forward-recovery evidence
- owner: Symphony Knowledge Vector maintainers
- scope: Prevents unjournaled host mutation and ambiguous recovery.
- relationships: governed_by -> `knowledge/FOUNDATIONAL-LIFECYCLE.md`; conforms_to -> `knowledge/TIME.md`
- consumers: SSIAG and STAV transaction engines, qxctl status and recovery
- deferred_projections: a separately ratified permanent-retirement record
- notes: Protected attempt evidence remains outside any module subtree the recorded operation may remove.
- status: canonical

##### Foundational Lifecycle Result Schema
- path: `knowledge/schemas/v1/foundation-lifecycle-result.schema.json`
- title: Foundational Lifecycle Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: validated observation, proposal, mutation, replay, recovery, and audit-disposition evidence
- owner: Symphony Knowledge Vector maintainers
- scope: Closes every qxctl-visible foundational lifecycle outcome with exact digests.
- relationships: governed_by -> `knowledge/FOUNDATIONAL-LIFECYCLE.md`; depends_on -> `knowledge/schemas/v1/foundation-lifecycle-observation.schema.json`; depends_on -> `knowledge/schemas/v1/foundation-lifecycle-plan.schema.json`
- consumers: SSIAG and STAV adapters, qxctl, SSFV administration assurance
- deferred_projections: remote presentation envelopes that preserve this result unchanged
- notes: Results are noncanonical administration evidence and cannot substitute for a STAV audit receipt.
- status: canonical

##### Feature Administration Assurance Contract
- path: `knowledge/FEATURE-ADMINISTRATION.md`
- title: Symphony Feature Administration Assurance Contract
- surface_type: cross-vector common contract
- truth_role: canonical engine-first administrative coverage, identity, exception, admission, and headless-machine semantics
- owner: Symphony Knowledge Vector maintainers
- scope: Defines stable feature, engine-operation, and qxctl-command identities; expected versus observed truth; three-axis coverage; explicit dispositions; third-party module admission; remediation evidence; bounded-envelope capacity posture; and the seventy-four-record forward gate.
- relationships: depends_on -> `knowledge/SPEC.md`; depends_on -> `knowledge/INVARIANTS.md`; depends_on -> `knowledge/ssfv/SPEC.md`; governs -> `knowledge/FEATURE-ADMINISTRATION-PROFILE.md`; governs -> `knowledge/schemas/v1/feature-administration-profile.schema.json`; governs -> `knowledge/schemas/v1/qxctl-command-registry.schema.json`; governs -> `knowledge/schemas/v1/administration-coverage-input.schema.json`; governs -> `knowledge/schemas/v1/administration-coverage-result.schema.json`; governs -> `knowledge/schemas/v2/engine-descriptor.schema.json`
- consumers: independent module developers, SSFV engine, qxctl, validators, reviewers, agentic tools
- deferred_projections: AI-assisted names, command-design proposals, Cobra scaffolding, and enforce-all-records coverage
- notes: It is an umbrella contract, not a vector, authority service, installation format, or dynamic command-extension mechanism.
- status: canonical

##### Feature Administration Bootstrap Profile
- path: `knowledge/FEATURE-ADMINISTRATION-PROFILE.md`
- title: Symphony Feature Administration Profile
- surface_type: cross-vector canonical bootstrap profile
- truth_role: exact registered-partial-catalog administration review and forward-gate policy
- owner: Symphony Knowledge Vector maintainers
- scope: Requires explicit representation of all seventy-four current SSFV records, treats unreviewed interactions as debt, and defines the active forward-enforcement gate without claiming catalog completeness.
- relationships: governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`; derives_from -> `knowledge/ssfv/REGISTRY.md`; declares -> `knowledge/FEATURE-ADMINISTRATION-PROFILE.json`; conforms_to -> `knowledge/schemas/v1/feature-administration-profile.schema.json`
- consumers: SSFV engine, qxctl, validators, maintainers, reviewers, agentic tools
- deferred_projections: future complete-catalog adjudication and installed-host inventory closure
- notes: Absence never means not applicable, and inheritance must be finite and acyclic.
- status: canonical

##### Feature Administration Machine Profile
- path: `knowledge/FEATURE-ADMINISTRATION-PROFILE.json`
- title: Symphony Feature Administration Machine Profile v1
- surface_type: checked-in canonical-derived JSON profile
- truth_role: machine-evaluable report-only bootstrap coverage for the exact current registered SSFV set
- owner: Symphony Knowledge Vector maintainers
- scope: Represents all seventy-four current feature IDs exactly once with 145 reviewed expectations under `enforce_new_records`.
- relationships: governed_by -> `knowledge/FEATURE-ADMINISTRATION-PROFILE.md`; derives_from -> `knowledge/ssfv/REGISTRY.md`; conforms_to -> `knowledge/schemas/v1/feature-administration-profile.schema.json`
- consumers: SSFV engine, qxctl, validators, direct diagnostic callers, reviewers
- deferred_projections: complete-catalog and installed-host inventory advancement
- notes: The profile is digest-bound and forward-gated but makes no repository- or installed-host-completeness claim.
- status: canonical

##### qxctl Command Registry Schema
- path: `knowledge/schemas/v1/qxctl-command-registry.schema.json`
- title: qxctl Command Registry v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical expected and exact observed stable command identity and machine-capability shape
- owner: Symphony Knowledge Vector and qxctl maintainers
- scope: Closes command lifecycle, grammar, aliases, visibility, feature and backend bindings, protocols, result validation, recovery, trust, and honest noninteractive/JSON support evidence.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`; expected_from -> `tools/qxctl/COMMANDS.json`
- consumers: qxctl, SSFV engine, validators, module developers, agentic tools
- deferred_projections: generated help and command-design proposals
- notes: The canonical expected registry remains evaluable when an observed qxctl binary is absent.
- status: canonical

##### qxctl Expected Command Registry
- path: `tools/qxctl/COMMANDS.json`
- title: qxctl Expected Command Registry
- surface_type: checked-in canonical-derived JSON registry
- truth_role: client-independent expected stable qxctl leaf identity, grammar, feature-binding, backend-binding, machine-output, and recovery evidence
- owner: qxctl maintainers under the cross-vector feature-administration contract
- scope: Projects the exact Go CommandSpec and Cobra command tree into a digest-bound expected registry that remains usable when no qxctl executable is present.
- relationships: governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`; conforms_to -> `knowledge/schemas/v1/qxctl-command-registry.schema.json`; implemented_by -> `tools/qxctl/internal/commandregistry/registry.go`
- consumers: SSFV engine, Symphony Validator, qxctl parity verification, module developers, reviewers, agentic tools
- deferred_projections: reviewed command scaffolding and AI-assisted command-design proposals
- notes: The registry is not a second handwritten grammar source; deterministic qxctl generation and validation must reproduce it exactly.
- status: canonical

##### Feature Administration Profile Schema
- path: `knowledge/schemas/v1/feature-administration-profile.schema.json`
- title: Symphony Feature Administration Profile v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical normalized administration expectation and disposition shape
- owner: Symphony Knowledge Vector maintainers
- scope: Closes registered feature interactions, requirement and delivery enums, stable command and engine-operation mappings, explicit inheritance, rationale, evidence, and forward-gate state.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`; derives_from -> `knowledge/FEATURE-ADMINISTRATION-PROFILE.md`
- consumers: SSFV engine, qxctl, validators, reviewers
- deferred_projections: complete-catalog normalized profile
- notes: Cross-record uniqueness and finite acyclic inheritance are semantic conformance requirements.
- status: canonical

##### Administration Coverage Input Schema
- path: `knowledge/schemas/v1/administration-coverage-input.schema.json`
- title: Symphony Administration Coverage Input v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical bounded repository-independent engine input shape
- owner: Symphony Knowledge Vector maintainers
- scope: Supplies exact SSFV snapshot, profile, expected registry, optional observed qxctl registry, zero or more engine descriptors, and an optional feature filter without assuming repository or qxctl presence.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`; consumes -> `knowledge/ssfv/schemas/v1/semantic-snapshot.schema.json`; consumes -> `knowledge/schemas/v2/engine-descriptor.schema.json`
- consumers: SSFV engine, qxctl, direct diagnostic callers, tests
- deferred_projections: installed-host coverage collection
- notes: All discovery and canonical snapshots are caller-supplied and bounded.
- status: canonical

##### Administration Coverage Result Schema
- path: `knowledge/schemas/v1/administration-coverage-result.schema.json`
- title: Symphony Administration Coverage Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic uncovered-surface and integration evidence shape
- owner: Symphony Knowledge Vector maintainers
- scope: Separates feature-level unresolved findings plus interaction-specific design, live, and authorization results; binds every input digest; records module integration without conflating installation; and emits proposal-only remediation constraints.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`
- consumers: SSFV engine, qxctl, validators, module developers, reviewers, agentic tools
- deferred_projections: reviewed command designs and implementation worklists
- notes: A finding cannot invent semantic names, grant authority, or mutate canonical files.
- status: canonical

##### Engine Descriptor v2 Schema
- path: `knowledge/schemas/v2/engine-descriptor.schema.json`
- title: Symphony Knowledge Engine Descriptor v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical side-by-side engine operation identity and administration descriptor shape
- owner: Symphony Knowledge Vector maintainers
- scope: Adds stable engine-operation IDs, feature mappings, administrative interactions, disposition, protocols, mutability, idempotency, expected state, authority, recovery, direct invocation, and thermal path without carrying installation state.
- relationships: depends_on -> `knowledge/schemas/v2/MANIFEST.md`; governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`; coexists_with -> `knowledge/schemas/v1/engine-descriptor.schema.json`
- consumers: C++ engines, SSFV coverage checking, qxctl, packagers, validators
- deferred_projections: exact installed-engine operation inventories
- notes: Descriptor v1 remains exact compatible evidence and is never silently upgraded or rewritten.
- status: canonical

##### Lifecycle Desired-State Schema
- path: `knowledge/schemas/v1/lifecycle-desired-state.schema.json`
- title: Symphony Lifecycle Desired State v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical protected noncanonical component-intent truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes exact component/package selection, presence, install scope/root, activation, docking, explicit dependencies, compatibility, extensions, and digest continuity.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl desired-profile administrator, coordinator lifecycle planner, validator, reviewers
- deferred_projections: policy-profile administration and applied-state comparison
- notes: Desired state expresses intent and carries no mutation authority.
- status: canonical

##### Lifecycle Profile Input Schema
- path: `knowledge/schemas/v1/lifecycle-profile-input.schema.json`
- title: Symphony Lifecycle Profile Input v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical bounded declarative lifecycle-profile intent truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes caller-supplied TOPS/profile identity, configured roots, boot mode, component intent, dependencies, compatibility, and extensions without generated state fields.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/LIFECYCLE.md`; implemented_by -> `tools/qxctl/internal/knowledgelifecycle/profile.go`
- consumers: qxctl lifecycle profile administration, validators, administrators, reviewers
- deferred_projections: generated profile examples and schema documentation
- notes: Callers declare intent; qxctl generates generation, continuity, and content digests.
- status: canonical

##### Lifecycle Profile Schema
- path: `knowledge/schemas/v1/lifecycle-profile.schema.json`
- title: Symphony Protected Lifecycle Profile v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical protected noncanonical profile-state contract
- owner: Symphony Knowledge Vector maintainers
- scope: Closes per-TOPS selected roots, boot mode, linked generation and profile digest, plus the exact generated desired-state document.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-profile-input.schema.json`; depends_on -> `knowledge/schemas/v1/lifecycle-desired-state.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; implemented_by -> `tools/qxctl/internal/knowledgelifecycle/profile.go`
- consumers: qxctl lifecycle administration/report, validators, administrators, reviewers
- deferred_projections: profile inventory and first-boot selection evidence
- notes: Protected profile state is noncanonical, caller-neutral, and never contains credentials.
- status: canonical

##### Lifecycle Observation Schema
- path: `knowledge/schemas/v1/lifecycle-observation.schema.json`
- title: Symphony Lifecycle Observation v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical disposable observed-installation evidence truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes configured roots, normalized platform compatibility, binding evidence, package integrity, selected component and exact docked-receptor state, capabilities, and unknown preserved receipts.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/LIFECYCLE.md`; reads -> `knowledge/schemas/v1/install-receipt.schema.json`; reads -> `knowledge/schemas/v2/install-receipt.schema.json`
- consumers: qxctl configured-root inventory collector, coordinator lifecycle planner, validator, reviewers
- deferred_projections: boot-journal observation checkpoints and Maestro presence adapters
- notes: Observation never executes discovered code and is rebuildable from validated evidence.
- status: canonical

##### Lifecycle Plan Command Schema
- path: `knowledge/schemas/v1/lifecycle-plan-command.schema.json`
- title: Symphony Report-Only Lifecycle Plan Command v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical exact caller-to-coordinator lifecycle planning request truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes supplied desired and observation evidence, optional prior applied-state anchor, process/schema/receipt reader versions, and named two-way planner capabilities.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-desired-state.schema.json`; depends_on -> `knowledge/schemas/v1/lifecycle-observation.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; implemented_by -> `modules/knowledge-session-coordinator/src/lifecycle.cpp`
- consumers: coordinator lifecycle planner, qxctl lifecycle client, conformance tests, validator, reviewers
- deferred_projections: rendered protocol documentation
- notes: The command carries evidence and compatibility claims, not authorization or apply permission.
- status: canonical

##### Lifecycle Plan Schema
- path: `knowledge/schemas/v1/lifecycle-plan.schema.json`
- title: Symphony Dependency-Driven Lifecycle Plan v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic two-way convergence-plan truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes dependency-ready-set scheduling, forward/inverse action relationships, stable action IDs, exact target-state and receptor identities, ordered safety phases, critical dependency blockers, noncritical advisories, cycles, bounded plan revisions, and disabled apply.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-desired-state.schema.json`; depends_on -> `knowledge/schemas/v1/lifecycle-observation.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; emitted_by -> `modules/knowledge-session-coordinator/src/lifecycle.cpp`
- consumers: C++ coordinator lifecycle planner, qxctl reporting, validator, reviewers
- deferred_projections: plan visualization and durable boot-journal execution
- notes: Dynamic action order cannot bypass authorization, integrity, compare-and-swap, verification, or audit.
- status: canonical

##### Lifecycle Applied-State Schema
- path: `knowledge/schemas/v1/lifecycle-applied-state.schema.json`
- title: Symphony Lifecycle Applied State v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical durable noncanonical last-convergence evidence truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes the stabilized observation key, exact input digests, component outcomes including exact docked receptor identity, actual execution order, unresolved blockers, extensions, and applied-state digest continuity.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-plan.schema.json`; depends_on -> `knowledge/schemas/v1/lifecycle-observation.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; persisted_by -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`
- consumers: coordinator lifecycle recovery, qxctl apply status, validator, reviewers
- deferred_projections: applied-state operator projection
- notes: Applied state advances only after verified convergence and is not canonical knowledge.
- status: canonical

##### Lifecycle Runtime-State Schema
- path: `knowledge/schemas/v1/lifecycle-runtime-state.schema.json`
- title: Symphony Lifecycle Runtime State v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical protected noncanonical generic selection and activation truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes linked per-TOPS/profile component selection, administrative active/inactive state, undocked boundary, component digests, and whole-document digest continuity.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-desired-state.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; implemented_by -> `tools/qxctl/internal/knowledgelifecycle/runtime.go`
- consumers: qxctl lifecycle executor, configured-root observation, coordinator plan verification, validator, reviewers
- deferred_projections: live service/process activation and Maestro presence adapters
- notes: Active means generic administrative eligibility; it does not prove process execution, engine binding, entry-point invocation, or docking.
- status: canonical

##### Lifecycle Boot-Journal Schema
- path: `knowledge/schemas/v1/lifecycle-boot-journal.schema.json`
- title: Symphony Lifecycle Boot Journal v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical durable transaction, dynamic-replan, blocker, and recovery-evidence truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes transaction identity, exact desired/observation/plan/applied anchors, bounded action attempts and plan revisions, blockers, checkpoint chain, compatibility, recovery, and disabled apply.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-plan.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: C++ coordinator lifecycle journal, qxctl recovery, validator, reviewers
- deferred_projections: applied-state/action execution runtime
- notes: The journal may record a changed action order but cannot rewrite prior attempt evidence.
- status: canonical

##### Lifecycle Boot-Head Schema
- path: `knowledge/schemas/v1/lifecycle-boot-head.schema.json`
- title: Symphony Lifecycle Boot Journal Head v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical active-slot selector and atomic lifecycle continuity truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes profile/TOPS/transaction identity, active slot, generation, journal digest, prior-head digest, and head digest.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-boot-journal.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: coordinator lifecycle recovery, validator, reviewers
- deferred_projections: rendered durability documentation
- notes: Journal slots remain recovery evidence; the head is replaceable operational evidence.
- status: canonical

##### Lifecycle Boot Command Schema
- path: `knowledge/schemas/v1/lifecycle-boot-command.schema.json`
- title: Symphony Durable Report-Only Lifecycle Boot Command v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical SSIAG-authorized lifecycle boot/status/recovery request truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes protected state root, exact operation and expected journal state, TOPS/profile identity and profile digest, desired/observed/stable-inventory evidence, SSIAG decision, and planner/journal compatibility declarations.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-desired-state.schema.json`; depends_on -> `knowledge/schemas/v1/lifecycle-observation.schema.json`; depends_on -> `knowledge/ssiag/schemas/v1/authorization-decision.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; implemented_by -> `tools/qxctl/cmd/qxctl/lifecycle.go`; implemented_by -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`
- consumers: qxctl lifecycle client, C++ coordinator, conformance tests, validator, reviewers
- deferred_projections: rendered protocol documentation
- notes: The command authorizes only protected report-only journal administration and never lifecycle action execution.
- status: canonical

##### Lifecycle Boot Result Schema
- path: `knowledge/schemas/v1/lifecycle-boot-result.schema.json`
- title: Symphony Durable Report-Only Lifecycle Boot Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical protected lifecycle journal/status/recovery response truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes compatibility, optional journal and plan, exact digest, mutation/recovery flags, bounded repair evidence, and disabled apply/canonical assertions.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-boot-journal.schema.json`; depends_on -> `knowledge/schemas/v1/lifecycle-plan.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; emitted_by -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`; implemented_by -> `tools/qxctl/cmd/qxctl/lifecycle.go`
- consumers: qxctl lifecycle client, conformance tests, validator, reviewers
- deferred_projections: rendered protocol documentation
- notes: A present journal must carry its exact digest; status is read-only and no result may authorize apply.
- status: canonical

##### Lifecycle Apply Command Schema
- path: `knowledge/schemas/v1/lifecycle-apply-command.schema.json`
- title: Symphony Lifecycle Apply Command v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical exact qxctl-to-coordinator apply coordination request truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes prepare/finalize/close/status/recovery operation identity, exact source-report/apply/applied compare-and-swap, desired and observation evidence, artifact digests, execution outcome, SSIAG decision, and v2 compatibility declaration.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-plan-command.schema.json`; depends_on -> `knowledge/ssiag/schemas/v1/authorization-decision.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; implemented_by -> `tools/qxctl/cmd/qxctl/lifecycle_apply.go`; implemented_by -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`
- consumers: qxctl lifecycle client, C++ coordinator, conformance tests, validator, reviewers
- deferred_projections: rendered protocol documentation
- notes: It authorizes only the exact noncanonical lifecycle phase; it cannot authorize canonical mutation or bypass evidence verification.
- status: canonical

##### Lifecycle Apply Result Schema
- path: `knowledge/schemas/v1/lifecycle-apply-result.schema.json`
- title: Symphony Lifecycle Apply Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical protected apply compatibility, state, action, applied-evidence, and recovery result truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes v2 compatibility, optional journal/plan/action/applied state, exact digest, mutation/recovery flags, repair evidence, read-only status, and noncanonical authority assertions.
- relationships: depends_on -> `knowledge/schemas/v2/lifecycle-boot-journal.schema.json`; depends_on -> `knowledge/schemas/v1/lifecycle-applied-state.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; emitted_by -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`; implemented_by -> `tools/qxctl/cmd/qxctl/lifecycle_apply.go`
- consumers: qxctl lifecycle client, conformance tests, validator, reviewers
- deferred_projections: rendered protocol documentation
- notes: Status is read-only; mutation authorization is exact-operation evidence and never canonical apply authority.
- status: canonical

##### Lifecycle Host Integration Schema
- path: `knowledge/schemas/v1/lifecycle-host-integration.schema.json`
- title: Symphony Linux Lifecycle Host Integration v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical protected systemd receptor descriptor truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes TOPS/profile/systemd identity, report-only and recovery posture, exact roots, unit digest, active and bounded fallback executor set, enablement, generation, predecessor, STSC timestamp, and descriptor digest.
- relationships: depends_on -> `knowledge/schemas/v1/temporal.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; implemented_by -> `tools/qxctl/internal/knowledgelifecycle/host.go`
- consumers: qxctl host administration, validator, tests, reviewers
- deferred_projections: compatible descriptor v2
- notes: The accepted executor sequence is explicit transition evidence and never semantic-version precedence.
- status: canonical

##### Lifecycle Host Integration Result Schema
- path: `knowledge/schemas/v1/lifecycle-host-integration-result.schema.json`
- title: Symphony Lifecycle Host Integration Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical noncanonical host administration result truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes operation identity, optional descriptor, drift and repair evidence, changed/recovered flags, and disabled apply/canonical assertions.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-host-integration.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; emitted_by -> `tools/qxctl/cmd/qxctl/lifecycle_host.go`
- consumers: qxctl callers, automation, validator, tests, reviewers
- deferred_projections: rendered host status documentation
- notes: A result reports or reconciles the receptor and grants no desired-state apply authority.
- status: canonical

##### Lifecycle Host Boot Result Schema
- path: `knowledge/schemas/v1/lifecycle-host-boot-result.schema.json`
- title: Symphony Lifecycle Host Boot Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical kernel-boot dispatch result truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes TOPS/profile and Linux boot UUID identity, exact report-journal digest, changed/replayed disposition, and disabled apply/canonical assertions.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-boot-result.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; emitted_by -> `tools/qxctl/cmd/qxctl/lifecycle_host.go`
- consumers: systemd, qxctl callers, automation, validator, tests, reviewers
- deferred_projections: boot-history projection from protected journal evidence
- notes: Wall-clock time is absent from boot identity; one kernel boot UUID replays without another plan revision.
- status: canonical

##### Temporal Semantics Schema
- path: `knowledge/schemas/v1/temporal.schema.json`
- title: Symphony Temporal Semantics Definitions v1
- surface_type: JSON Schema Draft 2020-12 definition library
- truth_role: canonical structural temporal-encoding truth
- owner: Symphony Knowledge Vector maintainers
- scope: Defines reusable closed structural profiles for civil dates, whole-second UTC, and exact-nine-digit nanosecond UTC.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; governed_by -> `knowledge/TIME.md`; implemented_by -> `libraries/knowledge-vector-engine-cpp/src/temporal.cpp`
- consumers: schema authors, C++ and Go implementers, validators, reviewers
- deferred_projections: generated schema documentation and cross-language conformance corpus
- notes: Regex and date-time format checks do not replace real Gregorian implementation validation.
- status: canonical

##### Install Receipt Schema v2
- path: `knowledge/schemas/v2/install-receipt.schema.json`
- title: Symphony Immutable Content-Addressed Install Receipt v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical immutable package ownership, integrity, capability, and compatibility truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes component/package identity, exact owned-file sizes/digests/kinds, reserved control/receipt namespaces, explicit entry points, provided/required capabilities, compatible receptors, platform requirements, and receipt digest.
- relationships: depends_on -> `knowledge/schemas/v2/MANIFEST.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: packagers, qxctl inventory/executor, coordinator lifecycle planner, installers, uninstallers, validator, reviewers
- deferred_projections: package-manager-specific receipt emission and additional strict v1 adapters
- notes: Mutable activation, selection, and docking state are deliberately absent; discovery never authorizes execution.
- status: canonical

##### Shared Install Receipt v2 Registration
- path: `cmake/SymphonyInstallReceiptV2.cmake`
- title: Symphony Shared Install Receipt v2 Registration
- surface_type: common CMake packaging implementation
- truth_role: deterministic receipt-v2 installation-order and package-registration truth
- owner: Symphony Knowledge Vector maintainers
- scope: Registers an existing-receipt preflight before owned-file install rules and receipt-v2 generation after every other package install rule; rejects reserved lifecycle/receipt ownership paths; and binds exact package identity, entry points, capabilities, receptors, and platform requirements.
- relationships: implements -> `knowledge/schemas/v2/install-receipt.schema.json`; configures -> `cmake/SymphonyInstallReceiptV2Preflight.cmake.in`, `cmake/SymphonyInstallReceiptV2.cmake.in`; copies_for_build_local_use -> `cmake/SymphonyUninstallReceiptV2.cmake`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: independently installable C++ foundation, coordinator, Maestro, vector-engine, and validator packages
- deferred_projections: package-manager-native manifest adapters
- notes: The shared registration is packaging mechanics only; it selects, activates, docks, or executes no package.
- status: canonical

##### Shared Install Receipt v2 Generator
- path: `cmake/SymphonyInstallReceiptV2.cmake.in`
- title: Symphony Shared Install Receipt v2 Generator
- surface_type: configured CMake install script
- truth_role: exact installed-file inventory, digest, ownership, and intrinsic receipt-digest implementation truth
- owner: Symphony Knowledge Vector maintainers
- scope: Validates the declared completed installation without following links, records every owned non-receipt path with size and SHA-256, rejects an existing final or temporary receipt target, and emits the immutable content-addressed receipt last.
- relationships: implements -> `knowledge/schemas/v2/install-receipt.schema.json`; configured_by -> `cmake/SymphonyInstallReceiptV2.cmake`; validated_by -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: CMake installers and uninstallers, qxctl lifecycle observation and binding, coordinator planning, validator, reviewers
- deferred_projections: signed package attestations and platform package-manager receipts
- notes: The receipt digest is intrinsic canonical JSON content; the receipt does not include itself in its owned-file set.
- status: canonical

##### Shared Install Receipt v2 Preflight
- path: `cmake/SymphonyInstallReceiptV2Preflight.cmake.in`
- title: Symphony Shared Install Receipt v2 Preflight
- surface_type: configured CMake install script
- truth_role: exact-version overwrite prevention and receipt immutability implementation truth
- owner: Symphony Knowledge Vector maintainers
- scope: Runs before any package-owned file install rule and rejects a final or dangling-link receipt already committed at the exact module/version path or any direct install into a root carrying the qxctl registry or compatibility fence.
- relationships: implements -> `knowledge/schemas/v2/install-receipt.schema.json`; configured_by -> `cmake/SymphonyInstallReceiptV2.cmake`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: independently installable C++ foundation, coordinator, Maestro, vector-engine, and validator packages
- deferred_projections: package-manager-native pre-transaction guards
- notes: An existing exact version must be explicitly uninstalled or a different side-by-side version selected; direct CMake install cannot silently replace committed receipt-v2 ownership evidence or race qxctl root administration.
- status: canonical

##### Shared Install Receipt v2 Uninstall Guard
- path: `cmake/SymphonyUninstallReceiptV2.cmake`
- title: Symphony Shared Install Receipt v2 Uninstall Guard
- surface_type: common CMake lifecycle implementation
- truth_role: receipt-set validation, content-integrity preflight, receipt-last removal, and idempotent retry truth
- owner: Symphony Knowledge Vector maintainers
- scope: Requires the generated package's configured path set to equal its receipt-v2 owned set, validates every remaining no-follow regular file against its recorded size and SHA-256 before mutation, refuses a present package beneath a root carrying the qxctl registry or compatibility fence, treats already-missing files as resumable prior work, and removes the receipt last.
- relationships: implements -> `knowledge/schemas/v2/install-receipt.schema.json`; copied_by -> `cmake/SymphonyInstallReceiptV2.cmake`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: build-local uninstall programs for the C++ foundation, coordinator, Maestro, vector engines, and Symphony Validator
- deferred_projections: package-manager-native transactional uninstallers
- notes: An absent receipt is accepted only when every configured owned path is also absent; content drift, links, directories, incomplete configured/receipt ownership sets, and direct bypass of shared-root claims fail closed before removal.
- status: canonical

##### Foundation Receipt-v1 Template Retirement Tombstone
- path: `libraries/knowledge-vector-engine-cpp/cmake/install-receipt.json.in`
- title: Foundation Receipt-v1 Template Retirement Tombstone
- surface_type: retired packaging compatibility surface
- truth_role: immutable SCLV path-continuity and receipt-v2 migration truth
- owner: SKV foundation maintainers
- scope: Preserves the historically referenced template path while declaring it nonoperational and routing all current packaging to the shared receipt-v2 generator.
- relationships: superseded_by -> `cmake/SymphonyInstallReceiptV2.cmake`; preserves -> `knowledge/sclv/CHANGELOG.md`
- consumers: SCLV reference validation, repository archaeology, reviewers
- deferred_projections: none
- notes: No CMake target consumes this tombstone and it cannot generate an install receipt.
- status: canonical

##### Maestro Receipt-v1 Template Retirement Tombstone
- path: `modules/maestro/cmake/install-receipt.json.in`
- title: Maestro Receipt-v1 Template Retirement Tombstone
- surface_type: retired packaging compatibility surface
- truth_role: immutable SCLV path-continuity and receipt-v2 migration truth
- owner: common SKV / Maestro maintainers
- scope: Preserves the historically referenced template path while declaring it nonoperational and routing all current packaging to the shared receipt-v2 generator.
- relationships: superseded_by -> `cmake/SymphonyInstallReceiptV2.cmake`; preserves -> `knowledge/sclv/CHANGELOG.md`
- consumers: SCLV reference validation, repository archaeology, reviewers
- deferred_projections: none
- notes: No CMake target consumes this tombstone and it cannot generate an install receipt.
- status: canonical

##### Validator Receipt-v1 Template Retirement Tombstone
- path: `tools/symphony-validator/cmake/install-receipt.json.in`
- title: Validator Receipt-v1 Template Retirement Tombstone
- surface_type: retired packaging compatibility surface
- truth_role: immutable SCLV path-continuity and receipt-v2 migration truth
- owner: validator maintainers
- scope: Preserves the historically referenced template path while declaring it nonoperational and routing all current packaging to the shared receipt-v2 generator.
- relationships: superseded_by -> `cmake/SymphonyInstallReceiptV2.cmake`; preserves -> `knowledge/sclv/CHANGELOG.md`
- consumers: SCLV reference validation, repository archaeology, reviewers
- deferred_projections: none
- notes: No CMake target consumes this tombstone and it cannot generate an install receipt.
- status: canonical

##### Lifecycle Apply Journal Schema v2
- path: `knowledge/schemas/v2/lifecycle-boot-journal.schema.json`
- title: Symphony Apply-Capable Lifecycle Journal v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical protected action-attempt, dynamic-replan, applied-state, compatibility, and recovery truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes exact report-journal source binding, prepared and completed attempts, active action, plan revisions, blockers, checkpoints, v1/v2 compatibility, content-addressed applied-state selection, and digest continuity.
- relationships: depends_on -> `knowledge/schemas/v1/lifecycle-boot-journal.schema.json`; depends_on -> `knowledge/schemas/v1/lifecycle-applied-state.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; persisted_by -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`
- consumers: C++ coordinator apply operations, qxctl apply/status/recovery, conformance tests, validator, reviewers
- deferred_projections: live service/process and Maestro action adapters
- notes: Version 2 lives beside and references report-only v1; it never rewrites a v1 stream or treats an orphan applied file as selected state.
- status: canonical

##### Lifecycle Apply Head Schema v2
- path: `knowledge/schemas/v2/lifecycle-boot-head.schema.json`
- title: Symphony Apply-Capable Lifecycle Head v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical active-slot selector and atomic apply-journal continuity truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes profile/TOPS/transaction identity, active slot, generation, selected v2 journal digest, prior-head digest, and head digest.
- relationships: depends_on -> `knowledge/schemas/v2/lifecycle-boot-journal.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`; persisted_by -> `modules/knowledge-session-coordinator/src/lifecycle_journal.cpp`
- consumers: coordinator apply recovery, qxctl apply status, conformance tests, validator, reviewers
- deferred_projections: rendered durability documentation
- notes: The head is the commit selector; journal slots and content-addressed applied files remain preserved recovery evidence.
- status: canonical

##### Reconciliation Command Schema
- path: `knowledge/schemas/v1/reconciliation-command.schema.json`
- title: Symphony Knowledge Reconciliation Command v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical qxctl-to-coordinator reconciliation request truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes operation identity, state root, expected journal state, path inventory, exact bound-engine inventory, and client compatibility declaration.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; implemented_by -> `tools/qxctl/cmd/qxctl/main.go`; implemented_by -> `modules/knowledge-session-coordinator/src/reconciliation.cpp`
- consumers: qxctl, coordinator, conformance tests, validator, reviewers
- deferred_projections: rendered protocol documentation
- notes: The command does not carry permission or canonical-apply authority.
- status: canonical

##### Reconciliation Head Schema
- path: `knowledge/schemas/v1/reconciliation-head.schema.json`
- title: Symphony Knowledge Reconciliation Head v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical active-slot selector and atomic continuity truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes worktree/context identity, active slot, generation, journal digest, prior-head digest, and head digest.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; persisted_by -> `modules/knowledge-session-coordinator/src/reconciliation.cpp`
- consumers: coordinator recovery, conformance tests, validator, reviewers
- deferred_projections: rendered durability documentation
- notes: The head is replaceable operational evidence; journal slots remain the recovery evidence.
- status: canonical

##### Reconciliation Journal Schema
- path: `knowledge/schemas/v1/reconciliation-journal.schema.json`
- title: Symphony Knowledge Reconciliation Journal v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical noncanonical-state format and recovery-evidence truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes repository/worktree identity, selected paths, exact engine inventory, content snapshots, checkpoint chain, compatibility, extensions, recovery disposition, and journal digest.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; persisted_by -> `modules/knowledge-session-coordinator/src/reconciliation.cpp`
- consumers: coordinator, qxctl status/recovery, conformance tests, validator, reviewers
- deferred_projections: future stepwise journal-format migrations
- notes: The journal records operational truth but is not canonical knowledge or an authority epoch.
- status: canonical

##### Reconciliation Result Schema
- path: `knowledge/schemas/v1/reconciliation-result.schema.json`
- title: Symphony Knowledge Reconciliation Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical compatibility, state, mutation, and repair result truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes compatibility mode, optional journal, change/recovery flags, repair actions, read-only status, and disabled canonical apply.
- relationships: depends_on -> `knowledge/schemas/v1/reconciliation-journal.schema.json`; emitted_by -> `modules/knowledge-session-coordinator/src/reconciliation.cpp`; consumed_by -> `tools/qxctl/cmd/qxctl/main.go`
- consumers: qxctl, conformance tests, validator, reviewers
- deferred_projections: rendered operator diagnostics
- notes: A successful recovery result reports local operational repair only.
- status: canonical

##### Authenticated-Session Command Schema
- path: `knowledge/schemas/v1/session-command.schema.json`
- title: Symphony Knowledge Authenticated-Session Command v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical qxctl-to-coordinator authenticated-session request truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes session operation identity, state root, repository/TOPS binding, expected journal state, context references, client compatibility, and complete SSIAG authorization evidence.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; depends_on -> `knowledge/ssiag/schemas/v1/authorization-decision.schema.json`; implemented_by -> `tools/qxctl/cmd/qxctl/main.go`; implemented_by -> `modules/knowledge-session-coordinator/src/authority_session.cpp`
- consumers: qxctl, coordinator, conformance tests, validator, reviewers
- deferred_projections: rendered protocol documentation
- notes: Safe capability evidence is non-transferable and cannot authorize canonical apply.
- status: canonical

##### Authenticated-Session Head Schema
- path: `knowledge/schemas/v1/session-head.schema.json`
- title: Symphony Knowledge Authenticated-Session Head v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical active-slot selector and atomic session continuity truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes session identity, active slot, generation, journal digest, prior-head digest, and head digest.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; persisted_by -> `modules/knowledge-session-coordinator/src/authority_session.cpp`
- consumers: coordinator recovery, conformance tests, validator, reviewers
- deferred_projections: rendered durability documentation
- notes: The head is replaceable operational evidence; journal slots remain the recovery evidence.
- status: canonical

##### Authenticated-Session Journal Schema
- path: `knowledge/schemas/v1/session-journal.schema.json`
- title: Symphony Knowledge Authenticated-Session Journal v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical noncanonical authority-epoch state and recovery-evidence truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes TOPS/subject/repository identity, linked epochs, effective state, decision/capability binding, context references, checkpoint chain, compatibility, extensions, recovery disposition, and journal digest.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; persisted_by -> `modules/knowledge-session-coordinator/src/authority_session.cpp`
- consumers: coordinator, qxctl status/recovery, conformance tests, validator, reviewers
- deferred_projections: future stepwise session-journal-format migrations
- notes: The journal records protected operational state but is not canonical knowledge or transferable authority.
- status: canonical

##### Authenticated-Session Result Schema
- path: `knowledge/schemas/v1/session-result.schema.json`
- title: Symphony Knowledge Authenticated-Session Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical compatibility, effective-state, mutation, and repair result truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes compatibility mode, optional journal, effective state, change/recovery flags, repair actions, noncanonical status, and disabled canonical apply.
- relationships: depends_on -> `knowledge/schemas/v1/session-journal.schema.json`; emitted_by -> `modules/knowledge-session-coordinator/src/authority_session.cpp`; consumed_by -> `tools/qxctl/cmd/qxctl/main.go`
- consumers: qxctl, conformance tests, validator, reviewers
- deferred_projections: rendered operator diagnostics
- notes: A successful result reports local authority-epoch coordination only and never canonical mutation.
- status: canonical

##### Authenticated-Session Transition Result Schema
- path: `knowledge/schemas/v1/session-transition-result.schema.json`
- title: Symphony Explicit Authenticated-Session Transition Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical qxctl host-event convergence result truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes login/refresh/logout event identity, deterministic disposition, initial/final state, composed operation evidence, bounded recovery flag, disabled canonical apply, and result digest.
- relationships: depends_on -> `knowledge/schemas/v1/session-result.schema.json`; implemented_by -> `tools/qxctl/cmd/qxctl/main.go`; emitted_by -> `tools/qxctl/cmd/qxctl/main.go`
- consumers: qxctl host lifecycle integrations, tests, validator, reviewers
- deferred_projections: host-specific login-manager and boot integration documentation
- notes: The result is noncanonical evidence; it neither installs host integration nor authorizes module lifecycle apply.
- status: canonical

##### Engine Process Request Schema
- path: `knowledge/schemas/v1/engine-process-request.schema.json`
- title: Symphony Knowledge Engine Process Request v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical local process request envelope truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes request fields and binds protocol, IDs, operation, target, deadline, and payload object.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; implemented_by -> `libraries/knowledge-vector-engine-cpp/SPEC.md`
- consumers: coordinator and vector engines, qxctl process client, conformance tests, validator
- deferred_projections: rendered protocol documentation
- notes: The process protocol is local standard I/O, not OpenAPI or HTTP.
- status: canonical

##### Engine Process Response Schema
- path: `knowledge/schemas/v1/engine-process-response.schema.json`
- title: Symphony Knowledge Engine Process Response v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical local process response envelope truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes success/error result shape and binds engine identity and response digest.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; implemented_by -> `libraries/knowledge-vector-engine-cpp/SPEC.md`
- consumers: coordinator and vector engines, qxctl process client, conformance tests, validator
- deferred_projections: rendered protocol documentation
- notes: Exactly one compact response is emitted in process mode.
- status: canonical

##### Engine Descriptor Schema
- path: `knowledge/schemas/v1/engine-descriptor.schema.json`
- title: Symphony Knowledge Engine Descriptor v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical engine identity, capability, limit, scope, and disabled-state truth
- owner: Symphony Knowledge Vector maintainers
- scope: Defines installed identity, operations, bounds, thermal placement, scope, docking state, and mutation flags.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; implemented_by -> `modules/knowledge-session-coordinator/SPEC.md`; implemented_by -> `modules/skvi-engine/SPEC.md`; implemented_by -> `modules/sclv-engine/SPEC.md`
- consumers: qxctl lifecycle planner, coordinator and vector engines, packagers, reviewers
- deferred_projections: installed engine inventory and Maestro presence graph
- notes: A descriptor reports capability; it does not grant permission or activate a version.
- status: canonical

##### Install Receipt Schema
- path: `knowledge/schemas/v1/install-receipt.schema.json`
- title: Symphony Knowledge Module Install Receipt v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical prefix-relative package ownership and docking-state truth
- owner: Symphony Knowledge Vector maintainers
- scope: Defines module/version, scope, prefix interpretation, state, activation, receptor, and exact owned files.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; implemented_by -> `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`; implemented_by -> `modules/knowledge-session-coordinator/CMakeLists.txt`; implemented_by -> `modules/skvi-engine/CMakeLists.txt`; implemented_by -> `modules/sclv-engine/CMakeLists.txt`
- consumers: qxctl lifecycle planner, installers, uninstallers, packagers, reviewers
- deferred_projections: lifecycle inventory and rollback evidence
- notes: A receipt does not authorize activation, canonical writes, or live docking.
- status: canonical

##### Engine Binding Registry Schema
- path: `knowledge/schemas/v1/engine-binding-registry.schema.json`
- title: Symphony Knowledge Engine Binding Registry v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical schema for protected noncanonical user-default exact-version selection
- owner: Symphony Knowledge Vector maintainers
- scope: Closes profile identity, generation, expected-state chain, exact receipt/executable identity and digests, inactive-undocked binding state, and noncanonical status.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; implemented_by -> `tools/qxctl/internal/knowledgebinding/registry.go`
- consumers: qxctl lifecycle administration, future coordinator reconciliation, validator, reviewers
- deferred_projections: repository/system/TOPS profiles and Maestro docking
- notes: A binding does not install, invoke, authenticate, authorize, activate a receipt, or dock an engine.
- status: canonical

##### Knowledge Proposal Schema
- path: `knowledge/schemas/v1/proposal.schema.json`
- title: Symphony Knowledge Proposal v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical provider-neutral immutable proposal envelope truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes proposal identity, repository, read/write sets, operations, validation, authority, expiry, and disabled-apply fields.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; implemented_by -> `modules/skvi-engine/SPEC.md`; implemented_by -> `modules/sclv-engine/SPEC.md`
- consumers: vector engines, qxctl process clients, conformance tests, validator, reviewers
- deferred_projections: rendered protocol documentation
- notes: A conforming proposal is noncanonical and never manufactures ratification.
- status: canonical

##### Provider Evidence Schema
- path: `knowledge/schemas/v1/provider-evidence.schema.json`
- title: Symphony Knowledge Provider Evidence v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical provider-neutral normalized evidence envelope truth
- owner: Symphony Knowledge Vector maintainers
- scope: Closes adapter identity, observation, repository revision/tree, change-request presence, ratification claim, and evidence digest fields.
- relationships: depends_on -> `knowledge/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sclv-engine/SPEC.md`
- consumers: SCLV evidence adapters, SCLV engine, qxctl proposal callers, conformance tests, validator, reviewers
- deferred_projections: rendered provider-adapter reference and conformance evidence
- notes: A well-formed evidence envelope does not grant permission, ratify, or establish provider truth by itself.
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
  - checked_by -> `tools/symphony-validator/SPEC.md`
  - interprets -> SCLV cross-reference validation
  - interprets -> SODV publication governance
- consumers:
  - maintainers
  - agentic reviewers
  - NotebookLM corpus alignment
  - symphony-validator
  - qxctl-derived evidence consumers
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
- consumers: humans, symphony-validator, `symphony-skvi`, qxctl, and future validator extensions
- relationships: checked_by -> `tools/symphony-validator/SPEC.md`; depends_on -> `knowledge/SPEC.md`
- deferred_projections: JSON/JSONL, search, analytical, and graph projections
- notes: Authorizes proposal/projection engine behavior but no canonical apply.
- status: canonical

##### MANIFEST.md
- path: `knowledge/skvi/MANIFEST.md`
- title: SKVI Manifest
- surface_type: vector contract truth
- truth_role: declared contract truth for SKVI
- owner: SKVI maintainer
- scope: Contractual requirements and independent proposal-engine installability.
- consumers: humans, symphony-validator and future validator extensions
- relationships: checked_by -> `tools/symphony-validator/SPEC.md`
- deferred_projections: digest-bound SKVI projections
- notes: `symphony-skvi` remains subordinate to canonical `INDEX.md`.
- status: canonical

##### SKILL.md
- path: `knowledge/skvi/SKILL.md`
- title: SKVI Skill
- surface_type: vector skill guidance
- truth_role: operational skill guidance
- owner: SKVI maintainer
- scope: Usage and interaction.
- consumers: humans, agentic tools
- relationships: depends_on -> `knowledge/skvi/SPEC.md`; depends_on -> `knowledge/SPEC.md`
- deferred_projections: none
- notes: Guides safe proposal/read engine use.
- status: canonical

##### SPEC.md
- path: `knowledge/skvi/SPEC.md`
- title: SKVI Specification
- surface_type: vector specification
- truth_role: declarative specification behavior
- owner: SKVI maintainer
- scope: Formatting, structure, engine operations, and projection boundaries.
- consumers: humans, symphony-validator, `symphony-skvi`, qxctl
- relationships: checked_by -> `tools/symphony-validator/SPEC.md`; depends_on -> `knowledge/SPEC.md`
- deferred_projections: JSON/JSONL, search, analytical, and graph projections
- notes: Initial engine operations are inspect, check, propose, and project.
- status: canonical

##### SKVI v1 Schema Manifest
- path: `knowledge/skvi/schemas/v1/MANIFEST.md`
- title: SKVI Schemas v1
- surface_type: vector-specific protocol schema manifest
- truth_role: canonical inventory and boundary for exact SKVI JSON schemas
- owner: SKVI maintainers
- scope: Declares normalized entry, proposal payload, check result, and projection schemas.
- relationships: depends_on -> `knowledge/skvi/SPEC.md`; implemented_by -> `modules/skvi-engine/SPEC.md`
- consumers: SKVI engine, qxctl, conformance tests, validator, reviewers
- deferred_projections: rendered SKVI protocol documentation
- notes: It does not authorize inferred membership or canonical apply.
- status: canonical

##### SKVI Entry Schema
- path: `knowledge/skvi/schemas/v1/entry.schema.json`
- title: SKVI Normalized Entry v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical normalized projection-entry shape
- owner: SKVI maintainers
- scope: Closes the ten semantic fields, canonical status, safe path, and entry digest.
- relationships: depends_on -> `knowledge/skvi/schemas/v1/MANIFEST.md`; implemented_by -> `modules/skvi-engine/SPEC.md`
- consumers: SKVI projections, qxctl, conformance tests, validator
- deferred_projections: rendered entry reference
- notes: Normalization does not make a derived entry canonical index truth.
- status: canonical

##### SKVI Operation Payload Schema
- path: `knowledge/skvi/schemas/v1/operation-payload.schema.json`
- title: SKVI Proposal Operation Payload v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical caller-declared proposal input shape
- owner: SKVI maintainers
- scope: Closes repository identity, optional session/context references, caller timestamps, and add/replace/remove semantics.
- relationships: depends_on -> `knowledge/skvi/schemas/v1/MANIFEST.md`; implemented_by -> `modules/skvi-engine/SPEC.md`
- consumers: qxctl proposal callers, SKVI engine, conformance tests, validator
- deferred_projections: proposal form and SDK documentation
- notes: The caller selects membership intent; the engine validates but does not decide it.
- status: canonical

##### SKVI Check Result Schema
- path: `knowledge/skvi/schemas/v1/check-result.schema.json`
- title: SKVI Check Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic structural evidence shape
- owner: SKVI maintainers
- scope: Closes index/contract digests, evidence, counts, valid/invalid state, read-only status, and disabled apply.
- relationships: depends_on -> `knowledge/skvi/schemas/v1/MANIFEST.md`; implemented_by -> `modules/skvi-engine/SPEC.md`
- consumers: qxctl check presentation, conformance tests, validator, reviewers
- deferred_projections: check reports and analytical evidence
- notes: Invalid state is evidence and does not authorize repair.
- status: canonical

##### SKVI Projection Schema
- path: `knowledge/skvi/schemas/v1/projection.schema.json`
- title: SKVI Structural Projection v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical disposable projection-result shape
- owner: SKVI maintainers
- scope: Closes engine/input identities, normalized entries, projection digest, and noncanonical rebuildable state.
- relationships: depends_on -> `knowledge/skvi/schemas/v1/MANIFEST.md`; implements_entries -> `knowledge/skvi/schemas/v1/entry.schema.json`; implemented_by -> `modules/skvi-engine/SPEC.md`
- consumers: qxctl project presentation, conformance tests, graph/search planners, validator
- deferred_projections: JSONL, search, analytical, and graph projections after separate authorization
- notes: This v1 implementation returns JSON in the process response and writes no projection file.
- status: canonical

#### SCLV
##### INTENT.md
- path: `knowledge/sclv/INTENT.md`
- title: SCLV Intent
- surface_type: vector intent seed
- truth_role: intent and purpose for SCLV
- owner: SCLV maintainer
- scope: SCLV records change truth.
- consumers: humans, symphony-validator, `symphony-sclv`, qxctl, and future validator extensions
- relationships: depends_on -> `knowledge/SPEC.md`
- deferred_projections: provider-neutral JSON/JSONL, graph, and recovery evidence
- notes: Proposal-only engine behavior is authorized; canonical append remains gated.
- status: canonical

##### MANIFEST.md
- path: `knowledge/sclv/MANIFEST.md`
- title: SCLV Manifest
- surface_type: vector contract truth
- truth_role: declared contract truth for SCLV
- owner: SCLV maintainer
- scope: Contractual requirements, independent proposal-engine installability, and provider-neutral v3 transition.
- consumers: humans, symphony-validator and future validator extensions
- relationships: depends_on -> `knowledge/SPEC.md`
- deferred_projections: provider-neutral ledger projections
- notes: V3 activation requires its exact schema/template and validator increment.
- status: canonical

##### SKILL.md
- path: `knowledge/sclv/SKILL.md`
- title: SCLV Skill
- surface_type: vector skill guidance
- truth_role: operational skill guidance
- owner: SCLV maintainer
- scope: Usage and interaction.
- consumers: humans, agentic tools
- relationships: depends_on -> `knowledge/sclv/SPEC.md`; depends_on -> `knowledge/SPEC.md`
- deferred_projections: none
- notes: Guides engine-assisted ephemeral recovery without canonical append.
- status: canonical

##### SPEC.md
- path: `knowledge/sclv/SPEC.md`
- title: SCLV Specification
- surface_type: vector specification
- truth_role: declarative specification behavior
- owner: SCLV maintainer
- scope: Structuring change records, provider-neutral v3 semantics, engine operations, and forward recovery.
- consumers: humans, symphony-validator, `symphony-sclv`, qxctl
- relationships: depends_on -> `knowledge/SPEC.md`; checked_by -> `tools/symphony-validator/SPEC.md`
- deferred_projections: provider-neutral JSON/JSONL, graph, and recovery evidence
- notes: Version 1/2 history remains immutable; programmatic append is disabled.
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
    checked by current `tools/symphony-validator/SPEC.md` rules for record shape, vocabulary, continuity, and SKVI references
    may be consumed by future qxctl-derived evidence projections
    does not replace Git history
    does not replace PR review
    does not replace SSCG interpretation
  consumers:
    - `maintainers`
    - `agentic reviewers`
    - `NotebookLM corpus alignment`
    - `symphony-validator`
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

##### RECOVERY.md
- path: `knowledge/sclv/RECOVERY.md`
- title: `SCLV Recovery and PR #59 Incident Record`
- surface_type: `sclv_recovery_runbook`
- truth_role: `canonical recovery procedure and incident evidence`
- owner: `SCLV`
- scope: `Defines forward-only reconciliation for interrupted closure sessions and records the verified PR #59 failure analysis.`
- relationships: `depends_on -> knowledge/sclv/SPEC.md; may_consume -> knowledge/sodv/RELEASES.md`
- consumers: `Architect, maintainers, reviewers, agentic tools, symphony-validator maintainers`
- deferred_projections: `future read-only qxctl recovery-status projection`
- status: `canonical`
- notes: `Ephemeral session state remains under .git and is never canonical. The runbook also distinguishes a stale temporary-proxy module cache from immutable public release state and requires empty-cache verification.`

##### SCLV v3 Schema Manifest
- path: `knowledge/sclv/schemas/v3/MANIFEST.md`
- title: SCLV Version 3 Schema Manifest
- surface_type: vector-specific protocol schema manifest
- truth_role: canonical inventory and boundary for exact provider-neutral SCLV v3 JSON schemas
- owner: SCLV maintainers
- scope: Declares record, proposal-input, recovery-input, check-result, and projection schemas.
- relationships: depends_on -> `knowledge/sclv/SPEC.md`; implemented_by -> `modules/sclv-engine/SPEC.md`
- consumers: SCLV engine, qxctl, conformance tests, validator, reviewers
- deferred_projections: rendered SCLV protocol documentation
- notes: Schema validity does not grant ratification, append, journal mutation, or apply authority.
- status: canonical

##### SCLV v3 Record Schema
- path: `knowledge/sclv/schemas/v3/record.schema.json`
- title: SCLV Provider-Neutral Record v3
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical normalized provider-neutral change-record shape
- owner: SCLV maintainers
- scope: Closes stable identity, time, disposition, change request, revision/tree, ratification, affected surfaces, consequences, evidence, and non-authorizations.
- relationships: depends_on -> `knowledge/sclv/schemas/v3/MANIFEST.md`; rendered_by -> `knowledge/sclv/templates/v3/record.md`; implemented_by -> `modules/sclv-engine/SPEC.md`
- consumers: SCLV engine, qxctl proposal callers, symphony-validator, conformance tests, reviewers
- deferred_projections: rendered record reference and form tooling
- notes: Version 3 is prospective; v1/v2 records remain immutable.
- status: canonical

##### SCLV v3 Proposal Input Schema
- path: `knowledge/sclv/schemas/v3/proposal-input.schema.json`
- title: SCLV Proposal Input v3
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical caller-declared SCLV proposal input shape
- owner: SCLV maintainers
- scope: Closes repository/session/context/expiry fields, one v3 record, and one-to-eight normalized provider-evidence envelopes.
- relationships: depends_on -> `knowledge/sclv/schemas/v3/record.schema.json`; depends_on -> `knowledge/schemas/v1/provider-evidence.schema.json`; implemented_by -> `modules/sclv-engine/SPEC.md`
- consumers: qxctl proposal callers, SCLV engine, conformance tests, validator
- deferred_projections: proposal forms and protocol documentation
- notes: The engine validates claims but neither grants permission nor ratifies.
- status: canonical

##### SCLV v3 Recovery Input Schema
- path: `knowledge/sclv/schemas/v3/recovery-input.schema.json`
- title: SCLV Recovery Input v3
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical non-mutating ephemeral-journal reconciliation input shape
- owner: SCLV maintainers
- scope: Closes the journal snapshot/digest, observed state, optional late proposal input, and factual recovery reason.
- relationships: depends_on -> `knowledge/sclv/schemas/v3/proposal-input.schema.json`; implemented_by -> `modules/sclv-engine/SPEC.md`
- consumers: qxctl recovery callers, SCLV engine, conformance tests, validator
- deferred_projections: recovery-status evidence
- notes: Recovery output may recommend deletion but never mutates or deletes the journal.
- status: canonical

##### SCLV Check Result Schema
- path: `knowledge/sclv/schemas/v3/check-result.schema.json`
- title: SCLV Check Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic ledger-diagnostic shape
- owner: SCLV maintainers
- scope: Closes ledger/contract digests, expected-state match, record/version counts, bounded exceptions, summary, read-only state, and disabled apply.
- relationships: depends_on -> `knowledge/sclv/schemas/v3/MANIFEST.md`; implemented_by -> `modules/sclv-engine/SPEC.md`
- consumers: qxctl check presentation, conformance tests, validator, reviewers
- deferred_projections: check reports and analytical evidence
- notes: Invalid state is evidence and does not authorize repair.
- status: canonical

##### SCLV Projection Schema
- path: `knowledge/sclv/schemas/v3/projection.schema.json`
- title: SCLV Provider-Neutral Projection v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical disposable projection-result shape
- owner: SCLV maintainers
- scope: Closes engine/input identities, normalized legacy and v3 records, projection digest, and noncanonical rebuildable state.
- relationships: depends_on -> `knowledge/sclv/schemas/v3/MANIFEST.md`; implemented_by -> `modules/sclv-engine/SPEC.md`
- consumers: qxctl project presentation, conformance tests, graph/search planners, validator
- deferred_projections: JSONL, search, analytical, and graph projections after separate authorization
- notes: The v1 implementation returns JSON in the process response and writes no projection file.
- status: canonical

##### SCLV v3 Record Template
- path: `knowledge/sclv/templates/v3/record.md`
- title: SCLV Canonical Record Template v3
- surface_type: canonical Markdown record template
- truth_role: exact field order and rendering truth for prospective v3 ledger records
- owner: SCLV maintainers
- scope: Defines the complete ordered Markdown representation emitted inside a noncanonical proposal.
- relationships: depends_on -> `knowledge/sclv/schemas/v3/record.schema.json`; implemented_by -> `modules/sclv-engine/SPEC.md`; checked_by -> `tools/symphony-validator/SPEC.md`
- consumers: SCLV engine, symphony-validator, reviewers, proposal callers
- deferred_projections: rendered authoring guidance
- notes: The template is canonical contract truth; an engine-rendered instance remains noncanonical until separately applied.
- status: canonical

#### SODV
##### INTENT.md
- path: `knowledge/sodv/INTENT.md`
- title: SODV Intent
- surface_type: vector intent seed
- truth_role: intent and purpose for SODV
- owner: SODV maintainer
- scope: SODV governs publication truth.
- consumers: humans, symphony-validator, `symphony-sodv`, qxctl, and future validator extensions
- relationships: checked_by -> `tools/symphony-validator/SPEC.md`; depends_on -> `knowledge/SPEC.md`
- deferred_projections: release and publication evidence
- notes: Proposal/read engine behavior is authorized; publication remains separately permission-backed.
- status: canonical

##### MANIFEST.md
- path: `knowledge/sodv/MANIFEST.md`
- title: SODV Manifest
- surface_type: vector contract truth
- truth_role: declared contract truth for SODV
- owner: SODV maintainer
- scope: Contractual requirements for publication and independent proposal-engine installability.
- consumers: humans, symphony-validator and future validator extensions
- relationships: checked_by -> `tools/symphony-validator/SPEC.md`
- deferred_projections: release and publication evidence
- notes: The engine cannot create tags, publish artifacts, or append canonical records.
- status: canonical

##### SKILL.md
- path: `knowledge/sodv/SKILL.md`
- title: SODV Skill
- surface_type: vector skill guidance
- truth_role: operational skill guidance
- owner: SODV maintainer
- scope: Usage and interaction.
- consumers: humans, agentic tools
- relationships: depends_on -> `knowledge/sodv/SPEC.md`; depends_on -> `knowledge/SPEC.md`
- deferred_projections: none
- notes: Guides safe proposal/read engine use without publication authority.
- status: canonical

##### SPEC.md
- path: `knowledge/sodv/SPEC.md`
- title: SODV Specification
- surface_type: vector specification
- truth_role: declarative specification behavior
- owner: SODV maintainer
- scope: Publication governance, release transactions, proposal/read engine operations, and derived evidence.
- consumers: humans, symphony-validator, `symphony-sodv`, qxctl
- relationships: checked_by -> `tools/symphony-validator/SPEC.md`; depends_on -> `knowledge/SPEC.md`
- deferred_projections: release and publication evidence
- notes: Canonical apply, tag publication, Mintlify, and NotebookLM automation remain unauthorized.
- status: canonical

##### RELEASES.md
- path: `knowledge/sodv/RELEASES.md`
- title: `SODV Release Publication Ledger`
- surface_type: `release_publication_ledger`
- truth_role: `canonical module-publication authorization and completion truth`
- owner: `SODV maintainer`
- scope: `Binds module versions to immutable source commits before publication and records clean-cache completion evidence afterward.`
- relationships: `depends_on -> knowledge/sodv/SPEC.md; depends_on -> knowledge/sclv/CHANGELOG.md; records -> module release publication; checked_by -> tools/symphony-validator/SPEC.md for local record relationships`
- consumers: `Architect, release maintainers, reviewers, agentic tools, symphony-validator and future validator extensions`
- deferred_projections: `release notes, package index, public documentation`
- status: `canonical`
- notes: `Authorization never implies completion; pending transaction state is noncanonical. External tag and package-provider state remains caller-supplied evidence.`

##### SODV v1 Schema Manifest
- path: `knowledge/sodv/schemas/v1/MANIFEST.md`
- title: SODV Operational Schema Manifest
- surface_type: vector-specific protocol schema manifest
- truth_role: canonical inventory and boundary for exact SODV release-engine schemas
- owner: SODV maintainers
- scope: Declares v2 records and v1 observation, check, verify, proposal-input, recovery-input, recovery-result, and projection schemas.
- relationships: depends_on -> `knowledge/sodv/SPEC.md`; implemented_by -> `modules/sodv-engine/SPEC.md`
- consumers: SODV engine, qxctl, conformance tests, validator, reviewers
- deferred_projections: rendered SODV protocol documentation
- notes: Operational schemas grant no tag, publication, completion, ratification, or apply authority.
- status: canonical

##### SODV Release Record v2 Schema
- path: `knowledge/sodv/schemas/v1/release-record-v2.schema.json`
- title: SODV Provider-Neutral Release Record v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical prospective release-record shape
- owner: SODV maintainers
- scope: Closes record identity, lineage, caller authority, publication units, immutable revisions, evidence, and non-authorizations.
- relationships: depends_on -> `knowledge/sodv/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sodv-engine/SPEC.md`
- consumers: SODV engine, qxctl proposal callers, conformance tests, validator
- deferred_projections: release-record reference documentation
- notes: Version 2 is prospective; historical v1 records remain immutable.
- status: canonical

##### SODV Observed State Schema
- path: `knowledge/sodv/schemas/v1/observed-state.schema.json`
- title: SODV Observed Publication State v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical caller-supplied external-state evidence shape
- owner: SODV maintainers
- scope: Closes authorization identity, observation time/source, tag objects/targets, public state, digests, and evidence digests.
- relationships: depends_on -> `knowledge/sodv/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sodv-engine/SPEC.md`
- consumers: qxctl verify/recovery callers, SODV engine, conformance tests
- deferred_projections: provider adapters after separate authorization
- notes: The engine performs no provider lookup; supplying evidence does not make it canonical.
- status: canonical

##### SODV Check Result Schema
- path: `knowledge/sodv/schemas/v1/check-result.schema.json`
- title: SODV Check Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic release-ledger diagnostic shape
- owner: SODV maintainers
- scope: Closes ledger/snapshot digests, expected-state match, record/transaction counts, evidence, summary, read-only state, and disabled apply.
- relationships: depends_on -> `knowledge/sodv/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sodv-engine/SPEC.md`
- consumers: qxctl check presentation, conformance tests, validator, reviewers
- deferred_projections: check reports
- notes: Invalid state is evidence and does not authorize repair.
- status: canonical

##### SODV Verify Result Schema
- path: `knowledge/sodv/schemas/v1/verify-result.schema.json`
- title: SODV Verification Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical noncanonical verification-result shape
- owner: SODV maintainers
- scope: Closes release state classification, per-unit findings, canonical completion reference, and explicit non-completion authority.
- relationships: depends_on -> `knowledge/sodv/schemas/v1/observed-state.schema.json`; implemented_by -> `modules/sodv-engine/SPEC.md`
- consumers: qxctl verify/recovery presentation, SODV engine, conformance tests
- deferred_projections: verification reports
- notes: `engine_declares_completion` is always false.
- status: canonical

##### SODV Proposal Input Schema
- path: `knowledge/sodv/schemas/v1/proposal-input.schema.json`
- title: SODV Proposal Input v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical caller-declared release-proposal input shape
- owner: SODV maintainers
- scope: Closes repository/session/context/expiry, expected ledger digest, one v2 record, and optional observed state.
- relationships: depends_on -> `knowledge/sodv/schemas/v1/release-record-v2.schema.json`; depends_on -> `knowledge/schemas/v1/proposal.schema.json`; implemented_by -> `modules/sodv-engine/SPEC.md`
- consumers: qxctl proposal callers, SODV engine, conformance tests
- deferred_projections: proposal forms
- notes: The engine validates but neither publishes nor ratifies.
- status: canonical

##### SODV Recovery Input Schema
- path: `knowledge/sodv/schemas/v1/recovery-input.schema.json`
- title: SODV Recovery Input v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical non-mutating interrupted-session reconciliation input shape
- owner: SODV maintainers
- scope: Closes a local journal snapshot/digest, caller observation, optional forward proposal, and recovery reason.
- relationships: depends_on -> `knowledge/sodv/schemas/v1/proposal-input.schema.json`; implemented_by -> `modules/sodv-engine/SPEC.md`
- consumers: qxctl recovery callers, SODV engine, conformance tests
- deferred_projections: recovery-status evidence
- notes: Recovery never moves a tag, edits a record, or mutates/deletes the journal.
- status: canonical

##### SODV Recovery Result Schema
- path: `knowledge/sodv/schemas/v1/recovery-result.schema.json`
- title: SODV Recovery Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical recovery recommendation shape
- owner: SODV maintainers
- scope: Closes the action, nested verification, optional proposal, journal disposition recommendation, digest, and disabled apply.
- relationships: depends_on -> `knowledge/sodv/schemas/v1/recovery-input.schema.json`; implemented_by -> `modules/sodv-engine/SPEC.md`
- consumers: qxctl recovery presentation, SODV engine, conformance tests
- deferred_projections: recovery reports
- notes: Delete recommendation is not journal mutation.
- status: canonical

##### SODV Projection Schema
- path: `knowledge/sodv/schemas/v1/projection.schema.json`
- title: SODV Release Transaction Projection v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical disposable projection-result shape
- owner: SODV maintainers
- scope: Closes engine/input identities, normalized records, transaction summaries, digest, and noncanonical rebuildable state.
- relationships: depends_on -> `knowledge/sodv/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sodv-engine/SPEC.md`
- consumers: qxctl project presentation, conformance tests, future analytical planners
- deferred_projections: JSONL, search, graph, and public release projections after separate authorization
- notes: The implementation writes no projection file.
- status: canonical

### SODV Engine Module

#### SODV Engine INTENT.md
- path: `modules/sodv-engine/INTENT.md`
- title: SODV Engine Intent
- surface_type: independently installable module intent
- truth_role: subordinate engine purpose and authority boundary
- owner: SODV engine maintainers
- scope: Declares provider-neutral proposal/read behavior for module-release truth.
- relationships: depends_on -> `knowledge/sodv/INTENT.md`; declares -> `modules/sodv-engine/MANIFEST.md`
- consumers: qxctl, implementers, reviewers, administrators, agentic tools
- deferred_projections: provider adapters and public documentation
- notes: The engine is not a publisher or source of canonical truth.
- status: canonical

#### SODV Engine MANIFEST.md
- path: `modules/sodv-engine/MANIFEST.md`
- title: SODV Engine Manifest
- surface_type: independently installable module manifest
- truth_role: executable, operation, dependency, installation, and authority truth
- owner: SODV engine maintainers
- scope: Declares six implemented proposal/read operations, three disabled mutation operations, and installed-undocked state.
- relationships: depends_on -> `modules/sodv-engine/INTENT.md`; implements -> `knowledge/sodv/SPEC.md`; statically_links -> `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- consumers: build/install tooling, qxctl, reviewers, conformance tests
- deferred_projections: provider adapters and lifecycle activation
- notes: No network, tag, publication, completion-declaration, or canonical-apply authority exists.
- status: canonical

#### SODV Engine INSTALL.md
- path: `modules/sodv-engine/INSTALL.md`
- title: SODV Engine Installation
- surface_type: module installation contract
- truth_role: versioned prefix install, receipt, coexistence, and owned uninstall procedure
- owner: SODV engine maintainers
- scope: Defines the exact inactive-undocked nine-file package.
- relationships: depends_on -> `modules/sodv-engine/MANIFEST.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/INSTALL.md`; consumed_by -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: administrators, packaging, qxctl, conformance tests
- deferred_projections: lifecycle-administrator automation
- notes: Installation creates no active alias or Maestro docking state.
- status: canonical

#### SODV Engine SKILL.md
- path: `modules/sodv-engine/SKILL.md`
- title: SODV Engine Skill
- surface_type: module operational guidance
- truth_role: safe direct/qxctl invocation and verification procedure
- owner: SODV engine maintainers
- scope: Guides bounded check, verify, proposal, recovery, and projection use.
- relationships: depends_on -> `modules/sodv-engine/SPEC.md`; depends_on -> `knowledge/sodv/SKILL.md`
- consumers: administrators, reviewers, agentic tools
- deferred_projections: operator runbooks
- notes: External observations remain caller-supplied and noncanonical.
- status: canonical

#### SODV Engine SPEC.md
- path: `modules/sodv-engine/SPEC.md`
- title: SODV Engine Specification
- surface_type: module implementation specification
- truth_role: exact process, parser, operation, recovery, and authority behavior
- owner: SODV engine maintainers
- scope: Defines bounded historical/v2 parsing, caller observations, proposals, recovery recommendations, and disposable inventories.
- relationships: implements -> `knowledge/sodv/SPEC.md`; depends_on -> `knowledge/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/SPEC.md`
- consumers: implementers, qxctl, conformance tests, validator, reviewers
- deferred_projections: providers, mutation, publication, and lifecycle activation
- notes: All canonical writes and external side effects are absent.
- status: canonical

#### SODV Engine CMakeLists.txt
- path: `modules/sodv-engine/CMakeLists.txt`
- title: SODV Engine Build Contract
- surface_type: CMake build/install implementation
- truth_role: C++26 target, tests, exact package layout, receipt, and uninstall implementation truth
- owner: SODV engine maintainers
- scope: Builds `symphony-sodv` and its conformance suite against the shared static foundation.
- relationships: implements -> `modules/sodv-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`
- consumers: builders, packagers, tests, administrators
- deferred_projections: external package formats
- notes: The exact versioned receipt owns nine installed files.
- status: canonical

#### SSFV

##### INTENT.md
- path: `knowledge/ssfv/INTENT.md`
- title: SSFV Intent
- surface_type: vector intent
- truth_role: canonical purpose and source-truth boundary for semantic feature truth
- owner: SSFV maintainers
- scope: Defines feature purpose, feature-worthiness, caller neutrality, and adjacent-vector boundaries.
- relationships: depends_on -> `knowledge/SPEC.md`; checked_by -> `tools/symphony-validator/SPEC.md`
- consumers: reviewers, SSFV engine, qxctl planners, agentic tools
- deferred_projections: feature catalogs, encyclopedia views, publication inputs, graph views
- notes: Defines the governing intent for the implemented engine and exact seventy-four-record partial catalog without granting unratified-record authority.
- status: canonical

##### MANIFEST.md
- path: `knowledge/ssfv/MANIFEST.md`
- title: SSFV Manifest
- surface_type: vector contract truth
- truth_role: canonical topology, identity, classification, and installability boundary
- owner: SSFV maintainers
- scope: Declares owned feature semantics, sparse distributed records, implemented engine identity, explicit coverage inventory, and exact seventy-four-record partial-catalog state.
- relationships: depends_on -> `knowledge/ssfv/INTENT.md`; checked_by -> `tools/symphony-validator/SPEC.md`
- consumers: reviewers, implementers, qxctl planners, packaging planners
- deferred_projections: installation descriptors and Maestro docking descriptors after separate review
- notes: The engine module and seventy-four experimental records are implemented; fifty-nine nested subfeatures are ratified while remaining nested review and repository-wide catalog completeness are not claimed.
- status: canonical

##### SKILL.md
- path: `knowledge/ssfv/SKILL.md`
- title: SSFV Skill
- surface_type: vector operational guidance
- truth_role: canonical feature-review and sparse-placement procedure
- owner: SSFV maintainers
- scope: Guides feature-worthiness review, 5W1H authoring, relationship evidence, and stop conditions.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`; depends_on -> `knowledge/SPEC.md`
- consumers: reviewers, maintainers, agentic tools, qxctl callers
- deferred_projections: authoring assistance and review checklists
- notes: Caller type is not an authority input.
- status: canonical

##### SPEC.md
- path: `knowledge/ssfv/SPEC.md`
- title: SSFV Specification
- surface_type: vector specification
- truth_role: canonical feature identity, semantics, hierarchy, lifecycle, routing, and protocol contract
- owner: SSFV maintainers
- scope: Defines stable IDs, feature kinds, 5W1H, sparse files, registry, implemented operations, freshness, proposals, and graphs.
- relationships: depends_on -> `knowledge/SPEC.md`; checked_by -> `tools/symphony-validator/SPEC.md`
- consumers: reviewers, SSFV engine, qxctl planners, validator
- deferred_projections: portable JSON graph, catalogs, search, documentation, and analytical views
- notes: Engine implementation and the exact seventy-four-record partial catalog are current; every additional record remains separately reviewed and remaining nested coverage remains incomplete.
- status: canonical

##### NAMESPACES.md
- path: `knowledge/ssfv/NAMESPACES.md`
- title: SSFV Namespace Registry
- surface_type: canonical stable-identity namespace registry
- truth_role: canonical allocation truth for SSFV identifier prefixes
- owner: SSFV maintainers
- scope: Allocates the first-party `ssfv:symphony:` prefix and governs future namespace allocation.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`
- consumers: reviewers, SSFV engine, qxctl planners
- deferred_projections: namespace inventory
- notes: Internal stable-ID allocation claims no external URI, package, repository, or trademark authority.
- status: canonical

##### REGISTRY.md
- path: `knowledge/ssfv/REGISTRY.md`
- title: SSFV Feature Registry
- surface_type: canonical distributed-feature routing registry
- truth_role: canonical mapping from stable feature IDs to owner records
- owner: SSFV maintainers
- scope: Defines the exact eight-field registry grammar and routes the seventy-four records in the current partial feature set.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`
- consumers: reviewers, SSFV engine, qxctl planners, validator
- deferred_projections: feature inventories and graph routing
- notes: The current catalog is partial; structural closure does not establish repository-wide completeness.
- status: canonical

##### SSFV v1 Schema Manifest
- path: `knowledge/ssfv/schemas/v1/MANIFEST.md`
- title: SSFV v1 Schema Manifest
- surface_type: vector-specific protocol schema manifest
- truth_role: canonical inventory and boundary for exact SSFV record and operation schemas
- owner: SSFV maintainers
- scope: Declares feature, namespace, registry, check, diff, proposal, and graph schemas.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`
- consumers: SSFV engine, qxctl planners, conformance tests, validator, reviewers
- deferred_projections: rendered protocol documentation
- notes: Machine-readable contracts do not implement an engine or grant mutation authority.
- status: canonical

##### SSFV Feature Record Schema
- path: `knowledge/ssfv/schemas/v1/feature-record.schema.json`
- title: SSFV Normalized Feature Record v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical normalized semantic feature-record shape
- owner: SSFV maintainers
- scope: Closes stable identity, kind, lifecycle, hierarchy, 5W1H, implementation evidence, relationships, distinctions, references, evidence, and non-claims.
- relationships: depends_on -> `knowledge/ssfv/schemas/v1/MANIFEST.md`
- consumers: SSFV engine, qxctl planners, conformance tests, validator
- deferred_projections: feature reference documentation
- notes: The schema does not decide feature-worthiness.
- status: canonical

##### SSFV Namespace Entry Schema
- path: `knowledge/ssfv/schemas/v1/namespace-entry.schema.json`
- title: SSFV Namespace Entry v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical normalized namespace-allocation shape
- owner: SSFV maintainers
- scope: Closes namespace, prefix, owner contract, scope, lifecycle, evidence, and notes.
- relationships: depends_on -> `knowledge/ssfv/schemas/v1/MANIFEST.md`
- consumers: SSFV engine, qxctl planners, conformance tests, validator
- deferred_projections: namespace inventories
- notes: Namespace allocation does not claim external provider identity.
- status: canonical

##### SSFV Registry Entry Schema
- path: `knowledge/ssfv/schemas/v1/registry-entry.schema.json`
- title: SSFV Registry Entry v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical normalized feature-routing entry shape
- owner: SSFV maintainers
- scope: Closes feature ID, owner file, contract, source scope, lifecycle, parent, digest, and notes.
- relationships: depends_on -> `knowledge/ssfv/schemas/v1/MANIFEST.md`
- consumers: SSFV engine, qxctl planners, conformance tests, validator
- deferred_projections: feature routing inventories
- notes: Routing metadata never replaces the distributed semantic record.
- status: canonical

##### SSFV Check Result Schema
- path: `knowledge/ssfv/schemas/v1/check-result.schema.json`
- title: SSFV Check Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic structural and semantic-freshness result shape
- owner: SSFV maintainers
- scope: Closes snapshot, coverage, counts, structural state, freshness state, findings, disabled apply, and digest.
- relationships: depends_on -> `knowledge/ssfv/schemas/v1/MANIFEST.md`
- consumers: SSFV engine, qxctl planners, conformance tests, validator
- deferred_projections: check reports
- notes: Semantic candidates remain unratified evidence.
- status: canonical

##### SSFV Diff Input Schema
- path: `knowledge/ssfv/schemas/v1/diff-input.schema.json`
- title: SSFV Diff Input v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical bounded content-addressed comparison input shape
- owner: SSFV maintainers
- scope: Closes baseline and current digests, bounded feature scope, and semantic-candidate selection.
- relationships: depends_on -> `knowledge/ssfv/schemas/v1/MANIFEST.md`
- consumers: SSFV engine, qxctl planners, conformance tests
- deferred_projections: comparison forms
- notes: Revision labels never replace content digests.
- status: canonical

##### SSFV Diff Result Schema
- path: `knowledge/ssfv/schemas/v1/diff-result.schema.json`
- title: SSFV Diff Result v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic feature-change evidence shape
- owner: SSFV maintainers
- scope: Closes added, changed, removed, uncovered, stale, and semantic-candidate results.
- relationships: depends_on -> `knowledge/ssfv/schemas/v1/diff-input.schema.json`
- consumers: SSFV engine, qxctl planners, conformance tests
- deferred_projections: change reports
- notes: A diff result is evidence and grants no ratification or mutation authority.
- status: canonical

##### SSFV Proposal Input Schema
- path: `knowledge/ssfv/schemas/v1/proposal-input.schema.json`
- title: SSFV Proposal Input v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical caller-declared bounded multi-file proposal input shape
- owner: SSFV maintainers
- scope: Closes session, context, expiry, expected digests, operation, feature IDs, target paths, semantic declaration, and permission reference.
- relationships: depends_on -> `knowledge/ssfv/schemas/v1/MANIFEST.md`; depends_on -> `knowledge/schemas/v1/proposal.schema.json`
- consumers: SSFV engine, qxctl planners, conformance tests
- deferred_projections: proposal forms
- notes: Proposal generation never applies canonical changes.
- status: canonical

##### SSFV Graph Projection Schema
- path: `knowledge/ssfv/schemas/v1/graph-projection.schema.json`
- title: SSFV Portable Graph Projection v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical shape for disposable semantic feature graphs
- owner: SSFV maintainers
- scope: Closes source snapshots, feature nodes, typed edges, counts, noncanonical state, rebuildability, and digest.
- relationships: depends_on -> `knowledge/ssfv/schemas/v1/MANIFEST.md`
- consumers: SSFV engine, qxctl planners, graph and documentation consumers
- deferred_projections: JSONL, search, visualization, analytical, and graph-database imports after separate review
- notes: The graph is portable JSON and never canonical authority.
- status: canonical

### SSFV Executable Contract Additions

#### SSFV Feature File Format
- path: `knowledge/ssfv/FEATURE-FILE-FORMAT.md`
- title: SSFV Feature File Format
- surface_type: canonical machine-managed Markdown format
- truth_role: exact distributed feature-file envelope, routing, normalization, and rendering contract
- owner: SSFV maintainers
- scope: Defines the single managed JSON region and byte-preserving owner-text boundary.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, validator, reviewers, agentic tools
- deferred_projections: rendered feature references
- notes: The format creates no feature record and grants no apply authority.
- status: canonical

#### SSFV Feature File Schema
- path: `knowledge/ssfv/schemas/v1/feature-file.schema.json`
- title: SSFV Feature File Envelope v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical embedded feature-file envelope shape
- owner: SSFV maintainers
- scope: Binds protocol, owner contract, source scope, and ordered complete records.
- relationships: depends_on -> `knowledge/ssfv/FEATURE-FILE-FORMAT.md`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, validator, conformance tests
- deferred_projections: schema documentation
- notes: The schema does not authorize an empty canonical owner file.
- status: canonical

#### SSFV Semantic Snapshot Schema
- path: `knowledge/ssfv/schemas/v1/semantic-snapshot.schema.json`
- title: SSFV Semantic Snapshot v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical content-addressed feature and evidence snapshot shape
- owner: SSFV maintainers
- scope: Binds contract, namespace, registry, owner-file, record, and source-evidence digests.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, freshness and diff callers
- deferred_projections: snapshot reports
- notes: Snapshots contain digests and metadata rather than source bodies.
- status: canonical

#### SSFV Graph Input Schema
- path: `knowledge/ssfv/schemas/v1/graph-input.schema.json`
- title: SSFV Graph Input v1
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical exact graph-operation input shape
- owner: SSFV maintainers
- scope: Restricts the graph request to portable JSON output.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, conformance tests
- deferred_projections: future separately reviewed transports
- notes: No persistent graph store or paging contract is authorized.
- status: canonical

#### SSFV v2 Schema Manifest
- path: `knowledge/ssfv/schemas/v2/MANIFEST.md`
- title: SSFV v2 Schema Manifest
- surface_type: vector protocol schema manifest
- truth_role: canonical executable-operation schema inventory
- owner: SSFV maintainers
- scope: Declares v2 record, registry, check, diff, and proposal payload/result contracts.
- relationships: depends_on -> `knowledge/ssfv/SPEC.md`; depends_on -> `knowledge/ssfv/schemas/v1/MANIFEST.md`
- consumers: SSFV engine, qxctl, validator, conformance tests
- deferred_projections: schema documentation
- notes: v1 historical contracts remain canonical and are not silently reinterpreted.
- status: canonical

#### SSFV v2 Check Input Schema
- path: `knowledge/ssfv/schemas/v2/check-input.schema.json`
- title: SSFV Check Input v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical structural and freshness check request shape
- owner: SSFV maintainers
- scope: Binds expected namespace/registry digests, freshness mode, and optional baseline.
- relationships: depends_on -> `knowledge/ssfv/schemas/v2/MANIFEST.md`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, conformance tests
- deferred_projections: check forms
- notes: Report and require modes need a semantic baseline.
- status: canonical

#### SSFV v2 Check Result Schema
- path: `knowledge/ssfv/schemas/v2/check-result.schema.json`
- title: SSFV Check Result v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical structural, coverage, snapshot, and freshness evidence shape
- owner: SSFV maintainers
- scope: Reports deterministic findings and caller-neutral semantic-review candidates.
- relationships: depends_on -> `knowledge/ssfv/schemas/v2/check-input.schema.json`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, conformance tests
- deferred_projections: check reports
- notes: Semantic candidates are evidence and remain unratified.
- status: canonical

#### SSFV v2 Diff Input Schema
- path: `knowledge/ssfv/schemas/v2/diff-input.schema.json`
- title: SSFV Diff Input v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical prior-snapshot versus live-state comparison request
- owner: SSFV maintainers
- scope: Carries one bounded baseline, optional expected current digest, scope, and candidate flag.
- relationships: depends_on -> `knowledge/ssfv/schemas/v1/semantic-snapshot.schema.json`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, conformance tests
- deferred_projections: diff forms
- notes: The engine does not invoke Git or a forge for historical state.
- status: canonical

#### SSFV v2 Diff Result Schema
- path: `knowledge/ssfv/schemas/v2/diff-result.schema.json`
- title: SSFV Diff Result v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical deterministic semantic-change evidence shape
- owner: SSFV maintainers
- scope: Reports added, changed, removed, uncovered, stale, and review-required evidence.
- relationships: depends_on -> `knowledge/ssfv/schemas/v2/diff-input.schema.json`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, conformance tests
- deferred_projections: change reports
- notes: The result is noncanonical and grants no ratification.
- status: canonical

#### SSFV v2 Feature Record Schema
- path: `knowledge/ssfv/schemas/v2/feature-record.schema.json`
- title: SSFV Normalized Feature Record v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical executable normalized semantic feature-record shape
- owner: SSFV maintainers
- scope: Closes stable identity, strict hierarchy, 5W1H, implementation evidence, relationships, distinctions, references, evidence, and non-claims.
- relationships: depends_on -> `knowledge/ssfv/schemas/v2/MANIFEST.md`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, conformance tests, reviewers
- deferred_projections: feature reference documentation
- notes: Record validation does not decide feature-worthiness.
- status: canonical

#### SSFV v2 Proposal Input Schema
- path: `knowledge/ssfv/schemas/v2/proposal-input.schema.json`
- title: SSFV Proposal Input v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical caller-declared namespace or feature proposal input
- owner: SSFV maintainers
- scope: Binds repository identity, expected state, desired semantics, targets, evidence, and authorization reference.
- relationships: depends_on -> `knowledge/ssfv/schemas/v2/feature-record.schema.json`; depends_on -> `knowledge/schemas/v1/proposal.schema.json`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, conformance tests
- deferred_projections: proposal forms
- notes: Proposal generation never applies canonical changes.
- status: canonical

#### SSFV v2 Registry Entry Schema
- path: `knowledge/ssfv/schemas/v2/registry-entry.schema.json`
- title: SSFV Registry Entry v2
- surface_type: JSON Schema Draft 2020-12 contract
- truth_role: canonical executable feature-routing entry shape
- owner: SSFV maintainers
- scope: Supports root scope, shared owner routing, lifecycle, parent, and normalized record digest.
- relationships: depends_on -> `knowledge/ssfv/schemas/v2/MANIFEST.md`; implemented_by -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV engine, qxctl, conformance tests
- deferred_projections: feature-routing inventories
- notes: Routing metadata never replaces the distributed semantic record.
- status: canonical

### SSFV Engine Module

#### SSFV Engine INTENT.md
- path: `modules/ssfv-engine/INTENT.md`
- title: SSFV Engine Intent
- surface_type: independently installable vector-engine intent
- truth_role: subordinate semantic feature inspection and proposal purpose
- owner: SSFV engine maintainers
- scope: Declares bounded freezing-path SSFV mechanics without semantic-decision or mutation authority.
- relationships: depends_on -> `knowledge/ssfv/INTENT.md`; declares -> `modules/ssfv-engine/MANIFEST.md`
- consumers: qxctl, implementers, administrators, reviewers, agentic tools
- deferred_projections: installed-engine inventory and conformance evidence
- notes: The engine implements SSFV truth but does not own it.
- status: canonical

#### SSFV Engine MANIFEST.md
- path: `modules/ssfv-engine/MANIFEST.md`
- title: SSFV Engine Manifest
- surface_type: independently installable vector-engine manifest
- truth_role: executable, operation, dependency, lifecycle, and authority truth
- owner: SSFV engine maintainers
- scope: Declares C++26 inspect, check, diff, propose, and graph operations with disabled apply.
- relationships: depends_on -> `modules/ssfv-engine/INTENT.md`; implements -> `knowledge/ssfv/SPEC.md`; statically_links -> `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- consumers: qxctl, packagers, implementers, reviewers, agentic tools
- deferred_projections: engine inventory and Maestro presence evidence
- notes: Installation is inactive and undocked with no default receptor.
- status: canonical

#### SSFV Engine INSTALL.md
- path: `modules/ssfv-engine/INSTALL.md`
- title: SSFV Engine Installation
- surface_type: module installation contract
- truth_role: versioned prefix build, test, receipt, and receipt-owned uninstall procedure
- owner: SSFV engine maintainers
- scope: Defines the exact inactive-undocked nine-file package.
- relationships: depends_on -> `modules/ssfv-engine/MANIFEST.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/INSTALL.md`; consumed_by -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: administrators, packagers, qxctl, conformance tests
- deferred_projections: lifecycle-administrator automation
- notes: No global alias, active binding, service, socket, or hook is installed.
- status: canonical

#### SSFV Engine SKILL.md
- path: `modules/ssfv-engine/SKILL.md`
- title: SSFV Engine Skill
- surface_type: module operational guidance
- truth_role: safe direct and qxctl invocation procedure
- owner: SSFV engine maintainers
- scope: Guides structural/freshness checks, diffs, proposals, and disposable graphs.
- relationships: depends_on -> `modules/ssfv-engine/SPEC.md`; depends_on -> `knowledge/ssfv/SKILL.md`
- consumers: administrators, reviewers, agentic tools
- deferred_projections: operator runbooks
- notes: Every operation remains caller-neutral and non-mutating.
- status: canonical

#### SSFV Engine SPEC.md
- path: `modules/ssfv-engine/SPEC.md`
- title: SSFV Engine Specification
- surface_type: module implementation specification
- truth_role: exact process, operation, parser, bound, freshness, and authority behavior
- owner: SSFV engine maintainers
- scope: Defines deterministic managed-region parsing, snapshots, diffs, proposals, graphs, the exact descriptor-v2 65,536-value complete-administration envelope, and disabled apply.
- relationships: implements -> `knowledge/ssfv/SPEC.md`; depends_on -> `knowledge/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/SPEC.md`
- consumers: implementers, qxctl, conformance tests, validator, reviewers
- deferred_projections: persistent graph, mutation, lifecycle activation, and docking
- notes: It has no session, authentication, network, semantic-decision, feature-bootstrap, or Maestro authority.
- status: canonical

#### SSFV Engine CMakeLists.txt
- path: `modules/ssfv-engine/CMakeLists.txt`
- title: SSFV Engine Build Contract
- surface_type: CMake build and install implementation
- truth_role: C++26 target, tests, exact package layout, receipt, and uninstall implementation truth
- owner: SSFV engine maintainers
- scope: Builds `symphony-ssfv` and its conformance suite against the shared static foundation.
- relationships: implements -> `modules/ssfv-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`
- consumers: builders, packagers, tests, administrators
- deferred_projections: external package formats
- notes: The exact versioned receipt owns nine installed files.
- status: canonical

### STSC Implementation Evidence

#### SKVI Engine Temporal Adoption
- path: `modules/skvi-engine/src/skvi.cpp`
- title: SKVI Engine Temporal Validation Adoption
- surface_type: C++26 engine implementation
- truth_role: SKVI proposal-time STSC conformance evidence
- owner: SKVI engine maintainers
- scope: Uses the shared whole-second UTC validator for proposal creation and expiry fields.
- relationships: implements -> `modules/skvi-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp`
- consumers: SKVI engine tests, qxctl, reviewers
- deferred_projections: cross-language temporal conformance evidence
- notes: SKVI retains no clock or timestamp authority.
- status: canonical

#### SCLV Provider Temporal Adoption
- path: `modules/sclv-engine/src/provider.cpp`
- title: SCLV Provider Temporal Validation Adoption
- surface_type: C++26 provider-evidence implementation
- truth_role: normalized provider-evidence STSC conformance
- owner: SCLV engine maintainers
- scope: Routes provider and ledger whole-second UTC validation through the shared temporal foundation.
- relationships: implements -> `modules/sclv-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp`
- consumers: SCLV engine and evidence adapters, tests, reviewers
- deferred_projections: provider conformance evidence
- notes: Provider timestamps remain evidence and cannot manufacture ratification or causal order.
- status: canonical

#### SACV Engine Temporal Adoption
- path: `modules/sacv-engine/src/sacv.cpp`
- title: SACV Engine Temporal Validation Adoption
- surface_type: C++26 engine implementation
- truth_role: SACV proposal-time STSC conformance evidence
- owner: SACV engine maintainers
- scope: Uses the shared whole-second UTC validator for API-contract proposal creation and expiry fields.
- relationships: implements -> `modules/sacv-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp`
- consumers: SACV engine tests, qxctl, reviewers
- deferred_projections: cross-language temporal conformance evidence
- notes: SACV owns API-contract semantics, not durable timestamp authority.
- status: canonical

#### SODV Engine Temporal Adoption
- path: `modules/sodv-engine/src/sodv.cpp`
- title: SODV Engine Temporal Validation Adoption
- surface_type: C++26 engine implementation
- truth_role: release-ledger and proposal STSC conformance evidence
- owner: SODV engine maintainers
- scope: Uses the shared whole-second UTC validator for release records, observations, proposals, and recovery inputs.
- relationships: implements -> `modules/sodv-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp`
- consumers: SODV engine tests, qxctl, reviewers
- deferred_projections: release-evidence temporal conformance report
- notes: Append-only record order remains SODV-owned and is not inferred solely from wall-clock text.
- status: canonical

#### SSFV Engine Temporal Adoption
- path: `modules/ssfv-engine/src/ssfv.cpp`
- title: SSFV Engine Temporal Validation Adoption
- surface_type: C++26 engine implementation
- truth_role: semantic-proposal STSC conformance evidence
- owner: SSFV engine maintainers
- scope: Uses the shared whole-second UTC validator for semantic proposal creation and expiry fields.
- relationships: implements -> `modules/ssfv-engine/SPEC.md`; depends_on -> `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp`
- consumers: SSFV engine tests, qxctl, reviewers
- deferred_projections: semantic lifecycle temporal conformance evidence
- notes: Semantic identity and freshness remain digest-bound rather than timestamp-identified.
- status: canonical

#### Validator SODV Temporal Checker
- path: `tools/symphony-validator/src/sodv_releases.cpp`
- title: Independent SODV Temporal Checker
- surface_type: C++26 read-only validator implementation
- truth_role: independent Gregorian release-ledger conformance evidence
- owner: symphony-validator maintainers
- scope: Rejects impossible whole-second UTC dates without linking the shared engine foundation.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; conforms_to -> `knowledge/TIME.md`; checks -> `knowledge/sodv/RELEASES.md`
- consumers: repository validation, reviewers, release gates
- deferred_projections: line-oriented temporal conformance evidence
- notes: Architectural independence from the vector-engine foundation is preserved.
- status: canonical

#### Validator SODV Temporal Tests
- path: `tools/symphony-validator/tests/sodv_release_test.cpp`
- title: Independent SODV Temporal Conformance Tests
- surface_type: C++26 validator test implementation
- truth_role: impossible-date and release-ledger regression proof
- owner: symphony-validator maintainers
- scope: Proves canonical release evidence passes and impossible Gregorian dates fail closed.
- relationships: verifies -> `tools/symphony-validator/src/sodv_releases.cpp`; conforms_to -> `knowledge/TIME.md`
- consumers: CTest, repository maintainers, reviewers
- deferred_projections: portable validator conformance report
- notes: Tests mutate only isolated temporary fixtures.
- status: canonical

#### Validator Required Canonical Surfaces
- path: `tools/symphony-validator/src/canonical_surfaces.cpp`
- title: Validator Required Canonical Surfaces
- surface_type: C++26 read-only validator implementation
- truth_role: exact required-surface presence evidence
- owner: symphony-validator maintainers
- scope: Requires the canonical STSC document alongside the existing root, vector, runtime-seed, and validator surfaces.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; requires -> `knowledge/TIME.md`
- consumers: repository validation, fixtures, reviewers
- deferred_projections: canonical-surface inventory evidence
- notes: Presence does not grant the validator authorship authority.
- status: canonical

#### Validator Knowledge Contract Shapes
- path: `tools/symphony-validator/src/knowledge_contracts.cpp`
- title: Validator Knowledge Contract Shape Checks
- surface_type: C++26 read-only validator implementation
- truth_role: deterministic canonical knowledge-anchor evidence
- owner: symphony-validator maintainers
- scope: Checks STSC authority, purpose, UTC profiles, timestamp authority, implementation, and promotion-boundary anchors with existing knowledge contracts.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; checks -> `knowledge/TIME.md`
- consumers: repository validation, fixtures, reviewers
- deferred_projections: contract-shape conformance report
- notes: Anchor presence does not interpret or rewrite temporal doctrine.
- status: canonical

#### Validator Doctrine Vocabulary Scan
- path: `tools/symphony-validator/src/doctrine_vocab.cpp`
- title: Validator Doctrine Vocabulary Scan
- surface_type: C++26 read-only validator implementation
- truth_role: bounded canonical terminology regression evidence
- owner: symphony-validator maintainers
- scope: Includes STSC in the fixed canonical vocabulary scan without granting semantic authority.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; scans -> `knowledge/TIME.md`
- consumers: repository validation, fixtures, reviewers
- deferred_projections: vocabulary conformance evidence
- notes: The fixed scan remains deterministic and non-remediating.
- status: canonical

#### Validator STSC Valid Fixture
- path: `tools/symphony-validator/tests/fixtures_valid/knowledge/TIME.md`
- title: Validator STSC Valid Fixture
- surface_type: canonical-contract test fixture
- truth_role: required-presence and anchor-shape regression proof
- owner: symphony-validator maintainers
- scope: Supplies the bounded STSC headings required by the pre-SSFV valid repository fixture.
- relationships: verifies -> `tools/symphony-validator/src/canonical_surfaces.cpp`; verifies -> `tools/symphony-validator/src/knowledge_contracts.cpp`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: The fixture is test evidence, not a competing temporal contract.
- status: canonical

#### Validator Valid Fixture Index
- path: `tools/symphony-validator/tests/fixtures_valid/knowledge/skvi/INDEX.md`
- title: Validator Valid Fixture Index
- surface_type: SKVI test fixture
- truth_role: positive fixture routing evidence
- owner: symphony-validator maintainers
- scope: Routes the valid pre-SSFV fixture's required temporal contract and established canonical surfaces.
- relationships: indexes -> `tools/symphony-validator/tests/fixtures_valid/knowledge/TIME.md`; verifies -> `tools/symphony-validator/tests/smoke.sh`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: Fixture evidence only.
- status: canonical

#### Validator Sparse-Ledger STSC Fixture
- path: `tools/symphony-validator/tests/fixtures_sclv_ledger_gap_warning/knowledge/TIME.md`
- title: Validator Sparse-Ledger STSC Fixture
- surface_type: canonical-contract test fixture
- truth_role: positive sparse-ledger temporal-contract evidence
- owner: symphony-validator maintainers
- scope: Keeps the sparse SCLV namespace fixture valid under the required STSC surface gate.
- relationships: verifies -> `tools/symphony-validator/tests/smoke.sh`; conforms_to -> `knowledge/TIME.md`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: Fixture evidence only.
- status: canonical

#### Validator Sparse-Ledger Fixture Index
- path: `tools/symphony-validator/tests/fixtures_sclv_ledger_gap_warning/knowledge/skvi/INDEX.md`
- title: Validator Sparse-Ledger Fixture Index
- surface_type: SKVI test fixture
- truth_role: positive fixture routing evidence
- owner: symphony-validator maintainers
- scope: Routes the sparse-ledger fixture's required temporal contract.
- relationships: indexes -> `tools/symphony-validator/tests/fixtures_sclv_ledger_gap_warning/knowledge/TIME.md`; verifies -> `tools/symphony-validator/tests/smoke.sh`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: Fixture evidence only.
- status: canonical

#### Validator Unindexed-Affected-Surface STSC Fixture
- path: `tools/symphony-validator/tests/fixtures_affected_surface_unindexed/knowledge/TIME.md`
- title: Validator Unindexed-Affected-Surface STSC Fixture
- surface_type: canonical-contract test fixture
- truth_role: warning-only affected-surface temporal-contract evidence
- owner: symphony-validator maintainers
- scope: Keeps the intentional unindexed-affected-surface fixture otherwise valid under STSC.
- relationships: verifies -> `tools/symphony-validator/tests/smoke.sh`; conforms_to -> `knowledge/TIME.md`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: Its intentional warning concerns another fixture path, not STSC.
- status: canonical

#### Validator Unindexed-Affected-Surface Fixture Index
- path: `tools/symphony-validator/tests/fixtures_affected_surface_unindexed/knowledge/skvi/INDEX.md`
- title: Validator Unindexed-Affected-Surface Fixture Index
- surface_type: SKVI test fixture
- truth_role: warning-only fixture routing evidence
- owner: symphony-validator maintainers
- scope: Routes STSC while intentionally leaving the designated affected surface unindexed.
- relationships: indexes -> `tools/symphony-validator/tests/fixtures_affected_surface_unindexed/knowledge/TIME.md`; verifies -> `tools/symphony-validator/tests/smoke.sh`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: Fixture evidence only.
- status: canonical

#### Validator Score-Vocabulary STSC Fixture
- path: `tools/symphony-validator/tests/fixtures_vocab_score/knowledge/TIME.md`
- title: Validator Score-Vocabulary STSC Fixture
- surface_type: canonical-contract test fixture
- truth_role: positive vocabulary temporal-contract evidence
- owner: symphony-validator maintainers
- scope: Keeps the allowed score-vocabulary fixture valid under STSC.
- relationships: verifies -> `tools/symphony-validator/tests/smoke.sh`; conforms_to -> `knowledge/TIME.md`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: Fixture evidence only.
- status: canonical

#### Validator Score-Vocabulary Fixture Index
- path: `tools/symphony-validator/tests/fixtures_vocab_score/knowledge/skvi/INDEX.md`
- title: Validator Score-Vocabulary Fixture Index
- surface_type: SKVI test fixture
- truth_role: positive fixture routing evidence
- owner: symphony-validator maintainers
- scope: Routes the score-vocabulary fixture's required temporal contract.
- relationships: indexes -> `tools/symphony-validator/tests/fixtures_vocab_score/knowledge/TIME.md`; verifies -> `tools/symphony-validator/tests/smoke.sh`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: Fixture evidence only.
- status: canonical

#### Validator Hyphenated-Term STSC Fixture
- path: `tools/symphony-validator/tests/fixtures_vocab_c_o_r_e/knowledge/TIME.md`
- title: Validator Hyphenated-Term STSC Fixture
- surface_type: canonical-contract test fixture
- truth_role: positive vocabulary temporal-contract evidence
- owner: symphony-validator maintainers
- scope: Keeps the allowed hyphenated-term vocabulary fixture valid under STSC.
- relationships: verifies -> `tools/symphony-validator/tests/smoke.sh`; conforms_to -> `knowledge/TIME.md`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: Fixture evidence only.
- status: canonical

#### Validator Hyphenated-Term Fixture Index
- path: `tools/symphony-validator/tests/fixtures_vocab_c_o_r_e/knowledge/skvi/INDEX.md`
- title: Validator Hyphenated-Term Fixture Index
- surface_type: SKVI test fixture
- truth_role: positive fixture routing evidence
- owner: symphony-validator maintainers
- scope: Routes the hyphenated-term fixture's required temporal contract.
- relationships: indexes -> `tools/symphony-validator/tests/fixtures_vocab_c_o_r_e/knowledge/TIME.md`; verifies -> `tools/symphony-validator/tests/smoke.sh`
- consumers: validator smoke fixtures, reviewers
- deferred_projections: none
- notes: Fixture evidence only.
- status: canonical

### Maestro Receptor Descriptor Schema
- path: `knowledge/schemas/v1/maestro-receptor-descriptor.schema.json`
- title: Maestro Receptor Descriptor Schema
- surface_type: canonical JSON Schema
- truth_role: Maestro receptor identity and compatibility truth
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Docking Command Schema
- path: `knowledge/schemas/v1/maestro-docking-command.schema.json`
- title: Maestro Docking Command Schema
- surface_type: canonical JSON Schema
- truth_role: qxctl-to-Maestro operation contract truth
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Docking Presence Schema
- path: `knowledge/schemas/v1/maestro-docking-presence.schema.json`
- title: Maestro Docking Presence Schema
- surface_type: canonical JSON Schema
- truth_role: component docking disposition contract truth
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Presence Registry Schema
- path: `knowledge/schemas/v1/maestro-docking-presence-registry.schema.json`
- title: Maestro Presence Registry Schema
- surface_type: canonical JSON Schema
- truth_role: durable receptor registry contract truth
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Presence Head Schema
- path: `knowledge/schemas/v1/maestro-docking-presence-head.schema.json`
- title: Maestro Presence Head Schema
- surface_type: canonical JSON Schema
- truth_role: atomic presence selector contract truth
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Docking Result Schema
- path: `knowledge/schemas/v1/maestro-docking-result.schema.json`
- title: Maestro Docking Result Schema
- surface_type: canonical JSON Schema
- truth_role: bounded presence result contract truth
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Intent
- path: `modules/maestro/INTENT.md`
- title: Maestro Intent
- surface_type: module contract
- truth_role: Maestro purpose and authority boundary
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Manifest
- path: `modules/maestro/MANIFEST.md`
- title: Maestro Manifest
- surface_type: module contract
- truth_role: Maestro identity, installability, and ownership truth
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Installation Contract
- path: `modules/maestro/INSTALL.md`
- title: Maestro Installation Contract
- surface_type: module install contract
- truth_role: independent build, install, and removal procedure
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Administration Skill
- path: `modules/maestro/SKILL.md`
- title: Maestro Administration Skill
- surface_type: module skill contract
- truth_role: safe qxctl administration procedure
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Receptor Presence Specification
- path: `modules/maestro/SPEC.md`
- title: Maestro Receptor Presence Specification
- surface_type: module specification
- truth_role: implemented process, authorization, durability, and operation truth
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Semantic Feature Record
- path: `modules/maestro/FEATURES.md`
- title: Authenticated Durable Maestro Docking Presence Feature Record
- surface_type: distributed SSFV feature record
- truth_role: canonical application-level semantics for exact authenticated Maestro receptor presence
- owner: Maestro and SSFV maintainers
- scope: Records why, what, who, how, when, and where the implemented presence authority and its separately identified complete derived-inventory subfeature apply without claiming engine execution.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/maestro/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`; composes_with -> `modules/knowledge-session-coordinator/FEATURES.md`
- consumers: SSFV engine, qxctl planners, reviewers, agentic tools, future documentation projections
- deferred_projections: feature catalog, graph view, operator documentation
- notes: Owns two experimental records: authenticated docking remains durable presence only, while complete inventory is a read-only derived projection; neither is invocation, scheduling, or supervision.
- status: canonical

### Maestro CMake Build
- path: `modules/maestro/CMakeLists.txt`
- title: Maestro CMake Build
- surface_type: implementation build surface
- truth_role: C++26 build, test, install, and uninstall wiring
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Uninstall Procedure
- path: `modules/maestro/cmake/uninstall.cmake.in`
- title: Maestro Uninstall Procedure
- surface_type: implementation lifecycle surface
- truth_role: receipt-bounded file removal procedure
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro C++ Interface
- path: `modules/maestro/src/maestro.hpp`
- title: Maestro C++ Interface
- surface_type: implementation source
- truth_role: presence authority interface identity
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro C++ Authority
- path: `modules/maestro/src/maestro.cpp`
- title: Maestro C++ Authority
- surface_type: implementation source
- truth_role: authenticated durable receptor presence implementation
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Process Entry
- path: `modules/maestro/src/main.cpp`
- title: Maestro Process Entry
- surface_type: implementation source
- truth_role: bounded process-envelope entrypoint
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Conformance Tests
- path: `modules/maestro/tests/maestro_test.cpp`
- title: Maestro Conformance Tests
- surface_type: implementation test
- truth_role: state, authorization, recovery, and safety evidence
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### Maestro Process Smoke Test
- path: `modules/maestro/tests/process_smoke.sh`
- title: Maestro Process Smoke Test
- surface_type: implementation test
- truth_role: installed process-envelope evidence
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### qxctl Maestro Administration
- path: `tools/qxctl/cmd/qxctl/maestro.go`
- title: qxctl Maestro Administration
- surface_type: implementation source
- truth_role: Cobra inspect, status, recovery, and SSIAG composition
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### qxctl Maestro Client
- path: `tools/qxctl/internal/maestroclient/client.go`
- title: qxctl Maestro Client
- surface_type: implementation source
- truth_role: exact installation, process, evidence, and result adapter
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical

### qxctl Maestro Client Tests
- path: `tools/qxctl/internal/maestroclient/client_test.go`
- title: qxctl Maestro Client Tests
- surface_type: implementation test
- truth_role: deterministic resource, evidence, and exact result-boundary evidence
- owner: common SKV / Maestro maintainers
- scope: Implements or governs the original Step 7 authenticated durable presence circuit without engine execution.
- relationships: conforms_to -> `knowledge/SPEC.md`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl, Maestro, lifecycle coordinator, validator, reviewers
- deferred_projections: engine invocation, supervision, scheduling, remote API
- notes: Freezing-path and caller-neutral; presence is not semantic or canonical authority.
- status: canonical


### Common Validation Evidence and Policy Contract
- path: `knowledge/VALIDATION.md`
- title: Symphony Validation Evidence and Policy Contract
- surface_type: common SKV contract
- truth_role: deterministic evidence, warning-policy, baseline, delta, and administration truth
- owner: common SKV maintainers
- scope: Governs immutable raw detection and protected qxctl evaluation in the administrative freezing path.
- relationships: declared_by -> `knowledge/SPEC.md`; implemented_by -> `tools/symphony-validator/SPEC.md`; administered_by -> `tools/qxctl/MANIFEST.md`
- consumers: validator, qxctl, reviewers, agentic tools
- deferred_projections: Markdown report, CI integration, automatic remediation
- notes: Not a vector or engine; policy cannot downgrade violations or narrow detection.
- status: canonical

### Validation Result Schema
- path: `knowledge/schemas/v1/validation-result.schema.json`
- title: Validation Result Schema
- surface_type: canonical JSON Schema
- truth_role: deterministic raw evidence and optional qxctl evaluation contract
- owner: common SKV maintainers
- scope: Governs stable findings, digests, summaries, and evaluation identity.
- relationships: governed_by -> `knowledge/VALIDATION.md`; implemented_by -> `tools/symphony-validator/src/projector.cpp`; consumed_by -> `tools/qxctl/internal/validation/client.go`
- consumers: validator, qxctl, reviewers, agentic tools
- deferred_projections: Markdown report, external analytics
- notes: Contains no collection timestamp and preserves immutable embedded evidence.
- status: canonical

### Validation Policy Schema
- path: `knowledge/schemas/v1/validation-policy.schema.json`
- title: Validation Policy Schema
- surface_type: canonical JSON Schema
- truth_role: protected warning disposition and presentation contract
- owner: common SKV maintainers
- scope: Governs caller-neutral noncanonical policy generations and exact rule overrides.
- relationships: governed_by -> `knowledge/VALIDATION.md`; implemented_by -> `tools/qxctl/internal/validation/policy.go`
- consumers: qxctl, reviewers, agentic tools
- deferred_projections: remote policy administration
- notes: Optional warning policy never changes raw detector behavior.
- status: canonical

### Validation Baseline Schema
- path: `knowledge/schemas/v1/validation-baseline.schema.json`
- title: Validation Baseline Schema
- surface_type: canonical JSON Schema
- truth_role: protected repository/version-bound warning inventory contract
- owner: common SKV maintainers
- scope: Governs new, unchanged, and resolved warning comparison evidence.
- relationships: governed_by -> `knowledge/VALIDATION.md`; implemented_by -> `tools/qxctl/internal/validation/policy.go`
- consumers: qxctl, reviewers, agentic tools
- deferred_projections: remote baseline service
- notes: A baseline is acknowledgement evidence, not ratification or resolution.
- status: canonical

### Validation Warning State Schema
- path: `knowledge/schemas/v1/validation-warning-state.schema.json`
- title: Validation Warning State Schema
- surface_type: canonical JSON Schema
- truth_role: protected warning-subject lifecycle and retained occurrence-history contract
- owner: common SKV maintainers
- scope: Governs stable warning subjects, evidence occurrences, classifications, transition lineage, compare-and-swap generations, and presentation-only mute state.
- relationships: governed_by -> `knowledge/VALIDATION.md`; implemented_by -> `tools/qxctl/internal/validation/warnings.go`; consumed_by -> `tools/qxctl/cmd/qxctl/validation.go`
- consumers: qxctl, administrators, reviewers, agentic tools
- deferred_projections: remote warning-state administration and analytics
- notes: Lifecycle and presentation state never suppress or rewrite raw detector evidence.
- status: canonical

### Validator Structured Projector Interface
- path: `tools/symphony-validator/src/projector.hpp`
- title: Validator Structured Projector Interface
- surface_type: implementation source
- truth_role: deterministic result-v1 projector interface
- owner: validator maintainers
- scope: Declares the complete-evidence structured output boundary.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; conforms_to -> `knowledge/schemas/v1/validation-result.schema.json`
- consumers: validator build, reviewers
- deferred_projections: additional formats
- notes: Authority-free and read-only.
- status: canonical

### Validator Structured Projector
- path: `tools/symphony-validator/src/projector.cpp`
- title: Validator Structured Projector
- surface_type: implementation source
- truth_role: stable finding identity and deterministic digest implementation
- owner: validator maintainers
- scope: Projects the complete in-memory evidence sequence without policy input.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; conforms_to -> `knowledge/schemas/v1/validation-result.schema.json`
- consumers: validator CLI, qxctl
- deferred_projections: additional formats
- notes: Filters and baselines never enter this detector-side projection.
- status: canonical

### Validator Uninstall Procedure
- path: `tools/symphony-validator/cmake/uninstall.cmake.in`
- title: Validator Uninstall Procedure
- surface_type: implementation lifecycle surface
- truth_role: exact receipt-owned removal procedure
- owner: validator maintainers
- scope: Removes only installed validator package files.
- relationships: governed_by -> `tools/symphony-validator/INSTALL.md`; preserves -> `knowledge/VALIDATION.md`
- consumers: CMake, administrators
- deferred_projections: external package-manager removal
- notes: Protected qxctl profiles and baselines remain outside the install prefix.
- status: canonical

### qxctl Validation Warning Lifecycle
- path: `tools/qxctl/internal/validation/warnings.go`
- title: qxctl Validation Warning Lifecycle
- surface_type: implementation source
- truth_role: protected warning subject, occurrence, transition, and presentation-state mechanics
- owner: qxctl maintainers
- scope: Synchronizes immutable raw warnings into open, accepted, resolved, superseded, and presentation-muted protected state with exact compare-and-swap and digest lineage.
- relationships: implements -> `knowledge/VALIDATION.md`; conforms_to -> `knowledge/schemas/v1/validation-warning-state.schema.json`; called_by -> `tools/qxctl/cmd/qxctl/validation.go`
- consumers: qxctl validation commands, administrators, agentic tools
- deferred_projections: automatic remediation and detector suppression
- notes: Muting is presentation-only; accepted state requires rationale and optional expiry; absent current subjects resolve without deleting history.
- status: canonical

### qxctl Validation Command Surface
- path: `tools/qxctl/cmd/qxctl/validation.go`
- title: qxctl Validation Command Surface
- surface_type: implementation source
- truth_role: Cobra scan, debug, profile, baseline, warning-lifecycle, and root-summary grammar
- owner: qxctl maintainers
- scope: Administers full-scan validation, protected noncanonical evaluation and warning lifecycle, and exact installed root-summary projection.
- relationships: implements -> `knowledge/VALIDATION.md`; invokes -> `tools/qxctl/internal/validation/client.go`
- consumers: qxctl, administrators, agentic tools
- deferred_projections: remote API, CI mutation
- notes: Debug filters apply only after full detector execution; root-summary is read-only and warning mutation uses target-host permission plus exact compare-and-swap.
- status: canonical

### qxctl Validation Types
- path: `tools/qxctl/internal/validation/types.go`
- title: qxctl Validation Types
- surface_type: implementation source
- truth_role: exact Go protocol and state representations
- owner: qxctl maintainers
- scope: Implements result, policy, baseline, warning lifecycle, and projection structures.
- relationships: conforms_to -> `knowledge/VALIDATION.md`; conforms_to -> `knowledge/schemas/v1/validation-result.schema.json`; conforms_to -> `knowledge/schemas/v1/validation-warning-state.schema.json`
- consumers: qxctl validation client and store
- deferred_projections: remote API
- notes: Contains no credentials or canonical knowledge.
- status: canonical

### qxctl Validation Digests
- path: `tools/qxctl/internal/validation/digest.go`
- title: qxctl Validation Digests
- surface_type: implementation source
- truth_role: canonical JSON, identity, timestamp, and digest validation mechanics
- owner: qxctl maintainers
- scope: Verifies deterministic evidence and protected state identities.
- relationships: implements -> `knowledge/VALIDATION.md`; conforms_to -> `knowledge/TIME.md`
- consumers: qxctl validation client and store
- deferred_projections: none
- notes: Uses STSC whole-second UTC for durable state.
- status: canonical

### qxctl Validator Client
- path: `tools/qxctl/internal/validation/client.go`
- title: qxctl Validator Client
- surface_type: implementation source
- truth_role: exact installed process invocation and raw evidence verification
- owner: qxctl maintainers
- scope: Invokes one receipt-validated validator with an empty environment and hard bounds.
- relationships: implements -> `knowledge/VALIDATION.md`; invokes -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: qxctl validation commands
- deferred_projections: remote execution
- notes: Rejects identity, summary, occurrence, subject, evidence, and result drift.
- status: canonical

### qxctl Validation Evaluation
- path: `tools/qxctl/internal/validation/evaluate.go`
- title: qxctl Validation Evaluation
- surface_type: implementation source
- truth_role: warning delta, lifecycle classification, disposition, presentation, and debug projection logic
- owner: qxctl maintainers
- scope: Evaluates verified raw evidence without modifying it.
- relationships: implements -> `knowledge/VALIDATION.md`; conforms_to -> `knowledge/schemas/v1/validation-result.schema.json`
- consumers: qxctl validation commands
- deferred_projections: automatic remediation
- notes: Violations always fail; incompatible baselines fail closed; ordinary scans present only actionable open warnings while debug can query retained history.
- status: canonical

### qxctl Validation Policy Store
- path: `tools/qxctl/internal/validation/policy.go`
- title: qxctl Validation Policy Store
- surface_type: implementation source
- truth_role: protected policy and baseline semantic state mechanics
- owner: qxctl maintainers
- scope: Implements exact compare-and-swap, lineage, retry, timestamp, and digest rules.
- relationships: implements -> `knowledge/VALIDATION.md`; conforms_to -> `knowledge/schemas/v1/validation-policy.schema.json`; conforms_to -> `knowledge/schemas/v1/validation-baseline.schema.json`
- consumers: qxctl validation commands
- deferred_projections: remote state service
- notes: Noncanonical state only.
- status: canonical

### qxctl Validation Unix State
- path: `tools/qxctl/internal/validation/state_unix.go`
- title: qxctl Validation Unix State
- surface_type: implementation source
- truth_role: protected Linux and macOS persistence mechanics
- owner: qxctl maintainers
- scope: Implements no-follow traversal, locks, ownership/modes, fsync, and atomic replacement.
- relationships: implements -> `knowledge/VALIDATION.md`; called_by -> `tools/qxctl/internal/validation/policy.go`
- consumers: qxctl validation store
- deferred_projections: native Windows state
- notes: Linux-first with supported macOS development behavior.
- status: canonical

### qxctl Validation Unsupported Platform Boundary
- path: `tools/qxctl/internal/validation/state_unsupported.go`
- title: qxctl Validation Unsupported Platform Boundary
- surface_type: implementation source
- truth_role: fail-closed unsupported native operating-system behavior
- owner: qxctl maintainers
- scope: Prevents weaker state persistence substitution.
- relationships: implements -> `knowledge/VALIDATION.md`; complements -> `tools/qxctl/internal/validation/state_unix.go`
- consumers: qxctl cross-builds
- deferred_projections: native Windows engine
- notes: Windows operation uses WSL or remote qxctl administration.
- status: canonical

### qxctl Validation Tests
- path: `tools/qxctl/internal/validation/validation_test.go`
- title: qxctl Validation Tests
- surface_type: implementation test
- truth_role: compare-and-swap, durability, delta, and fail-closed evidence
- owner: qxctl maintainers
- scope: Verifies protected profiles/baselines and deterministic evaluation.
- relationships: verifies -> `tools/qxctl/internal/validation/policy.go`; verifies -> `tools/qxctl/internal/validation/evaluate.go`; verifies -> `tools/qxctl/internal/validation/state_unix.go`
- consumers: qxctl maintainers, reviewers
- deferred_projections: none
- notes: Covers semantic retry, stale state, incompatible baseline, and symlink rejection.
- status: canonical

### SSFV Engine Canonical Repository Tests
- path: `modules/ssfv-engine/tests/ssfv_test.cpp`
- title: SSFV Engine Canonical Repository Tests
- surface_type: implementation test
- truth_role: canonical-catalog count, graph, hierarchy, freshness, and proposal-boundary evidence
- owner: SSFV engine maintainers
- scope: Verifies the exact partial SSFV catalog together with bounded fixture, graph, proposal, and failure behavior.
- relationships: verifies -> `modules/ssfv-engine/src/ssfv.cpp`; conforms_to -> `knowledge/ssfv/SPEC.md`
- consumers: SSFV engine maintainers, validator, reviewers
- deferred_projections: none
- notes: Current canonical assertions cover seventy-four records and fifteen owner files without claiming completion of the remaining nested review, a frozen derived-edge count, or repository-wide completeness.
- status: canonical

### qxctl SCLV Evidence Adapter Command Tests
- path: `tools/qxctl/cmd/qxctl/sclv_test.go`
- title: qxctl SCLV Evidence Adapter Command Tests
- surface_type: Go conformance-test implementation
- truth_role: typed adapter command, normalized evidence, and authority-boundary proof
- owner: qxctl maintainers
- scope: Verifies local-Git and air-gap result identity, exact field presence, nested evidence semantics, temporal and digest validation, adapter mismatch refusal, and disabled truth, permission, ratification, and canonical-apply claims.
- relationships: verifies -> `tools/qxctl/cmd/qxctl/main.go`; verifies -> `tools/qxctl/cmd/qxctl/commands.go`; verifies -> `tools/qxctl/internal/knowledgeengine/client.go`; conforms_to -> `knowledge/sclv/SPEC.md`
- consumers: qxctl and SCLV maintainers, validator integration, reviewers
- deferred_projections: installed multi-version adapter compatibility matrix
- notes: Successful provider-evidence normalization remains bounded evidence and never grants authority or establishes canonical truth.
- status: canonical

### qxctl Knowledge Engine Client Tests
- path: `tools/qxctl/internal/knowledgeengine/client_test.go`
- title: qxctl Knowledge Engine Client Tests
- surface_type: implementation test
- truth_role: exact receipt, bounded JSON, and installed-process trust evidence
- owner: qxctl maintainers
- scope: Verifies exact installation identity, receipt-owned files, bounded response parsing, validator package inspection, and fail-closed receipt-v2 typed SCLV adapter entry-point enforcement.
- relationships: verifies -> `tools/qxctl/internal/knowledgeengine/client.go`; governed_by -> `tools/qxctl/MANIFEST.md`
- consumers: qxctl maintainers, validator integration, reviewers
- deferred_projections: none
- notes: Does not authorize installation mutation, process activation, or weaker receipt traversal.
- status: canonical

### Symphony Validator CLI Implementation
- path: `tools/symphony-validator/src/cli.cpp`
- title: Symphony Validator CLI Implementation
- surface_type: implementation source
- truth_role: complete detector orchestration, exit status, and line/JSON projection dispatch truth
- owner: validator maintainers
- scope: Runs the full read-only checker set and emits either deterministic line evidence or the complete structured result projection.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; invokes -> `tools/symphony-validator/src/projector.cpp`; conforms_to -> `knowledge/VALIDATION.md`
- consumers: validator executable, qxctl, tests, reviewers
- deferred_projections: additional output formats
- notes: Policy, baselines, filters, remediation, and canonical mutation remain outside the detector process.
- status: canonical

### Maestro Receptor Inventory Command Schema
- path: `knowledge/schemas/v1/maestro-receptor-inventory-command.schema.json`
- title: Maestro Receptor Inventory Command Schema
- surface_type: canonical JSON Schema
- truth_role: authenticated complete derived-inventory request contract truth
- owner: common SKV and Maestro maintainers
- scope: Defines the bounded read-only inventory command, exact TOPS identity, fresh SSIAG evidence, and compatible qxctl client declaration.
- relationships: governed_by -> `modules/maestro/SPEC.md`; implemented_by -> `modules/maestro/src/maestro.cpp`; consumed_by -> `tools/qxctl/internal/maestroclient/client.go`
- consumers: Maestro, qxctl, SSFV maintenance, validator, reviewers
- deferred_projections: inventory analytics and graph views
- notes: The command authorizes no registry mutation, engine execution, or partial inventory.
- status: canonical

### Maestro Receptor Inventory Result Schema
- path: `knowledge/schemas/v1/maestro-receptor-inventory-result.schema.json`
- title: Maestro Receptor Inventory Result Schema
- surface_type: canonical JSON Schema
- truth_role: complete read-only derived receptor inventory evidence truth
- owner: common SKV and Maestro maintainers
- scope: Defines sorted receptor/component evidence, stable timestamp-independent inventory identity, timestamped observation identity, and fail-complete compatibility.
- relationships: governed_by -> `modules/maestro/SPEC.md`; emitted_by -> `modules/maestro/src/maestro.cpp`; consumed_by -> `tools/qxctl/internal/maestroclient/client.go`
- consumers: qxctl, SSFV maintenance, lifecycle administration, validator, reviewers
- deferred_projections: inventory analytics and graph views
- notes: Derived inventory is not a second registry, feature truth, desired state, or execution authority.
- status: canonical

### SSFV Maintenance Command Schema
- path: `knowledge/schemas/v1/ssfv-maintenance-command.schema.json`
- title: SSFV Maintenance Command Schema
- surface_type: canonical JSON Schema
- truth_role: persistent noncanonical semantic-maintenance request contract truth
- owner: common SKV, SSFV, and coordinator maintainers
- scope: Defines begin, status, checkpoint, close, and explicit recovery inputs with exact session, binding, engine, semantic, Maestro, compare-and-swap, and SSIAG evidence.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; implemented_by -> `modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp`; emitted_by -> `tools/qxctl/cmd/qxctl/ssfv_maintenance.go`
- consumers: qxctl, knowledge-session coordinator, validator, reviewers
- deferred_projections: operator review reports
- notes: The command cannot carry canonical apply authority or decide feature-worthiness.
- status: canonical

### SSFV Maintenance Head Schema
- path: `knowledge/schemas/v1/ssfv-maintenance-head.schema.json`
- title: SSFV Maintenance Head Schema
- surface_type: canonical JSON Schema
- truth_role: atomic maintenance-journal selector contract truth
- owner: common SKV, SSFV, and coordinator maintainers
- scope: Defines one exact dual-slot generation and journal digest selector for a protected maintenance context.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; persisted_by -> `modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp`
- consumers: knowledge-session coordinator, recovery tooling, validator, reviewers
- deferred_projections: none
- notes: A head selects noncanonical evidence and is never semantic or authorization truth.
- status: canonical

### SSFV Maintenance Journal Schema
- path: `knowledge/schemas/v1/ssfv-maintenance-journal.schema.json`
- title: SSFV Maintenance Journal Schema
- surface_type: canonical JSON Schema
- truth_role: persistent semantic baseline, checkpoint, lineage, and recovery evidence truth
- owner: common SKV, SSFV, and coordinator maintainers
- scope: Defines immutable baseline snapshot/engine evidence, separate current-engine lineage, linked checkpoints, review disposition, compatibility, extensions, and forward recovery.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; persisted_by -> `modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp`; reads -> `knowledge/ssfv/schemas/v1/semantic-snapshot.schema.json`; reads -> `knowledge/ssfv/schemas/v2/diff-result.schema.json`
- consumers: knowledge-session coordinator, qxctl, recovery tooling, validator, reviewers
- deferred_projections: operator review reports
- notes: Journal state is protected and noncanonical; review-required evidence cannot mutate feature truth.
- status: canonical

### SSFV Maintenance Result Schema
- path: `knowledge/schemas/v1/ssfv-maintenance-result.schema.json`
- title: SSFV Maintenance Result Schema
- surface_type: canonical JSON Schema
- truth_role: bounded maintenance status, mutation, and recovery result truth
- owner: common SKV, SSFV, and coordinator maintainers
- scope: Defines exact journal evidence, effective/review state, repair actions, read-only disposition, and disabled canonical apply.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; emitted_by -> `modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp`; consumed_by -> `tools/qxctl/cmd/qxctl/ssfv_maintenance.go`
- consumers: qxctl, administrators, reviewers, validator
- deferred_projections: operator review reports
- notes: A successful result proves maintenance-state handling, not canonical semantic ratification.
- status: canonical

### Knowledge Session Coordinator SSFV Maintenance Interface
- path: `modules/knowledge-session-coordinator/src/ssfv_maintenance.hpp`
- title: Knowledge Session Coordinator SSFV Maintenance Interface
- surface_type: C++ coordinator implementation surface
- truth_role: persistent SSFV maintenance capability and request-handler boundary
- owner: SKV coordinator and SSFV maintainers
- scope: Declares the bounded compatibility descriptor and maintenance dispatcher without canonical mutation authority.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implemented_by -> `modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp`
- consumers: coordinator dispatcher, tests, reviewers
- deferred_projections: automatic session-close hooks
- notes: The interface grants no feature-worthiness, Maestro-write, or canonical-apply authority.
- status: canonical

### Knowledge Session Coordinator SSFV Maintenance Implementation
- path: `modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp`
- title: Knowledge Session Coordinator Persistent SSFV Maintenance
- surface_type: C++ freezing-path state implementation
- truth_role: baseline durability, upgrade-order lineage, compare-and-swap, review, and recovery implementation truth
- owner: SKV coordinator and SSFV maintainers
- scope: Implements protected per-TOPS/subject/repository dual-slot streams, immutable semantic baselines, separate baseline/current engines, linked checkpoints, explicit Maestro lineage, idempotency, and unique-forward recovery.
- relationships: implements -> `modules/knowledge-session-coordinator/SPEC.md`; implements -> `knowledge/schemas/v1/ssfv-maintenance-command.schema.json`; persists -> `knowledge/schemas/v1/ssfv-maintenance-journal.schema.json`; persists -> `knowledge/schemas/v1/ssfv-maintenance-head.schema.json`; emits -> `knowledge/schemas/v1/ssfv-maintenance-result.schema.json`
- consumers: symphony-knowledge-session, qxctl, tests, reviewers
- deferred_projections: automatic hooks and review-report projections
- notes: Consumes caller-supplied SSFV/Maestro evidence and never invokes an engine, edits canonical files, or writes Maestro state.
- status: canonical

### Knowledge Session Coordinator SSFV Maintenance Tests
- path: `modules/knowledge-session-coordinator/tests/ssfv_maintenance_test.cpp`
- title: Knowledge Session Coordinator SSFV Maintenance Tests
- surface_type: C++26 conformance-test implementation
- truth_role: persistence, replay, compare-and-swap, upgrade-order, corruption, and recovery proof
- owner: SKV coordinator and SSFV maintainers
- scope: Verifies baseline capture, semantic replay, separate engine lineage, stale-state refusal, read-only status, damaged-head detection, and explicit digest-linked forward repair.
- relationships: verifies -> `modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp`; conforms_to -> `knowledge/ssfv/SPEC.md`
- consumers: coordinator and SSFV maintainers, reviewers, release gates
- deferred_projections: large-corpus performance fixtures
- notes: Tests mutate only private temporary noncanonical state.
- status: canonical

### qxctl SSFV Session Maintenance Administration
- path: `tools/qxctl/cmd/qxctl/ssfv_maintenance.go`
- title: qxctl SSFV Session Maintenance Administration
- surface_type: Go freezing-path orchestration source
- truth_role: cross-vector evidence collection, authorization, and maintenance-command composition truth
- owner: qxctl, SSFV, and coordinator maintainers
- scope: Implements `knowledge session features begin|status|checkpoint|close|recover` using exact coordinator/SSFV bindings, open-session evidence, read-only SSFV operations, optional complete Maestro inventory, and fresh SSIAG decisions.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; invokes -> `modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp`; invokes -> `modules/ssfv-engine/src/ssfv.cpp`; invokes -> `modules/maestro/src/maestro.cpp`
- consumers: qxctl users, administrators, tests, reviewers
- deferred_projections: automatic hooks and canonical apply
- notes: qxctl composes evidence but does not decide semantic truth or bypass review.
- status: canonical

### qxctl SSFV Session Maintenance Tests
- path: `tools/qxctl/cmd/qxctl/ssfv_maintenance_test.go`
- title: qxctl SSFV Session Maintenance Tests
- surface_type: Go conformance-test implementation
- truth_role: grammar, authorization binding, digest, and canonical-boundary proof
- owner: qxctl, SSFV, and coordinator maintainers
- scope: Verifies the command tree, operation resource, recursive canonical evidence identity, canonical-apply refusal, and exact journal-digest validation.
- relationships: verifies -> `tools/qxctl/cmd/qxctl/ssfv_maintenance.go`; conforms_to -> `knowledge/ssfv/SPEC.md`
- consumers: qxctl maintainers, reviewers, release gates
- deferred_projections: installed multi-version integration fixture
- notes: The tests authorize no canonical mutation or automatic session hook.
- status: canonical

### Knowledge Engine Foundation Receipt-v2 Uninstall Template
- path: `libraries/knowledge-vector-engine-cpp/cmake/uninstall.cmake.in`
- title: Knowledge Engine Foundation Receipt-v2 Uninstall Template
- surface_type: CMake package-lifecycle implementation template
- truth_role: receipt-owned integrity validation and idempotent removal implementation truth
- owner: common SKV engine-foundation maintainers
- scope: Generates the build-local uninstaller that validates receipt-v2 ownership, remaining-file digests, and receipt-last removal for the installed C++ foundation package.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `libraries/knowledge-vector-engine-cpp/INSTALL.md`
- consumers: CMake, qxctl lifecycle administration, package maintainers, tests
- deferred_projections: package-manager-native ownership claim adapters
- notes: The generated uninstaller refuses a present package when qxctl's root registry exists; authenticated qxctl lifecycle apply owns cross-profile reclamation.
- status: canonical

### Knowledge Engine Process Protocol Interface
- path: `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/protocol.hpp`
- title: Knowledge Engine Process Protocol Interface
- surface_type: C++26 foundation interface
- truth_role: bounded engine-process protocol and receipt-version compatibility implementation truth
- owner: common SKV engine-foundation maintainers
- scope: Declares shared engine-process parsing, response, descriptor, and supported immutable receipt-version constants.
- relationships: implements -> `knowledge/SPEC.md`; consumed_by -> independently installed C++ knowledge engines
- consumers: C++ vector engines, coordinator, Maestro, Symphony Validator, tests
- deferred_projections: additional compatible receipt readers
- notes: A supported reader version does not authorize package mutation.
- status: canonical

### Knowledge Session Coordinator Receipt-v2 Install Template
- path: `modules/knowledge-session-coordinator/cmake/install-receipt.json.in`
- title: Knowledge Session Coordinator Receipt-v2 Install Template
- surface_type: immutable package-receipt template
- truth_role: coordinator package identity and owned-file installation truth
- owner: knowledge-session coordinator maintainers
- scope: Defines the coordinator's exact receipt-v2 package identity, entry point, descriptors, and digest-bound owned paths.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/knowledge-session-coordinator/INSTALL.md`
- consumers: CMake, qxctl lifecycle observation and apply, validators, administrators
- deferred_projections: package catalog
- notes: Installation presence is distinct from binding, activation, or execution.
- status: canonical

### Knowledge Session Coordinator Receipt-v2 Uninstall Template
- path: `modules/knowledge-session-coordinator/cmake/uninstall.cmake.in`
- title: Knowledge Session Coordinator Receipt-v2 Uninstall Template
- surface_type: CMake package-lifecycle implementation template
- truth_role: coordinator receipt-owned integrity validation and idempotent removal truth
- owner: knowledge-session coordinator maintainers
- scope: Generates the exact receipt-v2 coordinator uninstaller with staged rollback proof and receipt-last removal.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/knowledge-session-coordinator/INSTALL.md`
- consumers: CMake, qxctl lifecycle administration, tests
- deferred_projections: package-manager-native ownership claim adapters
- notes: Uninstall does not remove protected journals or unrelated administrator files and refuses a present package when qxctl's root registry exists.
- status: canonical

### SACV Engine Receipt-v2 Install Template
- path: `modules/sacv-engine/cmake/install-receipt.json.in`
- title: SACV Engine Receipt-v2 Install Template
- surface_type: immutable package-receipt template
- truth_role: SACV engine package identity and owned-file installation truth
- owner: SACV engine maintainers
- scope: Defines exact receipt-v2 package and owned-file evidence for one independently installed SACV engine version.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/sacv-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle observation and apply, validators
- deferred_projections: package catalog
- notes: Receipt identity does not imply selection, activation, docking, or execution.
- status: canonical

### SACV Engine Receipt-v2 Uninstall Template
- path: `modules/sacv-engine/cmake/uninstall.cmake.in`
- title: SACV Engine Receipt-v2 Uninstall Template
- surface_type: CMake package-lifecycle implementation template
- truth_role: SACV receipt-owned integrity validation and idempotent removal truth
- owner: SACV engine maintainers
- scope: Generates the receipt-v2 SACV uninstaller with exact owned-file validation and receipt-last removal.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/sacv-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle administration, tests
- deferred_projections: package-manager-native ownership claim adapters
- notes: The package-local uninstaller refuses a present package when qxctl's root registry exists; authenticated lifecycle apply owns reclamation.
- status: canonical

### SCLV Engine Receipt-v2 Install Template
- path: `modules/sclv-engine/cmake/install-receipt.json.in`
- title: SCLV Engine Receipt-v2 Install Template
- surface_type: immutable package-receipt template
- truth_role: SCLV engine package identity and owned-file installation truth
- owner: SCLV engine maintainers
- scope: Defines exact receipt-v2 package and owned-file evidence for one independently installed SCLV engine version and its evidence adapters.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/sclv-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle observation and apply, validators
- deferred_projections: package catalog
- notes: Receipt identity does not imply selection, activation, docking, or execution.
- status: canonical

### SCLV Engine Receipt-v2 Uninstall Template
- path: `modules/sclv-engine/cmake/uninstall.cmake.in`
- title: SCLV Engine Receipt-v2 Uninstall Template
- surface_type: CMake package-lifecycle implementation template
- truth_role: SCLV receipt-owned integrity validation and idempotent removal truth
- owner: SCLV engine maintainers
- scope: Generates the receipt-v2 SCLV uninstaller with exact owned-file validation and receipt-last removal.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/sclv-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle administration, tests
- deferred_projections: package-manager-native ownership claim adapters
- notes: The package-local uninstaller refuses a present package when qxctl's root registry exists; authenticated lifecycle apply owns reclamation.
- status: canonical

### SKVI Engine Receipt-v2 Install Template
- path: `modules/skvi-engine/cmake/install-receipt.json.in`
- title: SKVI Engine Receipt-v2 Install Template
- surface_type: immutable package-receipt template
- truth_role: SKVI engine package identity and owned-file installation truth
- owner: SKVI engine maintainers
- scope: Defines exact receipt-v2 package and owned-file evidence for one independently installed SKVI engine version.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/skvi-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle observation and apply, validators
- deferred_projections: package catalog
- notes: Receipt identity does not imply selection, activation, docking, or execution.
- status: canonical

### SKVI Engine Receipt-v2 Uninstall Template
- path: `modules/skvi-engine/cmake/uninstall.cmake.in`
- title: SKVI Engine Receipt-v2 Uninstall Template
- surface_type: CMake package-lifecycle implementation template
- truth_role: SKVI receipt-owned integrity validation and idempotent removal truth
- owner: SKVI engine maintainers
- scope: Generates the receipt-v2 SKVI uninstaller with exact owned-file validation and receipt-last removal.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/skvi-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle administration, tests
- deferred_projections: package-manager-native ownership claim adapters
- notes: The package-local uninstaller refuses a present package when qxctl's root registry exists; authenticated lifecycle apply owns reclamation.
- status: canonical

### SODV Engine Receipt-v2 Install Template
- path: `modules/sodv-engine/cmake/install-receipt.json.in`
- title: SODV Engine Receipt-v2 Install Template
- surface_type: immutable package-receipt template
- truth_role: SODV engine package identity and owned-file installation truth
- owner: SODV engine maintainers
- scope: Defines exact receipt-v2 package and owned-file evidence for one independently installed SODV engine version.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/sodv-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle observation and apply, validators
- deferred_projections: package catalog
- notes: Receipt identity does not imply selection, activation, docking, or execution.
- status: canonical

### SODV Engine Receipt-v2 Uninstall Template
- path: `modules/sodv-engine/cmake/uninstall.cmake.in`
- title: SODV Engine Receipt-v2 Uninstall Template
- surface_type: CMake package-lifecycle implementation template
- truth_role: SODV receipt-owned integrity validation and idempotent removal truth
- owner: SODV engine maintainers
- scope: Generates the receipt-v2 SODV uninstaller with exact owned-file validation and receipt-last removal.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/sodv-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle administration, tests
- deferred_projections: package-manager-native ownership claim adapters
- notes: The package-local uninstaller refuses a present package when qxctl's root registry exists; authenticated lifecycle apply owns reclamation.
- status: canonical

### SSFV Engine Receipt-v2 Install Template
- path: `modules/ssfv-engine/cmake/install-receipt.json.in`
- title: SSFV Engine Receipt-v2 Install Template
- surface_type: immutable package-receipt template
- truth_role: SSFV engine package identity and owned-file installation truth
- owner: SSFV engine maintainers
- scope: Defines exact receipt-v2 package and owned-file evidence for one independently installed SSFV engine version.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/ssfv-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle observation and apply, validators
- deferred_projections: package catalog
- notes: Receipt identity does not imply selection, activation, docking, or execution.
- status: canonical

### SSFV Engine Receipt-v2 Uninstall Template
- path: `modules/ssfv-engine/cmake/uninstall.cmake.in`
- title: SSFV Engine Receipt-v2 Uninstall Template
- surface_type: CMake package-lifecycle implementation template
- truth_role: SSFV receipt-owned integrity validation and idempotent removal truth
- owner: SSFV engine maintainers
- scope: Generates the receipt-v2 SSFV uninstaller with exact owned-file validation and receipt-last removal.
- relationships: implements -> `knowledge/LIFECYCLE.md`; governed_by -> `modules/ssfv-engine/INSTALL.md`
- consumers: CMake, qxctl lifecycle administration, tests
- deferred_projections: package-manager-native ownership claim adapters
- notes: The package-local uninstaller refuses a present package when qxctl's root registry exists; authenticated lifecycle apply owns reclamation.
- status: canonical

### qxctl Lifecycle Binding Transition Tests
- path: `tools/qxctl/cmd/qxctl/lifecycle_binding_test.go`
- title: qxctl Lifecycle Binding Transition Tests
- surface_type: Go conformance-test implementation
- truth_role: exact side-by-side binding-switch and rollback proof
- owner: qxctl and lifecycle maintainers
- scope: Verifies exact receipt-v2 selection changes, inverse rollback, expected-state enforcement, and refusal of unsupported binding identities.
- relationships: verifies -> `tools/qxctl/cmd/qxctl/lifecycle_apply.go`; conforms_to -> `knowledge/LIFECYCLE.md`
- consumers: qxctl maintainers, reviewers, release gates
- deferred_projections: installed multi-version host matrix
- notes: The tests use private fixtures and grant no live-host authority.
- status: canonical

### qxctl SSIAG Lifecycle Grant Tests
- path: `tools/qxctl/cmd/qxctl/ssiag_test.go`
- title: qxctl SSIAG Lifecycle Grant Tests
- surface_type: Go conformance-test implementation
- truth_role: deterministic caller-neutral lifecycle grant-plan proof
- owner: qxctl and SSIAG maintainers
- scope: Verifies exact lifecycle operation/resource grant proposals, bounded identities, and disabled policy apply.
- relationships: verifies -> `tools/qxctl/cmd/qxctl/main.go`; conforms_to -> `knowledge/ssiag/SPEC.md`
- consumers: qxctl and SSIAG maintainers, reviewers, release gates
- deferred_projections: separately gated SSIAG policy administration
- notes: A generated plan is proposal evidence and never mutates SSIAG policy.
- status: canonical

### SSFV Coverage Contract
- path: `knowledge/ssfv/COVERAGE.md`
- title: Symphony Semantic Feature Vector Coverage
- surface_type: canonical coverage inventory
- truth_role: explicit SSFV source-universe, exclusion, freshness, and completion truth
- owner: SSFV maintainers
- scope: Enumerates every top-level application owner scope, three proposal-only exclusions, and the conditions required before coverage may become complete.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; routes_through -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, Symphony Validator, qxctl, maintainers, reviewers, agentic tools
- deferred_projections: remaining nested feature-adjudication inventory
- notes: Top-level owner routing and fifty-nine exact nested subfeatures are covered with reviewed F2 and F3 non-feature dispositions; remaining nested feature, subfeature, microfeature, and non-feature review is incomplete, so coverage is partial.
- status: canonical

### STAV Protocol Kernel Semantic Feature Record
- path: `libraries/stav-protocol-go/FEATURES.md`
- title: STAV Protocol Kernel Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the authority-free Go protocol kernel
- owner: STAV protocol maintainers
- scope: Owns `ssfv:symphony:stav-protocol-kernel` plus the F2 canonical-wire-representation and semantic-validation subfeatures for exact source scope `libraries/stav-protocol-go`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `libraries/stav-protocol-go/MANIFEST.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, STAV implementers, reviewers, agentic tools
- deferred_projections: feature catalog and portable graph
- notes: Records canonical bytes, digests, exact validation, and bounded local IPC framing without append, persistence, supervision, or authorization authority; checksummed durable ledger framing belongs to the append authority.
- status: canonical

### STAV Append Authority Semantic Feature Record
- path: `modules/stav-append-authority/FEATURES.md`
- title: STAV Append Authority Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for serialized durable STAV append and query behavior
- owner: STAV append-authority maintainers
- scope: Owns `ssfv:symphony:stav-append-authority` plus five F2 subfeatures for TOPS enrollment, serialized append, ledger durability, authorized query, and native supervision in exact source scope `modules/stav-append-authority`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/stav-append-authority/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, STAV maintainers, reviewers, agentic tools
- deferred_projections: feature catalog and portable graph
- notes: Records the parent and five separately identified implemented audit-authority subsystems without granting producer-owned ledger fields, direct file authority, or policy authority.
- status: canonical

### SSIAG Foundation Semantic Feature Record
- path: `modules/secure-identity-access-governance/FEATURES.md`
- title: SSIAG Foundation Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for local identity, endpoint trust, decisions, and audit production
- owner: SSIAG foundation maintainers
- scope: Owns `ssfv:symphony:ssiag-foundation` plus seven F2 subfeatures for TOPS enrollment, kernel peer trust, authorization capabilities, policy administration, configured provider metadata, safe audit production, and native supervision in exact source scope `modules/secure-identity-access-governance`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/secure-identity-access-governance/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, SSIAG maintainers, reviewers, agentic tools
- deferred_projections: feature catalog and portable graph
- notes: Records the parent and seven separately identified implemented security subsystems; operational provider execution, credential delivery, general safeguards, audit-deferred recovery, and canonical knowledge mutation remain unavailable.
- status: canonical

### SSIAG macOS Keychain Metadata Semantic Feature Record
- path: `modules/ssiag-provider-macos-keychain/FEATURES.md`
- title: SSIAG macOS Keychain Metadata Semantic Features
- surface_type: distributed SSFV subfeature record
- truth_role: canonical semantic truth for the isolated metadata-only Swift provider adapter
- owner: SSIAG macOS provider maintainers
- scope: Owns `ssfv:symphony:ssiag.macos-keychain-metadata` for exact source scope `modules/ssiag-provider-macos-keychain`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/ssiag-provider-macos-keychain/SPEC.md`; parent -> `ssfv:symphony:ssiag-foundation`
- consumers: symphony-ssfv, qxctl, provider maintainers, reviewers, agentic tools
- deferred_projections: provider capability catalog
- notes: The independently installed Swift adapter remains metadata-only; the Go foundation and qxctl do not invoke it, and operational trust, execution, Keychain access, and secret delivery remain deliberately unimplemented.
- status: canonical

### SKVI Engine Semantic Feature Record
- path: `modules/skvi-engine/FEATURES.md`
- title: SKVI Engine Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the independently installed SKVI engine
- owner: SKVI engine maintainers
- scope: Owns `ssfv:symphony:skvi-engine` for exact source scope `modules/skvi-engine`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/skvi-engine/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, SKVI maintainers, reviewers, agentic tools
- deferred_projections: feature catalog and portable graph
- notes: Records bounded evidence and proposal behavior without canonical apply or membership authority.
- status: canonical

### SCLV Engine Semantic Feature Record
- path: `modules/sclv-engine/FEATURES.md`
- title: SCLV Engine Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the independently installed SCLV engine
- owner: SCLV engine maintainers
- scope: Owns `ssfv:symphony:sclv-engine` for exact source scope `modules/sclv-engine`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/sclv-engine/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, SCLV maintainers, reviewers, agentic tools
- deferred_projections: feature catalog and portable graph
- notes: Records change-truth evidence behavior without ratification, Git-provider dependency, or canonical append authority.
- status: canonical

### SACV Engine Semantic Feature Record
- path: `modules/sacv-engine/FEATURES.md`
- title: SACV Engine Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the independently installed SACV engine
- owner: SACV engine maintainers
- scope: Owns `ssfv:symphony:sacv-engine` for exact source scope `modules/sacv-engine`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/sacv-engine/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, SACV maintainers, reviewers, agentic tools
- deferred_projections: feature catalog and portable graph
- notes: Records API-contract governance behavior without claiming an endpoint, SDK, publication, or canonical apply.
- status: canonical

### SAV Engine Semantic Feature Record
- path: `modules/sav-engine/FEATURES.md`
- title: SAV Engine Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the independently installed SAV engine
- owner: SAV engine maintainers
- scope: Owns `ssfv:symphony:sav-engine` for exact source scope `modules/sav-engine`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/sav-engine/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, SAV maintainers, reviewers, agentic tools
- deferred_projections: composition accord catalog and disposable graph
- notes: Records immutable CURRENT resolution and relationship-evaluation behavior without host discovery, canonical mutation, or apply authority.
- status: canonical

### SEV Engine Semantic Feature Record
- path: `modules/sev-engine/FEATURES.md`
- title: SEV Engine Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the independently installed SEV engine and SCSEV profile
- owner: SEV engine maintainers
- scope: Owns `ssfv:symphony:sev-engine` for exact source scope `modules/sev-engine`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/sev-engine/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, SEV and SCSEV maintainers, reviewers, agentic tools
- deferred_projections: evolution-case catalog, command-surface consequence evidence, and disposable graph
- notes: Records proposal-only evolution and SCSEV consequence assessment without durable session, execution, authorization, or canonical mutation authority.
- status: canonical

### SODV Engine Semantic Feature Record
- path: `modules/sodv-engine/FEATURES.md`
- title: SODV Engine Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the independently installed SODV engine
- owner: SODV engine maintainers
- scope: Owns `ssfv:symphony:sodv-engine` for exact source scope `modules/sodv-engine`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/sodv-engine/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, SODV maintainers, reviewers, agentic tools
- deferred_projections: feature catalog and portable graph
- notes: Records provider-neutral release evidence behavior without network, tag, upload, or completion authority.
- status: canonical

### SSFV Engine Semantic Feature Record
- path: `modules/ssfv-engine/FEATURES.md`
- title: SSFV Engine Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the independently installed SSFV engine
- owner: SSFV engine maintainers
- scope: Owns `ssfv:symphony:ssfv-engine` for exact source scope `modules/ssfv-engine`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `modules/ssfv-engine/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, SSFV maintainers, reviewers, agentic tools
- deferred_projections: feature catalog and portable graph
- notes: Records semantic evidence and proposal behavior without feature-worthiness or canonical mutation authority.
- status: canonical

### qxctl Semantic Feature Record
- path: `tools/qxctl/FEATURES.md`
- title: qxctl Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for the Go administrative and query CLI
- owner: qxctl maintainers
- scope: Owns the experimental qxctl feature plus nine ratified F1 subfeatures for engine bindings, authenticated sessions, lifecycle convergence, the Linux host receptor, SSIAG/STAV/Maestro administration, governed validation, and invariant assurance in exact source scope `tools/qxctl`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `tools/qxctl/MANIFEST.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`; declares -> nine `ssfv:symphony:qxctl.*` F1 subfeatures
- consumers: symphony-ssfv, qxctl maintainers, reviewers, agentic tools
- deferred_projections: command and capability catalog
- notes: Records the parent administrative interface and nine separately identified implemented subsystems without transferring vector-schema, policy, ledger, coordinator, Maestro, validator-detection, or runtime authority to the CLI.
- status: canonical

### Symphony Validator Semantic Feature Record
- path: `tools/symphony-validator/FEATURES.md`
- title: Symphony Validator Semantic Features
- surface_type: distributed SSFV feature record
- truth_role: canonical semantic feature truth for deterministic read-only repository validation
- owner: Symphony Validator maintainers
- scope: Owns `ssfv:symphony:symphony-validator` plus root-summary and invariant-ownership assurance subfeatures for exact source scope `tools/symphony-validator`.
- relationships: governed_by -> `knowledge/ssfv/SPEC.md`; owned_by -> `tools/symphony-validator/SPEC.md`; registered_in -> `knowledge/ssfv/REGISTRY.md`
- consumers: symphony-ssfv, qxctl, validator maintainers, reviewers, agentic tools
- deferred_projections: Markdown validation projection
- notes: Records deterministic line and JSON evidence, source-module admission, and derived-summary behavior without remediation, ratification, or runtime residency.
- status: canonical

### SSIAG Policy Implementation
- path: `modules/secure-identity-access-governance/internal/policy/policy.go`
- title: SSIAG Authorization Policy Implementation
- surface_type: Go authorization implementation
- truth_role: exact caller-neutral local allow/deny decision implementation truth
- owner: SSIAG foundation maintainers
- scope: Derives the authenticated subject from local peer evidence, evaluates exact operation/resource grants, and releases decisions only after safe STAV evidence commits.
- relationships: implements -> `knowledge/ssiag/SPEC.md`; conforms_to -> `knowledge/ssiag/schemas/v1/authorization-decision.schema.json`
- consumers: SSIAG server, qxctl, coordinator, tests, reviewers
- deferred_projections: policy explanation projection
- notes: The evaluator atomically exchanges one complete validated snapshot after protected state commit.
- status: canonical

### SSIAG Policy Administration Implementation
- path: `modules/secure-identity-access-governance/internal/policyadmin/manager.go`
- title: SSIAG Protected Local Policy Administration
- surface_type: Go operational state-machine implementation
- truth_role: exact proposal, CAS, attempt, commit, reset, and recovery implementation truth
- owner: SSIAG foundation maintainers
- scope: Selects config or overlay policy and binds target-host authority to durable audit-before-commit local mutation.
- relationships: implements -> `knowledge/ssiag/SPEC.md`; conforms_to -> `knowledge/ssiag/schemas/v1/policy-proposal.schema.json`; persists -> `knowledge/ssiag/schemas/v1/policy-state.schema.json`; persists -> `knowledge/ssiag/schemas/v1/policy-attempt.schema.json`
- consumers: SSIAG server, qxctl, tests, reviewers
- deferred_projections: policy explanation and history projection
- notes: Operational per-TOPS state only; canonical knowledge and enrolled config remain unchanged.
- status: canonical

### SSIAG Policy State Storage Implementation
- path: `modules/secure-identity-access-governance/internal/policyadmin/storage_unix.go`
- title: SSIAG Protected Policy State Storage
- surface_type: Go Darwin/Linux durable storage implementation
- truth_role: no-follow locked atomic persistence truth
- owner: SSIAG foundation maintainers
- scope: Opens owner-controlled per-TOPS state without following path components and serializes bounded fsynced state replacement.
- relationships: implements -> `knowledge/ssiag/schemas/v1/policy-state.schema.json`; implements -> `knowledge/ssiag/schemas/v1/policy-attempt.schema.json`
- consumers: SSIAG policy administration, tests, reviewers
- deferred_projections: none
- notes: Linux-first and macOS-supported; no Windows engine is provided.
- status: canonical

### SSIAG Provider Trust Implementation
- path: `modules/secure-identity-access-governance/internal/provider/trust.go`
- title: SSIAG Provider Trust Manager
- surface_type: Go provider provenance and mutual-verification implementation
- truth_role: exact protected declaration, receipt, foundation, handshake, and safe-result validation truth
- owner: SSIAG foundation maintainers
- scope: Implements snapshot inspection and permission-backed fresh verification for one exact receipt-bound metadata adapter without enabling provider operations.
- relationships: implements -> `knowledge/ssiag/SPEC.md`; conforms_to -> `knowledge/ssiag/schemas/v1/provider-executable-trust.schema.json`; conforms_to -> `knowledge/ssiag/schemas/v1/provider-trust-result.schema.json`
- consumers: SSIAG server, qxctl, provider launcher, tests, reviewers
- deferred_projections: additional provider kinds after separate contract extension
- notes: A named signing identity remains untrusted until an independent platform verifier is ratified and implemented.
- status: canonical

### SSIAG Provider Binding Lifecycle Implementation
- path: `modules/secure-identity-access-governance/internal/provider/binding.go`
- title: SSIAG Exact Provider Binding Manager
- surface_type: Go operational state-machine implementation
- truth_role: exact installation inventory, binding plan, candidate verification, audit-before-commit, compare-and-swap, and recovery implementation truth
- owner: SSIAG foundation maintainers
- scope: Catalogs admitted receipt-v2 foundation/adapter pairs, administers one current and one predecessor binding per TOPS, and advances only uniquely evidenced `prepared`, `candidate_verified`, `audited`, and `committed` transitions.
- relationships: implements -> `knowledge/ssiag/PROVIDER-LIFECYCLE.md`; persists_with -> `modules/secure-identity-access-governance/internal/provider/binding_storage_unix.go`; tested_by -> `modules/secure-identity-access-governance/internal/provider/binding_test.go`; consumed_by -> `modules/secure-identity-access-governance/internal/server/server.go`; consumed_by -> `tools/qxctl/internal/ssiagclient/provider_binding.go`
- consumers: SSIAG server, qxctl, installed-foundation offline recovery, tests, reviewers
- deferred_projections: provider-binding history and installed-host graph projections
- notes: No newest-version selection, path disclosure, implicit fallback, provider operation, Keychain access, secret channel, or alternate recovery authority is enabled.
- status: canonical

### SSIAG Provider Process Launcher
- path: `modules/secure-identity-access-governance/internal/provider/launcher.go`
- title: SSIAG Bounded Provider Process Launcher
- surface_type: Go out-of-process adapter boundary implementation
- truth_role: exact verified-byte staging, sanitized environment, deadline, framing, and cleanup truth
- owner: SSIAG foundation maintainers
- scope: Executes a private copy of the exact verified adapter bytes for one bounded request and response, rejecting extra output and process failure without exposing diagnostics.
- relationships: implements -> `knowledge/ssiag/schemas/v1/provider-one-shot-channel.schema.json`; consumes -> `knowledge/ssiag/schemas/v1/provider-control-response.schema.json`
- consumers: SSIAG provider trust manager, tests, security reviewers
- deferred_projections: separately authorized provider-operation and secret-channel launchers
- notes: The launcher carries only safe metadata; standard error is bounded and never returned to the caller.
- status: canonical

### SSIAG Provider Receipt Verification
- path: `modules/secure-identity-access-governance/internal/provider/receipt.go`
- title: SSIAG Exact Provider Receipt Verifier
- surface_type: Go receipt-v2 and executable-provenance implementation
- truth_role: exact installed adapter identity, platform, capability, ownership, size, and digest verification truth
- owner: SSIAG foundation maintainers
- scope: Validates one explicitly bound legacy executable or exact multi-file provider bundle before complete bounded private staging; it performs no discovery or newest-version selection.
- relationships: implements -> `knowledge/schemas/v2/install-receipt.schema.json`; governed_by -> `knowledge/ssiag/SPEC.md`; composes_with -> `modules/secure-identity-access-governance/internal/provider/launcher.go`
- consumers: SSIAG provider trust manager, invariant ownership assurance, tests, security reviewers
- deferred_projections: production signing/notarization release evidence
- notes: All receipt members are explicit, paths are sorted and unique, unknown bundle files fail closed, hashing is bounded, and operational provider access remains disabled.
- status: canonical

### SSIAG Provider Readiness Implementation
- path: `modules/secure-identity-access-governance/internal/provider/readiness.go`
- title: SSIAG Three-Layer Provider Readiness Manager
- surface_type: Go safe readiness process and result implementation
- truth_role: exact installation-bound observation, validation, and disabled operational-eligibility truth
- owner: SSIAG foundation maintainers
- scope: Invokes only the fixed readiness operation through complete private staging, validates the closed native observation, and binds it to one TOPS, provider, installation, UTC time, and result digest.
- relationships: implements -> `knowledge/ssiag/PROVIDER-READINESS.md`; conforms_to -> `knowledge/ssiag/schemas/v1/provider-readiness-result.schema.json`; consumes -> `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/SignedBundleReadiness.swift`
- consumers: SSIAG server, qxctl, tests, security reviewers
- deferred_projections: readiness history and future operational-eligibility evaluation
- notes: Structural and policy success can produce only `readiness_proven_operations_disabled`; all provider operations remain false.
- status: canonical

### SSIAG macOS Provider Foundation Trust
- path: `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/FoundationTrust.swift`
- title: SSIAG macOS Adapter Invoker Trust Verification
- surface_type: Swift parent-process provenance implementation
- truth_role: exact independent foundation executable and receipt verification truth
- owner: SSIAG macOS provider maintainers
- scope: Verifies the invoking Go foundation process, protected package ancestry, receipt-v2 identity, and exact executable digest before metadata response release.
- relationships: implements -> `knowledge/ssiag/SPEC.md`; composes_with -> `modules/secure-identity-access-governance/internal/provider/trust.go`
- consumers: macOS provider process, tests, security reviewers
- deferred_projections: independently verified Apple signing identity after separate ratification
- notes: Direct or unreceipted invocation fails closed; operational Keychain access remains disabled.
- status: canonical

### SSIAG macOS Provider Receipt Implementation
- path: `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/ReceiptV2.swift`
- title: SSIAG macOS Adapter Receipt-v2 Implementation
- surface_type: Swift receipt-v2 installation and provenance implementation
- truth_role: exact side-by-side adapter receipt identity and independent Go-foundation receipt verification truth
- owner: SSIAG macOS provider maintainers
- scope: Emits and validates legacy single-file or exact sorted multi-file bundle receipts and independently validates the invoking foundation receipt, platform, ownership, and executable digest.
- relationships: implements -> `knowledge/schemas/v2/install-receipt.schema.json`; governed_by -> `knowledge/ssiag/SPEC.md`; composes_with -> `modules/secure-identity-access-governance/internal/provider/receipt.go`
- consumers: macOS provider lifecycle, foundation trust verifier, invariant ownership assurance, tests, security reviewers
- deferred_projections: production Apple signing and notarization release evidence
- notes: macOS fixed root compatibility aliases are independently resolved and verified; aliases do not relax executable or receipt identity.
- status: canonical

### SSIAG macOS Signed-Bundle Readiness Implementation
- path: `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/SignedBundleReadiness.swift`
- title: SSIAG macOS Native Signed-Bundle Readiness
- surface_type: Swift Apple Security structural, requirement, and session implementation
- truth_role: decision-neutral native readiness observation truth
- owner: SSIAG macOS provider maintainers
- scope: Validates complete code-signature structure, compiles and evaluates one receipt-owned native requirement, observes bounded Security Session flags, and fixes operational eligibility false.
- relationships: implements -> `knowledge/ssiag/PROVIDER-READINESS.md`; conforms_to -> `knowledge/ssiag/schemas/v1/provider-readiness-observation.schema.json`; consumed_by -> `modules/secure-identity-access-governance/internal/provider/readiness.go`
- consumers: SSIAG readiness manager, qxctl through SSIAG, tests, release engineers, security reviewers
- deferred_projections: Phase 10C data-protection Keychain operational eligibility
- notes: It requests or emits no certificate/profile/entitlement payload, native error, requirement text, session ID, provider payload, Keychain item, key operation, or secret.
- status: canonical

### SSIAG macOS Production Bundle Builder
- path: `modules/ssiag-provider-macos-keychain/scripts/build-production-bundle.sh`
- title: SSIAG macOS Production Bundle Builder
- surface_type: macOS release packaging and signing tool
- truth_role: complete-before-signing Developer ID, hardened runtime, timestamp, notarization, and stapling procedure
- owner: SSIAG macOS provider maintainers
- scope: Builds a minimal app-like bundle from reviewed external identity/policy evidence, signs after all inputs are present, verifies, notarizes, staples, and emits one immutable bundle for receipt-v2 installation.
- relationships: implements -> `knowledge/ssiag/PROVIDER-READINESS.md`; produces_for -> `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/Lifecycle.swift`
- consumers: release engineers, security reviewers, installation tooling
- deferred_projections: signed release attestations governed by SODV
- notes: No Team ID, bundle ID, entitlement, access group, requirement, profile, or notary credential is hardcoded into Symphony source.
- status: canonical

### qxctl SSIAG Provider Readiness Client
- path: `tools/qxctl/internal/ssiagclient/provider_readiness.go`
- title: qxctl SSIAG Provider Readiness Client
- surface_type: Go authenticated safe-result consumer
- truth_role: exact request and closed response validation truth
- owner: qxctl maintainers
- scope: Sends one permission-backed readiness observation request to the authenticated SSIAG Unix endpoint and rejects identity, schema, digest, time, reason, layer, or operational-flag drift.
- relationships: conforms_to -> `knowledge/ssiag/schemas/v1/provider-readiness-observation-request.schema.json`; conforms_to -> `knowledge/ssiag/schemas/v1/provider-readiness-result.schema.json`; implements -> `knowledge/ssiag/PROVIDER-READINESS.md`
- consumers: qxctl readiness command, tests, administrators, headless agents
- deferred_projections: rendered readiness diagnostics
- notes: It accepts no adapter path, signing evidence, provider payload, or secret and never launches the adapter directly.
- status: canonical

### SSIAG Receipt-Backed Provider Integration Regression
- path: `modules/secure-identity-access-governance/tests/provider_trust_integration_darwin.sh`
- title: SSIAG Actual Go-to-Swift Mutual-Trust Integration
- surface_type: Darwin receipt-backed real-process integration test
- truth_role: end-to-end installed foundation, installed adapter, Unix service, qxctl, and mutual-trust regression evidence
- owner: SSIAG foundation and macOS provider maintainers
- scope: Builds and receipts both real executables in a disposable prefix, binds one protected provider declaration, invokes fresh verification through qxctl, and requires both trust directions with every operational flag false.
- relationships: validates -> `knowledge/ssiag/SPEC.md`; exercises -> `modules/secure-identity-access-governance/internal/provider/launcher.go`; exercises -> `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/FoundationTrust.swift`
- consumers: invariant ownership assurance, release gates, maintainers, security reviewers
- deferred_projections: Linux native-provider integration after a separate provider contract exists
- notes: Shell fixtures, mocks, direct adapter invocation, and unreceipted parent processes cannot satisfy this gate.
- status: canonical

### qxctl SSIAG Client
- path: `tools/qxctl/internal/ssiagclient/client.go`
- title: qxctl SSIAG Client
- surface_type: Go authenticated local-client implementation
- truth_role: exact SSIAG endpoint authentication and decision-validation truth
- owner: qxctl and SSIAG maintainers
- scope: Authenticates the configured local SSIAG endpoint and validates operation-bound caller-neutral authorization evidence plus exact provider-trust snapshot and fresh-verification results.
- relationships: conforms_to -> `knowledge/ssiag/schemas/v1/authorization-decision.schema.json`; conforms_to -> `knowledge/ssiag/schemas/v1/provider-trust-result.schema.json`; implements -> `tools/qxctl/MANIFEST.md`
- consumers: qxctl protected commands, tests, reviewers
- deferred_projections: remote node transport under a future contract
- notes: The client never reads provider trust files or invokes adapters directly; the authenticated SSIAG server independently proves authority and provider evidence.
- status: canonical

### Shared-Root Ownership Schema
- path: `knowledge/schemas/v1/lifecycle-root-ownership.schema.json`
- title: Shared-Root Lifecycle Ownership Registry Schema
- surface_type: common SKV JSON Schema
- truth_role: canonical root-local multi-profile receipt claim and reclamation protocol truth
- owner: Symphony Knowledge Vector umbrella maintainers
- scope: Defines enforced/adoption-required generations, retained/retiring/legacy claims, observed receipts, explicit releases, and digests.
- relationships: governed_by -> `knowledge/LIFECYCLE.md`; implemented_by -> `tools/qxctl/internal/knowledgelifecycle/ownership.go`
- consumers: qxctl lifecycle administration, package executor, validator, tests, reviewers
- deferred_projections: host-wide ownership graph
- notes: Operational claims do not grant permission or become canonical module truth.
- status: canonical

### Shared-Root Ownership Compatibility Fence Schema
- path: `knowledge/schemas/v1/lifecycle-root-ownership-fence.schema.json`
- title: Shared-Root Ownership Old-Client Compatibility Fence Schema
- surface_type: common SKV JSON Schema
- truth_role: canonical static receipt-layout fail-closed compatibility evidence truth
- owner: Symphony Knowledge Vector umbrella maintainers
- scope: Defines the exact document that ownership-aware clients consume and ownership-unaware lifecycle clients preserve as an unsupported critical package.
- relationships: governed_by -> `knowledge/LIFECYCLE.md`; implemented_by -> `tools/qxctl/internal/knowledgelifecycle/ownership_unix.go`
- consumers: qxctl lifecycle observation and ownership administration, direct package guards, validator, tests, reviewers
- deferred_projections: package-manager-native compatibility fences
- notes: The fence grants no permission, owns no package, and exists only to prevent an older mutation path from ignoring a newer root registry.
- status: canonical

### Shared-Root Ownership Result Schema
- path: `knowledge/schemas/v1/lifecycle-root-ownership-result.schema.json`
- title: Shared-Root Lifecycle Ownership Result Schema
- surface_type: common SKV JSON Schema
- truth_role: canonical bounded ownership administration response truth
- owner: Symphony Knowledge Vector umbrella maintainers
- scope: Defines status, reconciliation, adoption, and release results for one exact configured root.
- relationships: governed_by -> `knowledge/LIFECYCLE.md`; depends_on -> `knowledge/schemas/v1/lifecycle-root-ownership.schema.json`
- consumers: qxctl lifecycle administration, tests, reviewers
- deferred_projections: ownership dashboard
- notes: Results are noncanonical operational evidence.
- status: canonical

### Shared-Root Ownership Reconciliation Schema
- path: `knowledge/schemas/v1/lifecycle-root-ownership-reconciliation.schema.json`
- title: Shared-Root Lifecycle Reconciliation Schema
- surface_type: common SKV JSON Schema
- truth_role: canonical multi-root pre-action reconciliation evidence truth
- owner: Symphony Knowledge Vector umbrella maintainers
- scope: Defines deterministic ownership-result collection for one exact TOPS/profile action boundary.
- relationships: governed_by -> `knowledge/LIFECYCLE.md`; depends_on -> `knowledge/schemas/v1/lifecycle-root-ownership-result.schema.json`
- consumers: qxctl lifecycle apply, tests, reviewers
- deferred_projections: ownership dashboard
- notes: Reconciliation cannot bypass the coordinator's prepared action or SSIAG authorization.
- status: canonical

### qxctl Shared-Root Ownership Implementation
- path: `tools/qxctl/internal/knowledgelifecycle/ownership.go`
- title: qxctl Shared-Root Ownership State Machine
- surface_type: Go operational lifecycle implementation
- truth_role: exact multi-profile claim, adoption, release, and reclamation decision truth
- owner: qxctl maintainers
- scope: Derives stable control-domain claims, preserves legacy/unclaimed packages, authorizes reclamation only after all competing claims release, and prunes satisfied non-retention claims after observed durable absence.
- relationships: implements -> `knowledge/schemas/v1/lifecycle-root-ownership.schema.json`; governed_by -> `knowledge/LIFECYCLE.md`
- consumers: qxctl lifecycle apply and ownership commands, tests, reviewers
- deferred_projections: host-wide ownership graph
- notes: Caller class, version recency, and one profile's desired absence cannot establish reclamation authority.
- status: canonical

### qxctl Shared-Root Ownership Storage
- path: `tools/qxctl/internal/knowledgelifecycle/ownership_unix.go`
- title: qxctl Shared-Root Ownership Unix Storage
- surface_type: Go Darwin/Linux durable storage implementation
- truth_role: exact root-lock, no-follow, atomic replacement, and reclamation-gate truth
- owner: qxctl maintainers
- scope: Serializes registry and package mutation at the exact configured root, commits the receipt-layout compatibility fence before the first registry, and fails closed on untrusted paths or claim conflicts.
- relationships: implements -> `knowledge/schemas/v1/lifecycle-root-ownership.schema.json`; implements -> `knowledge/schemas/v1/lifecycle-root-ownership-fence.schema.json`; composes_with -> `tools/qxctl/internal/knowledgelifecycle/install_unix.go`
- consumers: qxctl lifecycle ownership and package executor, tests, reviewers
- deferred_projections: none
- notes: Native Windows is unsupported; Linux and the macOS development path use equivalent no-follow behavior.
- status: canonical

### qxctl Shared-Root Ownership Unsupported Platform Gate
- path: `tools/qxctl/internal/knowledgelifecycle/ownership_unsupported.go`
- title: qxctl Shared-Root Ownership Unsupported Platform Gate
- surface_type: Go fail-closed platform implementation
- truth_role: exact unsupported native-platform refusal truth
- owner: qxctl maintainers
- scope: Refuses shared-root ownership mutation outside Linux and macOS rather than substituting weaker filesystem semantics.
- relationships: governed_by -> `knowledge/LIFECYCLE.md`; complements -> `tools/qxctl/internal/knowledgelifecycle/ownership_unix.go`
- consumers: qxctl cross-platform builds, tests, reviewers
- deferred_projections: none
- notes: Windows users operate through WSL or a remote Linux node.
- status: canonical

### qxctl Shared-Root Ownership Regression Matrix
- path: `tools/qxctl/internal/knowledgelifecycle/ownership_test.go`
- title: qxctl Shared-Root Ownership Regression Matrix
- surface_type: Go deterministic lifecycle regression evidence
- truth_role: exact multi-profile, mixed-version, adoption, release, and symlink-failure test truth
- owner: qxctl maintainers
- scope: Exercises independent state roots, competing and unanimous retiring claims, interrupted-removal healing, legacy preservation, old-client fencing, unexpected old-writer inventory, and no-follow rejection.
- relationships: verifies -> `tools/qxctl/internal/knowledgelifecycle/ownership.go`; verifies -> `tools/qxctl/internal/knowledgelifecycle/ownership_unix.go`
- consumers: qxctl maintainers, reviewers, release validation
- deferred_projections: test-report projection
- notes: Passing tests do not grant permission or authorize unattended reclamation.
- status: canonical

### SSIAG Process Entrypoint
- path: `modules/secure-identity-access-governance/cmd/symphony-ssiag/main.go`
- title: SSIAG Supervised Process Entrypoint
- surface_type: Go process assembly implementation
- truth_role: exact configuration, policy-overlay, audit-producer, and protected-server assembly truth
- owner: SSIAG foundation maintainers
- scope: Loads the enrolled configuration, restores validated local policy state, constructs STAV production, and starts the protected local server.
- relationships: implements -> `modules/secure-identity-access-governance/SPEC.md`; depends_on -> `modules/secure-identity-access-governance/internal/policyadmin/manager.go`
- consumers: SSIAG installers, supervisors, operators, tests, reviewers
- deferred_projections: process topology projection
- notes: Assembly does not confer authority or permit canonical knowledge mutation.
- status: canonical

### SSIAG Configuration Validation
- path: `modules/secure-identity-access-governance/internal/config/config.go`
- title: SSIAG Configuration Validation
- surface_type: Go configuration implementation
- truth_role: exact enrolled configuration and authorization-policy validation truth
- owner: SSIAG foundation maintainers
- scope: Parses bounded configuration and validates the complete authorization snapshot used by initial and reset policy generations.
- relationships: implements -> `modules/secure-identity-access-governance/SPEC.md`; conforms_to -> `knowledge/ssiag/schemas/v1/authorization-policy.schema.json`
- consumers: SSIAG process entrypoint, policy administration, tests, reviewers
- deferred_projections: configuration explanation projection
- notes: Validation never promotes operational state to canonical knowledge.
- status: canonical

### SSIAG Policy Administration Regression Matrix
- path: `modules/secure-identity-access-governance/internal/policyadmin/manager_test.go`
- title: SSIAG Policy Administration Regression Matrix
- surface_type: Go deterministic regression evidence
- truth_role: exact CAS, receipt-binding, recovery, reset, and tamper-response test truth
- owner: SSIAG foundation maintainers
- scope: Exercises protected policy generations and interruption boundaries without granting runtime permission.
- relationships: verifies -> `modules/secure-identity-access-governance/internal/policyadmin/manager.go`; conforms_to -> `knowledge/ssiag/SPEC.md`
- consumers: SSIAG maintainers, reviewers, release validation
- deferred_projections: test-report projection
- notes: Passing tests are implementation evidence, not authorization or publication evidence.
- status: canonical

### SSIAG Protected Server
- path: `modules/secure-identity-access-governance/internal/server/server.go`
- title: SSIAG Protected Local Server
- surface_type: Go authenticated local IPC implementation
- truth_role: exact caller-neutral peer authentication and policy-administration endpoint truth
- owner: SSIAG foundation maintainers
- scope: Maps bounded local requests to status, proposal, apply, recovery, and authorization decisions after peer and permission validation.
- relationships: implements -> `modules/secure-identity-access-governance/SPEC.md`; depends_on -> `modules/secure-identity-access-governance/internal/policyadmin/manager.go`; conforms_to -> `knowledge/ssiag/schemas/v1/policy-result.schema.json`
- consumers: qxctl, SSIAG policy administration, tests, reviewers
- deferred_projections: remote transport under a future contract
- notes: No network listener, REST surface, caller-class authority, or unaudited mutation is authorized.
- status: canonical

### SSIAG Protected Server Regression Matrix
- path: `modules/secure-identity-access-governance/internal/server/server_test.go`
- title: SSIAG Protected Server Regression Matrix
- surface_type: Go local IPC regression evidence
- truth_role: exact host-owner, proposal binding, committed receipt, and live policy activation test truth
- owner: SSIAG foundation maintainers
- scope: Exercises the protected Unix-socket circuit and negative authority boundaries with isolated per-test state.
- relationships: verifies -> `modules/secure-identity-access-governance/internal/server/server.go`; verifies -> `modules/secure-identity-access-governance/internal/policyadmin/manager.go`
- consumers: SSIAG maintainers, qxctl maintainers, reviewers, release validation
- deferred_projections: test-report projection
- notes: Test fixtures do not become reusable credentials, policy truth, or audit evidence.
- status: canonical

### qxctl SSIAG Client Regression Matrix
- path: `tools/qxctl/internal/ssiagclient/client_test.go`
- title: qxctl SSIAG Client Regression Matrix
- surface_type: Go client regression evidence
- truth_role: exact bounded response, protocol, and error-handling test truth
- owner: qxctl and SSIAG maintainers
- scope: Verifies administrative response decoding and rejection behavior without manufacturing server authority.
- relationships: verifies -> `tools/qxctl/internal/ssiagclient/client.go`; conforms_to -> `knowledge/ssiag/SPEC.md`
- consumers: qxctl maintainers, SSIAG maintainers, reviewers, release validation
- deferred_projections: test-report projection
- notes: Client success cannot substitute for SSIAG peer authentication, permission evaluation, or STAV commitment.
- status: canonical

## Feature Administration Assurance Implementation Evidence

### Knowledge Engine Operation Specification API
- path: `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/operation.hpp`
- title: Knowledge Engine Stable Operation Specification API
- surface_type: C++26 public foundation header
- truth_role: exact stable engine-operation declaration and descriptor projection interface
- owner: SKV foundation maintainers
- scope: Defines operation identities and the reviewed administrative, protocol, authority, recovery, invocation, and thermal metadata projected by participating engines.
- relationships: implements -> `knowledge/FEATURE-ADMINISTRATION.md`; conforms_to -> `knowledge/schemas/v2/engine-descriptor.schema.json`; implemented_by -> `libraries/knowledge-vector-engine-cpp/src/operation.cpp`
- consumers: SSFV engine, participating knowledge engines, qxctl, validators, tests, reviewers
- deferred_projections: additional reviewed engine descriptor v2 implementations
- notes: Operation declarations record implementation evidence and confer no installation, authorization, or docking state.
- status: canonical

### Knowledge Engine Operation Specification Implementation
- path: `libraries/knowledge-vector-engine-cpp/src/operation.cpp`
- title: Knowledge Engine Stable Operation Specification Implementation
- surface_type: C++26 foundation implementation
- truth_role: deterministic operation validation, lookup, dispatch, and descriptor v2 projection truth
- owner: SKV foundation maintainers
- scope: Validates stable identifiers and exact metadata, rejects duplicate or unresolved recovery relationships, dispatches only declared operations, and projects digest-bound descriptor v2 evidence.
- relationships: implements -> `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/operation.hpp`; governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`
- consumers: participating C++ knowledge engines, foundation tests, SSFV administration assurance
- deferred_projections: additional engine operation tables
- notes: The implementation neither discovers executables nor changes installation or lifecycle state.
- status: canonical

### SSFV Engine Process Entrypoint
- path: `modules/ssfv-engine/src/main.cpp`
- title: SSFV Engine Process Entrypoint
- surface_type: C++26 engine process implementation
- truth_role: exact argument, descriptor-version, bounded-request, dispatch, and deterministic-error assembly truth
- owner: SSFV engine maintainers
- scope: Exposes descriptor v1, side-by-side descriptor v2, and bounded engine-process operation dispatch without a repository or qxctl presence assumption.
- relationships: implements -> `modules/ssfv-engine/SPEC.md`; invokes -> `modules/ssfv-engine/src/ssfv.cpp`; depends_on -> `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/operation.hpp`
- consumers: direct diagnostic callers, qxctl, packagers, process tests, reviewers
- deferred_projections: supervised installed-engine invocation
- notes: Direct invocation remains read-only or proposal-only according to the selected operation and confers no canonical apply authority.
- status: canonical

### SSFV Engine Public Interface
- path: `modules/ssfv-engine/src/ssfv.hpp`
- title: SSFV Engine Public Operation Interface
- surface_type: C++26 engine interface
- truth_role: exact SSFV operation declaration and dispatch interface
- owner: SSFV engine maintainers
- scope: Declares the SSFV operation table and bounded operation dispatcher used by the process entrypoint and tests.
- relationships: implements -> `modules/ssfv-engine/SPEC.md`; implemented_by -> `modules/ssfv-engine/src/ssfv.cpp`
- consumers: SSFV process entrypoint, tests, reviewers
- deferred_projections: none
- notes: The interface carries no installation, authorization, feature-worthiness, or canonical-mutation claim.
- status: canonical

### SSFV Engine Process Regression
- path: `modules/ssfv-engine/tests/process_smoke.sh`
- title: SSFV Engine Process Regression Matrix
- surface_type: POSIX shell process-conformance evidence
- truth_role: exact descriptor, bounded process, error, and read-only response regression truth
- owner: SSFV engine maintainers
- scope: Exercises executable help/version/descriptor surfaces, the descriptor-v2 finite value-count envelope above the unchanged shared default, and direct bounded engine requests including administration-assurance process behavior.
- relationships: verifies -> `modules/ssfv-engine/src/main.cpp`; verifies -> `modules/ssfv-engine/src/ssfv.cpp`; conforms_to -> `modules/ssfv-engine/SPEC.md`
- consumers: SSFV maintainers, packagers, reviewers, release validation
- deferred_projections: installed-package process matrix
- notes: Passing process tests do not establish feature completeness, installation, docking, or authorization.
- status: canonical

### qxctl Command Registry Contract
- path: `tools/qxctl/COMMANDS.md`
- title: qxctl Stable Command Registry Contract
- surface_type: qxctl administrative protocol documentation
- truth_role: exact expected/observed registry generation, identity lifecycle, and non-authorization behavior
- owner: qxctl maintainers under the cross-vector feature-administration contract
- scope: Documents the single-source CommandSpec/Cobra projection, client-independent and executable-bound evidence, parity verification, prohibited leaves, and retired fail-closed tombstones.
- relationships: governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`; describes -> `tools/qxctl/COMMANDS.json`; implemented_by -> `tools/qxctl/internal/commandregistry/registry.go`
- consumers: qxctl maintainers, module developers, SSFV engine, validators, reviewers, agentic tools
- deferred_projections: reviewed command-design assistance
- notes: The registry cannot inject commands, allocate names, decide feature-worthiness, or grant authority.
- status: canonical

### qxctl Command Manifest Operations
- path: `tools/qxctl/cmd/qxctl/command_manifest.go`
- title: qxctl Command Manifest Operations
- surface_type: Go administrative CLI implementation
- truth_role: exact expected, observed, and verification command behavior
- owner: qxctl maintainers
- scope: Emits the expected or executable-bound command registry and validates bounded no-follow packaged registry evidence against the complete current tree.
- relationships: implements -> `tools/qxctl/COMMANDS.md`; depends_on -> `tools/qxctl/internal/commandregistry/registry.go`; conforms_to -> `knowledge/schemas/v1/qxctl-command-registry.schema.json`
- consumers: headless callers, SSFV coverage collection, tests, reviewers
- deferred_projections: installation-receipt-bound client inventory
- notes: Successful verification proves parity with this qxctl source, not permission or live backend availability.
- status: canonical

### qxctl Command Manifest Regression
- path: `tools/qxctl/cmd/qxctl/command_manifest_test.go`
- title: qxctl Command Manifest Regression Matrix
- surface_type: Go deterministic regression evidence
- truth_role: exact command-count, stable-identity, feature-binding, parity, digest, path-safety, and JSON-debt test truth
- owner: qxctl maintainers
- scope: Verifies all registered executable leaves, expected-registry reproduction, exact hidden/prohibited behavior, bounded input rejection, and semantic vector bindings.
- relationships: verifies -> `tools/qxctl/cmd/qxctl/command_manifest.go`; verifies -> `tools/qxctl/internal/commandregistry/registry.go`; conforms_to -> `tools/qxctl/COMMANDS.md`
- consumers: qxctl maintainers, SSFV maintainers, validators, reviewers
- deferred_projections: cross-version client compatibility matrix
- notes: Passing tests neither ratify future command identities nor hide current non-JSON debt.
- status: canonical

### qxctl Command Specifications
- path: `tools/qxctl/cmd/qxctl/command_specs.go`
- title: qxctl Stable Command Specification Definitions
- surface_type: Go administrative CLI implementation
- truth_role: exact stable semantic specification attached to each constructed Cobra executable leaf
- owner: qxctl maintainers
- scope: Supplies identity, lifecycle, feature interaction, backend operation, protocols, mutability, authority, target, recovery, and machine-output declarations to the common command registry.
- relationships: implements -> `knowledge/FEATURE-ADMINISTRATION.md`; consumed_by -> `tools/qxctl/cmd/qxctl/commands.go`; projected_by -> `tools/qxctl/internal/commandregistry/registry.go`
- consumers: qxctl command construction, manifest generation, tests, SSFV administration assurance
- deferred_projections: reviewed specifications for future command leaves
- notes: A CommandSpec does not dynamically create grammar and cannot grant backend authority.
- status: canonical

### qxctl SSFV Regression Matrix
- path: `tools/qxctl/cmd/qxctl/ssfv_test.go`
- title: qxctl SSFV Administration Regression Matrix
- surface_type: Go deterministic regression evidence
- truth_role: exact SSFV result validation and administration-assurance CLI test truth
- owner: qxctl and SSFV maintainers
- scope: Verifies bounded administration input, complete result shape, self-digests, state enums, module admission, and rejection of malformed or authority-expanding engine responses.
- relationships: verifies -> `tools/qxctl/cmd/qxctl/ssfv.go`; conforms_to -> `knowledge/schemas/v1/administration-coverage-result.schema.json`
- consumers: qxctl maintainers, SSFV maintainers, reviewers, release validation
- deferred_projections: installed multi-version SSFV compatibility matrix
- notes: Successful result validation does not convert coverage evidence into installation, docking, authorization, or canonical truth.
- status: canonical

### qxctl Command Registry Implementation
- path: `tools/qxctl/internal/commandregistry/registry.go`
- title: qxctl Stable Command Registry Implementation
- surface_type: Go administrative foundation implementation
- truth_role: single-source executable-tree classification, deterministic projection, digest, and command-lifecycle truth
- owner: qxctl maintainers
- scope: Attaches CommandSpec records, enforces full-tree role parity, projects grammar from Cobra, binds executable identity, rejects duplicates and unsafe metadata, and preserves retired IDs as hidden null-grammar tombstones.
- relationships: implements -> `tools/qxctl/COMMANDS.md`; conforms_to -> `knowledge/schemas/v1/qxctl-command-registry.schema.json`; consumes -> `tools/qxctl/cmd/qxctl/command_specs.go`
- consumers: qxctl construction, manifest commands, SSFV administration assurance, validators, tests
- deferred_projections: additional reviewed command namespaces
- notes: Prohibited and retired leaves cannot satisfy required feature-administration coverage.
- status: canonical

### qxctl Command Registry Regression
- path: `tools/qxctl/internal/commandregistry/registry_test.go`
- title: qxctl Stable Command Registry Regression Matrix
- surface_type: Go deterministic regression evidence
- truth_role: exact structural parity, identity collision, schema, digest, trust, and retirement test truth
- owner: qxctl maintainers
- scope: Exercises missing registration, structural-handler replacement, duplicate identity, invalid metadata, deterministic aliases, observed trust, and fail-closed retirement tombstones.
- relationships: verifies -> `tools/qxctl/internal/commandregistry/registry.go`; conforms_to -> `knowledge/FEATURE-ADMINISTRATION.md`
- consumers: qxctl maintainers, reviewers, release validation
- deferred_projections: cross-version tombstone compatibility matrix
- notes: Fixtures are test evidence and cannot allocate production command identities.
- status: canonical

### qxctl Invariant Administration Operations
- path: `tools/qxctl/cmd/qxctl/knowledge_invariant.go`
- title: qxctl Common Invariant Administration
- surface_type: Go administrative CLI implementation
- truth_role: stable headless status, list, show, and complete-check operation truth
- owner: qxctl maintainers
- scope: Implements bounded read-only registry projections and exact receipt-validated Symphony Validator invocation while preserving complete result evidence and exit 26.
- relationships: implements -> `knowledge/INVARIANTS.md`; conforms_to -> `knowledge/schemas/v1/invariant-query-result.schema.json`; depends_on -> `tools/qxctl/internal/invariantregistry/registry.go`; invokes -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: headless administrators, module developers, reviewers, agentic tools
- deferred_projections: remote invariant-assurance transport
- notes: Direct queries state `semantic_validity: not_asserted`; only the exact complete validator operation supplies semantic assurance.
- status: canonical

### qxctl Invariant Administration Regression
- path: `tools/qxctl/cmd/qxctl/knowledge_invariant_test.go`
- title: qxctl Invariant Administration Regression Matrix
- surface_type: Go deterministic regression evidence
- truth_role: command identity, query shape, exact-validator, and exit-preservation test truth
- owner: qxctl maintainers
- scope: Verifies the four stable leaves, closed digest-bearing projections, hostile input rejection, exact installed validator invocation, and invariant exit-code preservation.
- relationships: verifies -> `tools/qxctl/cmd/qxctl/knowledge_invariant.go`; conforms_to -> `knowledge/INVARIANTS.md`
- consumers: qxctl and validator maintainers, reviewers, release validation
- deferred_projections: installed multi-version compatibility matrix
- notes: Fixtures provide bounded evidence and grant no authority to mutate the invariant registry.
- status: canonical

### qxctl Invariant Registry Consumer
- path: `tools/qxctl/internal/invariantregistry/registry.go`
- title: qxctl Invariant Registry Consumer
- surface_type: bounded Go registry-consumer implementation
- truth_role: consumer-side shape, identity, ordering, reference, and omit-self digest truth
- owner: qxctl maintainers
- scope: Loads one no-follow registry, rejects unsafe or ambiguous JSON, and emits deterministic status/list/show results without asserting semantic validity.
- relationships: consumes -> `knowledge/INVARIANT-OWNERSHIP.json`; conforms_to -> `knowledge/schemas/v1/invariant-query-result.schema.json`; governed_by -> `knowledge/INVARIANTS.md`
- consumers: qxctl invariant commands, tests, headless callers
- deferred_projections: generic remote registry client
- notes: This consumer does not duplicate validator semantics, execute regressions, allocate IDs, or write canonical truth.
- status: canonical

### qxctl Invariant Registry Consumer Regression
- path: `tools/qxctl/internal/invariantregistry/registry_test.go`
- title: qxctl Invariant Registry Consumer Regression Matrix
- surface_type: Go deterministic regression evidence
- truth_role: bounded parsing, no-follow, ordering, digest, and projection test truth
- owner: qxctl maintainers
- scope: Exercises duplicate keys, unknown fields, trailing values, control bytes, stale digests, invalid ordering and references, unsafe paths, and deterministic query results.
- relationships: verifies -> `tools/qxctl/internal/invariantregistry/registry.go`; conforms_to -> `knowledge/INVARIANTS.md`
- consumers: qxctl maintainers, validator maintainers, reviewers, release validation
- deferred_projections: cross-version query compatibility fixtures
- notes: Consumer tests do not replace complete validator or producer regression execution.
- status: canonical

### Validator Invariant Ownership Check
- path: `tools/symphony-validator/src/invariant_ownership.cpp`
- title: Symphony Validator Invariant Ownership Check
- surface_type: C++26 repository validation implementation
- truth_role: exact registry shape, identity, ownership, regression, adapter, and IPC evidence truth
- owner: Symphony Validator maintainers
- scope: Validates bounded no-follow registry access, exact constants and digest, ordering, owner/consumer evidence separation, named regressions, finite adapter closure, and receipt-backed real-process IPC evidence.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; governed_by -> `knowledge/INVARIANTS.md`; declared_by -> `tools/symphony-validator/src/invariant_ownership.hpp`
- consumers: repository validation, qxctl complete invariant check, reviewers
- deferred_projections: test-execution report projection and generic adapter protocol
- notes: Named-test discovery proves traceability only; applicable compiled and process suites remain separate execution evidence.
- status: canonical

### Validator Invariant Ownership Interface
- path: `tools/symphony-validator/src/invariant_ownership.hpp`
- title: Symphony Validator Invariant Ownership Interface
- surface_type: C++26 repository validation interface
- truth_role: exact invariant checker entrypoint declaration
- owner: Symphony Validator maintainers
- scope: Declares the read-only invariant-assurance result used by the complete validator and focused regressions.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; implemented_by -> `tools/symphony-validator/src/invariant_ownership.cpp`
- consumers: Symphony Validator CLI, unit tests, reviewers
- deferred_projections: none
- notes: The interface grants no registry repair or test-execution authority.
- status: canonical

### Validator Invariant Ownership Regression
- path: `tools/symphony-validator/tests/invariant_ownership_test.cpp`
- title: Symphony Validator Invariant Ownership Regression Matrix
- surface_type: C++26 deterministic regression evidence
- truth_role: malformed registry, ownership, adapter, path, test-reference, and IPC-boundary test truth
- owner: Symphony Validator maintainers
- scope: Exercises exact identifiers, omit-self digest, ordering, no-follow paths, role separation, finite adapters, named cases, real-process mechanics, and malformed input bounds.
- relationships: verifies -> `tools/symphony-validator/src/invariant_ownership.cpp`; conforms_to -> `knowledge/INVARIANTS.md`
- consumers: validator maintainers, module owners, reviewers, release validation
- deferred_projections: generic-adapter and complete installed-inventory matrices
- notes: The suite proves checker behavior against fixtures and does not claim the incremental registry is a complete legacy catalog.
- status: canonical

### Validator Feature Administration Check
- path: `tools/symphony-validator/src/feature_administration.cpp`
- title: Symphony Validator Feature Administration Check
- surface_type: C++26 repository validation implementation
- truth_role: exact static profile, registry, descriptor, digest, ordering, reference, and forward-gate conformance truth
- owner: Symphony Validator maintainers
- scope: Validates bounded no-follow feature-administration artifacts and enumerates source-implemented direct module scopes without executing qxctl or an engine.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`; declared_by -> `tools/symphony-validator/src/feature_administration.hpp`
- consumers: repository validation, qxctl validation surfaces, reviewers
- deferred_projections: forward-gate enforcement after profile adjudication
- notes: Static validation detects missing source-module feature/profile routing but does not establish installed-host inventory completeness, live backend availability, authorization, or docking.
- status: canonical

### Validator Feature Administration Interface
- path: `tools/symphony-validator/src/feature_administration.hpp`
- title: Symphony Validator Feature Administration Interface
- surface_type: C++26 repository validation interface
- truth_role: exact feature-administration checker entrypoint declaration
- owner: Symphony Validator maintainers
- scope: Declares the bounded static checker used by the validator CLI and regression suite.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; implemented_by -> `tools/symphony-validator/src/feature_administration.cpp`
- consumers: Symphony Validator CLI, unit tests, reviewers
- deferred_projections: none
- notes: The interface authorizes no repository mutation or dynamic command discovery.
- status: canonical

### Validator Feature Administration Regression
- path: `tools/symphony-validator/tests/feature_administration_test.cpp`
- title: Symphony Validator Feature Administration Regression Matrix
- surface_type: C++26 deterministic regression evidence
- truth_role: exact malformed, stale, unordered, unsafe-path, unreviewed, and valid-artifact test truth
- owner: Symphony Validator maintainers
- scope: Exercises the static checker across profile, expected command registry, engine descriptor, semantic record, digest, reference, forward-gate, and implemented-module admission boundaries.
- relationships: verifies -> `tools/symphony-validator/src/feature_administration.cpp`; conforms_to -> `knowledge/FEATURE-ADMINISTRATION.md`
- consumers: validator maintainers, SSFV and qxctl maintainers, reviewers, release validation
- deferred_projections: enforce-new and enforce-all regression matrices after ratification
- notes: Test success proves the declared static checks executed; it does not establish repository-wide or installed-host feature completeness.
- status: canonical

### Validator Root Summary Interface
- path: `tools/symphony-validator/src/root_summary.hpp`
- title: Symphony Validator Root Summary Interface
- surface_type: C++26 repository validation interface
- truth_role: deterministic root-summary projection and freshness entrypoint declaration
- owner: Symphony Validator maintainers
- scope: Declares JSON/Markdown projection and README managed-region assurance over validated canonical sources.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; implemented_by -> `tools/symphony-validator/src/root_summary.cpp`
- consumers: Symphony Validator CLI, qxctl, unit tests, documentation workflows
- deferred_projections: repository mutation and general documentation generation
- notes: This interface is read-only and does not make README canonical.
- status: canonical

### Validator Root Summary Implementation
- path: `tools/symphony-validator/src/root_summary.cpp`
- title: Symphony Validator Root Summary Implementation
- surface_type: C++26 deterministic projection implementation
- truth_role: validated SSFV, feature-administration, qxctl registry, and completed SODV summary evidence
- owner: Symphony Validator maintainers
- scope: Produces omit-self digest-bearing JSON and exact Markdown and rejects stale or malformed root README managed regions as the last derived repository gate.
- relationships: implements -> `tools/symphony-validator/SPEC.md`; consumes -> `knowledge/ssfv/REGISTRY.md`; consumes -> `knowledge/ssfv/COVERAGE.md`; consumes -> `knowledge/FEATURE-ADMINISTRATION-PROFILE.json`; consumes -> `knowledge/sodv/RELEASES.md`
- consumers: Symphony Validator CLI, qxctl, root-summary tests, reviewers
- deferred_projections: canonical apply, release currentness selection, and publication
- notes: All source contracts validate before projection; the implementation never rewrites README.
- status: canonical

### Validator Root Summary Regression
- path: `tools/symphony-validator/tests/root_summary_test.cpp`
- title: Symphony Validator Root Summary Regression Matrix
- surface_type: C++26 deterministic regression evidence
- truth_role: projection determinism, source validation, managed-region integrity, and freshness test truth
- owner: Symphony Validator maintainers
- scope: Exercises canonical projection, stale valid source changes, missing markers, same-line marker contamination, and deterministic JSON/Markdown output.
- relationships: verifies -> `tools/symphony-validator/src/root_summary.cpp`; conforms_to -> `tools/symphony-validator/SPEC.md`
- consumers: validator maintainers, qxctl maintainers, reviewers
- deferred_projections: repository-writing fixtures
- notes: Tests compare expected state and do not authorize the validator to repair README.
- status: canonical

### SAV Intent
- path: `knowledge/sav/INTENT.md`
- title: SAV Intent
- surface_type: canonical Markdown contract surface
- truth_role: SAV vocabulary, ownership, procedure, and boundary truth
- owner: SAV maintainers under Architect ratification
- scope: Defines the ratified SAV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: declares -> `knowledge/sav/MANIFEST.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SAV Manifest
- path: `knowledge/sav/MANIFEST.md`
- title: SAV Manifest
- surface_type: canonical Markdown contract surface
- truth_role: SAV vocabulary, ownership, procedure, and boundary truth
- owner: SAV maintainers under Architect ratification
- scope: Defines the ratified SAV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sav/INTENT.md`; declares -> `knowledge/sav/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SAV Skill
- path: `knowledge/sav/SKILL.md`
- title: SAV Skill
- surface_type: canonical Markdown contract surface
- truth_role: SAV vocabulary, ownership, procedure, and boundary truth
- owner: SAV maintainers under Architect ratification
- scope: Defines the ratified SAV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sav/SPEC.md`; depends_on -> `knowledge/SKILL.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SAV Spec
- path: `knowledge/sav/SPEC.md`
- title: SAV Spec
- surface_type: canonical Markdown contract surface
- truth_role: SAV vocabulary, ownership, procedure, and boundary truth
- owner: SAV maintainers under Architect ratification
- scope: Defines the ratified SAV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sav/INTENT.md`; depends_on -> `knowledge/sav/MANIFEST.md`; depends_on -> `knowledge/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SAV Relationships
- path: `knowledge/sav/RELATIONSHIPS.md`
- title: SAV Relationships
- surface_type: canonical Markdown contract surface
- truth_role: SAV vocabulary, ownership, procedure, and boundary truth
- owner: SAV maintainers under Architect ratification
- scope: Defines the ratified SAV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sav/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SAV Traits
- path: `knowledge/sav/TRAITS.md`
- title: SAV Traits
- surface_type: canonical Markdown contract surface
- truth_role: SAV vocabulary, ownership, procedure, and boundary truth
- owner: SAV maintainers under Architect ratification
- scope: Defines the ratified SAV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sav/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SAV Manifest
- path: `knowledge/sav/schemas/v1/MANIFEST.md`
- title: SAV Manifest
- surface_type: canonical Markdown contract surface
- truth_role: SAV vocabulary, ownership, procedure, and boundary truth
- owner: SAV maintainers under Architect ratification
- scope: Defines the ratified SAV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sav/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SAV Accord Reference
- path: `knowledge/sav/schemas/v1/accord-reference.schema.json`
- title: SAV Accord Reference
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SAV machine payload and result shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SAV Source Projection
- path: `knowledge/sav/schemas/v1/source-projection.schema.json`
- title: SAV Source Projection
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SAV machine payload and result shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SAV Current Resolution Input
- path: `knowledge/sav/schemas/v1/current-resolution-input.schema.json`
- title: SAV Current Resolution Input
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SAV machine payload and result shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SAV Current Snapshot
- path: `knowledge/sav/schemas/v1/current-snapshot.schema.json`
- title: SAV Current Snapshot
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SAV machine payload and result shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SAV Evaluation Input
- path: `knowledge/sav/schemas/v1/evaluation-input.schema.json`
- title: SAV Evaluation Input
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SAV machine payload and result shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SAV Evaluation Result
- path: `knowledge/sav/schemas/v1/evaluation-result.schema.json`
- title: SAV Evaluation Result
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SAV machine payload and result shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SAV Graph Projection
- path: `knowledge/sav/schemas/v1/graph-projection.schema.json`
- title: SAV Graph Projection
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SAV machine payload and result shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SAV Named Version
- path: `knowledge/sav/schemas/v1/named-version.schema.json`
- title: SAV Named Version
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SAV machine payload and result shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SEV Intent
- path: `knowledge/sev/INTENT.md`
- title: SEV Intent
- surface_type: canonical Markdown contract surface
- truth_role: SEV vocabulary, ownership, procedure, and boundary truth
- owner: SEV maintainers under Architect ratification
- scope: Defines the ratified SEV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: declares -> `knowledge/sev/MANIFEST.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SEV Manifest
- path: `knowledge/sev/MANIFEST.md`
- title: SEV Manifest
- surface_type: canonical Markdown contract surface
- truth_role: SEV vocabulary, ownership, procedure, and boundary truth
- owner: SEV maintainers under Architect ratification
- scope: Defines the ratified SEV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sev/INTENT.md`; declares -> `knowledge/sev/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SEV Skill
- path: `knowledge/sev/SKILL.md`
- title: SEV Skill
- surface_type: canonical Markdown contract surface
- truth_role: SEV vocabulary, ownership, procedure, and boundary truth
- owner: SEV maintainers under Architect ratification
- scope: Defines the ratified SEV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sev/SPEC.md`; depends_on -> `knowledge/SKILL.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SEV Spec
- path: `knowledge/sev/SPEC.md`
- title: SEV Spec
- surface_type: canonical Markdown contract surface
- truth_role: SEV vocabulary, ownership, procedure, and boundary truth
- owner: SEV maintainers under Architect ratification
- scope: Defines the ratified SEV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sev/INTENT.md`; depends_on -> `knowledge/sev/MANIFEST.md`; depends_on -> `knowledge/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SEV Dispositions
- path: `knowledge/sev/DISPOSITIONS.md`
- title: SEV Dispositions
- surface_type: canonical Markdown contract surface
- truth_role: SEV vocabulary, ownership, procedure, and boundary truth
- owner: SEV maintainers under Architect ratification
- scope: Defines the ratified SEV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sev/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SEV Qxctl Command Surface
- path: `knowledge/sev/profiles/qxctl-command-surface.md`
- title: SEV Qxctl Command Surface
- surface_type: canonical Markdown contract surface
- truth_role: SEV vocabulary, ownership, procedure, and boundary truth
- owner: SEV maintainers under Architect ratification
- scope: Defines the ratified SEV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sev/SPEC.md`; governed_by -> `knowledge/FEATURE-ADMINISTRATION.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SEV Manifest
- path: `knowledge/sev/schemas/v1/MANIFEST.md`
- title: SEV Manifest
- surface_type: canonical Markdown contract surface
- truth_role: SEV vocabulary, ownership, procedure, and boundary truth
- owner: SEV maintainers under Architect ratification
- scope: Defines the ratified SEV domain while preserving adjacent vector ownership and caller-neutral authority.
- relationships: depends_on -> `knowledge/sev/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Current implementation claims remain subordinate to exact source and regression evidence.
- status: canonical

### SEV Case Open Input
- path: `knowledge/sev/schemas/v1/case-open-input.schema.json`
- title: SEV Case Open Input
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SEV machine payload and result shape truth
- owner: SEV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SEV Evolution Case
- path: `knowledge/sev/schemas/v1/evolution-case.schema.json`
- title: SEV Evolution Case
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SEV machine payload and result shape truth
- owner: SEV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SEV Impact Result
- path: `knowledge/sev/schemas/v1/impact-result.schema.json`
- title: SEV Impact Result
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SEV machine payload and result shape truth
- owner: SEV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SEV Disposition Plan
- path: `knowledge/sev/schemas/v1/disposition-plan.schema.json`
- title: SEV Disposition Plan
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SEV machine payload and result shape truth
- owner: SEV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SEV Transition Verification Input
- path: `knowledge/sev/schemas/v1/transition-verification-input.schema.json`
- title: SEV Transition Verification Input
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SEV machine payload and result shape truth
- owner: SEV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SEV Transition Verification Result
- path: `knowledge/sev/schemas/v1/transition-verification-result.schema.json`
- title: SEV Transition Verification Result
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SEV machine payload and result shape truth
- owner: SEV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SEV Command Surface Assessment
- path: `knowledge/sev/schemas/v1/command-surface-assessment.schema.json`
- title: SEV Command Surface Assessment
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SEV machine payload and result shape truth
- owner: SEV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SEV Graph Projection
- path: `knowledge/sev/schemas/v1/graph-projection.schema.json`
- title: SEV Graph Projection
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: SEV machine payload and result shape truth
- owner: SEV maintainers under Architect ratification
- scope: Defines one bounded strict v1 object without granting source ownership, authority, apply, or persistence.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, Symphony Validator, knowledge coordinator, Maestro integration, reviewers, agents
- deferred_projections: canonical apply, persistent graph/database authority, public publication
- notes: Unknown critical versions and fields fail closed; a valid shape is evidence, not permission.
- status: canonical

### SAV Named Version Doctrine
- path: `knowledge/sav/NAMED-VERSIONS.md`
- title: SAV Named Versions
- surface_type: canonical vector doctrine
- truth_role: immutable composition-envelope lifecycle truth
- owner: SAV maintainers under Architect ratification
- scope: Defines validation, protected sealing, alias, predecessor, rollback, and publication boundaries.
- relationships: depends_on -> `knowledge/sav/SPEC.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, knowledge coordinator, SODV, reviewers, agents
- deferred_projections: alias indexes and publication views
- notes: Identity and equality remain digest-backed; aliases and projections are not authority.
- status: canonical

### SAV Extension Doctrine
- path: `knowledge/sav/EXTENSIONS.md`
- title: SAV Extension Capsules and Installation Blueprints
- surface_type: canonical vector doctrine
- truth_role: portable module-admission and two-way composition planning truth
- owner: SAV maintainers under Architect ratification
- scope: Defines third-party Capsule gaps, Blueprint forward/reverse edges, and authority separation.
- relationships: depends_on -> `knowledge/sav/SPEC.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, SEV engine, qxctl, lifecycle coordinator, third-party module authors, reviewers, agents
- deferred_projections: module ecosystem catalogs and installation-plan graphs
- notes: Incomplete third-party declarations remain visible and cannot fabricate missing qxctl coverage.
- status: canonical

### SEV Novelty Doctrine
- path: `knowledge/sev/NOVELTY.md`
- title: SEV Novelty Bundles
- surface_type: canonical vector doctrine
- truth_role: voluntary private-by-default novelty export truth
- owner: SEV maintainers under Architect ratification
- scope: Defines offline export, redaction, authority, prohibited payload, and STAV metadata boundaries.
- relationships: depends_on -> `knowledge/sev/SPEC.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, SSIAG, STAV, reviewers, agents
- deferred_projections: explicitly approved offline export files
- notes: Network transfer, publication, and model ingestion remain separate unimplemented operations.
- status: canonical

### SEV Watch Doctrine
- path: `knowledge/sev/WATCH.md`
- title: SEV Session Trigger and Watch Policy
- surface_type: canonical vector doctrine
- truth_role: opt-in freezing-path trigger and coalescing truth
- owner: SEV maintainers under Architect ratification
- scope: Defines default-disabled watch, configurable session boundary, debounce, recovery, and non-mutation.
- relationships: depends_on -> `knowledge/sev/SPEC.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, knowledge coordinator, SSIAG, reviewers, agents
- deferred_projections: coalesced event and case-candidate reports
- notes: The default is disabled and direct engine IPC remains available without a watcher.
- status: canonical

### SAV Extension Capsule Schema
- path: `knowledge/sav/schemas/v1/extension-capsule.schema.json`
- title: SAV Extension Capsule v1
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: portable third-party module admission shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines exact package, SSFV, qxctl, operation, receptor, relationship, trait, and extension-point evidence.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, qxctl, Symphony Validator, module authors, agents
- deferred_projections: extension ecosystem catalogs
- notes: Shape validity cannot grant namespace, command, installation, docking, or execution authority.
- status: canonical

### SAV Installation Blueprint Schema
- path: `knowledge/sav/schemas/v1/installation-blueprint.schema.json`
- title: SAV Installation Blueprint v1
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: noncanonical forward/reverse desired-composition shape truth
- owner: SAV maintainers under Architect ratification
- scope: Defines exact component requirements, Capsules, dependency edges, receptor defaults, and disabled apply.
- relationships: depends_on -> `knowledge/sav/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sav-engine/SPEC.md`
- consumers: SAV engine, SEV engine, qxctl, lifecycle coordinator, agents
- deferred_projections: disposable forward/reverse dependency graphs
- notes: Receptor entries are defaults that qxctl-administered lifecycle policy may explicitly override.
- status: canonical

### SEV Case Recalculation Input Schema
- path: `knowledge/sev/schemas/v1/case-recalculation-input.schema.json`
- title: SEV Case Recalculation Input v1
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: dynamic ready-set successor input truth
- owner: SEV maintainers under Architect ratification
- scope: Binds one case, exact plan, reobserved CURRENT, completed/failed action sets, and STSC time.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/src/sev.cpp`
- consumers: SEV engine, qxctl, lifecycle coordinator, agents
- deferred_projections: case timelines and ready-set graphs
- notes: Every successor remains content-addressed and binds an exact reobservation and predecessor.
- status: canonical

### SEV Novelty Bundle Schema
- path: `knowledge/sev/schemas/v1/novelty-bundle.schema.json`
- title: SEV Novelty Bundle v1
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: redactable offline novelty projection shape truth
- owner: SEV maintainers under Architect ratification
- scope: Binds case/CURRENT evidence, disclosure classes, redactions, approval, and disabled network transfer.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, SSIAG, STAV, agents
- deferred_projections: explicitly approved offline novelty files
- notes: Valid shape does not authorize export and prohibited security material remains excluded.
- status: canonical

### SEV Watch Policy Schema
- path: `knowledge/sev/schemas/v1/watch-policy.schema.json`
- title: SEV Watch Policy v1
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: opt-in session trigger policy shape truth
- owner: SEV maintainers under Architect ratification
- scope: Defines session boundary, bounded scopes/events, debounce, coalescing, lineage, and freezing-only operation.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/SPEC.md`
- consumers: SEV engine, qxctl, knowledge coordinator, SSIAG, agents
- deferred_projections: coalesced trigger reports
- notes: A policy cannot open a case, mutate a host, or enable export on its own.
- status: canonical

### SEV Evolution Session Binding Schema
- path: `knowledge/sev/schemas/v1/evolution-session-binding.schema.json`
- title: SEV Evolution Session Binding v1
- surface_type: Draft 2020-12 JSON Schema protocol contract
- truth_role: exact case/CURRENT binding into the shared durable lifecycle stream
- owner: SEV maintainers under Architect ratification
- scope: Binds case, source CURRENT, lifecycle profile, source report journal, desired state, direction, and STSC time without creating or authorizing apply.
- relationships: depends_on -> `knowledge/sev/schemas/v1/MANIFEST.md`; implemented_by -> `modules/sev-engine/src/sev.cpp`
- consumers: SEV engine, qxctl, knowledge coordinator, lifecycle administrator, agents
- deferred_projections: disposable evolution-session timelines
- notes: Prepare, finalize, status, recover, and close remain the existing coordinator lifecycle operations under fresh SSIAG decisions.
- status: canonical


## Deferred Projections
Unless a surface is explicitly authorized by its Contract Quad, generated indexes, graphs, DuckDB, JSONL, HDF5 outputs, qxctl integrations, validator implementations outside the bounded `tools/symphony-validator/` contract, and publication pipelines remain deferred and are not canonical authority. Projections authorized by `knowledge/SPEC.md` and a vector Contract Quad remain disposable and digest-bound. The indexed STAV JSON Schemas/fixtures, sixty-six common SKV v1 JSON Schemas, four common SKV v2 JSON Schemas, thirty-two SSIAG authorization, grant-planning, policy-administration, provider-control, provider-trust, provider-binding, and provider-readiness JSON Schemas, four SKVI JSON Schemas, five SCLV JSON Schemas, six SACV JSON Schemas, eight SODV operational JSON Schemas, and eighteen SSFV v1/v2 JSON Schemas are Architect-ratified protocol truth, not generated projections.

## Non-Authorized Artifacts
This index authorizes none of the following unless an indexed vector Contract Quad and `knowledge/SPEC.md` explicitly permit the bounded derived form:
- canonical generated index
- canonical generated graph or graph database
- projection treated as source truth
- qxctl canonical mutation before the apply gate
- validator implementation outside the bounded `tools/symphony-validator/` contract
- parser or projector behavior outside an owned engine/tool contract
- unregistered or generated schemas
- templates
- docs directory
- mint.json
- public documentation
- Mintlify configuration
- documentation publication configuration
- publication pipeline
- NotebookLM automation
- implementation, source, or build files outside an authorized module/tool/library contract
- CI files

Note on terminology: The term `c-o-r-e` is forbidden as an active project term.
