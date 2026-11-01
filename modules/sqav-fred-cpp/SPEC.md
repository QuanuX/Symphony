# FRED and ALFRED native adapter: exact 0.1.0-dev contract

## Supported extent

Provider surface: FRED unversioned HTTP API observed 2026-09-28; endpoints series/observations and series/vintagedates, JSON, ascending order. Explicit series, real-time start/end and observation dates prevent current-date defaults from changing identity. Observations use units=lin and output_type=1; actual physical units and seasonal metadata are not invented from that parameter. Exact date-level real-time bounds permit ALFRED snapshots without inventing publication instants. Page size 1–100,000 for observations or 1–10,000 for vintage dates; uint32 offsets/counts checked before arithmetic. The response must echo the requested ranges, limit, offset and selected output settings. Records must fill the exact remaining page count. Missing value dot is distinct from zero; values remain decimal text. Capture preserves original JSON, request/vintage identity and page scope; a single page is complete only when offset zero covers the declared result count. No multi-request atomicity is implied. Partial pages expose the next offset for an explicitly bounded caller workflow. Collection uses the shared native HTTP boundary and an actual supplied credential-use bridge.

## Native/package boundary

C++26, initially verified on macOS amd64 / AppleClang 21 with compatible native consumers. Fallible preparation/admission preserves output on failure unless the API explicitly reports an attempted operation. Immutable handles may be retained; mutation, reset, destruction and move must not race use of the same handle. Source data is untrusted. Errors contain status only, never source contents or credentials. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev. No runtime discovery or implicit upgrade. The module exports no standalone process or qxctl command.

## Verification and limits

`tests/test.cpp` exercises supported positive and negative boundaries; `tests/sdk-consumer/main.cpp` exercises the installed public API outside the checkout. The SQV non-live source composition tests preserve selected capture evidence through exact local retention/replay. The tests and package receipts do not establish provider access, production conformance or a licence grant.

No FRED key provisioning, operational SSIAG bridge, arbitrary output types, frequency aggregation, series metadata API, cross-page snapshot atomicity or provider-authenticated fixtures.

## Bounded page collection

`Collection::begin` admits an existing exact `Plan` at offset zero plus explicit
limits: 1–4,096 pages, 1–1,000,000 records, 1–65,536 collection calls and
1–67,108,864 original response bytes. It is a move-only, single-owner, in-memory
cursor; concurrent use is not supported. `next_plan()` is the exact pending
request. Each `collect_next` call makes at most one existing `collect` operation;
there is no automatic loop, delay or retry. Pre-existing cancellation makes no
call. Every other admitted collection call consumes its local call allowance,
including credential refusal, invalid transport bounds and incomplete responses.

`accept` admits an actual `Page` for the pending request, preserving every
original response. All pages must agree with the first observed result count,
fit the total record/page/byte bounds and advance the exact offset. Cross-page
date reversals, duplicate observation date/realtime tuples and non-increasing
vintage dates are refused. An empty result requires an observed first page.
Updates are staged: refusal/allocation failure preserves pending selection and
retained pages. A failed HTTP call likewise leaves the pending plan unchanged;
only the local call allowance is consumed. A retry is a separate caller action.

`progress().complete` means contiguous admitted pages exhausted the agreed
observed count. It does not turn individually partial page captures into an
atomic provider snapshot, detect every possible provider revision, invent
publication timestamps or authenticate a response. Count drift requires review
and a new explicit collection decision, not an automatically reset cursor.

After restart, a caller can reconstruct the initial exact plan and replay actual
retained/admitted pages through `accept` to recover the next offset. Replay
validates request/count/order/duplicate bindings and does not perform HTTP.
The original bytes, request and attribution must be retained by their owner.
This cursor does not persist itself, restore prior call counts, prove previous
network activity or provide an account-wide/durable rate budget. Operational
hosts must preserve that separate history and choose any new call allowance
explicitly; they must not infer permission to repeat an uncertain request.

## Operational qualification, prepared but not performed

The selected endpoints are observations and vintage dates. Qualification needs
an authorized FRED profile, the exact installed native SDK/receipts, a trusted
host and an operational `source::CredentialUse` adapter. The adapter must bind
its authenticated recipient/lease to each `HttpRequest::reference()`, enforce
expiry/revocation and the supplied maximum duration, deliver borrowed key bytes
only during the sink call and clean them afterward. The adapter must not convert
callback or transport refusal into success. A generic callback is not proof of
these properties.

On each required Apple Silicon macOS, Ubuntu LTS x86_64 and Ubuntu LTS ARM64
profile, select explicit finite observation/realtime dates and page limits;
collect a bounded authorized observation sequence and vintage-date sequence.
Retain each request reference, original response/capture, observed count,
progress and acquisition-time attribution. Check partial-page versus sequence
completion, decimal/missing fidelity and original-byte retention. Exercise
actual lease expiry/revocation, cancellation and interrupted response handling;
verify the pending request is preserved and any retry separately authorized.
Reconstruct progress from retained pages without network access, preserving the
separate prior request history. Record any provider changes between requests;
no cross-request snapshot guarantee may be inferred from a stable count.

Local fabricated-response and credential-refusal tests are implementation
evidence. They do not satisfy provider-authenticated acceptance. No profile/key
has been provisioned, no live FRED operation is claimed and generic Ubuntu ARM64
qualification is not physical DGX or GPU qualification.
