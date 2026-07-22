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
- scope: Defines bounded JSON, digest, path, snapshot, and process mechanics without semantic authority.
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
- scope: Declares the C++26 static target, `0.1.0-dev` components, pinned JSON dependency, and versioned install paths.
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
- scope: Guides limits, strict parsing, path safety, response framing, and authority separation.
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
- scope: Defines `0.1.0-dev` mechanics and adversarial rejection requirements.
- relationships: depends_on -> `knowledge/SPEC.md`; governs -> `libraries/knowledge-vector-engine-cpp/CMakeLists.txt`
- consumers: coordinator and future vector engines, testers, reviewers
- deferred_projections: protocol conformance report
- notes: The version is developmental and not published.
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
- scope: Defines graph-like nodes/edges, immutable IDs, Go-only foundation, provider boundary, qxctl, and STAV projection.
- relationships: depends_on -> `knowledge/ssiag/MANIFEST.md`; governs -> `modules/secure-identity-access-governance/SPEC.md`; governs -> `modules/ssiag-provider-macos-keychain/SPEC.md`
- consumers: implementers, reviewers, qxctl, provider modules, agentic tools
- deferred_projections: graph view, conformance schema after ratification
- notes: Caller authentication, endpoint trust, native supervision/runtime ownership, and STAV producer integration are implemented; mutation and credential delivery remain gated.
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
- scope: Defines common values plus candidate, event, receipt, query, query-page, and verification structures.
- relationships: depends_on -> `knowledge/stav/SPEC.md`; governs -> `libraries/stav-protocol-go/`
- consumers: protocol-kernel, append-authority, qxctl, producer implementers, reviewers, symphony-validator and future validator extensions
- deferred_projections: generated documentation and conformance reports
- notes: Configuration, status, and local request/response schemas are intentionally absent.
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
- notes: Implemented SSIAG/STAV commands and ratified-but-unimplemented vector-engine grammar are explicitly distinguished.
- status: canonical

#### qxctl MANIFEST.md
- path: `tools/qxctl/MANIFEST.md`
- title: qxctl Manifest
- surface_type: tool manifest
- truth_role: command, dependency, installation, and non-authorization contract
- owner: qxctl maintainer
- scope: Enumerates operational commands, reserved vector-engine grammar, constrained dependencies, and lifecycle boundaries.
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
- scope: Implements current repository, module, SSIAG, STAV, SKVI, SCLV, SACV, and SODV administrative operation handlers.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; invokes -> `tools/qxctl/internal/knowledgeengine/client.go`
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
- scope: Implements current repository, SSIAG, STAV, and exact-installation SKVI/SCLV/SACV/SODV command grammar without owning domain semantics.
- relationships: implements -> `tools/qxctl/MANIFEST.md`; invokes -> `tools/qxctl/internal/knowledgeengine/client.go`
- consumers: qxctl executable, compatibility tests, maintainers, reviewers
- deferred_projections: generated CLI reference documentation
- notes: Cobra grammar does not authorize lifecycle activation or canonical apply.
- status: canonical

#### qxctl Knowledge Engine Process Client
- path: `tools/qxctl/internal/knowledgeengine/client.go`
- title: qxctl Knowledge Engine Process Client
- surface_type: bounded Go process-client implementation
- truth_role: trusted receipt resolution, child-process bounds, and response verification implementation truth
- owner: qxctl maintainers
- scope: Resolves exact installed SKVI, SCLV, SACV, and SODV versions, validates their inactive-undocked receipts and owned paths, invokes them with an empty environment and hard deadline, and verifies response identity and digest.
- relationships: implements -> `knowledge/SPEC.md`; implements -> `knowledge/skvi/SPEC.md`; implements -> `knowledge/sclv/SPEC.md`; implements -> `knowledge/sacv/SPEC.md`; implements -> `knowledge/sodv/SPEC.md`; called_by -> `tools/qxctl/cmd/qxctl/commands.go`
- consumers: qxctl SKVI/SCLV/SACV/SODV commands, tests, reviewers, future compatible vector clients
- deferred_projections: additional compatible vector clients
- notes: It does not install, activate, dock, infer membership, grant permission, ratify, mutate journals, or apply.
- status: canonical

