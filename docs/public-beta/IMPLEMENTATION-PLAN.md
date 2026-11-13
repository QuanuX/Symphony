# Public beta implementation and qualification plan

Prepared 13 November 2026 for manual review. **This plan is not approved for execution.** The current action prepares W01/W06 and the complete W01–W12 plan. Continue implementation only after the user's explicit approval and requested model switch, as described in [ASTRA-HANDOFF.md](ASTRA-HANDOFF.md).

The reviewed product source is `887e074780e1d0633bd6e6f57c3d4e07ca3925fc`, tree `3c989eb589bc2e84eff012aa5428a7e93170d81d`. It includes the Cloudflare stabilization changes merged in PR #196. The feature baseline remains `51d4806a60bceca1f47513f90206155d9775d510`; correcting existing behavior did not reopen new features. The clean-source [inventory](source-inventory.json) is a dated input, not a self-reference to the later documentation commit or a candidate build.

Read this with the [dossier index](README.md), [capability matrix](CAPABILITY-MATRIX.md), [package and trust proposal](PACKAGE-AND-TRUST.md), [owners and decisions](OWNERS-AND-DECISIONS.md), [dossier state](dossier.json) and [acceptance mapping](acceptance-mapping.json). They supply selection proposals and unresolved decisions. None supplies a successful installation, live-provider observation, publisher signature or release acceptance.

## Governing decisions and evidence

1. **Keep the feature freeze.** Existing-behavior fixes, focused tests, documentation, packaging and qualification may proceed after approval. General credential completion, a new TWS bridge, a new risk/execution-state service, Cloudflare project deployment and Tunnel rotation are not automatically authorized by the old W03–W05 proposals. Each retained promise needing unavailable implementation requires D03: explicitly defer/change that claim, retain it as blocked, or separately authorize an exact freeze exception. A deferred claim remains visible in the mapping; a blocked claim is not a pass.
2. **Package before user-led live qualification.** The current [release gates](../../.github/RELEASE-GATES.md) and [acceptance procedure](../../tests/launch-acceptance/README.md) assign live-provider work to users after packaging. Missing live profiles or results do not prevent W01/W06, source stabilization, harness preparation or candidate construction. They do prevent acceptance of the applicable live-provider cells. Necessary implementation, permitted payload and trust decisions still constrain what the candidate can claim.
3. **Keep the complete acceptance inventory.** Preserve all 36 cells in the current plan: nine scenarios on each of `macos15-arm64`, `ubuntu24.04-amd64`, `ubuntu24.04-arm64` and `ubuntu26.04-amd64`. Preserve all 58 paper criteria, including UE1–UE6, in the companion mapping. Component applicability supplements these cells; it does not delete them. Cloudflare's admitted vendor runtime is presently `darwin-amd64`, requiring a separate exact OS/profile decision and supplemental campaign. It does not qualify the four core profiles. Ubuntu ARM64 does not imply physical DGX, GPU or native Windows qualification.
4. **Keep two evidence axes.** Record implementation as implemented/partial/missing/not assessed, and execution as not run/blocked/failed/passed pending review/accepted/invalidated. Record scope decisions separately. Source tests, installed development-prefix tests, final-download execution and actual provider results are different observations. Never rename synthetic provider callbacks as operational provider evidence.
5. **Preserve exact bytes and uncertainty.** A changed executable, SDK, receipt, bundled runtime, archive or signature needs its own identity and affected requalification. A timeout, failed acknowledgement or missing runtime file is not proof of no effect. Retain original attempt identities and supported recovery; do not blindly retry uncertain mutations.
6. **Separate acceptance, a testing offer and publication.** A package may be prepared while the full acceptance packet is incomplete. An explicitly approved testing offer must disclose that state and its exact claims. W12 reviews acceptance and records a decision; neither this plan nor checker exit zero publishes anything. No workflow run is a pass until its exact-source result is inspected. The current instruction to push the dossier without waiting for CI does not waive later candidate checks.

The paper's per-category 8.5 threshold and its proposed E3/E4 requirements remain proposals requiring D07. After-packaging timing can coexist with a before-release acceptance gate. The unresolved policy distinction is who must provide which evidence, and which decision requires it. Do not silently substitute an internal mandatory live campaign for the user's user-led sequence, or treat a simulator as proof of a real endpoint. Until a different policy is explicitly approved, preserve the original proposed thresholds as unresolved requirements; do not calculate new scores from this dossier.

## Roles, sequencing and completion records

Role IDs below refer to [owners.json](owners.json). Only @quantDIY's release-approval and primary rollback assignments are carried forward as assigned. Other role labels identify needed work, not named people. D05 assigns implementers, platform operators, reviewers, backup and security-response coverage before the step that actually needs that responsibility. Null operational roles do not prevent authorized local coding, synthetic tests, documentation or campaign preparation. Human signing, live-provider, independent acceptance and operational authority are needed when their respective actions occur. An agent may prepare or review evidence; it does not invent human accountability or signing authority.

| Stage | Work and dependency | Exit evidence |
| --- | --- | --- |
| Preparation now | W01/W06 dossier and this entire plan; no runtime implementation | Exact-source inventory, complete mapping, proposed selections and visible decisions; manual review pending |
| G0: selection | Approve W01 decisions; refine W06; prepare W02–W05 and W08–W11 procedures in parallel | Selected claims, packages, profiles, policy, owners, inputs and recorded blockers |
| G1: existing implementation ready | W02a; W03–W05 audit/stabilization required by selected claims; focused source checks; W06a selection/trust/rights-input readiness | Owner contracts and passing relevant regressions; no silent feature additions; final-byte campaigns ready |
| G2: candidate ready | W07 consumes W01/W06a, constructs actual stages, closes W06b on those bytes, then seals/signs; W13 build interface only if selected | Immutable per-profile candidates, exact source/recipe/runtime/receipt identities, trust and real retrieval identity when available |
| G3: qualification | W08 fresh hosts; W09 external-engine/research/agent; W10 user-led provider/fault/performance; W11 restore/operations | Artifact-bound observations and independent reviews; missing required results remain incomplete |
| G4: decision | W12 reviews all mandatory results and remaining promises | Explicit acceptance/no-go/testing-scope decision and rollback selection; publication authority remains separate |

W02a and local W03–W05 checks can run in parallel with W06a. W06b reviews the actual stages produced during W07 and must close before signing/distribution; it is not a prerequisite to creating the bytes it reviews. W08 host provisioning and W09–W11 campaign design can start before a candidate exists, but their final execution cannot. W09/W11 do not wait for live access unless the particular selected workflow actually depends on it. W10's local fault preparation and performance specification can proceed independently of live profiles; actual provider execution follows packaging. No calendar date substitutes for an exit criterion. Owners estimate effort and external-input dates after selection rather than inheriting the paper's expired illustrative schedule.

For every work package retain: selected source/tree; decision and capability IDs; named implementer/reviewer; dependencies; exact commands/toolchains/input hashes; complete result and refusal observations; skips and unresolved outcomes; output hashes; review decision/time; next blocker owner/date. Use new private evidence directories outside the checkout and installation. Do not commit secrets, private SDK material, paid data, signing keys, raw sensitive provider replies or copied production state. Safe observations may be summarized in the dossier with digest-bound attachments retained through the approved evidence channel.

## W01 — Freeze claims, reconcile policy and assign owners

**State:** dossier prepared for review; scope and assignments are not approved. **Dependencies:** none. **Roles:** `release_coordinator`, `component_maintainers`, `acceptance_reviewer`; `release_approver` decides D01–D03/D05–D07.

**Inputs and owners:** `.github/RELEASE-GATES.md`, `.github/LAUNCH.md`, `.github/launch-tasks.json`, `cmake/mvp-dependencies.json`, `cmake/mvp-release-identities.json`, `cmake/mvp-platform-profiles.json`, `tests/launch-acceptance/plan.json`, current owner SPEC/INSTALL files and this dossier's inventory. The prior readiness review and subsequent freeze explain provenance; current contracts and explicit later decisions govern execution.

**Tasks:**

