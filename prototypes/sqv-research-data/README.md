# SQV research-data native prototype

Status: historical development precursor, 26 September 2026. The admitted
SQFV implementation is [sqfv-batch-cpp](../../modules/sqfv-batch-cpp/MANIFEST.md).
This prototype remains an offline fixture, without an installed package,
admitted wire schema, qxctl command, or release.
The source whitepaper and its decision register are design input; this code's
tests are the evidence for the narrower implemented behavior below.

## Implemented slice

- Native C++26 preparation copies caller bytes into an immutable allocation.
- Exact in-process access-scope equality grants a read lease. Multiple trusted
  readers share the same allocation; the allocation remains live until the
  producer and every reader release their handles.
- A deterministic development frame records dataset revision, schema name and
  version, representation, scope, provenance, opaque source position, producer
  generation, partition, internal cursor range, record count, and raw payload.
- The frame uses big-endian integers, an explicit version and lengths, 128-byte
  maximum per ASCII metadata field, a 4096-byte maximum header, and a 1 MiB
  maximum payload. Unsupported versions, malformed/truncated/trailing frames,
  invalid metadata, and inconsistent cursors fail before payload allocation.
- In-process fan-out uses independent per-consumer byte budgets and pending
  limits. A blocked or oversize offer is an explicit result. A taken delivery
  retains its byte credit until its lease is destroyed. Exact access-scope
  mismatch is rejected before sharing.

The opaque source position is never equated with the internal cursor. Distinct
dataset revisions and schema versions remain distinct. There is no implicit
upgrade from an old provider or schema binding to a newer one.

## Focused verification

```sh
cmake -S prototypes/sqv-research-data -B /private/tmp/sqv-g02-build -DCMAKE_BUILD_TYPE=Debug
cmake --build /private/tmp/sqv-g02-build --parallel 2
ctest --test-dir /private/tmp/sqv-g02-build --output-on-failure
```

The original C++20 prototype's two native tests passed on macOS x86_64 with
AppleClang 21, both in Debug and under AddressSanitizer/UndefinedBehaviorSanitizer.
The retained source now requires C++26. The tests check original
input mutation after publication, two-reader shared allocation and lifetime,
concurrent read consistency, scope rejection, exact encode/decode, all
truncated-frame lengths, malformed
version/length/metadata, oversize payload, and distinct revision/version
identities. The flow test checks independent branches when one is blocked,
credit retention during processing, credit return on release, oversize and
scope rejection, independent pending-queue limits, and a taken delivery
outliving its Fanout owner. No provider
session, persistence, IPC, remote transport, performance threshold, or secret
handling is claimed.

## Evolution boundary

The development frame has no cryptographic integrity, authentication,
compression, cross-process handles, or stable C++ ABI. Scope equality is only
an in-process guard for trusted code. A reader must keep its lease while using
the returned span. Those boundaries need explicit contracts and negative tests
before any cross-process or private-data use. The 1 MiB limit is a prototype
input bound, not an SQV-wide batch limit or workload profile.

Ports must be configured before concurrent use. Per-port accounting does not
yet provide a global allocation budget or reserved control capacity. A blocked
offer remains the caller's explicit responsibility; this prototype provides
no catch-up, durable position, or silent-loss policy.

The prototype remains outside canonical registries. Current SQV child ownership
and the admitted module contract are recorded separately; this earlier frame
and its limits do not supersede them. No command or feature status is inferred
from this directory.
