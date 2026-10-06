# SBV engine manifest

Module `sbv-engine`, engine `symphony-sbv`, vector `sbv`, release `0.14.0-dev`. Native executable with exact static source dependencies and versioned receipt-v2 ownership. Independent interop headers and the native C ABI/C++ calculation SDK with exact CMake targets are installed. Vendor model runtime adapters remain optional future work. Backtest artifacts are user-owned outputs outside the install receipt.

## Canonical Surfaces


- `modules/sbv-engine/tests/sdk_consumer.c`

- `modules/sbv-engine/tests/sdk_consumer.cpp`

- `modules/sbv-engine/tests/sdk_test.cpp`

- `modules/sbv-engine/cmake/SymphonySbvSdkConfig.cmake.in`


- `modules/sbv-engine/include/symphony/sbv/sdk.hpp`

- `modules/sbv-engine/include/symphony/sbv/sdk.h`

- `modules/sbv-engine/src/sdk.cpp`

- `modules/sbv-engine/tests/research_test.cpp`

- `modules/sbv-engine/src/experiment.cpp`

- `modules/sbv-engine/src/resample.cpp`
- `modules/sbv-engine/src/split.cpp`
- `modules/sbv-engine/tests/split_test.cpp`

- `modules/sbv-engine/tests/analysis_test.cpp`

- `modules/sbv-engine/src/compare.cpp`

- `modules/sbv-engine/src/analyze.cpp`

- `modules/sbv-engine/tests/interop_consumer.cpp`

- `modules/sbv-engine/tests/planning_test.cpp`

- `modules/sbv-engine/cmake/SymphonySbvInteropConfig.cmake.in`

- `modules/sbv-engine/include/symphony/sbv/interop.hpp`

- `modules/sbv-engine/src/planning.cpp`

- `modules/sbv-engine/src/select.cpp`

- `modules/sbv-engine/tests/allocation_economics_test.cpp`

- `modules/sbv-engine/src/wide_rational.hpp`

- `modules/sbv-engine/src/allocation_economics.cpp`

- `modules/sbv-engine/tests/liquidity_test.cpp`

- `modules/sbv-engine/src/liquidity.cpp`

- `modules/sbv-engine/tests/book_test.cpp`

- `modules/sbv-engine/src/book.cpp`

- `modules/sbv-engine/tests/economics_test.cpp`

- `modules/sbv-engine/src/economics.cpp`

- `modules/sbv-engine/tests/models_test.cpp`

- `modules/sbv-engine/src/rational.hpp`

- `modules/sbv-engine/src/compose_joint.cpp`

- `modules/sbv-engine/src/evaluate.cpp`

- `modules/sbv-engine/src/model.cpp`

- `modules/sbv-engine/include/symphony/sbv/models.hpp`

- `modules/sbv-engine/CMakeLists.txt`
- `modules/sbv-engine/FEATURES.md`
- `modules/sbv-engine/INSTALL.md`
- `modules/sbv-engine/INTENT.md`
- `modules/sbv-engine/INTERFACE-GENERATOR.json`
- `modules/sbv-engine/MANIFEST.md`
- `modules/sbv-engine/OWNER-INTERFACE.json`
- `modules/sbv-engine/SKILL.md`
- `modules/sbv-engine/SPEC.md`
- `modules/sbv-engine/cmake/uninstall.cmake.in`
- `modules/sbv-engine/include/symphony/sbv/sbv.hpp`
- `modules/sbv-engine/schemas/v1/admin.schema.json`
- `modules/sbv-engine/schemas/v1/admin.templates.json`
- `modules/sbv-engine/schemas/v1/pointer-stream.schema.json`
- `modules/sbv-engine/schemas/v1/result.schema.json`
- `modules/sbv-engine/src/compose.cpp`
- `modules/sbv-engine/src/detail.hpp`
- `modules/sbv-engine/src/interface.generated.hpp`
- `modules/sbv-engine/src/main.cpp`
- `modules/sbv-engine/src/result.cpp`
- `modules/sbv-engine/src/run.cpp`
- `modules/sbv-engine/tests/administration_test.cpp`
- `modules/sbv-engine/tests/fixtures/interface-history.v1.json`
- `modules/sbv-engine/tests/sbv_test.cpp`

## Shared Implementation

- `cmake/SbvInterface.generated.cmake`
- `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- `modules/sqav-databento-dbn-cpp/MANIFEST.md`
- `tools/qxctl/cmd/qxctl/sbv.go`
- `tools/qxctl/internal/knowledgeengine/sbv.go`

- `modules/sbv-engine/src/dataset.hpp`
- `modules/sbv-engine/src/dataset.cpp`
- `modules/sbv-engine/src/resident.cpp`
- `modules/sbv-engine/tests/dataset_test.cpp`
