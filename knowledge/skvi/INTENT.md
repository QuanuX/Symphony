# Symphony Knowledge Vector Index Intent
## Symphony Knowledge Vector Index Intent

### Purpose
To explicitly declare the indexing boundaries for Symphony's structural knowledge and module contracts.

### Scope
Mapping of the `knowledge/`, `modules/`, and `tools/` contract files into a holistic discovery layer.

### Non-scope
Implementation logic (e.g., C++ files, runtime scripts) is outside the indexing scope. 

### Role of SKVI
SKVI maps the locations, scopes, descriptors, ownership boundaries, and relationships of all canonical knowledge-vector files.
SKVI makes the knowledge layer discoverable without requiring agents or humans to infer structure.

### Autonomy Statement
SKVI is autonomous and discoverable as a peer vector surface.
SKVI is not hidden inside SKV.
SKVI is not a generated database yet.
SKVI is not a generated index yet.

### Relationship to SKV
SKVI indexes the knowledge surfaces established by the SKV.

### Relationship to SCLV
SCLV is one of the knowledge surfaces indexed by SKVI. SKVI tracks SCLV's location and role.

### Relationship to SODV
SODV uses SKVI to discover the canonical source truth necessary for deriving public documentation.

### Relationship to Validator
The checked-in `tools/symphony-validator/` implementation consumes SKVI as declared routing truth and produces deterministic, read-only evidence for entry shape, required-surface coverage, indexed-path safety and existence, and SCLV cross-references. SKVI does not grant the validator authority to create or rewrite canonical entries.

### Relationship to qxctl
qxctl may later read or invoke knowledge-vector operations, but qxctl integration is not part of this task.

### Relationship to NotebookLM and Mintlify
NotebookLM aligns corpus context.
Mintlify publishes derived official documentation.
SKVI provides the structural roadmap that NotebookLM and Mintlify consume as derived projections. No documentation publication pipeline is authorized by this task.

### Non-authorization Statement
This canonical surface authorizes no new implementation files, generated indexes, generated reports, schemas, templates, CI files, documentation publication configuration, Mintlify configuration, qxctl integration, validator capability beyond the separately bounded `tools/symphony-validator/` contract, NotebookLM automation, publication pipeline, database files, service files, runtime processes, deployment scripts, installer scripts, binary assets, or binary renames.