### Knowledge Session Coordinator Module

#### Knowledge Session Coordinator INTENT.md
- path: `modules/knowledge-session-coordinator/INTENT.md`
- title: Knowledge Session Coordinator Intent
- surface_type: coordinator module intent
- truth_role: domain-neutral session/reconciliation purpose and implemented boundary
- owner: SKV coordinator maintainers
- scope: Declares the read-only `0.1.0-dev` slice and the deferred authenticated-session lifecycle.
- relationships: depends_on -> `knowledge/SPEC.md`; declares -> `modules/knowledge-session-coordinator/MANIFEST.md`
- consumers: qxctl and vector-engine implementers, reviewers, administrators, agentic tools
- deferred_projections: session and worktree reconciliation evidence
- notes: Successful inspect/check does not establish authentication or a session.
- status: canonical

#### Knowledge Session Coordinator MANIFEST.md
- path: `modules/knowledge-session-coordinator/MANIFEST.md`
- title: Knowledge Session Coordinator Manifest
- surface_type: independently installable coordinator manifest
- truth_role: executable, protocol, operation, dependency, and lifecycle truth
- owner: SKV coordinator maintainers
- scope: Declares implemented inspect/check, reserved session operations, disabled apply, and installed-undocked state.
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
- scope: Guides descriptor, inspect, check, deadline, stdout, and stop-condition handling.
- relationships: depends_on -> `modules/knowledge-session-coordinator/SPEC.md`; depends_on -> `knowledge/SKILL.md`
- consumers: administrators, implementers, reviewers, agentic tools
- deferred_projections: qxctl command procedure
- notes: Reserved and disabled descriptor states must be reported literally.
- status: canonical

#### Knowledge Session Coordinator SPEC.md
- path: `modules/knowledge-session-coordinator/SPEC.md`
- title: Knowledge Session Coordinator Specification
- surface_type: coordinator module specification
- truth_role: exact read-only operation, exit, descriptor, install, and non-authorization contract
- owner: SKV coordinator maintainers
- scope: Defines process inspect/check and explicitly excludes authenticated session mutation and apply.
- relationships: depends_on -> `knowledge/SPEC.md`; implements -> `knowledge/schemas/v1/engine-process-request.schema.json`; implements -> `knowledge/schemas/v1/engine-process-response.schema.json`
- consumers: C++ implementers, qxctl planners, testers, reviewers
- deferred_projections: authenticated-session and reconciliation conformance evidence
- notes: System/TOPS provisioning, qxctl, SSIAG/STAV, and Maestro remain unimplemented.
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
- scope: Root definition of SKVI, SCLV, SODV, SACV, SSIAG, and STAV vector domains.
- consumers: humans, reviewers, agentic tools, symphony-validator and future validator extensions
- relationships: declares -> `knowledge/MANIFEST.md`; declares -> `knowledge/sacv/INTENT.md`; declares -> `knowledge/ssiag/INTENT.md`; declares -> `knowledge/stav/INTENT.md`; checked_by -> `tools/symphony-validator/SPEC.md`
- deferred_projections: vector-authorized JSON/JSONL, search, graph, analytical, and documentation evidence
- notes: Owns the cross-vector engine foundation contract without owning vector-specific semantics.
- status: canonical

##### MANIFEST.md
- path: `knowledge/MANIFEST.md`
- title: Symphony Knowledge Vector Manifest
- surface_type: SKV umbrella manifest
- truth_role: common vector-engine identity, namespace, installability, and authority boundary
- owner: Symphony Knowledge Vector maintainers
- scope: Declares independently installed C++ engines, the implemented shared mechanics/read-only coordinator/SKVI/SCLV/SACV/SODV slices, qxctl administration, Linux-first delivery, Maestro readiness, and proposal-only initial state.
- relationships: depends_on -> `knowledge/INTENT.md`; declares -> `knowledge/SPEC.md`; governs -> `libraries/knowledge-vector-engine-cpp/`; governs -> `modules/knowledge-session-coordinator/`; governs -> `modules/skvi-engine/`; governs -> `modules/sclv-engine/`; governs -> future cleared vector-engine module paths
- consumers: vector maintainers, engine implementers, qxctl, Maestro planners, reviewers, agentic tools
- deferred_projections: engine inventory, install receipts, Maestro presence graph
- notes: Foundation/coordinator and SKVI/SCLV/SACV/SODV `0.1.0-dev` slices now exist; other vector engines, session mutation, canonical apply, live docking, and SSFV remain gated.
- status: canonical

