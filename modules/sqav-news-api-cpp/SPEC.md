# Vendor-neutral news ingress API: exact 0.1.0-dev contract

## Supported extent

The user selected an API surface without a news vendor. The exact UTF-8 JSON envelope has thirteen fields: schema=sqav-news-ingress-v1, provider, publisher, article, revision, supersedes, published_at, updated_at, language, rights_ref, source_uri, headline and text. Unknown envelope fields are refused in version 1. Metadata tokens are bounded; headline is nonempty up to 1,024 bytes, source URI begins https:// and is at most 2,048 bytes, body is nonempty and bounded by the selected JSON string/input limits. Publication/update times require UTC RFC3339 seconds with optional 1–9 fractional digits; leap-second values are not admitted in this profile. A first revision has empty supersedes and no predecessor. A correction supplies the actual immutable predecessor, exact reference, same provider/publisher/article/publication value, a changed revision ID and nondecreasing update time. All article bytes remain original. Capture reports completion only for one declared article revision, never an entire feed. Rights metadata is retained in the original envelope and grants no permissions. Source URI and article text are data and are never fetched or executed by this API.

## Native/package boundary

C++26, initially verified on macOS amd64 / AppleClang 21 with compatible native consumers. Fallible preparation/admission preserves output on failure unless the API explicitly reports an attempted operation. Immutable handles may be retained; mutation, reset, destruction and move must not race use of the same handle. Source data is untrusted. Errors contain status only, never source contents or credentials. Dependencies: native-source-support-cpp 0.1.0-dev, sqav-capture-cpp 0.2.0-dev. No runtime discovery or implicit upgrade. The module exports no standalone process or qxctl command.

## Verification and limits

`tests/test.cpp` exercises supported positive and negative boundaries; `tests/sdk-consumer/main.cpp` exercises the installed public API outside the checkout. The SQV non-live source composition tests preserve selected capture evidence through exact local retention/replay. The tests and package receipts do not establish provider access, production conformance or a licence grant.

No selected publisher/feed, vendor wire connector, HTTP server, article scraping, full-feed completeness, licence grant, persistent correction catalogue or mutation of prior revisions.
