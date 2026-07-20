# Symphony Validator Specification

****

## Specification Status
- normative contract for the checked-in deterministic C++26 parser/checker
- implementation authorized within the bounded source/build surfaces
- not a JSON schema
- not a Markdown template
- not CI configuration
- not build logic
- not projector, qxctl-integration, or publication authorization

## Purpose
Define the exact behavior and authority boundaries of the implemented C++ validator.

## Deterministic, Non-Autonomous Validator Doctrine
The validator is deterministic, explainable, and non-autonomous. It produces structured evidence for every caller, CI systems, qxctl, and agentic tools, but it does not perform interpretation, remediation, or architectural decision-making.
The validator does not infer intent.
The validator does not rewrite files.
The validator does not choose remedies.
The validator does not make architecture decisions.
The validator does not replace permission-backed ratification or semantic review.

## Evidence Model
Evidence model is truth. JSON is the structured evidence projection. Markdown is the caller-ingestion projection.

## Dual Output Model
Evidence model is truth.
JSON is the structured evidence projection.
Markdown is the caller-ingestion projection.
Markdown must not introduce claims, conclusions, or remediation steps that are not present in the source evidence model.

## JSON Evidence Projection
A future structured machine-readable projection may support the administrative spine and CI. It is not implemented or authorized by this contract increment.

## Markdown Caller-Ingestion Projection
A future Markdown projection may provide a stable, context-friendly ingestion surface. It is not implemented or authorized by this contract increment.

## Synchronization Rules
JSON and Markdown must share the same run ID.
Markdown must mirror rule IDs from JSON.
Markdown must preserve severity, path, status, and reason.
Markdown must not invent interpretation.
Markdown must not suggest architecture unless the rule itself encodes the expected condition.
Any caller may consume Markdown, but validator authority remains deterministic evidence only.

## Output Modes
- Default local mode
- CI mode
- Agent mode
- Strict machine mode
- Permission-backed review mode

## Rule ID Taxonomy
Families including `REPO.*`, `MODULE.*`, `INSTALL.*`, `NAMESPACE.*`, `TROLL.*`, `BUS.*`, `PYTHON.*`, `DOMAIN.*`, `INFRA.*`, `TERMS.*`, `LEAKAGE.*`, `PR.*`.

## Severity Model
`pass`, `info`, `warning`, `error`, `blocker`

## Status Model
Status reflects the deterministic outcome of a rule check.

## Exit-code Model
`0` to `21`, mapping to passes, errors, blockers, malformed repositories, or internal failures. The exact mapping is:

- `0`: success;
- `1`: invalid CLI usage or unknown command;
- `2`: invalid repository path;
- `3`: SKVI parsing/shape;
- `4`: SCLV parsing/shape prerequisite;
- `5`: SKVI/SCLV cross-reference;
- `6`: status/change-type vocabulary;
- `7`: SCLV record shape;
- `8`: unauthorized artifacts;
- `9`: required canonical surfaces;
- `10`: validator contract shape;
- `11`: runtime contract shape;
- `12`: knowledge contract shape;
- `13`: root contract shape;
- `14`: SCLV ledger continuity;
- `15`: doctrine vocabulary;
- `16`: SKVI coverage;
- `17`: SKVI path safety/existence;
- `18`: SCLV referenced surfaces;
- `19`: SCLV/SKVI membership;
- `20`: validator build-source integrity;
- `21`: caller-authority regression.

## Historical/Migration Exception Behavior
Stale names (e.g. `legacy node execution label`, `legacy native hot-path label`, `legacy bus residency label`) are rejected except in historical contexts or rename records.

## Allowlist Behavior
Allowlists must never become silent bypasses. Every allowlist entry must produce evidence in JSON and Markdown.

The owner-ratified STAV v1 JSON Schema and conformance-fixture files are canonical protocol truth, not generated projections. The artifact checker may allow only their exact paths and must emit `artifact.canonical_json_authorized` evidence for every encountered file. Prefix or extension-wide JSON allowlisting is prohibited; any new canonical JSON artifact requires an explicit contract and validator update.

## Refusal/Non-Remediation Behavior
The validator may report failures and identify expected/observed conditions. It must not rewrite files or choose remedies.

## Relationship to qxctl
The validator is currently invoked directly. `qxctl` mediation remains deferred.

## Relationship to CI / PR gates
The implementation provides deterministic line-oriented evidence, a summary, and exit status. CI/PR-gate wiring and structured artifacts remain deferred.

## Relationship to SKV / SKVI / SCLV / SODV
The validator does not replace SKV / SKVI / SCLV / SODV records. It provides evidence to support them.

## Behavioral Non-goals
The validator must not choose infrastructure for users, assume Docker/Kubernetes/cloud providers, impose market-data/order-flow/trading doctrine, require Python for hot-path or administrative spine, ban optional isolated Python habitats, treat contract seeds as runtime implementation, convert monorepo modularity into microservices doctrine, absorb module sovereignty into root-level logic, become a runtime daemon, become a hidden coordinator, replace qxctl, replace Maestro, replace SKV / SKVI / SCLV / SODV records, perform autonomous semantic decisions, infer intent, auto-remediate files, or make architecture decisions.
Active project term c-o-r-e is absent except inside explicit forbidden-term scan descriptions.

