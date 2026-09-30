# Observer installation

CMake 3.30+, a C++26 compiler and the source KVE dependency build this independent module.

```sh
cmake -S modules/snv-local-observer -B build/snv-local-observer -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/chosen/prefix
cmake --build build/snv-local-observer
ctest --test-dir build/snv-local-observer --output-on-failure
cmake --install build/snv-local-observer
```

An installed-dependency build uses `-DSYMPHONY_SNV_USE_INSTALLED_DEPENDENCIES=ON` and `CMAKE_PREFIX_PATH` for exact receipt-bound KVE `0.2.0-dev`. The installed executable is `libexec/symphony/snv-local-observer/0.1.0-dev/symphony-snv-local-observer`; SDK paths and schemas are versioned separately. Receipt: `share/symphony/receipts/snv-local-observer/0.1.0-dev/install-receipt.json`. `uninstall-snv-local-observer` verifies the exact owned bytes before removal and retains unrelated/user files.

Native Linux reads require ordinary read access to selected proc/sysfs files. No elevation, external command, network, listener or daemon is used. Kernel filesystem type, regular file type and no-follow path traversal are checked. Restricted exposure is an explicit result.

Synthetic reader tests and macOS unavailable output do not establish Linux runtime conformance. Actual Linux VM/container evidence describes only that process's exposure, never bare-metal inventory.
