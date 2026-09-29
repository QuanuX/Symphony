# SQMV Metadata C++ Specification

## Exact identity and extent

`sqmv-metadata-cpp` `0.2.0-dev` is an SQMV-owned C++26 static library in namespace
`symphony::sqmv`. Its source interface is [metadata.hpp](include/symphony/sqmv/metadata.hpp).
The exact package and compatible compiler/runtime select the interface for
trusted callers in one address space. There is no unspecified cross-toolchain
ABI, resident service, process entry point, provider parser, authorization
system, catalogue publication, or persistence operation.

This profile resolves exact manifest bytes into a bounded immutable description
and binds that description to SQFV. It does not implement the entire SQMV domain
of source semantics, schema interpretation, timestamp parsing, units, discovery,
or coverage analysis. Those remain with their actual owner contracts.

## Description and attribution

Six required, nonempty fields are exact opaque byte strings: `dataset_id`,
`dataset_revision`, `schema_version`, `layout_version`, `access_scope`, and
`producer_ref`. Embedded NUL and non-UTF-8 bytes are permitted. No normalization,
version ordering, latest-version lookup, identity issuance, or timestamp
substitution occurs. Dataset identity names the caller-selected lineage; its
revision identifies the exact caller-selected snapshot or view. Neither is
inferred from a path, ticker, source-native revision, or acquisition time.

Each `EvidenceReference` contains a role, nonempty `producer_ref`, and nonempty
`evidence_ref`. Supported role numbers are schema=1, layout=2, access=3,
source=4, time=5, coverage=6, lineage=7, and units=8. The first three roles each
require exactly one reference. This is the minimum interpretation evidence
declared by this profile, not a requirement imposed on all user modules.
Optional roles permit multiple distinct references. An exact duplicate tuple
is invalid. Omitting a role makes no assertion about that subject.

The description producer asserts the mapping of these references to the exact
dataset revision and representation. Each evidence producer identifies the
attributed origin of the referenced assertion. Strings are references already
selected by the caller; this module does not dereference them or establish their
authenticity, truth, entitlement, or applicability. The caller must resolve and
interpret the evidence before using it as semantic or authorization evidence.
In particular, access evidence does not grant access, time evidence does not
invent a missing source time, and coverage evidence does not certify a complete
dataset. An SQFV `source_binding` names a separate acquisition context and is
not silently equated to an SQMV source evidence reference.

Evidence order has no semantic meaning. Creation sorts tuples by numeric role,
then unsigned-byte lexicographic producer reference, then unsigned-byte
lexicographic evidence reference. A shorter equal-prefix string sorts first.
Resolution accepts only this strict order and rejects duplicates; it never
silently canonicalizes a noncanonical encoded input.

## Native API and lifetime

`Manifest::create(description, limits, out)` validates and copies caller input,
sorts evidence, computes the exact reference, and publishes one immutable owned
result. The caller can mutate or destroy its input after success.
`Manifest::resolve(bytes, expected_reference, limits, out)` requires the exact
expected reference, validates canonical structure and digest, and copies the
result before returning. There is no operation that selects whatever reference
untrusted input advertises. The caller supplies the expected reference from its
selected binding or other appropriate trust context.

`Manifest` is move-only. `retain(out)` explicitly shares the immutable state
without allocating. Retained handles remain valid after the original handle is
destroyed or moved. `description()` requires a nonempty handle; `reference()`
and `encoded()` return empty views on an empty handle. References and spans
remain valid only while their retaining handle remains live. Distinct live
handles may read concurrently. A caller must not destroy or move a particular
handle concurrently with its use. Self-retain and output aliasing of a source
manifest are supported.

All fallible operations are `noexcept`, return an enum-class `Status`, and
preserve every output on failure. An empty handle is `invalid_argument` for
status-returning methods. Allocation failure is `no_memory`; unexpected
internal exceptions are `internal_error`. No payload, evidence reference, or
secret material is emitted into diagnostics by this library.

## SQFV binding

