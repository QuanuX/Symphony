# SQV administration verification

Run the focused installed qualification with CMake 3.30+, a C++26 compiler, Go 1.26.5 on `PATH`, and curl development files exactly version 8.7.1 (the existing Databento writer dependency):

```sh
CXX=clang++ sh tests/sqv-administration/run_regressions.sh /absolute/fresh/private/work-directory
```

The directory must not exist, and its ancestors must satisfy the store path contract. Prefer a short path under the platform's canonical temporary directory. The harness creates a private installation, writer fixtures, temporary files, a qxctl executable and per-step logs there; it retains them for inspection. It honors `CXX`, `CMAKE_PREFIX_PATH`, `SQV_ADMIN_BUILD_JOBS` (default 2), and caller-provided Go caches. Additional arguments are passed to CMake configuration. Go runs with `GOWORK=off` and the selected local toolchain. Dependency downloads may require network access; fixture generation and inspection use no provider network or credentials.

The gate builds all six exact owner executables, runs four CTests (metadata, flow, conversion and read-only storage boundaries), installs receipt-v2 packages into the fresh prefix, and runs all six installed native process tests. `sqv-admin-fixtures` creates synthetic retained state through the existing SQPV store, SQDV checkpoint and SQAV attempt writers. The Go tests then invoke the installed owners and the real qxctl executable for all three retained-state routes: `sqv store inspect`, `sqv checkpoint inspect` and `sqv acquisition attempts`. They verify receipts and resources, response identity, clean and dirty states, locks, full-width counters, metadata/generation fences, typed-owner refusal, tampering, malformed inputs and nonmutation. Selected Go tests run with the race detector and caching disabled; missing installed inputs or skipped tests fail the gate.

For an existing private installation, run `TestSQVAdministrationInstalled` under `tools/qxctl/internal/knowledgeengine` with `SQV_ADMIN_TEST_PREFIX` and `SQV_ADMIN_FIXTURES`. Set `SQV_ADMIN_TEST_QXCTL` to the absolute qxctl executable to exercise the command boundary, and `SQV_ADMIN_REQUIRE_INSTALLED=1` to require all inputs. Ordinary Go suites retain the optional installed-fixture skip. Existing `TestSQAVInstalled` separately protects the original acquisition-request contract.

Production observation targets link only metadata, frame and process foundations. SQPV/SQDV/Databento writers and provider transport are fixture-only dependencies. Source and installed dependency builds are exact-version checked. This gate qualifies the recorded OS, architecture and filesystem for retained-state observation; Ubuntu ARM64 alone is not DGX Spark/DGX OS storage or GPU qualification. Broad unrelated runtime suites, provider delivery and writer crash-recovery qualification are outside this focused gate.
