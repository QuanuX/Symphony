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

## How qxctl May Later Consume SKVI
qxctl may later consume SKVI, but qxctl integration is not authorized here.

## How SKVI Relates to NotebookLM
NotebookLM aligns corpus context.
NotebookLM is not canonical authority.
NotebookLM must use the corpus interpretation rule in `knowledge/INTENT.md`: current contracts and the latest applicable append-only records determine present posture, while older records remain historical evidence.

## How SKVI Relates to Mintlify
Mintlify publishes derived official documentation.
Mintlify is not canonical authority.

## Safe-Use Rules
- SKVI indexes source truth; it does not create source truth.
- Do not attempt to execute SKVI.
- Do not treat an isolated historical SCLV or SODV statement as current when a later canonical record corrects, supersedes, or completes it.

## Non-Scope
SKVI is not a generated database yet.
SKVI is not a generated index yet.
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
SKVI does not create runtime behavior.
SKVI does not enforce runtime behavior.

## Non-Authorization Statement
This canonical surface authorizes no generated indexes, generated reports, new implementation or source files, schemas, templates, CI files, documentation publication configuration, Mintlify configuration, qxctl integration, validator capability beyond the separately bounded `tools/symphony-validator/` contract, NotebookLM automation, publication pipeline, database files, service files, runtime processes, deployment scripts, installer scripts, binary assets, or binary renames.
