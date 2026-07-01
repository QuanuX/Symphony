# Symphony Validator Skill

****

## Skill Purpose
Provide deterministic, structured evidence of repository compliance with Symphony doctrinal boundaries.

## Intended Users
- humans
- CI systems
- qxctl
- agentic tools consuming reports

## Planned Skill Surface
Execution via direct binary, qxctl, or CI. Emits JSON and Markdown evidence.

## Non-agentic Behavior
The validator is deterministic, explainable, and non-agentic.
The validator produces evidence.
The validator does not fix the repository.
The validator does not choose remedies.
Agentic tools may consume the Markdown projection, but the validator itself remains non-agentic.

## Output Consumption Behavior
Evidence model is truth. JSON is the structured evidence projection. Markdown is the agent/human ingestion projection. Markdown must not introduce claims, conclusions, or remediation steps that are not present in the source evidence model.

## Refusal/non-remediation Behavior
The validator does not infer intent.
The validator does not rewrite files.
The validator does not choose remedies.
The validator does not make architecture decisions.
The validator does not replace human review.
The validator does not replace agentic review.

## Non-goals
The validator must not become agentic, infer intent, rewrite files, choose remedies, make architecture decisions, replace qxctl, replace Maestro, replace SKV / SKVI / SCLV / SODV records, become a runtime daemon, become hidden coordinator logic, become a module implementation, choose infrastructure for users, impose market-data/order-flow/trading doctrine, require Python for hot-path or administrative spine, or ban optional isolated Python habitats.

## Non-authorization Statement
This canonical seed does not authorize C++ validator implementation, or executable schema generation.
