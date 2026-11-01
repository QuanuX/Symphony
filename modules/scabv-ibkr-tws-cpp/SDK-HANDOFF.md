# External Stable SDK handoff

Selected release: **10.45**, the Stable release observed on 2026-09-28. Duncan selected Stable and confirmed no authorized local SDK. No vendor SDK has been downloaded, licensed, included, or tested here. A future Stable release requires an explicit compatibility increment.

## Authorized SDK compile gate

After its owner supplies the authorized release, independently confirm its distribution version and provenance. `tests/sdk1045-conformance` explicitly instantiates the public driver against vendor `EClient`, `Contract`, and `TagValueListSPtr`. It requires an absolute `IBKR_SDK_INCLUDE_DIR`, explicit `IBKR_SDK_RELEASE=10.45`, and exact installed `SymphonyScabvIbkrTws` 0.1.0 via `CMAKE_PREFIX_PATH`. This target performs no download or connection. Its release parameter is an assertion by the SDK owner, not SDK version attestation. Compilation has not been run because no authorized SDK is available.

## Runtime integration gate

The embedding SCABV SDK owner must supply the SDK library and authenticated TWS/Gateway read session and the concrete socket/EReader/EWrapper bridge. `Session` now supplies per-connection generations, monotonic read request IDs, conservative local pacing, account/readiness admission and finite-request lifecycle. It does not implement or attest that missing vendor bridge. Route `positionMulti`, `positionMultiEnd`, `historicalData`, `historicalDataEnd`, relevant request errors and disconnects to Session using the exact request ID and the generation captured when that reader was created. Convert vendor Decimal values through the SDK's exact decimal-string facility; do not convert them to binary floating point. Historical daily bar dates and intraday timestamps retain provider meaning; callback time strings are not invented event instants.

The Driver must outlive Collector and must not synchronously reenter it. Drive `poll` within the chosen deadline and stop policy. Match errors to request scope; do not treat unrelated connection notices as a completed request. Stop/cancel is local closure and dispatch, not a provider cancellation acknowledgment. Confirm selected account/model, initial end markers, disconnect/reconnect generation refusal, SDK precision, cancellation, pacing and private capture scope against the authorized deployment. Authentication/authorization must use its actual identity owner; fixture account names and caller-supplied strings provide none.

Both read requests and capture projections are deliberately finite. The TWS module exposes no order entry or live market-data request. Client Portal is a distinct adapter with its own authenticated HTTPS Gateway contract.

## Prepared host integration

The owner implements `RuntimeDriver` using its authorized SDK. Construct EClientSocket with the host EWrapper and signal; use synchronous connection handshake, then create/start EReader after the connection succeeds. `connect` must return within the chosen host deadline. The owner bounds signal waits and queued-message processing; `pump` calls EReader message processing on the serialized owner thread so callbacks can reach Session. Do not dispatch a read before both nextValidId and managed-account callbacks have admitted readiness. `nextValidId` supplies an order identifier; it is not reused as a read request ID. Hold an exclusive client ID and coordinate this session's budget with any other provider clients.

| Vendor callback | Host routing |
| --- | --- |
| `nextValidId` | `Session::next_valid_id(captured_generation, order_id)` |
| `managedAccounts` | Parse the exact bounded account list, then `managed_accounts`; no invented fallback account |
| `positionMulti` | Preserve exact account/model/contract and SDK Decimal string, then `position` |
| `positionMultiEnd` | `end` for that request ID |
| `historicalData` | Preserve time, exact price bits and SDK Decimal strings, then `historical_bar` |
| `historicalDataEnd` | `end` with the provider start/end evidence |
| Request error | `fail` for the exact request ID; unrelated/global notices are not request completion |
| `connectionClosed` or fatal connection failure | `disconnected` for the captured generation |

Before explicit reconnect, close and join the old reader, retain/release its terminal request, then create a new reader route capturing the new generation. Never relabel queued old callbacks with the newest generation. Session does not retry a failed read. The owner implements complete EWrapper overrides required by the exact SDK, without adding reachable order-entry methods to the read adapter. The independent conformance object currently checks only `Sdk1045Driver`; it cannot substitute for compiling/linking the complete host bridge and exact Decimal library.

Public references checked 2026-11-01: [connection handshake](https://www.interactivebrokers.com/docs/tws-api/doc/connectivity/establishing-an-api-connection), [EReader threading](https://www.interactivebrokers.com/docs/tws-api/doc/connectivity/e-reader-thread), [architecture and EWrapper ownership](https://www.interactivebrokers.com/docs/tws-api/doc/architecture), [pacing](https://www.interactivebrokers.com/docs/tws-api/doc/pacing-limitations), [historical pacing](https://www.interactivebrokers.com/docs/tws-api/doc/md/introduction), and [position callbacks](https://www.interactivebrokers.com/docs/tws-api/doc/accounts/positions). Current web documentation is integration guidance, not attestation of a privately supplied SDK 10.45 distribution.

## Qualification handoff: acceptance remains open

On 2026-11-01 the user confirmed that an authorized SDK and provider test profiles were not yet available and requested integration/qualification preparation. No vendor SDK download, provider connection, login or order operation is part of the local regressions.

1. Supply SDK 10.45 distribution provenance: authorized local archive/include/library paths, independently verified archive/version and hashes, applicable license/redistribution decision, and exact Decimal dependency. Do not put credentials or licensed vendor sources into repository evidence.
2. Build/install the module and run its native CTests. Compile the independent public consumer gate with the supplied SDK:

   ```sh
   cmake -S modules/scabv-ibkr-tws-cpp/tests/sdk1045-conformance -B /tmp/tws-1045-conformance -DCMAKE_PREFIX_PATH="$SYMPHONY_SDK_PREFIX" -DIBKR_SDK_INCLUDE_DIR="$AUTHORIZED_IBKR_INCLUDE" -DIBKR_SDK_RELEASE=10.45
   cmake --build /tmp/tws-1045-conformance
   ```

   These variables must be explicitly selected local paths. Compile/link the owner's complete RuntimeDriver/EWrapper/Decimal bridge separately against the same exact SDK; the command above establishes only request-template shape conformance.
3. Supply an already authenticated, explicitly authorized bounded read-only TWS/IB Gateway profile: application/version, loopback port, exclusive client ID, selected account/model/private-scope reference, allowed instruments/history window, limits, test window and cancellation/disconnect permission. No login automation or session takeover is implied.
4. On macOS ARM64, Ubuntu 24.04 ARM64, Ubuntu 24.04 amd64 and Ubuntu 26.04 amd64, verify readiness/account admission, exact Decimal quantities/volume/WAP, provider end markers, stop/cancel, disconnect/reconnect generation rejection and provider pacing with that bridge. Record source/tree, toolchain, SDK/Decimal provenance, host/profile references, exact read cases and outcomes, redacted request identities, and retained capture digests. Record actual provider cancel acknowledgments only if observed; dispatch alone is insufficient.
5. Review bounded read evidence before closing L15. Local fixtures, source stubs, compile-only conformance and a connected socket are insufficient operational acceptance. Physical DGX Spark and GPU execution remain untested; Ubuntu ARM64 is the user-approved sprint target.
