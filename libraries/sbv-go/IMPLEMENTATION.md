# SBV Go transport and data contract

Contract: `symphony.sbv.go-adapter.v1`. Module version `v0.1.6-dev`; native owner
release `0.32.0-dev`; dependency Go 1.26.5 standard library only. Native source
contracts and receipt-owned schemas outrank this convenience API. The adapter
introduces no scientific protocol, administration state or registry authority.

| API | qxctl-first mapping | Meaning |
| --- | --- | --- |
| `Operations`, `Invoke` | Exact 42 `qxctl sbv` native leaves | Caller supplies complete native input; no scientific values are recomputed |
| `Resource` | `sbv schema`, `sbv template --variant` | Exact installed schema/example; no inferred newest version |
| `Invoke(result_inspect/result_query)` | `sbv result inspect/query` flags | Closed read request fields become exact flags |
| `Stream(result_export)` | `sbv result export --format ndjson` | Legacy complete artifact, native inspection and logical hash verification |
| `Invoke(bundle_export)` | `sbv bundle export --receipt-only --format json` | Bounded receipt; caller selects native export format |
| `Stream(bundle_export)` | `sbv bundle export --format ndjson` | Full native publication/stream validation and adapter logical hash verification |
| `Rows` | Same exports, optionally preceded by `sbv result select` | Consumer shape/feature-order projection; native filters and all retained fields stay available |

`result_export` uses the existing `result-inspect-input.v1` request. A full
content digest is required for streams; bundle streams additionally require the
exact manifest path and digest. `ReadStream` accepts a caller-owned reader and
expected reference; it checks bytes/structure and does not authenticate origin
or reopen a manifest. `Client.Stream` additionally requires successful qxctl
execution. Raw node data remain strings, booleans, null and empty container
shells. Duplicate keys, invalid UTF-8/surrogates, numeric JSON scalars, changed
source identities, invalid preorder, child counts, missing completion, trailing
data and mismatched logical hashes fail with typed adapter errors.

`Table` selects an array pointer; its named columns have relative pointers and
one of `json`, `string`, `bool`, `int64`, `uint64`, `rational`. Integer types require
canonical decimal strings in the declared range. Rational accepts canonical
arbitrary-precision numerator/positive-denominator strings and never reduces,
rounds or evaluates them. Native checked rational domains still govern native
operations. Missing optional fields produce `Present:false` and JSON null;
actual null produces `Present:true` and requires `Nullable:true`. Complete status
does not make an unavailable scientific field available. The `json` dtype
retains nested diagnostic/availability evidence. Empty column selection and empty
arrays are valid; absent arrays and shape mismatches are errors.

Transport ownership is copying throughout. Request bytes are read during the
synchronous call; callers must not mutate them concurrently. Requests use unique
mode-0600 temporary files in a caller-owned private directory and are removed
when the call returns. Nodes, cells and replies own their bytes and remain valid
after subsequent callbacks, stream cancellation, process exit or adapter removal.
Callbacks may retain copies but must avoid concurrent mutation of values they
share. The client holds immutable configuration and supports concurrent calls;
native destination/journal/instance contracts still govern conflicts.

An `Error` supplies a code, detail, optional unmodified qxctl failure response and
wrapped cause. No failure is translated into an empty dataset, zero probability
or safe-to-retry assertion. `recovery_required` and other native statuses remain
in successfully transported replies. Calls do not execute external scripts,
authorize endpoints, create services or contact model providers. An external
program supplies native `external_outcomes` or `experiment.trials` with its own
model/proposal provenance. Native code decides contract admissibility; calibration
and producer declarations remain scoped evidence. Native 0.28 owns adaptive revisions, proposal events and automatic selection/refit
for caller-authored graphs; the adapter transports them without selecting policy.

Host selection is explicit. SHA256 pins qxctl bytes; qxctl validates native
receipt ownership. These are integrity checks under a trusted local host, not
distribution signatures or protection against a hostile concurrent owner who
replaces the executable between checking and launch. Native isolation and
remote identity are not added by this adapter. The tested process-control profile
is macOS; Unix SIGTERM cleanup is used. Windows and remote execution are not
qualified. No forced kill grace period or background retry is imposed.

User resource constraints remain optional. Stream parser state retains full open
ancestor addresses plus one record; combined pointer lengths can grow quadratically
with nesting depth. Row decoding scales with an entire source row (including fields
not selected as columns), plus copied selected cells. Huge single rows/strings
therefore require available memory or native preselection. Native and Go host
integer/address/framing limits still exist; no total dataset/row count ceiling is
introduced. A blocked custom `io.Reader` or callback requires caller cooperation;
context cancellation is checked between reads, never claimed as a completion
fence for foreign asynchronous buffers. No borrowed/device memory is admitted.


Native 0.29 adds selectable binary classification, probability reliability and
ranking studies to analyze. This 0.1.3 adapter transports the full `binary` choices
and exact results unchanged; it does not select thresholds, labels or objectives.


Native 0.30 / transport 0.1.4 adds account_simulate to the exact operation inventory.
The adapter carries streams, guards, constraints, state references and ledger results
unchanged; C++ owns sequencing, arithmetic and admission.


Native 0.31 / transport 0.1.5 carries account liquidity pools, target-position
rules, explicit splits and pool-aware v2 snapshots unchanged. No new transport
operation is needed; qxctl/native schema and result controls own these fields.

Native 0.32 / transport 0.1.6 preserves feedback selection, explicit memory,
v3 snapshots and linear variation settlement fields unchanged through qxctl.

qxctl renders full installed schema definitions through its receipt-verified
resource reader. Schema growth does not alter runtime control-frame admission;
feedback provider inspection verifies the separately selected ABI symbol and
descriptor protocol before returning portable evidence.
