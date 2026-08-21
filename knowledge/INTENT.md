# Symphony Knowledge Vector Intent
## Symphony Knowledge Vector Intent

### Purpose
To establish declarative boundaries for the Symphony Knowledge Vector layer and formally map the relationships between truth surfaces, indexes, changes, and publication governance.

### Scope
Defines the overarching knowledge framework structure (`knowledge/`) and houses autonomous vector surfaces including SKVI, SCLV, SODV, SACV, SSFV, SAV, SEV, SSIAG, and STAV.

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
`knowledge/ssfv/` owns canonical application-feature identity, semantics, hierarchy, lifecycle, distinctions, distributed record routing, coverage inventory, and graph-projection contracts. Its partial catalog contains eighty-nine experimental records across the repository root and seventeen implemented owner scopes, with sixty-nine ratified nested features and three proposal-only scopes explicitly excluded. The newest records capture the Accordare STAV runtime producer and its exact grant administration while operational Keychain access remains disabled. Remaining nested feature, subfeature, and microfeature review is incomplete, so this incremental catalog is not a repository-completeness claim. SSFV references other vectors without duplicating their truth.

### Relationship to SAV
`knowledge/sav/` owns Accord Reference, immutable derived CURRENT snapshot, Named Version composition-envelope, relationship, trait, three-axis evaluation, comparison, explanation, and disposable graph contracts. SAV composes exact typed projections but never takes source truth from their owner vectors. Its independently installable C++ engine is freezing-path and read-only.

### Relationship to SEV and SCSEV
`knowledge/sev/` owns planned-change and encountered-novelty cases, impact assessment, disposition, dependency-ready-set, recalculation, verification, recovery-advice, and closure contracts. SCSEV is the first governed SEV profile and reuses SSFV feature administration, stable qxctl command truth, and engine-operation descriptors rather than creating a duplicate engine or registry. The first engine slice is report-only and proposal-only; durable external action remains separately authorized through qxctl and the knowledge coordinator.

### Relationship to STSC
`knowledge/TIME.md` owns the common Symphony Temporal Semantics Contract for canonical UTC profiles, civil dates, duration units, monotonic elapsed time, timestamp authority, and the separation of wall-clock evidence from causal order. STSC is not a vector and creates no engine or runtime surface.

### Relationship to Foundational Service Lifecycle
`knowledge/FOUNDATIONAL-LIFECYCLE.md` owns the narrow common machine envelope, target-host bootstrap authority, expected-state transaction, audit-deferred recovery, and version-safe activation rules shared by SSIAG and STAV enrollment and native supervision. Domain semantics remain with SSIAG and STAV. This exception does not authorize arbitrary lifecycle entry points or generic live-service activation.

### Relationship to Common Invariant Ownership
`knowledge/INVARIANTS.md` assigns cross-component invariants to the lowest layer that can authoritatively enforce them. `knowledge/INVARIANT-OWNERSHIP.json` currently links thirteen stable invariant IDs to owner contracts/components, producer implementations and regressions, consumer boundary rejections, and the finite set of allowed versioned adapters. It also makes forgotten administration work visible during module admission: a new installable module or administrator-facing interaction without a same-change feature mapping or evidence-backed disposition is uncovered, not exempt. The thirteen-record inventory remains incremental and proves neither complete legacy-invariant coverage nor installed-host package completeness. Neither the registry nor a consumer may invent domain semantics.

### Relationship to Validation Policy
`knowledge/VALIDATION.md` owns the common deterministic evidence, warning-profile, baseline, and delta contract. It is not a vector and creates no engine. The independently installed C++ validator produces immutable raw evidence; qxctl administers caller-neutral protected presentation and warning disposition without changing detection.

### Relationship to Module Contracts
Module contracts (`MANIFEST.md`, etc.) are distinct domains. SKV maps them but does not replace them.

### Relationship to symphony-validator
The checked-in, independently installable `tools/symphony-validator/` implementation produces deterministic, read-only line or structured JSON evidence. It checks required Knowledge Vector contract anchors, SKVI structure, SKVI coverage and paths, SCLV record shape and continuity, SACV registry structure, SODV local release-record relationships, and bounded repository doctrine. qxctl validates its exact receipt and structured digests before applying protected warning policy. Neither surface creates canonical truth, inspects external publication state, publishes documentation, or remediates files.

