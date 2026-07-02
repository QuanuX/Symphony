# Symphony Change Log Vector Skill

## Canonical Target
`knowledge/sclv/SKILL.md`

## Purpose
To provide operational guidance for humans, reviewers, validators, CI systems, and tools consuming SCLV.

## Intended Users
- maintainers
- reviewers
- validators
- CI systems
- qxctl in the future
- agentic tools consuming canonical knowledge
- documentation maintainers

## How Humans Should Read SCLV
Humans read SCLV to understand the canonical consequences, relationships, and boundaries of architectural changes.

## How Reviewers Should Use SCLV
Reviewers verify that SCLV correctly declares the truth of the change and its downstream consequences before merging.

## How Validators May Later Check SCLV
symphony-validator may later check SCLV structure, but validator implementation is not authorized here.

## How Agentic Tools May Consume SCLV
Agentic tools may consume SCLV to understand canonical change context, but SCLV does not make architectural decisions.

## How qxctl May Later Consume SCLV
qxctl may later consume SCLV, but qxctl integration is not authorized here.

## How SCLV Relates to Git History
Git history and PR history are supporting evidence. They do not replace SCLV change truth.

## How SCLV Relates to PR History
Git history and PR history are supporting evidence. They do not replace SCLV change truth.

## How SCLV Relates to SKVI
SCLV records change truth; it does not create source truth. SKVI indexes source truth.

## How SCLV Relates to SODV
SCLV records architectural change truth. SODV governs the publication of that truth.

## How SCLV Relates to NotebookLM
NotebookLM aligns corpus context. NotebookLM is not canonical authority.

## How SCLV Relates to Mintlify
Mintlify publishes derived official documentation. Mintlify is not canonical authority.

## Safe-Use Rules
SCLV records change truth; it does not create source truth. It does not replace PR reviews or Git history.

## Non-Scope
SCLV is not a generated changelog yet.
SCLV is not a generated index yet.
SCLV is not a database.
SCLV is not Git history.
SCLV is not PR history.
SCLV is not a replacement for PR review.
SCLV is not a replacement for SKVI.
SCLV is not a replacement for SODV.
SCLV is not a replacement for SSCG.
SCLV is not NotebookLM.
SCLV is not Mintlify.
SCLV is not a docs site.
SCLV is not qxctl.
SCLV is not symphony-validator.
SCLV does not create runtime behavior.
SCLV does not enforce runtime behavior.

## Non-Authorization Statement
This canonical seed authorizes no CHANGELOG.md, no INDEX.md, no INSTALL.md, no generated changelogs, no generated indexes, no generated reports, no implementation files, no source files, no schemas, no templates, no CI files, no documentation publication configuration, no Mintlify configuration, no qxctl integration, no validator implementation, no NotebookLM automation, no publication pipeline, no database files, no service files, no runtime processes, no deployment scripts, no installer scripts, no binary assets, and no binary renames.
