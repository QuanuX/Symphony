# SQFV flow administration install

Build `modules/sqfv-flow-engine` with CMake 3.30+, a C++26 compiler and the exact native dependencies listed in SPEC. Install into an explicitly selected prefix. The engine, contracts, resources and licenses are receipt-v2 owned. Use the existing qxctl lifecycle receipt inspection. Target `uninstall-sqfv-flow-engine` performs guarded receipt-v2 removal. No global installation or automatic version selection is performed.

## Build and verification

```sh
cmake -S modules/sqfv-flow-engine -B BUILD -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=PREFIX
cmake --build BUILD
cmake --install BUILD
```

`SYMPHONY_SQV_ADMIN_INSTALLED=ON` selects exact installed dependencies through CMAKE_PREFIX_PATH. The default builds the pinned source dependencies. Configure `tests/sqv-administration` to run the shared focused native gate and generate real writer fixtures for the installed qxctl tests. Test-only writers are absent from production inspection executables. See `tests/sqv-administration/README.md`.
