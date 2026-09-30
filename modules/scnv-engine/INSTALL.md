# SCNV engine installation

Configure this module directly with CMake 3.30 or later and a C++26 compiler. Default development mode compiles the exact knowledge-vector and neutral SNV mechanics from this source checkout. `SYMPHONY_SNV_USE_INSTALLED_DEPENDENCIES=ON` requires the exact independently installed SDKs through `CMAKE_PREFIX_PATH`.

```
cmake -S modules/scnv-engine -B build/scnv -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/chosen/prefix
cmake --build build/scnv
ctest --test-dir build/scnv --output-on-failure
cmake --install build/scnv
```

Installed artifacts have versioned locations and an exact receipt under `share/symphony/receipts/scnv-engine/0.1.0-dev/`. Installed SDK consumers use `find_package(SymphonyScnv 0.1.0 EXACT CONFIG REQUIRED)` and link `Symphony::Scnv`. Process execution needs no parent engine, Maestro or provider SDK. Use the package's receipt guarded uninstall target or the existing qxctl package lifecycle; uninstall only removes receipt-owned installation files and never private evidence.
