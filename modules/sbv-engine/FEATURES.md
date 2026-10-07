# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sbv-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Completed source changes receive attributable closure evidence at their actual gates.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "Canonical owner truth, implementation and checks have explicit indexed routes.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [],
      "evidence": [
        "modules/sbv-engine/tests/allocation_economics_test.cpp",
        "modules/sbv-engine/tests/analysis_test.cpp",
        "modules/sbv-engine/tests/book_test.cpp",
        "modules/sbv-engine/tests/census_test.cpp",
        "modules/sbv-engine/tests/combinatorial_test.cpp",
        "modules/sbv-engine/tests/compose_economics_test.cpp",
        "modules/sbv-engine/tests/dataset_test.cpp",
        "modules/sbv-engine/tests/dependencies_test.cpp",
        "modules/sbv-engine/tests/economics_partitioned_test.cpp",
        "modules/sbv-engine/tests/economics_test.cpp",
        "modules/sbv-engine/tests/evaluate_partitioned_test.cpp",
        "modules/sbv-engine/tests/exact_id_index_test.cpp",
        "modules/sbv-engine/tests/fit_test.cpp",
        "modules/sbv-engine/tests/generate_partitioned_test.cpp",
        "modules/sbv-engine/tests/history_test.cpp",
        "modules/sbv-engine/tests/interop_consumer.cpp",
        "modules/sbv-engine/tests/liquidity_test.cpp",
        "modules/sbv-engine/tests/logical_value_test.cpp",
        "modules/sbv-engine/tests/models_test.cpp",
        "modules/sbv-engine/tests/partitioned_output_test.cpp",
        "modules/sbv-engine/tests/planning_test.cpp",
        "modules/sbv-engine/tests/provider_integration_test.cpp",
        "modules/sbv-engine/tests/provider_runtime_test.cpp",
        "modules/sbv-engine/tests/reader_extensions_test.cpp",
        "modules/sbv-engine/tests/research_test.cpp",
        "modules/sbv-engine/tests/row_spool_test.cpp",
        "modules/sbv-engine/tests/run_partitioned_test.cpp",
        "modules/sbv-engine/tests/sbv_test.cpp",
        "modules/sbv-engine/tests/sdk_consumer.c",
        "modules/sbv-engine/tests/sdk_consumer.cpp",
        "modules/sbv-engine/tests/sdk_test.cpp",
        "modules/sbv-engine/tests/source_dataset_test.cpp",
        "modules/sbv-engine/tests/source_owners_test.cpp",
        "modules/sbv-engine/tests/split_test.cpp",
        "modules/sbv-engine/tests/stream_census_test.cpp",
        "modules/sbv-engine/tests/stream_model_test.cpp"
      ],
      "feature_id": "ssfv:symphony:sbv-engine",
      "how": "C++26 computations behind exact native process and installed C ABI; optional C++ ownership wrapper; external language consumers reuse identical native results and explicit contracts.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the exact owner contract and its focused checks."
        },
        {
          "language": "CMake",
          "role": "Builds, installs and receipts the independently controlled package."
        }
      ],
      "implementation_paths": [
        "modules/sbv-engine/cmake/SymphonySbvSdkConfig.cmake.in",
        "modules/sbv-engine/include/symphony/sbv/interop.hpp",
        "modules/sbv-engine/include/symphony/sbv/sdk.h",
        "modules/sbv-engine/include/symphony/sbv/sdk.hpp",
        "modules/sbv-engine/schemas/v1/node-stream.schema.json",
        "modules/sbv-engine/schemas/v1/partitioned-result.schema.json",
        "modules/sbv-engine/schemas/v1/result-page.schema.json",
        "modules/sbv-engine/src/allocation_economics.cpp",
        "modules/sbv-engine/src/analyze.cpp",
        "modules/sbv-engine/src/book.cpp",
        "modules/sbv-engine/src/bundle_operations.cpp",
        "modules/sbv-engine/src/census.cpp",
        "modules/sbv-engine/src/combinatorial.hpp",
        "modules/sbv-engine/src/compare.cpp",
        "modules/sbv-engine/src/compose.cpp",
        "modules/sbv-engine/src/compose_economics.cpp",
        "modules/sbv-engine/src/compose_joint.cpp",
        "modules/sbv-engine/src/dataset.cpp",
        "modules/sbv-engine/src/dataset.hpp",
        "modules/sbv-engine/src/economic_census_stream.cpp",
        "modules/sbv-engine/src/economic_census_stream.hpp",
        "modules/sbv-engine/src/economics.cpp",
        "modules/sbv-engine/src/economics_kernel.hpp",
        "modules/sbv-engine/src/economics_partitioned.cpp",
        "modules/sbv-engine/src/evaluate.cpp",
        "modules/sbv-engine/src/evaluate_partitioned.cpp",
        "modules/sbv-engine/src/exact_id_index.cpp",
        "modules/sbv-engine/src/exact_id_index.hpp",
        "modules/sbv-engine/src/experiment.cpp",
        "modules/sbv-engine/src/fit.cpp",
        "modules/sbv-engine/src/generate_census.cpp",
        "modules/sbv-engine/src/history.cpp",
        "modules/sbv-engine/src/liquidity.cpp",
        "modules/sbv-engine/src/logical_value.cpp",
        "modules/sbv-engine/src/logical_value.hpp",
        "modules/sbv-engine/src/model.cpp",
        "modules/sbv-engine/src/partitioned_output.cpp",
        "modules/sbv-engine/src/partitioned_output.hpp",
        "modules/sbv-engine/src/planning.cpp",
        "modules/sbv-engine/src/provider.cpp",
        "modules/sbv-engine/src/reader_extensions.inc",
        "modules/sbv-engine/src/resample.cpp",
        "modules/sbv-engine/src/resident.cpp",
        "modules/sbv-engine/src/result.cpp",
        "modules/sbv-engine/src/result_store.cpp",
        "modules/sbv-engine/src/result_store.hpp",
        "modules/sbv-engine/src/row_spool.cpp",
        "modules/sbv-engine/src/row_spool.hpp",
        "modules/sbv-engine/src/run.cpp",
        "modules/sbv-engine/src/run_math.hpp",
        "modules/sbv-engine/src/run_partitioned.cpp",
        "modules/sbv-engine/src/sdk.cpp",
        "modules/sbv-engine/src/select.cpp",
        "modules/sbv-engine/src/source_owners.cpp",
        "modules/sbv-engine/src/source_owners.hpp",
        "modules/sbv-engine/src/split.cpp",
        "modules/sbv-engine/src/stream_census.cpp",
        "modules/sbv-engine/src/stream_census.hpp",
        "modules/sbv-engine/src/stream_model.cpp",
        "modules/sbv-engine/src/stream_model.hpp",
        "modules/sbv-engine/src/streaming_sha256.hpp",
        "modules/sbv-engine/src/wide_rational.hpp",
        "modules/sbv-engine/tests/bundle_operations_test.cpp",
        "modules/sbv-engine/tests/dataset_test.cpp",
        "modules/sbv-engine/tests/economics_partitioned_test.cpp",
        "modules/sbv-engine/tests/evaluate_partitioned_test.cpp",
        "modules/sbv-engine/tests/exact_id_index_test.cpp",
        "modules/sbv-engine/tests/generate_partitioned_test.cpp",
        "modules/sbv-engine/tests/logical_value_test.cpp",
        "modules/sbv-engine/tests/partitioned_output_test.cpp",
        "modules/sbv-engine/tests/reader_extensions_test.cpp",
        "modules/sbv-engine/tests/result_store_test.cpp",
        "modules/sbv-engine/tests/row_spool_test.cpp",
        "modules/sbv-engine/tests/run_partitioned_test.cpp",
        "modules/sbv-engine/tests/stream_census_test.cpp",
        "modules/sbv-engine/tests/stream_model_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Experimental release; no live calls, calibrated fills, portfolio/account simulation, full study catalogue or performance ranking."
      ],
      "owner_contract": "modules/sbv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Decode exact Databento historical bytes through their existing native owner.",
          "target_feature_id": "ssfv:symphony:sqav-databento-dbn-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sbv-engine",
      "status": "experimental",
      "title": "Native historical experiments and portable results",
      "what": "Native multipass backtesting, replay, exact economics, selectable studies and comparison, deterministic bootstrap and durable local native trial execution. Temporal interval partitions expose expanding/rolling/explicit schedules, optional purging, embargo and availability filtering. Optional user-constrained resident DBN preloads share decoded events across native jobs, experiments, qxctl and SDK calls with explicit memory and lifecycle controls. Dataset bytes, records, metadata and load memory have nullable user-selected limits, with no built-in dataset-volume cap. Process v2 exposes optional user deadlines without an engine-selected duration or resident handshake timeout. Combinatorial temporal partitions expose exact fold addressing, complementary groups, per-group embargo and selected-page reuse counts. Native exact linear fitting and prediction preserve explicit weights, penalties, column identities and training-overlap disclosure. Explicit local dependency graphs bind immutable results, preserve resolved claims and block unavailable prerequisites without duplicate execution. Supplied research history records observation reuse, duplicate-artifact choices and explicit comparison-backed selections without a holdout gate. Retained native/imported census references and source-bound economic composition connect selected models to explicit heterogeneous or joint distributions. Independently compiled trusted native strategies and models use a plain C ownership ABI with exact MBO access, selected concurrency and provider/configuration provenance. Optional offline source ownership composes exact native SQAV/SQMV/SQFV/SQPV/SQDV interfaces, feeds file or retained originals into the same Dataset/RAM path, exports original/capture/manifest bytes, and carries explicit acknowledgement, release and publication-recovery evidence through terminal and SDK results. Explicit partitioned core producers preserve closed census identity through alternate models, heterogeneous economics, selected composition and self-contained replay; typed bundle input arrays and complete terminal/SDK streams avoid whole-result control frames.",
      "when": "Explicit caller-selected invocation, optional deadlines and constraints.",
      "where": "Independently versioned and installed owner package.",
      "who": "Direct native consumers, operators and qxctl agents.",
      "why": "Give users control of quantitative research choices and portable inspectable evidence."
    }
  ],
  "source_scope": "modules/sbv-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->

Experimental 0.17 adds explicit local experiment dependencies and immutable result-reference bindings for split/fit/predict graphs. Resolved claims, parent receipts, blocked states and stable wave order are terminal-queryable; exact-plan reconciliation reuses completed trials. See SPEC.md for scope and remaining orchestration work.
