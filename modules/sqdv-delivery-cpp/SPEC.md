# SQDV Delivery C++ Specification

## Exact identity and selected scope

`sqdv-delivery-cpp` `0.1.0-dev` is an SQDV-owned C++26 static library in namespace
`symphony::sqdv`, exposing [delivery.hpp](include/symphony/sqdv/delivery.hpp).
Its exact package and compatible compiler/runtime select a trusted same-process
source API. This release delivers complete SQFV batches under one immutable
SQMV binding. No projection, conversion, remote recipient, transport, destination
commit, access authorization, asynchronous retention worker, or background
catalogue is implied.

Two profiles are admitted: `Profile::disposable` (numeric 1), and
`Profile::retained_before_delivery` (numeric 2). Disposable delivery admits an
exact caller-supplied live batch without asserting retention. The retained
profile admits only actual committed SQPV evidence through its owned source.
It selects either the already-retained batch's live SQFV allocation or a decoded
retained read. `Origin::live` and `Origin::retained` distinguish these paths.
Both paths in the retained profile include actual local-retention evidence.

The retained profile inherits `sqpv-local-store-cpp` `0.1.0-dev`: macOS with local
APFS, explicit private roots, no concurrent external file edits, exact options,
cooperative exclusive store lock, bounded immutable history, and its stated
synchronization/recovery extent. It supplies no additional power-loss, remote
commit, anti-rollback, or universal exactly-once guarantee.

## Owned retained source and proof

`RetainedSource::create` and `open` invoke the actual SQPV operations; they do not
accept an arbitrary caller `Store`, receipt, committed-position assertion, or
snapshot as proof. The source's immutable identity includes exact absolute root
path, manifest reference, partition, producer generation, store generation,
first sequence, and all SQPV limits. Paths and source partition strings are
bounded to 4,096 bytes in this wrapper. SQPV validates the actual path and state.
The wrapper allocates all source identity/control state before the underlying
operation can mutate persistent state. If SQPV returns an uncertain outcome,
that status is preserved and no source handle is published.

Explicit `retain(out)` shares the source's actual locked Store. Sessions retain
that same Store state, so resetting an external source handle does not close the
store while a session still uses it. The source serializes its actual commit
and read operations. It neither duplicates a writer nor opens a second store
to give independent recipients their own cursors.

`commit(context, batch, out)` preallocates a proof and retains the actual batch
before calling `Store::append`. Only `ok` or verified `duplicate` publishes a
`RetainedBatch`. The proof contains the actual returned receipt, immutable
retained batch, and private source-instance identity. All wrapper operations
after successful append are nonallocating moves. A caller cannot create a proof
by supplying a public receipt. Proofs are move-only with explicit nonallocating
`retain`; they retain the batch and source identity, not the Store lock itself.

A newly opened source has a new private instance identity. Earlier proofs are
rejected by the new source's sessions even when their stable path/options match.
An exact duplicate commit can mint a new proof through the reopened actual
store. This conservative boundary avoids importing old in-memory evidence as a
new source-instance assertion.

## View identity and checkpoint

`Config` contains required nonempty `view_id`, `recipient_id`,
`recipient_interface`, and `partition`, a nonzero 16-byte producer generation,
an unsigned first sequence, and the selected profile. Strings are exact opaque
bytes (including NUL), at most 4,096 bytes each. The manifest supplies the exact
dataset/revision/schema/layout/access binding; the recipient interface string
identifies the caller-selected receiving contract without interpreting it.

Retained sessions require the exact source manifest binding, partition, and
producer generation. Their selected first sequence may be at or above the
source's first sequence. This permits later selected view starts without
inventing availability or completeness. Disposable sessions require no source.

The stable view reference is `sqdv1-sha256-` followed by 64 lowercase hexadecimal
SHA-256 digits. Its canonical input is the UTF-8 domain
`symphony.sqdv.delivery-view.v1` **including one trailing NUL**, then:

