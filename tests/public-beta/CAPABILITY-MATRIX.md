# Public beta capability and package selection proposal

Status: **prepared for manual review; no capability selection or publication approved**. This matrix binds source commit `887e074780e1d0633bd6e6f57c3d4e07ca3925fc`. It is the W01 capability dossier and the capability input to W06 package/trust planning. It does not implement the remaining plan, run acceptance, enable providers or change the feature freeze.

The [source inventory](source-inventory.json) was captured from the clean source and preserves all 73 component roots, 60 native CMake components (59 installable), nine Go modules and one Swift provider. The three roots without separate native/Go/Swift package entries are the unimplemented hotpath proposal and two authoring/shared-source roots. `tools/authoring-cpp` is itself one of the 60 native roots but is not an installable runtime. Inventory membership is not a decision to ship every component.

[capability-groups.json](capability-groups.json) assigns **every root exactly once** to 26 review groups. Each group binds independent versions, exact owner references, operations, implementation limits, proposed profiles, qualification state and applicable acceptance scenarios. All selections below are proposals. The inventory's `selected_components`, `promised_provider_operations`, `reviewer` and `reviewed_at` remain unfilled; its acceptance/publication flags remain false.

## Proposed selection for review

Prepare four-profile `PKG-CORE-<profile>` selections for the existing local administrative, knowledge and quantitative workflows: CLI/foundations, SSIAG/STAV foundations, coordination, governance/composition, SCV/SHV/SNV evidence, SQV data/administration, bounded SBV research and explicit SOV carriers. Every component retains its own package/receipt/version. Service installation does not enroll TOPS, start a service, grant permission, acquire credentials or activate a provider.

The proposed default closure is **51 complete native receipt units and six Go command modules**, plus the separately accounted Go library source dependencies. It follows the `proposed_core*` groups and exact native dependency DAG, with the protected profile administrator separately selectable. Four otherwise optional roots are required dependency bytes: `native-source-support-cpp`, Databento DBN/reference and FRED. Their inclusion does not select or qualify live-provider operations. Eight native units remain separately selectable: IBKR Client Portal/TWS, both DuckDB connectors, SHV PDF, Linux observer, news ingress and SRRAPIV. The JSON records the exact roots and derivation.

Offer explicit optional selections for PDF ingestion, DuckDB, Linux observation, provider-native SDK boundaries, Apple metadata readiness, protected profile administration and the supplemental Cloudflare runtime. These labels describe proposed payload organization. They **do not remove a retained promise or resolve decision D03**. If the beta promise still requires unimplemented general credentials, TWS linkage, Cloudflare rotation or project execution, that promise remains blocked pending an explicit scope/freeze decision.

An SDK selection is an alias for the complete native receipt units that export SDKs. Do not strip headers/libraries from their receipt or invent a headers-only package. Exact CMake export/SDK versions are recorded per root in the JSON; [PACKAGE-AND-TRUST.md](PACKAGE-AND-TRUST.md) owns payload recipes, notices and trust. `PKG-NATIVE-<component_id>` units remain independently installable. The package arrangement grants no universal architectural authority: users may compose their own engines, strategies, providers, buses and storage.

`PKG-CF-INTEL` is a proposed **supplemental** package/qualification profile. It must bind the actual Intel macOS host and exact C0 runtime bytes; it is not one of the four core profiles and not automatic approval of the broader optional legacy profile. Core-profile native adapter packages may expose separately qualified credentialless source/inspection/planning behavior, but they must preserve the refusal of unadmitted vendor execution. This distinction belongs in both package metadata and operation-level support claims.

Capability bundles are usage selectors. Each selected receipt must retain the exact transitive SDK/runtime closure in the dependency map, including dependencies assigned to optional capability groups. For example, retaining a provider library required by SBV does not advertise its unavailable credentialed operations. No runtime files, credentials, private provider responses, user datasets, provider entitlements or licensed SDK material are silently included. Build-support roots are retained as build/source material, and `hotpath-runtime` stays visibly proposal-only with no binary payload. Final artifact selection still requires the W01/W06 review.

## Profile and operation boundaries

| Profile | Proposed core status | Important exceptions |
| --- | --- | --- |
| `macos15-arm64` | Required final-download acceptance target | Swift metadata adapter may be selected; operational Keychain is unavailable. Cloudflare vendor execution is not admitted on this architecture. |
| `ubuntu24.04-amd64` | Required final-download acceptance target | Linux observer requires actual selected proc/sysfs evidence. No Apple provider. Cloudflare vendor execution is not admitted. |
| `ubuntu24.04-arm64` | Required final-download acceptance target | Generic Ubuntu ARM64 is accepted for this sprint; no physical DGX/GPU claim. No supplied SDK-linked R API+ Linux ARM64 lane. Cloudflare vendor execution is not admitted. |
| `ubuntu26.04-amd64` | Required final-download acceptance target | Exact kernel/compiler/runtime evidence required; no inheritance from Ubuntu 24.04. No Apple provider. Cloudflare vendor execution is not admitted. |
| `darwin-amd64` Cloudflare runtime | Supplemental proposed `PKG-CF-INTEL` only | Exact cf `1.0.0-beta.28`, Node `26.12.0`, cloudflared `2026.10.7` C0 artifacts. Actual host OS/profile, package closure and live acceptance remain open. |

