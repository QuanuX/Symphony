# SQAV attempt administration spec

Release `0.1.0-dev`, process protocol `symphony.knowledge.engine-process.v1`, SQAV ownership. Operations and exact input/output protocols are declared in OWNER-INTERFACE; schemas and templates are installed and content-bound. C++26 only. The shared transport support in tools/sqv-administration-cpp owns JSON/envelope mechanics, never vector semantics.

Requests: at most 262144 payload bytes, 524288 process bytes; five-second deadline. All unsigned integers are canonical decimal strings, including the full uint64 range. Opaque native strings are lower-case hexadecimal bytes; no Unicode normalization or implicit UTF-8 conversion. Exact object fields are required. No network or persistent mutation. Returned results bind the canonical request, semantic owner and result digest. Input errors never echo private data.

Dependencies: KnowledgeVectorEngine 0.2.0, SQFV batch 0.3.0, SQMV metadata 0.2.0; conversion additionally uses SQTV integer conversion 0.2.0. Observation engines share the SQPV-owned v1 store reader from sqpv-inspection-engine; they do not link a writer or call Store::open. Source dependency versions are checked. Dependencies are statically embedded; existing native package versions and ABI are preserved.


## Persisted acquisition attempt contract

`attempts_inspect` uses the SQPV-owned read-only reader for the exact sqav-attempt-v1 profile emitted by sqav-databento-dbn-cpp 0.5. It verifies canonical budget metadata, source/binding/generations, complete SQA1 reservations and terminal records, unique IDs, bounded quote times/retry ordinals, the request reference derived from exact parameters, valid terminal transitions and monotone conservative charge accounting. It performs no provider call.

offset and limit select a page (at most 128 records); accounting still verifies the whole selected store within explicit finite bounds. Reserved ceilings, prior charge and remaining ceiling are reported as uint64 decimal nano-USD strings. They are not invoices or observed provider costs, and no failure/cancellation refund is inferred. Raw request parameters and quote evidence are not emitted.

Persisted reserved remains visibly reserved, with restart_outcome=indeterminate. Inspection issues no ticket or execution capability. Dirty storage is refused before accounting is returned. Empty clean ledgers retain their recorded prior charge. The native observation never calls AttemptLedger::open, which would perform storage recovery and synthesize restart state.
