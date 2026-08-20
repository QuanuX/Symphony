# Symphony Evolution Vector Intent

## Purpose

The Symphony Evolution Vector (SEV) defines the deterministic procedure for admitting a planned change or encountered novelty, assessing its consequences against one exact SAV CURRENT snapshot, selecting a bounded disposition, recalculating after observed results, and verifying closure.

SEV makes Symphony adaptive without making it autonomous in authority. It explains what may change, which relationships are affected, which actions are ready, which blockers remain, what evidence is required, and whether an externally applied transition actually converged.

## Source-Truth Boundary

`knowledge/sev/` owns:

- `sevcase:` and `sevdisp:` identity grammar;
- evolution-case, impact, disposition, recalculation, verification, and closure semantics;
- the closed disposition vocabulary;
- dependency-graph and ready-set rules;
- novelty admission and voluntary export boundaries;
- the SCSEV qxctl command-surface evolution profile.

SEV does not own SAV composition truth, SSFV feature truth, qxctl command truth, engine operations, package receipts, Maestro state, SSIAG decisions, STAV ledgers, coordinator journals, or source-vector contracts.

## SCSEV Intent

SCSEV is the first governed SEV profile. It evaluates the repercussions of adding, changing, deprecating, or retiring a qxctl leaf by consuming existing SSFV feature-administration truth, the expected qxctl registry, observed qxctl evidence, and engine descriptors. It does not create a third engine or duplicate registry in v1.

## Caller-Neutral Intent

Any authenticated subject with effective host ownership or the required SSIAG grant may request, approve, apply, verify, recover, or close an evolution operation. Caller class is never an authority input. Optional AI assistance is ordinary caller assistance and is never required.

## Implementation and Thermal Intent

The independently installable C++26 SEV engine is a freezing-path, report-only and proposal-only process in its first runtime slice. qxctl remains the preferred headless administrator. Durable transition mutation reuses the knowledge-session coordinator and external qxctl lifecycle adapters under separate SSIAG authorization and STAV evidence.

## Non-Authorization Statement

SEV does not apply its plans, mutate canonical truth, invent feature/command/operation identities, grant exemptions, reorder hard safety dependencies, select newest versions, execute Maestro components, require a database or AI service, or enter a hot/warm path.