The [platform catalog](../../cmake/mvp-platform-profiles.json) declares targets, not completed qualification. Native Windows is unsupported. SSH/WSL access is a separate Linux environment and does not establish WSL filesystem qualification. Physical DGX Spark, its DGX OS/drivers and GPU execution remain unqualified. Source compilation, installed SDK consumption, synthetic fixtures, vendor runtime admission and fresh-download/live acceptance are separate evidence classes.

SRRAPIV's source/control package is distinct from its SDK-linked variant. Supplied SDK lanes are `darwin-10`, `darwin-20.6-arm64`, `linux-gnu-3.10.0-x86_64` and `linux-gnu-4.18-x86_64`. Their presence is not qualification of an arbitrary host/compiler/ABI. The four-profile source selection must not become a four-profile live Rithmic promise. See [SRRAPIV installation](../../modules/srrapiv-runtime-cpp/INSTALL.md).

## Retained promises and the freeze

The [freeze](../../.github/RELEASE-GATES.md) permits stabilization of existing behavior, focused qualification and package preparation. It defers new features. The earlier I6 next-action instruction to implement private rotation remains an unresolved implementation target; it does not override that freeze. D03 requires an explicit decision for each conflict below, rather than a silent waiver or a new implementation campaign.

| Boundary | Existing source behavior | Unavailable or remaining work | Dossier treatment |
| --- | --- | --- | --- |
| General SSIAG L14 | Internal policy-bound dispatch, durable exact-owner admission, intent/outcome journal and audit/recovery primitives | Permission-backed lease lifecycle/audit wiring, native provider/recipient admission, protected one-shot delivery and actual native consumer bridges | Core foundations may be packaged; turnkey general credential operations are not supported. Retained operational promise stays blocked. |
| Cloudflare-specific credentials | Opt-in private-file provider, exact native peer/target binding, private secret/result delivery, SSIAG authority and committed STAV intent/outcome | Exact deployed services, profile administration and live qualification | Existing route is implemented and fixture-qualified; not an OS keychain or completion of general L14. |
| Apple/Linux credential providers | Apple metadata/code/signing/session readiness; portable SSIAG foundations | Apple data-protection Keychain operations and actual deployment policy/context; separate Linux Secret Service implementation | Apple metadata may be an optional package. Operational credential use is unavailable. |
| FRED/Databento | Native parsing, bounded requests and caller-supplied credential-use composition; offline DBN/replay | Actual operational SSIAG bridge plus authorized account/entitlements and provider success/refusal/recovery | Preserve useful native/offline libraries; do not advertise automatic credentialed acquisition. |
| IBKR Client Portal | Bounded reads through explicit Gateway/session admission | Externally authenticated actual Gateway, selected account and installed transport qualification | Qualification of existing behavior; no login automation or order entry. |
| IBKR TWS | Read-side request/callback host coordinator | Authorized Stable 10.45 SDK provenance and concrete socket/EReader/EWrapper/Decimal bridge | Existing source/SDK boundary may be retained; operational bridge is deferred implementation. |
| Cloudflare rotation | Existing selected CLI exposes PATCH primitive | Native private secret-body/result/recovery route missing; M1-02 incomplete | Unsupported, not an upstream impossibility. D03 remains open. |
| Cloudflare I5/I6 | Explicit local sessions, connector lifecycle and 23 resource profiles | Protected deployment, actual Access login, live Tunnel/DNS/Access, all nine remote operations, fault/refusal/recovery and observed cleanup | Existing-behavior qualification remains open; local gate cannot certify milestone/live completion. |
| Cloudflare project execution | Selected-file inspect and persisted preparation/migration proposals | Immutable preparation, selected build, deploy/status/reconcile/migrate execution | Proposal-only behavior is selectable; new execution features stay deferred. |
| SRRAPIV | Native mapping/control/runtime with separately selected SDK lane | Authorized SDK redistribution, exact linked target and actual provider acceptance | SDK-less control and SDK-linked variants stay separate. Provider/trading policy requires explicit action and does not supply risk authority. |
| Risk/state and execution composition | Existing SSIAG policy/audit, SBV research and SRRAPIV local admission are separate owners | No complete beta risk/state composition is selected or qualified by this dossier; independent order/position/risk reconciliation evidence remains required | Qualify explicitly selected existing composition. Any missing implementation stays frozen and D03 remains blocked. |
| `hotpath-runtime`, SIV/SMCV and general provisioning | Proposal/emerging domains or knowledge/planning surfaces | No implemented generic hotpath loader, SMCV translator, general provisioning/Terraform apply/Habitat/Nest/bus install | No binary/executable claim. These labels are not restrictions on user-owned compositions. |

The three PR196 P1 corrections are existing-behavior stabilization, not completion of these missing features. Its [verification record](../../knowledge/sov/cloudflare/VERIFICATION.md) describes the repaired collection permission, object identity and proven pre-launch connector refusal boundaries. Older unknown connector attempts without the new authentic refusal receipt remain unknown. No source/fixture pass upgrades provider readiness or retrospectively authorizes a retry.