1. Profile as one unsigned byte.
2. Manifest reference, view ID, recipient ID, receiving-interface ID, and
   partition, in that order; each is unsigned 32-bit big-endian byte length
   followed by exact bytes.
3. The 16 producer-generation bytes and first sequence as unsigned 64-bit
   big endian.
4. A length-prefixed source identity, or a zero-length string for disposable.

The source identity bytes begin with
`symphony.sqdv.retained-source.v1` **including one trailing NUL**, then
length-prefixed absolute root path, manifest reference, and source partition;
16 producer-generation bytes; 16 store-generation bytes; unsigned 64-bit
big-endian first sequence, maximum frame bytes, and maximum store bytes; and
unsigned 32-bit big-endian maximum retained batches. The entire view input is
at most 65,536 bytes. There is no padding, Unicode normalization, filename
equivalence, identity issuance, or new Symphony colon namespace. Root relocation
and a change to any bound field require a distinct view reference.

Delivery byte credit and ledger capacity are operational allowances, not part of
the stable view identity; a resumed session can select different valid finite
allowances. SQPV source limits remain part of its exact store identity.

A `Checkpoint` contains this exact `view_reference`, `next_sequence`, and
`sequence_exhausted`. It reports the first not-yet-acknowledged processing
position, never the dispatch cursor. Resume rejects another view reference,
a sequence below the configured first sequence, or exhaustion with a value
other than UINT64_MAX. A checkpoint is a trusted caller-selected value, not a
signature or proof that its claimed earlier processing happened. The caller
owns checkpoint persistence. Loss of a newer checkpoint may cause replay;
no receiver-side durable deduplication is implemented here.

## Transfer, cutover, and bounded processing ledger

Session creation allocates an opaque session-instance token, fixed ledger, and
one real SQFV Port. Every offered batch must belong to the Port's actual SQFV
context. SQFV remains the owner of the only payload queue, immutable allocation,
lease lifetime, and per-port outstanding-byte credit. SQDV adds no payload queue
or hidden live cache.

`offer_live(batch)` is available only in disposable mode. Offers must match the
exact metadata binding, partition, generation, and next dispatch sequence.
SQFV verifies the actual context and latest exact duplicate/conflict state.

`offer_next(context, live_candidate)` is available only in retained mode. Any
provided candidate must be a proof from this actual source instance, match the
session binding, and belong to the supplied context. If its sequence equals the
next dispatch position, its live batch is offered. Otherwise the source reads
the **exact next position** from SQPV into the supplied context. No jump to the
newest available batch occurs. An absent sequence, corrupt history, invalid
candidate, exhausted resource, or failed read leaves both cursors unchanged.
A candidate from another context or source is rejected even when its sequence
would otherwise trigger fallback. An empty candidate is invalid; pass null to
select retained reading without a live candidate.

Before `Port::offer` can advance transfer state, SQDV allocates the delivery
control object, copies any receipt, and verifies an available ledger slot.
There are no fallible allocations after successful offer. `take(out)` transfers
the previously allocated handle and actual Port lease without a later fallible
allocation. The ring records only fixed sequence/content identity and delivery
state after take; it retains no Batch, RetainedBatch, or payload allocation.

Dispatch and processing cursors are separate. Taking a delivery advances
neither cursor: dispatch advanced at successful offer, and processing advances
only through explicit acknowledgement. A session may dispatch several batches
within both byte-credit and unacknowledged-count limits. Independent sessions
have independent Ports, cursors, credits, and ledgers while sharing the actual
retained source.

## Processing acknowledgement and payload release

Every `Delivery` contains a private token for the exact Session instance.
`acknowledge_processed` accepts only an actually taken delivery from that
instance. An older session's ticket cannot advance a reopened session even when
the stable view reference matches. A ticket from another session returns
`binding_mismatch`; an empty ticket is `invalid_argument`.

