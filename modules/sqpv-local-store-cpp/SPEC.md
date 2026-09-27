# SQPV Local Store C++ 0.1.0-dev

## Selected contract

`sqpv-local-store-cpp` is a native C++26 library. Its exact dependencies are
SQMV metadata 0.1.0-dev, SQFV batch 0.2.0-dev and the knowledge-vector C++ foundation
0.2.0-dev. One store root binds exact canonical SQMV manifest bytes, all five SQFV
binding fields, one partition, one nonzero producer generation, one nonzero store
generation, one initial batch sequence and all resource limits. Reopen requires
those same caller-supplied values. A different representation, revision, scope,
generation or capacity uses a separate explicitly selected store.

The store generation is an immutable instance identity chosen by the caller. It
is separate from producer generation and exclusive writer ownership. It does not
rotate on reopen and does not independently fence a process. SHA-256 proves
integrity and exact matching; it authenticates no producer or access grant.
SQFV source-binding and source-position strings remain preserved acquisition
references, without an invented equality to SQMV evidence references.

## Platform and authority

The initial admitted target is macOS with local APFS. Runtime filesystem checks
reject nonlocal and non-APFS roots. The root must already exist, be owned by the
effective user and have mode 0700. Path components are opened relative to held
directory descriptors with no symbolic-link following. Ancestors must be owned
by this user or root and exclude group/other writes, except root-owned sticky
temporary directories. Relative paths, traversal and `/tmp` symlink aliases are
rejected; use the exact `/private/tmp` path for a disposable temporary store.

Files must be owned regular mode-0600 files with one link. Nonblocking opens allow
special files to be rejected before a read can block. Roots and files are private
caller-managed state, outside any concurrent external file editor, sync provider
or hostile same-user writer. Root and lock inode identity are rechecked before
operations and publication. Replacing either does not transfer an existing
handle's authority to the replacement.

A lifetime `flock(LOCK_EX | LOCK_NB)` supplies cooperative exclusive ownership.
Another handle returns `busy`; there is no forced takeover. Read, snapshot and
append calls on one handle are serialized. Handles reject use after fork before
locking an inherited C++ mutex. Moving or destroying a handle must not race its
operations. `reset` releases the handle and lock. A reader uses the owning Store;
this release has no separately concurrent read-only handle.

## Admission and finite capacity

Every limit is explicit, positive and finite. Technical ceilings for this release
are 64 MiB per encoded frame, 65,536 retained batches and 1 TiB of logical store
file bytes. The caller may choose smaller values. `max_store_bytes` covers every
module file, including metadata, lock, immutable frames, immutable commit records,
head and bounded staging. It is neither filesystem allocated-block accounting nor
a promise that the device has sufficient free space. Creation reserves a 4 KiB
control allowance beyond metadata; append preflights its peak staging footprint
before writing. Directory entry count is independently bounded. Exhaustion returns
`limit`, preserves existing committed data and does not overwrite history.

The complete recovery scan and receipt cache are bounded by `max_batches`; frame
read/encoding/hash work is bounded by `max_frame_bytes`. The SHA foundation may
copy a bounded hashing input. Recovery decodes one retained frame at a time in a
bounded private SQFV context; caller reads decode into the caller's own bounded
context. This slice supplies bounded behavior, without a throughput or latency
claim. No prune, delete, rotation, migration or reclamation API is present.

Appends accept precisely the next batch sequence. The first is `first_sequence`.
Earlier committed sequences accept only exact retry; changed content conflicts.
A higher sequence is a gap, and a lower unretained sequence is stale. Sequence
UINT64_MAX can be committed once and makes the stream exhausted without wrapping.
Contiguity is batch position in this exact stream; it proves neither source
completeness nor event-time completeness.

## Files and integrity format

All integers are unsigned 64-bit big endian. Generation fields are their exact
16 bytes; content identity is SQFV's exact 32 bytes. SHA-256 text fields are exactly
64 lowercase hexadecimal ASCII characters. Each metadata/commit/head document
ends in the SHA-256 text digest of all preceding bytes. The format magic supplies
the domain; no C++ object representation or padding is serialized.

- `writer.lock`: empty, permanent lock inode.
- `metadata`: magic `SQPS0001`; length-prefixed exact SQMV encoded bytes and
  partition; producer generation; store generation; first sequence; maximum frame
  bytes; maximum store bytes; maximum batches; digest. Each length is a u64.
- `frame-N`: the exact SQF1 byte frame emitted by SQFV 0.2.0-dev. `N` is the
  zero-padded 20-digit decimal batch sequence.
