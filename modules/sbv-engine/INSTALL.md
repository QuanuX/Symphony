# SBV installation

Build with a CMake release that recognizes C++26 and a supported C++26 compiler. This source release uses the exact in-tree dependency revisions; it does not fetch upgrades.

```sh
cmake -S modules/sbv-engine -B build/sbv -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/absolute/private/prefix
cmake --build build/sbv
ctest --test-dir build/sbv --output-on-failure
cmake --install build/sbv
```

The independently installable executable is `libexec/symphony/sbv-engine/0.21.0-dev/symphony-sbv`. Receipt, contracts, schemas and licenses are versioned and owned by its v2 install receipt. Static dependencies are embedded in this executable; it does not require a running bus, provider connection, another engine process or a GPU. The static `Symphony::Sbv` target is available to in-tree builds; the shared `Symphony::SbvSdk` target supplies the installed C ABI/C++ wrapper.

The generated `uninstall-sbv-engine` target verifies the exact receipt-owned paths and refuses changed files or shared-root lifecycle ownership; it never removes backtest results. Configure a separate prefix when evaluating this experimental release.

The receipt also installs the independent header-only interop contract. Configure a separate consumer with `-DSymphonySbvInterop_DIR=<prefix>/lib/cmake/SymphonySbvInterop/0.21.0-dev`, call `find_package(SymphonySbvInterop CONFIG REQUIRED)` and link `Symphony::SbvInterop`. No engine library or vendor SDK is needed for these capability/ownership types. The separate calculation SDK below shares the receipt; neither target supplies a CUDA/tensor runtime.

The exact prefix also owns the native shared SDK under `lib/symphony/sbv-engine/<release>/`, C/C++ headers, `lib/cmake/SymphonySbvSdk/<release>/SymphonySbvSdkConfig.cmake`. Select that CMake config directory explicitly; consumers link `Symphony::SbvSdk`. External language bindings may load this exact native ABI. No package manager or interpreter is required by the native engine. The receipt/uninstaller owns all 22 files; user result and experiment files remain separate.

Experimental 0.17 adds explicit local experiment dependencies and immutable result-reference bindings for split/fit/predict graphs. Resolved claims, parent receipts, blocked states and stable wave order are terminal-queryable; exact-plan reconciliation reuses completed trials. See SPEC.md for scope and remaining orchestration work.

Experimental 0.18 adds `qxctl sbv research-history`: source-bound fit/prediction reuse counts and explicit comparison-backed selection records, with caller-selected coverage, order, duplicate counting and observation retention. All result fields use the existing terminal/SDK contract.

Experimental 0.19 adds retained native/imported census references and
`qxctl sbv compose-economics`. Use `qxctl sbv schema --operation evaluate`,
`--operation economics` or `--operation compose-economics` with the exact prefix
and version to inspect the input alternatives and typed result definitions.
The same operations are available through `Symphony::SbvSdk`; result export
preserves every retained field for terminal, GUI and external research consumers.

Experimental 0.20 installs `symphony/sbv/provider.h` and the interface-only
`Symphony::SbvProvider` target. A standalone provider uses
`-DSymphonySbvProvider_DIR=<prefix>/lib/cmake/SymphonySbvProvider/0.21.0-dev`,
`find_package(SymphonySbvProvider CONFIG REQUIRED)`, and links that interface
target. It exports `symphony_sbv_provider_api_v1` and needs no SBV library.
Select its exact binary path/digest in `qxctl sbv provider-inspect`,
`qxctl sbv generate-census`, or the native provider model in `qxctl sbv evaluate`.
Inspect schemas/templates for these operations before supplying the user's own
parameters. Loading executes trusted native code; the precise lifetime,
concurrency, dependency and cancellation contract is in SPEC.md and provider.h.

Experimental 0.21 adds `qxctl sbv source-retain` and `source-export`, with the same operations in `Symphony::SbvSdk`. Request the exact installed schema/template, select existing private owner storage and new result destinations, and supply actual caller acquisition/access evidence. For run/evaluate/book/generate-census/dataset-load, replace the entire file source tuple with `retained_source` to use that stored original. The optional profile's limits are discoverable in capabilities; standalone file/RAM limits remain user selected. For a recovery_required response preserve the full returned evidence; qxctl exits 5, while SDK consumers must inspect status. No automatic retry occurs.