1. Review the complete root partition and capability groups. Decide shipped versus optional development/SDK material versus unavailable/proposal-only behavior. Bind every selected group to exact independently versioned owners, operations, schemas, target profiles and channels. Repository presence alone is not a shipment decision.
2. Approve or revise the proposed core/SDK packages and optional Cloudflare Intel closure from W06. State source-only/library/fixture capability separately from operational provider support. Resolve all-platform promises that conflict with current vendor artifact admission.
3. For each of the 58 paper criteria, name applicable capabilities, required evidence, responsible role, reviewer and actual implementation gap. Retain all 36 core scenario cells and additional provider/engine/channel cells. The current acceptance checker has no automatic not-applicable pass; an applicability decision cannot be encoded as a fabricated operational result.
4. Resolve D03 for each missing W03–W05 promise and D07 for scoring/simulator policy. Record exact original claim, proposed disposition, rationale, approver and time. Unresolved promises remain blockers; do not change core plan/schema merely to suppress them.
5. Assign D05 people and external-input owners. Distinguish needed-before-candidate inputs from needed-before-provider-campaign inputs. Record dates as requests/estimates, not guaranteed availability. Decide D06 channels; do not presume a separate MCP directory is approved or current.

**Outputs:** reviewed capability/package selections, decision register, owner map, criterion/scenario mapping and prioritized blocker register. Current files are their proposed starting point.

**Focused checks:** dossier validator; `cmake -P tests/mvp-dependencies/verify.cmake`; `cmake -P tests/release-qualification/verify.cmake`; compare inventory/map hashes and every root disposition. Review negative cases for omitted/duplicate roots, unmapped paper IDs, missing core cells, unknown profile, unsupported-as-implemented claims and false approval flags. These are documentation consistency checks, not product acceptance.

**Done:** a reviewer can trace every retained promise to exact scope, owner, dependency and required proof, with explicit approved decisions or named blockers. **Blockers:** unassigned people, undecided scope/platform/channel/policy, and unavailable implementation whose disposition has not been approved. A complete draft alone does not close W01.

## W02 — Prove an independent engine through existing public SDKs

**State:** planned after manual approval. **Dependencies:** W01 claim/profile selection for W02a; final W07/W08 SDKs for W02b. **Roles:** `sdk_research_owner`, relevant `component_maintainers`, independent `acceptance_reviewer` or nominated engine author.

**Existing surfaces:** foundation `libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/operation.hpp` and `INSTALL.md`; SBV `modules/sbv-engine/include/symphony/sbv/{provider.h,sdk.h,sdk.hpp,models.hpp}` and `INSTALL.md`; `libraries/sbv-go/examples/roundtrip`; `tests/native-sdk-distribution/{README.md,manifest.json,run.sh}`; `tests/sbv-launch-boundary/{README.md,run.sh}`; `knowledge/sov/PROCESS-EXECUTION.md`. The installed SDK manifest owns current producer/consumer coverage.

**W02a tasks before candidate:**

1. Select a small meaningful deterministic calculation, independent expected-result oracle, explicit input limits and compatibility contract. Keep its source project independently buildable outside Symphony's checkout with its own identity/version. An example or qualification fixture is not a newly mandated platform owner.
2. Select two already supported public routes. A candidate composition is a standalone native SDK-linked program and the existing SBV provider/feedback ABI exercised by installed consumers. Confirm that both meaningfully execute the chosen calculation and preserve their own documented semantics before coding. A route cannot be claimed merely because a process launches.
3. Copy only public example source, use installed CMake imports/headers and exact package versions, and remove access to private headers, build caches and developer prefixes. Retain build/run recipes and complete numerical output. Publish author instructions with the selected SDK payload or its exact matching source material.
4. Prepare compatible and incompatible engine/SDK versions, a bounded interrupted lifecycle, unrelated user-data sentinels and an administrative-stall experiment. If no two existing routes meet the chosen criterion, record the specific gap for D03; do not build a generic loader, universal engine framework or new deployment service under this work package.

**W02b in W09:** repeat the independent build and both routes from final downloaded SDK/package bytes on each applicable core profile; execute UE1–UE6, including a documentation-only agent attempt. Keep first failure and undocumented assumptions visible.

**Outputs:** independent example source/recipe, oracle and complete expected outputs, public-route contract, compatibility/lifecycle test matrix, artifact-bound author and agent records.

**Focused checks:** run only affected installed consumers plus `tests/native-sdk-distribution/run.sh` for the selected group (command C2 below); use `tests/sbv-launch-boundary/run.sh` where that route is selected. Negative cases: missing/wrong SDK, private-header fallback, incompatible identity/schema, malformed input, lost reply, interruption, changed owned bytes, removal preserving sentinels, and admin stalls where no synchronous dependency is declared. The existing SDK distribution gate's removal checks remain on a disposable prefix.

**Done:** UE1–UE6 have reviewed final-byte evidence with complete result agreement and no hidden checkout/setup. **Blockers:** unapproved routes, unavailable target SDK bytes or an exact criterion that the existing public interfaces cannot satisfy. Preliminary development-prefix passes close preparation only.

## W03 — Qualify existing credential boundaries and resolve missing promises

**State:** freeze-bound audit/stabilization; broad implementation from the earlier plan is deferred. **Dependencies:** W01 exact credential/provider claims; W06 identities; W07/W08 for final execution. **Roles:** `authority_security_owner`, affected `component_maintainers`, `provider_review_owner`, independent `acceptance_reviewer`.

**Existing owners:** `modules/secure-identity-access-governance/CREDENTIAL-QUALIFICATION.md`, `CREDENTIAL-ADMISSION.md`, `CREDENTIAL-JOURNAL.md` and `internal/credential/{admission.go,admission_store.go,dispatch.go,use.go,outcome.go}`; `internal/server/cloudflare_use.go`; `modules/ssiag-provider-private-store`; `modules/sov-cloudflare-adapter`; `modules/ssiag-provider-macos-keychain`; native consumer interfaces in `modules/native-source-support-cpp/include/symphony/source/support.hpp` and `modules/sqav-databento-dbn-cpp/include/symphony/sqav/databento/http.hpp`.

| Earlier W03 promise | Current decision-bound treatment |
| --- | --- |
| W03a permission-backed lease issuance/revocation and audit wiring | Inspect/test implemented durable admission, generation and uncertainty rules. General L14 lifecycle wiring remains missing; retain as blocked, explicitly defer the dependent claim, or obtain a separately scoped freeze exception. |
| W03b general native provider/recipient admission and protected one-shot delivery | The Cloudflare route has its separate implemented native delivery. Qualify that exact route; do not generalize it into completion of the general dispatcher. Missing general backend stays in D03. |
| W03c operational Apple data-protection Keychain | Existing Swift protocol/peer/readiness tests do not establish item access, signing policy or actual login context. No operational Keychain implementation is scheduled automatically. |
| W03d Linux Secret Service | No existing provider is inferred from a name or from Cloudflare private-file storage. This remains a distinct missing implementation and explicit decision, not a new module to create during the freeze. |
| W03e native source/Databento consumers | Preserve the exact public callback interfaces and their qualification limits. Do not claim an admitted operational bridge until that bridge is implemented and tested; construction requires D03 if it changes frozen capability. |

**Tasks:** audit selected credential routes against exact provider/executable/recipient/account/resource/operation identities; verify expiry/revocation, intent/outcome audit separation, byte/deadline limits and native cleanup. Review the merged Cloudflare fixes rather than repeating them as open defects: delete recovery requires list/collection authority; object observations bind the selected ID; configuration responses use exact-path and optional tunnel/account identity; proven pre-child connector failure is distinguished from possible execution. Reproduce any newly found defect before an owner-scoped stabilization change. Keep opt-in private-store confidentiality/same-UID limits explicit; installation does not activate a provider or seed a secret.

**Outputs:** route-by-profile implementation/qualification table; safe native credential evidence; explicit disposition of W03a–e promises; focused fixes only if needed; final deployment campaign instructions for user testers. Actual operational provisioning is a separate authorized action with selected nonsecret profile references.

