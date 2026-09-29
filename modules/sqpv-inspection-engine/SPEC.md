# SQPV inspection administration spec

Release `0.1.0-dev`, process protocol `symphony.knowledge.engine-process.v1`, SQPV ownership. Operations and exact input/output protocols are declared in OWNER-INTERFACE; schemas and templates are installed and content-bound. C++26 only. The shared transport support in tools/sqv-administration-cpp owns JSON/envelope mechanics, never vector semantics.

Requests: at most 262144 payload bytes, 524288 process bytes; five-second deadline. All unsigned integers are canonical decimal strings, including the full uint64 range. Opaque native strings are lower-case hexadecimal bytes; no Unicode normalization or implicit UTF-8 conversion. Exact object fields are required. No network or persistent mutation. Returned results bind the canonical request, semantic owner and result digest. Input errors never echo private data.

Dependencies: KnowledgeVectorEngine 0.2.0, SQFV batch 0.3.0, SQMV metadata 0.2.0; conversion additionally uses SQTV integer conversion 0.2.0. Observation engines share the SQPV-owned v1 store reader from sqpv-inspection-engine; they do not link a writer or call Store::open. Source dependency versions are checked. Dependencies are statically embedded; existing native package versions and ABI are preserved.


## Read-only SQPV store contract

`store_inspect` consumes only the frozen SQPS0001/SQPC0001/SQPH0001 retained format, SQMV v1 manifests and SQF1 frames. Conformance fixtures come from SQPV 0.2. It never constructs Store or calls any writer/recovery function.

The selection supplies an absolute root, expected metadata reference and store generation, plus positive max_read_bytes (at most 512 MiB) and max_batches (at most 65536). The whole directory inventory must fit those bounds; the five-second deadline also applies. This release supports local APFS, matching the retained writer's tested platform. Root must be owned by the caller and mode 0700; files must be regular, singly linked, caller-owned mode 0600. Path traversal and each entry use no-follow opens. Unsafe ancestors, symlinks, hardlinks, FIFOs, missing files and unexpected names are refused.

A shared nonblocking lock on the existing writer.lock excludes the writer's exclusive lock. All opens are O_RDONLY. Identity, size, mode, owner, modification/change times, absolute-path binding and directory inventory are rechecked before returning. There is no mkdir, create, truncate, write, fsync, rename, unlink, automatic lock creation, recovery or head publication. Filesystem-managed access-time accounting may still occur during reads; persistent_mutation=false describes SQV application state.

For a clean store, the reader checks metadata sealing, store configuration, the complete published commit chain, each frame hash/content identity, native frame decoding, metadata binding, partition/generation/sequence and the final head hash. Full-width counters are decimal strings; large opaque partitions use hex chunks.

A missing head, extra retained records or staging artifacts yield state=recovery_required. Only the published-head prefix is verified and returned; the tail is explicitly unvalidated. A complete-looking commit beyond that head is never promoted to an acknowledgment. Malformed acknowledged records and identity/bound failures return fixed errors instead of partial success. Digests establish local content correspondence, not publisher authenticity or power-loss durability. Cooperating processes follow the lock protocol; these checks are not isolation from a hostile same-user process with equivalent filesystem authority.
