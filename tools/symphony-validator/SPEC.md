# Symphony Validator Specification

**Candidate planning draft only. Not canonical. Not imported.**

## Specification Status
- declarative only
- non-executable
- not a JSON schema
- not a Markdown template
- not CI configuration
- not build logic
- not implementation authorization

## Purpose
Declare the exact declarative boundaries and expected non-agentic behavior of the C++ validator before implementation begins.

## Non-agentic Validator Doctrine
The validator is deterministic, explainable, and non-agentic. It produces structured evidence for humans, CI systems, qxctl, and agentic tools, but it does not perform interpretation, remediation, or architectural decision-making.
The validator does not infer intent.
The validator does not rewrite files.
The validator does not choose remedies.
The validator does not make architecture decisions.
The validator does not replace human review.
The validator does not replace agentic review.

## Evidence Model
Evidence model is truth. JSON is the structured evidence projection. Markdown is the agent/human ingestion projection.

## Dual Output Model
Evidence model is truth.
JSON is the structured evidence projection.
Markdown is the agent/human ingestion projection.
Markdown must not introduce claims, conclusions, or remediation steps that are not present in the source evidence model.

## JSON Evidence Projection
Provides structured machine-readable evidence for the administrative spine and CI.

## Markdown Agent/Human Ingestion Projection
Provides a stable, context-friendly ingestion surface without requiring agents to context-switch into raw JSON parsing.

## Synchronization Rules
JSON and Markdown must share the same run ID.
Markdown must mirror rule IDs from JSON.
Markdown must preserve severity, path, status, and reason.
Markdown must not invent interpretation.
Markdown must not suggest architecture unless the rule itself encodes the expected condition.
Agents may consume Markdown, but validator authority remains deterministic evidence only.

## Output Modes
- Default local mode
- CI mode
- Agent mode
- Strict machine mode
- Human review mode

## Rule ID Taxonomy
Families including `REPO.*`, `MODULE.*`, `INSTALL.*`, `NAMESPACE.*`, `TROLL.*`, `BUS.*`, `PYTHON.*`, `DOMAIN.*`, `INFRA.*`, `TERMS.*`, `LEAKAGE.*`, `PR.*`.

## Severity Model
`pass`, `info`, `warning`, `error`, `blocker`

## Status Model
Status reflects the deterministic outcome of a rule check.

## Exit-code Model
`0` to `5`, mapping to passes, errors, blockers, malformed repositories, or internal failures.

## Historical/Migration Exception Behavior
Stale names (e.g. `execution-node`, `native-execution`, `bus-agent`) are rejected except in historical contexts or rename records.

## Allowlist Behavior
Allowlists must never become silent bypasses. Every allowlist entry must produce evidence in JSON and Markdown.

## Refusal/Non-Remediation Behavior
The validator may report failures and identify expected/observed conditions. It must not rewrite files or choose remedies.

## Relationship to qxctl
The validator declares the tool boundary that `qxctl` will eventually mediate.

## Relationship to CI / PR gates
Provides deterministic exit codes and structured JSON/Markdown artifacts for gates.

## Relationship to SKV / SKVI / SCLV / SODV
The validator does not replace SKV / SKVI / SCLV / SODV records. It provides evidence to support them.

## Behavioral Non-goals
The validator must not choose infrastructure for users, assume Docker/Kubernetes/cloud providers, impose market-data/order-flow/trading doctrine, require Python for hot-path or administrative spine, ban optional isolated Python habitats, treat contract seeds as runtime implementation, convert monorepo modularity into microservices doctrine, absorb module sovereignty into root-level logic, become a runtime daemon, become a hidden coordinator, replace qxctl, replace Maestro, replace SKV / SKVI / SCLV / SODV records, become agentic, infer intent, auto-remediate files, or make architecture decisions.
Active project term c-o-r-e is absent except inside explicit forbidden-term scan descriptions.

## Non-authorization Statement
This candidate does not authorize canonical repository mutation, C++ validator implementation, or executable schema generation.
