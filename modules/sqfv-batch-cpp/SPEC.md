# SQFV Batch C++ Specification

## Exact identity and trust scope

`sqfv-batch-cpp` release `0.1.0-dev` is an SQFV-owned C++20 static library exposing ABI version 1 through [batch.h](include/symphony/sqfv/batch.h). The public boundary consists only of C symbols, fixed-width integers, pointer-and-length spans, sized/versioned structures, opaque handles, status codes, and module-owned release calls. A call must return a status rather than let a C++ exception cross the ABI. The package is for trusted callers in one address space. It supplies no process entry point, network channel, hardware mapping, authorization system, or durable replay.

All inputs that carry `struct_size` and `abi_version` require the exact v1 structure size and `SQFV_BATCH_ABI_VERSION`; reserved fields are zero. Callers also initialize those two fields on descriptor and statistics output structures before invoking a view or statistics operation. An output size/version mismatch returns `SQFV_UNSUPPORTED_ABI` without publishing borrowed pointers. A nonzero span requires a nonnull pointer. Input spans are borrowed only during the call. `sqfv_batch_descriptor_view` and `sqfv_lease_descriptor_view` borrow their spans from the exact retained batch or lease handle. `sqfv_lease_view` borrows payload bytes from the exact lease and becomes invalid when that lease is released. Each returned opaque handle is released through its matching module function. Status-returning functions catch C++ allocation and synchronization failures; void destruction and release functions terminate on an internal synchronization failure rather than falsely claim reclamation. Caller-owned input and frame-output buffers remain outside the module's allocation accounting.

## Binding, descriptor, and identity

The five required binding byte strings are `metadata_ref`, `dataset_revision`, `schema_version`, `layout_version`, and `access_scope`. SQFV compares exact bytes; SQMV defines meaning and reference grammar. No field invokes latest-version resolution or grants access. The descriptor adds a required partition, optional source binding and opaque source-native position, a nonzero caller-supplied 16-byte producer generation, an unsigned batch sequence, and a separate positive record count. A nonempty source position requires a nonempty source binding. Source-native position remains uninterpreted; it need not be arithmetic or contiguous. Dataset/view revision, partition, generation, and sequence form the module's logical position key, under the exact metadata binding.

The v1 canonical descriptor is the following byte sequence, with no padding: eight unsigned 16-bit big-endian lengths each followed immediately by raw bytes in this order: metadata reference, dataset revision, schema version, layout version, access scope, partition, source binding, source position. The sequence ends with the 16 generation bytes, batch sequence as unsigned 64-bit big-endian, and record count as unsigned 64-bit big-endian. A field can contain arbitrary bytes, including NUL; each field is at most 65,535 bytes and the combined canonical descriptor must fit the configured and technical descriptor limits.

The 32-byte content ID is `SHA-256(domain || canonical_descriptor || SHA-256(payload))`, where `domain` is the UTF-8 bytes of `symphony.sqfv.batch-content.v1` **including one trailing NUL byte**. This digest distinguishes changed bytes at one logical position and accidental corruption; it does not authenticate the producer. Exact descriptor bytes, including source-native position and record count, affect the content ID.

## Lifetime and bounded flow

`sqfv_batch_prepare_copy` validates descriptor and budgets, reserves global accounting, then copies caller bytes into one immutable module-owned allocation. The caller may mutate or free its original input once the call returns. A retained batch handle or read lease keeps that copied allocation alive until its own release; releasing a producer handle does not invalidate an outstanding lease. A taken lease remains valid after producer batch, port, and context handle destruction. Operations on distinct live handles synchronize through shared context/port state; callers must not concurrently destroy the same opaque handle while another thread uses that exact pointer. A payload shared by ports is one physical payload allocation, while each port charges its own outstanding-byte credit.

Each port binds the five exact binding fields, partition, and producer generation and begins at its caller-selected `next_sequence`. Offers must use that next sequence. A blocked offer consumes no cursor; a new higher sequence reports a gap. Repeating only the most recently accepted position with the same content ID reports duplicate, and changed content reports conflict. An older position reports stale without a full-history deduplication claim. Taking a delivery yields a lease; the accepted cursor is not rolled back by take, cancel, or release. Port cancellation removes a still-pending delivery. Destroying a port cancels its pending deliveries and returns their credits, while already taken leases remain valid and return their credits only on release. These calls require no new data-queue credit, so release and cancellation can progress when the queue is full.

