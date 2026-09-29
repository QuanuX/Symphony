# FRED and ALFRED native adapter manifest

## Identity

- Module/component/package: `sqav-fred-cpp`
- Owner: SQAV
- Release: `0.1.0-dev`; native static C++26 library
- Public target: `Symphony::SqavFred`

## Canonical surfaces

- `modules/sqav-fred-cpp/CMakeLists.txt`
- `modules/sqav-fred-cpp/FEATURES.md`
- `modules/sqav-fred-cpp/INSTALL.md`
- `modules/sqav-fred-cpp/INTENT.md`
- `modules/sqav-fred-cpp/MANIFEST.md`
- `modules/sqav-fred-cpp/SKILL.md`
- `modules/sqav-fred-cpp/SPEC.md`
- `modules/sqav-fred-cpp/cmake/SymphonySqavFredConfig.cmake.in`
- `modules/sqav-fred-cpp/cmake/uninstall.cmake.in`
- `modules/sqav-fred-cpp/include/symphony/sqav/fred.hpp`
- `modules/sqav-fred-cpp/src/fred.cpp`
- `modules/sqav-fred-cpp/tests/sdk-consumer/CMakeLists.txt`
- `modules/sqav-fred-cpp/tests/sdk-consumer/main.cpp`
- `modules/sqav-fred-cpp/tests/test.cpp`

## Extent

Exact observation and vintage-date requests, original decimal/missing values, pagination and capture.

No FRED key provisioning, operational SSIAG bridge, arbitrary output types, frequency aggregation, series metadata API, cross-page snapshot atomicity or provider-authenticated fixtures.
