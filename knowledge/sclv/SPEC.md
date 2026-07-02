# Symphony Change Log Vector Specification

## Canonical Target
`knowledge/sclv/SPEC.md`

## Specification Status
- declarative only
- non-executable
- not a generated changelog
- not a generated index
- not a JSON schema
- not a Markdown template
- not CI configuration
- not qxctl integration
- not validator implementation

## Purpose
To establish the declarative behavioral specification for future SCLV change records.

## SCLV Behavioral Model
Canonical repository knowledge files are source truth.
SKVI indexes source truth.
SCLV records change truth.
SODV governs publication truth.
Published documentation is a derived public projection.

MANIFEST.md is declared contract truth.
Code is implementation truth.
Generated JSON is a derived projection.
SSCG state is the compatibility interpretation.

## Initial SCLV Record Scope
SCLV records are organized conceptually into layers defining canonical change events, relationships, supporting evidence, and (deferred) generated projections.

## Layer 0 Canonical Change Events
- root governance changes
- module contract seed changes
- tool contract seed changes
- Knowledge Vector changes
- SKVI changes
- SCLV changes
- SODV changes
- doctrine changes
- naming migrations
- compatibility-affecting changes

## Layer 1 Change Relationships
- source PRs
- merge commits
- affected canonical files
- affected modules
- affected tools
- affected knowledge-vector surfaces
- dependency consequences
- compatibility consequences
- follow-up tasks
- rollback considerations

## Layer 2 Supporting Evidence
- Git history
- PR history
- validator evidence
- SKVI mappings
- SSCG compatibility interpretations
- NotebookLM corpus alignment
- SODV publication decisions

## Layer 3 Future Generated Projections
- generated changelogs
- generated change indexes
- generated JSON projections
- generated Markdown projections
- published documentation projections

Future generated projections are not authorized by this canonical seed.
Future generated changelogs are not authorized by this canonical seed.
Future generated indexes are not authorized by this canonical seed.
Future JSON schemas are not authorized by this canonical seed.
Future Markdown templates are not authorized by this canonical seed.

## Change Truth versus Supporting Evidence Boundaries
SCLV is the change truth. Git history is version-control evidence, not SCLV itself. PR history is review and merge evidence, not SCLV itself.

## What Future SCLV Records May Claim
SCLV records canonical changes, relationships, dependencies, migration events, compatibility consequences, and architectural deltas.

## What Future SCLV Records Must Not Claim
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

## Relationship to SKVI
SKVI indexes source truth. SCLV records change truth.

## Relationship to SODV
SODV governs publication truth.

## Relationship to SSCG
SSCG interprets compatibility.

## Relationship to Git history
Git history is version-control evidence, not SCLV itself.

## Relationship to PR history
PR history is review and merge evidence, not SCLV itself.

## Relationship to symphony-validator evidence
symphony-validator produces deterministic evidence.

## Relationship to qxctl
qxctl may later consume SCLV, but qxctl integration is not authorized here.

## Relationship to NotebookLM
NotebookLM aligns corpus context.

## Relationship to Mintlify
Mintlify publishes derived official documentation.

## Deferred Surfaces
Future generated changelogs are not authorized yet.
Future generated indexes are not authorized yet.
Future public documentation is not authorized yet.

## Non-Authorization Statement
This canonical seed authorizes no CHANGELOG.md, no INDEX.md, no INSTALL.md, no generated changelogs, no generated indexes, no generated reports, no implementation files, no source files, no schemas, no templates, no CI files, no documentation publication configuration, no Mintlify configuration, no qxctl integration, no validator implementation, no NotebookLM automation, no publication pipeline, no database files, no service files, no runtime processes, no deployment scripts, no installer scripts, no binary assets, and no binary renames.
