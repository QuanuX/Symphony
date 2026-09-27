# SQAV Databento DBN Specification

## Exact implemented scope

`sqav-databento-dbn-cpp` `0.1.0-dev` exports the C++26 static-library target
`Symphony::SqavDatabentoDbn`, namespace `symphony::sqav::databento`. Runtime
requires SQAV capture 0.1.0-dev, SQMV 0.1.0-dev, SQFV 0.2.0-dev and the shared
foundation 0.2.0-dev. Initial verified platform: macOS amd64 / AppleClang 21.

This release admits **uncompressed DBN version 3, single-schema MBO files**.
It parses the provider layout directly using explicit little-endian loads;
unaligned input is supported without casting bytes to native structs. Databento
C++ `v0.68.0` declarations and public fixtures anchor conformance. The SDK is a
source/fixture reference, not a runtime dependency or network-support claim.
Provider documentation reviewed 27 September 2026:

- https://databento.com/docs/standards-and-conventions/databento-binary-encoding
- https://databento.com/docs/schemas-and-data-formats/mbo
- https://github.com/databento/databento-cpp/blob/v0.68.0/include/databento/record.hpp
- https://github.com/databento/databento-cpp/blob/v0.68.0/tests/src/dbn_decoder_tests.cpp

No older/newer version is silently upgraded. DBZ/Zstandard, mixed live schema,
other record types, self-describing schema definitions and network I/O are outside
this release. Full-book data means MBO events, not a reconstructed or complete book.

## Bounded structural inspection

`FileView::inspect` accepts immutable borrowed bytes and mandatory positive Limits:
file bytes at most 64 MiB, metadata bytes including the 8-byte prefix at most
1 MiB, and records at most 1,048,576. It returns a borrowed view only after the
complete file validates. Failure preserves the prior view. It allocates no memory.
The caller keeps storage alive and unchanged during every view/field access;
mutation, move, destruction or concurrent writes invalidate that contract.

The prefix must contain DBN magic, version 3 and a bounded metadata length.
Total metadata is at least 128 bytes and 8-byte aligned. Dataset is a nonempty
NUL-terminated 16-byte field; schema is MBO (0). ts_out must be 0 or 1. Declared
symbol width is positive and at most 4,096 bytes. The future schema-definition
length must be zero. Symbol/partial/not-found lists and mapping/interval arrays
are traversed with division-based remaining-byte checks before loops. Fixed
strings must contain a NUL. Mapping dates, symbol bytes, symbology enums and
reserved padding remain uninterpreted and preserved. Final alignment padding may
occupy at most seven bytes. This is structural parsing, not date/symbol semantic
resolution or validation of every reserved byte.

Metadata exposes dataset, raw start/end/limit, input/output symbology bytes,
optional-gateway-timestamp flag, symbol width and list/mapping counts. The encoded
metadata view retains every mapping interval and original byte. Record count is
computed from the actual body, separately from the request's maximum-record limit.
A metadata-only file is valid and has zero records; EOF at a complete record
boundary may be valid even if the original file had further records. No checksum
or expected length in DBN proves complete transport. Capture adds an exact
integrity identity to whichever complete bytes the caller supplies.

## MBO fields and time semantics

Every record is exactly 56 bytes, or 64 with appended ts_out, with matching
length-in-32-bit-words and rtype 160. Mixed/control records are unsupported.
`record(index)` rejects out-of-range indices. `decode_mbo` validates one record
under the same DBNv3 layout. Both preserve their output on failure and allocate
nothing.

Exposed fields preserve publisher/instrument/order IDs, event and receive
nanoseconds, signed integer price, quantity, flags, channel, action, side, signed
input timestamp delta, venue sequence and optional gateway timestamp. Price
remains fixed-point units of 1e-9. Unknown action/side values and flag bits remain
raw bytes; they are not interpreted as supported book operations. Undefined
price/time sentinels remain exact integer values. No floating-point conversion,
time subtraction, rescaling, event sorting, sequence-gap inference or snapshot
completion inference occurs. Venue sequence and gateway/event/receive timestamps
remain distinct from Symphony transfer sequence and acquisition time.

## Owned capture bridge

`capture_file` fully inspects input first. Source provider_ref must be `databento`,
dataset_id must exactly match file metadata, native_schema_ref must be
`databento:mbo`, and native_encoding_ref must be
`databento:dbn-v3-uncompressed`. A supplied source_record_count must equal the
computed count. A complete-coverage assertion conflicts with nonzero partial or
not-found symbol counts and is rejected. Other coverage statements remain caller
assertions; absence of those lists does not prove completeness.

The adapter sets source.adapter_ref=`sqav-databento-dbn-cpp`, adapter_version=
`0.1.0-dev` and source_record_count to the actual count. Other source fields,
acquisition evidence, coverage and access scope retain their caller attribution.
Caller evidence string sizes and time-vector count are bounded before copying;
SQAV's own exact validity and encoded-size rules then apply. Its original payload
is the **entire unchanged DBN file**, including metadata and mapping bytes. All
allocation failures preserve the previous Capture. `inspect_capture` verifies
that exact provider/adapter/schema/dataset/count binding before returning a view
borrowed from the live Capture. The SQAV/SQMV bridge still carries one capture
envelope per transfer record; provider record count is a separate inner fact.

## Package and evidence

Receipt-v2 owns 13 files: archive, header, four CMake exports/configuration files,
six documents and license. No executable, credential resolution, service or SQV
qxctl command is installed. Native tests additionally compose SQPV/SQDV. Public
fixture fields are checked against upstream expected values; truncation, bounds,
raw extrema, unaligned inputs, optional ts_out, zero-allocation inspection,
allocation rollback and retained replay have focused evidence. The independently
installed consumer exercises the same public fixture through exported C++26.

No authentication, entitlement, transport completeness, source authenticity,
redistribution, live reconnect, symbol resolution, book state, reference-data
client, broad provider conformance or performance guarantee follows from parsing.
