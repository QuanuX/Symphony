# SQTV conversion administration spec

Release `0.1.0-dev`, process protocol `symphony.knowledge.engine-process.v1`, SQTV ownership. Operations and exact input/output protocols are declared in OWNER-INTERFACE; schemas and templates are installed and content-bound. C++26 only. The shared transport support in tools/sqv-administration-cpp owns JSON/envelope mechanics, never vector semantics.

Requests: at most 262144 payload bytes, 524288 process bytes; five-second deadline. All unsigned integers are canonical decimal strings, including the full uint64 range. Opaque native strings are lower-case hexadecimal bytes; no Unicode normalization or implicit UTF-8 conversion. Exact object fields are required. No network or persistent mutation. Returned results bind the canonical request, semantic owner and result digest. Input errors never echo private data.

Dependencies: KnowledgeVectorEngine 0.2.0, SQFV batch 0.3.0, SQMV metadata 0.2.0; conversion additionally uses SQTV integer conversion 0.2.0. Observation engines share the SQPV-owned v1 store reader from sqpv-inspection-engine; they do not link a writer or call Store::open. Source dependency versions are checked. Dependencies are statically embedded; existing native package versions and ABI are preserved.


## Native integer conversion contract

`conversion_validate` accepts two exact `sqtv-int-v1-*` layout IDs, a positive element count and the three mandatory SQTV 0.2 limits. All sixteen signed/unsigned, 8/16/32/64-bit, little/big-endian layouts are admitted through native parse_layout and operation_reference. The kernel and adapter share its private limits predicate; the public library ABI, converter identity and conversion behavior are unchanged.

Count must fit max_elements and both byte limits using checked division before multiplication. No payload values are supplied or converted. Success explicitly requires a value-range check during actual conversion; it never promises that narrowing or a signedness change will succeed for an unknown input.
