# Bounded native source support: exact 0.1.0-dev contract

## Supported extent

JSON: 1–64 MiB input, 1–1,048,576 syntax events, caller-bounded strings, depth 1–64, duplicate-key and invalid UTF-8 refusal. Numbers may be inspected in the DOM; exact financial evidence remains the original bytes and is never reserialized from binary floating point. Date validation covers Gregorian years 1–9999. HTTP: 1–300,000 ms total deadline, positive connection deadline no greater than total, 1–65,536 header bytes, 1–64 MiB body. GET/POST only, explicit HTTPS origin/path, verified TLS, no redirects, proxies, netrc, cookies or diagnostics. HTTP error bodies are counted against the budget and discarded. Interrupted bodies cannot become complete when a credential callback swallows an error. The actual credential owner must enforce its authenticated lease/recipient and callback deadline; no operational SSIAG bridge is provided. FRED query-key and Databento Basic username are separate exact credential profiles. All secret-bearing URL/key copies are transient and locally overwritten; libcurl/system copies do not carry an impossible erasure guarantee. The production library has no test endpoint override. After fork, network calls refuse before using inherited library state.

## Native/package boundary

C++26, initially verified on macOS amd64 / AppleClang 21 with compatible native consumers. Fallible preparation/admission preserves output on failure unless the API explicitly reports an attempted operation. Immutable handles may be retained; mutation, reset, destruction and move must not race use of the same handle. Source data is untrusted. Errors contain status only, never source contents or credentials. Dependencies: CURL 8.7.1 SDK/system library, knowledge-vector-engine-cpp 0.2.0-dev. No runtime discovery or implicit upgrade. The module exports no standalone process or qxctl command.

## Verification and limits

`tests/test.cpp` exercises supported positive and negative boundaries; `tests/sdk-consumer/main.cpp` exercises the installed public API outside the checkout. The SQV non-live source composition tests preserve selected capture evidence through exact local retention/replay. The tests and package receipts do not establish provider access, production conformance or a licence grant.

No operational identity authority, automatic retry, durable spend authority, provider semantics, generic guarantee of bounded allocator overhead or knowledge-process payload envelope.
