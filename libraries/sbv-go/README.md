# SBV Go adapter

Experimental source module `github.com/QuanuX/Symphony/libraries/sbv-go`, version
`v0.1.7-dev`, targeting Go 1.26.5 and native SBV `0.33.0-dev`. It uses only the Go
standard library. This is a local source release; no published module/tag is
claimed. C++ owns every scientific calculation and admission.

The adapter invokes an explicitly selected, SHA256-pinned qxctl executable.
qxctl validates the exact native installation receipt, request and response.
`Invoke` exposes all 42 native operations, including resource controls,
external outcomes and experiment proposals. `Resource` reads the installed
schema/template. The adapter carries complete requests without selecting
criteria, objectives, weights, models, limits, deadlines or retry policies.

`Stream` reads legacy pointer NDJSON or partitioned node NDJSON into owned,
exact-value nodes. It checks tree structure, source correspondence, completion
and the native logical content SHA256 without accumulating the whole result.
`Rows` supplies selected array rows in caller-declared column order, with explicit
types, nullability and missing-field status. It preserves integer/rational
strings; it never silently converts to float. Native `result_select` provides
time/source filters, ordering and column selection before downstream transport.

```go
client, err := sbv.New(sbv.Config{
    Executable: "/selected/bin/qxctl",
    ExecutableSHA256: "<exact executable SHA256>",
    Prefix: "/selected/native-prefix", Version: sbv.NativeVersion,
    WorkDir: "/private/work", RequestDir: "/private/requests",
})
// Handle err; requestDir must already exist with private permissions.
// Native request JSON comes from client.Resource(ctx, operation, "template", "").
result, err := client.Invoke(ctx, "evaluate", requestJSON)
// Inspect native result status and reconcile write destinations after errors.
```

Use `context.Background()` for no deadline, or a caller-selected context for
cancellation/deadlines. A synchronous callback or arbitrary reader can block;
the caller owns that I/O. Client cancellation sends SIGTERM to qxctl, which owns
native process-group cleanup, and waits for the subprocess. No hidden timeout,
automatic retry, dataset cap or fallback is added.

Every callback value is **provisional** until the final completion is true and
the returned error is nil. An early callback failure cancels and reaps qxctl;
already published files remain for native inspection. `Rows` holds one entire
source row; `Stream` holds open ancestor addresses plus the largest record.
Combined ancestor pointer lengths can grow quadratically with nesting depth. Consumer-retained
values add their own memory use. `MaxRecordBytes: 0` selects no record limit;
otherwise the caller's limit applies. Control replies and legacy operations
retain their native format limits. Choose partitioned results for scalable
output. No terabyte/RSS bound is inferred from these allocation descriptions.

Install independently by copying this source module to a caller-selected
directory and adding an exact `require` plus local `replace` in the consumer's
`go.mod`. Run with `GOWORK=off` when testing outside the parent development
workspace. The [round-trip example](examples/roundtrip/main.go) builds in an
independent consumer module and accepts a client config, an existing source
result and a fresh output directory. It exports Databento replay, selects columns
and receive times through qxctl, consumes a partitioned copy, submits two
synthetic external model proposals, verifies zero-work experiment resume, and
consumes native comparison feedback to select the declared objective's winner.
Its declared probabilities are examples, never calibrated predictions or a
claim of an integrated ML/optimizer vendor.

Remove the consumer's module requirement/replace and its selected adapter source
copy to uninstall. No native prefix, qxctl binary, resident data or result is
owned by this package. Already compiled consumers contain their own library
copy; remove/rebuild those binaries separately. Source package identity and
removal are ordinary Go dependency choices, not a new runtime installation
service. See [implementation contract](IMPLEMENTATION.md) for qxctl mappings,
trust, precision and lifetime details.

Verification: `GOWORK=off go test -race ./...` with Go 1.26.5. Installed evidence
is recorded separately in the workspace R5.2 packet; tests do not contact a data
provider or connect live services. Arrow, DLPack, borrowed buffers, CUDA/tensor
runtimes, authentication of producer claims, built-in proposal algorithms and other language
adapters remain separate capabilities. This adapter neither requires them nor
prevents an independently selected implementation.
