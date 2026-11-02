# Databento native reference and cost adapter: exact 0.1.0-dev contract

## Supported extent

Reference HTTP v0 observed 2026-09-28, parameters cross-checked against official databento-python v0.84.0 source (research evidence only, no Python runtime). Corporate actions get_range, adjustment factors get_range, security master get_range/get_last support 1–128 explicit raw symbols; ranges use explicit ISO dates with start before end. All requests set allocate_isins=false and compression=zstd; corporate index is event_date, security range index ts_effective. No ALL_SYMBOLS or implicit current range is selected. Point-in-time rows and nested fields are retained without flattening or latest-record collapse. Exact zstd 1.5.7 static dependency; compressed and expanded limits each 1–64 MiB, at most 65,536 rows, bounded JSON per record and decoder window log 10–23. Full compressed-frame completion is mandatory; truncation, trailing garbage, excess output and malformed/duplicate JSON fail atomically. Captures preserve original compressed bytes; decoded JSONL is also available unchanged. Identifier shape validation is deliberately narrower than full provider schema semantics. Reference coverage remains partial because allocation filtering and provider entitlements can omit rows. CostQuote uses metadata.get_cost for the exact existing historical MBO request. Positive/zero JSON decimal USD is converted upward to nano-USD using integer decimal arithmetic, with overflow refusal and no binary-float conversion. Quotes bind exact request, caller-recorded time and 1–86,400,000 ms validity; only matching, unexpired quotes produce AttemptQuote inputs for the durable ledger. Provider estimates are not guaranteed invoices or account-wide budget authority.

## Native/package boundary

C++26, initially verified on macOS amd64 / AppleClang 21 with compatible native consumers. Fallible preparation/admission preserves output on failure unless the API explicitly reports an attempted operation. Immutable handles may be retained; mutation, reset, destruction and move must not race use of the same handle. Source data is untrusted. Errors contain status only, never source contents or credentials. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev, sqav-databento-dbn-cpp 0.5.0-dev, zstd 1.5.7 static library. No runtime discovery or implicit upgrade. The module exports no standalone process or qxctl command.

## Verification and limits

`tests/test.cpp` exercises supported positive and negative boundaries; `tests/sdk-consumer/main.cpp` exercises the installed public API outside the checkout. The SQV non-live source composition tests preserve selected capture evidence through exact local retention/replay. The tests and package receipts do not establish provider access, production conformance or a licence grant.

No provider activation, reference entitlement guarantee, source authenticity from fixtures, new ISIN allocation, live data, full reference-schema normalization, account-wide billing reconciliation, automatic paid retry or operational SSIAG bridge.

## Explicit historical acquisition handoff

`acquire_quoted_historical` joins an already obtained `CostQuote`, a separately
approved `AcquisitionAuthorization`, the DBN owner's durable `AttemptLedger`
and its one-attempt historical HTTPS operation. Obtaining a quote and approving
spend remain separate actions: `CostQuote::for_attempt` alone is not approval
and its estimate must not silently become a guaranteed charge ceiling.

Authorization supplies a nonsecret evidence reference, the exact historical
request reference, exact quote evidence reference, unique attempt ID, independently
selected ceiling in nano-USD, and explicit authorization/expiry UTC milliseconds.
The lifetime is positive and at most one day. The trusted caller owns the
approval's authenticity and the basis for its ceiling; this SDK validates their
binding, not the authority behind them. The ceiling must cover the observed
estimate. Neither field configures a provider-side spending cap or proves an
invoice/account-wide limit.

The operation checks the runtime UTC clock, refuses a mismatched, stale,
future-dated or underfunded authorization, validates HTTP bounds and respects
pre-existing cancellation before reserving. It derives a nonsecret
`db-acquisition-v1-` reference from the request, quote, approval, attempt ID,
ceiling and authorization times. The ledger durably reserves the full approved
ceiling and the intersection of quote/authorization validity before any key use.
The caller retains the original quote and authorization evidence alongside this
reference; the ledger is not a substitute for those records.

After persistence, the clock and cancellation are checked again. Expiry,
backward clock movement or cancellation closes the reserved attempt without
obtaining a key. Otherwise the HTTP duration is bounded by both remaining
validities and the caller's HTTP limit, then the DBN owner is invoked once.
The actual native credential bridge still owns lease/recipient/request binding,
revocation and timely cleanup; arbitrary blocking callbacks cannot be forcibly
interrupted by this wrapper.

`AcquisitionResult` keeps local admission, durable reservation, whether execution
was invoked, transport/response classification and terminal persistence separate.
`admission=ok` does not mean authorization authenticated or data acquired;
`reservation=ok` means the charge ceiling was durably reserved; transport
completion does not mean complete market-data coverage. No result refunds a
charge automatically. A duplicate attempt ID cannot produce a new ticket.
An uncertain reservation or missing terminal requires explicit ledger reopening
and inspection; unresolved attempts recover as indeterminate with their charge
retained. Recovery never sends another request. Another paid attempt requires
a new explicit authorization/attempt decision under the remaining budget.

## Operational qualification, prepared but not performed

The selected scope remains historical MBO and the four declared reference
operations, not live feeds. Before launch qualification, provide the exact
installed SDK/receipt identities, a trusted native host, an authorized Databento
account/entitlement profile and both native credential-use adapters. The quote
and reference adapters receive `source::HttpRequest::reference()`; the historical
adapter receives `HistoricalPlan::reference()`. These are distinct identities:
one lease must not be replayed under another request identity. No API key belongs
in parameters, command arguments, environment variables, logs or evidence files.

For each required Apple Silicon macOS, Ubuntu LTS x86_64 and Ubuntu LTS ARM64
profile, the authorized operator must:

1. Record the exact source/SDK receipts, platform/toolchain and clock evidence;
   select an entitled dataset/symbol/window, fixed response limits and a finite
   private ledger budget. Record the independent basis for the approved ceiling.
2. Obtain the exact bounded quote using a request-bound native credential lease,
   retain its original nonsecret evidence and receive approval for the exact
   request, quote, attempt ID, ceiling and expiry. Invoke the joined operation
   once and retain its separate reservation, HTTP, DBN coverage and persistence
   results. Reconcile any actual charge independently; estimate agreement is not
   invoice proof.
3. Exercise authorized refusal/expiry/revocation and cancellation cases with the
   real bridge. Confirm no key delivery before reservation and no credential
   bytes in diagnostics. Exercise controlled transport interruption and a
   retained unresolved attempt; reopen/inspect without issuing a replacement
   request, verify the charge is retained, and explicitly decide any retry.
4. Independently call each promised reference operation with selected finite
   bounds and entitlements. Retain original compressed bytes, decoded-row
   evidence, request identity and partial-coverage attribution; these reference
   operations do not consume the historical MBO attempt ledger or assert a
   historical quote covers reference charges.

Fixtures, denial callbacks and local SDK tests establish implementation
boundaries only. Profiles, credentials and operational bridge delivery are
external prerequisites; no provider-authenticated acquisition, invoice
reconciliation, full-feed completeness or physical DGX qualification is claimed.
