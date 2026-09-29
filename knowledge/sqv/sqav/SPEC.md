# Symphony Quantitative Acquisition Vector Specification

## Source Binding

An admitted collector identifies the exact provider surface, operation, endpoint or SDK and adapter release, source dataset, requested scope, supported target, authorized credential reference, and finite resource and provider limits. Unversioned provider interfaces require dated source evidence; an adapter must not silently replace an older binding with a newer one. Provider-required external applications are declared separately from the first-party C++ implementation.

Acquisition records distinguish provider event, publication, revision, and receipt times where evidenced from Symphony acquisition time. Their precision, calendar, and uncertainty remain explicit. Source order is not automatically Symphony transfer order or a total order across providers.

## Collection and Coverage

The selected collector defines finite requests or a bounded long-running session, its cursor scope, retry and reconnect policy, and supported backfill. Pagination, rate limits, partial responses, skipped records, replay expiry, revisions, and missing observations produce attributable coverage or gap evidence. A successful request or active socket alone does not prove a complete dataset.

Provider payloads retain their actual rights and access scope; collection does not make them public knowledge. Rights to collect, retain, transform, share, model, and redistribute are distinct where the source distinguishes them. Source output and downloaded metadata are untrusted data, not instructions. Routine payloads do not pass through qxctl, a knowledge query, or an audit service.

## Native and Integration Boundary

First-party collection, parsing, and provider-facing movement are native C++. Suitable native dependencies are explicit and versioned. IBKR research collection must consume a separately admitted SCABV-owned read-only provider binding; SQAV does not fabricate that adapter or a duplicate broker authority. FIX remains SOOV-owned. No listed source is operational until its exact connector and conformance evidence exist.

## First Native Capture Contract

`modules/sqav-capture-cpp/SPEC.md` owns the first `0.1.0-dev` C++26 optional offline capture contract. Bounded original bytes and attributable exact source/interface/adapter fields, attempt identity, native position, distinct time roles/precision, coverage assertions and optional source count have an immutable encoding. The source identity and capture identity remain distinct from SQFV transfer sequence. An exact SQMV bridge admits the capture envelope through existing movement, retention and delivery owners.

This library resolves no credentials, reads no source file, opens no provider connection and interprets no provider schema. Exact source operations, provider parsers, access enforcement, reconnect/backfill and workload thresholds need separate admission. A caller's coverage assertion or digest does not establish source authority. Other native representations remain valid choices; no new colon namespace or SQV qxctl surface is allocated.

## First provider-file contract

`modules/sqav-databento-dbn-cpp/SPEC.md` admits bounded uncompressed DBNv1/v3 single-schema MBO inspection and exact capture binding. All original file bytes and raw integer/time fields remain preserved. Public official fixtures anchor this narrow conformance claim. Network acquisition, compression, mixed live records, reference data, book reconstruction and source authenticity/entitlements remain outside this release.

## Non-Authorization Statement

SQAV does not authorize a provider session, credential use, subscription, data purchase, account access, order entry, or redistribution.

## Selected Databento credential authority

Duncan selected SSIAG with its local Keychain provider for the Databento source.
`knowledge/sqv/sqav/DATABENTO-SSIAG-BINDING.md` maps the historical, live and
reference surfaces to exact SSIAG identity, credential-reference, provider and
lease ownership. The current provider is metadata-only; the map preserves its
operational blockers and does not create a secret-access protocol or fallback.


## Broader non-live source extent

The independently selectable C++26 0.1.0-dev modules `native-source-support-cpp`, `sqav-fred-cpp`, `sqav-news-api-cpp` and `sqav-databento-reference-cpp` add bounded HTTPS/JSON support, FRED/ALFRED observation/vintage pages, vendor-neutral original/corrected news article ingress, and Databento reference/cost response handling. Each module SPEC owns exact operations, bounds, response fidelity and non-claims. Provider authentication, credentials and entitlements are external prerequisites. No source activation is established by fixture evidence.

`knowledge/sqv/scabv/SPEC.md` admits separate IBKR Client Portal and TWS read bindings; their captures consume the existing research-data path. TWS targets external Stable SDK 10.45 without bundled vendor code or actual SDK conformance. Live remains future work. The shared HTTP CredentialUse interface is a required composition boundary, not an operational SSIAG bridge. Databento remains mapped to SSIAG/local Keychain in `knowledge/sqv/sqav/DATABENTO-SSIAG-BINDING.md`.

Focused source tests preserve all five adapter capture families through reopened local retention/replay. Provider schemas and source scope remain in original JSON/JSONL or explicitly declared TWS callback projections; exact financial bytes and price bits are not silently reserialized from binary floating-point values. API availability does not imply completion of all SQV programme objectives.
