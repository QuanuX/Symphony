# SQAV Capture C++ Specification

## Exact scope

`sqav-capture-cpp` `0.1.0-dev` is a C++26 static library exposing
`Symphony::SqavCapture` and `symphony::sqav`. It captures caller-supplied original
bytes and attributable acquisition evidence in an immutable optional object.
It implements no provider connection, file reader, clock sampling, schema parser,
credential lookup, retry scheduler or source authority verification. Source
content is data; the library performs no command evaluation or instruction dispatch.

The exact development target is trusted same-process C++26 on the tested macOS
host. SQMV 0.1.0-dev and SQFV 0.2.0-dev provide the metadata/flow bridge; the
knowledge-engine foundation 0.2.0-dev supplies SHA-256. SQPV and SQDV are test
composition dependencies only; using capture does not require retention or delivery.
Other representations and user-written source adapters remain selectable.

## Source and observation meaning

`Source` contains twelve required exact opaque byte strings, in this order:
provider reference, interface reference, interface version, acquisition operation, adapter reference,
adapter version, dataset ID, dataset revision, selection reference, native schema
reference, native encoding reference, and access scope. A version string may name
an exact dated observation of an unversioned interface. The library does not
resolve these assertions or silently select a version. References must be safe
identifiers or evidence references; secret credentials do not belong in them.

`Description` adds required attempt ID and attribution reference, an optional
opaque source position, coverage state/scope/evidence, optional source record
count, and timestamp evidence. Empty source position means absent. A present
zero record count differs from an unknown count. A retry can have a distinct
attempt ID while retaining the same source identity. No provider ordering is
inferred from attempt ID or source position; neither is a transfer sequence.

Coverage is `unknown` (0), `complete` (1), `partial` (2), or `gap` (3). Scope is
always required, and non-unknown coverage requires an evidence reference.
These are attributable caller assertions about that scope, not verified source
completeness, source authenticity, access grants or proof of publication rights.
Empty original bytes are permitted, including gap or empty-response captures.

Time roles are acquisition (1), event (2), publication (3), revision (4), and
receipt (5). Exactly one acquisition role is required; each other role is optional
and unique. Each time has required value, format reference, clock/calendar
reference, precision reference and evidence reference. All remain opaque: a date
is never promoted to an instant, no precision or timezone is invented, and no
ordering or semantic consistency is inferred. Provider-specific adapters must
validate their selected time semantics. The immutable description sorts roles
numerically; input vector order has no meaning. Strings may include NUL and
non-UTF-8 bytes. Nothing is normalized or interpreted as instructions.

## SQA1 capture encoding

All integer fields are unsigned big endian, with no padding. The exact header is
24 bytes: ASCII `SQA1`, u16 major 1, u16 minor 0, u32 flags 0, u32 metadata byte
length, u64 original byte length. Unsupported major/minor/flags are rejected.

Metadata follows in this order:

1. Twelve Source fields in the order above, each u16 byte length plus exact bytes.
2. Attempt ID, attribution reference, source position, with the same string grammar.
3. Coverage u8; scope string; coverage evidence string.
4. Count-presence u8 (0 or 1); count u64 (must be zero when absent).
5. Time count u8 (1 through 5); each time is role u8 followed by its five strings
   in value/format/clock/precision/evidence order. Roles must be strictly ascending,
   beginning with acquisition.

Exact original bytes follow metadata. The trailer is the 32 raw SHA-256 bytes
of the entire preceding header, metadata and original bytes. No compression,
text decoding, provider-record rewriting or trailing extension is admitted.
`reference()` is `sqac1-sha256-` plus 64 lowercase digest digits. Resolve requires
the expected reference and rejects a valid object with a different reference.

`source_reference()` is `sqas1-sha256-` plus SHA-256 of the six bytes `SQAS1\0`
followed by the twelve encoded Source fields, including their length prefixes.
Attempt, observation position, times, coverage and original bytes do not change
source identity; they do change capture identity. These are local reference
formats, not newly allocated Symphony colon namespaces or authenticated claims.