Current sequencing explicitly assigns supported live-provider qualification to users **after packaging**. Unavailable live accounts do not block preparation of candidates. Missing implementation still blocks advertising that operation as supported, and neither sequencing nor optional labels close the retained promise. [Credential handoff](../../modules/secure-identity-access-governance/CREDENTIAL-QUALIFICATION.md); [release gates](../../.github/RELEASE-GATES.md).

## Capability groups

“Core” and “optional” below mean proposed selection for manual review, never approved scope. Each row links to complete root/version accounting below and to exact machine-readable operation declarations. Qualification remains pending for final bytes unless a separate artifact-bound record proves it.

| Group | Implemented capability proposed for selection | Explicit boundary | Proposed organization |
| --- | --- | --- | --- |
| `CAP-CORE` — CLI and native foundation | qxctl version/registry/schema/template discovery and explicit installed-owner invocation<br>Authority-free native JSON, framing, hashing, manifests and receipts<br>Separately installed read-only repository validator | CLI smoke does not establish semantic validation or final-download acceptance<br>No automatic activation, newest-version selection or mandatory resident trading-loop CLI | core; unapproved |
| `CAP-IDENTITY-AUDIT` — Identity, policy and audit | Receipt-v2 lifecycle, TOPS enrollment, kernel peers, deny-by-default permissions, policy overlay and provider bindings<br>Serialized STAV append authority and distinct committed intent/outcome receipts<br>Accordare audit producer for four ratified Named Version mutations<br>Internal general credential admission/dispatcher/journal primitives | General permission-backed credential lease lifecycle/audit wiring, native provider/recipient admission and protected delivery remain incomplete<br>Installation does not configure producer grants or prove credential use | core; unapproved |
| `CAP-COORDINATION` — Sessions, lifecycle journals and presence | Authenticated knowledge sessions, SSFV maintenance and SAV Named Version evidence<br>Locked worktree reconciliation, bounded lifecycle planning, protected report-only boot and separately authorized apply-capable journals<br>Maestro authenticated receptor presence inventory, CAS dock/undock and recovery | Maestro does not schedule or execute engines<br>Boot receptor reports plans and does not automatically apply them<br>Presence is not deployment success | core; unapproved |
| `CAP-GOVERNANCE` — Index, change, API, release and feature evidence | SKVI inspect/check/propose/project<br>SCLV change evidence and immutable proposals<br>SACV validation/compatibility/proposals<br>SODV release-ledger checks and recovery proposals<br>SSFV structural validation, snapshots, comparisons and administration assurance | Evidence does not ratify canonical changes or publish releases<br>SACV supports bounded OpenAPI 3.2.0 JSON/internal references, not YAML or external references | core; unapproved |
| `CAP-COMPOSITION` — Accordare and evolution | SAV Accord References, CURRENT resolution, rule evaluation, Named Versions, Extension Capsules and Installation Blueprint plans<br>SEV evidence-bound cases, impacts, dependency-safe ready sets, reobserved outcomes and recovery/closure proposals | No implicit canonical apply, live scheduling, automatic deployment or hot-path authority | core; unapproved |
| `CAP-CLOUD-KNOWLEDGE` — Cloud/provider source knowledge | Eight independent source-knowledge engines: SCV, hyperscaler/edge families, AWS, Azure, DigitalOcean, GCP and Cloudflare<br>Explicit source capture/interpretation, corpus, versions/constraints, graph projections, comparisons and selected retained transactions | Knowledge is not provisioning, Terraform apply, credentials or provider mutation<br>Selected evidence coverage is not a complete live inventory or entitlement claim | core; unapproved |
| `CAP-HARDWARE` — Hardware source, catalogue, composition and graph | Source provenance/revisions, caller-defined vocabularies, catalogue facts and requirement evaluation<br>Exact partition/source/kernel composition and separately authorized selected-head publication<br>Backend-independent graph exchange and portable reference adapter | No automatic hardware selection/provisioning or physical attestation<br>Historical dependency combinations retain exact admissions, not automatic version substitution | core; unapproved |
| `CAP-HARDWARE-PDF` — Selected SHV PDF ingestion | Provenance-bearing selected AMD table extraction and native graph project/validate/roundtrip | No general PDF/table parser, arbitrary conversion or SMCV translator<br>PDFium payload/runtime/notices require exact review | optional; unapproved |
| `CAP-DUCKDB` — Optional retained graph indexes | Caller-selected SCV/SHV DuckDB database and namespace storage, indexing, bounded retrieval/export, inventory and owner-specific publication/transfer recovery | No mandatory universal database<br>No live-database backup, cross-version migration or power-loss guarantee beyond established evidence | optional; unapproved |
| `CAP-NODES` — Supplied Node evidence | SNIV identity/participation, SNRV resources, SCIV cluster/bus relationships and SCNV scoped attributed names<br>SNV exact replay/composition and protected retained-view selection<br>Semantic-neutral shared mechanics | Records/digests are not physical-machine proof or operating authority<br>No identity issuance, bus connection or deployment certification | core; unapproved |
| `CAP-NODE-OBSERVER` — Linux exposure observer | Explicit read-only proc/sysfs CPU/memory exposure with attribution and restricted-exposure reporting | macOS reports unavailable<br>Synthetic or macOS checks do not prove Linux conformance<br>No bare-metal inventory, identity allocation or elevation | optional linux; unapproved |
| `CAP-SQV-DATA` — Native quantitative data libraries | Original-byte capture/provenance and immutable metadata<br>Trusted same-process immutable batches, flow/credit/lifetime accounting and recipient evidence<br>Local retention/replay, exact integer conversion and disposable or retention-before-delivery profiles | No universal bus, network transport, resident catalogue or automatic provider selection<br>Exact integer conversion is a bounded domain, not arbitrary transformation<br>Transfer is distinct from processing/retention/release evidence | core; unapproved |
| `CAP-SQV-ADMIN` — Quantitative administration owners | Owner-specific request validation, acquisition-attempt observation, metadata/flow administration, conversion validation, local-store inspection and delivery checkpoint observation | Shared transport is not a generic SQV semantic dispatcher or operational credential authority | core; unapproved |
| `CAP-SBV` — Native research and optional Go transport | Current owner-declared census, historical evaluation, scientific result access, selected research/economic composition and partitioned-result operations<br>Resident dataset lifecycle and durable local jobs with explicit cancellation/reconciliation<br>Installed C/C++ SDK and optional Go 0.1.7-dev transport pinned to native 0.33.0-dev | Capabilities retain exact operation/profile domains<br>Transport success is not experiment success<br>No arbitrary executable strategy loader, universal strategy support, calibrated broker fills, remote scheduling, CUDA/tensor runtime or live trading implied<br>Users may compose independent engines and supply admitted census/model data | core; unapproved |
| `CAP-SOURCE-TRANSPORT` — Native source I/O support | Strict bounded provider JSON and explicit one-shot HTTPS primitives for selected native consumers | No credential authority, provider semantics, automatic retry, durable spend authority or complete allocator-overhead guarantee | optional sdk; unapproved |
| `CAP-DATABENTO` — Databento file/replay and acquisition composition | Bounded supported DBN decoding/replay preserving original bytes and MBO structure<br>Reference/cost quotes and quote/request/attempt-bound one-attempt historical acquisition with explicitly supplied credential-use integration | General operational SSIAG bridge remains incomplete<br>No live feed, account-wide billing reconciliation, automatic paid retry or comprehensive normalization | optional sdk; unapproved |
| `CAP-FRED` — FRED/ALFRED composition | Exact observation/vintage requests, original decimal/missing values, bounded pages, explicit retry and retained-page recovery | Key provisioning and operational SSIAG bridge unavailable<br>No arbitrary output types, frequency aggregation, series-metadata API or cross-page atomic snapshot | optional sdk; unapproved |
| `CAP-NEWS` — Vendor-neutral news ingress | Immutable original/corrected article revisions, predecessor evidence and distinct time roles from caller-supplied content | No selected publisher/feed, HTTP server, wire connector, URL fetching/scraping, licence grant or full-feed completeness | optional sdk; unapproved |
| `CAP-IBKR-CP` — IBKR Client Portal reads | Explicit Gateway transport, authenticated status/account admission and bounded account/position/historical-bar reads with expiry/refusal | Gateway must already be externally authenticated<br>No login automation, account mutation, order entry, TLS bypass or atomic account snapshot<br>No operational SSIAG authorization supplied | optional sdk; unapproved |
| `CAP-IBKR-TWS` — IBKR TWS host boundary | Read-only request/callback coordinator, generation/request binding, pacing, cancellation and disconnect handling | Authorized Stable 10.45 SDK provenance and concrete socket/EReader/EWrapper/Decimal bridge remain missing<br>No vendor conformance/live collection, login automation, order entry or operational SSIAG authorization | source sdk boundary only; unapproved |
| `CAP-SRRAPIV` — Sovereign R API+ control/runtime | Eleven native operations for catalogue/bindings/coverage/API and runtime start/status/invoke/events/stop/attempts<br>Version-specific native mapping, generation/attempt binding and conservative lost-reply behavior<br>Explicit SDK-linked variant with separate provider/trading admission | No bundled R API+ headers/libraries, credentials or account data<br>SDK-less invoke reports unavailable<br>SDK return is not provider completion<br>Local policy is not a risk engine or account-parameter filter<br>No live/trading qualification inferred<br>No supplied SDK-linked Linux ARM64 lane; Ubuntu ARM64 control-source builds do not qualify R API+ vendor invocation. | optional control sdk variant pending; unapproved |
| `CAP-SOV` — Explicit local/SSH carriers and profiles | Explicit local stdio/scoped SSH carriers for nine existing bounded SBV job/result/bundle operations<br>Exact endpoint, host key, agent and launcher selection with conservative uncertainty<br>Separately administrator-gated protected profile publication/recovery, ordinary-user execution | No generic remote shell, automatic host provisioning, credential management, target scheduling or uncertain retry<br>SSH v1 remains independent and protected Access profiles remain optional | core carriers optional profile admin; unapproved |
| `CAP-CLOUDFLARE` — Pinned Cloudflare administration/private credentials | Exact offline installation lifecycle/context and credentialless resource plans<br>23 closed Tunnel/DNS/Access profiles with native private credential/result channels and committed SSIAG/STAV authority<br>Authenticated non-root connector lifecycle<br>Explicit browser Access sessions and protected SSH transport<br>Project selected-file inspect and proposal-only plan | Vendor execution admitted only for C0 darwin-amd64 artifacts: cf 1.0.0-beta.28, Node 26.12.0, cloudflared 2026.10.7<br>Rotation and project prepare/build/deploy/status/reconcile/migrate execution unavailable<br>Private-file store is not general L14, OS keychain, encryption or hostile same-user isolation<br>Connector liveness is not connectivity<br>Local revoke/clear does not revoke provider credentials or kill admitted streams | core source surfaces optional cf intel; unapproved |
| `CAP-APPLE-METADATA` — Apple metadata/readiness adapter | Swift metadata handshake, native peer/code checks, receipt-owned executable/app-bundle lifecycle and separate signed-bundle/policy/login-session readiness | Operational data-protection Keychain item/access-control operations, deployment identity and protected delivery incomplete<br>No Linux provider or secret access inferred from metadata readiness | optional metadata; unapproved |
| `CAP-AUTHORING` — Build-time authoring/shared support | Build-time code generation and shared bounded SQV transport/package mechanics for owning runtime builds | Three roots are not three independently installable runtime packages<br>Authoring does not grant semantic authority or create a universal SQV dispatcher | build source only; unapproved |
| `CAP-HOTPATH` — hotpath-runtime proposal | No implemented runtime. | Proposal-only seed with no implementation, installer or executable<br>No requirement for user strategies/hot paths to adopt it | no binary proposal visible; unapproved |

