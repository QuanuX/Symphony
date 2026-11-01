# MVP dependency qualification

The checked-in map at `cmake/mvp-dependencies.json` is a derived build projection. It records the currently implemented package baseline; it does not approve a release or replace any module's contract.

Verify the current source snapshot with CMake 3.30 or later:

```sh
cmake -P tests/mvp-dependencies/verify.cmake
```

The check compares all immediate module/library/tool roots, source-input hashes, runtime dependency order, literal CMake project versions, Go module/toolchain identities, Swift platform/toolchain, owner versions, generated Go admission versions and source schema IDs. Generated CMake project versions are checked after actual `project()` evaluation during the build gate. Source schema folders and protocol versions are independent of package versions.

Run one native group in a new private directory with a C++26 toolchain:

```sh
CXX=clang++ sh tests/mvp-dependencies/run.sh knowledge /absolute/fresh-work
CXX=clang++ sh tests/mvp-dependencies/run.sh quantitative /absolute/another-fresh-work \
  -DCMAKE_PREFIX_PATH=/absolute/exact-external-dependency-prefix
```

Groups are `bootstrap`, `knowledge`, `cloud`, `quantitative`, `duckdb`, and `build_support`. The driver adds each selected component's runtime SDK closure, preserves the mapped build order, and installs into one isolated prefix. `duckdb` builds both connectors and their installed process, native interruption recovery, adjacent-library and removal tests. Prepare its exact dependency files with `.github/scripts/prepare-duckdb-dependency.sh`, then pass `-C /absolute/dependency/dependency.cmake` to the driver. `build_support` compiles repository authoring tools without installing them. Set `MVP_BUILD_JOBS` to control parallel compilation (default 2). The quantitative group requires the caller's exact curl 8.7.1 and zstd 1.5.7 development packages. The harness does not download native dependencies.

Every configure uses Release, PIC, disabled package registries and the component's exact installed-dependency options. A deferred configure check requires mapped artifact targets, exact SDK package versions and imported first-party SDKs whose headers and libraries resolve beneath the selected prefix. Recursive subdirectories and compiled sources outside the selected component are rejected unless the map explicitly admits a shared implementation. A negative configure disables a required SDK and must fail explicitly. The complete default package target is built before installation, including auxiliary executable entry points. After each install, receipt identity, self-digest, every owned file and executable entry point are checked. These checks operate on a private quiescent prefix, not a concurrent hostile-filesystem boundary.

The driver first exercises three isolated configure fixtures: project/package version-name collision must pass, while wrong visible and helper-scoped SDK versions must fail. Existing SDK consumer directories are copied outside the checkout and built/run against installed exports. Quantitative qualification includes the capture private-export regression, C/C++ SBV SDK correspondence, provider/feedback ABI and interop checks, SBV feedback-settlement contracts, and queue scenarios including wide-integer overflow boundaries. Knowledge qualification enables only the coordinator lifecycle-planner regression after installation. Cloud qualification requires Go 1.26.5 on `PATH`, builds qxctl once using public dependencies and a private cache, runs actual current-version inspect/interface/schema commands for all eight domains from outside the checkout, and verifies that the preserved legacy default does not silently select the current package. No service or provider is activated.

Qualification artifacts retain link-path install RPATHs so their installed executables and shared libraries can load the exact private dependencies. They are bound to these private prefixes and are not relocatable release packages. After the SBV consumer tests, a separate loader trace must show curl loaded from a supplied external prefix: macOS uses `DYLD_PRINT_LIBRARIES`, and Linux uses `LD_TRACE_LOADED_OBJECTS`. A library search attempt or an ambient system curl does not pass this check. An additional Apple framework load of `/usr/lib/libcurl.4.dylib` is recorded separately and cannot satisfy the private-load requirement. Release relocation and distribution remain separate packaging work.

Logs, host/toolchain evidence, component selection, imported-target proof, receipts and the private installation remain under the work directory. To freeze the harness during concurrent development, copy this directory outside the repository and set `MVP_REPOSITORY` to the absolute source checkout when invoking the copied scripts.

Resolve, compare and build all standalone Go modules with the exact Go 1.26.5 toolchain:

```sh
sh tests/mvp-dependencies/go-graphs.sh /absolute/fresh-go-work
```

This uses `GOWORK=off`, public proxy/checksum verification and fresh task caches. The selected `Path@Version` graph must match the map exactly; replacements and manifest writes fail. Every module then builds all its packages, with command binaries written under the work directory and checked against the map. The SBV example is a compile check, not a release artifact. No built service or provider is installed or activated.

When adding or changing a component, edit the semantic map entries first: root disposition, exact package/project versions, runtime/test dependency distinction, installed options, exported SDK targets, build order and qualification group. Add new authoritative source paths to `source_inputs`. After reviewing those changes, refresh the existing input hashes without inferring or changing dependency semantics:

```sh
cmake -DMVP_ACCEPT_SOURCE_CHANGES=ON -P tests/mvp-dependencies/refresh-inputs.cmake
cmake -P tests/mvp-dependencies/verify.cmake
```

Review the resulting map diff and rerun the affected installed group and Go graph checks. The refresh command only changes hashes of already listed inputs; it cannot discover a new module, approve a version change or update admission contracts automatically.