`binding(out)` returns `sqfv::Binding` with this exact manifest reference,
dataset revision, schema version, layout version, and access scope.
`verify_binding(binding)` accepts only when all five fields match exactly.
Any mismatch returns `binding_mismatch`, with no conversion or version
negotiation. Dataset lineage, attribution, and all evidence are bound through
the exact manifest reference even though they are not duplicated inline in the
SQFV binding. Per-batch partition, source-native position, generation, sequence,
record count, payload ownership, and cursor mechanics remain SQFV concerns.
Equality is a compatibility check inside a trusted process; it grants no rights.

## SQM1 canonical encoding and exact reference

This is a local byte-defined encoding with no native struct layout or padding.
Its 16-byte prefix is:

| Offset | Width | Value |
| ---: | ---: | --- |
| 0 | 4 | ASCII `SQM1` |
| 4 | 2 | unsigned big-endian major version, exactly 1 |
| 6 | 2 | unsigned big-endian minor version, exactly 0 |
| 8 | 4 | unsigned big-endian required flags, exactly 0 |
| 12 | 4 | unsigned big-endian body length |

The body contains the six description fields in the order listed above, each
encoded as an unsigned 16-bit big-endian byte length followed immediately by
the raw bytes. Next is an unsigned 16-bit big-endian evidence count. Each
strictly sorted evidence tuple is its one-byte role followed by the producer
and evidence strings in the same length-prefixed representation. A 32-byte
binary SHA-256 digest follows the body; no trailing bytes are permitted.

The digest is SHA-256 of the ASCII/UTF-8 domain
`symphony.sqmv.metadata-manifest.v1` **including one trailing NUL byte**, followed
by the exact 16-byte prefix and canonical body, excluding the digest field.
The reference grammar is exactly `sqmv1-sha256-` followed by 64 lowercase
hexadecimal digits representing that digest. This is a local format reference,
not admission of a new Symphony colon namespace. The header version and domain
separate its interpretation from any other digest format.

Invalid reference syntax or invalid caller configuration is `invalid_argument`.
Unsupported major/minor, flags, or evidence role is `unsupported_manifest`.
Malformed lengths, truncation, trailing bytes, missing or repeated required
roles, unsorted or duplicate evidence, or a changed digest is `corrupt_manifest`.
A valid manifest with another valid expected reference is `reference_mismatch`.
Configured oversize is `limit`. Structural rejection may precede digest
verification; failure publishes no manifest. The digest establishes byte
integrity and identity, not a signature or authenticated source.

## Finite resource contract

Every operation supplies positive `max_manifest_bytes`, `max_field_bytes`, and
`max_evidence_refs`. Their technical ceilings are respectively **65,536 encoded
bytes**, **4,096 bytes per string**, and **128 evidence tuples**. The encoded
limit includes prefix and digest. Limits are checked before copying input or
allocating owned description state; decoding uses a fixed stack array of at
most 128 borrowed evidence tuples during structural validation. These ceilings
are local to this release and do not establish universal SQV capacities.

The immutable state owns the description, canonical bytes, and 77-byte reference.
Description strings collectively cannot exceed the encoded bound, and the
vector has at most 128 entries. The reused foundation SHA-256 helper makes
bounded temporary copies of the at-most-64-KiB canonical input; no payload or
knowledge-engine envelope is involved. The module has no process-global cache,
registry, queue, hidden catalogue, or independent engine context. Callers control
the number of retained manifests and any aggregate workload budget. Allocation
failure rolls back all partial state; retaining and releasing an existing
manifest require no new allocation.

## Focused acceptance

The native tests cover an independently generated golden digest and length,
canonical sorting, binary strings, immutable retained lifetime, aliasing,
all five SQFV binding fields, exact expected references, every truncated frame,
digest-valid noncanonical and missing-role inputs, version/flags rejection,
exact configured limits, and injected failure at every allocation point in
create, resolve, and binding. Installed consumer tests exercise the public
interface through exact receipt-owned packages. These are bounded development
checks, not proof of provider fidelity, full SQMV semantics, or a supported
platform/performance release matrix.

## Coordinated local pipeline admission

This development package admits the exact SQV-20 dependency chain. The native
representation/codec rules remain as specified above. The six-owner offline
pipeline verifies attributable capture, conversion, preview, asynchronous local
retention and confirmed replay; this is not a new provider or transport claim.