## Exact independent versions and all-root accounting

The following is the complete 73-root partition. Native cells show **package release / CMake project version**. Null project versions remain null; they are not guessed from a package name. Native exported CMake package versions/targets are retained separately in `independent_versions[].exported_packages` in [the JSON](capability-groups.json). Go cells distinguish the source runtime string from previously verified public dependency-module versions; public source tags do not qualify a new executable. Swift retains its own draft identity. No monolithic “Symphony version” is substituted for these identities.

The inventory's dependency-map baseline field is historical projection provenance, not this dossier's source commit. This dossier is bound to the explicit current commit/tree and map digest in [source-inventory.json](source-inventory.json). Existing public Go module identities are source releases, not evidence of a platform binary launch.

| Component root | Group | Exact version / disposition |
| --- | --- | --- |
| [tools/qxctl](../../tools/qxctl/) | `CAP-CORE` | runtime `0.1.0-dev`; Go `1.26.5`; verified public source none in this inventory |
| [libraries/knowledge-vector-engine-cpp](../../libraries/knowledge-vector-engine-cpp/) | `CAP-CORE` | `0.2.0-dev` / `0.2.0`; complete installable receipt |
| [tools/symphony-validator](../../tools/symphony-validator/) | `CAP-CORE` | `0.2.0-dev` / `null`; complete installable receipt |
| [modules/secure-identity-access-governance](../../modules/secure-identity-access-governance/) | `CAP-IDENTITY-AUDIT` | runtime `dev`; Go `1.26.5`; verified public source none in this inventory |
| [modules/stav-append-authority](../../modules/stav-append-authority/) | `CAP-IDENTITY-AUDIT` | runtime `dev`; Go `1.26.5`; verified public source `v0.2.1` |
| [modules/accordare-stav-producer](../../modules/accordare-stav-producer/) | `CAP-IDENTITY-AUDIT` | runtime `0.1.0-dev`; Go `1.26.5`; verified public source `v0.1.0-dev` |
| [libraries/stav-protocol-go](../../libraries/stav-protocol-go/) | `CAP-IDENTITY-AUDIT` | runtime `null`; Go `1.26.5`; verified public source `v0.2.0` |
| [modules/knowledge-session-coordinator](../../modules/knowledge-session-coordinator/) | `CAP-COORDINATION` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/maestro](../../modules/maestro/) | `CAP-COORDINATION` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/skvi-engine](../../modules/skvi-engine/) | `CAP-GOVERNANCE` | `0.2.0-dev` / `0.2.0`; complete installable receipt |
| [modules/sclv-engine](../../modules/sclv-engine/) | `CAP-GOVERNANCE` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sacv-engine](../../modules/sacv-engine/) | `CAP-GOVERNANCE` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sodv-engine](../../modules/sodv-engine/) | `CAP-GOVERNANCE` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/ssfv-engine](../../modules/ssfv-engine/) | `CAP-GOVERNANCE` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sav-engine](../../modules/sav-engine/) | `CAP-COMPOSITION` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sev-engine](../../modules/sev-engine/) | `CAP-COMPOSITION` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/scv-engine](../../modules/scv-engine/) | `CAP-CLOUD-KNOWLEDGE` | `0.10.0-dev` / `0.10.0`; complete installable receipt |
| [modules/schv-engine](../../modules/schv-engine/) | `CAP-CLOUD-KNOWLEDGE` | `0.10.0-dev` / `0.10.0`; complete installable receipt |
| [modules/scev-engine](../../modules/scev-engine/) | `CAP-CLOUD-KNOWLEDGE` | `0.10.0-dev` / `0.10.0`; complete installable receipt |
| [modules/schv-aws-engine](../../modules/schv-aws-engine/) | `CAP-CLOUD-KNOWLEDGE` | `0.10.0-dev` / `0.10.0`; complete installable receipt |
| [modules/schv-azure-engine](../../modules/schv-azure-engine/) | `CAP-CLOUD-KNOWLEDGE` | `0.10.0-dev` / `0.10.0`; complete installable receipt |
| [modules/schv-do-engine](../../modules/schv-do-engine/) | `CAP-CLOUD-KNOWLEDGE` | `0.10.0-dev` / `0.10.0`; complete installable receipt |
| [modules/schv-gcp-engine](../../modules/schv-gcp-engine/) | `CAP-CLOUD-KNOWLEDGE` | `0.10.0-dev` / `0.10.0`; complete installable receipt |
| [modules/scev-cf-engine](../../modules/scev-cf-engine/) | `CAP-CLOUD-KNOWLEDGE` | `0.10.0-dev` / `0.10.0`; complete installable receipt |
| [modules/shv-engine](../../modules/shv-engine/) | `CAP-HARDWARE` | `0.4.0-dev` / `0.4.0`; complete installable receipt |
| [modules/shv-source-engine](../../modules/shv-source-engine/) | `CAP-HARDWARE` | `0.2.0-dev` / `0.2.0`; complete installable receipt |
| [modules/shv-profile-engine](../../modules/shv-profile-engine/) | `CAP-HARDWARE` | `0.3.0-dev` / `0.3.0`; complete installable receipt |
| [modules/shv-partition-engine](../../modules/shv-partition-engine/) | `CAP-HARDWARE` | `0.4.0-dev` / `0.4.0`; complete installable receipt |
| [modules/shv-publication-engine](../../modules/shv-publication-engine/) | `CAP-HARDWARE` | `0.4.0-dev` / `0.4.0`; complete installable receipt |
| [modules/shv-graph-adapter](../../modules/shv-graph-adapter/) | `CAP-HARDWARE` | `0.2.0-dev` / `0.2.0`; complete installable receipt |
| [modules/shv-pdf-adapter](../../modules/shv-pdf-adapter/) | `CAP-HARDWARE-PDF` | `0.3.0-dev` / `0.3.0`; complete installable receipt |
| [modules/scv-graph-duckdb-connector](../../modules/scv-graph-duckdb-connector/) | `CAP-DUCKDB` | `0.2.0-dev` / `0.2.0`; complete installable receipt |
| [modules/shv-graph-duckdb-connector](../../modules/shv-graph-duckdb-connector/) | `CAP-DUCKDB` | `0.4.0-dev` / `0.4.0`; complete installable receipt |
| [libraries/snv-common-cpp](../../libraries/snv-common-cpp/) | `CAP-NODES` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sniv-engine](../../modules/sniv-engine/) | `CAP-NODES` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/snrv-engine](../../modules/snrv-engine/) | `CAP-NODES` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sciv-engine](../../modules/sciv-engine/) | `CAP-NODES` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/scnv-engine](../../modules/scnv-engine/) | `CAP-NODES` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/snv-engine](../../modules/snv-engine/) | `CAP-NODES` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/snv-local-observer](../../modules/snv-local-observer/) | `CAP-NODE-OBSERVER` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqav-capture-cpp](../../modules/sqav-capture-cpp/) | `CAP-SQV-DATA` | `0.2.0-dev` / `0.2.0`; complete installable receipt |
| [modules/sqmv-metadata-cpp](../../modules/sqmv-metadata-cpp/) | `CAP-SQV-DATA` | `0.2.0-dev` / `0.2.0`; complete installable receipt |
| [modules/sqfv-batch-cpp](../../modules/sqfv-batch-cpp/) | `CAP-SQV-DATA` | `0.3.0-dev` / `0.3.0`; complete installable receipt |
| [modules/sqpv-local-store-cpp](../../modules/sqpv-local-store-cpp/) | `CAP-SQV-DATA` | `0.2.0-dev` / `0.2.0`; complete installable receipt |
| [modules/sqtv-integer-conversion-cpp](../../modules/sqtv-integer-conversion-cpp/) | `CAP-SQV-DATA` | `0.2.0-dev` / `0.2.0`; complete installable receipt |
| [modules/sqdv-delivery-cpp](../../modules/sqdv-delivery-cpp/) | `CAP-SQV-DATA` | `0.3.0-dev` / `0.3.0`; complete installable receipt |
| [modules/sqav-attempt-engine](../../modules/sqav-attempt-engine/) | `CAP-SQV-ADMIN` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqav-request-engine](../../modules/sqav-request-engine/) | `CAP-SQV-ADMIN` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqmv-metadata-engine](../../modules/sqmv-metadata-engine/) | `CAP-SQV-ADMIN` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqfv-flow-engine](../../modules/sqfv-flow-engine/) | `CAP-SQV-ADMIN` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqtv-conversion-engine](../../modules/sqtv-conversion-engine/) | `CAP-SQV-ADMIN` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqpv-inspection-engine](../../modules/sqpv-inspection-engine/) | `CAP-SQV-ADMIN` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqdv-checkpoint-engine](../../modules/sqdv-checkpoint-engine/) | `CAP-SQV-ADMIN` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sbv-engine](../../modules/sbv-engine/) | `CAP-SBV` | `0.33.0-dev` / `0.33.0`; complete installable receipt |
| [libraries/sbv-go](../../libraries/sbv-go/) | `CAP-SBV` | runtime `0.1.7-dev`; Go `1.26.5`; verified public source none in this inventory |
| [modules/native-source-support-cpp](../../modules/native-source-support-cpp/) | `CAP-SOURCE-TRANSPORT` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqav-databento-dbn-cpp](../../modules/sqav-databento-dbn-cpp/) | `CAP-DATABENTO` | `0.5.0-dev` / `0.5.0`; complete installable receipt |
| [modules/sqav-databento-reference-cpp](../../modules/sqav-databento-reference-cpp/) | `CAP-DATABENTO` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqav-fred-cpp](../../modules/sqav-fred-cpp/) | `CAP-FRED` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/sqav-news-api-cpp](../../modules/sqav-news-api-cpp/) | `CAP-NEWS` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/scabv-ibkr-client-portal-cpp](../../modules/scabv-ibkr-client-portal-cpp/) | `CAP-IBKR-CP` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/scabv-ibkr-tws-cpp](../../modules/scabv-ibkr-tws-cpp/) | `CAP-IBKR-TWS` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/srrapiv-runtime-cpp](../../modules/srrapiv-runtime-cpp/) | `CAP-SRRAPIV` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [tools/sov-stdio-adapter](../../tools/sov-stdio-adapter/) | `CAP-SOV` | runtime `0.1.0-dev`; Go `1.26.5`; verified public source none in this inventory |
| [tools/sov-ssh-adapter](../../tools/sov-ssh-adapter/) | `CAP-SOV` | runtime `0.1.0-dev`; Go `1.26.5`; verified public source `v0.1.0-dev` |
| [tools/sov-transport-profile-admin](../../tools/sov-transport-profile-admin/) | `CAP-SOV` | runtime `0.1.0-dev`; Go `1.26.5`; verified public source none in this inventory |
| [modules/sov-cloudflare-adapter](../../modules/sov-cloudflare-adapter/) | `CAP-CLOUDFLARE` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/ssiag-provider-private-store](../../modules/ssiag-provider-private-store/) | `CAP-CLOUDFLARE` | `0.1.0-dev` / `0.1.0`; complete installable receipt |
| [modules/ssiag-provider-macos-keychain](../../modules/ssiag-provider-macos-keychain/) | `CAP-APPLE-METADATA` | Swift provider `0.1.0-draft`; metadata only |
| [tools/authoring-cpp](../../tools/authoring-cpp/) | `CAP-AUTHORING` | `null` / `null`; build support, no runtime installer |
| [tools/shv-interface-codegen](../../tools/shv-interface-codegen/) | `CAP-AUTHORING` | authoring source built by tools authoring cpp not runtime package |
| [tools/sqv-administration-cpp](../../tools/sqv-administration-cpp/) | `CAP-AUTHORING` | shared source support not independent package |
| [modules/hotpath-runtime](../../modules/hotpath-runtime/) | `CAP-HOTPATH` | proposal only no implementation or installer |

