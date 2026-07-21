# Symphony Knowledge Vector Index Skill

## Canonical Target
`knowledge/skvi/SKILL.md`

## Purpose
To outline how humans and automated systems should safely read, interpret, and operate against SKVI.

## Intended Users
- maintainers
- reviewers
- validators
- CI systems
- future qxctl consumers
- agentic tools consuming canonical knowledge

## How Readers Consume SKVI
Readers use SKVI to discover the location of SCLV records, SODV publication rules, and module boundaries.

## How the Validator Checks SKVI
The checked-in `tools/symphony-validator/` implementation may read SKVI to check entry shape, required-surface coverage, relative-path safety, path existence, uniqueness, and SCLV cross-references. Treat its output as deterministic evidence, not permission to rewrite SKVI or infer architectural intent.

## How Agentic Tools May Consume SKVI
Agentic tools may consume SKVI to orient themselves, but SKVI does not make architectural decisions.

## How qxctl Consumes SKVI
qxctl may invoke installed `symphony-skvi` inspect, check, propose, and project operations through the common bounded process protocol. Treat proposals and projections as noncanonical evidence.

## How SKVI Relates to NotebookLM
NotebookLM aligns corpus context.
NotebookLM is not canonical authority.
NotebookLM must use the corpus interpretation rule in `knowledge/INTENT.md`: current contracts and the latest applicable append-only records determine present posture, while older records remain historical evidence.

## How SKVI Relates to Mintlify
Mintlify publishes derived official documentation.
Mintlify is not canonical authority.

## Safe-Use Rules
- SKVI indexes source truth; it does not create source truth.
- Use only the installed SKVI engine operations authorized by `knowledge/skvi/SPEC.md`.
- Never treat engine output as ratification or write it directly into canonical files through an unratified path.
- Do not treat an isolated historical SCLV or SODV statement as current when a later canonical record corrects, supersedes, or completes it.

## Non-Scope
SKVI is not a generated database.
SKVI projections are not canonical indexes.
SKVI is not a search engine.
SKVI is not NotebookLM.
SKVI is not Mintlify.
SKVI is not a docs site.
SKVI is not qxctl.
SKVI is not symphony-validator.
SKVI does not replace module contracts.
SKVI does not replace SCLV.
SKVI does not replace SODV.
SKVI does not replace SSCG.
SKVI does not execute or enforce application runtime behavior.

## Non-Authorization Statement
This skill authorizes no canonical apply or self-ratification. Engine proposals require review by a caller holding the applicable permission; derived projections remain disposable and SODV governs publication.
