# SQAV Databento DBN Specification

## Exact implemented scope

`sqav-databento-dbn-cpp` `0.5.0-dev` exports the C++26 static-library target
`Symphony::SqavDatabentoDbn`, namespace `symphony::sqav::databento`. Runtime
requires SQAV capture 0.2.0-dev, SQMV 0.2.0-dev, SQFV 0.3.0-dev, SQPV 0.2.0-dev, libcurl 8.7.1 SDK and the shared
foundation 0.2.0-dev. Initial verified platform: macOS amd64 / AppleClang 21.

This release admits **uncompressed DBN versions 1 and 3, single-schema MBO files**.
It parses the provider layout directly using explicit little-endian loads;
unaligned input is supported without casting bytes to native structs. Databento
C++ `v0.68.0` declarations and public fixtures anchor conformance. The SDK is a
source/fixture reference, not a runtime dependency or network-support claim.
Provider documentation reviewed 27 September 2026:

- https://databento.com/docs/standards-and-conventions/databento-binary-encoding
- https://databento.com/docs/schemas-and-data-formats/mbo
- https://github.com/databento/databento-cpp/blob/v0.68.0/include/databento/record.hpp
- https://github.com/databento/databento-cpp/blob/v0.68.0/tests/src/dbn_decoder_tests.cpp

No older/newer version is silently upgraded. DBZ/Zstandard, mixed live schema,
other record types and self-describing schema definitions remain outside this
release. The optional historical HTTP executor is specified below. Full-book data means MBO events, not a reconstructed or complete book.

## Bounded structural inspection

`FileView::inspect` accepts immutable borrowed bytes and mandatory positive Limits:
file bytes at most 64 MiB, metadata bytes including the 8-byte prefix at most
1 MiB, and records at most 1,048,576. It returns a borrowed view only after the
complete file validates. Failure preserves the prior view. It allocates no memory.
The caller keeps storage alive and unchanged during every view/field access;
mutation, move, destruction or concurrent writes invalidate that contract.

The prefix must contain DBN magic, version 1 or 3 and a bounded metadata length.
Total metadata is at least 128 bytes. Version 3 metadata must be 8-byte aligned;
version 1 metadata need not be aligned. DBNv1 keeps the legacy eight reserved
bytes at offset 50, reads symbology/ts_out at offsets 58/59/60, and uses fixed
22-byte symbols. DBNv3 reads offsets 50/51/52 and its declared symbol width.
Both versions start the variable metadata grammar at offset 108. Dataset is a nonempty
NUL-terminated 16-byte field; schema is MBO (0). ts_out must be 0 or 1. Declared
symbol width is positive and at most 4,096 bytes. The future schema-definition
length must be zero. Symbol/partial/not-found lists and mapping/interval arrays
are traversed with division-based remaining-byte checks before loops. Fixed
strings must contain a NUL. Mapping dates, symbol bytes, symbology enums and
reserved padding remain uninterpreted and preserved. DBNv3 final alignment padding may
occupy at most seven bytes; DBNv1 must end exactly after its mapping array. This is structural parsing, not date/symbol semantic
resolution or validation of every reserved byte.

Metadata exposes original wire version, dataset, raw start/end/limit, input/output symbology bytes,
optional-gateway-timestamp flag, symbol width and list/mapping counts. The encoded
metadata view retains every mapping interval and original byte. Record count is
computed from the actual body, separately from the request's maximum-record limit.
A metadata-only file is valid and has zero records; EOF at a complete record
boundary may be valid even if the original file had further records. No checksum
or expected length in DBN proves complete transport. Capture adds an exact
integrity identity to whichever complete bytes the caller supplies.

## MBO fields and time semantics

Every record is exactly 56 bytes, or 64 with appended ts_out, with matching
length-in-32-bit-words and rtype 160. Mixed/control records are unsupported.
`record(index)` rejects out-of-range indices. `decode_mbo` validates one record
under the same DBNv1/v3 layout. Both preserve their output on failure and allocate
nothing.

