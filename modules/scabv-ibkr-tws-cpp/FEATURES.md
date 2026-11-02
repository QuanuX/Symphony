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
        "modules/scabv-ibkr-tws-cpp/tests/sdk-consumer/main.cpp verifies public installed admission and refusal independently of source include paths.",
        "modules/scabv-ibkr-tws-cpp/tests/session.cpp verifies the host session lifecycle with a local injected transport; no SDK or live-provider qualification is claimed."
      ],
      "feature_id": "ssfv:symphony:scabv-ibkr-tws-cpp",
      "how": "Duncan selected the Stable line, observed as exact SDK 10.45 on 2026-09-28, and confirmed no authorized local SDK. The adapter therefore includes a typed external-SDK template and tested callback collector, without downloading, accepting terms for or bundling IBKR code. Sdk1045Driver instantiates against the authorized EClient, Contract and TagValueListSPtr types. It calls only reqPositionsMulti/cancelPositionsMulti and reqHistoricalData/cancelHistoricalData. Historical requests use TRADES, useRTH=1, formatDate=2 and keepUpToDate=false. The embedding SDK owner must provide an authenticated account session, the concrete vendor transport and exact callback routing. The optional host Session supplies read request IDs, local pacing and lifecycle admission within one exclusive client-ID owner. Collector binds account/model, request ID and session generation. Wrong generation/ID is refused; malformed or wrong-account matching callbacks close the request as failed. Finite records 1–100,000, callback payload 1 KiB–64 MiB plus a separately bounded 4 KiB snapshot header, and deadline 1–300,000 ms. Caller drives poll and its stop token; no hidden polling thread exists. End markers send cancellation and close local callback admission. Cancellation dispatch is not a provider acknowledgment. Disconnect, error, cancellation, resource or allocation failure never becomes complete data. Mutex serialization covers concurrent callbacks; reset/move/destruction must not race any method. Driver must outlive Collector and not reenter it synchronously. After fork methods refuse; destruction does not call the inherited Driver. Callback projections retain exact IEEE754 price bits, native decimal strings, selected contract fields and provider end evidence; these are explicitly selected callback fields, not original TWS wire frames. Complete means the initial provider response reached its end marker, not an atomic account state. Host session ownership: `Session` is an optional serialized host coordinator over `RuntimeDriver`, not a vendor SDK implementation. The runtime connects only the explicitly selected loopback host/port/client ID; login remains external. Each connection attempt receives a new generation derived from a caller-unique session tag. `nextValidId` admits readiness only and does not seed read IDs; a bounded, duplicate-free `managedAccounts` list must also contain the selected account. The host converts the provider list without inventing entries. A failed or timed-out handshake closes the transport. Session callbacks from earlier generations and unrelated request IDs cannot affect the active collector. Only one finite read may be active. Request identity fields must be unset; the coordinator inserts its bound account/model/private scope and monotonically allocates positive signed-32-bit read IDs. An attempted ID is never reused, including failed dispatch or reconnect; exhaustion refuses. Host ownership of the client ID and no competing request allocator are prerequisites. Starts are separated by at least 100 ms; historical starts by at least 16 seconds across reconnects. This conservative local budget covers this Session only; the host must coordinate other sessions and provider-specific limits. Refusals return `limit` without queueing or sleeping. Invalid attempts may consume a local ID/budget, but never dispatch an invalid request. The runtime pump must drain bounded queued EReader input on the owner thread and return; the host owns socket/reader shutdown and joining. Only pump may synchronously deliver callbacks. Other runtime methods must not reenter Session or Collector. Methods, destruction and reset are serialized by the host. Forked children refuse without calling the inherited runtime. Deadline/stop, failed dispatch and disconnect leave terminal gap snapshots. A terminal request must be explicitly released before another start or reconnect. There is no automatic reconnect, replay or cancellation acknowledgment. A preflight open failure preserves output; once connection dispatch starts a failed Session is retained. `poll` returns `stale` when an end callback completed the collector; `snapshot` then reports its terminal state.\n",
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
        "modules/scabv-ibkr-tws-cpp/src/session.cpp",
        "modules/scabv-ibkr-tws-cpp/src/tws.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "No installed or licensed vendor SDK, verified SDK 10.45 conformance, concrete vendor socket/EReader/EWrapper/Decimal bridge, login automation, live collection, order entry, provider cancellation receipt or operational SSIAG authorization. Host session fixtures do not establish SDK or operational acceptance."
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
