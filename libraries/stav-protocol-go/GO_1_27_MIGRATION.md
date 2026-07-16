# Go 1.27 Confirmed-Release Migration

## Current and Target State

- Production/CI baseline: Go 1.26.5.
- Intended target: Go 1.27 after general availability.
- Wire contract: unchanged by the toolchain migration.

The root `go.work` composes the kernel and its consumers from one reviewed source tree. The operational durability/IPC increment declares kernel `v0.2.0` and append-authority `v0.1.0` as coordinated release tags; qxctl and SSIAG pin those real module versions for independent source installation. Publish the tags only from the reviewed merge tree, then verify each module once with `GOWORK=off`. The Go 1.27 pin change remains a separate gate.

Draft Go 1.27 documentation may inform experiments but is not an implementation dependency. No draft package or behavior is exposed through the kernel API.

## Expected Implementation Review

The current Go 1.27 draft introduces `encoding/json/v2`, `encoding/json/jsontext`, and the root `uuid` package. These names and behaviors must be rechecked against the final release. They are optional internal substitutions. The existing implementation remains authoritative unless a replacement passes every behavioral gate.

Go 1.27 is also expected to back the existing `encoding/json` v1 API with the v2 implementation by default. That means the kernel changes underneath even without an import edit, and exact native error text may differ. The migration must therefore test both the Go 1.27 default and `GOEXPERIMENT=nojsonv2` while that opt-out exists. STAV never places native parser errors on the wire and tests must assert stable STAV error classes, not standard-library wording.

The final platform matrix requires explicit review. The draft raises the Darwin minimum to macOS 13. A Go 1.27 pin therefore cannot be ratified for TOPS nodes until Symphony either declares macOS 13+ as the supported floor or provides a separately governed legacy-build lane.

The migration is expected to change:

- `go` versions in the root `go.work` and each `go.mod`;
- CI toolchain matrices and cross-build images;
- possibly private parser/UUID adapter internals;
- the supported Darwin deployment floor if the final release retains macOS 13+;
- `go test`/vet evidence and `go mod tidy` output if the final tool retains its draft behavior;
- release documentation and the compatibility record.

It must not change:

- schema files or identifiers;
- accepted or rejected fixture sets;
- canonical JSON bytes or SHA-256 digest vectors;
- frame format or limits;
- exported kernel function/type behavior;
- qxctl grammar or any authorization boundary.

## Gate Procedure

1. Confirm `go1.27` general availability and read final release notes, platform support, security notes, JSON/UUID behavior, and module/workspace changes.
2. Add Go 1.27 as a non-blocking experimental lane while Go 1.26.5 stays required.
3. On Go 1.27, run the corpus once with the default JSON implementation and once with `GOEXPERIMENT=nojsonv2` if the final release still provides that opt-out. Compare both with Go 1.26.5.
4. Run unit, race, fuzz-seed, fixture, digest, partial-frame, vet, and cross-platform build tests on both versions.
5. Require byte-for-byte equality for every canonical output and digest.
6. If evaluating `encoding/json/v2`, `encoding/json/jsontext`, or `uuid`, keep the current implementation available for differential tests until equality and strict rejection are demonstrated.
7. Confirm the final Darwin floor against the supported TOPS node matrix before changing the production pin.
8. Update `go.work` and all `go.mod` files atomically; do not leave mixed production pins. Run Go 1.27 `go mod tidy` and review its diff rather than accepting mechanical changes blindly.
9. Make Go 1.27 required, retain Go 1.26.5 as a temporary compatibility lane, and record any compiler-only changes.
10. Remove the older lane only after the supported TOPS target matrix is green and an owner-reviewed release is cut.

Official draft reference, to be replaced by the final release record: https://go.dev/doc/go1.27

## Failure and Rollback

Any canonical-byte, digest, validation, platform, or race-test difference blocks the pin change. Revert internal substitutions first; if the toolchain itself remains incompatible, keep Go 1.26.5 as production while continuing an experimental Go 1.27 lane. A Go release never authorizes a STAV protocol version change.