Exposed fields preserve publisher/instrument/order IDs, event and receive
nanoseconds, signed integer price, quantity, flags, channel, action, side, signed
input timestamp delta, venue sequence and optional gateway timestamp. Price
remains fixed-point units of 1e-9. Unknown action/side values and flag bits remain
raw bytes; they are not interpreted as supported book operations. Undefined
price/time sentinels remain exact integer values. No floating-point conversion,
time subtraction, rescaling, event sorting, sequence-gap inference or snapshot
completion inference occurs. Venue sequence and gateway/event/receive timestamps
remain distinct from Symphony transfer sequence and acquisition time.

## Owned capture bridge

`capture_file` fully inspects input first. Source provider_ref must be `databento`,
dataset_id must exactly match file metadata, native_schema_ref must be
`databento:mbo`, and native_encoding_ref must be
`databento:dbn-v1-uncompressed` for version 1 or
`databento:dbn-v3-uncompressed` for version 3. `encoding_for_version` returns the
exact encoding reference, or an empty string for unsupported versions. The
existing `native_encoding` constant still denotes v3; `native_encoding_v1` is
explicit. Changing only a file version byte is not conversion. A supplied source_record_count must equal the
computed count. A complete-coverage assertion conflicts with nonzero partial or
not-found symbol counts and is rejected. Other coverage statements remain caller
assertions; absence of those lists does not prove completeness.

The adapter sets source.adapter_ref=`sqav-databento-dbn-cpp`, adapter_version=
`0.5.0-dev` and source_record_count to the actual count. Other source fields,
acquisition evidence, coverage and access scope retain their caller attribution.
Caller evidence string sizes and time-vector count are bounded before copying;
SQAV's own exact validity and encoded-size rules then apply. Its original payload
is the **entire unchanged DBN file**, including metadata and mapping bytes. All
allocation failures preserve the previous Capture. `inspect_capture` verifies
that exact provider/adapter/schema/dataset/count binding before returning a view
borrowed from the live Capture. The SQAV/SQMV bridge still carries one capture
envelope per transfer record; provider record count is a separate inner fact.

## Package and evidence

Receipt-v2 owns 16 files: archive, four headers, four CMake exports/configuration files,
six documents and license. No executable, operational SSIAG credential resolver, service or SQV
qxctl command is installed. Native tests additionally compose SQPV/SQDV. Public
fixture fields are checked against upstream expected values; truncation, bounds,
raw extrema, unaligned inputs, optional ts_out, zero-allocation inspection,
allocation rollback and retained replay have focused evidence. The independently
installed consumer exercises the same public fixture through exported C++26.

No operational SSIAG authentication, entitlement, source authenticity,
redistribution, live reconnect, symbol resolution, book state, reference-data
client, broad provider conformance or performance guarantee follows from parsing.

## Version separation and private sample acceptance

The 0.2.0-dev metadata API introduced DBNv1/v3 support. The 0.3.0-dev
package selected the SQV-20 dependency chain. The 0.4.0-dev added the
explicit historical planning and response contract below.
The 0.1.0-dev v3-only package can coexist in its immutable prefix. Its captures
are not silently relabeled as 0.2 captures. DBNv2 and future versions remain
unsupported. Public v1/v3 fixtures describe identical MBO events; focused tests
check field equality, version-specific grammar, encoding mismatches, legacy
reserved bytes, optional timestamps, atomic rejection and allocation-free reads.

The explicit test-only `--sample FILE DATASET` path checks the selected SQV-13
private 100,000-record regular-session samples, every exposed field digest, and
exact original-byte SQAV/SQMV/SQFV/SQPV/SQDV retained replay. It performs no
network activity and embeds no private records or credentials. Reaching the
request record cap is reported as partial coverage of the requested time window.

## Coordinated local pipeline admission

This development package admits the exact SQV-20 dependency chain. The native
representation/codec rules remain as specified above. The six-owner offline
pipeline verifies attributable capture, conversion, preview, asynchronous local
retention and confirmed replay; this is not a new provider or transport claim.

## Historical request and response profile (0.4.0-dev)

