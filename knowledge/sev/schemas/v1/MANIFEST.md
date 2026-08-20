# SEV v1 Schema Manifest

This directory owns the machine-readable v1 contracts for evolution cases, impact evidence, disposition plans, transition verification, SCSEV assessment, and disposable evolution graphs.

| Schema | Purpose |
|---|---|
| `case-open-input.schema.json` | Planned-change or encountered-novelty admission |
| `evolution-case.schema.json` | Append-forward noncanonical case evidence |
| `impact-result.schema.json` | Affected and unresolved surface evidence |
| `disposition-plan.schema.json` | Dependency graph, ready set, blockers, and actions |
| `transition-verification-input.schema.json` | Exact attempted action and reobservation input |
| `transition-verification-result.schema.json` | Evidence-based action outcome |
| `case-recalculation-input.schema.json` | Evidence-bound dynamic ready-set successor input |
| `evolution-session-binding.schema.json` | Exact SEV case/CURRENT binding to the existing lifecycle journal stream |
| `command-surface-assessment.schema.json` | SCSEV consequence coverage |
| `graph-projection.schema.json` | Disposable deterministic evolution graph |
| `novelty-bundle.schema.json` | Optional redactable offline export projection |
| `watch-policy.schema.json` | Opt-in freezing-path session trigger policy |

These schemas do not authorize external action, canonical mutation, command creation, novelty export, or coordinator-journal writes.
