# Symphony Official Documentation Vector Skill

## Canonical Target
`knowledge/sodv/SKILL.md`

## Purpose
To provide operational guidance for consuming and interacting with the declarative SODV publication-governance contract surface.

## Intended Users
- maintainers
- documentation maintainers
- reviewers
- validators
- CI systems
- future qxctl consumers
- agentic tools consuming canonical knowledge

## How Humans Should Read SODV
Read SODV to understand the governance rules determining how source truth maps into published projections.

## How Documentation Maintainers Should Use SODV
Documentation maintainers must reference SODV to ensure generated docs and Mintlify configurations align with canonical publication expectations.

## How Reviewers Should Use SODV
Reviewers verify that changes to SKV structures are accurately permitted by SODV publication boundaries before being merged.

## How to Publish a Module Release

1. Append and merge an authorization in `RELEASES.md` that binds every module path and version to an exact commit.
2. Confirm each proposed tag is absent, or already resolves to the authorized immutable object. Any mismatch fails closed.
3. Test the module subtree at the authorized commit, not the current checkout by convenience. If simulating its release archive, use VCS-aware module packaging from the repository root; never hand-roll a proxy zip from only the module directory.
4. Create and push only the authorized annotated tags.
5. Resolve each version with `GOWORK=off`, the public Go proxy, and an empty module cache.
6. Append a completion record with tag objects, public checksums, tests, and consumer consequences.
7. Update consumers only from public artifacts and record the closure through SCLV when architecturally significant.

An interrupted session resumes from `.git/symphony/releases/pending/` and reconciles observed external state. Never move a tag, rewrite authorization, or claim completion from a warm local cache.

## How the Validator Checks SODV
The checked-in validator checks required SODV contract anchors and indexed-path presence. It does not currently parse the release ledger, contact Git hosting or package proxies, validate checksums, or declare a release complete.

## How Agentic Tools May Consume SODV
Agentic tools may consume SODV to understand publication-governance context, but SODV does not make architectural decisions.

## How qxctl Consumes SODV
qxctl may invoke installed `symphony-sodv` inspect, check, propose, verify, recover, and project operations. Treat all outputs as evidence or noncanonical proposals until a caller with the required permission completes the separately governed action.

## How SODV Relates to SKVI
SKVI indexes source truth; SODV defines how those indexes may be published.

## How SODV Relates to SACV
SACV governs API contracts. Before publishing an API, verify its SACV registry entry, owner, OpenAPI 3.2.0 compatibility, audience, server policy, interactive-request decision, SDK state, and MCP exposure decision.

## How SODV Relates to SCLV
SCLV records change truth; SODV dictates how changes are reflected in public release documentation.

## How SODV Relates to Canonical Repository Knowledge Files
SODV governs publication truth; it does not create source truth. Canonical repository knowledge files are source truth.

## How SODV Relates to Public Documentation
Public documentation is a derived projection. Public documentation is not source truth.

## How SODV Relates to Mintlify
Mintlify may later publish derived official documentation. Mintlify is not canonical authority.

## How SODV Relates to NotebookLM
NotebookLM aligns corpus context. NotebookLM is not canonical authority.
NotebookLM must read authorization, correction, recovery, and completion records as one forward-only history. An earlier `not published` or `completion required` statement remains historical evidence and is not current posture after a later canonical completion record.

## How SODV Relates to Git History
Git history is version-control evidence, acting as a substrate for publication context.

## How SODV Relates to PR History
PR history is review and merge evidence.

## Safe-Use Rules
Do not treat SODV as a documentation generation engine. Treat it as a declarative governance boundary.
Do not treat release authorization as release completion.
Do not rewrite historical release records to make them read like the present. Append a forward record and use the latest applicable record for current-state answers.
Do not let the SODV engine create/move tags, upload artifacts, append canonical records, or infer public completion from a local cache.

## Non-Scope
SODV is not public documentation. SODV is not a docs site. SODV is not Mintlify. SODV is not NotebookLM. SODV is not a publication pipeline. SODV is not a generated documentation system yet. SODV is not a generated index yet. SODV is not a documentation template system yet. SODV is not a schema system. SODV is not qxctl. SODV is not symphony-validator. SODV is not SKVI. SODV is not SCLV. SODV is not SSCG. SODV does not replace canonical repository knowledge files. SODV does not replace module contracts. SODV does not replace tool contracts. SODV does not replace PR review. SODV does not create runtime behavior. SODV does not enforce runtime behavior.

## Non-Authorization Statement
This skill guides the bounded SODV proposal/projection engine but authorizes no canonical apply, tag creation, public publication, Mintlify configuration, NotebookLM automation, general publication pipeline, or release-completion claim.
