# SQDV checkpoint administration spec

Release `0.1.0-dev`, process protocol `symphony.knowledge.engine-process.v1`, SQDV ownership. Operations and exact input/output protocols are declared in OWNER-INTERFACE; schemas and templates are installed and content-bound. C++26 only. The shared transport support in tools/sqv-administration-cpp owns JSON/envelope mechanics, never vector semantics.

Requests: at most 262144 payload bytes, 524288 process bytes; five-second deadline. All unsigned integers are canonical decimal strings, including the full uint64 range. Opaque native strings are lower-case hexadecimal bytes; no Unicode normalization or implicit UTF-8 conversion. Exact object fields are required. No network or persistent mutation. Returned results bind the canonical request, semantic owner and result digest. Input errors never echo private data.

Dependencies: KnowledgeVectorEngine 0.2.0, SQFV batch 0.3.0, SQMV metadata 0.2.0; conversion additionally uses SQTV integer conversion 0.2.0. Observation engines share the SQPV-owned v1 store reader from sqpv-inspection-engine; they do not link a writer or call Store::open. Source dependency versions are checked. Dependencies are statically embedded; existing native package versions and ABI are preserved.


## Persisted checkpoint contract

`checkpoint_inspect` uses the SQPV-owned read-only reader and interprets the exact sqdv-checkpoint-v1 metadata/payload profile emitted by SQDV 0.3. It verifies the canonical view/baseline description, owner evidence, partition, generations, record count/source binding, 13-byte SQC1 payload and strictly advancing checkpoints. An exhausted cursor must be UINT64_MAX. The first persisted record must equal the declared baseline.

An empty, clean journal returns baseline_not_persisted and latest_persisted=null; it never appends the baseline. Dirty storage is refused with sqv.store.recovery_required. A persisted position represents local processing acknowledgment only. No remote commit, delivery lease, running session or remote exactly-once state is established. The request uses the same explicit storage identity and observation bounds as SQPV inspection.
