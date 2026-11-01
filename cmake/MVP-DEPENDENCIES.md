# MVP dependency and identity projection

The [machine-readable projection](mvp-dependencies.json) maps all 69 current component roots: 58 modules, four libraries and seven tools. It records 57 native build roots (56 installable packages and one authoring build), eight independent Go modules, one Swift provider, and the remaining proposal/shared-source roots. The baseline is main commit `4499b25170fc958d9f15f2c8d9446878060d3b78`, before this dependency alignment. `source_inputs` binds the actual projected files by SHA-256; the baseline is not the final change commit.

This map does not authorize a release or activate a provider. L01 owns the approved release set. The required platforms are Apple Silicon macOS, Ubuntu LTS on x86_64 and ARM64, and physical DGX Spark ARM64 running a supported DGX OS. Ubuntu ARM64 CI does not qualify Spark hardware, its image, kernel, drivers or GPU behavior. Native Windows is unsupported and no port is planned. The native package matrix contains 54 packages for required-platform qualification and two current DuckDB packages reserved for legacy Intel macOS qualification; building either set must be evidenced separately.

Identity fields have different owners. A CMake project version selects an SDK configuration; a receipt package release selects exact installed artifacts; an owner-interface version selects admitted behavior; a Go tag identifies an immutable public module; and a schema ID or wire protocol identifies a data contract. Matching the digits in two fields does not make them interchangeable. No version is changed merely to make this map uniform.

Every component root appears once below. Native rows show `CMake project / receipt release`; Go rows show the compiled source default and separately verified public identity. SDK pins are exact CMake package selections; a `(test)` suffix denotes a test-only edge. The JSON retains imported target names, exported package versions, installed-only options and topological build order.

