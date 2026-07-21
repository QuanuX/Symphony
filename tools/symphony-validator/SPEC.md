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
The evidence model is truth. The implemented projection is deterministic, line-oriented evidence followed by one summary line. JSON and Markdown projections remain future surfaces.

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

The Architect-ratified STAV v1 JSON Schema/conformance fixtures, six common SKV process/descriptor/receipt/proposal/provider-evidence schemas, four SKVI operation/result schemas, and five SCLV v3 operation/result schemas are canonical protocol truth, not generated projections. The artifact checker may allow only their 43 exact paths and must emit `artifact.canonical_json_authorized` evidence for every encountered file with `knowledge/stav/SPEC.md`, `knowledge/SPEC.md`, `knowledge/skvi/SPEC.md`, or `knowledge/sclv/SPEC.md` authority as applicable. Prefix or extension-wide JSON allowlisting is prohibited; any new canonical JSON artifact requires an explicit contract and validator update.

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

## Caller-Authority Regression Check
The checker detects a bounded vocabulary of active Markdown statements that make authority depend on caller class. It is deterministic regression evidence, not natural-language understanding and not proof that every possible semantic expression is safe.

### Discovery scope

The checker scans `README.md` and `INTENT.md` at the repository root when present, plus Markdown files beneath `knowledge/`, `modules/`, `libraries/`, `tools/qxctl/`, and `tools/symphony-validator/`. A directory named `build` is pruned at any depth. `tools/symphony-validator/tests/` is excluded so adversarial fixtures do not become repository findings. Markdown fixtures beneath `knowledge/stav/fixtures/` remain in scope; canonical non-Markdown STAV fixtures are outside this check.

Discovery uses lexical repository-relative paths. It inspects link metadata without following targets. Every in-scope Markdown symlink, including a broken link, produces `caller_authority.symlink_unsupported`; its target is never opened, resolved, or reported as the scanned path. Metadata and directory-iteration failures produce `caller_authority.discovery_failed`. Pending discovery findings are sorted by lexical path, rule ID, and detail before content findings are emitted.

### Historical record boundaries

`knowledge/sclv/CHANGELOG.md` is scanned through the line immediately before its first `- record_id:` boundary. `knowledge/sodv/RELEASES.md` is scanned through the line immediately before its first `- release_record_id:` boundary. The boundary and append-only historical body are not scanned, and the checker emits `caller_authority.historical_region_exempt`. This is a structural exception for the two canonical ledger surfaces, not a phrase allowlist or a general path exemption.

### Matching and evidence

The checker lowercases and tokenizes bounded paragraphs, treats `.`, `!`, and `?` as sentence boundaries, and applies fixed phrase and distance windows. It detects the following rule families:

- `caller_authority.class_subject_modal`;
- `caller_authority.class_subject_status`;
- `caller_authority.class_exclusive_operation`;
- `caller_authority.class_targeted_availability`;
- `caller_authority.human_exclusive_governance`;
- `caller_authority.caller_type_decision`.

Negation is evaluated only within the bounded predicate construction that it can negate; unrelated sentence text does not suppress a finding. Evidence uses stable rule IDs, lexical paths, one-line or wrapped line ranges, and canonical class IDs. Duplicate matches of the same rule within one paragraph emit one finding.

This matcher is not an NLP engine, does not parse source code or abstract syntax trees, and does not claim arbitrary semantic coverage. A clean result means only that the implemented bounded rules found no regression in the scanned surfaces.

### Failure and resource behavior

The maximum physical line is 64 KiB, the maximum normalized paragraph is 256 KiB, and the maximum active Markdown file is 4 MiB. The structurally bounded SCLV and SODV ledgers are exempt from the whole-file limit because only their active preambles are scanned; line and paragraph limits still apply to those preambles. Limit violations use `caller_authority.line_length_exceeded`, `caller_authority.paragraph_size_exceeded`, and `caller_authority.file_size_exceeded`. File-open or negative-position failures use `caller_authority.unreadable`. These conditions, symlink findings, discovery failures, and caller-authority findings all fail closed.

### Authority and CLI behavior

The checker is read-only and non-remediating. It does not modify repository content, select a remedy, or make an authorization decision. When execution reaches this checker and it fails, the CLI emits one final summary and exits `21`. Earlier checkers retain their existing fail-fast precedence; exit `21` precedes only the checks that follow caller-authority validation in the CLI sequence.

Runtime source/AST analysis, remediation, `qxctl` mediation, and CI/PR-gate integration are deferred and unauthorized by this increment.

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
