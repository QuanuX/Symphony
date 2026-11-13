# Empirical trade-price public SDK example

This independently buildable qualification project exercises two existing public
interfaces: a standalone process linked to the installed foundation SDK, and an
SBV model provider loaded through the installed SBV SDK C ABI. It adds no
Symphony owner, registry entry, generic loader or provider deployment feature.
The same qualification author wrote both routes; this is not an independent
human author or documentation-only agent acceptance record.

## Inputs and exact calculation

The example accepts 1–64 canonical decimal integer prices in nanos, each between
-10^15 and 10^15 inclusive. It returns the sorted empirical distribution with
reduced rational weights and the exact rational mean. The Go qualification
harness uses arbitrary-precision `math/big.Rat` probability accumulation and
`big.Int` price totals as an independent complete numerical oracle. It does not
reuse the example's bounded integer arithmetic. No tolerance, market forecast, calibrated price model, fill-probability
or performance claim follows from this deliberately synthetic calculation.

The provider reads only trades at or before the current signal, using a bounded
window of four prices in the campaign. It polls cancellation, retains no host
pointers, and releases ABI-owned output explicitly. The public model context
wraps the signal in its `signal` member. A first development attempt incorrectly
read the signal ID at the outer level; it failed before producing a result and
was corrected against the published SBV contract. Preserve that first failure in
an author journey instead of presenting a flawless independent discovery.

## Reproduce outside the source checkout

Copy this entire directory to a private directory outside the Symphony checkout.
Obtain the **complete**, matching public receipt units for foundation
`knowledge-vector-engine-cpp/0.2.0-dev` and SBV `sbv-engine/0.33.0-dev` plus their
approved runtime closure. Do not extract selected headers from the receipts.
Use CMake >=3.31, a C++26 compiler and Go 1.26.5. Preserve the C++
driver name (for example `clang++`) when selecting a compiler symlink; resolving
it to `clang` changes linker driver behavior. The Go harness is qualification
tooling, not a product runtime dependency. Select absolute paths and a new private
work directory:

```sh
GOWORK=off GOENV=off GOTOOLCHAIN=local go run qualify.go --sdk-prefix /absolute/installed-sdk \
  --cxx /absolute/clang++ --work /absolute/new-example-campaign
```

The campaign copies only these public sources, uses explicit installed CMake
imports, clears include/loader overrides, hashes the complete SDK receipts and
owned files before and after, retains generated compile/dependency/link records,
and records loader dependencies. On a final fresh host, the original checkout,
SDK build prefixes and compiler caches must be absent. A local development SDK
run does not establish final-download relocation or fresh-host isolation.

Both compatible example versions (0.1.0 and 0.1.1) build and install under the
private campaign prefix. The SDK client invokes the actual installed SBV C ABI
for census, evaluation and native result inspection. It then reads the complete
result, checking its byte hash and length against the native inspection. The
`result_export` convenience operation belongs to qxctl; it is not a native SDK
operation. An early harness attempt confused these routes and was corrected
against the installed templates; its refusal is retained. Every execution outcome is
compared against a complete expected record, including all numerical support,
provenance and explicit unavailable fields. The standalone route calculates the
same causal windows independently of the provider invocation.

Retained refusal cases cover missing/wrong SDK, malformed/oversized inputs,
identity/schema/config changes, expired deadline, changed source/library digest,
invalid ABI tables, cancellation, interrupted incomplete process input and
changed owned bytes. Side-by-side compatible rollback and removal preserve an
unrelated user-data sentinel and the original SDK. Removal is only for this
example's exact two-file inventory; it is not a product uninstaller.

The committed-output scenario discards an already received acknowledgement,
reopens the durable result, and proves a same-output retry refuses without
replacement. It is a retained-result/no-overwrite fixture, **not an injected
transport failure**. Use the owner fault campaign for actual lost-response
acceptance. The SIGTERM check proves incomplete input emits no completion; it
does not prove arbitrary mid-calculation cancellation or power-loss durability.

## Administrative-stall and independent final-host procedure

After final artifacts and hosts are selected, an independent operator records
receipt, candidate, policy and loader identities and verifies baseline complete
outputs. For these local public SDK routes, confirm the declared absence of a
synchronous administrative-service dependency. In a disposable installation,
stop only the selected administrative service through its documented owner
procedure, record that it actually stopped, repeat both routes with identical
inputs, and compare complete results and provenance. Restart and verify the
service through its owner. Do not interrupt a user service or invent an owner
control command. If no separate service is selected for the tested composition,
retain that fact for reviewer applicability instead of reporting an injected
stall. No such service interruption is performed by `qualify.go`.

A separately assigned author/agent receives only the final SDKs, this directory
and shipped documentation. Record first failures, questions, all setup steps,
UE1–UE6 results and compatible rollback. Attach complete logs and outputs to the
artifact-bound acceptance packet. Final artifacts, independent authorship,
documentation-only agent work and administrative-stall observations remain
pending until that campaign is executed and reviewed.

`evidence.json` describes private installed-SDK fixture evidence and always leaves
release acceptance and publication false. No live provider or network is used.
The attributed synthetic DBN fixture helper is under `fixtures/`.

Focused harness checks exercise exact arithmetic beyond int64 aggregate range,
receipt tampering and unsafe members, command refusals, and preflighted example
removal. Invoke the explicit Go files so the adjacent C++ example sources are
not treated as Go package build inputs:

```sh
GOWORK=off GOENV=off GOTOOLCHAIN=local go test -race -count=1 qualify.go qualify_test.go
```
