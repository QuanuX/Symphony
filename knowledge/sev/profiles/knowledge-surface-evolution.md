# SEV Knowledge-Surface Evolution Profile

## Status

Architect-ratified documentation-only SEV profile for reasoning about changes to canonical Symphony knowledge surfaces. It uses the existing SEV case, impact, disposition, verification, recalculation, status, recovery-advice, and closure operations. It defines no new engine operation, protocol, schema, qxctl command, registry, journal, or mutation authority.

## Purpose

Ensure a proposed addition, change, rename, supersession, deprecation, retirement, or removal is evaluated through the correct owner contracts and preserves the distinction between present canonical truth and historical evidence.

This profile identifies consequences. It does not decide the user's domain semantics, manufacture an owner, choose a final name or identity, write a patch, or authorize the transition.

## Applicable Surface

The profile applies to reviewed changes involving one or more of:

- a Contract Quad or Quad-governed companion;
- a vector or subvector contract surface;
- a namespace family, allocation, stable identity, or owner-routed term;
- a canonical registry, profile, schema, or relationship declaration;
- an implementation claim reflected by canonical knowledge; or
- a current surface whose history or compatibility obligations survive its active form.

It does not make every documentation edit an architectural evolution. Pure presentation changes remain with their owner unless they change meaning, identity, ownership, compatibility, current-state interpretation, or a governed relationship.

## Existing SEV Operation Sequence

1. `case_open` admits one `planned_change` against the exact current SAV CURRENT snapshot and caller-declared target.
2. `impact_assess` resolves affected owner contracts and reports missing coverage, evidence, relationships, and compatibility knowledge.
3. `disposition_plan` uses only `knowledge/sev/DISPOSITIONS.md` and preserves hard safety and semantic dependency edges.
4. Any canonical review or external action occurs outside SEV under its existing owner, permission, expected-state, audit, and recovery contracts.
5. `transition_verify` compares the attempted action with a complete successor CURRENT snapshot and the declared success predicate.
6. `case_recalculate` preserves prior evidence and derives the next case generation after every success, failure, blocker, or newly observed consequence.
7. `case_status` and `case_recover` provide inspection or deterministic recovery advice without mutation.
8. `case_close` proposes closure only after convergence or explicit evidence-preserving abandonment.

`command_surface_assess` remains the separate SCSEV operation and is used only when the knowledge change also adds, changes, deprecates, replaces, or retires a qxctl leaf. This profile does not wrap or duplicate it.

## Required Evidence

The existing SEV envelopes receive caller-supplied references and digests for the applicable evidence; this profile adds no fields. Assessment requires, where applicable:

- the exact baseline and proposed owner-controlled surfaces;
- the complete source CURRENT coverage state and applicable Accord References;
- the owning Contract Quad and any governed companions;
- `knowledge/NAMESPACES.md` plus the delegated family owner for identity consequences;
- `knowledge/SLANG.md` plus the semantic owner for nomenclature consequences;
- SKVI path, ownership, truth-role, and relationship evidence;
- exact schema, protocol, feature, command, engine-operation, receipt, binding, lifecycle, invariant, validation, and publication evidence whose owners declare them applicable;
- immutable predecessor, supersession, correction, retirement, or removal evidence needed to interpret history; and
- a complete successor reobservation for verification.

Missing ownership or partial CURRENT coverage is unresolved evidence. It cannot be converted into completeness by a prose assertion, model judgment, successful process exit, or absence from the latest tree.

## Consequence Families

Every applicable evolution must account for:

