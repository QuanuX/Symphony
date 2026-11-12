# SRRAPIV local verification record

## Scope and provenance

Recorded 2026-11-06 for the local implementation working tree. These results concern the exact exercised source and local targets. They do not establish hosted CI results, compatibility with another SDK release or qualification of another platform.

## Completed local checks

- The user-supplied R API+ C++ 13.7.0.0 SDK compiled and linked with the native bridge on Darwin Intel using strict first-party compiler warnings. The selected header identity is `sha256:b996e76fbfe20c7f4d7ba2d645c45cb13d6b19df37d7522125e94e61e74df5b4`; vendor binaries remain external to the repository.
- Independent compiler AST inspection accounted for 118 REngine signatures, 91 RCallbacks, 3 AdmCallbacks and 1,231 direct public data fields. The timer adapter adds the 95th callback. Full typed dispatch compilation checks native signatures; it does not execute those provider operations.
- SDK-linked offline fixture tests passed under AddressSanitizer. They exercised all 95 callback adapters, native local object/version utilities, quantity encodings, bounded codecs, presence handling, shared array-count refusal and optional-null arrays. Callback fixtures are synthetic data, not provider observations.
- The catalog suite passed 90 focused assertions under AddressSanitizer and UndefinedBehaviorSanitizer with C++26 and strict warnings. It covers exact immutable selection, compatibility evidence scope, version coexistence, maintenance comparison, directional reader declarations and refusal behavior.
- The native runtime and test-only slow-worker checks passed locally. The slow fixture exercises responsive control while an invocation is blocked, duplicate-attempt recovery, refusal of stop during active invocation, and asynchronous shutdown. These two checks use the local control profile and do not exercise a live provider.
- Installed Go/qxctl integration checks passed locally for exact native routing and control/schema handling.

The final SDK-enabled CMake gate passed all four tests: catalog, native runtime, slow-worker supervision and SDK-linked vendor fixtures (9.04 seconds total on the exercised host). Source-validator checks reported 18,211 passes, zero violations and four existing historical-path warnings; the generated README projection remained current. These are repository and local implementation evidence, not SDK conformance.

## Linker observations

The vendor Darwin archives produced linker warnings that some objects lack a platform load command and are being treated as macOS objects. Linking succeeded on the exercised Darwin Intel target. This observation does not establish archive compatibility with other macOS releases, Apple Silicon, Linux or another toolchain; those require exact target builds and tests.

## External qualification not established

The checks did not construct REngine, log in, submit provider requests or place orders. They establish local source, type, serialization and control behavior only. Actual credentials, account entitlements, vendor session behavior, reconnect recovery, order acknowledgement/reconciliation and Rithmic conformance remain subject to separately authorized testing.

The checked-in CI workflow declares SDK-free Ubuntu x86_64, Ubuntu ARM64 and macOS Intel jobs. Hosted execution was not awaited for this local record. Proprietary SDK redistributability, package delivery and continued server support for retained client versions are separate concerns.

## Reproduction surfaces

- `tests/catalog_test.cpp` and `tests/catalog_fixture.hpp`: immutable catalog and binding tests.
- `tests/runtime_test.cpp` and `tests/slow_vendor.cpp`: SDK-free local lifecycle, recovery and responsiveness tests.
- `tests/vendor_test.cpp`: SDK-linked offline callback and codec fixtures.
- `tools/README.md`: exact-header regeneration and SDK fixture compilation guidance.
- `CMakeLists.txt`: native test targets and explicit SDK selection.
- `.github/workflows/srrapiv.yml`: declared hosted SDK-free checks.

## CI integration follow-up

The initial PR revision `1c688724` passed the dedicated SRRAPIV workflow on Ubuntu x86_64, Ubuntu ARM64 and macOS Intel. Review of its completed runs found 20 failed jobs across the dependency-map, DuckDB and release workflows. Sixteen stopped at the shared component-map coverage check because SRRAPIV had not been registered. Four rejected the three unclassified installation-dependent Go test skips. The release identity catalog also needed the new component and owner before its downstream gate could run.

The correction registers the native dependency, owner, source identities, release candidate and SDK-free native/installed test selections. Its installed build requires SymphonyKnowledgeVectorEngine 0.2.0 EXACT and rejects source fallback. The resident-lifecycle Go test now creates its own synthetic offline binding from local binary discovery: its prior extra configuration requirement could silently skip this test in the dedicated workflow. Provider/trading admission is disabled and no credentials are supplied.

Focused follow-up checks passed on Darwin Intel: all three SDK-free native tests; ten top-level SRRAPIV Go tests and six subtests under the race detector with zero skips; the three release-result regression tests; exact imported-SDK proof, missing-SDK refusal, installed receipt validation and CTest discovery. Dependency-source, release-manifest and candidate-identity verification passed with 70 roots and 24 owner declarations. The repository validator retained 18,211 passes, zero violations and four historical-path warnings. This follow-up used Clang 23.1.2 and Go 1.27.1 locally; hosted jobs retain their declared Go 1.26.5 toolchain. The complete shared campaigns and hosted reruns are not claimed by these focused checks.

