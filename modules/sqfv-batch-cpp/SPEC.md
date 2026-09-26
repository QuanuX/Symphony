# SQFV Batch C++ Specification

## Exact identity and trust scope

`sqfv-batch-cpp` release `0.2.0-dev` is an SQFV-owned C++26 static library exposing its native API through [batch.hpp](include/symphony/sqfv/batch.hpp) in namespace `symphony::sqfv`. It replaces the `0.1.0-dev` C ABI with owning C++ strings, `std::span` byte views, an enum-class `Status`, and move-only RAII `Context`, `Batch`, `Lease`, and `Port` handles. The exact CMake package and platform/compiler runtime select this development API; no standalone C symbols or C header are exported. The package is for trusted callers in one address space. It supplies no process entry point, network channel, hardware mapping, authorization system, or durable replay.

All configuration limits are explicit and finite. A `ByteView` input is borrowed only for the call; `prepare_copy` owns its result before returning. `Batch::descriptor()` and `Batch::content_id()` return references valid while that batch handle remains live. `Lease::payload()`, `Lease::descriptor()`, and `Lease::content_id()` return views or references valid while that lease remains live. Handle destruction, move assignment, or `reset()` discharges its exact ownership obligation. A caller must not destroy or move a handle concurrently with a call using that same handle. Operations that can fail return `Status` and publish output parameters only on success; exception failures are contained inside the library's `noexcept` API. Caller-owned input and frame-output buffers remain outside module reservation accounting.

## Binding, descriptor, and identity

The five required binding byte strings are `metadata_ref`, `dataset_revision`, `schema_version`, `layout_version`, and `access_scope`. SQFV compares exact bytes; SQMV defines meaning and reference grammar. No field invokes latest-version resolution or grants access. The descriptor adds a required partition, optional source binding and opaque source-native position, a nonzero caller-supplied 16-byte producer generation, an unsigned batch sequence, and a separate positive record count. A nonempty source position requires a nonempty source binding. Source-native position remains uninterpreted; it need not be arithmetic or contiguous. Dataset/view revision, partition, generation, and sequence form the module's logical position key, under the exact metadata binding.

The v1 canonical descriptor is the following byte sequence, with no padding: eight unsigned 16-bit big-endian lengths each followed immediately by raw bytes in this order: metadata reference, dataset revision, schema version, layout version, access scope, partition, source binding, source position. The sequence ends with the 16 generation bytes, batch sequence as unsigned 64-bit big-endian, and record count as unsigned 64-bit big-endian. A field can contain arbitrary bytes, including NUL; each field is at most 65,535 bytes and the combined canonical descriptor must fit the configured and technical descriptor limits.

The 32-byte content ID is `SHA-256(domain || canonical_descriptor || SHA-256(payload))`, where `domain` is the UTF-8 bytes of `symphony.sqfv.batch-content.v1` **including one trailing NUL byte**. This digest distinguishes changed bytes at one logical position and accidental corruption; it does not authenticate the producer. Exact descriptor bytes, including source-native position and record count, affect the content ID.

## Lifetime and bounded flow

`Context::prepare_copy` validates descriptor and budgets, reserves global accounting, then copies caller bytes into one immutable module-owned allocation. The caller may mutate or free its original input once the call returns. `Batch::retain` creates another owning producer handle. A retained batch or read lease keeps that copied allocation alive until its own destruction or reset; releasing a producer handle does not invalidate an outstanding lease. A taken lease remains valid after producer batch, port, and context handle destruction. Operations on distinct live handles synchronize through shared context/port state. A payload shared by ports is copied once, while each port charges its own outstanding-byte credit.

Each port binds the five exact binding fields, partition, and producer generation and begins at its caller-selected `next_sequence`. Offers must use that next sequence. A blocked offer consumes no cursor; a new higher sequence reports a gap. Repeating only the most recently accepted position with the same content ID reports duplicate, and changed content reports conflict. An older position reports stale without a full-history deduplication claim. `Port::take` moves one delivery into a `Lease`; the accepted cursor is not rolled back by take, cancel, or lease destruction. `Port::cancel` removes a still-pending delivery. Destroying or resetting a port cancels its pending deliveries and returns their credits, while already taken leases remain valid and return their credits only on destruction or reset. These calls require no new data-queue credit, so release and cancellation can progress when the queue is full.

Acceptance is a local queue event, not a receipt of processing or durable commit. The library does not persist a generation, prevent its reuse after a process crash, enforce private access against malicious same-process code, or silently drop a stalled port's payload. The caller must decide whether and when to retry blocked offers, select optional ports, and provide any later retention/replay policy.

## Resource bounds

Every context sets positive finite `max_payload_bytes`, `max_frame_bytes`, `max_descriptor_bytes`, `global_allocation_bytes`, and `max_ports`. Each port sets positive `outstanding_byte_credit` and `max_pending_entries`. The implementation rejects any configured value exceeding its v1 technical ceilings: **64 MiB payload**, **128 MiB frame**, **64 KiB canonical descriptor**, **1,024 ports**, and **65,536 pending entries per port**. It also rejects lengths that cannot be represented by the wire fields or host address space. These are package ceilings, not an SQV-wide batch rule or throughput promise.

Module-owned context, batch, descriptor, payload, lease, port, and queue/index dynamic allocations charge reservation units against `global_allocation_bytes` before allocation. The charge includes requested payload and descriptor bytes, queue/object extents, and a fixed per-allocation allowance. Context base overhead remains charged while the context handle lives. The codec uses a fixed stack SHA-256 state, block, and schedule, with no heap or payload-sized scratch; that bounded call-stack workspace is not retained context allocation. A batch transitions from preparation into port/lease use without a second payload charge. Cache capacity is zero in this release. Statistics report current and peak charged reservation units as well as each port's pending entries and outstanding bytes. A bounded failure returns an explicit status such as `Status::limit` or `Status::blocked` before exceeding configured charge or credit limits. The caller's frame output buffer and the borrowed decode input are outside context-owned allocation accounting.

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

The decoder rejects unsupported major/minor versions or nonzero flags with `Status::unsupported_frame`; malformed lengths, truncation, trailing bytes, or a changed digest return `Status::corrupt_frame`; configured oversize returns `Status::limit`. No decoded batch is published on failure. [batch.hpp](include/symphony/sqfv/batch.hpp) provides `frame_measure`, `frame_encode`, and `frame_decode` using `std::span` input/output byte views and a caller-owned output buffer. The frame digest is an integrity check, not a signature or authorization proof. This first frame is not an admitted IPC, cross-process memory, or network transport format.

## Numerical acceptance fixture

The focused stream fixture uses one producer, two ports (`fast` and `slow`), 65,536-byte deterministic payloads, 10,000 ordered batches per partition/generation, 4,096-byte maximum descriptor, 73,728-byte maximum frame, 8 MiB charged reservation budget, and each port's 4 MiB outstanding-byte credit and 64 pending entries. `slow` holds 64 leases while `fast` continues to receive and release; the next `slow` offer must block explicitly and later succeed after credit returns. This fixture profile is a reproducible test configuration. It is not a product throughput or latency threshold and does not establish persistence or provider delivery.
