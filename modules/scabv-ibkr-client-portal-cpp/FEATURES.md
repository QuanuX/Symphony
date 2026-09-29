# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/scabv-ibkr-client-portal-cpp/SPEC.md",
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
        "modules/scabv-ibkr-client-portal-cpp/tests/test.cpp verifies the selected positive and refusal boundaries.",
        "modules/scabv-ibkr-client-portal-cpp/tests/sdk-consumer/main.cpp verifies public installed admission and refusal independently of source include paths."
      ],
      "feature_id": "ssfv:symphony:scabv-ibkr-client-portal-cpp",
      "how": "Distinct Client Portal v1 HTTP surface observed 2026-09-28. Requires an already authenticated local Gateway with a trusted TLS certificate at an explicitly selected https://localhost:port or https://127.0.0.1:port. Account discovery GET /v1/api/portfolio/accounts precedes a binding; the selected account must appear once in the bounded response and scope must use this profile’s private: reference grammar. Admission of caller-supplied bytes does not authenticate them. Read plans expose only GET portfolio/{account}/positions/{page} and iserver/marketdata/history. Position pages are 0–9,999, with at most 100 records, exact acctId and nonduplicated conid. An empty page indicates exhaustion; nonempty pages expose the next page. Each request remains partial evidence, not an atomic multi-page account snapshot. Historical plans require conid, explicit UTC startTime, period/bar, direction=-1, source=Last and explicit regular-hours choice. Local bar cap is 1–1,000; provider interval/entitlement rules still apply. Selected numeric fields and increasing bar timestamps are shape-checked, while every original JSON byte—including financial decimals and additional provider fields—is retained. No binary-float reserialization occurs.",
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
        "modules/scabv-ibkr-client-portal-cpp/CMakeLists.txt",
        "modules/scabv-ibkr-client-portal-cpp/include/symphony/scabv/client_portal.hpp",
        "modules/scabv-ibkr-client-portal-cpp/src/client_portal.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No session login, account mutation, order entry, OAuth replacement, TLS bypass, TWS protocol, atomic account snapshot, production entitlement/conformance claim or operational SSIAG authorization."
      ],
      "owner_contract": "modules/scabv-ibkr-client-portal-cpp/SPEC.md",
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
      "source_scope": "modules/scabv-ibkr-client-portal-cpp",
      "status": "experimental",
      "title": "IBKR Client Portal read adapter",
      "what": "Exact private-account binding and bounded Client Portal account, position and historical-bar reads.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "The independently installed scabv-ibkr-client-portal-cpp library.",
      "who": "Explicit native C++26 callers within the selected source contract.",
      "why": "Exact private-account binding and bounded Client Portal account, position and historical-bar reads."
    }
  ],
  "source_scope": "modules/scabv-ibkr-client-portal-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
