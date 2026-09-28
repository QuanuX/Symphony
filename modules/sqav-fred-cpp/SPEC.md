# FRED and ALFRED native adapter: exact 0.1.0-dev contract

## Supported extent

Provider surface: FRED unversioned HTTP API observed 2026-09-28; endpoints series/observations and series/vintagedates, JSON, ascending order. Explicit series, real-time start/end and observation dates prevent current-date defaults from changing identity. Observations use units=lin and output_type=1; actual physical units and seasonal metadata are not invented from that parameter. Exact date-level real-time bounds permit ALFRED snapshots without inventing publication instants. Page size 1–100,000 for observations or 1–10,000 for vintage dates; uint32 offsets/counts checked before arithmetic. The response must echo the requested ranges, limit, offset and selected output settings. Records must fill the exact remaining page count. Missing value dot is distinct from zero; values remain decimal text. Capture preserves original JSON, request/vintage identity and page scope; a single page is complete only when offset zero covers the declared result count. No multi-request atomicity is implied. Partial pages expose the next offset for an explicitly bounded caller workflow. Collection uses the shared native HTTP boundary and an actual supplied credential-use bridge.

## Native/package boundary

C++26, initially verified on macOS amd64 / AppleClang 21 with compatible native consumers. Fallible preparation/admission preserves output on failure unless the API explicitly reports an attempted operation. Immutable handles may be retained; mutation, reset, destruction and move must not race use of the same handle. Source data is untrusted. Errors contain status only, never source contents or credentials. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev. No runtime discovery or implicit upgrade. The module exports no standalone process or qxctl command.

## Verification and limits

`tests/test.cpp` exercises supported positive and negative boundaries; `tests/sdk-consumer/main.cpp` exercises the installed public API outside the checkout. The SQV non-live source composition tests preserve selected capture evidence through exact local retention/replay. The tests and package receipts do not establish provider access, production conformance or a licence grant.

No FRED key provisioning, operational SSIAG bridge, arbitrary output types, frequency aggregation, series metadata API, cross-page snapshot atomicity or provider-authenticated fixtures.
