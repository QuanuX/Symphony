# Symphony C++ API Broker Vector Specification

## Admitted ownership

SCABV owns non-FIX broker-specific request, account/session binding and response interpretation. SQAV consumes its attributable read captures through the existing SQAV capture contract; SQMV/SQFV/SQPV/SQDV retain their independent catalogue, movement, retention and delivery ownership. SSIAG owns credential/identity authority. Private account scope is explicit; caller-supplied provider responses do not authenticate themselves.

## Exact initial contracts

- `modules/scabv-ibkr-client-portal-cpp/SPEC.md`: independently installed C++26 Client Portal v1 account discovery, positions and bounded historical bars through an already authenticated local HTTPS Gateway.
- `modules/scabv-ibkr-tws-cpp/SPEC.md`: independently installed C++26 finite read request/callback boundary targeting the externally supplied Stable 10.45 SDK, selected 2026-09-28. The Stable alias cannot silently change the exact SDK contract. No authorized SDK is installed; type-shape fixture tests are not vendor SDK conformance. Actual authenticated session, SDK event loop, EWrapper callback routing, request-ID uniqueness and pacing remain embedding prerequisites.

End markers describe the selected initial response, not an atomic account snapshot. Disconnects, mismatched generations, cancellation, overflow and errors cannot become complete responses. All future adapters require their own exact provider/version/operation and conformance evidence.

## Non-authorization

No broker login, account mutation, order entry, live collection, credentials, provider subscription, SDK license acceptance or redistribution authority is conferred by this contract. No SCABV colon identity family, graph engine, process, qxctl surface or complete multi-broker abstraction is introduced.
