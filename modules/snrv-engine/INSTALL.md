# SNRV installation

Use CMake 3.30 or newer and a compiler supporting C++26, with language extensions disabled.

```sh
cmake -S modules/snrv-engine -B build/snrv -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/chosen/prefix
cmake --build build/snrv
ctest --test-dir build/snrv --output-on-failure
cmake --install build/snrv
```

For an independent installed-dependency build, add `-DSYMPHONY_SNV_USE_INSTALLED_DEPENDENCIES=ON` and the exact installed SDK prefix through `CMAKE_PREFIX_PATH`. Packages are versioned side by side. The engine is installed at `libexec/symphony/snrv-engine/0.1.0-dev/symphony-snrv`. Its SDK exports the exact CMake package and public header.

The receipt resides at `share/symphony/receipts/snrv-engine/0.1.0-dev/install-receipt.json`. Use the existing receipt-backed qxctl package lifecycle, or the package-specific `uninstall-snrv-engine` CMake target against the selected prefix. Guarded removal verifies the owned installation files and retains user evidence.

Development evidence on macOS and Linux compilation/runtime evidence are separate platform claims. The pure owner has no filesystem or platform probe.
