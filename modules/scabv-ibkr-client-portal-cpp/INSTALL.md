# Installing scabv-ibkr-client-portal-cpp

Exact release `0.1.0-dev`; CMake package `SymphonyScabvIbkrClientPortal` version `0.1.0` and public target `Symphony::ScabvIbkrClientPortal`. CMake >=3.30, C++26, single-configuration generator. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev.

Configure from this module or set `SYMPHONY_SCABVIBKRCLIENTPORTAL_INSTALLED=ON` with an explicit CMake prefix for exact installed dependencies. Source dependency builds exclude their install rules.

```sh
cmake -S modules/scabv-ibkr-client-portal-cpp -B /tmp/scabv-ibkr-client-portal-cpp-build -DCMAKE_BUILD_TYPE=Release
cmake --build /tmp/scabv-ibkr-client-portal-cpp-build
ctest --test-dir /tmp/scabv-ibkr-client-portal-cpp-build --output-on-failure
cmake --install /tmp/scabv-ibkr-client-portal-cpp-build --prefix /tmp/scabv-ibkr-client-portal-cpp-prefix
cmake -DINSTALL_PREFIX=/tmp/scabv-ibkr-client-portal-cpp-prefix -P /tmp/scabv-ibkr-client-portal-cpp-build/uninstall.cmake
```

Receipt-v2 owns only this version’s archive, header(s), exact CMake exports, six owner documents and AGPL license. Guarded removal checks exact receipt/digest/path ownership. Installation starts no service or provider operation. Operational credentials, provider host applications and rights remain separate.

## Prepared external Gateway integration

Create an `HttpsGatewayTransport` and a `SessionConfig` with an explicit trusted loopback HTTPS Gateway, selected account, private-scope reference, HTTP/JSON limits and a local admission lifetime. Keep the transport alive and serialize Session calls. `Session::open` checks the read-only status endpoint and discovers the selected account. Call `positions(page, stop, out)` for each explicitly authorized page and `historical_bars(history, cap, stop, out)` for the bounded history window. Respect the five-second account-discovery cooldown, including open before the first read; an early read returns `limit` without I/O. Check every status; `Page::next_page` proposes a subsequent page but causes no automatic request. Retain each original-byte Page/capture separately with partial coverage. `close` closes local admission, not the remote Gateway. There is no automatic login, keepalive, reauthentication, retry or provider logout.

`GatewayTransport` can be implemented by a separately authorized host transport. Such injection is a trusted-code boundary, not authenticated evidence; fixtures exercise this seam only. The default HTTPS transport keeps certificate/hostname verification enabled, does not accept ambient credentials/cookies or redirects, and exposes no TLS bypass. The current status route is [POST /iserver/auth/status](https://www.interactivebrokers.com/docs/web-api/api-reference/trading/trading-session/get-brokerage-status); account/position/history reads remain GET. This status POST reads state and does not initialize a brokerage session.

## Qualification handoff: acceptance remains open

On 2026-11-01 the user confirmed no provider test profile was available and requested preparation. The local `scabv-ibkr-client-portal-session` CTest uses injected responses and makes no live request. The `HttpsGatewayTransport` cancellation case is stopped before network access.

Before operational qualification, supply the externally authenticated Gateway distribution/version/provenance, trusted certificate installation, compatible account/session and market-data entitlements, selected account/private-scope reference, permitted instruments/history window/pages, bounds and test window. Session credentials remain with the authentication owner; do not copy them into command arguments, tests or evidence. Account/login prerequisites are described in [IBKR's Gateway lesson](https://www.interactivebrokers.com/campus/trading-lessons/launching-and-authenticating-the-gateway/); its example TLS bypass is not admitted by this adapter. Expiry/competing-session behavior is documented in [IBKR's authentication FAQ](https://www.interactivebrokers.com/docs/web-api/authentication/faq).

Use the same exact installed SDK on macOS ARM64, Ubuntu 24.04 ARM64, Ubuntu 24.04 amd64 and Ubuntu 26.04 amd64. Record source/tree, native toolchain, Gateway/TLS/profile references and exact bounded outcomes for status/discovery, correct-account positions through explicit empty-page exhaustion, history with exact original decimal bytes, account mismatch, local stop, actual session expiry and provider entitlement refusal. An expiry or entitlement refusal must be actually observed or remain unqualified; synthetic responses do not close that item. Do not revoke entitlements, log out competing users or force a provider session change without separately scoped authorization.

Every page remains a non-atomic read. Bind retained private captures by digest, omit raw account/credential/error-body content from public evidence, and record observed versus fixture-only cases separately. L16 remains open until these authorized provider cases pass. Physical DGX Spark/GPU execution is untested; Ubuntu ARM64 is the accepted sprint ARM target.
