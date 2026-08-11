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
The implemented skill surface is direct execution through `symphony-validator check --repo <path> [--json]` and exact installed invocation through `qxctl validate scan|debug`. It emits deterministic line evidence or one `symphony.validation.result.v1` object plus a process status. CI invocation and Markdown projection remain planned but unimplemented.

The repository check includes SACV registry shape, ownership, classification, SKVI coverage, and no-follow owner-document presence. Use `qxctl sacv check` with an exact installed engine for OpenAPI syntax/profile, reference, security, example-safety, and registry/document alignment evidence; do not mistake the validator's registry check for a partial OpenAPI parser.

The repository check also includes SODV local release-record shape, time order, lineage, publication-unit preservation, and no-follow ledger presence. Use `qxctl sodv check|verify|recover` with an exact installed engine for richer release evidence and caller-supplied external observations; do not mistake the validator for a Git-host or package-provider client.

The repository check requires the canonical SSFV contract, namespace, empty-or-future registry, feature-file format, engine Contract Quad/build surface, and qxctl grammar/client anchors. It recognizes eighteen exact v1/v2 JSON schemas. It does not decide feature-worthiness, validate distributed semantic records, or substitute for the independently installed `symphony-ssfv` engine.

## Deterministic, Non-Autonomous Behavior
The validator is deterministic, explainable, and non-autonomous.
The validator produces evidence, including caller-authority regression findings (exit code 21).
The validator does not fix the repository.
The validator does not choose remedies.
Every permission-bearing caller may consume either implemented projection. A future Markdown projection remains deferred. The validator itself remains non-autonomous and authority-free. Caller types remain descriptive.

## Invocation Procedure
1. Build the checked-in C++26 target according to `INSTALL.md`.
2. Run `symphony-validator check --repo <path>` or add `--json`; for protected policy/baseline evaluation use `qxctl validate scan` with the exact prefix, version, TOPS ID, and optional profile/baseline IDs.
3. Consume every evidence line through the single final `summary` line; do not stop after the first matching line.
4. Treat exit `21` as a bounded caller-authority, discovery, symlink, stream, or resource failure and use the stable rule ID and lexical path to locate the evidence.
5. Treat exit `22` as a SACV registry failure and use `sacv.registry.*` evidence to locate the exact entry boundary.
6. Treat exit `23` as a SODV release-ledger failure and use `sodv.releases.*` evidence to locate the record boundary.
7. Refer to the normative Caller-Authority Regression Check, SACV Registry Boundary, and SODV Release Ledger Boundary in `SPEC.md` before interpreting scope or exclusions.

## Output Consumption Behavior
Evidence lines plus the summary and the structured JSON result are equivalent current projections. Consumers must retain the complete selected projection and process status. qxctl verifies evidence/result digests, evaluates warning policy only after the complete scan, and keeps filters presentation-only. Any future Markdown projection must introduce no claims, conclusions, or remediation steps absent from that model. A clean bounded scan is not universal semantic proof.

## Refusal/non-remediation Behavior
The validator does not infer intent.
The validator does not rewrite files.
The validator does not choose remedies.
The validator does not make architecture decisions.
The validator does not replace permission-backed ratification or semantic review.

## Non-goals
The validator must not perform autonomous semantic decisions, infer intent, rewrite files, choose remedies, make architecture decisions, replace qxctl, replace Maestro, replace SKV / SKVI / SCLV / SACV / SODV / SSFV records, become a runtime daemon, become hidden coordinator logic, become a module implementation, choose infrastructure for users, impose market-data/order-flow/trading doctrine, require Python for hot-path or administrative spine, or ban optional isolated Python habitats.

## Non-authorization Statement
This skill authorizes use of the checked-in deterministic C++26 parser/checker, JSON projector, versioned install, and qxctl validation surface. It does not authorize executable schema generation, CI mutation, repository mutation, publication, or remediation.
