# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqav-fred-cpp/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Source evolution and scope claims remain reviewable through SCLV.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "The exact module contract, implementation and focused evidence have independent indexed routes.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [],
      "evidence": [
        "modules/sqav-fred-cpp/tests/test.cpp verifies the selected positive and refusal boundaries.",
        "modules/sqav-fred-cpp/tests/sdk-consumer/main.cpp verifies public installed admission and refusal independently of source include paths."
      ],
      "feature_id": "ssfv:symphony:sqav-fred-cpp",
      "how": "Provider surface: FRED unversioned HTTP API observed 2026-09-28; endpoints series/observations and series/vintagedates, JSON, ascending order. Explicit series, real-time start/end and observation dates prevent current-date defaults from changing identity. Observations use units=lin and output_type=1; actual physical units and seasonal metadata are not invented from that parameter. Exact date-level real-time bounds permit ALFRED snapshots without inventing publication instants. Page size 1\u2013100,000 for observations or 1\u201310,000 for vintage dates; uint32 offsets/counts checked before arithmetic. The response must echo the requested ranges, limit, offset and selected output settings. Records must fill the exact remaining page count. Missing value dot is distinct from zero; values remain decimal text. Capture preserves original JSON, request/vintage identity and page scope; a single page is complete only when offset zero covers the declared result count. No multi-request atomicity is implied. Partial pages expose the next offset for an explicitly bounded caller workflow. Collection uses the shared native HTTP boundary and an actual supplied credential-use bridge.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the module owner contract and focused native verification."
        },
        {
          "language": "CMake",
          "role": "Builds, exports, receipts, and guarded-uninstalls the exact library package."
        }
      ],
      "implementation_paths": [
        "modules/sqav-fred-cpp/CMakeLists.txt",
        "modules/sqav-fred-cpp/src/fred.cpp",
        "modules/sqav-fred-cpp/include/symphony/sqav/fred.hpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No FRED key provisioning, operational SSIAG bridge, arbitrary output types, frequency aggregation, series metadata API, cross-page snapshot atomicity or provider-authenticated fixtures."
      ],
      "owner_contract": "modules/sqav-fred-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Consumes the exact selected native dependency.",
          "target_feature_id": "ssfv:symphony:native-source-support-cpp",
          "type": "depends_on"
        },
        {
          "rationale": "Consumes the exact selected native dependency.",
          "target_feature_id": "ssfv:symphony:sqav-capture-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqav-fred-cpp",
      "status": "experimental",
      "title": "FRED and ALFRED native adapter",
      "what": "Exact observation and vintage-date requests, original decimal/missing values, pagination and capture.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "The independently installed sqav-fred-cpp library.",
      "who": "Explicit native C++26 callers within the selected source contract.",
      "why": "Exact observation and vintage-date requests, original decimal/missing values, pagination and capture."
    }
  ],
  "source_scope": "modules/sqav-fred-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
