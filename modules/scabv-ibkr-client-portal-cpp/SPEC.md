# IBKR Client Portal read adapter: exact 0.1.0-dev contract

## Supported extent

Distinct Client Portal v1 HTTP surface observed 2026-09-28. Requires an already authenticated local Gateway with a trusted TLS certificate at an explicitly selected https://localhost:port or https://127.0.0.1:port. Account discovery GET /v1/api/portfolio/accounts precedes a binding; the selected account must appear once in the bounded response and scope must use this profile’s private: reference grammar. Admission of caller-supplied bytes does not authenticate them. Read plans expose only GET portfolio/{account}/positions/{page} and iserver/marketdata/history. Position pages are 0–9,999, with at most 100 records, exact acctId and nonduplicated conid. An empty page indicates exhaustion; nonempty pages expose the next page. Each request remains partial evidence, not an atomic multi-page account snapshot. Historical plans require conid, explicit UTC startTime, period/bar, direction=-1, source=Last and explicit regular-hours choice. Local bar cap is 1–1,000; provider interval/entitlement rules still apply. Selected numeric fields and increasing bar timestamps are shape-checked, while every original JSON byte—including financial decimals and additional provider fields—is retained. No binary-float reserialization occurs.

## Native/package boundary

C++26, initially verified on macOS amd64 / AppleClang 21 with compatible native consumers. Fallible preparation/admission preserves output on failure unless the API explicitly reports an attempted operation. Immutable handles may be retained; mutation, reset, destruction and move must not race use of the same handle. Source data is untrusted. Errors contain status only, never source contents or credentials. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev. No runtime discovery or implicit upgrade. The module exports no standalone process or qxctl command.

## Verification and limits

`tests/test.cpp` exercises supported positive and negative boundaries; `tests/sdk-consumer/main.cpp` exercises the installed public API outside the checkout. The SQV non-live source composition tests preserve selected capture evidence through exact local retention/replay. The tests and package receipts do not establish provider access, production conformance or a licence grant.

No session login, account mutation, order entry, OAuth replacement, TLS bypass, TWS protocol, atomic account snapshot, production entitlement/conformance claim or operational SSIAG authorization.