## Operation and acceptance coverage

The group JSON pins 24 owner-interface declarations plus Cloudflare's owner/session declarations by exact SHA-256 and enumerates their operation names/availability where declared. Roots without an owner-interface JSON retain their current SPEC/INTENT/INSTALL references; that absence is not an invented command catalogue. qxctl's current registry and installed schema/template/capability responses remain the exact CLI authority. Native library APIs do not become shell commands merely because they appear in this inventory.

The nine [existing acceptance scenarios](../../tests/launch-acceptance/plan.json) remain required across the four core profiles: **36 core cells**. The [acceptance mapping](acceptance-mapping.json) owns paper-criterion traceability. Group scenario references are obligations within that map, not additional passing results or replacements for whole-package scenarios. Non-runtime authoring/proposal roots have no fabricated runtime cells; their source/provenance and explicit unavailability still require review. Cloudflare Intel and any vendor-linked variants add supplemental obligations and cannot replace missing core cells.

Every selected package needs exact download/digest/trust/tamper refusal; clean-host installation, discovery and loader closure; applicable bounded quantitative/knowledge workflows; protected administration denial and success; every promised provider operation/refusal; timeout/cancellation/uncertainty recovery; side-by-side upgrade/compatible rollback; and removal preserving unrelated and user data. Full tests are reserved for planned release gates; stabilization increments use meaningful focused checks.

