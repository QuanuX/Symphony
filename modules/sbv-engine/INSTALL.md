# SBV installation

Build with a CMake release that recognizes C++26 and a supported C++26 compiler. This source release uses the exact in-tree dependency revisions; it does not fetch upgrades.

```sh
cmake -S modules/sbv-engine -B build/sbv -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/absolute/private/prefix
cmake --build build/sbv
ctest --test-dir build/sbv --output-on-failure
cmake --install build/sbv
```

The independently installable executable is `libexec/symphony/sbv-engine/0.6.0-dev/symphony-sbv`. Receipt, contracts, schemas and licenses are versioned and owned by its v2 install receipt. Static dependencies are embedded in this executable; it does not require a running bus, provider connection, another engine process or a GPU. The static `Symphony::Sbv` target is available to in-tree builds; a separately installed SDK is deferred.

The generated `uninstall-sbv-engine` target verifies the exact receipt-owned paths and refuses changed files or shared-root lifecycle ownership; it never removes backtest results. Configure a separate prefix when evaluating this experimental release.