**Focused checks:** credential package race/process regressions; complete private-store native suite; Cloudflare resource/connector suites; SSIAG Cloudflare authority tests; existing Swift tests for its actual readiness scope (C3/C4). Required negatives include wrong recipient/reference/generation, expired/revoked admission, unavailable audit, interrupted delivery, lost acknowledgement, persistence failure, denied collection read with zero dispatch, wrong object before retention, and connector pre-start versus post-start failure. Preserve unknown state and verify zero blind redispatch.

**Done:** each selected implemented route has source evidence and then final-package success/refusal/recovery evidence for its promised profile; every missing route has an approved claim disposition or remains an explicit blocker. **Blockers:** missing implementation decisions, operational identities, signing/login deployment context, real authorized profile for later user qualification. Missing live credentials do not block package preparation for disclosed existing capabilities.

## W04 — Specify and qualify selected provider operations

**State:** inventory, existing-behavior stabilization and campaign preparation; no automatic new vendor bridge. **Dependencies:** W01, applicable W03/W05 contracts; final W07/W08 for W04b in W10. **Roles:** `component_maintainers`, `provider_review_owner`, user-led `provider_testers` after packaging.

| Selected owner | Required boundary and preparation |
| --- | --- |
| `modules/sqav-databento-dbn-cpp`, `sqav-databento-reference-cpp`, `sqav-request-engine`, `sqav-attempt-engine`, `sqav-capture-cpp` | Distinguish wire/capture/request/attempt fixtures and retained results from actual entitled historical acquisition and the missing general credential bridge. |
| `modules/sqav-fred-cpp` | Distinguish FRED/ALFRED request/response support and public SDK consumers from actual admitted native credential use. |
| `modules/sqav-news-api-cpp` | Vendor-neutral ingress; select authorized content and correction/retention semantics rather than inventing a universal news-provider integration. |
| `modules/scabv-ibkr-client-portal-cpp` | Externally authenticated session assumptions, endpoint and read-only scope; owning session methods do not perform external login automatically. |
| `modules/scabv-ibkr-tws-cpp` | Exact TWS SDK 10.45, licensed provenance and concrete vendor bridge/conformance gap; SDK-free host fixtures do not close it. Preserve `modules/scabv-ibkr-tws-cpp/tests/sdk1045-conformance` as a separately supplied SDK campaign. |
| `modules/srrapiv-runtime-cpp`; `knowledge/sqv/srrapiv/{BINDINGS.md,MAINTENANCE.md}` | Exact R API+ 13.7.0.0 and approved platform/library closure; SDK-free runtime tests, actual SDK compile/link and provider qualification are separate. `src/vendor.hpp` is the existing bridge owner. |
| `modules/sov-cloudflare-adapter`; `knowledge/sov/cloudflare`; `tests/cloudflare-release/support-matrix.json` | Closed resource operations, native credentials, connector/session and offline toolchain boundaries. All actual live cells require user evidence. Project inspect/plan is not prepare/build/deploy/status/reconcile/migrate; rotation remains unavailable. |

**W04a tasks before candidate:** enumerate exact selected operations, parameters, account/environment model, SDK and vendor binary versions/digests, target support, session/login assumptions, pacing, deadlines, cancellation and shutdown dispositions. Mark callable source/library surfaces separately from a turnkey live capability. Prepare safe request/response metadata, correlation and independent endpoint truth capture. Determine which faults an actual endpoint or authorized controlled environment can produce. A generic simulator does not fill an endpoint gap. Inspect/build existing owners and fix proven defects within their contract; a required missing bridge triggers D03 rather than automatic implementation.

**W04b tasks after packaging:** user testers select authorized profiles, entitlement and content rights, bounded read-only operations and a time/request/spend ceiling. Run exact final artifacts, capture success/refusal/expiry/disconnect/cancellation and supported reconciliation, then return redacted artifact-bound results. A proposed trading or resource-creating scenario needs explicit effect scope and separate authorization; plan approval alone does not authorize orders, charges or provider resource creation. Do not resume the historical Orangeburg investigation without a new authorized task.

**Outputs:** provider operation/profile/SDK matrix, exact commands discovered from installed owner interfaces, tester instructions, redaction rules, raw safe observations and review results; missing SDKs/endpoints recorded by owner and needed-by stage.

**Focused checks:** affected native suites and copied SDK consumers from `tests/release-qualification/manifest.json`; SRRAPIV CTest and installed qxctl tests; Cloudflare reducer and selected source/native suites (C3/C4/C5). Negatives include unavailable SDK, wrong artifact/platform/endpoint/account, unentitled response, pacing refusal, malformed/bounded responses, event gaps, lost reply and no automatic duplicate request. An optional test skip remains not run.

**Done:** W04a has exact supported/unsupported scope and executable campaign specifications without requiring live access; W04b closes only with actual reviewed results for every promised cell. **Blockers:** missing bridge decisions, vendor SDK provenance, unqualified vendor target, authorized user profile/content and unavailable mandatory fault mechanisms. W04b blockers do not prevent W07 packaging of disclosed implemented scope.

## W05 — Verify the selected account, risk and execution-state composition

**State:** audit and qualification specification; new economic-risk, account-enforcement or lossless-state implementation is deferred under the freeze. **Dependencies:** W01 selected trading/state claims; W03/W04 exact authority and provider semantics. **Roles:** `state_recovery_owner`, `provider_review_owner`, `authority_security_owner`, relevant `component_maintainers` and independent reviewer.

**Existing owners:** `modules/srrapiv-runtime-cpp/SPEC.md`, native runtime/attempt/callback implementation and tests; `knowledge/sqv/srrapiv/BINDINGS.md`; provider owner contracts from W04; `knowledge/sov/PROCESS-EXECUTION.md`; `modules/sbv-engine` and its existing account-template/feedback tests where selected. qxctl owns administration, not strategy decisions or an invented risk engine.

**Tasks:**

1. Trace each promised order/control path and identify the actual owner of account/environment checks, economic limits, stale-data policy, durable state and reconciliation. A supplied, user-built or external composition is valid if explicitly selected and qualified; it is not automatically part of Symphony's package.
2. Preserve existing SRRAPIV boundaries: `allow_provider_calls`/`allow_trading` and method allow-lists are local admission; `account_ref` is correlation, not an AccountInfo filter. SDK return is not endpoint completion. The bounded callback queue is not lossless state, and host stop is not order cancellation or flattening.
3. Define independent expected state for duplicate/reordered events, gaps/overflow, partial fills, cancel/replace races, wrong account and endpoint/local disagreement. Record actual provider identities and query methods needed to establish truth. Prepare safe local fixture tests for existing semantics without using them as evidence of live account risk.
4. Verify that durable attempts retain unknown outcomes across lost replies/restarts and never redispatch automatically. Specify inhibit, cancellation request, optional explicitly selected flatten and administrative stop separately. If the selected owner has no authoritative reconciliation, report unresolved state; do not manufacture convergence from queue emptiness.
5. For each original W05 construction promise choose D03: retain as blocked; explicitly defer/change the affected reference-composition claim; or authorize an exact exception with its owner, contract and tests. Do not turn a qualification example into a mandatory universal user policy or claim hostile same-UID containment.

**Outputs:** supported route/trust model, control/disposition table, state-machine and economic oracle, fault/correlation plan, existing-behavior regression evidence, missing-feature decisions and final campaign envelope for W10.

**Focused checks:** SRRAPIV attempt/idempotence/callback/supervision CTests, its installed lifecycle and lost-status-reply regressions; existing SBV/SOV loss/cancellation tests for selected research execution. Negative cases: conflicting attempt identity, loss before/after persistence, overflow/gap, wrong generation, cancellation differing from stop, and unavailable reconciliation. Any economic/account guarantee needs its selected owner's actual independent tests; a local binding check is insufficient.

**Done:** every selected claim is either demonstrated through its actual owner or remains explicitly blocked/deferred by decision; no false terminal state or duplicate dispatch is accepted. **Blockers:** unassigned composition owner, absent enforcement/reconciliation implementation, missing endpoint truth or authorized effect scope. These are blockers to the associated claims, not a reason to halt unrelated offline/package work.

## W06 — Close exact payload, notices and publisher trust

