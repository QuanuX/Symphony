# Independent SCIV installation

CMake 3.30 or newer and a C++26 toolchain are required. Source dependency builds use the exact disclosed foundation/common source packages. `-DSYMPHONY_SNV_USE_INSTALLED_DEPENDENCIES=ON` instead requires exact installed `SymphonySnvCommon` 0.1.0 and `SymphonyKnowledgeVectorEngine` 0.2.0 SDKs in `CMAKE_PREFIX_PATH`.

```sh
cmake -S modules/sciv-engine -B build/sciv -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/chosen/private/prefix
cmake --build build/sciv
ctest --test-dir build/sciv --output-on-failure
cmake --install build/sciv
```

The exact receipt owns the executable, static library, header, resources and CMake exports. `uninstall-sciv-engine` removes only unchanged receipt-owned SCIV files, preserving separate dependencies and caller evidence. Configure the selected install prefix before building the removal target, or invoke `cmake -DINSTALL_PREFIX=/chosen/private/prefix -P build/sciv/uninstall.cmake` explicitly. Shared qxctl lifecycle controls selection and optional docking. Independent supplied-evidence use needs neither docking nor SNV parent installation.

Installed process, SDK consumption, dynamic dependency and guarded removal evidence is required for each claimed platform. MacOS development does not alone certify Linux or filesystem profiles.
