# SQFV flow administration spec

Release `0.1.0-dev`, process protocol `symphony.knowledge.engine-process.v1`, SQFV ownership. Operations and exact input/output protocols are declared in OWNER-INTERFACE; schemas and templates are installed and content-bound. C++26 only. The shared transport support in tools/sqv-administration-cpp owns JSON/envelope mechanics, never vector semantics.

Requests: at most 262144 payload bytes, 524288 process bytes; five-second deadline. All unsigned integers are canonical decimal strings, including the full uint64 range. Opaque native strings are lower-case hexadecimal bytes; no Unicode normalization or implicit UTF-8 conversion. Exact object fields are required. No network or persistent mutation. Returned results bind the canonical request, semantic owner and result digest. Input errors never echo private data.

Dependencies: KnowledgeVectorEngine 0.2.0, SQFV batch 0.3.0, SQMV metadata 0.2.0; conversion additionally uses SQTV integer conversion 0.2.0. Observation engines share the SQPV-owned v1 store reader from sqpv-inspection-engine; they do not link a writer or call Store::open. Source dependency versions are checked. Dependencies are statically embedded; existing native package versions and ABI are preserved.


## Native flow contract

`flow_validate` accepts one explicit Context resource configuration, one Port configuration and one proposed frame descriptor/payload size. It constructs a temporary Context/Port, prepares a zero-filled trial payload, measures its frame and offers the immutable trial batch. These existing SQFV 0.3 APIs check binding, generation, sequence, allocation budget, credit, pending queue and frame bounds together. The trial ends with process exit; its peak allocation reports this build's isolated trial, never a running application's memory or flow status.

All descriptor/binding strings in this administrative profile are at most 4096 decoded bytes. Payload size is finite, at most 64 MiB, and must fit the explicit native limits. This describes a local configuration trial; SQMV reference resolution, payload value meaning, external topology, sustained throughput and concurrent runtime readiness are not established.
