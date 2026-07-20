# SODV Release Publication Ledger

## Purpose

This append-only ledger binds a module version to an immutable Git commit before publication and records completion only after the public artifact is independently resolvable. It is publication truth, not a package proxy, tag generator, or substitute for SCLV change truth.

## Transaction Model

A release transaction has two immutable records:

1. `authorization` names the exact module path, semantic version, source commit, source PR, expected evidence, and completion gates. No tag may be pushed before this record is ratified and merged.
2. `completion` records the actual tag object, public module checksum, clean-cache verification, and consumer consequence. It is appended after publication.

There is no canonical `pending` release record. Local preparation state belongs under `.git/symphony/releases/pending/`. If a session is interrupted, the next session reconciles actual tags and proxy state against the merged authorization. It either completes the authorized transaction, records a forward recovery, or fails closed for review by a caller holding the applicable release permission. It never moves an existing tag or edits an earlier record.

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

## Correction Records

- release_record_id: `SODV-REL-002`
- record_version: `1`
- record_type: `authorization_correction`
- status: `canonical`
- disposition: `forward_recovery`
- corrects: `SODV-REL-001 checksum expectations only`
- discovered_at: `2026-07-18T07:21:27Z`
- ratified_by: `Architect`
- finding: |
    The exact temporary-proxy archives used during PR #59 contained the complete nested module subtrees, but they omitted the repository-root `LICENSE`. Canonical Go VCS module packaging automatically adds that file to nested modules. The immutable source commits and published tag targets are correct; the two temporary-proxy content checksums in SODV-REL-001 are not canonical release checksums and cannot be satisfied without publishing a noncanonical archive.
- corrected_publication_units:
  - module_path: `github.com/QuanuX/Symphony/libraries/stav-protocol-go`
    version: `v0.2.0`
    tag_object: `f1274b6971941f8b60f991eb9b4422cc15703bb3`
    source_commit: `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`
    canonical_go_sum: `h1:DGVd771sqzeRpEkTUuuF+9TOK1JVQtyMh2GYR840g70=`
    go_mod_sum: `h1:kYeJSvzp7ezK+0CJzHD4v2euyRqXuAfXocYxRACrxoM=`
  - module_path: `github.com/QuanuX/Symphony/modules/stav-append-authority`
    version: `v0.1.0`
    tag_object: `dfa637080cf7e3b21cdd0b7e45fd5b0010a7fd5f`
    source_commit: `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`
    canonical_go_sum: `h1:iijcegHcZ8EXfKJ8v/ToZWvBuf2y81UDWpAjj+g8OpI=`
    go_mod_sum: `h1:pRWSy0nSQu5dYtiKpvTEmYTFrgf1O0bAqtmU3MDowlc=`
  - module_path: `github.com/QuanuX/Symphony/modules/stav-append-authority`
    version: `v0.2.0`
    tag_object: `aeb61f13c7e306a45818cde972307209d070dc28`
    source_commit: `ed7484d70607aa96e64916dd4e59d3972a61980b`
    canonical_go_sum: `h1:DvWWrt7MbJFfEA/ROnTCDJYwoVWRgXSzy6IkTEpkMPI=`
    go_mod_sum: `h1:pRWSy0nSQu5dYtiKpvTEmYTFrgf1O0bAqtmU3MDowlc=`
- unchanged_authorization: |
    SODV-REL-001 remains authoritative for module paths, semantic versions, tag names, exact source commits, and non-authorizations. This correction supersedes only its temporary-proxy checksum expectations and the requirement that public checksums equal those noncanonical values.
- completion_gate: |
    Completion still requires public-proxy and checksum-database propagation, empty-cache `GOWORK=off` resolution, consumer regeneration against append-authority v0.2.0, and a separate immutable completion record.
- non_authorizations:
  - `moving or replacing any published tag`
  - `constructing an archive that omits canonical Go packaging behavior`
  - `editing SODV-REL-001 or PR #59 history`
  - `claiming public-proxy completion before it is observed`