**State:** proposal prepared for review, no final payload or signing approval. **Dependencies:** W01 selection for W06a; actual W07 stages for W06b. W06a can progress alongside W02–W05. **Roles:** `packaging_engineer`, `payload_reviewer`, `signing_custodian`, `release_approver`.

**Existing owners:** [PACKAGE-AND-TRUST.md](PACKAGE-AND-TRUST.md), `cmake/mvp-dependencies.json`, `cmake/mvp-release-identities.json`, `cmake/notices/PUBLISHER-REVIEW.md`, `tests/dependency-notices`, `cmake/release/{FinalizeNativePayload,PrepareCandidate,SignCandidate,VerifyCandidate}.cmake`, `tests/package-release`, and each selected module's installation/receipt contract.

**W06a — selection and construction inputs:** approve exact package/profile contents and independently versioned component closures, proposed release-only delivery contract below, trust policy and necessary rights/source inputs. The proposed `PKG-CORE-<profile>` entries are delivery sets of separately sealed artifacts. Public SDK selection aliases complete `PKG-NATIVE-<component_id>` receipts; it is not a headers-only repack. Review `PKG-CF-INTEL` separately from the four core sets, plus selected `PKG-GO-<component_id>`, `PKG-SWIFT-MACOS-METADATA` and `PKG-SOURCE` artifacts. Decide shipped versus system dependencies, and optional DuckDB/PDFium/vendor material; no private SDK or paid data is silently bundled. W06a feeds W07 construction without pretending final hashes already exist.

Construct fresh finalization-stage native builds using the mapped owner options and dependency DAG. Qualification prefixes have absolute install RPATHs and external build dependencies; do not package them directly. Resolve relocation/loader changes in the approved finalization stage before receipt hashing. Native `PrepareCandidate.cmake` consumes its native receipt closure; it does not automatically absorb the seven Go installer formats or Swift bundles into one universal `candidate.tar`. Retain the existing exact owner installers/receipts for those artifacts and define the delivery-set manifest/trust binding explicitly. Any claimed cross-artifact final-download verification gap remains a W06/W07 blocker until a reviewed bounded packaging solution is authorized; do not invent a successful universal verifier.

**W06b — actual-byte review during W07:** run the existing notice collector on actual selected final stages, then resolve recursive loader/TLS identities, embedded/static notices and corresponding source/build/install material. The collector prepares evidence and does not make the publisher's redistribution decision. Use the exact selected curl/zstd/DuckDB/PDFium/vendor provenance; do not upgrade to a convenient current version. Verify finalization order so any strip/sign/RPATH change happens before immutable receipt hashing. If a platform signing/notarization step changes sealed bytes, rebuild/reseal the complete candidate with the approved recipe rather than editing a receipt in place. Close the actual payload review before candidate/index signing and distribution, then verify that signed final identities match that review.

D04 must select publisher identity, independent signer-policy distribution, key custody/rotation/recovery and platform trust procedure. Keep key bytes private and never use package fixture keys as publisher authority. Do not invent a public URL, notarization result or legal approval. Test authorized trust mechanics with disposable fixtures; final candidate approval binds real identities. The Swift `scripts/build-production-bundle.sh` path requires a notarization profile and performs an actual submission/staple, so it is not a harmless metadata-build command; use it only through the approved identity and external-action scope. A metadata-only Swift package does not establish operational Keychain access.

**Outputs:** reviewed per-profile recipe, exact path/component/runtime inventory, notice/source material register, unresolved payload issues, independent trust policy and authorized signing procedure; all decisions bind the eventual candidate identity. The current package-plan proposal is not the final file inventory.

**Focused checks:** dependency-map verification; `tests/dependency-notices/run.sh`; package lifecycle fixture; exact receipt and loader checks (C6/C7). Negatives: changed/missing/symlinked notice, unclassified transitive library, wrong SDK/target, wrong signer, tampered manifest/archive/file, unknown/duplicate inventory entry, unsafe finalizer path and interruption before sealing. Check corresponding source against the actual candidate, not an unrelated main URL.

**Done:** W06a has approved construction selections, contract and trust/rights inputs; W06b accounts for every actual final payload byte and runtime dependency, its source/notices, finalization order and trust policy, with no unresolved mandatory payload issue before signing/distribution. **Blockers:** unselected contents, unresolved runtime/license material, proprietary distribution permission, signer/custodian/policy and platform-signing decisions. Record whether each blocks construction or later signing/distribution. Live provider profiles are not W06 prerequisites.

### W06/W07 release-only delivery contract — proposed implementation after approval

The current `acceptance-gates.go` binds one primary artifact per profile, and `candidate-host.go` admits the native candidate format. The selected core delivery set also contains Go commands and may contain Swift/vendor artifacts. This is a concrete packaging/evidence gap to implement and test after approval; primary-native verification is not proof of the whole set. The following is a proposed bounded release-tooling design, not a product installer, new runtime feature or existing implemented protocol. D01/D04 must approve its schema/trust choices before coding.

**Future file owners:** add a delivery schema beside existing release helpers at proposed `cmake/release/delivery-set-v1.schema.json`, a narrowly scoped proposed `cmake/release/SignDeliverySet.cmake`, and proposed `tests/launch-acceptance/delivery-set.go` plus `delivery-set_test.go` for prepare/check of release evidence. Extend `tests/launch-acceptance/README.md` and the existing package workflow with focused fixtures. Keep `acceptance-gates.go`, its v1 packet/schema semantics and native candidate schema unchanged. New names are design targets and do not exist in this dossier increment. Use the repository's existing Go/CMake toolchains; add no product Python dependency or service.

**Signed delivery index, proposed `symphony.release.delivery-set.v1`:**

- Top-level fields: protocol/version; immutable set ID; exact first-party source commit/tree; profile/OS/architecture; approved capability/package-selection digests; dependency/identity-map digests; independently selected signer-policy digest; ordered artifact records; matching source/notice/build-evidence references; `publication_authorized=false`. Sign exact index bytes under a separately approved SSHSIG namespace (proposed `symphony-delivery-set`); the independent policy must explicitly admit that signer/namespace. Do not trust a signer key or policy solely because the index names it.
- Each artifact record: unique artifact/package ID; closed artifact kind; safe relative filename, size and SHA-256; component/package versions and target; dependencies by artifact ID; exact first-party source identity or separately typed upstream vendor provenance; final payload inventory digest; expected receipt/manifest, signature and per-artifact independent signer-policy references where applicable; closed verification-adapter ID/version; reviewed install/removal recipe digest; matching notice/source/build-record references. Native records bind the entire native candidate file set, including its manifest/signature and evidence, not only `candidate.tar`. The index-signing policy and each artifact-signing policy are distinct bindings even when the approved policy bytes happen to match. Vendor/source artifacts remain explicitly typed and never masquerade as a first-party same-source executable.
- Refuse duplicate/unknown fields and IDs, escaping/symlink paths, unsupported artifact kinds/adapters, cycles or missing required dependencies, absent/extra selected artifacts, wrong sizes/hashes, contradictory targets, mixed first-party source commits, and mismatched map/selection/policy bindings. Bound metadata/file counts and bytes before allocation; start with the existing 16 MiB evidence and native archive ceilings where applicable and define separate finite limits for other carriers. Index verification performs no download, install, service start or provider action and never executes commands embedded in metadata.

**Owner-specific verification adapters:**

