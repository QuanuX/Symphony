# SQMV metadata administration spec

Release `0.1.0-dev`, process protocol `symphony.knowledge.engine-process.v1`, SQMV ownership. Operations and exact input/output protocols are declared in OWNER-INTERFACE; schemas and templates are installed and content-bound. C++26 only. The shared transport support in tools/sqv-administration-cpp owns JSON/envelope mechanics, never vector semantics.

Requests: at most 262144 payload bytes, 524288 process bytes; five-second deadline. All unsigned integers are canonical decimal strings, including the full uint64 range. Opaque native strings are lower-case hexadecimal bytes; no Unicode normalization or implicit UTF-8 conversion. Exact object fields are required. No network or persistent mutation. Returned results bind the canonical request, semantic owner and result digest. Input errors never echo private data.

Dependencies: KnowledgeVectorEngine 0.2.0, SQFV batch 0.3.0, SQMV metadata 0.2.0; conversion additionally uses SQTV integer conversion 0.2.0. Observation engines share the SQPV-owned v1 store reader from sqpv-inspection-engine; they do not link a writer or call Store::open. Source dependency versions are checked. Dependencies are statically embedded; existing native package versions and ABI are preserved.


## Native metadata contract

`metadata_validate` calls SQMV 0.2 Manifest::create with the supplied Description and mandatory limits. Six description strings and evidence producer/reference strings are exact hex bytes, including NUL and invalid UTF-8. Required schema/layout/access roles, evidence uniqueness, canonical ordering and size ceilings remain SQMV semantics.

`metadata_inspect` calls Manifest::resolve on consecutive `encoded_hex_chunks`, each 16384 decoded bytes except the last, with the exact expected reference. The full 65536-byte native manifest range fits the transport. It returns the same binding and description plus the selected evidence projection (`all`, schema, layout, access, source, time, coverage, lineage or units). This single inspection operation supplies all metadata projections. Evidence references are preserved; their external truth, availability and access authority are not resolved.