## Implemented Authorization Boundary
This specification authorizes the checked-in C++26 command-line parser/checker, its CMake build contract, and smoke fixtures. The implementation may read repository surfaces, emit deterministic evidence, and return deterministic exit status. It remains non-autonomous and read-only.


## Troll Doctrine
trolls are the local residents.
A troll is a bounded local resident of a Symphony runtime domain.
A troll is a runtime-residency role, not a caller identity or authorization class.

node-troll represents the node.
bus-troll manages bus residency and bus compatibility.
hotpath-runtime owns the native hot path.
hotpath-runtime is not a troll; it is the native hot-path runtime substrate.


## Bus-Troll Optionality
bus-troll is first-class and individually installable.
bus-troll is required only for deployments that use a managed bus boundary.
Bus bypass remains valid when declared by deployment constraints.
The existence of bus-troll does not make bus traversal mandatory.

## Parser / Checker / Projector Contract

### Parser Boundary

The implemented validator parser may read canonical Markdown surfaces including:
- `knowledge/skvi/INDEX.md`
- `knowledge/skvi/SPEC.md`
- `knowledge/sclv/CHANGELOG.md`
- `knowledge/sclv/SPEC.md`
- `knowledge/sodv/SPEC.md`
- `tools/symphony-validator/SPEC.md`

The parser may extract SKVI entry fields:
- `path`
- `title`
- `surface_type`
- `truth_role`
- `owner`
- `scope`
- `relationships`
- `consumers`
- `deferred_projections`
- `status`
- `notes`

The parser may extract SCLV record fields:
- `record_id`
- `title`
- `status`
- `date`
- `change_type`
- `related_pr`
- `merge_commit`
- `affected_surfaces`
- `skvi_references`
- `change_summary`
- `relationship_changes`
- `doctrine_changes`
- `compatibility_consequences`
- `publication_consequences`
- `projection_consequences`
- `evidence`
- `non_authorizations`
- `notes`

The parser must not:
- infer missing intent
- rewrite Markdown
- normalize doctrine by invention
- create records
- choose remedies
- decide architecture
- generate canonical truth
- publish documentation
- integrate with qxctl
- emit projections unless explicitly authorized by a future task

### Checker Boundary

The implemented validator checks may produce deterministic evidence for:
- canonical path existence
- SKVI entry shape
- SCLV record shape
- SCLV record ID uniqueness
- SCLV record status vocabulary
- SCLV change_type vocabulary
- SCLV related_pr shape
- SCLV merge_commit shape
- SCLV affected_surfaces path existence
- SCLV skvi_references alignment
- deferred projection declarations
- non-authorized artifact declarations
- stale namespace terms
- forbidden active-term scans
- generated artifact absence
- candidate-only wording
- task-local wording
- doctrine-preservation evidence

Checks produce evidence, not decisions.

Checks must not:
- rewrite files
- remove files
- create files
- open PRs
- merge PRs
- decide architecture
- decide compatibility state
- replace SSCG interpretation
- replace permission-backed ratification or semantic review
- replace NotebookLM alignment
- replace SODV publication governance

### Projector Boundary

Future validator projection behavior is deferred.

Future projection targets may include:
- JSON / JSONL portable evidence
- DuckDB analytical projection
- HDF5 dense quantitative / vector / compatibility substrate
- graph view relationship projection
- qxctl-readable evidence projection
- readable Markdown report

No projection is canonical authority.
All projections are derived, disposable, and rebuildable.
This spec section authorizes no generated projection.
Future projection formats require separate planning and canonical authorization.

### Evidence Categories

- pass: Indicates successful alignment with canonical truth. Applies when structural, vocabulary, and path checks meet expectations.
- warning: Indicates a non-blocking anomaly or potential drift. Applies when permission-backed review is needed but deterministic evidence does not require blocking.
- violation: Indicates a strict defect or architectural break. Applies when path existence, required shape, or vocabulary rules definitively fail.
- deferred: Indicates a condition that is known but intentionally not checked yet. Applies when a rule or boundary is planned but unimplemented.
- absent: Indicates missing expected canonical files or fields. Applies when a required artifact or structured entry cannot be found.
- stale: Indicates references to outdated nomenclature or deprecated terms. Applies during terminology scans.
- unknown: Indicates unparseable or unrecognized structure. Applies when the parser cannot deterministically extract fields.
- blocked: Indicates a check skipped due to prerequisite failure. Applies when an earlier dependency failure prevents execution.

### Authority Boundaries

The validator is deterministic, explainable, and non-autonomous.
Evidence model is truth.
JSON is the structured evidence projection.
Markdown is the caller-ingestion projection.
MANIFEST.md is declared contract truth.
Code is implementation truth.
Generated JSON is a derived projection.
SSCG state is the compatibility interpretation.
SKVI indexes source truth.
SCLV records change truth.
SODV governs publication truth.
Published documentation is a derived public projection.
NotebookLM is a corpus alignment and context tool, not canonical authority.
Mintlify is a publication surface, not canonical authority.

### Explicit Non-Authorizations

This contract does not authorize:
- generated changelog
- generated index
- generated report
- generated graph
- graph database
- graph visualization
- JSON / JSONL projection
- DuckDB projection
- HDF5 projection
- qxctl integration
- projector implementation
- schemas
- templates
- docs directory
- mint.json
- public documentation
- Mintlify configuration
- publication pipeline
- NotebookLM automation
- CI files
