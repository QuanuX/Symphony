# SHV profile engine installation

Build independently with CMake 3.25+, a C++26 compiler and the exact foundation SDK
0.1.0, or the repository foundation target. Select an explicit install prefix.

```sh
cmake -S modules/shv-profile-engine -B build/shv-profile \
  -DSYMPHONY_KVE_USE_INSTALLED=ON -DCMAKE_PREFIX_PATH=/absolute/foundation-sdk \
  -DCMAKE_INSTALL_PREFIX=/absolute/profile-prefix
cmake --build build/shv-profile
ctest --test-dir build/shv-profile --output-on-failure
cmake --install build/shv-profile
```

The v2 install receipt owns the versioned executable, documentation, resources and
licenses. `uninstall-shv-profile-engine` validates the selected receipt before removal.
Caller evidence, recipes and universes are never installed or removed by that target.
The kernel reader and PDF interpretation code are compiled dependencies; runtime PDF
binding additionally requires the explicitly selected exact decoder installation.
