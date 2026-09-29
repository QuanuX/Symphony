# SQTV Exact Integer Conversion C++ Specification

## Identity and admitted domain

`sqtv-integer-conversion-cpp` `0.2.0-dev` is a C++26 static library exported as
`Symphony::SqtvIntegerConversion`, namespace `symphony::sqtv`. It depends on
SQFV `0.3.0-dev`, SQMV `0.2.0-dev` and the knowledge-engine foundation `0.2.0-dev`.
Its initial verified development platform is macOS amd64, AppleClang 21.

The input semantic schema is exactly `sqtv-integer-values-v1`: a nonempty dense
array of mathematical integers, without nulls, missing values, sentinels, padding,
headers or record framing. Layout IDs are `sqtv-int-v1-{i|u}{8|16|32|64}-{le|be}`.
`i` is two's-complement signed; `u` is unsigned; `le`/`be` specify byte order.
All 16 layouts and 256 ordered pairs are admitted. An 8-bit layout still carries
its declared byte-order identity. All bit patterns represent integer values.
Units and time evidence remain opaque and unchanged; this operation never
rescales values, changes units or interprets integer timestamps.

`Format` encodes width in bits, Signedness signed=1/unsigned=2, ByteOrder
little=1/big=2. Invalid enum values or widths are unsupported. `layout_id`,
`parse_layout` and `operation_reference` preserve output on failure.

## Conversion and publication

`Result::convert` requires a live immutable SQFV Batch and SQMV Manifest whose
binding verifies exactly. It acquires an input lease using that manifest's scope.
Input record_count is the element count; payload size must equal count times
input width in bytes. Empty input or size/count disagreement is malformed.

Every input value must fit the target mathematical range. Negative-to-unsigned,
unsigned-to-signed above the signed maximum, and out-of-range narrowing return
`overflow`. Widening and byte-order conversion preserve value. No clipping,
rounding, wraparound or partial output is permitted. Validation scans the complete
batch before allocating the output vector; encoding makes a second linear pass.
Unsigned arithmetic handles all extrema including INT64_MIN and UINT64_MAX.

The caller selects an output Context and Position (partition, producer generation,
batch sequence) subject to SQFV's exact validity rules. Output may use a different
Context within the trusted process. The source batch remains immutable. A new
manifest and batch are staged locally and Result is replaced only after both
succeed. Input may alias the previous Result's batch and metadata. Every failure
preserves that previous Result and releases temporary reservations.

## Derived metadata and identities

The derived manifest preserves dataset ID/revision, semantic schema, access scope,
and all evidence except layout-role entries. This is a representation change
within the same dataset revision, not an analytic derived dataset. It replaces
layout_version with the selected exact target layout, producer_ref with
`sqtv-integer-conversion-cpp/0.2.0-dev`, and layout evidence with one entry whose
producer is that converter identity and reference is the target layout ID.

Three lineage entries under the same converter producer identify the input
manifest reference, input batch content reference and operation reference. Exact
preexisting identical lineage entries are deduplicated. All other input lineage
is retained. Input manifests remain separately resolvable by the caller; a
reference is not an embedded copy or a storage guarantee.

Operation reference is `sqtv-int1-sha256-` plus lowercase SHA-256 hex of the ASCII
domain `symphony.sqtv.integer-conversion.v1`, one NUL byte, then six bytes:
input width, input signedness, input order, output width, output signedness,
output order. Version v1 fixes these exact mathematical rules. Output transfer
position and resource limits are not operation parameters. Input reference is
`sqtv-input-sha256-` plus lowercase hex of the actual 32-byte SQFV ContentId;
it is not an additional hash. That ContentId binds the input descriptor and bytes.

The output descriptor uses the derived manifest's exact binding, caller Position,
unchanged element count, source_binding=operation reference and
source_position=input reference. Original provider position and attribution
remain reachable through the input batch/manifest lineage; they are not reused
as output transfer sequence.

## Bounds, lifetime and failure

All Limits fields are positive: max_elements at most 8,388,608, max_input_bytes
and max_output_bytes each at most 64 MiB. Count and multiplication bounds are
checked before payload scanning or output allocation. These are explicit per-call
ceilings. The output vector is temporary workspace separate from the SQFV-owned
copied output; peak payload allocation can therefore include both, plus input
ownership and bounded metadata/identity scratch. SQFV separately enforces its
Context allocation budget. There is no global cache or aggregate process budget.
Derived SQMV manifests are limited to 65,536 encoded bytes, 4,096 bytes per field
and 128 evidence entries. Exhausted lineage capacity fails with `limit`; evidence
is never silently discarded. All allocations and owner statuses are checked.

Status distinguishes invalid arguments, unsupported representation, binding
mismatch, malformed input, overflow, limits, allocation failure, stale/closed
owners and internal failures. When multiple conditions fail, validation order is
not a public diagnostic precedence guarantee. Exceptions do not cross public
noexcept functions. A successful Result owns its Batch and Manifest. batch() and
metadata() require a nonempty live Result; views expire with owner replacement,
move or destruction. Existing Batch/Manifest retain APIs provide independent
handles. Caller synchronization must prevent racing move/destruction with reads.
Stateless conversions on independent objects add no shared mutable state.

## Verification and scope

Native fixtures compare all layout pairs at integer boundaries against C++ typed
range checks and object-representation bytes, round-trip accepted conversions,
verify lineage and aliasing, reject unsupported/malformed/bounded inputs, compose
SQPV/SQDV retained delivery, and inject allocation failures. An installed-only
consumer verifies an independently calculated operation digest and exact bytes.
Receipt-v2 owns 13 package files; lifecycle tests cover exact installation and
guarded removal. No network, provider parser, Databento fidelity, decimal/float,
nullable, lossy conversion, stateful operation, sharing cache, foreign ABI,
performance guarantee or SQV qxctl surface is admitted by this release.

## Coordinated local pipeline admission

This development package admits the exact SQV-20 dependency chain. The native
representation/codec rules remain as specified above. The six-owner offline
pipeline verifies attributable capture, conversion, preview, asynchronous local
retention and confirmed replay; this is not a new provider or transport claim.