`historical.hpp` adds transport-independent planning, response assembly and
coverage/recovery classification to the same provider adapter. The historical
endpoint is fixed at `https://hist.databento.com/v0/timeseries.get_range`.
`HistoricalPlan::parameters()` is a deterministic percent-encoded parameter
string for explicit raw symbols, MBO, instrument-id output, uncompressed DBN,
and a finite record limit. Symbols are sorted and duplicates rejected; commas,
controls, non-ASCII bytes, empty symbols and `ALL_SYMBOLS` are outside this local
profile. Other printable characters are percent-escaped. Dataset codes are
1–15 uppercase ASCII letters, digits or dots. Symbols are at most 70 bytes.
This does not establish exchange symbology validity, trading-session membership,
entitlement, cost or scheduling authority.

All limits are mandatory: DBN file/metadata/record bounds above; at most 128
symbols, an explicit maximum window no longer than 24 hours, 1–8 attempts and a
maximum retry delay of 1–86,400 seconds. Positive record limits cannot exceed the
DBN record bound. Start and end are nonnegative signed-64-bit-range Unix
nanoseconds with `start < end`; the window filters MBO receive time as
`[start,end)`. `sqdh1-sha256-…` identifies the exact endpoint and canonical
parameters. Execution limits are local policy and do not change that selection
identity. No auth field, header or credential is part of this plan. No request,
retry, timer or billable operation is executed by this library.

`HistoricalResponse::begin` retains immutable plan ownership and a caller-supplied
attempt ordinal, final HTTP status (or zero when unavailable) and optional parsed
Retry-After seconds. Status zero is never success; redirects are not followed.
The transport caller is responsible for matching the response to its actual
request, parsing HTTP framing/compression, honoring deadlines and preserving
these observations. Raw headers are not accepted. Non-200 bodies are counted
against the same byte limit and discarded. For 200, append copies unchanged
bytes, checks remaining capacity before addition, rejects self-aliasing input,
and bounds requested vector capacity to the selected file ceiling. Allocation
failure preserves accepted bytes, so the caller may retry the same chunk.
Allocator overhead is not part of the byte-capacity accounting. Oversize input
sets a terminal overflow flag; no further chunk is admitted. Per-response state
is single-thread owned; independent plans can share immutable state.

`finish` is one-shot and allocation-free. Its return status says whether the
classification succeeded; `HistoricalReport` carries the outcome and retains the
exact immutable request/policy even after the response is destroyed. Transport
interruption, cancellation, byte exhaustion, HTTP failure, malformed DBN or a
binding mismatch cannot produce complete coverage or a capture through this
API. A structurally valid prefix at an exact record boundary is insufficient.
Only transport-complete HTTP 200 is checked as DBN. Binding checks require exact
dataset, start, end, record limit, raw-symbol input, instrument-id output, no
appended gateway timestamp, and the exact unique requested symbol set (any
metadata ordering). Every record's receive time must fall within the requested
window and be nondecreasing; equal times are admitted. This is metadata/record
conformance, not source authentication or proof of exchange completeness.

For a bound response, reaching the record cap or any partial/not-found symbol
list yields partial coverage. Below-cap, transport-complete responses with no
unresolved lists are complete **for this observed request window**. A valid
empty response can establish that same scoped result; it does not imply a valid
book. Reported first/last receive times are observations, never resume cursors.
The transport's completion assertion remains attributed to the caller.

Interruption (status zero, 200 or a retryable HTTP status), HTTP 429, 502, 503 or
504 can recommend repeating the **entire original window**, subject to the
caller-supplied attempt ordinal and retry-delay policy. Other HTTP errors require
review. Cancellation recommends no automatic recovery. Retry-After is preserved,
never shortened; a value above policy requires review. These are recommendations,
not a persistent attempt/spending ledger; callers must count attempts and admit
each actual billable operation separately. Generic HTTP 500 is left for review.

A cap-reached response without unresolved symbols recommends splitting the
window. `HistoricalPlan::split` produces adjacent `[start,mid)` / `[mid,end)`
plans, preserving symbols and limits. Both halves must be reacquired, and neither
is presumed complete. It fails atomically for a one-nanosecond window or aliased
outputs; one output may replace the original plan. Same-timestamp density can
require a different cap/selection and explicit review. Advancing to last-time+1,
concatenating retries as new data, byte-range resume and automatic deduplication
are not implemented.

