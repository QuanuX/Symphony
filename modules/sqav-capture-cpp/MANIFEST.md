# SQAV Capture C++ Manifest

## Identity

- Module/package: `sqav-capture-cpp`; owner: SQAV
- Exact development release: `0.1.0-dev`; C++26 static library
- Public target: `Symphony::SqavCapture`

## Canonical Surfaces

- `modules/sqav-capture-cpp/CMakeLists.txt`
- `modules/sqav-capture-cpp/FEATURES.md`
- `modules/sqav-capture-cpp/INSTALL.md`
- `modules/sqav-capture-cpp/INTENT.md`
- `modules/sqav-capture-cpp/MANIFEST.md`
- `modules/sqav-capture-cpp/SKILL.md`
- `modules/sqav-capture-cpp/SPEC.md`
- `modules/sqav-capture-cpp/cmake/SymphonySqavCaptureConfig.cmake.in`
- `modules/sqav-capture-cpp/cmake/uninstall.cmake.in`
- `modules/sqav-capture-cpp/include/symphony/sqav/capture.hpp`
- `modules/sqav-capture-cpp/src/capture.cpp`
- `modules/sqav-capture-cpp/tests/capture_test.cpp`
- `modules/sqav-capture-cpp/tests/package_lifecycle_test.cmake`
- `modules/sqav-capture-cpp/tests/sdk-consumer/CMakeLists.txt`
- `modules/sqav-capture-cpp/tests/sdk-consumer/main.cpp`

## Dependencies and scope

Runtime dependencies are SQMV 0.1.0-dev, SQFV 0.2.0-dev and the knowledge-engine foundation 0.2.0-dev. SQPV 0.1.0-dev and SQDV 0.1.0-dev are used only in native composition tests. Receipt-v2 installation owns the library, header, exact exports, six documents and license. There is no process entry point, provider session, credential access, background collector, namespace allocation or SQV qxctl surface.
