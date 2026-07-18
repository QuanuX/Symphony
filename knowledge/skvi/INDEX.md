# Symphony Knowledge Vector Index

## Status
Status:
  Canonical declarative SKVI index.

## Purpose
A human-authored declarative knowledge routing table.

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
A human-authored declarative knowledge routing table.

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
Humans ratify.
Agents assist.

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
- consumers: humans, reviewers, agentic tools, future validators, qxctl maintainers
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
- consumers: humans, reviewers, qxctl, agentic tools, future validators
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
- consumers: implementers, reviewers, agentic tools, future validators
- deferred_projections: protocol schema, conformance evidence
- notes: Mutation endpoints remain disabled pending security gates.
- status: canonical

##### ARCHITECTURE.md
- path: `modules/secure-identity-access-governance/ARCHITECTURE.md`
- title: Symphony Secure Identity and Access Governance Architecture
- surface_type: module architecture
- truth_role: component, trust-boundary, provider, qxctl, and SKV design
- owner: secure-identity-access-governance maintainer
- scope: Preserves monorepo agent context and module-bounded install/runtime authority.
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
- consumers: implementers, reviewers, testers, agentic tools, future validators
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
- consumers: humans, reviewers, qxctl, agentic tools, future validators
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
- truth_role: safe agent interaction and implementation stop conditions
- owner: STAV append-authority maintainer
- scope: Permits inspection and verification while prohibiting schema invention and unauthorized ledger mutation.
- relationships:
  - depends_on -> `knowledge/stav/SKILL.md`
  - interprets -> `modules/stav-append-authority/THREAT-MODEL.md`
- consumers: humans, reviewers, agentic tools
- deferred_projections: none
- notes: Agent lifecycle actions still require explicit user authorization.
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
- consumers: implementers, reviewers, qxctl, agentic tools, future validators
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
- consumers: implementers, reviewers, testers, agentic tools, future validators
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
- scope: Defines build-time shared code as distinct from independently installed runtime modules.
- relationships: depends_on -> `INTENT.md`; governs -> `libraries/stav-protocol-go/MANIFEST.md`
- consumers: implementers, reviewers, agentic tools, future validators
- deferred_projections: dependency graph and release evidence
- notes: Libraries own no canonical protocol truth or operational identity.
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

### SACV Canonical Knowledge Vector

#### SACV INTENT.md
- path: `knowledge/sacv/INTENT.md`
- title: Symphony API Contract Vector Intent
- surface_type: knowledge-vector intent
- truth_role: canonical API-contract governance intent
- owner: SACV maintainer
- scope: Defines API-first source truth, OpenAPI 3.2.0 targeting, distributed semantic ownership, and security/publication boundaries.
- relationships: declares -> `knowledge/sacv/MANIFEST.md`; depends_on -> `knowledge/sodv/SPEC.md`
- consumers: humans, reviewers, API owners, agentic tools, future validators
- deferred_projections: OpenAPI validation evidence, documentation, SDK, and graph projections
- notes: Authorizes governance only, not an endpoint or remote listener.
- status: canonical

#### SACV MANIFEST.md
- path: `knowledge/sacv/MANIFEST.md`
- title: Symphony API Contract Vector Manifest
- surface_type: knowledge-vector manifest
- truth_role: canonical API ownership and placement contract
- owner: SACV maintainer
- scope: Declares SACV-owned policy and registry truth while retaining endpoint semantics with domain owners.
- relationships: depends_on -> `knowledge/sacv/INTENT.md`; declares -> `knowledge/sacv/SPEC.md`; declares -> `knowledge/sacv/REGISTRY.md`
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
- scope: Guides agents and humans without authorizing endpoints, publication, live requests, or MCP exposure.
- relationships: depends_on -> `knowledge/sacv/SPEC.md`; interprets -> `knowledge/sodv/SPEC.md`
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
- scope: Defines OpenAPI versioning, ownership, registry, compatibility, security, derivation, and publication boundaries.
- relationships: depends_on -> `knowledge/sacv/MANIFEST.md`; governs -> future owner-controlled OpenAPI descriptions; depends_on -> `knowledge/sodv/SPEC.md`
- consumers: API owners, implementers, reviewers, future validators and generators
- deferred_projections: generated bindings, SDKs, Mintlify documentation, MCP tools
- notes: Canonical descriptions target OpenAPI 3.2.0; none are registered yet.
- status: canonical