- evidence:
  - `knowledge/sclv/RECOVERY.md`
  - `git VCS Origin metadata for all three tagged versions`
  - `archive comparison showing LICENSE as the only temporary-proxy/VCS difference`
  - `https://go.dev/ref/mod#module-zip-files`

## Completion Records

- release_record_id: `SODV-REL-003`
- record_version: `1`
- record_type: `completion`
- status: `completed`
- disposition: `verified_forward_completion`
- completes: `SODV-REL-001 as corrected by SODV-REL-002`
- completed_at: `2026-07-18T07:33:12Z`
- completed_by: `Architect-directed release procedure`
- publication_units:
  - module_path: `github.com/QuanuX/Symphony/libraries/stav-protocol-go`
    version: `v0.2.0`
    tag: `libraries/stav-protocol-go/v0.2.0`
    tag_object: `f1274b6971941f8b60f991eb9b4422cc15703bb3`
    source_commit: `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`
    public_go_sum: `h1:DGVd771sqzeRpEkTUuuF+9TOK1JVQtyMh2GYR840g70=`
    public_go_mod_sum: `h1:kYeJSvzp7ezK+0CJzHD4v2euyRqXuAfXocYxRACrxoM=`
  - module_path: `github.com/QuanuX/Symphony/modules/stav-append-authority`
    version: `v0.1.0`
    tag: `modules/stav-append-authority/v0.1.0`
    tag_object: `dfa637080cf7e3b21cdd0b7e45fd5b0010a7fd5f`
    source_commit: `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`
    public_go_sum: `h1:iijcegHcZ8EXfKJ8v/ToZWvBuf2y81UDWpAjj+g8OpI=`
    public_go_mod_sum: `h1:pRWSy0nSQu5dYtiKpvTEmYTFrgf1O0bAqtmU3MDowlc=`
  - module_path: `github.com/QuanuX/Symphony/modules/stav-append-authority`
    version: `v0.2.0`
    tag: `modules/stav-append-authority/v0.2.0`
    tag_object: `aeb61f13c7e306a45818cde972307209d070dc28`
    source_commit: `ed7484d70607aa96e64916dd4e59d3972a61980b`
    public_go_sum: `h1:DvWWrt7MbJFfEA/ROnTCDJYwoVWRgXSzy6IkTEpkMPI=`
    public_go_mod_sum: `h1:pRWSy0nSQu5dYtiKpvTEmYTFrgf1O0bAqtmU3MDowlc=`
- external_verification: |
    Each version resolved through `https://proxy.golang.org` and `sum.golang.org` with `GOWORK=off` and a distinct empty `GOMODCACHE`. The public checksums match SODV-REL-002. The pre-tag negative-cache 404 cleared without moving or recreating a tag.
- consumer_completion: |
    qxctl and SSIAG now require supervised append-authority v0.2.0. qxctl, SSIAG, and append-authority checksums were regenerated from canonical release artifacts. The root workspace replacement aligns to v0.2.0. All four Go module suites passed independently through the public proxy with `GOWORK=off` and empty caches.
- evidence:
  - `https://github.com/QuanuX/Symphony/pull/66`
  - `proxy.golang.org download metadata for all three versions`
  - `sum.golang.org authentication for all three versions`
  - `clean-cache public-proxy go test ./... for libraries/stav-protocol-go`
  - `clean-cache public-proxy go test ./... for modules/stav-append-authority`
  - `clean-cache public-proxy go test ./... for modules/secure-identity-access-governance`
  - `clean-cache public-proxy go test ./... for tools/qxctl`
- non_authorizations:
  - `moving or replacing any published tag`
  - `binary, container, SDK, OpenAPI, Mintlify, or public documentation release`
  - `Go 1.27 production pin`
  - `new SSIAG, STAV, qxctl, provider, or trading-node authority`
- notes: |
    This closes the active PR #59 module-publication error. The historical temporary-proxy discrepancy remains documented as resolved evidence; no unresolved flag or mutable error state persists in the canonical release record.