## Bounds and failure behavior

All three limits are mandatory and positive. Maximum encoded capture is 64 MiB,
metadata is at most 65,536 bytes, and each field is at most 4,096 bytes. Caller
limits may be lower. Header, metadata, original bytes and trailer all count toward
the capture ceiling. There are at most five timestamp entries. Before creation
allocates, it checks every field, count, required role and total size. Resolve
checks total size, fixed header, lengths, strings, counts and canonical order
using bounded views before allocating parsed state or hashing. Every subtraction
is guarded and every encoded byte is accounted for. No input controls an
unbounded queue or implicit decompression.

Handles own a shared immutable description and encoded byte vector. `original()`
is a span into that vector, not a second retained payload. Explicit `retain`
shares immutable state without allocation. Resolve makes its own owning copy.
Create copies the caller's original bytes; the caller must keep inputs stable
for the duration of each call. Borrowed views require their retaining handle to
remain alive and unmoved. Empty handles return empty views; `description()`
requires a nonempty handle. All fallible public methods are `noexcept`; failure
preserves output handles and publishes no partial object. Allocation failures
return `no_memory`. Output/input handle aliasing is supported because publication
occurs only after all input reads and fallible work complete.

Limits bound object lengths, not process RSS. Description copies, SHA-256 scratch
(the current foundation copies and pads its input), source-hash scratch, object
control blocks and allocator overhead are separate local control/preparation
costs. Preparing an SQFV batch copies the encoded capture into that context's
charged allocation budget. The Capture object itself is not charged to SQFV.
Callers govern aggregate live Capture handles and concurrent operations; this
release introduces no process-global budget or hidden background cache.

## Exact metadata and flow bridge

`prepare(context, manifest, position, out)` requires manifest dataset ID, revision
and access scope to equal Source fields. Both schema and layout must be exactly
`sqav-capture-v1`. Manifest evidence must contain a source reference with the
capture's exact source identity and the description's attribution reference as
producer. This describes the optional capture envelope; the native provider
schema/encoding remain explicit inside it.

The caller selects partition, nonzero producer generation and transfer sequence;
SQFV validates these and the context's actual resource bounds. The SQFV descriptor
sets source binding to source identity, source position to capture identity, and
record count to **one capture envelope**, independently of the optional provider
record count and provider-native position inside the capture. SQMV supplies the
exact immutable binding. SQFV owns the copied immutable payload and its leases.

`from_delivery(bytes, descriptor, manifest, limits, out)` accepts actual payload
and descriptor views from a selected SQFV/SQDV delivery. It verifies the complete
SQMV binding and envelope count, decodes against descriptor capture identity,
checks manifest/source/attribution consistency, and verifies descriptor source
identity. It does not authenticate caller-supplied descriptors or certify that a
recipient processed the data. SQDV continues to own processing acknowledgement;
SQPV continues to own retained commit evidence. Failed bridge operations preserve
outputs; failures during SQFV preparation release temporary charged reservations.

## Acceptance and remaining scope

Focused native tests cover exact bytes and time-role fidelity, malformed/truncated
input, canonical integrity, limits, independent identity fields, output rollback,
allocation failures and real five-owner retention/delivery/replay. A standalone
installed consumer checks round-trip and incompatible/corrupt-input rejection.
An independent encoder checks the exact binary format and both references.

This is offline capture infrastructure. Exact FRED/ALFRED, Databento, news and
SCABV-owned IBKR operations, parser validation, request limits, reconnect/backfill,
credentials, source access enforcement and provider conformance remain future
contracts. No sample is assumed to be a particular format before inspection.
SQTV conversion and SQV qxctl surfaces remain subsequent work. No throughput or
broader platform-support claim follows from these focused fixtures.
