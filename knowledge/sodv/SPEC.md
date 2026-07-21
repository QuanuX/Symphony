# Symphony Official Documentation Vector Specification

## Canonical Target
`knowledge/sodv/SPEC.md`

## Specification Status
- declarative only
- proposal/projection engine architecture authorized
- not public documentation
- not a docs site
- not Mintlify configuration
- not a publication pipeline
- not generated documentation
- not a generated index
- not a JSON schema
- not a Markdown template
- not CI configuration
- bounded qxctl proposal/read integration authorized after contract merge
- not validator implementation; checked in part by the separately bounded validator

## Purpose
To define the declarative behavioral boundaries for SODV publication governance.

## SODV Behavioral Model
SODV establishes publication boundaries declaratively. Its append-only module-release protocol is operational through permission-backed ratified repository records and external-state verification. The subordinate SODV engine may inspect, check, propose, verify, reconcile safe noncanonical state, and build disposable projections. Future generators and pipelines may build derived public-documentation artifacts only under separately ratified contracts.

SODV also governs independently consumable module publication. `RELEASES.md` is the canonical append-only authorization and completion ledger for that bounded purpose. It does not authorize a general publication pipeline.

## Module Publication Protocol

A module release is a transaction with two canonical stages:

- authorization: merged before publication and binding module path, semantic version, tag name, source commit, source PR, expected evidence, completion gates, and explicit exclusions;
- completion: appended only after immutable tag publication and clean-cache public resolution, recording actual tag objects, checksums, verification, and consumer consequences.

Authorization and completion are immutable records. Pending state is noncanonical and may exist only under `.git/symphony/releases/pending/`. A resumed session must compare the authorization with actual repository and package-proxy state. A mismatched existing tag blocks publication and requires review by a caller holding the applicable release permission. A matching published tag may be verified and completed forward. Tags must never be moved to repair a ledger.

For Go modules, a workspace-composed test or warm module cache is insufficient completion evidence. The completion gate requires `GOWORK=off`, an empty `GOMODCACHE`, and resolution through the public module proxy or its documented authoritative fallback.

Any pre-publication Go archive or checksum simulation for a module nested in the monorepo must use the VCS-aware module-zip algorithm against the repository root, exact revision, and module subdirectory, equivalent to `golang.org/x/mod/zip.CreateFromVCS`. A raw subdirectory archive, `git archive` limited to the module subtree, or directory-only proxy builder is not canonical packaging evidence. In particular, VCS-aware packaging must preserve Go's repository-root `LICENSE` inheritance and all nested-module, vendor, symlink, path, and size rules.

## Initial SODV Governance Scope
The initial SODV scope covers sources, relationships, and evidence. Projections are deferred.

## Layer 0 Canonical Publication Sources
- root governance files
- module contract files
- tool contract files
- Knowledge Vector files
- SACV registry and owner-controlled API contracts
- SKVI files
- SCLV files
- SODV files
- current validator evidence for contract shape and indexed-path presence
- future generated projections
- future release documentation inputs

## Layer 1 Publication Relationships
Relationships among:
- canonical source files
- public documentation pages
- documentation sections
- publication surfaces
- Mintlify publication surfaces
- derived projections
- source truth dependencies
- change-truth dependencies
- compatibility interpretation dependencies
- publication approval states

## Layer 2 Publication Evidence
References to:
- SKVI mappings
- SCLV change records
- validator evidence
- SSCG compatibility interpretations
- Git history
- PR history
- NotebookLM corpus alignment
- Mintlify publication targets

## Layer 3 Future Publication Projections
- generated public documentation
- generated documentation indexes
- generated JSON projections
- generated Markdown documentation projections
- Mintlify configuration
- publication pipelines

Future publication projections are not authorized by this canonical seed.
Future public documentation files are not authorized by this canonical seed.
Future docs directory creation is not authorized by this canonical seed.
Future Mintlify configuration is not authorized by this canonical seed.
Future publication pipelines are not authorized by this canonical seed.
Future generated documentation is not authorized by this canonical seed.
Future JSON schemas are not authorized by this canonical seed.
Future Markdown templates are not authorized by this canonical seed.

