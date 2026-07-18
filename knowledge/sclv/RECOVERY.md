# SCLV Recovery and PR #59 Incident Record

## Outcome

PR #59 did not merge out of order and its SCLV record is not missing. Two independent issues made the closure appear unhealthy:

1. The validator treated GitHub pull-request numbers as a contiguous SCLV sequence and warned about every absent number.
2. qxctl and SSIAG recorded checksums for planned module versions that existed in a temporary local Go proxy but had not been published from immutable Git commits. The temporary proxy also omitted the repository-root `LICENSE` that canonical Go VCS packaging automatically includes for nested modules.

The first issue was a false positive. The second was a real independent-installation defect hidden by the authoring machine's module cache.

## Verified Timeline

- PR #59 was created at `2026-07-16T22:20:48Z` and merged at `2026-07-16T22:47:37Z` as `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`.
- Closure PR #60 was created at `2026-07-16T22:51:36Z` and merged at `2026-07-16T22:52:24Z`.
- PR #61 was not created until `2026-07-18T02:14:53Z`.

Therefore, concurrent authoring or a later ratification preceding PR #59 closure did not cause this incident. PR #60 is a closure carrier and does not recursively require an SCLV record. The same rule applies to PR #63.

## Module Evidence

The temporary proxy artifacts match the module subtrees at the PR #59 merge, but they are not canonical Go module archives because they omit the automatically inherited repository-root `LICENSE`:

- `github.com/QuanuX/Symphony/libraries/stav-protocol-go v0.2.0`
  - noncanonical temporary-proxy checksum: `h1:nC5yAA3CnaLzQoryEJTyU3SFDLYA2svVh3U57vNNjac=`
  - canonical VCS-produced checksum: `h1:DGVd771sqzeRpEkTUuuF+9TOK1JVQtyMh2GYR840g70=`
  - publication target: `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`
- `github.com/QuanuX/Symphony/modules/stav-append-authority v0.1.0`
  - noncanonical temporary-proxy checksum: `h1:r4IzwWGKj6llzWzy8IVzsRIW4QfNsy33slz3o7IimM0=`
  - canonical VCS-produced checksum: `h1:iijcegHcZ8EXfKJ8v/ToZWvBuf2y81UDWpAjj+g8OpI=`
  - publication target: `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`

The STAV protocol kernel has not changed since PR #59. The append authority changed materially in PR #64 when native supervision was completed. Publishing `v0.1.0` from the current tree would therefore be false history and would not match the recorded checksum.

## Executed Forward Recovery

SODV authorization bound each version before publication, and `SODV-REL-003` records completion:

- protocol kernel `v0.2.0` was published from the PR #59 merge commit;
- append authority `v0.1.0` was published from the PR #59 merge commit;
- supervised append authority `v0.2.0` was published from PR #64 merge commit `ed7484d70607aa96e64916dd4e59d3972a61980b`;
- consumer checksums were regenerated from canonical public-proxy artifacts using empty module caches;
- qxctl and SSIAG were updated to append-authority `v0.2.0` in PR #66;
- SCLV and SODV completion records preserve the real merges, tags, checksums, and clean-cache evidence.

The immutable tags remain correct because their source commits are correct. SODV records the checksum correction forward, consumers replace the invalid temporary-proxy sums, and no existing SCLV record, Git commit, tag, or historical release record is rewritten to simulate compliance.

The canonical packaging behavior is specified by the [Go Modules Reference](https://go.dev/ref/mod#module-zip-files). The public mirror also documents that a version requested before its tag exists may retain a negative result for up to 30 minutes at [proxy.golang.org](https://proxy.golang.org/).

## Local Cache Contamination Symptom

A workstation module cache may retain the old temporary-proxy kernel archive with checksum `h1:nC5yAA3CnaLzQoryEJTyU3SFDLYA2svVh3U57vNNjac=`. When a consumer correctly records the public checksum `h1:DGVd771sqzeRpEkTUuuF+9TOK1JVQtyMh2GYR840g70=`, Go rejects that cached archive with a checksum-mismatch security error before contacting the public proxy. This is correct fail-closed behavior and is not evidence that an immutable public tag moved.

Prove release state with a new isolated cache, explicit public proxy, enabled checksum database, and `GOWORK=off`:

```text
GOWORK=off \
GOMODCACHE=/new/empty/module-cache \
GOCACHE=/new/empty/build-cache \
GOPROXY=https://proxy.golang.org \
GOSUMDB=sum.golang.org \
go test ./...
```

If the empty-cache test passes with the canonical sum, quarantine or replace the contaminated local cache under an owner-controlled maintenance procedure. Do not disable `GOSUMDB`, edit `go.sum` back to the temporary checksum, move a tag, or use a warm cache as publication evidence.

## General Recovery Rule

An interrupted authoring session is represented only in `.git/symphony/sclv/pending/`. On the next session, tooling reconciles that marker with GitHub. A completed but unrecorded change receives a forward-only `late_recovery` record with actual start, completion, and recording timestamps. Once that record is committed, the ephemeral marker is removed. Canonical history retains the factual recovery explanation, but no active error state survives as permanent truth.
