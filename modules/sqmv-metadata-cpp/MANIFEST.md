# SQMV Metadata C++ Manifest

## Identity

- Module, component, and package ID: `sqmv-metadata-cpp`
- Owner: `sqmv`
- Exact development release: `0.1.0-dev`
- Package kind: `module`; library-only receipt-v2 installation
- Public interface: C++26 static library, `Symphony::SqmvMetadata`

## Canonical Surfaces

- `modules/sqmv-metadata-cpp/INTENT.md`
- `modules/sqmv-metadata-cpp/MANIFEST.md`
- `modules/sqmv-metadata-cpp/SPEC.md`
- `modules/sqmv-metadata-cpp/SKILL.md`
- `modules/sqmv-metadata-cpp/INSTALL.md`
- `modules/sqmv-metadata-cpp/FEATURES.md`
- `modules/sqmv-metadata-cpp/CMakeLists.txt`
- `modules/sqmv-metadata-cpp/include/symphony/sqmv/metadata.hpp`
- `modules/sqmv-metadata-cpp/src/metadata.cpp`
- `modules/sqmv-metadata-cpp/tests/metadata_test.cpp`
- `modules/sqmv-metadata-cpp/tests/package_lifecycle_test.cmake`
- `modules/sqmv-metadata-cpp/tests/sdk-consumer/CMakeLists.txt`
- `modules/sqmv-metadata-cpp/tests/sdk-consumer/main.cpp`
- `modules/sqmv-metadata-cpp/cmake/SymphonySqmvMetadataConfig.cmake.in`
- `modules/sqmv-metadata-cpp/cmake/uninstall.cmake.in`

## Boundary

The public header and SPEC define the source API and SQM1 local encoding.
The exact package selects `sqfv-batch-cpp` `0.2.0-dev` for its public binding
type and `knowledge-vector-engine-cpp` `0.2.0-dev` for SHA-256 mechanics.
No knowledge-engine request or response envelope carries metadata or payloads.

The receipt has no engine entry point. This module allocates no `sqmv:` colon
namespace, provider adapter, qxctl operation, catalogue service, data-store
format, authentication method, or public network protocol. Referenced evidence
retains its actual producer and interpretation; structural acceptance does not
establish that evidence is authentic, available, complete, or authorized.