| Artifact family | Reuse and required qualification |
| --- | --- |
| Native candidate | Invoke the existing reviewed `VerifyCandidate.cmake` and `candidate-host.go` through fixed adapter behavior; require their source/profile/archive/manifest/policy identities to equal the signed index. Preserve bounded extraction and exact receipt/file checks. |
| SSIAG, STAV and Accordare Go receipt packages | Build from the exact recipes in `.github/scripts/build-mvp-go-identities.sh`. Use actual receipt-v2 owner readers: SSIAG `internal/packageinstall/package.go` (`InspectExecutable`), STAV `internal/lifecycle/lifecycle.go` (`VerifyExecutable`/`VerifyInstalled`), and Accordare `internal/packageinstall/package.go` (`Inspect`). These paths are under their respective module roots. Existing qxctl `internal/knowledgeengine/receipt_v2.go` supplies a shared strict entry-point boundary where its declared spec fits. Add a qualification harness within an owning Go module if needed to call an internal reader; do not duplicate permissive receipt parsing across modules or invent a public receipt API. Retain exact byte/identity/owner/mode/lifecycle evidence and verify no enrollment or supervision occurs. |
| qxctl and explicit SOV executables | Use exact executable/build identity and a release-only immutable file inventory, plus their existing selection/install/removal contracts. They do not acquire a fictional native receipt merely by inclusion in a delivery set. `tools/qxctl/tests/reproducible_build.sh` and the other command identity recipe supply build evidence, not final signing proof. Protected profile-admin/carrier fixtures retain their actual manifest, receipt and root-ancestry requirements; inspect `tools/sov-transport-profile-admin/internal/store/receipt.go` and `tools/sov-ssh-adapter/profile/receipt.go` rather than treating ordinary SSH v1 as protected installation. |
| Swift metadata bundle | Use the actual finished-bundle installer and `modules/ssiag-provider-macos-keychain/Sources/SSIAGMacOSKeychainSupport/ReceiptV2.swift` / `Lifecycle.swift` validation. Record all receipt-owned bundle files, compiled identity and separate approved codesign/notarization evidence. Add a qualification-only harness if no read-only public route supplies the required evidence. All operational credential flags remain false. |
| Selected vendor bundle and source material | Reuse the Cloudflare bundle/admission and protected runtime owner checks for exact cf/Node/cloudflared/carrier/SSH files. Source archives get an exact safe file inventory and matching provenance/notice review, not an executable receipt. Missing upstream rights/admission or unsafe archive layout fails that artifact. |

Verification-tool source and executable hashes are retained independently. An untrusted archive cannot choose an arbitrary program to run; adapter IDs are a fixed implementation allow-list. A claimed owner-verification JSON file is not sufficient: the release harness must run the reviewed verification step against the actual artifact/install and retain its exact input/output hashes. Where an owner check requires executing authenticated code, perform that step only after byte/trust verification, in the selected private fresh-host campaign. Do not weaken receipt ownership or private-prefix rules to reuse a convenient staging directory.

**Versioned acceptance companion, proposed `symphony.release.delivery-acceptance.v1`:** retain the unmodified core v1 packet and add a companion binding its hash, plan hash, approved mapping hash, signed delivery-index hash, profile and primary native artifact ID. Resolve that ID in the index; require its native archive SHA-256, manifest digest, independent artifact-signer-policy digest and source commit to equal the v1 profile fields `artifact_sha256`, `manifest_sha256`, `signer_policy_sha256` and `source_commit`, respectively, and require its profile identity to equal the packet's profile key. An artifact ID is never written into a digest field. Record verification/install/receipt/loader/trust evidence per artifact, and for every core/supplemental scenario list exact required artifact IDs and their digests, receipt/manifest/policy identities, complete input/output evidence, result/scope/skips and applicable paper criterion IDs. The approved mapping determines the required artifact set; observations cannot omit an artifact simply because it failed. Record all 36 core cells, all 58 criteria and applicable supplemental cells without pretending they share a single runtime dependency set.

The companion checker runs the existing v1 checker and then validates artifact-set coverage and its own exact bindings. It must preserve v1 incomplete/invalid outcomes; it cannot turn a primary-only pass into a delivery-set pass when another selected artifact is unverified. Return separate structural completeness and missing/invalid records; release acceptance/publication remain false. Preserve actual operational provider meaning and human review. The companion adds evidence binding; it does not reclassify fixture/provider outcomes or infer that a valid signature proves a correct build.

**Implementation increments and tests:** first schema/parser/binding tests with tiny synthetic artifacts; then native adapter reuse; then each selected Go/Swift/vendor owner adapter and its real staged artifact; then companion coverage; finally focused workflow integration and four-profile final-artifact execution. Extend existing package/evidence fixtures rather than introducing a parallel broad product suite. Mandatory negatives: altered index/signature/archive/receipt/file; omitted or substituted Go/Swift artifact; valid native primary with missing secondary artifact; mixed source/target/map; wrong signer/policy/namespace; duplicate ID; unselected artifact; dependency mismatch; unsafe path/link; changed owner modes; invalid adapter; incomplete/failed/skipped artifact evidence; scenario bound to the wrong artifact set; mutable/reused output; and post-verification byte drift. Include a valid multi-artifact fixture whose component versions differ intentionally, proving independent versions remain supported. Add positive/negative owner install/removal cases that preserve sentinels and create no service, grant, secret or provider effect.

**Release-only done criterion:** a clean fresh-host campaign can verify the complete signed delivery set, execute each existing owner install contract and bind every required scenario to the artifacts it actually used; omission/substitution cannot pass through a valid primary-native record. All source tests remain fixture evidence until W08 executes final bytes. If a required owner cannot be verified without changing a product contract, stop that adapter at D03 rather than disguising feature work as packaging.

## W07 — Build and retain immutable candidates

**State:** planned, no candidates created by this dossier. **Dependencies:** approved W01/W06a; W02a; implemented and stabilized selected W03–W05 claims; conditional W13 build interface if selected. W06b closes on actual W07 stages before signing/distribution. Live results are not a prerequisite. **Roles:** `packaging_engineer`, `component_maintainers`, `release_coordinator`, independent source/CI and payload reviewers.

**Existing owners:** `.github/workflows/{mvp-dependencies,release-qualification,package-release,qxctl-standalone,ownership-regression,duckdb-connectors,sqv-retained-inspection,cloudflare-native,transport-profiles,srrapiv,codeql}.yml`; dependency/release manifests; `tests/platform-qualification/capture.sh`; `cmake/release`; owner finalization/receipt tooling. Workflow presence is not a run result.

**Tasks:** select and record a clean committed candidate source/tree after approved changes, refresh the exact-source scope inventory and reconcile dossier fingerprints. Build all selected native/Go/Swift/runtime closures on the chosen target recipe. Run affected source checks during changes, then the applicable complete candidate source gates once at this milestone. Inspect exact run/test identities, failure/skip records and static-analysis coverage; do not infer a pass from a push or stale run.

Build the real product payload into new finalization stages; complete finalization, receipt sealing, W06b notice/source review and platform/build observations. Implement and qualify the approved release-only delivery index/adapters/acceptance companion above. Prepare the native receipt candidate with `PrepareCandidate.cmake`, and separately prepare each selected Go/Swift artifact through its reviewed owner packaging contract. Bind all constituents into the approved delivery-set inventory rather than assuming the native carrier admits every installer format. Authorize/sign through the selected custody process and verify each artifact through its independent policy and exact verifier (native example C7). Retain build recipe/toolchains, map and native-interface versions, actual files/sizes/digests and artifact/manifest/policy identities. Establish a real retrieval location only through its authorized channel; record the resolved URL/time when bytes are actually retrieved. A local archive with no URL is a local candidate, not a public download.

**Outputs:** immutable per-profile candidates, signed manifests and independent policy references, complete safe build/payload evidence, matching source material, selected capability manifest, failed-run records and actual retrieval identities when available.

**Focused and milestone checks:** source gates for the chosen closures; package verification, extraction and exact receipt admission; wrong source/tree/profile/signature/evidence/archive bindings must fail. Run production binaries/SDK consumers from staged/relocated payloads and verify loader closure without developer paths. The two-probe package fixture alone cannot qualify the real bundle.

**Done:** actual candidate identities and reviewed provenance/trust exist for each selected package/profile, and W08 can retrieve precisely those bytes. **Blockers:** unresolved W01/W06 decisions, mandatory implementation gaps, failed candidate gates, unavailable build hosts/dependencies, unapproved signing or retrieval channel. Record whether a blocker affects construction, testing distribution or full acceptance; do not collapse them.

## W08 — Qualify final downloads on fresh hosts

**State:** host/procedure preparation may start after approval; final execution waits for W07. **Dependencies:** exact candidate, independent policy and approved package/profile manifest. **Roles:** `platform_operator`, `packaging_engineer`, `acceptance_reviewer`.

