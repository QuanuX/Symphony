# SQAV Databento DBN C++ Manifest

## Identity

- Module/package: `sqav-databento-dbn-cpp`; owner: SQAV
- Exact development release: `0.5.0-dev`; C++26 static library
- Public target: `Symphony::SqavDatabentoDbn`

## Canonical Surfaces

- `modules/sqav-databento-dbn-cpp/CMakeLists.txt`
- `modules/sqav-databento-dbn-cpp/FEATURES.md`
- `modules/sqav-databento-dbn-cpp/INSTALL.md`
- `modules/sqav-databento-dbn-cpp/INTENT.md`
- `modules/sqav-databento-dbn-cpp/MANIFEST.md`
- `modules/sqav-databento-dbn-cpp/SKILL.md`
- `modules/sqav-databento-dbn-cpp/SPEC.md`
- `modules/sqav-databento-dbn-cpp/cmake/SymphonySqavDatabentoDbnConfig.cmake.in`
- `modules/sqav-databento-dbn-cpp/cmake/uninstall.cmake.in`
- `modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/dbn.hpp`
- `modules/sqav-databento-dbn-cpp/src/dbn.cpp`
- `modules/sqav-databento-dbn-cpp/tests/dbn_test.cpp`
- `modules/sqav-databento-dbn-cpp/tests/package_lifecycle_test.cmake`
- `modules/sqav-databento-dbn-cpp/tests/sdk-consumer/CMakeLists.txt`
- `modules/sqav-databento-dbn-cpp/tests/sdk-consumer/main.cpp`

- `modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/historical.hpp`

- `modules/sqav-databento-dbn-cpp/src/historical.cpp`

- `modules/sqav-databento-dbn-cpp/tests/historical_test.cpp`

- `modules/sqav-databento-dbn-cpp/tests/public_fixture.hpp`

- `modules/sqav-databento-dbn-cpp/tests/third_party/README.md`

- `modules/sqav-databento-dbn-cpp/tests/third_party/LICENSE.Apache-2.0`

- `modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/attempts.hpp`

- `modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/http.hpp`

- `modules/sqav-databento-dbn-cpp/src/attempts.cpp`

- `modules/sqav-databento-dbn-cpp/src/http.cpp`

- `modules/sqav-databento-dbn-cpp/tests/attempts_test.cpp`

- `modules/sqav-databento-dbn-cpp/tests/http_test.cpp`

## Dependencies and scope

Runtime dependencies are SQAV capture 0.2.0-dev, SQMV 0.2.0-dev, SQFV 0.3.0-dev and the knowledge-engine foundation 0.2.0-dev. SQPV 0.2.0-dev and the libcurl 8.7.1 SDK/system library are runtime dependencies; SQDV 0.3.0-dev is used only in native composition tests. Receipt-v2 installation owns the library, four headers, exact exports, six documents and license. There is no process entry point, installed provider session or operational SSIAG credential resolver, background collector, namespace allocation or SQV qxctl surface.