### Relationship to qxctl
qxctl is the Go administrative surface for the ratified vector-engine family. `qxctl knowledge ...` owns cross-vector lifecycle and persistent SSFV-session administration, `qxctl maestro inventory` owns the derived receptor-inventory presentation, `qxctl validate ...` owns protected warning policy, baseline, and display grammar, and `qxctl skvi|sclv|sacv|sodv|ssfv ...` owns implemented vector-specific grammar. Ratified Accordare grammar will use `qxctl accord ...` and `qxctl evolution ...` only after its command implementations and stable registry evidence exist. qxctl implements contracts but does not own vector semantics or canonical knowledge truth.

### Vector Engine Foundation
Each active application-level vector may have an independently installable, out-of-process C++ engine. Shared authority-free C++ mechanics may live under `libraries/`; a separate C++ coordinator owns authenticated-session and worktree-reconciliation mechanics. Engines inspect, validate, project, and propose within vector-owned contracts. They do not acquire authority merely by generating content.

The implemented `0.1.0-dev` foundation and coordinator slice provides strict local process framing, read-only inspect/snapshot checking, explicit compatibility negotiation, durable worktree reconciliation, SSIAG-authorized noncanonical authority epochs, persistent SSFV maintenance baselines, deterministic lifecycle planning, report-only boot journals, and separate apply-capable attempt/applied-state coordination over fully supplied desired/observed evidence. Independent C++ slices implement SKVI inspect/check/propose/project, SCLV inspect/check/propose/recover/project with provider-neutral evidence adapters, SACV inspect/check/diff/propose/project with bounded OpenAPI 3.2.0 JSON validation, SODV inspect/check/verify/propose/recover/project, SSFV inspect/check/diff/propose/graph with content-addressed semantic freshness, and Maestro receptor inspection, durable presence, and complete derived inventory. qxctl invokes exact receipt-validated installations, records vector bindings separately, supplies freshly observed SSFV/Maestro evidence, and performs only reviewed generic receipt-v2/runtime/Maestro-presence actions after the coordinator durably prepares them. SSFV's eighty-nine-record partial catalog establishes current top-level owner-scope routing and sixty-nine exact nested features without asserting complete remaining nested coverage. SSIAG separately catalogs exact compatible provider installations and converges a protected active/previous binding through compare-and-swap, safe STAV-before-commit evidence, and bounded forward or reverse recovery. Binding, reconciliation, authenticated-session coordination, maintenance review state, profile/runtime persistence, observation, or a lifecycle plan alone does not establish Maestro presence, ratify semantic truth, mutate canonical knowledge, or prove installed-host completeness.

The generic lifecycle schema family defines declarative profile input, protected profiles/runtime state, desired and observed evidence, report-only plan commands, dependency-driven plans, applied state, report-journal v1, apply-journal v2, exact commands/results, immutable receipt-v2 truth, root-local multi-profile ownership/reclamation evidence, and Maestro receptor/presence contracts. The foundational-service lifecycle family separately governs only SSIAG/STAV enrollment and native supervision through exact module-owned adapters. The common temporal contract and schema define normalized UTC and civil-date representations without becoming an engine. qxctl implements profile, observation, authorization, report/boot, explicit apply, shared-root adoption/reconciliation/release, Maestro inspection/status/recovery, lifecycle dock/undock administration, and the exact foundational route families. The coordinator dynamically recomputes generic component action order from explicit readiness in either supported migration direction, while each Go foundation owns its own protected transition attempts. Canonical knowledge apply, arbitrary live process activation, receipt-v1 generic mutation, and Maestro engine execution remain unavailable.

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
This canonical surface recognizes SACV and SSFV governance and the common lifecycle contract family but authorizes no endpoint or unratified feature record by itself. It authorizes the bounded vector-engine architecture, the implemented read-only/proposal slices, protected lifecycle profile/observation/report/boot administration, the explicitly bounded local apply-compatible circuit, the ratified eighty-nine-record SSFV partial catalog, derived projections, and qxctl grammar defined by `knowledge/MANIFEST.md` and `knowledge/SPEC.md`. It does not authorize canonical apply, an unratified SSFV feature record, a repository- or installed-host-completeness claim, complete legacy-invariant coverage, a network API, Mintlify configuration, NotebookLM automation, general publication pipeline, database authority, direct STAV mutation, hot/warm-path participation, or any capability outside a vector's own Contract Quad.
