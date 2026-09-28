# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sqav-databento-reference-cpp/SPEC.md",
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
        "modules/sqav-databento-reference-cpp/tests/test.cpp verifies the selected positive and refusal boundaries.",
        "modules/sqav-databento-reference-cpp/tests/sdk-consumer/main.cpp verifies public installed admission and refusal independently of source include paths."
      ],
      "feature_id": "ssfv:symphony:sqav-databento-reference-cpp",
      "how": "Reference HTTP v0 observed 2026-09-28, parameters cross-checked against official databento-python v0.84.0 source (research evidence only, no Python runtime). Corporate actions get_range, adjustment factors get_range, security master get_range/get_last support 1\u2013128 explicit raw symbols; ranges use explicit ISO dates with start before end. All requests set allocate_isins=false and compression=zstd; corporate index is event_date, security range index ts_effective. No ALL_SYMBOLS or implicit current range is selected. Point-in-time rows and nested fields are retained without flattening or latest-record collapse. Exact zstd 1.5.7 static dependency; compressed and expanded limits each 1\u201364 MiB, at most 65,536 rows, bounded JSON per record and decoder window log 10\u201323. Full compressed-frame completion is mandatory; truncation, trailing garbage, excess output and malformed/duplicate JSON fail atomically. Captures preserve original compressed bytes; decoded JSONL is also available unchanged. Identifier shape validation is deliberately narrower than full provider schema semantics. Reference coverage remains partial because allocation filtering and provider entitlements can omit rows. CostQuote uses metadata.get_cost for the exact existing historical MBO request. Positive/zero JSON decimal USD is converted upward to nano-USD using integer decimal arithmetic, with overflow refusal and no binary-float conversion. Quotes bind exact request, caller-recorded time and 1\u201386,400,000 ms validity; only matching, unexpired quotes produce AttemptQuote inputs for the durable ledger. Provider estimates are not guaranteed invoices or account-wide budget authority.",
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
        "modules/sqav-databento-reference-cpp/CMakeLists.txt",
        "modules/sqav-databento-reference-cpp/src/reference.cpp",
        "modules/sqav-databento-reference-cpp/include/symphony/sqav/databento/reference.hpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No provider activation, reference entitlement guarantee, source authenticity from fixtures, new ISIN allocation, live data, full reference-schema normalization, account-wide billing reconciliation, automatic paid retry or operational SSIAG bridge."
      ],
      "owner_contract": "modules/sqav-databento-reference-cpp/SPEC.md",
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
        },
        {
          "rationale": "Consumes the exact selected native dependency.",
          "target_feature_id": "ssfv:symphony:sqav-databento-dbn-cpp",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sqav-databento-reference-cpp",
      "status": "experimental",
      "title": "Databento native reference and cost adapter",
      "what": "Bounded Zstandard/JSONL reference responses and exact request-bound historical cost quotes.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "The independently installed sqav-databento-reference-cpp library.",
      "who": "Explicit native C++26 callers within the selected source contract.",
      "why": "Bounded Zstandard/JSONL reference responses and exact request-bound historical cost quotes."
    }
  ],
  "source_scope": "modules/sqav-databento-reference-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
