# Symphony Knowledge Vector Index Intent
## Symphony Knowledge Vector Index Intent

### Purpose
To explicitly declare the indexing boundaries for Symphony's structural knowledge and module contracts.

### Scope
Mapping of the `knowledge/`, `modules/`, and `tools/` contract files into a holistic discovery layer.

### Non-scope
Implementation logic and runtime state are not canonical index content. The independently installed SKVI engine may inspect, validate, propose, and project this surface only within the common `knowledge/SPEC.md` boundary.

### Role of SKVI
SKVI maps the locations, scopes, descriptors, ownership boundaries, and relationships of all canonical knowledge-vector files.
SKVI makes the knowledge layer discoverable without requiring agents or humans to infer structure.

### Autonomy Statement
SKVI is autonomous and discoverable as a peer vector surface.
SKVI is not hidden inside SKV.
SKVI's repository-maintained Markdown remains canonical. Generated indexes, search stores, and graphs are disposable projections.

### Relationship to SKV
SKVI indexes the knowledge surfaces established by the SKV.

### Relationship to SCLV
SCLV is one of the knowledge surfaces indexed by SKVI. SKVI tracks SCLV's location and role.

### Relationship to SODV
SODV uses SKVI to discover the canonical source truth necessary for deriving public documentation.

### Relationship to Validator
The checked-in `tools/symphony-validator/` implementation consumes SKVI as declared routing truth and produces deterministic, read-only evidence for entry shape, required-surface coverage, indexed-path safety and existence, and SCLV cross-references. SKVI does not grant the validator authority to create or rewrite canonical entries.

### Relationship to qxctl
qxctl administers installed SKVI engine operations through `qxctl skvi ...` and cross-vector lifecycle through `qxctl knowledge ...`. Initial operations are inspect, check, propose, and project only. qxctl does not own SKVI membership or mutate canonical entries.

### Relationship to NotebookLM and Mintlify
NotebookLM aligns corpus context.
Mintlify publishes derived official documentation.
SKVI provides the structural roadmap that NotebookLM and Mintlify consume as derived projections. No documentation publication pipeline is authorized by this task.

### Non-authorization Statement
This canonical surface governs the implemented proposal-only C++ SKVI engine at `modules/skvi-engine/`, shared foundation use, deterministic derived projections, and bounded exact-installation qxctl integration. It authorizes no programmatic canonical apply, autonomous membership decision, competing source-truth database, NotebookLM automation, Mintlify/publication pipeline, or SSFV behavior.
