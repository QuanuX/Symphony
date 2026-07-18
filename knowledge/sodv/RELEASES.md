# SODV Release Publication Ledger

## Purpose

This append-only ledger binds a module version to an immutable Git commit before publication and records completion only after the public artifact is independently resolvable. It is publication truth, not a package proxy, tag generator, or substitute for SCLV change truth.

## Transaction Model

A release transaction has two immutable records:

1. `authorization` names the exact module path, semantic version, source commit, source PR, expected evidence, and completion gates. No tag may be pushed before this record is ratified and merged.
2. `completion` records the actual tag object, public module checksum, clean-cache verification, and consumer consequence. It is appended after publication.

There is no canonical `pending` release record. Local preparation state belongs under `.git/symphony/releases/pending/`. If a session is interrupted, the next session reconciles actual tags and proxy state against the merged authorization. It either completes the authorized transaction, records a forward recovery, or fails closed for human review. It never moves an existing tag or edits an earlier record.

## Release Records

- release_record_id: `SODV-REL-001`
- record_version: `1`
- record_type: `authorization`
- status: `authorized`
- disposition: `recovery_authorization`
- authorized_at: `2026-07-18T07:07:20Z`
- authorized_by: `Architect`
- purpose: |
    Recover PR #59's coordinated Go-module publication without manufacturing compliance. The first two versions are bound to the exact PR #59 merge tree that produced their recorded temporary-proxy checksums. The supervised append-authority increment receives a new version bound to the PR #64 merge tree.
- publication_units:
  - module_path: `github.com/QuanuX/Symphony/libraries/stav-protocol-go`
    version: `v0.2.0`
    tag: `libraries/stav-protocol-go/v0.2.0`
    source_commit: `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`
    source_pr: `https://github.com/QuanuX/Symphony/pull/59`
    expected_go_sum: `h1:nC5yAA3CnaLzQoryEJTyU3SFDLYA2svVh3U57vNNjac=`
  - module_path: `github.com/QuanuX/Symphony/modules/stav-append-authority`
    version: `v0.1.0`
    tag: `modules/stav-append-authority/v0.1.0`
    source_commit: `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`
    source_pr: `https://github.com/QuanuX/Symphony/pull/59`
    expected_go_sum: `h1:r4IzwWGKj6llzWzy8IVzsRIW4QfNsy33slz3o7IimM0=`
  - module_path: `github.com/QuanuX/Symphony/modules/stav-append-authority`
    version: `v0.2.0`
    tag: `modules/stav-append-authority/v0.2.0`
    source_commit: `ed7484d70607aa96e64916dd4e59d3972a61980b`
    source_pr: `https://github.com/QuanuX/Symphony/pull/64`
    expected_go_sum: `to_be_recorded_from_public_proxy`
- preconditions:
  - `authorization merged before tag publication`
  - `authorized tags do not already resolve to different objects`
  - `module subtree at each source commit is complete and testable`
  - `PR #59 temporary-proxy archives match the two v0.1/v0.2 source subtrees exactly`
- completion_requirements:
  - `create annotated tags at the authorized commits without moving or replacing any existing tag`
  - `push only the three authorized tags`
  - `resolve every module version through the public Go proxy using an empty module cache and GOWORK=off`
  - `verify the first two public checksums equal their expected checksums`
  - `record the v0.2.0 append-authority public checksum`
  - `update qxctl and SSIAG to append-authority v0.2.0 and regenerate go.sum from public artifacts`
  - `append a separate SODV completion record and SCLV closure record using real evidence`
- non_authorizations:
  - `moving or replacing an existing tag`
  - `publishing any version from the current working tree by convenience`
  - `editing PR #59 history or its SCLV record`
  - `treating a populated local Go cache as independent-install evidence`
  - `Go 1.27 adoption`
  - `public documentation, SDK, OpenAPI, Mintlify, or release-binary publication`
- evidence:
  - `knowledge/sclv/RECOVERY.md`
  - `knowledge/sclv/CHANGELOG.md#sclv-pr-059`
  - `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`
  - `ed7484d70607aa96e64916dd4e59d3972a61980b`

## Completion Records

No completion record exists yet. Authorization is not evidence that a tag or public module is available.