- `commit-N`: magic `SQPC0001`; SHA-256 of the complete metadata file; store and
  producer generations; batch sequence; frame byte count; SQFV content identity;
  SHA-256 of the complete frame; preceding commit digest (64 ASCII zeros for the
  first); own digest.
- `head`: magic `SQPH0001`; SHA-256 of the complete metadata file; committed batch
  count; last commit digest (64 ASCII zeros for zero batches); own digest.
- Recognized staging: `stage-metadata-H` (H binds the complete intended metadata), `stage-head`,
  `stage-commit`, and at most
  one `stage-frame-N-H`, where H is the intended complete frame's SHA-256 text.

Recovery rejects unexpected entries, malformed names, inappropriate file types,
oversize files, sequence holes, broken chains, missing referenced frames/commits,
corrupt frames and changed root metadata. The head detects a missing acknowledged
suffix instead of silently choosing a shorter surviving chain. Coordinated
external rollback of all store evidence is outside this local integrity contract;
no remote witness, trusted monotonic counter or anti-rollback authority is claimed.

## Commit and acknowledgement

After full admission and peak-capacity preflight, the synchronous operation:

1. Writes the exact frame to its input-specific stage name, synchronizes the file
   and reads the bytes back for equality.
2. Publishes the immutable frame with exclusive rename and synchronizes the root
   directory. An existing final frame is accepted only for exact retry.
3. Writes, synchronizes and verifies the deterministic commit record; publishes
   it with exclusive rename; synchronizes the directory.
4. Writes, synchronizes and verifies the next head; atomically replaces the head;
   synchronizes the directory; then returns the prepared receipt.

Only a complete verified commit record establishes a retained batch. A staged
frame alone never advances the committed range. The returned receipt binds the
exact store/producer generation, sequence, frame size and digest, content identity
and commit digest; the commit transitively binds exact metadata. It is local
retention evidence, separate from consumer processing or destination commit.
The original SQFV batch can remain live for delivery after this operation.

Ordinary failures preserve output arguments. `duplicate` is the documented
exception: it supplies the verified original receipt. Any failure once persistent
mutation may have begun returns `outcome_uncertain`, preserves outputs and poisons
the handle. Close and reopen to reconcile; never interpret that result as proof
that nothing committed. Repeated work on a poisoned handle returns `closed`.

## Recovery and staging

Open validates the entire bounded immutable chain and exact frames. A complete
verified commit beyond the prior head is reconciled by repeating file/directory
synchronization and publishing the corresponding head before open succeeds. This
covers interruption before acknowledgement. An incomplete frame remains explicitly
staged. Its filename binds the intended bytes; only the same complete input can
resume it. A surviving published frame without a commit likewise requires exact
retry. Recognized partial staging can be rewritten only when it is a prefix of the
same deterministic intended bytes. Committed files are never rewritten.

Creation interrupted before metadata publication can retry `create` with the same
metadata/options. After metadata publication, use `open`; an empty store lacking
its initial head can finish initialization. Missing head with retained commits is
a failure, not an invitation to infer acknowledged history. Staging cleanup is
bounded operation bookkeeping through file replacement/rename; no retained-data
deletion authority is implied.

## Proven guarantee and evidence limits

The selected guarantee is **local process-crash recovery with checked file and
directory fsync barriers**. Native tests kill subprocesses with SIGKILL at thirteen
initialization/append boundaries, then reconcile exact state and retry. Successful
calls require all specified synchronization calls to succeed on the admitted
filesystem. They also verify bytes before publication and again on recovery/read.

This release does not certify power-loss, kernel-crash, device-failure or replicated
durability. Apple documents that `fsync` can return while a drive still holds
buffered writes, and distinguishes `F_FULLFSYNC` for stronger device flushing.
This module does not substitute a power-loss claim for process-kill evidence.
See [Apple fsync](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/fsync.2.html)
and [Apple fcntl](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/fcntl.2.html).
A future stronger guarantee requires its own selected platform/device semantics
and verification. No generic durable-before-delivery composition or asynchronous
retention worker is admitted merely by this library.

## Verification and invariant ownership

Native tests cover exact stream admission, lost acknowledgement, all thirteen
SIGKILL boundaries, uncertainty poisoning, metadata/frame/commit/head corruption,
missing committed state, full capacity, sequence exhaustion, private-path checks,
exclusive locking and fork rejection while another thread holds the mutex.
Independent installed consumers use only installed public headers/archives.

Cross-owner invariants are registered in `knowledge/INVARIANT-OWNERSHIP.json`:
`sqpv.retained-commit-chain`, `sqpv.exact-stream-admission` and
`sqpv.exclusive-store-ownership` under the existing Symphony invariant family.
No `sqpv:` identity family, process-engine IPC surface or qxctl command is allocated.
