# IBKR TWS Stable read adapter boundary manifest

## Identity

- Module/component/package: `scabv-ibkr-tws-cpp`
- Owner: SCABV
- Release: `0.1.0-dev`; native static C++26 library
- Public target: `Symphony::ScabvIbkrTws`

## Canonical surfaces

- `modules/scabv-ibkr-tws-cpp/CMakeLists.txt`
- `modules/scabv-ibkr-tws-cpp/FEATURES.md`
- `modules/scabv-ibkr-tws-cpp/INSTALL.md`
- `modules/scabv-ibkr-tws-cpp/INTENT.md`
- `modules/scabv-ibkr-tws-cpp/MANIFEST.md`
- `modules/scabv-ibkr-tws-cpp/SDK-HANDOFF.md`
- `modules/scabv-ibkr-tws-cpp/SKILL.md`
- `modules/scabv-ibkr-tws-cpp/SPEC.md`
- `modules/scabv-ibkr-tws-cpp/cmake/SymphonyScabvIbkrTwsConfig.cmake.in`
- `modules/scabv-ibkr-tws-cpp/cmake/uninstall.cmake.in`
- `modules/scabv-ibkr-tws-cpp/include/symphony/scabv/tws.hpp`
- `modules/scabv-ibkr-tws-cpp/src/tws.cpp`
- `modules/scabv-ibkr-tws-cpp/tests/allocation.cpp`
- `modules/scabv-ibkr-tws-cpp/tests/sdk-consumer/CMakeLists.txt`
- `modules/scabv-ibkr-tws-cpp/tests/sdk-consumer/main.cpp`
- `modules/scabv-ibkr-tws-cpp/tests/sdk1045-conformance/CMakeLists.txt`
- `modules/scabv-ibkr-tws-cpp/tests/sdk1045-conformance/conformance.cpp`
- `modules/scabv-ibkr-tws-cpp/tests/test.cpp`

## Extent

Finite read-only native request/callback state around the external Stable 10.45 SDK dependency.

No installed or licensed vendor SDK, verified SDK 10.45 conformance, connection/login runtime, complete EWrapper implementation, automatic request-ID allocation/pacing, live collection, order entry, provider cancellation receipt or operational SSIAG authorization.
