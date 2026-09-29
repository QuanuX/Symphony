# Symphony Quantitative Acquisition Vector Manifest

## Canonical Target

`knowledge/sqv/sqav/`

## Declared Contract Truth Role

SQAV owns research-data source collection semantics beneath SQV. It preserves source-specific authority and evidence rather than defining one universal provider interface.

## Canonical Surfaces

- `knowledge/sqv/sqav/INTENT.md`
- `knowledge/sqv/sqav/MANIFEST.md`
- `knowledge/sqv/sqav/SPEC.md`
- `knowledge/sqv/sqav/SKILL.md`
- `knowledge/sqv/sqav/DATABENTO-SSIAG-BINDING.md`

## Implementation Status

`modules/sqav-capture-cpp/` owns the `0.2.0-dev` C++26 optional offline original-byte capture library, its exact binary format, metadata/flow bridge and package. The current provider and request-administration extents are recorded below; live remains deferred. No `sqav:` identity family is allocated by this Quad.

## Language Boundary

First-party research-data collection and parsing on the data plane are native C++. Exact adapter dependencies, ABI, target, and provider compatibility require separate admission and tests.

## Non-Authorization Statement

This manifest authorizes no provider session, credential use, account access, subscription, data purchase, live order, or redistribution.

## Databento provider-file extent

`modules/sqav-databento-dbn-cpp/MANIFEST.md` declares the separately installable `0.5.0-dev` C++26 DBNv1/v3 single-schema MBO file inspector and original-byte capture bridge. It includes bounded historical request/response handling. Separate reference support is admitted below; live remains deferred.


## Broader non-live source extent

The independently selectable C++26 0.1.0-dev modules `native-source-support-cpp`, `sqav-fred-cpp`, `sqav-news-api-cpp` and `sqav-databento-reference-cpp` add bounded HTTPS/JSON support, FRED/ALFRED observation/vintage pages, vendor-neutral original/corrected news article ingress, and Databento reference/cost response handling. Each module SPEC owns exact operations, bounds, response fidelity and non-claims. Provider authentication, credentials and entitlements are external prerequisites. No source activation is established by fixture evidence.

`knowledge/sqv/scabv/SPEC.md` admits separate IBKR Client Portal and TWS read bindings; their captures consume the existing research-data path. TWS targets external Stable SDK 10.45 without bundled vendor code or actual SDK conformance. Live remains future work. The shared HTTP CredentialUse interface is a required composition boundary, not an operational SSIAG bridge. Databento remains mapped to SSIAG/local Keychain in `knowledge/sqv/sqav/DATABENTO-SSIAG-BINDING.md`.

Focused source tests preserve all five adapter capture families through reopened local retention/replay. Provider schemas and source scope remain in original JSON/JSONL or explicitly declared TWS callback projections; exact financial bytes and price bits are not silently reserialized from binary floating-point values. API availability does not imply completion of all SQV programme objectives.

## Local request administration

`modules/sqav-request-engine/MANIFEST.md` admits one C++26 0.1.0-dev process operation calling the FRED/ALFRED, Databento historical and reference request validators. qxctl adds exactly `sqv acquisition validate`, `sqv schema`, and `sqv template`; schema/template select `--operation request_validate`, adapter and exact installed release. Validation establishes no provider, credential, entitlement, acquisition or store state. Shared module/inventory discovery exposes source contracts separately from lifecycle receipt observations.

## Non-live administration

`modules/sqav-attempt-engine/MANIFEST.md` and `modules/sqav-attempt-engine/SPEC.md` declare the independently installed C++26 owner adapter. Operations: `attempts_inspect`. Shared qxctl schema/template and source discovery supply the resource surface. Live runtime activation remains deferred.