**Existing owners:** `tests/launch-acceptance/{README.md,plan.json,candidate-host.go,candidate-host_test.go}`, `tests/package-release`, `tools/qxctl/internal/{knowledgeengine,knowledgelifecycle}`, selected owner installers and SDK receipts.

**Tasks:** prepare four independent fresh-host lanes with actual image/kernel/CPU/runtime/filesystem/toolchain and clean pre-install inventory. Retrieve every approved constituent artifact and independent trust policy separately; retain actual resolved URL/time/size/digest. Use `candidate-host` for the admitted native candidate to verify and safely extract into a new private prefix; use the reviewed exact owner verification/install procedures for selected Go/Swift artifacts. Bind these additional artifact identities and complete delivery-set evidence into the profile's scenario observations; the current packet's single primary artifact field does not prove arbitrary extra downloads. If the selected delivery set cannot be fully bound/verified through the reviewed recipe, retain that gap for resolution rather than falsely treating the native helper as universal. Then execute the approved installation/discovery/version and loader scenarios against the actual payloads. The helper never runs downloaded code and deliberately leaves fresh-host/provider/release flags false; the operator supplies substantive execution evidence.

Exercise side-by-side install, exact version selection, compatible upgrade/rollback, interruption and resumable removal. Preserve unrelated files and actual user-state digests. Run without the source checkout, build-time dependency prefix or module cache; do not repair a broken candidate from the developer environment. A supplemental Cloudflare Intel host remains a separate approved lane; it does not replace macOS ARM64.

**Outputs:** per-profile retrieval/trust/extraction/receipt records, host provenance, complete command and loader logs, lifecycle/rollback/removal state comparisons and scenario envelopes for relevant core cells.

**Focused checks:** helper/package regressions and actual candidate C8/C9 execution. Negatives: wrong signer/profile, tampered bytes/manifest/receipt, missing or unlisted files, unsafe archive member, existing destination, incompatible version, partial installation/removal, foreign-file preservation and source/cache fallback. Inspect failures without overwriting evidence; retries use fresh work directories.

**Done:** applicable fresh-host/lifecycle cells passed on all four exact final candidates with no mandatory skips and independent review. Other core scenarios remain visible for W09–W11; extraction alone never closes all 36. **Blockers:** unavailable clean host, missing candidate/trust/loader dependency or incompatible retained data. A failed profile does not become optional without an explicit scope decision and preserved original requirement.

## W09 — Execute research, author and agent journeys

**State:** preparation independent of live-provider access. **Dependencies:** W02a and W07/W08 for final execution; selected source data/decoder rights for documentary workflows. **Roles:** `sdk_research_owner`, relevant `component_maintainers`, `platform_operator`, independent numerical/author/`acceptance_reviewer`.

**Existing owners:** `tests/sbv-launch-boundary`, `tests/native-sdk-distribution`, `tests/shv-release-qualification`, `tests/shv-release-qualification/examples/mvp-hardware`, `tests/snv-qualification`, `modules/sacv-engine/tests/fixtures/README.md`, installed public SDK examples and exact qxctl help/registry. Use these as source specifications for final artifacts, not as substitute evidence.

**Tasks:** execute W02b/UE1–UE6 and applicable C01–C06 criteria on fresh final installations. Run bounded CPU research with attributed deliberately synthetic input, independent full-output oracle, exact/tolerance policy and selected time/correction/ordering semantics. Retain complete input and result evidence rather than screenshots or selected scalar metrics. Run the selected hardware/source/node documentary lineage, receipt-bound native replay and protected workflow; distinguish unavailable macOS observer or optional PDF decoding from a passing implementation.

Give an independent author/agent only the shipped docs and package/SDK identities. Record discovery, meaningful calculation, both public routes, compatibility change, recovery/removal and every clarification needed. Test administrative stalls only for routes whose contract declares no synchronous administrative dependency. Do not infer a latency/performance claim from a small correctness test. Update broken documentation or examples as stabilization, rebuild affected package material if necessary, and repeat invalidated final-byte cells.

**Outputs:** complete numerical/temporal comparisons, source/hardware/knowledge results, independent author and agent transcripts with safe inputs, all UE results, supported cancellation/recovery and compatible lifecycle evidence.

**Focused checks:** selected C2 and installed SHV/SNV/SBV campaigns (C10), with malformed inputs, unsupported semantic request, incompatible public version, cancellation, killed worker, lost reply, mixed revision/export chunks, stale selection and tampered receipts. For an adopted criterion not covered by an existing harness, add a narrowly scoped fixture/observation script through the owner, not a new engine framework.

**Done:** every applicable mapped research/agent/author criterion has reviewed final-byte observations, independent oracle agreement and no hidden setup or mandatory skip. **Blockers:** missing exact SDK, undocumented route, absent selected data/decoder provenance, unavailable required native target, or a criterion requiring new behavior that D03 has not resolved.

## W10 — Execute provider, fault, recovery and performance campaigns

**State:** prepare local harnesses, workload definitions and tester instructions after approval; live execution is user-led after packaging. **Dependencies:** W04a/W05 contract and W03 route evidence; W07/W08 final artifacts for actual execution. **Roles:** `provider_testers`, `provider_review_owner`, `state_recovery_owner`, `performance_owner`, `authority_security_owner`, independent `acceptance_reviewer`.

**Existing owners:** selected W03–W05 providers and tests; `tests/cloudflare-release/{manifest.json,support-matrix.json,gate.go}` and `knowledge/sov/cloudflare/I6-QUALIFICATION.md`; `modules/srrapiv-runtime-cpp` attempt/callback supervision; SBV/SOV process recovery; owner journal and public operation schemas. No existing reducer is a universal endpoint or benchmark oracle.

**Tasks:** predeclare the exact candidate, operations, account/environment, allowed effects, workload, request/spend ceilings, clocks, output correctness oracle, latency/throughput/backlog/resource budgets, fault points and stop conditions. Select duration from the promised claim; the paper's proposed 60-minute nominal-plus-burst floor is not a reliability certification. Separate small correctness, production-path functional, load and soak evidence. Preserve all samples, failures and slow tails.

Execute W04b from user profiles, then the applicable authorized fault campaign. Prove the fault actually occurred at the selected boundary: loss before/after persistence, lost acknowledgement, denied/revoked authority, disconnect/cancellation, event duplication/reordering/gap, overflow, or unavailable audit/reconciliation. Compare effects against independent endpoint/retained-state truth, preserve original identities and require no blind retry or false terminal success. Cloudflare deletion recovery must request the exact list/collection authority and reject mismatched object observations; connector proof distinguishes pre-child failure from possible execution. Running a connector process alone does not prove provider readiness.

Where an endpoint cannot induce a required fault, retain the gap and separately describe any simulator's version, validated behavior, preserved production boundaries and omissions. D07 determines the simulator's permitted contribution; it never becomes an operational provider result under the current checker. Required trading/risk behavior stays blocked if its selected owner is absent, even if an SDK call succeeds.

**Outputs:** user-safe campaign instructions; artifact/profile/operation-bound results and fault proof; independent provider/state comparisons; complete performance samples and budget decision; residual-resource/account state, cleanup/recovery evidence and explicit unperformed cases.

**Focused checks:** C3–C5 for preparation; actual selected endpoint operations only under authorized tester scope. Include wrong account/environment/entitlement, deadline, expiry, unavailable audit, duplicate attempt, post-effect lost reply, restart, callback gap and limit enforcement as applicable. Cloudflare reducer exit zero means local suites only; `first_milestone` remains open until its separate live requirements are reviewed.

**Done:** all required selected provider/fault/performance cells have substantive accepted evidence and no hidden skipped fault. **Blockers:** authorized profiles, SDK/environment/endpoint availability, effect authorization, missing selected state owner, uninducible required fault and unmet budgets. Report partial completion accurately and keep independent lanes moving; package preparation is not rolled back solely because live access is unavailable.

## W11 — Exercise restore, support and backup operation

**State:** runbook/owner preparation may proceed before candidates; actual final-install restore depends on W08. **Dependencies:** W01 assignments/policy, W08 exact installation, selected W09/W10 retained state. **Roles:** `rollback_primary` (@quantDIY), `rollback_backup`, `security_response_owner`, `state_recovery_owner`, `platform_operator`, `acceptance_reviewer`.

