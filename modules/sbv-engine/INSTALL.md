# SBV installation

Build with a CMake release that recognizes C++26 and a supported C++26 compiler. This source release uses the exact in-tree dependency revisions; it does not fetch upgrades.

```sh
cmake -S modules/sbv-engine -B build/sbv -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/absolute/private/prefix
cmake --build build/sbv
ctest --test-dir build/sbv --output-on-failure
cmake --install build/sbv
```

The independently installable executable is `libexec/symphony/sbv-engine/0.13.0-dev/symphony-sbv`. Receipt, contracts, schemas and licenses are versioned and owned by its v2 install receipt. Static dependencies are embedded in this executable; it does not require a running bus, provider connection, another engine process or a GPU. The static `Symphony::Sbv` target is available to in-tree builds; the shared `Symphony::SbvSdk` target supplies the installed C ABI/C++ wrapper.

The generated `uninstall-sbv-engine` target verifies the exact receipt-owned paths and refuses changed files or shared-root lifecycle ownership; it never removes backtest results. Configure a separate prefix when evaluating this experimental release.

The receipt also installs the independent header-only interop contract. Configure a separate consumer with `-DSymphonySbvInterop_DIR=<prefix>/lib/cmake/SymphonySbvInterop/0.13.0-dev`, call `find_package(SymphonySbvInterop CONFIG REQUIRED)` and link `Symphony::SbvInterop`. No engine library or vendor SDK is needed for these capability/ownership types. The separate calculation SDK below shares the receipt; neither target supplies a CUDA/tensor runtime.

The exact prefix also owns the native shared SDK under `lib/symphony/sbv-engine/<release>/`, C/C++ headers, `lib/cmake/SymphonySbvSdk/<release>/SymphonySbvSdkConfig.cmake`. Select that CMake config directory explicitly; consumers link `Symphony::SbvSdk`. External language bindings may load this exact native ABI. No package manager or interpreter is required by the native engine. The receipt/uninstaller owns all 20 files; user result and experiment files remain separate.