`capture` accepts only a finished, bound complete/partial response and explicit
observer, attempt, access, revision and acquisition attribution. It constructs
source identity, exact version-specific encoding, actual record count and
coverage internally. The capture retains the canonical request in coverage_scope
and bounded HTTP/transport/attempt/byte/count/cap/unresolved facts in
source_position, with a digest reference in coverage_evidence_ref. All original
DBN bytes, including mappings, remain unchanged. Existing SQAV field/total limits
still apply: a valid many-symbol plan may exceed a selected capture field limit
and be refused without altering the previous capture. This is explicit evidence,
not a durable retry journal. The new package's adapter version changes capture
identity; older captures are not relabeled.

Focused evidence covers v1/v3 fixtures, synthetic complete/empty/unresolved
responses, byte-sized transport chunks, cap and same-timestamp behavior,
interruption at valid EOF, binding failures, error-body discard, budgets,
allocation rollback, and capture through asynchronous preview, drain, close,
reopen and exact retained replay. Installed consumers independently check the
public boundary. Version 0.5.0-dev supplies the HTTP executor specified below.
An operational SSIAG bridge and deployment remain required for local-key-ring
retrieval; the response interface supplies no alternative credential path.

Protocol sources reviewed 28 September 2026: Databento historical API documentation
at https://databento.com/docs/api-reference-historical/metadata/metadata-get-cost
(the combined page includes `timeseries.get_range`), and the unchanged v0.68.0
DBN declarations/fixtures linked above. HTTP parameters also match the bounded
SQV-13 experimental request evidence. No provider SDK version was upgraded.

## Persistent attempt and spending admission (0.5.0-dev)

`attempts.hpp` adds an SQPV-backed AttemptLedger for one immutable budget root.
Ceiling and prior charges use unsigned integer nanodollars (10^-9 USD), with no
floating-point rounding. Budget, prior charge, generation, entry limit and disk
budget are immutable. At most 4,096 attempts and two records per attempt are
admitted; each canonical request parameter string is bounded by the planner.
The ledger's retained metadata/state is finite under those ceilings. These are
local profile limits, not account-wide authority across arbitrary other roots.

A caller-verified quote supplies a nonsecret evidence reference, conservative
charge ceiling and explicit quote/expiry times. Quote age spans at most one day;
the trusted runtime supplies current time. The ledger neither obtains quotes
nor proves the provider's eventual invoice amount. It enforces the sum of the
**supplied charge ceilings**, including prior charges. A provider estimate must
not be represented as an authoritative upper bound without supporting evidence.
Operational collection still needs that quote/account-budget integration.

Reserve writes the exact request reference/parameters, unique attempt ID,
quote/times, cost ceiling and per-request attempt ordinal before returning a
move-only execution ticket. Checked subtraction rejects overflow/overspend.
Duplicate IDs never mint a new ticket. Per-plan and ledger attempt bounds both
apply. Ticket claim is one-shot, exact-request-bound, expires with the quote,
and requires its original live ledger/process. Reset or restart invalidates old
tickets. Claims serialize under the ledger mutex, including concurrent callers.

Terminal outcomes are completed, rejected, cancelled or indeterminate. Repeating
an identical terminal outcome is idempotent; conflicting outcomes are refused.
Completion requires a prior claim. No outcome automatically refunds its reserved
ceiling. Reopen revalidates the complete record history, request digests, ordinals,
unique IDs, terminal transitions and total accounting. A reservation without a
terminal record recovers as indeterminate and retains its charge; restart never
reissues execution capability. Uncertain writes close the handle until reopen.
These rules prevent this ledger from blindly repeating an uncertain paid attempt;
they do not deduplicate provider records or automatically orchestrate backfill.

## Native bounded HTTP execution (0.5.0-dev)

`http.hpp` adds one historical HTTP attempt using the selected libcurl 8.7.1 SDK
and system runtime. This is a C++26 static-library surface. The production endpoint
is fixed to the v0 historical timeseries URL. HTTPS protocol restriction, peer and
hostname verification, TLS 1.2 minimum, redirects disabled, proxy disabled, netrc
ignored, verbose diagnostics disabled and exact POST parameters are configured
explicitly. No ambient API key, cookie file, command argument or environment
credential is consumed. Test-only HTTP loopback endpoint substitution is absent
from the installed archive.

