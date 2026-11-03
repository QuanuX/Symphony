# MVP parser example

`mvp-example.openapi.json` is synthetic test content, not a canonical API contract or installed service. Its `/fixture`, mutual-TLS scheme and fixture security-profile reference are inherited from the engine's disposable test repository; they do not describe Symphony's actual SSIAG transport or grant caller authority. The canonical SACV registry remains empty. Nothing here registers, publishes or starts an endpoint.

The shipped example uses OpenAPI **3.2.0 JSON**, effective security and an internal `#/components/schemas/FixtureValue` reference. The SACV native suite reads these exact example bytes into its disposable registry and requires successful validation. It then replaces that reference with a relative external file and an HTTPS reference and requires `sacv.reference.external_unavailable`. Existing YAML tests require `sacv.document.parser_unavailable`; parser unavailability must not be relabeled as an invalid canonical contract.

| Input | Current engine behavior |
| --- | --- |
| Bounded JSON OpenAPI 3.2.0; internal document references | Implemented profile validation |
| `.openapi.yaml` entry | Explicit parser-unavailable finding; no fallback |
| External file or remote `$ref` | Explicit external-reference-unavailable finding; no network fetch |
| Downgraded OpenAPI, secret-shaped example, identity/security drift | Profile violation |
| Publish, apply, generate or dock | Unsupported operation |

Canonical governance may permit owner-justified YAML or split files; this engine version does not implement those parser paths. No YAML/external-reference support is promised for the MVP parser. Do not silently convert, downgrade, register this fixture, invent a production server URL, or infer publication approval from conformance.

Run the `sacv-engine-tests` and `sacv-engine-process-smoke` CTest targets from a build of `modules/sacv-engine` with `BUILD_TESTING=ON`. The same tests are part of installed knowledge-group qualification.