Acknowledgements are in order. A later taken sequence returns `gap`; the latest
exact repeated acknowledgement is `duplicate`; older acknowledged tickets are
`stale`. Successful acknowledgement frees one processing-ledger entry and
advances only the contiguous processed cursor. UINT64_MAX is admitted once and
marks the corresponding dispatch or processed cursor exhausted without wrapping.
There is no silent acknowledgement on take, payload release, cancellation,
destruction, or disconnect.

`release_payload()` returns the actual SQFV lease credit and ends payload and
descriptor access while retaining the acknowledgement ticket. Conversely,
acknowledging processing leaves a live lease and its byte credit outstanding.
A released but unacknowledged delivery still occupies its ledger entry. A held
payload can block further offers after its processing acknowledgement if byte
credit is exhausted. `reset()` releases payload and discards the local ticket;
an unacknowledged entry remains in the Session. The caller can preserve the
processed checkpoint, reset that session, and resume to replay unresolved work.

Session reset cancels its pending Port deliveries and ends new session work.
Already taken Delivery leases remain valid through SQFV until released, even
after the source and session handles close. A processing acknowledgement reports
the trusted caller's completed-processing assertion. It establishes neither
destination commit nor source rights or recipient authorization.

## Failure ordering and concurrency

All fallible methods are `noexcept` and preserve output parameters on failure.
`RetainedSource::commit` treats verified `duplicate` as success evidence and
publishes its actual receipt/proof. Allocation failure before a state transition
is `no_memory`. SQPV statuses including `busy`, missing/corrupt history, path
failure, limit, and `outcome_uncertain` remain explicit.

An uncertain append closes its shared source to new commits, source retention,
new sessions, offers, and takes. Existing sessions can still export checkpoint
and statistics, and acknowledge already-issued deliveries; existing payloads
can still be released. Uncertainty never rewinds or advances their cursors or
retroactively invalidates earlier processing evidence. Close all source/session
handles to release the actual store lock, reopen the same root/options for
SQPV recovery, then resume a new session. A proof handle alone does not prevent
reopening.

Calls on each Session and actual source are serialized. A method checks the
creating process ID before acquiring its mutex and returns `stale` on inherited
use after fork. Underlying SQFV handles do not provide post-fork cleanup safety;
an inherited child must avoid using or destroying these inherited graphs and
use an exec or immediate process-exit boundary. Fresh child work needs fresh
owner contexts. Move, reset, or destruction must not race calls on the same
handle. Output replacement occurs after releasing the owning source/session
mutex, so replacing an old output cannot reenter a held owner mutex.

## Finite control state and ownership

`outstanding_byte_credit` is positive and at most **64 MiB**.
`max_unacknowledged_batches` is positive and at most **65,536**; it also sets
the Port's pending-entry allowance. Creation preallocates exactly that many
fixed ledger slots. Each pending slot may own one bounded delivery control
object and, in retained mode, the actual fixed-shape SQPV receipt (two 64-byte
hex digests, fixed generations, sequence, size, and content ID). There is no
unbounded acknowledgement history: only the latest acknowledged identity is
kept for duplicate classification.

SQDV's bounded strings, identity hashing scratch, fixed ledger, delivery control
objects, proof objects, and source/session bookkeeping are SQDV/caller control
state. They are not claimed as charged to SQFV's allocation ledger. SQFV owns
and accounts actual batches, Ports, queues, and leases. RetainedBatch values keep
their explicit batch obligation under that context; delivery and acknowledgement
state do not retain an extra Batch. Callers control aggregate numbers of sources,
sessions, retained proofs, taken delivery handles, and checkpoint values. This
release supplies per-session bounds, not an undisclosed process-global budget.

## Focused acceptance extent

Native and independently installed consumers exercise exact identity rejection,
live/retained continuity, lost-acknowledgement replay, session-token isolation,
separate ledger and byte credit, retained-source ownership, terminal sequences,
uncertain retention outcomes, output preservation, and bounded allocation
failure. These development checks do not establish a supported-platform matrix,
provider fidelity, recipient rights, production throughput, remote delivery, or
completion of every SQDV mode in its architectural Quad.
