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
- qxctl in the future
- agentic tools consuming canonical knowledge

## How Humans Should Read SKVI
Humans should use SKVI to discover the location of SCLV records, SODV publication rules, and module boundaries.

## How Validators May Later Check SKVI
symphony-validator may later check SKVI structure, but validator implementation is not authorized here.

## How Agentic Tools May Consume SKVI
Agentic tools may consume SKVI to orient themselves, but SKVI does not make architectural decisions.

## How qxctl May Later Consume SKVI
qxctl may later consume SKVI, but qxctl integration is not authorized here.

## How SKVI Relates to NotebookLM
NotebookLM aligns corpus context.
NotebookLM is not canonical authority.

## How SKVI Relates to Mintlify
Mintlify publishes derived official documentation.
Mintlify is not canonical authority.

## Safe-Use Rules
- SKVI indexes source truth; it does not create source truth.
- Do not attempt to execute SKVI.

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
this canonical seed authorizes no generated indexes, no generated reports, no implementation files, no source files, no schemas, no templates, no CI files, no documentation publication configuration, no Mintlify configuration, no qxctl integration, no validator implementation, no NotebookLM automation, no publication pipeline, no database files, no service files, no runtime processes, no deployment scripts, no installer scripts, no binary assets, and no binary renames.