**Existing owners:** `.github/LAUNCH.md`, `SECURITY.md`, `tests/launch-operations/{README.md,RETENTION.md,AUTHORIZATION.md}`, `tools/qxctl/internal/{snvstate,scvstate,scvcorpus,shvstate,shvpublicationstate,scvtransfer,shvtransfer,ssiagclient}`, selected native owner stores and package lifecycle contracts.

**Tasks:** name backup/module escalation/acceptance/security-response people and a realistic coverage commitment. The selected support destination is `support.quanux.org`; verify private intake, access, routing and response ownership through an explicitly authorized drill. Drafting a plan does not authorize sending a ticket, email or message, or invent an SLA.

Declare recovery-time/data-loss objectives and supported backup mode. Record exact installations, state roots, pending IDs, references and ownership, quiesce cooperating writers and database/WAL activity, then use the established authorized backup tool to copy the complete selected set into a new private destination. Compare before/after inventories. Restore to a clean installation with the original required owner/receipt bindings; do not rewrite history/prefixes to make it fit. Inspect pending operations and recover with fresh authority. Preserve unknown effects and the immutable backup.

Have the backup operator follow shipped procedures to diagnose an injected incident, select a verified compatible prior package, replay a restored copy, stop further rollout if applicable and record observed recovery. Exercise bounded capacity refusal while preserving acknowledged state; do not delete history or enlarge limits to make the demonstration pass. Quiescent copy supports a quiescent-backup claim only, not live backup, power-loss durability or arbitrary cross-version migration.

**Outputs:** named handoff/escalation map, verified intake evidence from authorized drill, backup/restore/incident/rollback logs, exact data comparisons, measured recovery against objectives and retained uncertainty.

**Focused checks:** C11 store/transfer/authority regressions and selected real final-install restore. Negatives: capacity exhaustion, missing/mixed export chunks, changed backup byte, wrong owner/receipt, active-writer refusal, stale authority, unresolved operation, incompatible rollback and removal preserving user data. Same-UID and distinct-account boundaries remain explicit; local private modes are not hostile-owner isolation.

**Done:** an independent backup operator completes the supported recovery path within declared limits and responsibilities/intake are verified. **Blockers:** unnamed people, unavailable clean restore target, incomplete state dependencies, unapproved support drill or an unsupported backup/compatibility promise. A source fixture restore alone does not close the final campaign.

## W12 — Review acceptance and record the release decision

**State:** planned; acceptance and publication remain false. **Dependencies:** all selected mandatory W01–W11 results, conditional selected W13 interface/channel results and resolved D01–D07 decisions. **Roles:** `acceptance_reviewer`, `payload_reviewer`, `release_coordinator`, `release_approver` (@quantDIY), `rollback_primary`/backup.

**Existing owners:** `tests/launch-acceptance/{plan.json,acceptance-gates.go,README.md}`, `.github/RELEASE-GATES.md`, this dossier/mapping, exact candidate signatures/policy and owner evidence. Review source and any subsequent product changes before relying on the dated inventory.

**Tasks:** check every required input, all 36 core scenario/profile cells, 58 paper criteria and selected additional provider/engine/channel cells against actual artifact-bound observations. Retain original failures and invalidations. Verify reviewers examined numerical/endpoint/host/trust evidence, not only an attachment hash or a successful tool exit. Apply the explicitly selected score/applicability policy; under the unchanged paper proposal each category needs its own 8.5 result and mandatory gaps prevent acceptance. Do not average away a blocked category or mark an unperformed test passed.

Run the existing acceptance checker in a new evidence directory (C9), plus the approved versioned delivery companion after its W07 implementation, to cover every separately sealed artifact. Exit 2 means incomplete; exit 1 means invalid/tampered evidence or output conflict; exit 0 means structurally ready for independent review only. The v1 output always leaves `release_acceptance_complete=false` and `publication_authorized=false`; the companion must preserve that boundary and must not override v1 incompleteness. Do not change the existing schema or flags to manufacture authority. A named human release decision is separate evidence.

Record one clear outcome: accepted scope with all required proof; incomplete/no-go with concrete blockers; or an explicitly authorized bounded testing offer with exact unqualified limitations and user-led qualification still open. The third outcome is not full acceptance and must not satisfy pending core cells. Bind any decision to candidate digests, scope, policy, reviewer, time and compatible rollback. W12 does not automatically create a tag, GitHub Release, registry publication, mirror, support announcement or provider activation. Later publication and verification of delivered bytes require the applicable explicit authority.

**Outputs:** independent acceptance report, complete or explicitly incomplete packet, current criterion ratings only where supported, exact decision/rollback record and remaining work. Preserve all unresolved mandatory defects.

**Focused checks:** acceptance evidence regressions plus dossier consistency; reject wrong source/profile/artifact/policy, attachment drift, unsafe paths, missing or duplicate coverage, unknown fields, failed/fixture/skipped operational cells and reused output paths. Human review must also detect semantically false but well-formed observations.

**Done:** a concrete reviewable decision packet accurately states what is and is not accepted. Full acceptance requires all selected mandatory proof; publication requires its separate decision/action. **Blockers:** any unresolved mandatory implementation, qualification, trust, ownership, policy or evidence problem. Lack of a publication decision is not an excuse to fabricate one.

## Conditional W13 interface — separate MCP/client repository

W13 is outside this repository's implementation scope. If D06 selects an MCP/client channel, name its `channel_owner`, inspect the current separate `QuanuX/symphony-codex-ipc-mcp-and-skills` revision and agree a compatibility interface: exact beta qxctl/native catalogue, operation ownership/schema/help, package version and bytes, registry metadata, actual client behavior and distribution approval. Historical PR/version records are provenance, not current verification.

W13 build inputs must join W07 only if that channel is promised; final channel tests consume W07/W08 bytes. Require actual discovery/pagination/read/preview-apply/refusal/drift/lost-response behavior and real URL/digest replacements before metadata publication. Direct archives, MCPB/registry and an approved OpenAI directory are separate channel claims. No approval is inferred from equal version labels. This plan authorizes no edits, commits, uploads or messages in the separate repository or directory. An unavailable optional channel cannot erase a previously retained promise without D06.

## Existing command catalogue and execution boundaries

Commands below are **future execution recipes**, not a record that they ran. Paths in variables must be explicit absolute paths selected by the operator; output/work paths must be new, private and outside the source and installation. Use the exact pinned toolchain in the owner/CI recipe (currently Go 1.26.5; CMake 3.30+, with SBV requiring 3.31+; selected C++26 and Apple toolchains). Never use secret bytes in arguments/environment. Keep source-integration workspace tests distinct from standalone `GOWORK=off` distribution tests. Select the smallest meaningful command during an increment; run full selected source gates at the candidate milestone, not on every documentation edit.

### C1 — Dossier and source-map checks

From the repository root:

```sh
python3 docs/public-beta/validate.py
cmake -P tests/mvp-dependencies/verify.cmake
cmake -P tests/release-qualification/verify.cmake
```

`validate.py` checks this preparation dossier, not product readiness. Regenerate the exact-source inventory only when a new clean committed source is selected, using `prepare-review.cmake` and a new output outside the checkout; do not run it against the current dirty preparation tree or overwrite the recorded baseline.

### C2 — Installed SDK and external-engine preparation

```sh
sh tests/sbv-launch-boundary/run.sh "$installed" "$sbv_work"
sh tests/native-sdk-distribution/run.sh quantitative "$installed" "$sdk_work" "$sbv_work"
```

`knowledge` consumes the completed SNV campaign; `cloud` does not take that fourth argument. A narrowly focused `bootstrap` SDK run is available. These consume completed installed group outputs; they do not create final downloadable product bundles. The independent engine project is built outside the repository using its reviewed public-import recipe.

### C3 — Native Cloudflare/private-store regressions

Use the exact installed foundation/private-channel dependency configuration in `.github/workflows/cloudflare-native.yml`, then:

