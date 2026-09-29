# IBKR Client Portal read adapter manifest

## Identity

- Module/component/package: `scabv-ibkr-client-portal-cpp`
- Owner: SCABV
- Release: `0.1.0-dev`; native static C++26 library
- Public target: `Symphony::ScabvIbkrClientPortal`

## Canonical surfaces

- `modules/scabv-ibkr-client-portal-cpp/CMakeLists.txt`
- `modules/scabv-ibkr-client-portal-cpp/FEATURES.md`
- `modules/scabv-ibkr-client-portal-cpp/INSTALL.md`
- `modules/scabv-ibkr-client-portal-cpp/INTENT.md`
- `modules/scabv-ibkr-client-portal-cpp/MANIFEST.md`
- `modules/scabv-ibkr-client-portal-cpp/SKILL.md`
- `modules/scabv-ibkr-client-portal-cpp/SPEC.md`
- `modules/scabv-ibkr-client-portal-cpp/cmake/SymphonyScabvIbkrClientPortalConfig.cmake.in`
- `modules/scabv-ibkr-client-portal-cpp/cmake/uninstall.cmake.in`
- `modules/scabv-ibkr-client-portal-cpp/include/symphony/scabv/client_portal.hpp`
- `modules/scabv-ibkr-client-portal-cpp/src/client_portal.cpp`
- `modules/scabv-ibkr-client-portal-cpp/tests/sdk-consumer/CMakeLists.txt`
- `modules/scabv-ibkr-client-portal-cpp/tests/sdk-consumer/main.cpp`
- `modules/scabv-ibkr-client-portal-cpp/tests/test.cpp`

## Extent

Exact private-account binding and bounded Client Portal account, position and historical-bar reads.

No session login, account mutation, order entry, OAuth replacement, TLS bypass, TWS protocol, atomic account snapshot, production entitlement/conformance claim or operational SSIAG authorization.
