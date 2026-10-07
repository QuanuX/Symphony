# SBV engine manifest

Module `sbv-engine`, engine `symphony-sbv`, vector `sbv`, release `0.23.0-dev`. Native executable with exact static source dependencies and versioned receipt-v2 ownership. Independent interop headers and the native C ABI/C++ calculation SDK with exact CMake targets are installed. Vendor model runtime adapters remain optional future work. Backtest artifacts are user-owned outputs outside the install receipt.

## Canonical Surfaces

- `modules/sbv-engine/src/row_spool.hpp`
- `modules/sbv-engine/src/row_spool.cpp`
- `modules/sbv-engine/src/exact_id_index.hpp`
- `modules/sbv-engine/src/exact_id_index.cpp`
- `modules/sbv-engine/src/reader_extensions.inc`
- `modules/sbv-engine/src/logical_value.hpp`
- `modules/sbv-engine/src/logical_value.cpp`
- `modules/sbv-engine/src/stream_census.hpp`
- `modules/sbv-engine/src/stream_census.cpp`
- `modules/sbv-engine/src/partitioned_output.hpp`
- `modules/sbv-engine/src/partitioned_output.cpp`
- `modules/sbv-engine/src/stream_model.hpp`
- `modules/sbv-engine/src/stream_model.cpp`
- `modules/sbv-engine/src/evaluate_partitioned.cpp`
- `modules/sbv-engine/src/economics_kernel.hpp`
- `modules/sbv-engine/src/economics_partitioned.cpp`
- `modules/sbv-engine/src/economic_census_stream.hpp`
- `modules/sbv-engine/src/economic_census_stream.cpp`
- `modules/sbv-engine/src/run_math.hpp`
- `modules/sbv-engine/src/run_partitioned.cpp`
- `modules/sbv-engine/tests/reader_extensions_test.cpp`
- `modules/sbv-engine/tests/row_spool_test.cpp`
- `modules/sbv-engine/tests/exact_id_index_test.cpp`
- `modules/sbv-engine/tests/logical_value_test.cpp`
- `modules/sbv-engine/tests/stream_census_test.cpp`
- `modules/sbv-engine/tests/partitioned_output_test.cpp`
- `modules/sbv-engine/tests/stream_model_test.cpp`
- `modules/sbv-engine/tests/generate_partitioned_test.cpp`
- `modules/sbv-engine/tests/evaluate_partitioned_test.cpp`
- `modules/sbv-engine/tests/economics_partitioned_test.cpp`
- `modules/sbv-engine/tests/run_partitioned_test.cpp`

- `modules/sbv-engine/tests/result_store_test.cpp`

- `modules/sbv-engine/tests/bundle_operations_test.cpp`

- `modules/sbv-engine/src/result_store.hpp`
- `modules/sbv-engine/src/result_store.cpp`
- `modules/sbv-engine/src/streaming_sha256.hpp`
- `modules/sbv-engine/src/bundle_operations.cpp`
- `modules/sbv-engine/schemas/v1/partitioned-result.schema.json`
- `modules/sbv-engine/schemas/v1/result-page.schema.json`
- `modules/sbv-engine/schemas/v1/node-stream.schema.json`

- `modules/sbv-engine/src/source_owners.hpp`
- `modules/sbv-engine/src/source_owners.cpp`
- `modules/sbv-engine/tests/source_owners_test.cpp`
- `modules/sbv-engine/tests/source_dataset_test.cpp`
- `modules/sbv-engine/tests/fixture_requests.hpp`

- `modules/sbv-engine/cmake/SymphonySbvProviderConfig.cmake.in`

- `modules/sbv-engine/tests/provider_integration_test.cpp`

- `modules/sbv-engine/tests/provider_runtime_test.cpp`

- `modules/sbv-engine/tests/provider_fixture.cpp`

- `modules/sbv-engine/tests/provider_abi_test.c`

- `modules/sbv-engine/src/generate_census.cpp`

- `modules/sbv-engine/src/provider.cpp`

- `modules/sbv-engine/src/provider.hpp`

- `modules/sbv-engine/include/symphony/sbv/provider.h`

- `modules/sbv-engine/tests/compose_economics_test.cpp`

- `modules/sbv-engine/tests/census_test.cpp`

- `modules/sbv-engine/src/compose_economics.cpp`

- `modules/sbv-engine/src/census.cpp`

- `modules/sbv-engine/src/census.hpp`

- `modules/sbv-engine/tests/history_test.cpp`

- `modules/sbv-engine/src/history.cpp`

- `modules/sbv-engine/tests/dependencies_test.cpp`

- `modules/sbv-engine/tests/fit_test.cpp`

- `modules/sbv-engine/src/fit.cpp`

- `modules/sbv-engine/src/combinatorial.hpp`
- `modules/sbv-engine/tests/combinatorial_test.cpp`


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

Experimental 0.17 adds explicit local experiment dependencies and immutable result-reference bindings for split/fit/predict graphs. Resolved claims, parent receipts, blocked states and stable wave order are terminal-queryable; exact-plan reconciliation reuses completed trials. See SPEC.md for scope and remaining orchestration work.

Experimental 0.18 adds `qxctl sbv research-history`: source-bound fit/prediction reuse counts and explicit comparison-backed selection records, with caller-selected coverage, order, duplicate counting and observation retention. All result fields use the existing terminal/SDK contract.