#### SACV REGISTRY.md
- path: `knowledge/sacv/REGISTRY.md`
- title: Symphony API Contract Registry
- surface_type: canonical API-contract registry
- truth_role: routing and ownership map for HTTP API entry documents
- owner: SACV maintainer
- scope: Registers owner, path, versions, audience, transport, security, publication, SDK, and lifecycle state.
- relationships: depends_on -> `knowledge/sacv/SPEC.md`; indexes -> future canonical owner API descriptions
- consumers: humans, SKVI, SODV, future validators and generators
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
- consumers: API owners, implementers, reviewers, future validators and generators
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
- truth_role: safe human and agent procedure for SSIAG changes
- owner: SSIAG knowledge maintainer
- scope: Defines reading order, agent authority, change procedure, and stop conditions.
- relationships: depends_on -> `knowledge/ssiag/SPEC.md`; interprets -> `knowledge/stav/SPEC.md`
- consumers: humans, maintainers, reviewers, agentic tools
- deferred_projections: none
- notes: Agents may query and propose but not bypass policy or handle credentials.
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
- scope: Declares schema authority, append authority, producers, qxctl, agents, and operational storage.
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
- notes: Agents never edit, repair, reorder, or append ledger files.
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
- consumers: protocol-kernel, append-authority, qxctl, producer implementers, reviewers, future validators
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
- consumers: protocol-kernel tests, toolchain migration, future validators and language implementations
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
- consumers: humans, reviewers, SSIAG implementers, future validators
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
- scope: Defines agent restrictions and ratification required before Apple Security access.
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
- consumers: implementers, reviewers, SSIAG foundation, future validators
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

### Validator Declarative Tool Contract Seed
##### INTENT.md
- path: `tools/symphony-validator/INTENT.md`
- title: Validator Intent
- surface_type: tool intent seed
- truth_role: intent and purpose for symphony-validator
- owner: validator maintainer
- scope: Define validator boundaries. The validator is deterministic, explainable, and non-agentic. It produces structured evidence for humans, CI systems, qxctl, and agentic tools. It does not perform interpretation, remediation, or architectural decision-making. Evidence model is truth. JSON is the structured evidence projection. Markdown is the agent/human ingestion projection.
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
- scope: Root definition of SKVI, SCLV, SODV, SACV, SSIAG, and STAV vector domains.
- consumers: humans, reviewers, agentic tools, future validators
- relationships: declares -> `knowledge/sacv/INTENT.md`; declares -> `knowledge/ssiag/INTENT.md`; declares -> `knowledge/stav/INTENT.md`
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
- scope: Human-authored declarative index of canonical Symphony knowledge-vector surfaces, their truth roles, ownership boundaries, relationships, consumers, deferred projections, and status.
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
  - human-readable Markdown report
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
    Human-authored declarative ledger for canonical SCLV records. Records canonical change truth against SKVI-indexed surfaces and preserves evidence, relationship changes, doctrine changes, compatibility consequences, publication consequences, projection consequences, and non-authorizations.
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
    - `human-readable Markdown report`
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
Unless a surface is explicitly indexed above, generated indexes, graphs, DuckDB, JSONL, HDF5 outputs, new qxctl integrations, validator implementations, and publication pipelines remain deferred and are not canonical authority. The indexed STAV JSON Schemas and fixtures are human-ratified protocol truth, not generated projections.

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
- unregistered or generated schemas
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
