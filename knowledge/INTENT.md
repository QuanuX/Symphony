# Symphony Knowledge Vector Intent
## Symphony Knowledge Vector Intent

### Purpose
To establish declarative boundaries for the Symphony Knowledge Vector layer and formally map the relationships between truth surfaces, indexes, changes, and publication governance.

### Scope
Defines the overarching knowledge framework structure (`knowledge/`) and houses the primary governance models for derived sub-vectors (SKVI, SCLV, SODV).

### Non-scope
It does not house implementation logic, build systems, deployment orchestration, or runtime modules.

### Role of the SKV
The SKV is the living knowledge framework of Symphony. It preserves architectural truth, module boundaries, contracts, doctrine, compatibility knowledge, operational knowledge, and publication knowledge in a structure that humans, validators, qxctl, CI, and agentic tools can consume consistently.
The SKV is the whole knowledge-vector framework, not merely a folder.
SKV is not a replacement for module contracts.

### Relationship to SKVI
SKVI indexes the knowledge surfaces declared by the SKV framework.

### Relationship to SCLV
SCLV records the changes made to the surfaces within the SKV framework over time.

### Relationship to SODV
SODV governs how knowledge within the SKV framework becomes official public documentation.

### Relationship to Module Contracts
Module contracts (`MANIFEST.md`, etc.) are distinct domains. SKV maps them but does not replace them.

### Relationship to symphony-validator
symphony-validator produces deterministic evidence.
The validator may later check knowledge-vector structure, but validator implementation is not part of this task.

### Relationship to qxctl
qxctl may later read or invoke knowledge-vector operations, but qxctl integration is not part of this task.

### Relationship to NotebookLM
NotebookLM aligns corpus context.
NotebookLM is a corpus alignment and context tool, not canonical authority.

### Relationship to Mintlify
Mintlify publishes derived official documentation.
Mintlify is a publication surface, not canonical authority.
No documentation publication pipeline is authorized by this task.

### Truth Hierarchy
MANIFEST.md is declared contract truth.
Code is implementation truth.
Generated JSON is a derived projection.
SSCG state is the compatibility interpretation.

### Publication Hierarchy
Canonical repository knowledge files are source truth.
SKVI indexes source truth.
SCLV records change truth.
SODV governs publication truth.
Published documentation is a derived public projection.

### Non-authorization Statement
This canonical seed authorizes no implementation files, no generated indexes, no generated reports, no schemas, no templates, no CI files, no documentation publication configuration, no Mintlify configuration, no qxctl integration, no validator implementation, no NotebookLM automation, no publication pipeline, no database files, no service files, no runtime processes, no deployment scripts, no installer scripts, no binary assets, and no binary renames.
