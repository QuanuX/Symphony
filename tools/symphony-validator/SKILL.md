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
The implemented skill surface is direct execution through `symphony-validator check --repo <path>`. It emits deterministic line-oriented evidence, a summary, and a process exit status. qxctl/CI invocation and JSON/Markdown projectors remain planned but unimplemented.

## Deterministic, Non-Autonomous Behavior
The validator is deterministic, explainable, and non-autonomous.
The validator produces evidence, including caller-authority regression findings (exit code 21).
The validator does not fix the repository.
The validator does not choose remedies.
Every permission-bearing caller may consume the implemented line-oriented evidence. A future Markdown projection remains deferred. The validator itself remains non-autonomous and authority-free. Caller types remain descriptive.

## Invocation Procedure
1. Build the checked-in C++26 target according to `INSTALL.md`.
2. Run `symphony-validator check --repo <path>` against the intended repository root.
3. Consume every evidence line through the single final `summary` line; do not stop after the first matching line.
4. Treat exit `21` as a bounded caller-authority, discovery, symlink, stream, or resource failure and use the stable rule ID and lexical path to locate the evidence.
5. Refer to the normative Caller-Authority Regression Check in `SPEC.md` before interpreting scope or exclusions.

## Output Consumption Behavior
Evidence lines and the final summary are the current implementation output. Consumers must retain the complete output and process status. Any future JSON or Markdown projection must derive from one evidence model, share stable rule identifiers, and introduce no claims, conclusions, or remediation steps absent from that model. A clean bounded scan provides deterministic evidence but must not be interpreted as universal semantic proof or an all-encompassing semantic analysis of the codebase.

## Refusal/non-remediation Behavior
The validator does not infer intent.
The validator does not rewrite files.
The validator does not choose remedies.
The validator does not make architecture decisions.
The validator does not replace permission-backed ratification or semantic review.

## Non-goals
The validator must not perform autonomous semantic decisions, infer intent, rewrite files, choose remedies, make architecture decisions, replace qxctl, replace Maestro, replace SKV / SKVI / SCLV / SODV records, become a runtime daemon, become hidden coordinator logic, become a module implementation, choose infrastructure for users, impose market-data/order-flow/trading doctrine, require Python for hot-path or administrative spine, or ban optional isolated Python habitats.

## Non-authorization Statement
This skill authorizes use of the checked-in deterministic C++26 parser/checker. It does not authorize executable schema generation, structured projectors, qxctl/CI integration, repository mutation, publication, or remediation.