## Native feature-registration follow-up

Completed hosted runs for revision `f5c3f3b0dd1730ab65d53f349e5931788c6e4e61` passed eight of nine workflows. The remaining dependency-map workflow passed six jobs and failed eight: four knowledge jobs failed SSFV checks, and four quantitative jobs failed the SBV administration contract. Each failure reached the same native feature-registry validation path.

Three metadata defects affected the new SRRAPIV records. The qxctl record used `sqv` in the governance-only cross-vector vocabulary, causing rejection of the entire qxctl feature file and 90 cascading validation errors. Both SRRAPIV registry entries were outside the canonical-entries section. The native feature-file envelope also used noncanonical JSON key ordering. The correction uses the existing native-owner dependency relationship and evidence reference for SQV ownership, moves the entries into the canonical section, and canonicalizes the envelope. Record, registry, administration-profile and dependency-map digests were refreshed; registry prose now reflects 149 records across 67 owner scopes. No runtime behavior or accepted cross-vector vocabulary was changed.

Focused local verification on Darwin Intel with Clang 23.1.2 passed:

- A fresh build of the current SSFV engine and its two CTest checks: engine tests and process smoke.
- The SBV administration contract against the corrected current repository, using a retained SBV 0.33 test binary. Its administration test and SSFV reader sources match the current sources; this is not a fresh full SBV build or a complete SBV campaign.
- Dependency-source verification: 70 roots, 58 native components and 1,031 source hashes, including Go, Swift, interface and schema identities.
- Repository validation: 18,211 passes, zero violations and four existing historical-path warnings. The generated README projection remained current.

These checks establish the corrected native registration locally. Hosted results for the correction are not claimed, and no provider session or trading operation was performed.

## Installed Go evidence follow-up

Completed hosted runs for revision `e668dac29bc7af079620a07d4a18a67907e231e5` passed all four knowledge profiles. The four remaining failures in dependency-map run `42318885876` were quantitative jobs on Ubuntu 24.04 x86_64, Ubuntu 26.04 x86_64, Ubuntu 24.04 ARM64 and macOS 15. Each completed its declared native checks and the first installed Go suite (four top-level tests and three subtests), then stopped before the second suite because `installed-go/tools-qxctl.jsonl` already existed. Both selected packages belong to the same Go module; report names previously included only the module.

Installed report names now include the complete suite-selection digest, keeping event, stderr and summary files independent across packages and named-test selections. Exclusive file creation remains enforced. A new regression reproduced the original `file exists` failure before the fix, then passed with three real Go test invocations across two packages, including two distinct selections in the same package. It also verifies that an identical repeated selection is refused without altering any retained evidence.

All four runner regression tests passed locally with the race detector on Darwin Intel using Go 1.27.1. Release-manifest and dependency-source verification passed; repository validation reported 18,211 passes, zero violations and four existing historical-path warnings. These focused checks do not claim the complete installed campaign or a local exact-Go-1.26.5 run; hosted jobs retain that toolchain requirement. No provider session or trading operation was performed, and hosted reruns for this correction were not awaited.

## PR 195 shutdown observation follow-up

On 2026-11-12, revision `771d4b62afb20471af162901308a2e3af1eb2a9e` had one failed job and one cancelled job. The [Ubuntu 24.04 native job](https://github.com/QuanuX/Symphony/actions/runs/43274911512/job/131380637616) built successfully and passed catalog/runtime tests, then failed `srrapiv-runtime-supervision` with `runtime_status` reporting `srrapiv.transport.uncertain`. Its installed Go step did not run. The macOS Intel and Ubuntu ARM64 native jobs passed.

The owner can finish shutdown and close its listener after a status client connects but before replying. Previously only connection unavailability could select retained status. Status now also reads retained evidence after a lost reply, with the existing generation checks, unreachable state and unknown provider readiness. It never retries a request or converts uncertain mutation results into success. Without retained evidence it preserves the original error.

A same-UID socket fixture accepts a complete request and closes without replying. It reproduced the exact status error before the fix. The focused `srrapiv-runtime-supervision` CTest then passed on Darwin Intel with Clang 23.1.2 and the exact installed foundation 0.2.0. The fixture checks retained shutdown evidence, stale-generation refusal, uncertain stop/invoke results and preserved uncertainty when no record exists. Only this CTest and its required build targets were selected; unrelated passing suites were not rerun.

The same single CTest also passed in an offline Linux amd64 container: Ubuntu 26.04.1, Clang 21.1.8 and CMake 4.2.3, using the current source foundation and a read-only repository mount. This is additional local Linux evidence, not a reproduction of the hosted Ubuntu 24.04/Clang 18 toolchain. Both focused runs used the SDK-free build and test-only slow vendor.

The [cancelled SQV macOS 15 job](https://github.com/QuanuX/Symphony/actions/runs/43274911103/job/131380637356) had no assigned runner and no executed steps; it supplied no failing test evidence. Its three Linux siblings passed. No SQV or workflow change is made for that cancellation. The repair push schedules fresh PR checks; their results are not awaited. SSH v1 and Cloudflare code are unchanged, and no licensed SDK or live provider was used in this repair.
