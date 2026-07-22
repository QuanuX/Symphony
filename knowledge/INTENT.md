# Symphony Knowledge Vector Intent
## Symphony Knowledge Vector Intent

### Purpose
To establish declarative boundaries for the Symphony Knowledge Vector layer and formally map the relationships between truth surfaces, indexes, changes, and publication governance.

### Scope
Defines the overarching knowledge framework structure (`knowledge/`) and houses autonomous vector surfaces including SKVI, SCLV, SODV, SACV, SSIAG, and STAV.

### Non-scope
It does not house implementation source, build systems, deployment orchestration, or runtime state. It does own the cross-vector contracts that bound independently installed vector engines and their qxctl administration.

### Role of the SKV
The SKV is the living knowledge framework of Symphony. It preserves architectural truth, module boundaries, contracts, doctrine, compatibility knowledge, operational knowledge, and publication knowledge in a structure that humans, validators, qxctl, CI, and agentic tools can consume consistently.
The SKV is the whole knowledge-vector framework, not merely a folder.
SKV is not a replacement for module contracts.

### Caller Authority Rule
Every SKV contract and its implementation must be caller-class neutral. Human, AI, agent, service, workload, organizational, and future actor classifications are valid descriptive facts and consumer labels, but none is an authorization input.

Supported authority derives from target-host ownership or granted permission, the requested operation and resource, proposal and expected state, and owner-configured safeguards. A target-host administrator remains sovereign over configurable governance and receives the same supported qxctl controls regardless of caller type. An engine, validator, adapter, or proposal does not manufacture permission merely by producing evidence.

Optional governance safeguards may be conservative by default and administrator-removable. Protocol-integrity requirements remain mandatory within supported tooling. When an applicable contract permits audit-deferred administrator recovery, the interruption must be explicit, durably journaled, and reconciled forward; it is never a silent ledger bypass. Symphony does not decide external legal or financial capacity by caller type.

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
The checked-in `tools/symphony-validator/` implementation produces deterministic, read-only evidence. It currently checks required Knowledge Vector contract anchors, SKVI structure, SKVI coverage and paths, SCLV record shape and continuity, SACV registry structure, SODV local release-record relationships, and bounded repository doctrine. It does not create canonical truth, inspect external publication state, publish documentation, or remediate files.

### Relationship to qxctl
qxctl is the Go administrative surface for the ratified vector-engine family. `qxctl knowledge ...` owns cross-vector engine, authenticated-session, worktree-reconciliation, proposal, and later apply coordination. `qxctl skvi|sclv|sacv|sodv|ssfv ...` owns vector-specific grammar. qxctl implements these contracts but does not own vector semantics or canonical knowledge truth.

### Vector Engine Foundation
Each active application-level vector may have an independently installable, out-of-process C++ engine. Shared authority-free C++ mechanics may live under `libraries/`; a separate C++ coordinator owns authenticated-session and worktree-reconciliation mechanics. Engines inspect, validate, project, and propose within vector-owned contracts. They do not acquire authority merely by generating content.

The implemented `0.1.0-dev` foundation and coordinator slice provides strict local process framing plus read-only inspect/snapshot checking. Independent C++ slices implement SKVI inspect/check/propose/project, SCLV inspect/check/propose/recover/project with provider-neutral evidence adapters, and SACV inspect/check/diff/propose/project with bounded OpenAPI 3.2.0 JSON validation. qxctl invokes each only from an exact inactive-undocked installation. None of these slices establishes an authenticated session or mutates canonical knowledge.

Initial vector-engine releases are read/query/validate/propose only. Programmatic canonical apply remains disabled until its SSIAG permission verification, expected-state transaction, qxctl safeguard, STAV event, recovery, and negative-test contracts are implemented and verified. `knowledge/SPEC.md` owns the common boundary; each vector Contract Quad owns its domain operations.

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
This canonical surface recognizes SACV governance but authorizes no endpoint document by itself. It authorizes the bounded vector-engine architecture, the implemented read-only/proposal development slices, derived projections, and qxctl grammar defined by `knowledge/MANIFEST.md` and `knowledge/SPEC.md`. It does not authorize canonical apply, an SSFV implementation, a network API, Mintlify configuration, NotebookLM automation, general publication pipeline, database authority, direct STAV mutation, hot/warm-path participation, or any capability outside a vector's own Contract Quad.
