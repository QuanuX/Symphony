# Symphony Official Documentation Vector Intent
## Symphony Official Documentation Vector Intent

### Purpose
To declare knowledge expectations and publication rules for how internal canonical knowledge becomes public-facing documentation.

### Scope
Governing the mapping between internal structured knowledge and published artifacts, including official documentation and independently consumable module releases.

### Non-scope
Authority to publish. The independently installed SODV engine may inspect, validate, reconcile noncanonical transaction state, propose, and project; it is not the publication decision or publisher.

### Role of SODV
SODV governs official documentation publication, including how internal canonical knowledge becomes public-facing documentation.
SODV supports future Mintlify publication without making Mintlify the source of truth.
SODV is not the public documentation itself.
SODV does not authorize a documentation publication pipeline.
SODV does not authorize Mintlify configuration.
SODV authorizes module publication only through an immutable authorization record followed by an evidence-backed completion record.

### Relationship to SKV
SODV is an autonomous peer vector within the overarching SKV framework responsible for publication.

### Relationship to SKVI
SKVI indexes knowledge surfaces. SODV relies on SKVI to map what exists to be published.

### Relationship to SCLV
SCLV records change. SODV relies on SCLV to understand when canonical truth has shifted and requires documentation updates.

### Relationship to Mintlify
Mintlify publishes derived official documentation. SODV governs what Mintlify projects. 

### Relationship to SACV
SACV governs canonical API contracts and their registry. SODV alone decides whether a SACV-registered API may be projected into Mintlify, SDK documentation, a live playground, or MCP tooling.

### Relationship to NotebookLM
NotebookLM aligns corpus context but is not canonical authority. Append-only release records must be interpreted under the corpus interpretation rule in `knowledge/INTENT.md`.

### Relationship to Validator
The checked-in `tools/symphony-validator/` implementation produces deterministic, read-only evidence for required SODV contract anchors, indexed-path presence, and local release-record relationships. It does not contact Git hosts or package providers and cannot prove publication completion; `RELEASES.md`, caller-supplied immutable external evidence, and permission-backed review carry those roles.

### Relationship to qxctl
qxctl may invoke implemented SODV inspect, check, verify, propose, recover, and project operations under `qxctl sodv ...`. It does not publish, create tags, append completion records, or own SODV semantics. No documentation publication pipeline is authorized by this contract.

### Non-authorization Statement
This canonical surface authorizes the implemented proposal/read-only C++ SODV engine at `modules/sodv-engine/`, bounded derived release evidence, operational protocol schemas, and qxctl invocation. It authorizes no canonical apply, tag creation, external publication, public documentation, Mintlify configuration, NotebookLM automation, general publication pipeline, or release-completion claim.
