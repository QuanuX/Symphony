# Functional release qualification

This manifest adds functional test execution to the existing installed dependency qualification. It is a test selection, not release approval or a new runtime owner.

`manifest.json` accounts for every component in `cmake/mvp-dependencies.json`. Its native selection contains 59 roots: 58 packaged components and the authoring support root, with 172 registered CTests and 17 additional process suites. Both DuckDB connectors run in the separate `duckdb` group, followed by four copied installed consumer CTests and receipt-owned removal checks on all four required profiles. The required SHV campaign below supplies the exact PDF and decoder for the adapter's additional installed replay; its ordinary unit and generated-interface tests still run. The two IBKR host-session suites exercise read-only state, pacing and recovery locally; optional IBKR vendor SDK object conformance is separately excluded. SRRAPIV runs three SDK-free native suites and three installed qxctl checks; the qxctl lifecycle fixture disables provider/trading admission. Its licensed SDK fixture remains separate. Cloudflare adds two native suites and one installed process suite; cloud qualification requires its installed Go receipt consumer without skips. Real cf/Node bundle execution remains separately qualified. Helper binaries already invoked through registered owner tests are not counted twice.

The existing `tests/mvp-dependencies/run.sh GROUP /absolute/fresh-work` builds and installs the complete selected dependency closure first. It then enables each declared suite against that prefix, compares the configured CTest names with the manifest, builds its tests, and executes every declared case. Test-only dependency cycles use the already installed SDKs. The import proof permits only the explicitly mapped authoring helper and exact test source files; it continues to reject SDK source fallback. Store fixtures are produced by real installed writer SDKs in a private directory. Knowledge, cloud and quantitative jobs additionally run the selected SNV, Cloudflare, SBV, SQAV and SRRAPIV Go consumer tests against the same prefix, requiring every named test to pass without a skip. The installed validator checks the actual checkout in each knowledge job.

Verbose native logs, discovery JSON, JUnit, direct-process evidence and exact selections are retained. Missing tests, failed tests, JUnit skips and unknown internal skips fail the job. Current SCV process tests retain one explicitly historical internal skip per owner: `SKIP provider coverage is an exact .5 addition`. It is recorded separately and contributes no executed coverage.

Installed Go event logs, stderr and summaries use the module label plus a SHA-256 digest of the complete suite selection (group, module, package, named tests and prefix-environment names). Multiple packages or distinct test selections in one module retain separate evidence. Reusing the same selection in an existing evidence directory still fails rather than overwriting its results. Source-only full-module report names remain unchanged.

The same four installed-SDK profiles run the additional campaigns declared in `selected_installed_campaigns`; their evidence is separate from the native totals above. [SHV](../shv-release-qualification/README.md) locks PDFium and documentary source assets, exercises real private SSIAG/STAV authorization and checks one catalogue through an installed DuckDB commit and publication preview. [SNV](../snv-qualification/README.md) checks all five supplied-record owners through copied SDK consumers and qxctl, with Linux proc/sysfs observation and explicit macOS native unavailability. [SBV](../sbv-launch-boundary/README.md) checks the small capped CPU census and second pass over production same-host SOV stdio IPC, including a lost completed reply and native worker interruption. SDK distribution and dependency evidence are described below. Each required runner rejects skips and binds its receipts, source and execution evidence. Historical Intel installations are useful development evidence; only reviewed current-source target runs qualify the required platforms.

Run the independent Go suites with:

```sh
sh tests/release-qualification/go-tests.sh /absolute/fresh-work
```

This requires exact Go 1.26.5, resolves public dependencies in private caches with `GOWORK=off`, and runs all eight modules with the race detector. It preserves every JSON event and reports top-level tests, subtests and explicitly classified optional skips separately. Historical installed-prefix/retained-fixture tests and subprocess helper entry points are listed in the manifest; unclassified skips fail. Tests requiring ordinary local sockets, subprocesses or OpenSSH must execute. Readonly module manifests are hashed before and after. The prior unexplained process-startup failure remains a required regression; its five-second startup bound is unchanged and failures now expose the invocation result.

On macOS, run the provider's tests with:

```sh
sh tests/release-qualification/swift-tests.sh /absolute/fresh-work
```

The test runner selects the binary built in that scratch directory through a test-only environment variable. It requires actual Swift Testing execution and retains JUnit and logs. The evidence checker accepts SwiftPM's `results.xml` and `results-swift-testing.xml` names, requires at least one nonempty report with executed cases, and rejects missing/malformed XML, count mismatches, failures and skips across either file. `sh tests/release-qualification/swift-evidence-tests.sh` exercises these refusal paths. The tests exercise local protocol, peer trust and readiness contracts; they do not activate a provider or perform operational credential access.

`release-qualification.yml` runs the eight Go suites on Ubuntu 24.04 x86_64, Ubuntu 26.04 x86_64, Ubuntu 24.04 ARM64 and macOS 15 ARM64. It then records the five non-qxctl Go candidate identities. Swift tests run on macOS. The existing MVP workflow supplies the same four native profiles, and the separate qxctl standalone workflow owns its reproducible build gate. These checks do not qualify physical DGX hardware, relocatable distribution archives, mirrors, native Windows, or live provider credentials.

When adding a component or test, update the dependency projection and this explicit manifest, run `cmake -P tests/release-qualification/verify.cmake`, and review configured discovery plus execution evidence. A submitted workflow is pending evidence until its exact-source results have been reviewed; skipped fixtures never become coverage through a successful job conclusion.

## SDK distribution and dependency evidence

The selected installed campaign registry also includes L22's copied SDK/source consumers and disposable-prefix removal checks, and L23's artifact-specific dependency/notice inventory. They run after the existing installed DAG and SHV/SNV/SBV campaigns on each required profile. Their assertions are counted separately from the 172 native CTests and 17 direct process suites. See `tests/native-sdk-distribution/README.md` and `tests/dependency-notices/README.md` for the exact observation boundaries and pending release gates.