1. **semantic ownership** — which existing contract owns the meaning, or whether owner ratification must precede the change;
2. **Contract Quad and companion routing** — which current canonical surfaces state the result without turning a companion into another Quad member;
3. **identity and namespace** — preservation, new allocation, collision, lineage, and non-reuse under `knowledge/NAMESPACES.md` and the delegated owner;
4. **nomenclature** — preferred term, owner route, examples, counterexamples, and current/historical interpretation under `knowledge/SLANG.md`;
5. **SKVI** — path membership, title, owner, truth role, relationships, consumers, and projection eligibility;
6. **schema and protocol compatibility** — unchanged, additive, incompatible, versioned-successor, adapter, or preserved-unknown treatment as declared by each owner;
7. **implementation evidence** — code, descriptor, validator, test, and installed-process claims only where implementation exists;
8. **feature and administration evidence** — SSFV, qxctl, and backend-operation consequences only for applicable features and administrator-facing interactions;
9. **installation and lifecycle evidence** — receipts, bindings, selection, activation, docking, rollback, and retained state only where the affected surface has those relationships;
10. **current and historical interpretation** — which surface becomes current and which predecessor evidence remains historical;
11. **publication** — SODV consequences when a public projection exists or is proposed;
12. **verification and recovery** — exact successor observation, negative cases, interrupted ordering, rollback, and unresolved blockers;
13. **authority and thermal boundaries** — no authority transfer, canonical mutation, or new hot/warm dependency by implication; and
14. **nonclaims** — capabilities, integrations, migrations, or completeness that the change does not establish.

`not_applicable` is an owner-evidenced result, not a shortcut for omitted analysis. A consequence owned by another vector is referenced, not redefined here.

## Change-Kind Treatment

### Add

An addition establishes new current owner truth only after that owner ratifies it. It must not inherit an identity, namespace, feature, command, implementation, publication, or runtime claim merely from directory placement or naming symmetry.

### Change

A change compares exact baseline and proposed meaning. Compatible clarification, additive capability, and incompatible semantics remain distinct owner decisions. An incompatible meaning does not silently retain an identity whose owner requires a successor.

### Rename

A rename separates presentation, path, alias, and stable identity. The owner decides which of those changed and whether identity continuity is valid. Current routes and references move deliberately; preserved history keeps the spelling and path that were true at that time.

### Supersede

Supersession establishes an explicit successor relationship. The successor becomes current only after its own acceptance evidence passes. The predecessor remains addressable as historical evidence and is not rewritten to express the successor's semantics.

### Deprecate

Deprecation is a current owner-declared lifecycle state with explicit compatibility and replacement consequences. It is not removal, proof of disuse, or permission to ignore existing readers, installations, references, or user state.

### Retire

Retirement ends active use under the owner's lineage rules. Stable identities and namespaces remain tombstoned where their contracts require it; retirement does not free them for reuse or erase prior evidence.

### Remove

Removal deletes an active current surface only after ownership, references, compatibility, installation, retained state, publication, rollback, and historical disposition are explicit. Absence from the current tree does not assert that the surface, identity, behavior, or evidence never existed.

## Current Truth and Historical Evidence

Current owner contracts answer what Symphony declares now. Immutable records, prior versions, released schemas, stored identities, receipts, journals, and version-control evidence describe earlier states or continuing compatibility obligations. Both may be canonical within their bounded roles, but they answer different questions.

Evolution is prospective and append-forward:

- current surfaces may be corrected or replaced through reviewed owner changes;
- historical evidence retains the values, names, paths, and conclusions that were true when recorded;
- a correction or successor changes the active interpretation without falsifying the predecessor record;
- a historical identifier is not assumed active merely because it remains readable;
- a removed current file does not erase installed or externally persisted evidence; and
- disagreement among current owner truth, implementation evidence, and the latest applicable history remains visible drift until explicitly resolved.

Agents and publication consumers must identify whether an answer describes current behavior, historical behavior, or an unresolved transition. They must not splice an old rule into a new surface and present the result as one coherent current contract.

## Completion Evidence

The case is complete only when applicable owners, current routes, compatibility treatment, and successor observation agree; every required consequence is resolved or explicitly blocked; historical evidence remains interpretable; and rollback or abandonment preserves the last proven state. A green documentation build, validator exit, or engine response alone cannot prove semantic convergence.

## Non-Authorization Statement

This profile does not create a canonical writer, migration executor, history editor, namespace allocator, terminology owner, compatibility authority, publication pipeline, background watcher, new SEV operation, or qxctl surface. It neither runs nor requires a model and creates no hot/warm-path dependency.
