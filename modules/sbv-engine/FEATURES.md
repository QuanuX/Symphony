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
        "modules/sbv-engine/tests/dataset_test.cpp",
        "modules/sbv-engine/tests/economics_test.cpp",
        "modules/sbv-engine/tests/interop_consumer.cpp",
        "modules/sbv-engine/tests/liquidity_test.cpp",
        "modules/sbv-engine/tests/models_test.cpp",
        "modules/sbv-engine/tests/planning_test.cpp",
        "modules/sbv-engine/tests/research_test.cpp",
        "modules/sbv-engine/tests/sbv_test.cpp",
        "modules/sbv-engine/tests/sdk_consumer.c",
        "modules/sbv-engine/tests/sdk_consumer.cpp",
        "modules/sbv-engine/tests/sdk_test.cpp",
        "modules/sbv-engine/tests/split_test.cpp"
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
        "modules/sbv-engine/src/allocation_economics.cpp",
        "modules/sbv-engine/src/analyze.cpp",
        "modules/sbv-engine/src/book.cpp",
        "modules/sbv-engine/src/compare.cpp",
        "modules/sbv-engine/src/compose.cpp",
        "modules/sbv-engine/src/compose_joint.cpp",
        "modules/sbv-engine/src/dataset.cpp",
        "modules/sbv-engine/src/dataset.hpp",
        "modules/sbv-engine/src/economics.cpp",
        "modules/sbv-engine/src/evaluate.cpp",
        "modules/sbv-engine/src/experiment.cpp",
        "modules/sbv-engine/src/liquidity.cpp",
        "modules/sbv-engine/src/model.cpp",
        "modules/sbv-engine/src/planning.cpp",
        "modules/sbv-engine/src/resample.cpp",
        "modules/sbv-engine/src/resident.cpp",
        "modules/sbv-engine/src/result.cpp",
        "modules/sbv-engine/src/run.cpp",
        "modules/sbv-engine/src/sdk.cpp",
        "modules/sbv-engine/src/select.cpp",
        "modules/sbv-engine/src/split.cpp",
        "modules/sbv-engine/src/wide_rational.hpp",
        "modules/sbv-engine/tests/dataset_test.cpp"
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
      "what": "Native multipass backtesting, replay, exact economics, selectable studies and comparison, deterministic bootstrap and durable local native trial execution. Temporal interval partitions expose expanding/rolling/explicit schedules, optional purging, embargo and availability filtering. Optional user-constrained resident DBN preloads share decoded events across native jobs, experiments, qxctl and SDK calls with explicit memory and lifecycle controls. Dataset bytes, records, metadata and load memory have nullable user-selected limits, with no built-in dataset-volume cap.",
      "when": "Explicit bounded caller-selected invocation.",
      "where": "Independently versioned and installed owner package.",
      "who": "Direct native consumers, operators and qxctl agents.",
      "why": "Give users control of quantitative research choices and portable inspectable evidence."
    }
  ],
  "source_scope": "modules/sbv-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