The group-specific obligations are retained below so later W02–W12 work cannot mistake implementation inventory for acceptance:

**CAP-CORE**

- Record exact Go graph/pins and source stamp. Source-workspace success is not standalone public-module qualification.
- Installed SDK consumers must compile/link without source/build-tree prefixes.

**CAP-IDENTITY-AUDIT**

- Prove selected kernel peers, deny/success, audit intent/outcome, expiry/revocation, restart and lost-acknowledgment recovery.
- General credential gaps are implementation, not just absent test profiles; D03 must resolve retained operational promises.

**CAP-COORDINATION**

- Separate boot report, prepared apply attempt, authenticated observation, host effect and audit commitment.

**CAP-GOVERNANCE**

- Verify exact installed owner behavior and refusal/recovery limits for selected operations; preserve independent authority and evidence.

**CAP-COMPOSITION**

- Verify exact installed owner behavior and refusal/recovery limits for selected operations; preserve independent authority and evidence.

**CAP-CLOUD-KNOWLEDGE**

- Verify exact installed owner behavior and refusal/recovery limits for selected operations; preserve independent authority and evidence.

**CAP-HARDWARE**

- Run the retained-source hardware starter from final packages with exact historical owner composition and independently authorized publication.

**CAP-HARDWARE-PDF**