| Component root | Exact identity | First-party dependencies | Qualification/disposition |
| --- | --- | --- | --- |
| [modules/accordare-stav-producer](../modules/accordare-stav-producer) | source `0.1.0-dev`; public `v0.1.0-dev` | `libraries/stav-protocol-go@v0.2.0`; `modules/stav-append-authority@v0.2.1` | independent Go module |
| [modules/hotpath-runtime](../modules/hotpath-runtime) | No independent runtime package/version | Proposal; no implementation or installer | proposal only no implementation or installer |
| [modules/knowledge-session-coordinator](../modules/knowledge-session-coordinator) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/maestro](../modules/maestro) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/native-source-support-cpp](../modules/native-source-support-cpp) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | quantitative |
| [modules/sacv-engine](../modules/sacv-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/sav-engine](../modules/sav-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/sbv-engine](../modules/sbv-engine) | `0.33.0` / `0.33.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqavDatabentoDbn 0.5.0 EXACT`; `SymphonySqdvDelivery 0.3.0 EXACT` | quantitative |
| [modules/scabv-ibkr-client-portal-cpp](../modules/scabv-ibkr-client-portal-cpp) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonyNativeSourceSupport 0.1.0 EXACT`; `SymphonySqavCapture 0.2.0 EXACT` | quantitative |
| [modules/scabv-ibkr-tws-cpp](../modules/scabv-ibkr-tws-cpp) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonyNativeSourceSupport 0.1.0 EXACT`; `SymphonySqavCapture 0.2.0 EXACT` | quantitative |
| [modules/scev-cf-engine](../modules/scev-cf-engine) | `0.10.0` / `0.10.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | cloud |
| [modules/scev-engine](../modules/scev-engine) | `0.10.0` / `0.10.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | cloud |
| [modules/schv-aws-engine](../modules/schv-aws-engine) | `0.10.0` / `0.10.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | cloud |
| [modules/schv-azure-engine](../modules/schv-azure-engine) | `0.10.0` / `0.10.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | cloud |
| [modules/schv-do-engine](../modules/schv-do-engine) | `0.10.0` / `0.10.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | cloud |
| [modules/schv-engine](../modules/schv-engine) | `0.10.0` / `0.10.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | cloud |
| [modules/schv-gcp-engine](../modules/schv-gcp-engine) | `0.10.0` / `0.10.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | cloud |
| [modules/sciv-engine](../modules/sciv-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySnvCommon 0.1.0 EXACT` | knowledge |
| [modules/sclv-engine](../modules/sclv-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/scnv-engine](../modules/scnv-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySnvCommon 0.1.0 EXACT` | knowledge |
| [modules/scv-engine](../modules/scv-engine) | `0.10.0` / `0.10.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | cloud |
| [modules/scv-graph-duckdb-connector](../modules/scv-graph-duckdb-connector) | `0.2.0` / `0.2.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | legacy_duckdb |
| [modules/secure-identity-access-governance](../modules/secure-identity-access-governance) | source `dev`; public none in the verified dependency-pin set | `libraries/stav-protocol-go@v0.2.0`; `modules/stav-append-authority@v0.2.1` | independent Go module |
| [modules/sev-engine](../modules/sev-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/shv-engine](../modules/shv-engine) | `0.4.0` / `0.4.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/shv-graph-adapter](../modules/shv-graph-adapter) | `0.2.0` / `0.2.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/shv-graph-duckdb-connector](../modules/shv-graph-duckdb-connector) | `0.4.0` / `0.4.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | legacy_duckdb |
| [modules/shv-partition-engine](../modules/shv-partition-engine) | `0.4.0` / `0.4.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/shv-pdf-adapter](../modules/shv-pdf-adapter) | `0.3.0` / `0.3.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/shv-profile-engine](../modules/shv-profile-engine) | `0.3.0` / `0.3.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/shv-publication-engine](../modules/shv-publication-engine) | `0.4.0` / `0.4.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/shv-source-engine](../modules/shv-source-engine) | `0.2.0` / `0.2.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/skvi-engine](../modules/skvi-engine) | `0.2.0` / `0.2.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/sniv-engine](../modules/sniv-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySnvCommon 0.1.0 EXACT` | knowledge |
| [modules/snrv-engine](../modules/snrv-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySnvCommon 0.1.0 EXACT` | knowledge |
| [modules/snv-engine](../modules/snv-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySnvCommon 0.1.0 EXACT`; `SymphonySniv 0.1.0 EXACT`; `SymphonySnrv 0.1.0 EXACT`; `SymphonySciv 0.1.0 EXACT`; `SymphonyScnv 0.1.0 EXACT` | knowledge |
| [modules/snv-local-observer](../modules/snv-local-observer) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/sodv-engine](../modules/sodv-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/sqav-attempt-engine](../modules/sqav-attempt-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT` | quantitative |
| [modules/sqav-capture-cpp](../modules/sqav-capture-cpp) | `0.2.0` / `0.2.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT`; `SymphonySqdvDelivery 0.3.0 EXACT` (test) | quantitative |
| [modules/sqav-databento-dbn-cpp](../modules/sqav-databento-dbn-cpp) | `0.5.0` / `0.5.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT`; `SymphonySqavCapture 0.2.0 EXACT`; `SymphonySqpvLocalStore 0.2.0 EXACT`; `SymphonySqdvDelivery 0.3.0 EXACT` (test) | quantitative |
| [modules/sqav-databento-reference-cpp](../modules/sqav-databento-reference-cpp) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonyNativeSourceSupport 0.1.0 EXACT`; `SymphonySqavCapture 0.2.0 EXACT`; `SymphonySqavDatabentoDbn 0.5.0 EXACT` | quantitative |
| [modules/sqav-fred-cpp](../modules/sqav-fred-cpp) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonyNativeSourceSupport 0.1.0 EXACT`; `SymphonySqavCapture 0.2.0 EXACT` | quantitative |
| [modules/sqav-news-api-cpp](../modules/sqav-news-api-cpp) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonyNativeSourceSupport 0.1.0 EXACT`; `SymphonySqavCapture 0.2.0 EXACT` | quantitative |
| [modules/sqav-request-engine](../modules/sqav-request-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqavFred 0.1.0 EXACT`; `SymphonySqavDatabentoReference 0.1.0 EXACT` | quantitative |
| [modules/sqdv-checkpoint-engine](../modules/sqdv-checkpoint-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT` | quantitative |
| [modules/sqdv-delivery-cpp](../modules/sqdv-delivery-cpp) | `0.3.0` / `0.3.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT`; `SymphonySqpvLocalStore 0.2.0 EXACT` | quantitative |
| [modules/sqfv-batch-cpp](../modules/sqfv-batch-cpp) | `0.3.0` / `0.3.0-dev` | None | quantitative |
| [modules/sqfv-flow-engine](../modules/sqfv-flow-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT` | quantitative |
| [modules/sqmv-metadata-cpp](../modules/sqmv-metadata-cpp) | `0.2.0` / `0.2.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT` | quantitative |
| [modules/sqmv-metadata-engine](../modules/sqmv-metadata-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT` | quantitative |
| [modules/sqpv-inspection-engine](../modules/sqpv-inspection-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT` | quantitative |
| [modules/sqpv-local-store-cpp](../modules/sqpv-local-store-cpp) | `0.2.0` / `0.2.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT` | quantitative |
| [modules/sqtv-conversion-engine](../modules/sqtv-conversion-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT`; `SymphonySqtvIntegerConversion 0.2.0 EXACT` | quantitative |
| [modules/sqtv-integer-conversion-cpp](../modules/sqtv-integer-conversion-cpp) | `0.2.0` / `0.2.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT`; `SymphonySqfvBatch 0.3.0 EXACT`; `SymphonySqmvMetadata 0.2.0 EXACT`; `SymphonySqdvDelivery 0.3.0 EXACT` (test) | quantitative |
| [modules/ssfv-engine](../modules/ssfv-engine) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [modules/ssiag-provider-macos-keychain](../modules/ssiag-provider-macos-keychain) | `0.1.0-draft`; Swift tools `6.0`; macOS `13.0+` | Apple Security.framework and bsm; no external Swift packages | macOS provider; L14 operational qualification remains open |
| [modules/stav-append-authority](../modules/stav-append-authority) | source `dev`; public `v0.2.1` | `libraries/stav-protocol-go@v0.2.0` | independent Go module |
| [libraries/knowledge-vector-engine-cpp](../libraries/knowledge-vector-engine-cpp) | `0.2.0` / `0.2.0-dev` | None | bootstrap |
| [libraries/sbv-go](../libraries/sbv-go) | source `0.1.7-dev`; public none in the verified dependency-pin set | No first-party Go requirements; native `modules/sbv-engine 0.33.0-dev` | independent Go module |
| [libraries/snv-common-cpp](../libraries/snv-common-cpp) | `0.1.0` / `0.1.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |
| [libraries/stav-protocol-go](../libraries/stav-protocol-go) | source `no compiled release string`; public `v0.2.0` | No first-party Go requirements | independent Go module |
| [tools/authoring-cpp](../tools/authoring-cpp) | `unversioned project` / `no installable receipt` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | build_support |
| [tools/qxctl](../tools/qxctl) | source `qxctl dev`; public none in the verified dependency-pin set | `libraries/stav-protocol-go@v0.2.0`; `modules/accordare-stav-producer@v0.1.0-dev`; `modules/stav-append-authority@v0.2.1`; `tools/sov-ssh-adapter@v0.1.0-dev` | independent Go module |
| [tools/shv-interface-codegen](../tools/shv-interface-codegen) | No independent runtime package/version | Built by `tools/authoring-cpp` | authoring source built by tools authoring cpp not runtime package |
| [tools/sov-ssh-adapter](../tools/sov-ssh-adapter) | source `0.1.0-dev`; public `v0.1.0-dev` | No first-party Go requirements | independent Go module |
| [tools/sov-stdio-adapter](../tools/sov-stdio-adapter) | source `0.1.0-dev`; public none in the verified dependency-pin set | No first-party Go requirements | independent Go module |
| [tools/sqv-administration-cpp](../tools/sqv-administration-cpp) | No independent runtime package/version | Shared source consumed by SQV owners | shared source support not independent package |
| [tools/symphony-validator](../tools/symphony-validator) | `unversioned project` / `0.2.0-dev` | `SymphonyKnowledgeVectorEngine 0.2.0 EXACT` | knowledge |

The native groups separate build cost and dependency order: `bootstrap` contains the foundation; `knowledge` contains knowledge owners, SNV SDK/owners and the SHV graph adapter; `cloud` contains the eight SCV domains; `quantitative` contains the remaining source/data SDKs, SQV owners and SBV. `legacy_duckdb` is deliberately excluded from required-platform qualification until L09 is resolved. `build_support` is non-installable authoring tooling. Groups are scheduling boundaries, not different protocol authorities.

Each installed-only native build must resolve imported first-party targets from a fresh prefix with its recorded `installed_options` enabled. Missing or wrong exact versions must fail configuration instead of silently adding source dependencies. A separately copied SDK consumer must configure and link without the producer source tree. The topological build order follows runtime dependencies; test-only edges can point to later packages, so tests are enabled after their required SDKs are installed. Receipts, installed file hashes, imported library locations and execution checks provide different evidence; success of a source-tree build alone is insufficient.

All Go modules declare Go `1.26.5`. The projection stores their `go.mod` requirements and complete resolved module graphs, including available sums and remote origin hashes; it strips local cache paths. It explicitly maps six production commands: `qxctl`, `symphony-ssiag`, `symphony-stav-append-authority`, `symphony-accordare-stav-producer`, `sov-stdio-adapter` and `sov-ssh-adapter`, with exact main packages, entrypoints and separate source identities. `libraries/stav-protocol-go` is a library; the SBV `examples/roundtrip` command is an example excluded from the installed release executable set. Resolution is metadata evidence, not proof that every archive or package was downloaded, compiled or executed. Independent builds use `GOWORK=off`, `GOTOOLCHAIN=local`, no module replacements and the exact public module pins. The [workspace](../go.work) remains useful for development but cannot substitute for this gate.

| Verified public first-party dependency | Public tag | Immutable repository commit |
| --- | --- | --- |
| `libraries/stav-protocol-go` | [v0.2.0](https://github.com/QuanuX/Symphony/tree/libraries/stav-protocol-go/v0.2.0) | [`55f8faf26f4f85213ac23cc1de7ba897b2129a4c`](https://github.com/QuanuX/Symphony/commit/55f8faf26f4f85213ac23cc1de7ba897b2129a4c) |
| `modules/accordare-stav-producer` | [v0.1.0-dev](https://github.com/QuanuX/Symphony/tree/modules/accordare-stav-producer/v0.1.0-dev) | [`f36c3372ee5fb2ab9d1eb059eac30b543185db2d`](https://github.com/QuanuX/Symphony/commit/f36c3372ee5fb2ab9d1eb059eac30b543185db2d) |
| `modules/stav-append-authority` | [v0.2.1](https://github.com/QuanuX/Symphony/tree/modules/stav-append-authority/v0.2.1) | [`967c005e21a8f153ed453c79baa09fe2ab549f01`](https://github.com/QuanuX/Symphony/commit/967c005e21a8f153ed453c79baa09fe2ab549f01) |
| `tools/sov-ssh-adapter` | [v0.1.0-dev](https://github.com/QuanuX/Symphony/tree/tools/sov-ssh-adapter/v0.1.0-dev) | [`967c005e21a8f153ed453c79baa09fe2ab549f01`](https://github.com/QuanuX/Symphony/commit/967c005e21a8f153ed453c79baa09fe2ab549f01) |

The source defaults `dev` (STAV and SSIAG), `qxctl dev`, and the other `-dev`/`-draft` package labels remain visible. L06 owns reproducible release identities and artifact metadata. A public STAV Go module tag such as `v0.2.1` does not set a running daemon's compiled package version. No unverified public release is invented for modules outside the four verified first-party pins.

The 23 owner declarations below retain their historical exact admission lists. Generated C++/Go interface files are included in `source_inputs`, and the JSON carries generated Go admission versions separately. The SCV declaration is shared by eight domains. Current SCV packages are `0.10.0-dev`; older CLI defaults and supported historical declarations are intentional. A fresh current installation must be selected explicitly with `--version 0.10.0-dev`; `latest` is not a compatible substitute.

| Owner declaration | Current package identity | Declaration protocol |
| --- | --- | --- |
| [knowledge/scv/OWNER-INTERFACE.json](../knowledge/scv/OWNER-INTERFACE.json) | `0.10.0-dev` | `symphony.scv.owner-interface.v1` |
| [modules/sbv-engine/OWNER-INTERFACE.json](../modules/sbv-engine/OWNER-INTERFACE.json) | `0.33.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/sciv-engine/OWNER-INTERFACE.json](../modules/sciv-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/scnv-engine/OWNER-INTERFACE.json](../modules/scnv-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/shv-engine/OWNER-INTERFACE.json](../modules/shv-engine/OWNER-INTERFACE.json) | `0.4.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/shv-graph-adapter/OWNER-INTERFACE.json](../modules/shv-graph-adapter/OWNER-INTERFACE.json) | `0.2.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/shv-graph-duckdb-connector/OWNER-INTERFACE.json](../modules/shv-graph-duckdb-connector/OWNER-INTERFACE.json) | `0.4.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/shv-partition-engine/OWNER-INTERFACE.json](../modules/shv-partition-engine/OWNER-INTERFACE.json) | `0.4.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/shv-pdf-adapter/OWNER-INTERFACE.json](../modules/shv-pdf-adapter/OWNER-INTERFACE.json) | `0.3.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/shv-profile-engine/OWNER-INTERFACE.json](../modules/shv-profile-engine/OWNER-INTERFACE.json) | `0.3.0-dev` | `symphony.shv.profile-owner-interface.v1` |
| [modules/shv-publication-engine/OWNER-INTERFACE.json](../modules/shv-publication-engine/OWNER-INTERFACE.json) | `0.4.0-dev` | `symphony.shv.owner-interface.v1` |
| [modules/shv-source-engine/OWNER-INTERFACE.json](../modules/shv-source-engine/OWNER-INTERFACE.json) | `0.2.0-dev` | `symphony.shv.owner-interface.v1` |
| [modules/sniv-engine/OWNER-INTERFACE.json](../modules/sniv-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/snrv-engine/OWNER-INTERFACE.json](../modules/snrv-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/snv-engine/OWNER-INTERFACE.json](../modules/snv-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/snv-local-observer/OWNER-INTERFACE.json](../modules/snv-local-observer/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/sqav-attempt-engine/OWNER-INTERFACE.json](../modules/sqav-attempt-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/sqav-request-engine/OWNER-INTERFACE.json](../modules/sqav-request-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/sqdv-checkpoint-engine/OWNER-INTERFACE.json](../modules/sqdv-checkpoint-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/sqfv-flow-engine/OWNER-INTERFACE.json](../modules/sqfv-flow-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/sqmv-metadata-engine/OWNER-INTERFACE.json](../modules/sqmv-metadata-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/sqpv-inspection-engine/OWNER-INTERFACE.json](../modules/sqpv-inspection-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |
| [modules/sqtv-conversion-engine/OWNER-INTERFACE.json](../modules/sqtv-conversion-engine/OWNER-INTERFACE.json) | `0.1.0-dev` | `symphony.shv.owner-interface.v2` |

The schema/resource catalog records 458 source files, their IDs where declared, protocol strings, hashes and explicit test/example flags. A source catalog entry is not an assertion that a package installs or admits that resource. Test fixtures and examples are excluded from installed scope; malformed negative JSON fixtures are labeled separately. Historical schema generations and owner-admission entries remain distinct from the current package release. The SCV discovery catalog maps 112 protocol entries onto 29 installed schema filenames; protocol-entry counts therefore differ from file counts.

Two embedded SHV identities intentionally stay frozen: the SHV owner embeds PDF `0.2.0-dev` while the standalone PDF owner is `0.3.0-dev`; the SHV DuckDB connector embeds graph adapter `0.1.0-dev` while the standalone graph adapter is `0.2.0-dev`. The JSON records their source evidence. Dependency alignment does not silently replace these embedded behavioral contracts.

External dependencies and build tools retain their own selection rules:

| Dependency | Version or selection | Boundary |
| --- | --- | --- |
| `nlohmann-json` | `3.12.0` | vendored header, installed by foundation |
| `CURL` | `8.7.1` | find_package EXACT and CURL::libcurl; no dependency download by module builds |
| `zstd` | `1.5.7` | find_package EXACT CONFIG; zstd::libzstd_static |
| `DuckDB` | `1.5.5` | supplied exact hashed files, macos-amd64 only |
| `IBKR-TWS-SDK` | `10.45` | optional authorized external conformance fixture only; not a required library for normal scabv-ibkr-tws-cpp build |
| `Threads` | `platform/caller selected` | platform Threads::Threads from compiler/OS, no project semver |
| `dynamic-loader` | `platform/caller selected` | CMAKE_DL_LIBS from target platform, no project semver |
| `PDFium` | `caller-selected; example 153.0.7999.0` | Optional extraction-time caller-selected shared library path and SHA256; code checks required FPDF C symbols, not a universal library semver pin. macOS is the documented qualification; decoder transitive libraries are outside its file digest. |
| `Git` | `platform/caller selected` | Local SCLV evidence adapter invokes fixed /usr/bin/git under sanitized environment; no semver pin. |
| `SBV-provider-feedback-plugins` | `platform/caller selected` | Optional trusted caller-selected native shared libraries; versioned C ABI and exact path/digest selection are distinct from SBV module release or a vendor SDK dependency. |
| CMake / C++ | CMake `3.30+`; C++26 as declared per root | Compiler, standard library, architecture and build flags require concrete profile evidence. |
| Go | `1.26.5` | Exact local toolchain for independent module builds; automatic toolchain download is disabled in the gate. |
| Swift / Apple SDK | Swift tools `6.0`; macOS `13.0+` | macOS Keychain provider only; Security.framework and bsm come from the selected Apple SDK. |

CURL's exact version does not alone select its TLS backend, link mode or transitive libraries. DuckDB's current hashes describe the recorded Intel macOS artifact, not an ARM64/Linux promise. Optional PDFium and SBV provider plugins are selected by explicit path/digest and owner ABI checks. IBKR's optional SDK conformance fixture does not license or qualify live broker access. L09, L14–L17 and physical Spark qualification remain separate launch work; fixture-only success cannot complete promised provider capabilities.

The qualification build source archives are [curl 8.7.1](https://curl.se/download/curl-8.7.1.tar.xz), SHA-256 `6fea2aac6a4610fbd0400afb0bcddbe7258a64c63f1f68e5855ebc0c659710cd`, and [zstd 1.5.7](https://github.com/meta/zstd/releases/download/v1.5.7/zstd-1.5.7.tar.gz), SHA-256 `eb33e51f49a15e023950cd7825ca74a4a2b43db8354825ac24fc1b7ee09e6fa3`. These source pins are recorded for repeatable qualification builds; they do not imply completed runtime or redistribution qualification.

The macOS Keychain provider has a declared draft package and native framework dependencies, but its source explicitly keeps operational access, provider operations and the secret channel disabled. L14 must complete the promised credential/provider behavior and qualification. The hotpath-runtime proposal may remain in progress if L01 leaves it outside the release set.

Update the JSON projection when a bound source input changes. Review exact pins, exported dependencies, generated owner declarations, schema identities and frozen embedded identities together; then update their SHA-256 bindings. The projection and its verification harness are deliberately excluded from their own `source_inputs` to avoid self-reference. A hash match confirms consistency with recorded bytes; it is not release approval, a signature, or execution proof.