##### SPEC.md
- path: `knowledge/SPEC.md`
- title: Symphony Knowledge Vector Engine Foundation Specification
- surface_type: SKV umbrella specification
- truth_role: normative cross-vector engine, session, proposal, projection, installation, and isolation contract
- owner: Symphony Knowledge Vector maintainers
- scope: Defines process identifiers, authenticated authority epochs, worktree reconciliation, proposal/apply separation, provider neutrality, qxctl grammar, install receipts, Maestro docking readiness, and hot/warm isolation.
- relationships: depends_on -> `knowledge/MANIFEST.md`; governs -> `knowledge/schemas/v1/MANIFEST.md`; governs -> `libraries/knowledge-vector-engine-cpp/SPEC.md`; governs -> `modules/knowledge-session-coordinator/SPEC.md`; depends_on -> `knowledge/ssiag/SPEC.md`; depends_on -> `knowledge/stav/SPEC.md`
- consumers: C++ engine and coordinator implementers, qxctl, SSIAG/STAV integrators, reviewers, agentic tools
- deferred_projections: proposal/session/provider/docking schemas, conformance evidence, engine inventory, docking graph
- notes: Six common schemas and the foundation/coordinator/SKVI/SCLV/SACV/SODV slices are implemented; programmatic apply is disabled.
- status: canonical

##### SKILL.md
- path: `knowledge/SKILL.md`
- title: Symphony Knowledge Vector Engine Skill
- surface_type: SKV umbrella skill guidance
- truth_role: safe engine implementation, review, session, and recovery procedure
- owner: Symphony Knowledge Vector maintainers
- scope: Guides proposal-only implementation and records stop conditions for apply, namespaces, external packages, networking, SSFV, and hot/warm isolation.
- relationships: depends_on -> `knowledge/SPEC.md`; governs -> future vector-engine implementation procedure
- consumers: implementers, maintainers, reviewers, qxctl contributors, agentic tools
- deferred_projections: conformance checklist and requirements traceability evidence
- notes: Does not authorize canonical mutation or self-ratification.
- status: canonical

##### Common v1 Schema Manifest
- path: `knowledge/schemas/v1/MANIFEST.md`
- title: Symphony Knowledge Vector Common Schemas v1
- surface_type: common protocol schema manifest
- truth_role: canonical inventory and boundary for exact common JSON schemas
- owner: Symphony Knowledge Vector maintainers
- scope: Declares process request/response, descriptor, install-receipt, immutable-proposal, and provider-evidence schemas.
- relationships: depends_on -> `knowledge/SPEC.md`; governs -> `libraries/knowledge-vector-engine-cpp/SPEC.md`; governs -> `modules/knowledge-session-coordinator/SPEC.md`; governs -> `modules/skvi-engine/SPEC.md`; governs -> `modules/sclv-engine/SPEC.md`
- consumers: C++ foundation and engine implementers, qxctl planners, validator, reviewers
- deferred_projections: generated schema documentation and conformance evidence
- notes: Operation-specific payload/result schemas remain with the applicable coordinator or vector.
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

## Deferred Projections
Unless a surface is explicitly authorized by its Contract Quad, generated indexes, graphs, DuckDB, JSONL, HDF5 outputs, qxctl integrations, validator implementations outside the bounded `tools/symphony-validator/` contract, and publication pipelines remain deferred and are not canonical authority. Projections authorized by `knowledge/SPEC.md` and a vector Contract Quad remain disposable and digest-bound. The indexed STAV JSON Schemas/fixtures, six common SKV JSON Schemas, four SKVI JSON Schemas, five SCLV JSON Schemas, six SACV JSON Schemas, and eight SODV operational JSON Schemas are Architect-ratified protocol truth, not generated projections.

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