Acceptance is a local queue event, not a receipt of processing or durable commit. The library does not persist a generation, prevent its reuse after a process crash, enforce private access against malicious same-process code, or silently drop a stalled port's payload. The caller must decide whether and when to retry blocked offers, select optional ports, and provide any later retention/replay policy.

## Resource bounds

Every context sets positive finite `max_payload_bytes`, `max_frame_bytes`, `max_descriptor_bytes`, `global_allocation_bytes`, and `max_ports`. Each port sets positive `outstanding_byte_credit` and `max_pending_entries`. The implementation rejects any configured value exceeding its v1 technical ceilings: **64 MiB payload**, **128 MiB frame**, **64 KiB canonical descriptor**, **1,024 ports**, and **65,536 pending entries per port**. It also rejects lengths that cannot be represented by the wire fields or host address space. These are package ceilings, not an SQV-wide batch rule or throughput promise.

Module-owned context, batch, descriptor, payload, lease, port, and queue/index dynamic allocations charge reservation units against `global_allocation_bytes` before allocation. The charge includes requested payload and descriptor bytes, queue/object extents, and a fixed per-allocation allowance. It bounds this logical sum, **not physical RSS or allocator footprint**: allocator size-class rounding can exceed the allowance, while some charged units can exceed physical usage. Context base overhead remains charged while the context handle lives. The codec uses a fixed stack SHA-256 state, block, and schedule, with no heap or payload-sized scratch; that bounded call-stack workspace is not retained context allocation. A batch transitions from preparation into port/lease use without a second physical-payload charge. Cache capacity is zero in this release. Statistics report current and peak charged reservation units as well as each port's pending entries and outstanding bytes. A bounded failure returns an explicit status such as `SQFV_LIMIT` or `SQFV_BLOCKED` before exceeding configured charge or credit limits. The caller's frame output buffer and the borrowed decode input are outside context-owned allocation accounting.

## Local frame v1

The frame is a byte-defined, big-endian local serialization of one canonical descriptor and payload. Its first 24 bytes are a fixed prefix:

| Offset | Width | Value |
| ---: | ---: | --- |
| 0 | 4 | ASCII `SQF1` |
| 4 | 2 | major version, unsigned big-endian, exactly 1 |
| 6 | 2 | minor version, unsigned big-endian, exactly 0 |
| 8 | 4 | required-feature flags, unsigned big-endian, exactly 0 |
| 12 | 4 | header bytes, unsigned big-endian |
| 16 | 4 | payload bytes, unsigned big-endian |
| 20 | 4 | canonical descriptor bytes, unsigned big-endian |

The prefix is followed by the canonical descriptor, a 32-byte SHA-256 digest, then the payload. `header_bytes = 24 + descriptor_bytes + 32`; total frame bytes equal `header_bytes + payload_bytes`. There is no alignment padding, compression, optional required field, or trailing byte. The frame digest is SHA-256 of the UTF-8 domain `symphony.sqfv.batch-frame.v1` **including one trailing NUL byte**, followed by the exact 24-byte prefix, exact descriptor bytes, and exact payload bytes; the digest field itself is excluded. A decoder parses byte spans and never casts untrusted input to a native struct.

The decoder rejects unsupported major/minor versions or nonzero flags with `SQFV_UNSUPPORTED_FRAME`; malformed lengths, truncation, trailing bytes, or a changed digest return `SQFV_CORRUPT_FRAME`; configured oversize returns `SQFV_LIMIT`. No decoded batch is published on failure. [batch.h](include/symphony/sqfv/batch.h) provides `frame_measure`, `frame_encode`, and `frame_decode`; encode writes only into a caller-owned output buffer. The frame digest is an integrity check, not a signature or authorization proof. This first frame is not an admitted IPC, cross-process memory, or network transport format.

## Numerical acceptance fixture

The focused stream fixture uses one producer, two ports (`fast` and `slow`), 65,536-byte deterministic payloads, 10,000 ordered batches per partition/generation, 4,096-byte maximum descriptor, 73,728-byte maximum frame, 8 MiB charged reservation budget, and each port's 4 MiB outstanding-byte credit and 64 pending entries. `slow` holds 64 leases while `fast` continues to receive and release; the next `slow` offer must block explicitly and later succeed after credit returns. This fixture profile is a reproducible test configuration. It is not a product throughput or latency threshold and does not establish persistence or provider delivery.
