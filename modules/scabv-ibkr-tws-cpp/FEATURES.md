# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/scabv-ibkr-tws-cpp/SPEC.md",
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
        "modules/scabv-ibkr-tws-cpp/tests/test.cpp verifies the selected positive and refusal boundaries.",
        "modules/scabv-ibkr-tws-cpp/tests/sdk-consumer/main.cpp verifies public installed admission and refusal independently of source include paths."
      ],
      "feature_id": "ssfv:symphony:scabv-ibkr-tws-cpp",
      "how": "Duncan selected the Stable line, observed as exact SDK 10.45 on 2026-09-28, and confirmed no authorized local SDK. The adapter therefore includes a typed external-SDK template and tested callback collector, without downloading, accepting terms for or bundling IBKR code. Sdk1045Driver instantiates against the authorized EClient, Contract and TagValueListSPtr types. It calls only reqPositionsMulti/cancelPositionsMulti and reqHistoricalData/cancelHistoricalData. Historical requests use TRADES, useRTH=1, formatDate=2 and keepUpToDate=false. The embedding SDK owner must provide an authenticated account session, unique request IDs, provider pacing, event loop and exact callback routing. Collector binds account/model, request ID and session generation. Wrong generation/ID is refused; malformed or wrong-account matching callbacks close the request as failed. Finite records 1–100,000, callback payload 1 KiB–64 MiB plus a separately bounded 4 KiB snapshot header, and deadline 1–300,000 ms. Caller drives poll and its stop token; no hidden polling thread exists. End markers send cancellation and close local callback admission. Cancellation dispatch is not a provider acknowledgment. Disconnect, error, cancellation, resource or allocation failure never becomes complete data. Mutex serialization covers concurrent callbacks; reset/move/destruction must not race any method. Driver must outlive Collector and not reenter it synchronously. After fork methods refuse; destruction does not call the inherited Driver. Callback projections retain exact IEEE754 price bits, native decimal strings, selected contract fields and provider end evidence; these are explicitly selected callback fields, not original TWS wire frames. Complete means the initial provider response reached its end marker, not an atomic account state.",
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
        "modules/scabv-ibkr-tws-cpp/CMakeLists.txt",
        "modules/scabv-ibkr-tws-cpp/include/symphony/scabv/tws.hpp",
        "modules/scabv-ibkr-tws-cpp/src/tws.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No installed or licensed vendor SDK, verified SDK 10.45 conformance, connection/login runtime, complete EWrapper implementation, automatic request-ID allocation/pacing, live collection, order entry, provider cancellation receipt or operational SSIAG authorization."
      ],
      "owner_contract": "modules/scabv-ibkr-tws-cpp/SPEC.md",
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
      "source_scope": "modules/scabv-ibkr-tws-cpp",
      "status": "experimental",
      "title": "IBKR TWS Stable read adapter boundary",
      "what": "Finite read-only native request/callback state around the external Stable 10.45 SDK dependency.",
      "when": "Only when a compatible trusted caller explicitly invokes the installed library; installation starts no service.",
      "where": "The independently installed scabv-ibkr-tws-cpp library.",
      "who": "Explicit native C++26 callers within the selected source contract.",
      "why": "Finite read-only native request/callback state around the external Stable 10.45 SDK dependency."
    }
  ],
  "source_scope": "modules/scabv-ibkr-tws-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
