# External Stable SDK handoff

Selected release: **10.45**, the Stable release observed on 2026-09-28. Duncan selected Stable and confirmed no authorized local SDK. No vendor SDK has been downloaded, licensed, included, or tested here. A future Stable release requires an explicit compatibility increment.

## Authorized SDK compile gate

After its owner supplies the authorized release, independently confirm its distribution version and provenance. `tests/sdk1045-conformance` explicitly instantiates the public driver against vendor `EClient`, `Contract`, and `TagValueListSPtr`. It requires an absolute `IBKR_SDK_INCLUDE_DIR`, explicit `IBKR_SDK_RELEASE=10.45`, and exact installed `SymphonyScabvIbkrTws` 0.1.0 via `CMAKE_PREFIX_PATH`. This target performs no download or connection. Its release parameter is an assertion by the SDK owner, not SDK version attestation. Compilation has not been run because no authorized SDK is available.

## Runtime integration gate

The embedding SCABV SDK owner must supply the SDK library and authenticated TWS/Gateway read session, EReader/event loop, stable session generation, unique request IDs and provider pacing. Route `positionMulti`, `positionMultiEnd`, `historicalData`, `historicalDataEnd`, relevant request errors and disconnects to the collector using the exact request ID and captured session generation. Convert vendor Decimal values through the SDK's exact decimal-string facility; do not convert them to binary floating point. Historical daily bar dates and intraday timestamps retain provider meaning; callback time strings are not invented event instants.

The Driver must outlive Collector and must not synchronously reenter it. Drive `poll` within the chosen deadline and stop policy. Match errors to request scope; do not treat unrelated connection notices as a completed request. Stop/cancel is local closure and dispatch, not a provider cancellation acknowledgment. Confirm selected account/model, initial end markers, disconnect/reconnect generation refusal, SDK precision, cancellation, pacing and private capture scope against the authorized deployment. Authentication/authorization must use its actual identity owner; fixture account names and caller-supplied strings provide none.

Both read requests and capture projections are deliberately finite. The TWS module exposes no order entry or live market-data request. Client Portal is a distinct adapter with its own authenticated HTTPS Gateway contract.
