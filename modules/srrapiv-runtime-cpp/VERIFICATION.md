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