- Qualify actual selected document extraction and original provenance; close exact PDFium dependency/notices before redistribution.

**CAP-DUCKDB**

- Close exact DuckDB loader/notices and qualify installed namespace/read/write/recovery behavior.

**CAP-NODES**

- Verify exact installed owner behavior and refusal/recovery limits for selected operations; preserve independent authority and evidence.

**CAP-NODE-OBSERVER**

- Record actual Linux proc/sysfs exposure and host/container topology, not synthetic fixtures alone.

**CAP-SQV-DATA**

- Compile installed SDK consumers; prove metadata/payload lifetime, exact conversion rejection and local-store uncertain publication recovery.

**CAP-SQV-ADMIN**

- Verify exact installed owner behavior and refusal/recovery limits for selected operations; preserve independent authority and evidence.

**CAP-SBV**

- Exercise all selected scientific operation/profile variants and complete result access, not only process exit.
- Verify installed C/C++ and optional Go consumer, durable job cancellation/restart, partial/recovery-required outcomes and no automatic uncertain retry.

**CAP-SOURCE-TRANSPORT**

- Bind exact TLS/runtime dependency closure; transport success is distinct from provider content/entitlement success.

**CAP-DATABENTO**

- Offline file/replay qualification can proceed without accounts. Any promised acquisition needs actual bridge, entitlements, bounded spend/request authorization and success/refusal/recovery evidence.