## Publication Truth Versus Source Truth Boundaries
Canonical repository knowledge files are source truth. Published documentation is a derived public projection. SODV governs publication truth.

## Publication Truth Versus Change Truth Boundaries
SCLV records change truth. SODV uses those records to govern publication.

## What Future Publication Records May Govern
They may govern what files, indices, diagrams, and structures are assembled for public presentation on docs sites or embedded in binary outputs.

## What Future Published Documentation May Claim
It may claim derived alignment with canonical contract truth.

## What Future Published Documentation Must Not Claim
It must not claim to be the source of canonical truth itself.

## Relationship to SKVI
SKVI indexes source truth.

## Relationship to SACV
SACV owns API-contract governance and registration. SODV may authorize a derived documentation or vendor projection only after the canonical owner contract is registered and validated. Mintlify settings, combined specifications, SDK examples, live playgrounds, and MCP tools remain derived publication surfaces.

## Relationship to SCLV
SCLV records change truth.

## Relationship to SSCG
MANIFEST.md is declared contract truth. Code is implementation truth. Generated JSON is a derived projection. SSCG state is the compatibility interpretation.

## Relationship to Canonical Repository Knowledge Files
Canonical repository knowledge files are source truth.

## Relationship to Public Documentation
Published documentation is a derived public projection.

## Relationship to Mintlify
Mintlify is a publication surface, not canonical authority.

## Relationship to NotebookLM
NotebookLM is a corpus alignment and context tool, not canonical authority.

## Relationship to symphony-validator Evidence
The checked-in validator produces deterministic evidence for required SODV contract anchors and indexed-path presence. It does not currently parse `RELEASES.md` transactions or verify Git tags, public-proxy state, checksum-database state, or release completion.

## Relationship to qxctl
qxctl may invoke bounded SODV proposal/read operations but does not own release semantics, create tags, publish artifacts, or append canonical records.

## SODV Engine Operations

The initial `symphony-sodv` operation set is:

- `inspect`: report release ledger, provider, and contract compatibility;
- `check`: produce deterministic authorization/completion and projection-readiness evidence;
- `propose`: create immutable forward-only authorization, completion, recovery, or documentation-change proposals without writing canonical files;
- `verify`: compare a bounded authorization with observed immutable Git/package state without declaring publication complete by itself;
- `recover`: reconcile safe noncanonical pending state and emit a no-op, failure, or forward proposal;
- `project`: build disposable release and publication evidence.

The engine uses `symphony.knowledge.engine-process.v1` and is administered through `qxctl sodv ...`. It MUST NOT create or move tags, upload artifacts, warm caches to simulate public evidence, append `RELEASES.md`, publish Mintlify content, or automate NotebookLM ingestion in its initial release.

## Relationship to Git History
Git history is version-control evidence.

## Relationship to PR History
PR history is review and merge evidence.

## Deferred Surfaces
`docs/`, `mint.json`, public documentation, schemas, templates, implementation, index generation.

SODV is not public documentation. SODV is not a docs site. SODV is not Mintlify. SODV is not NotebookLM. SODV is not a publication pipeline. SODV is not a generated documentation system yet. SODV is not a generated index yet. SODV is not a documentation template system yet. SODV is not a schema system. SODV is not qxctl. SODV is not symphony-validator. SODV is not SKVI. SODV is not SCLV. SODV is not SSCG. SODV does not replace canonical repository knowledge files. SODV does not replace module contracts. SODV does not replace tool contracts. SODV does not replace PR review. SODV does not create runtime behavior. SODV does not enforce runtime behavior.

## Non-Authorization Statement
This specification authorizes the bounded SODV proposal/projection engine and qxctl invocation after contract merge. It authorizes no canonical apply, public documentation files, docs directory, `mint.json`, Mintlify configuration, tag creation, external publication, NotebookLM automation, general publication pipeline, or release-completion claim.