```sh
ctest --test-dir "$provider_build" --output-on-failure --output-junit "$evidence/provider.xml"
ctest --test-dir "$cloudflare_build" \
  -R '^(cloudflare-resource-execution|cloudflare-connector-lifecycle)$' \
  --output-on-failure --output-junit "$evidence/recovery.xml"
```

Expand to the complete owner suite for a changed adjacent contract or selected candidate gate. Include `cloudflare-access-sessions`, `cloudflare-native-contract`, `cloudflare-project-source` and projections where applicable. Installed qxctl tests need the documented exact prefix; a skipped installed test is not coverage. Privileged transport/root fixtures require their documented disposable host, not a personal host.

### C4 — Credential and Cloudflare authority/source checks

From `modules/secure-identity-access-governance`, with its selected source workspace and private caches:

```sh
go test -race -count=1 -json ./internal/credential
go test -race -count=1 -json ./internal/server ./internal/stavproducer \
  -run 'TestCloudflare|TestTransport|Test.*Candidate|Test.*Producer|Test.*Receipt'
```

For the actual Swift protocol/readiness scope on macOS, use `sh tests/release-qualification/swift-tests.sh "$swift_work"` from the repository root. For Cloudflare complete local evidence, follow `tests/cloudflare-release/README.md`, retain original JSON/JUnit and run its reducer with an exact selection file. Its local exit zero does not close LIVE-01.

### C5 — SRRAPIV and selected native providers

```sh
ctest --test-dir "$srrapiv_build" --output-on-failure --output-junit "$evidence/srrapiv.xml"
```

From `tools/qxctl`, with the exact installed-prefix variables from the owning tests/workflow:

```sh
GOWORK=off GOTOOLCHAIN=local go test -count=1 -json \
  ./internal/knowledgeengine ./cmd/qxctl -run TestSRRAPIV
```

SDK-free tests cannot qualify licensed SDK calls. Use owner `INSTALL.md`, CMake options and `tests/release-qualification/manifest.json` for the selected FRED/Databento/news/IBKR owner tests and copied consumers; do not invent an SDK-enabled configuration without its authorized material. Source-only runs must disclose installed-case skips.

### C6 — Actual installed payload and notice inventory

```sh
sh tests/dependency-notices/run.sh "$installed" "$notice_evidence" "$external_prefix" "$qxctl"
cmake -DWORK="$package_fixture_work" -P tests/package-release/run.cmake
```

The notice wrapper requires exactly four arguments; use an empty string for each unselected optional external/Go path, which records missing observations. It verifies the qxctl module cache while inventorying the supplied Go artifact. For each other selected Go command retain its own module verification/buildinfo and confirm the collector covers that command's exact linked modules before claiming notice closure. The collector does not resolve recursive rights/trust automatically. The package command builds synthetic probes and disposable signatures; it does not authorize or qualify the real candidate.

### C7 — Native candidate preparation and verification

For the admitted native receipt carrier, after approved finalization, sealed receipts and real platform/build evidence:

```sh
cmake -DRELEASE_STAGE="$stage" -DRELEASE_OUTPUT="$candidate" \
  -DRELEASE_REPOSITORY="$repository" \
  -DRELEASE_PLATFORM_EVIDENCE="$platform_evidence" \
  -DRELEASE_BUILD_EVIDENCE="$build_evidence" \
  -P cmake/release/PrepareCandidate.cmake
```

The source must be clean and committed; output must be new. Platform observation uses `sh tests/platform-qualification/capture.sh "$profile" "$platform_work"` on the actual selected host. The approved signing custodian may invoke `SignCandidate.cmake` with `RELEASE_CANDIDATE` and a private `RELEASE_SIGN_KEY` **path**, through the selected signing process. This command catalogue supplies no signing authorization. Selected Go/Swift artifacts use their approved owner recipes and independent trust bindings; do not feed arbitrary installer layouts into this native carrier or claim the command covered them.

```sh
cmake -DRELEASE_CANDIDATE="$candidate" \
  -DRELEASE_ALLOWED_SIGNERS="$allowed_signers" -DRELEASE_SIGNER="$signer" \
  -DRELEASE_EXPECTED_PROFILE="$profile" \
  -DRELEASE_VERIFICATION_OUTPUT="$evidence/downloaded-bytes.json" \
  -P cmake/release/VerifyCandidate.cmake
```

After separately verified extraction, repeat with `RELEASE_UNPACKED_ROOT` and a new verification output. Neither record alone proves clean-host execution, live providers or approval.

### C8 — Bounded final-byte extraction

```sh
GOWORK=off GOTOOLCHAIN=local go build -trimpath \
  -o "$tools/candidate-host" tests/launch-acceptance/candidate-host.go
"$tools/candidate-host" -repository "$verification_repository" \
  -candidate "$retrieved_candidate" -allowed-signers "$allowed_signers" \
  -signer "$signer" -profile "$profile" -work "$host_work"
```

Retain verification-tool identity separately. The helper performs no network retrieval and no downloaded-code execution. Follow W08 with actual clean-host scenarios.

### C9 — Complete core evidence packet

```sh
GOWORK=off GOTOOLCHAIN=local go build -trimpath \
  -o "$tools/acceptance-gates" tests/launch-acceptance/acceptance-gates.go
"$tools/acceptance-gates" -mode init \
  -plan "$repository/tests/launch-acceptance/plan.json" -packet "$packet"
"$tools/acceptance-gates" -mode check \
  -plan "$repository/tests/launch-acceptance/plan.json" \
  -packet "$packet" -report "$new_report"
```

Initialize once into a new private packet. Check subsequent observations into new reports. Pending inputs and all 36 cells initially produce exit 2; this expected incomplete state must not be converted into success. Required operational scenarios need digest-bound substantive observations with zero skips. Paper criteria and supplemental cells remain separately mapped and reviewed. After the approved W07 release-tooling implementation, run its separately versioned delivery companion against this packet, the signed index and all constituent artifact proofs. Its concrete CLI must be documented and tested with that implementation; no nonexistent command is presented here as runnable. A passing primary-native v1 packet alone cannot qualify the multi-artifact delivery set.

### C10 — Existing installed research/knowledge campaigns

```sh
sh tests/shv-release-qualification/run.sh "$installed" "$shv_work"
sh tests/snv-qualification/run.sh "$installed" "$snv_work" "$qxctl"
sh tests/sbv-launch-boundary/run.sh "$installed" "$sbv_work"
```

Follow each README's exact prerequisite data/decoder/toolchain/prefix contract. Existing runs from source-installed private prefixes remain development evidence. For final acceptance, adapt the invocation through a reviewed exact-version recipe to the retrieved artifacts and preserve provenance; do not copy an old evidence directory into a new packet.

### C11 — Focused operations/state checks

From `tools/qxctl`:

```sh
GOWORK=off GOTOOLCHAIN=local go test -count=1 -race -json \
  ./internal/snvstate ./internal/scvstate ./internal/scvcorpus \
  ./internal/shvstate ./internal/shvpublicationstate \
  ./internal/scvtransfer ./internal/shvtransfer ./internal/ssiagclient
```

Review helper/installed/owner-dependent skips by their documented scope. These fixtures prove their local boundary; W11's actual final-install backup/restore and authorized support drill remain separate.

## Requalification and change control

Retain an evidence dependency graph: source and owner versions → build/toolchain/dependency recipe → finalized files/receipts → candidate/signature/policy → host installation → scenario/criterion review. Record exactly which dependencies justify reuse. Every new candidate needs new byte/trust verification and affected installed/runtime checks; a new implementation or route invalidates its dependent behavior evidence. A documentation-only dossier commit does not retroactively rerun the product, and old green CI is not new evidence.

During increments run focused tests at changed owner boundaries. At G2 run the required selected source gates and at G3 the complete selected final-artifact matrix. Add tests for substantive defects or uncovered contract boundaries, not to manufacture activity for reversible prose edits. Any exception to frozen scope, required profiles, an existing promise or acceptance policy needs a recorded human decision before dependent work. Keep optional convenience commands/exporters/frameworks and new Cloudflare deployment/rotation features deferred.

This plan preserves the complete work inventory while allowing independent preparation to progress. It leaves execution authorization, product acceptance and publication visibly unfilled until their respective decisions and evidence exist.
