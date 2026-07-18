# Symphony Knowledge Vector Intent
## Symphony Knowledge Vector Intent

### Purpose
To establish declarative boundaries for the Symphony Knowledge Vector layer and formally map the relationships between truth surfaces, indexes, changes, and publication governance.

### Scope
Defines the overarching knowledge framework structure (`knowledge/`) and houses autonomous vector surfaces including SKVI, SCLV, SODV, SACV, SSIAG, and STAV.

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

### Relationship to SACV
`knowledge/sacv/` owns cross-cutting API-contract governance, the OpenAPI 3.2.0 profile, and the API-contract registry. Endpoint semantics remain with their domain-owning vector or module. SODV governs any public projection.

### Relationship to SSIAG
`knowledge/ssiag/` owns canonical secure identity and access governance vocabulary, relationships, extensions, provider protocol, and authority boundaries. Runtime code implements that truth but does not replace it.

### Relationship to STAV
`knowledge/stav/` owns canonical TOPS audit protocol truth. Per-TOPS operational ledgers live outside the repository and are not SKV content.

### Relationship to Module Contracts
Module contracts (`MANIFEST.md`, etc.) are distinct domains. SKV maps them but does not replace them.

### Relationship to symphony-validator
The checked-in `tools/symphony-validator/` implementation produces deterministic, read-only evidence. It currently checks required Knowledge Vector contract anchors, SKVI structure, SKVI coverage and paths, SCLV record shape and continuity, and bounded repository doctrine. It does not create canonical truth, validate SODV release-transaction semantics, publish documentation, or remediate files.

### Relationship to qxctl
qxctl may later read or invoke knowledge-vector operations, but qxctl integration is not part of this task.

### Relationship to NotebookLM
NotebookLM aligns corpus context.
NotebookLM is a corpus alignment and context tool, not canonical authority.

### Corpus Interpretation Rule
Current contract surfaces state present posture. Append-only SCLV and SODV records state what was known, authorized, or completed at the recorded point in time and must remain unchanged as historical evidence.

Corpus consumers, including NotebookLM and agentic tools, must interpret an append-only record together with later correction, recovery, supersession, and completion records. The latest applicable canonical record and current contract surface govern present-state answers; an older record remains valid history but must not be presented as current posture after a later canonical record changes its active interpretation.

When current contracts, implementation evidence, and the latest applicable record disagree, the disagreement is drift to be surfaced for review. Corpus tooling must not silently invent a reconciliation.

### Relationship to Mintlify
Mintlify publishes derived official documentation.
Mintlify is a publication surface, not canonical authority.
No documentation publication pipeline is authorized by this contract.

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
This canonical surface recognizes SACV governance but authorizes no endpoint document by itself. It authorizes no new implementation files, generated indexes, generated reports, templates, CI files, documentation publication configuration, Mintlify configuration, qxctl integration, validator capability beyond the separately bounded `tools/symphony-validator/` contract, NotebookLM automation, publication pipeline, database files, service files, runtime processes, deployment scripts, installer scripts, binary assets, or binary renames.