**CAP-FRED**

- Require concrete credential-use composition before turnkey acquisition claims; then authorized installed success, partial-page refusal and recovery.

**CAP-NEWS**

- Use authorized content and verify installed retention/corrections and substituted-predecessor refusal.

**CAP-IBKR-CP**

- Qualify actual externally authenticated Gateway, exact accounts, expiry/disconnect/cancellation and refusal.

**CAP-IBKR-TWS**

- Concrete vendor bridge is deferred implementation; D03 stays blocked for any retained TWS operational promise.

**CAP-SRRAPIV**

- Separate SDK-less control from exact SDK-linked variant. Review selected R API+ 13.7.0.0 header/binary provenance and redistribution authority.
- No provider/trading calls in package smoke; explicit operational authorization and provider acceptance remain separate.
- Select an actual supplied SDK platform lane and exact host/compiler/ABI; no four-profile SDK-linked claim.

**CAP-SOV**

- Keep SSH v1 and protected profiles separate. Qualify actual OpenSSH/host-key/agent/launcher and all nine operations.
- Disposable-root publication and Linux installed OpenSSH cases do not pass when skipped/absent.

**CAP-CLOUDFLARE**

- Core4 native credentialless operation qualification does not admit vendor execution. PKG-CF-INTEL requires its own exact OS/image/runtime/loader/notices and fresh-host evidence.
- All23 profiles remain live-unqualified. I6 remains open for private rotation, protected SSIAG/STAV deployment, live Tunnel/DNS/Access lifecycle, nine remote operations, denial/expiry/audit/profile/connector/lost-reply recovery and observed cleanup.
- PR196 does not retroactively settle old unresolved attempts without authentic pre-launch refusal evidence.

**CAP-APPLE-METADATA**

- Select metadata compatibility artifact or separately approved signed bundle. Preserve operational_access=false until missing implementation and exact deployment qualification are completed.

**CAP-AUTHORING**

- Retain generated-source provenance, reproducible build inputs and source/notices; do not invent runtime acceptance cells for source-only roots.

**CAP-HOTPATH**

- Keep proposal visible. D03 must resolve retained promises requiring it; no binary or successful runtime evidence can be invented.

## Review decisions still required

- Approve the exact W01 component, operation and platform selection, including which provider operations remain promised for beta. Optional organization is not approval of omission.
- Resolve D03 conflicts with retained promises while preserving the freeze. Do not relabel missing implementation as merely missing credentials, or implement it implicitly as qualification work.
- Approve W06 complete package/receipt selections, exact build graph and compiler/SDK inputs, third-party payload/notices, independent signer policy and ownership. Final files, hashes, loader closure and signatures require actual candidate bytes.
- Select any supplemental Cloudflare Intel or SRRAPIV SDK-linked profile explicitly; core source support is insufficient evidence for these runtime combinations.
- Retain the original paper criteria and all 36 core cells. Leave acceptance, reviewer and publication approvals incomplete until the corresponding evidence and manual decisions exist.

The next implementation campaign begins only after the requested manual review and model handoff. This matrix itself performs no build, provider acquisition, credential use, package publication or source-feature change.
