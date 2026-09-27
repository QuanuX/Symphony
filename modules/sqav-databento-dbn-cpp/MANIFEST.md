# SQAV Databento DBN C++ Manifest

## Identity

- Module/package: `sqav-databento-dbn-cpp`; owner: SQAV
- Exact development release: `0.1.0-dev`; C++26 static library
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

## Dependencies and scope

Runtime dependencies are SQAV capture 0.1.0-dev, SQMV 0.1.0-dev, SQFV 0.2.0-dev and the knowledge-engine foundation 0.2.0-dev. SQPV 0.1.0-dev and SQDV 0.1.0-dev are used only in native composition tests. Receipt-v2 installation owns the library, header, exact exports, six documents and license. There is no process entry point, provider session, credential access, background collector, namespace allocation or SQV qxctl surface.