Execution claims an existing durable AttemptTicket before invoking a supplied
`SsiagHistoricalUse` native bridge. That bridge must authenticate and pin the
actual SSIAG request/recipient/lease, enforce its deadline/cancellation, and
release/clean up after the borrowed key use. This package ships **no operational
implementation of that bridge**. A fixture callback is not SSIAG authentication
and cannot establish deployment readiness. Installation starts no collector.
The selected Databento credential format is exactly 32 ASCII bytes, db- followed
by alphanumerics. The borrowed key is used only in the native HTTPS sink. Its
local copy is overwritten; libcurl/framework/kernel copies are not claimed
perfectly erased. No secret or native diagnostic is included in results.

The sink admits one key use and one HTTP operation. A provider return value
cannot turn a sink timeout, truncation or callback failure into success. Total
HTTP/credential-use allowance is positive and at most 300,000 ms, with a positive
connect allowance no larger than total. The provider bridge must honor the same
bounded-use contract; the library cannot preempt an arbitrary blocking callback.
Stop tokens cancel before execution or through the transfer progress callback.
Cancellation is cooperative and inherits libcurl callback scheduling/DNS limits;
no submillisecond cancellation latency is promised.

All received header blocks count against a mandatory ceiling of at most 65,536
bytes. Strict status lines, header framing and encoding admission precede body
assembly. Numeric Retry-After seconds are preserved; HTTP-date or invalid delta
values are explicitly uninterpreted, with recovery requiring review rather than
an invented/shortened delay. Repeated Retry-After, unsupported content encoding
and trailers are refused in this profile. Informational responses do not create
a body accumulator. Original body bytes pass directly to HistoricalResponse;
non-200 bodies are discarded under its finite budget.

HttpResult.status reports transport/bridge execution; callers must also inspect
HistoricalReport.outcome. A fully received HTTP 429, for example, has successful
transport and a source HTTP-error outcome. HTTP completion does not establish
complete market coverage. Incomplete framing/timeout/cancellation cannot produce
a completed capture, even when the body ends at a valid DBN record boundary.
The attempt ledger records the terminal operational outcome; completed means the
HTTP operation returned a complete response, not successful data acquisition.
Persistence failure is explicit. No automatic retry, delay, new quote, budget
increase or paid request follows from a recovery recommendation.

Focused evidence uses loopback servers and fabricated credentials for framing,
status, truncation, header bounds, encoding rejection, redirects, timeout,
cancellation, refused bridges, swallowed sink errors and one-shot execution.
No real provider call, operational SSIAG retrieval, production TLS deployment,
live subscription or reference operation is implied. Live data remains a future
objective under Duncan's 28 September 2026 direction.

Primary libcurl references reviewed 28 September 2026:
https://curl.se/libcurl/c/CURLOPT_PROTOCOLS_STR.html,
https://curl.se/libcurl/c/CURLOPT_TIMEOUT_MS.html,
https://curl.se/libcurl/c/CURLOPT_XFERINFOFUNCTION.html.


## Additive local dataset reader profile

`dataset_limits_contract` identifies
`symphony.sqav.databento.dataset-user-limits.v1`. `FileView::inspect_dataset`
uses `DatasetLimits` with optional uint64 `max_file_bytes`, `max_metadata_bytes`
and `max_records`. Absence imposes no application policy ceiling; present values
must be positive and may use the full uint64 representation. Format constraints,
valid byte spans and host size representation still apply. Parsing is borrowed,
allocation-free and preserves the previous output view on failure. The complete
file is validated before publication of a view.

This profile supports user-governed SBV file/resident datasets. It is additive to
the existing 0.5.0-dev development source and does not alter the bounded `Limits`
contract used by `inspect`, capture/import, historical acquisition or transport.
The selected Databento C++ source reference remains v0.68.0. Callers choose the
new profile explicitly; no existing acquisition safety/budget settings change.
