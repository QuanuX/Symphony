# Symphony Knowledge Vector Intent
## Symphony Knowledge Vector Intent

### Purpose
To establish declarative boundaries for the Symphony Knowledge Vector layer and formally map the relationships between truth surfaces, indexes, changes, and publication governance.

### Scope
Defines the overarching knowledge framework structure (`knowledge/`) and houses autonomous vector surfaces including SKVI, SCLV, SODV, SACV, SSFV, SSIAG, and STAV.

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

### Relationship to SSFV
`knowledge/ssfv/` owns canonical application-feature identity, semantics, hierarchy, lifecycle, distinctions, distributed record routing, coverage inventory, and graph-projection contracts. Its partial catalog contains sixty-nine experimental records across the repository root and fourteen implemented owner scopes, with fifty-four ratified nested subfeatures, reviewed F2 and F3 non-feature dispositions, and three proposal-only scopes explicitly excluded. Remaining nested feature, subfeature, and microfeature review is incomplete, so this incremental catalog is not a repository-completeness claim. SSFV references other vectors without duplicating their truth.

### Relationship to STSC
`knowledge/TIME.md` owns the common Symphony Temporal Semantics Contract for canonical UTC profiles, civil dates, duration units, monotonic elapsed time, timestamp authority, and the separation of wall-clock evidence from causal order. STSC is not a vector and creates no engine or runtime surface.

### Relationship to Validation Policy
`knowledge/VALIDATION.md` owns the common deterministic evidence, warning-profile, baseline, and delta contract. It is not a vector and creates no engine. The independently installed C++ validator produces immutable raw evidence; qxctl administers caller-neutral protected presentation and warning disposition without changing detection.

### Relationship to Module Contracts
Module contracts (`MANIFEST.md`, etc.) are distinct domains. SKV maps them but does not replace them.

### Relationship to symphony-validator
The checked-in, independently installable `tools/symphony-validator/` implementation produces deterministic, read-only line or structured JSON evidence. It checks required Knowledge Vector contract anchors, SKVI structure, SKVI coverage and paths, SCLV record shape and continuity, SACV registry structure, SODV local release-record relationships, and bounded repository doctrine. qxctl validates its exact receipt and structured digests before applying protected warning policy. Neither surface creates canonical truth, inspects external publication state, publishes documentation, or remediates files.

### Relationship to qxctl
qxctl is the Go administrative surface for the ratified vector-engine family. `qxctl knowledge ...` owns cross-vector lifecycle and persistent SSFV-session administration, `qxctl maestro inventory` owns the derived receptor-inventory presentation, `qxctl validate ...` owns protected warning policy, baseline, and display grammar, and `qxctl skvi|sclv|sacv|sodv|ssfv ...` owns vector-specific grammar. Implemented surfaces bind exact inactive-undocked installations, administer durable contexts, compose stable login/refresh/logout events, preserve SSFV semantic baselines, converge `apply-compatible` profiles, and classify full validator evidence without modifying it. qxctl implements these contracts but does not own vector semantics or canonical knowledge truth.

### Vector Engine Foundation
Each active application-level vector may have an independently installable, out-of-process C++ engine. Shared authority-free C++ mechanics may live under `libraries/`; a separate C++ coordinator owns authenticated-session and worktree-reconciliation mechanics. Engines inspect, validate, project, and propose within vector-owned contracts. They do not acquire authority merely by generating content.

The implemented `0.1.0-dev` foundation and coordinator slice provides strict local process framing, read-only inspect/snapshot checking, explicit compatibility negotiation, durable worktree reconciliation, SSIAG-authorized noncanonical authority epochs, persistent SSFV maintenance baselines, deterministic lifecycle planning, report-only boot journals, and separate apply-capable attempt/applied-state coordination over fully supplied desired/observed evidence. Independent C++ slices implement SKVI inspect/check/propose/project, SCLV inspect/check/propose/recover/project with provider-neutral evidence adapters, SACV inspect/check/diff/propose/project with bounded OpenAPI 3.2.0 JSON validation, SODV inspect/check/verify/propose/recover/project, SSFV inspect/check/diff/propose/graph with content-addressed semantic freshness, and Maestro receptor inspection, durable presence, and complete derived inventory. qxctl invokes exact receipt-validated installations, records vector bindings separately, supplies freshly observed SSFV/Maestro evidence, and performs only reviewed generic receipt-v2/runtime/Maestro-presence actions after the coordinator durably prepares them. SSFV's sixty-nine-record partial catalog establishes current top-level owner-scope routing and fifty-four exact nested subfeatures without asserting complete remaining nested coverage. Binding, reconciliation, authenticated-session coordination, maintenance review state, profile/runtime persistence, observation, or a lifecycle plan alone does not establish Maestro presence, ratify semantic truth, or mutate canonical knowledge.

The common lifecycle schema family defines declarative profile input, protected profiles/runtime state, desired and observed evidence, report-only plan commands, dependency-driven plans, applied state, report-journal v1, apply-journal v2, exact commands/results, immutable receipt-v2 truth, root-local multi-profile ownership/reclamation evidence, and Maestro receptor/presence contracts. The common temporal contract and schema define normalized UTC and civil-date representations without becoming an engine. qxctl implements profile, observation, authorization, report/boot, explicit apply, shared-root adoption/reconciliation/release, Maestro inspection/status/recovery, and lifecycle dock/undock administration. The coordinator dynamically recomputes component action order from explicit readiness in either supported migration direction and persists separate report/apply journals with exact compare-and-swap, timestamp-stable identities, linked revisions, active-attempt recovery, and content-addressed applied evidence. Host boot integration, canonical knowledge apply, receipt-v1 mutation, live process activation, and Maestro engine execution remain future gates.

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
This canonical surface recognizes SACV and SSFV governance and the common lifecycle contract family but authorizes no endpoint or unratified feature record by itself. It authorizes the bounded vector-engine architecture, the implemented read-only/proposal slices, protected lifecycle profile/observation/report/boot administration, the explicitly bounded local apply-compatible circuit, the ratified sixty-nine-record SSFV partial catalog, derived projections, and qxctl grammar defined by `knowledge/MANIFEST.md` and `knowledge/SPEC.md`. It does not authorize canonical apply, an unratified SSFV feature record, a repository-completeness claim, a network API, Mintlify configuration, NotebookLM automation, general publication pipeline, database authority, direct STAV mutation, hot/warm-path participation, or any capability outside a vector's own Contract Quad.
