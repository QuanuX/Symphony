# Symphony Quantitative Vector Manifest

## Canonical Target

`knowledge/sqv/`

## Declared Contract Truth Role

SQV owns Symphony-authored quantitative framework semantics while preserving complete user ownership of strategy logic.

## Canonical Surfaces

- `knowledge/sqv/INTENT.md`
- `knowledge/sqv/MANIFEST.md`
- `knowledge/sqv/SPEC.md`
- `knowledge/sqv/SKILL.md`
- `knowledge/sqv/RESEARCH-DATA.md`
- `tests/sqv-research-pipeline/CMakeLists.txt`
- `tests/sqv-research-pipeline/pipeline.cpp`
- `tests/sqv-research-pipeline/nonlive.cpp`
- `tests/sqv-research-pipeline/preview_progress.cpp`

## Research-Data Child Posture

The six research-data children have Contract Quads under `knowledge/sqv/sqav/`, `knowledge/sqv/sqmv/`, `knowledge/sqv/sqfv/`, `knowledge/sqv/sqtv/`, `knowledge/sqv/sqpv/`, and `knowledge/sqv/sqdv/`. All six research-data owners have bounded first native libraries described below. Root knowledge-manifest and SKVI routes make these separate owners discoverable. SOOV's separate FIX Quad remains under `knowledge/sqv/soov/`.

## Implementation Status

SQFV's `modules/sqfv-batch-cpp/` owns the independently installable `0.3.0-dev` C++26 trusted-process batch library, with immutable payloads, explicit leases, independent port credits/cursors, and a local frame codec. SQMV's `modules/sqmv-metadata-cpp/` `0.2.0-dev` resolves bounded immutable manifests into exact metadata bindings. SQPV's `modules/sqpv-local-store-cpp/` `0.2.0-dev` retains one exact bound stream in a finite local filesystem store with explicit commit/recovery behavior. SQDV's `modules/sqdv-delivery-cpp/` `0.3.0-dev` supplies trusted same-process full-batch delivery, exact retained/live resume and independent processing acknowledgements. Each module's own contract and focused evidence define its limited capability. SQAV's `modules/sqav-capture-cpp/` `0.2.0-dev` preserves bounded original bytes and attributable acquisition evidence with an exact metadata/flow bridge. SQTV's `modules/sqtv-integer-conversion-cpp/` `0.2.0-dev` performs exact dense integer representation conversion with derived lineage. SOOV remains a separate architecture owner without an implemented module. SQAV additionally owns `sqav-databento-dbn-cpp` `0.5.0-dev` for bounded DBNv1/v3 MBO provider-file inspection and exact capture. Provider acquisition, further domain interpretation, transport adapters, richer retention policies and consumer integrations remain later increments. The older `tests/sqv-research-prototype/` fixture is historical development evidence. None of these Quads allocates a colon identity family or SQV qxctl command.

## Non-Authorization Statement

This manifest does not authorize Symphony to inspect, rewrite, constrain, execute, deploy, or publish user strategy logic.

## Scoped offline pipeline

The coordinated local profile composes all six research-data owners: attributable
capture and exact metadata, optional dense integer conversion, independent flow
credits, bounded asynchronous local retention, non-durable preview and confirmed
replay. `tests/sqv-research-pipeline` runs the same public-interface fixture from
source or installed packages. The worker and proof interfaces are optional;
existing disposable and retention-before-delivery profiles remain independent.
This profile does not complete production source connectors, credential delivery,
new representations, payload IPC/network transport or external destination adapters.

## Bounded historical source planning

The Databento adapter 0.4.0-dev adds canonical finite historical MBO plans,
chunked bounded response assembly, exact selection/time checks, attributable
complete/partial/gap outcomes and conservative retry/split recommendations.
Original request and outcome evidence survives asynchronous capture retention
and replay. This advances source-facing preparation while actual network
execution, persistent spend/attempt admission, SSIAG credential retrieval,
entitlements and other source families remain open. No qxctl surface is added.

## Non-live runtime closure progress

SQDV 0.3.0-dev adds durable processing checkpoints and confirmed-prefix replay
admission. The Databento 0.5.0-dev library adds SQPV-backed attempt/charge-ceiling
state and bounded native historical HTTP. The non-live pipeline verifies capture,
coverage, retention, budget recovery and checkpoint resume together. The actual
SSIAG deployment/bridge, verified quote source and selected receiving contracts
remain operational prerequisites. Live data remains explicitly deferred.


## Broader non-live source extent

The independently selectable C++26 0.1.0-dev modules `native-source-support-cpp`, `sqav-fred-cpp`, `sqav-news-api-cpp` and `sqav-databento-reference-cpp` add bounded HTTPS/JSON support, FRED/ALFRED observation/vintage pages, vendor-neutral original/corrected news article ingress, and Databento reference/cost response handling. Each module SPEC owns exact operations, bounds, response fidelity and non-claims. Provider authentication, credentials and entitlements are external prerequisites. No source activation is established by fixture evidence.

`knowledge/sqv/scabv/SPEC.md` admits separate IBKR Client Portal and TWS read bindings; their captures consume the existing research-data path. TWS targets external Stable SDK 10.45 without bundled vendor code or actual SDK conformance. Live remains future work. The shared HTTP CredentialUse interface is a required composition boundary, not an operational SSIAG bridge. Databento remains mapped to SSIAG/local Keychain in `knowledge/sqv/sqav/DATABENTO-SSIAG-BINDING.md`.

Focused source tests preserve all five adapter capture families through reopened local retention/replay. Provider schemas and source scope remain in original JSON/JSONL or explicitly declared TWS callback projections; exact financial bytes and price bits are not silently reserialized from binary floating-point values. API availability does not imply completion of all SQV programme objectives.

## Source composition verification

- `tests/sqv-nonlive-sources/CMakeLists.txt`
- `tests/sqv-nonlive-sources/retention.hpp`
