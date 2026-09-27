# SQDV Delivery C++ Manifest

## Identity

- Module, component, and package ID: `sqdv-delivery-cpp`
- Owner: `sqdv`
- Exact development release: `0.1.0-dev`
- Package kind: `module`; C++26 static library with receipt-v2 installation
- Public target: `Symphony::SqdvDelivery`

## Canonical Surfaces

- `modules/sqdv-delivery-cpp/INTENT.md`
- `modules/sqdv-delivery-cpp/MANIFEST.md`
- `modules/sqdv-delivery-cpp/SPEC.md`
- `modules/sqdv-delivery-cpp/SKILL.md`
- `modules/sqdv-delivery-cpp/INSTALL.md`
- `modules/sqdv-delivery-cpp/FEATURES.md`
- `modules/sqdv-delivery-cpp/CMakeLists.txt`
- `modules/sqdv-delivery-cpp/include/symphony/sqdv/delivery.hpp`
- `modules/sqdv-delivery-cpp/src/delivery.cpp`
- `modules/sqdv-delivery-cpp/cmake/SymphonySqdvDeliveryConfig.cmake.in`
- `modules/sqdv-delivery-cpp/cmake/uninstall.cmake.in`
- `modules/sqdv-delivery-cpp/tests/delivery_test.cpp`
- `modules/sqdv-delivery-cpp/tests/package_lifecycle_test.cmake`
- `modules/sqdv-delivery-cpp/tests/sdk-consumer/CMakeLists.txt`
- `modules/sqdv-delivery-cpp/tests/sdk-consumer/main.cpp`

## Exact dependencies and extent

The public library composes `sqfv-batch-cpp` `0.2.0-dev`,
`sqmv-metadata-cpp` `0.1.0-dev`, and `sqpv-local-store-cpp` `0.1.0-dev`.
The implementation uses `knowledge-vector-engine-cpp` `0.2.0-dev` solely for
bounded identity hashing. Its package exports the exact dependencies and C++26
requirement. No knowledge-engine envelope carries the routine data payload.

The receipt has no process entry point. The module allocates no `sqdv:` colon
namespace and introduces no provider, recipient authorization, network channel,
destination commit, background worker, source parser, transformation, global
catalogue, or SQV qxctl surface. The retained profile inherits SQPV's admitted
macOS/local-APFS scope and private caller-managed root contract.
