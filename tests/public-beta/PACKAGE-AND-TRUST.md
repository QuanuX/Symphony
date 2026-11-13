# Public Beta package and trust proposal

This is the W06 reviewable package proposal and the detailed W07 execution design, grounded in source `887e074780e1d0633bd6e6f57c3d4e07ca3925fc`. It does not approve a package selection, create a release, qualify a download, or authorize a build/signing/provider action. The public beta feature freeze remains in force. Real builds are a later, manually approved handoff after the requested model switch. [package-plan.json](package-plan.json) is the machine-readable companion; [source-inventory.json](source-inventory.json) preserves the clean-source inventory.

The proposal preserves the source's independent package and SDK versions. A release-set name must not overwrite the existing identities, public Go module tags, protocol versions or caller-selected owner versions. Source CI, local fixtures and a signed manifest are different evidence from a working final download. No artifact hash, runtime inventory, download URL, signer approval or successful qualification is invented here.

## Proposed delivery units

| Proposed ID | Contents and selection | Current boundary / decision |
|---|---|---|
| `PKG-CORE-macos15-arm64` | Delivery set for macOS 15 ARM64: 51 native receipt units derived from proposed core capabilities and their exact dependency closure, plus six Go commands; complete SDK exports stay with their owning native receipts. | Pending D01/D02 selection and actual final-host qualification. Other native units, the protected profile administrator and Apple metadata provider are separately selectable choices, not silent promise exclusions. |
| `PKG-CORE-ubuntu24.04-amd64` | Same source-defined units, built for Ubuntu 24.04 amd64. | Pending D01/D02 and target-specific closure/qualification. |
| `PKG-CORE-ubuntu24.04-arm64` | Same source-defined units, built for Ubuntu 24.04 ARM64. | Generic Ubuntu ARM64 is the existing required profile; this is not physical DGX Spark/GPU evidence. |
| `PKG-CORE-ubuntu26.04-amd64` | Same source-defined units, built for Ubuntu 26.04 amd64. | Pending D01/D02 and target-specific closure/qualification. |
| `PKG-NATIVE-<component_id>` | 59 independently versioned, installable native receipt units in the source map: 51 proposed core units plus eight separately selectable units. Actual installed contents come from each complete CMake install/receipt contract. | Preserve full runtime, SDK, metadata and notice ownership. Build targets are not an installed-file list. Actual file paths/hashes await staging. |
| `PKG-GO-<component_id>` | Seven exact commands listed below, with source/build identity, linked-module/runtime notices and each command's own installation contract. | Native candidate tooling is not a prequalified universal wrapper for these formats. Review and qualify the selected delivery/install recipe. |
| `PKG-SWIFT-MACOS-METADATA` | The macOS SSIAG provider, complete app bundle plus its receipt installer, version `0.1.0-draft`. | Metadata/trust/readiness only. Operational flags remain false; no general Keychain credential delivery. Candidate profile proposed: macOS 15 ARM64. |
| `PKG-CF-INTEL` | Separate macOS amd64 supplemental set: native foundation, private-store, Cloudflare adapter/Access helper, relevant Go commands, exact separate cf/Node toolchain closure, and explicitly chosen protected carrier/SSH/proxy closures. | Owner runtime admission is `darwin-amd64`; the exact OS/deployment/host profile is still D02 input. No Linux/ARM vendor execution is inferred. |
| `PKG-SOURCE` | Matching first-party source, build/install material and reviewed dependency source/provenance for the selected binaries. All 73 source roots remain inventoried. | Delivery method and corresponding-source completeness require publisher review. Proposal/support roots are not advertised as working native products. |

These are **delivery sets and receipt units**, not a claim that one `candidate.tar` already contains every language/runtime. The existing native packer seals an exact receipt-owned regular-file closure. A delivery index must bind the separately delivered files, their versions, source/notice material and approved verification/install recipes. That index is still packaging work, not an existing finished artifact.

D01 must resolve the proposed selections against retained promises; D03 must resolve implementation gaps. Calling a runtime “optional,” choosing a smaller archive, or shipping source alone cannot close a promise that requires a working operation. The 51/59 distinction is an explicit review choice, not approval to omit a retained capability. The core closure includes native-source-support, Databento DBN/reference and FRED receipts because selected core owners depend on them; their bytes do not automatically select optional provider operations. The eight other native units are IBKR Client Portal/TWS, both DuckDB connectors, PDF adapter, Linux node observer, News API and SRRAPIV. The observer is selectable only for the three Linux profiles; macOS compile/refusal evidence does not qualify Linux observation there. Six Go commands are proposed core; the protected profile administrator is separately selectable. SDK selection is an alias over the 24 SDK-exporting receipt units and their dependency closures, not a stripped headers-only repackaging. The quantOS remains open to user-authored compositions; these constraints describe these published package contracts only.

## Source authority and version inventory

The structured maps, rather than older count summaries, govern this proposal:

- [cmake/mvp-dependencies.json](../../cmake/mvp-dependencies.json): 73 roots, 60 native CMake roots, 59 installable native units, runtime DAG, installed-only switches, exported SDKs, selected dependency versions and source-input hashes.
- [cmake/mvp-release-identities.json](../../cmake/mvp-release-identities.json): component package/SDK/source identities and seven Go command recipes.
- [cmake/mvp-platform-profiles.json](../../cmake/mvp-platform-profiles.json): four declared core targets and their OS/ABI/toolchain scope; declaration is not qualification.
- Each component's `CMakeLists.txt`, `INSTALL.md`/`README.md`, owner declarations and receipt helpers own its actual installed files and behavior. `package-plan.json` names these content sources, exact installed options, dependency recipes and SDK exports per component.

The sole noninstallable native root is `tools/authoring-cpp`. `tools/shv-interface-codegen` is authoring source; `tools/sqv-administration-cpp` is shared build source. The two Go library roots are SDK/source modules, not additional command binaries. `modules/hotpath-runtime` remains proposal-only without a runtime/installer. Record that gap under D03; the freeze does not authorize constructing the proposed engine or treating it as delivered.

The command recipes are concrete:

| Recipe suffix | Command / proposed package version | Installation boundary |
|---|---|---|
| `qxctl` | `qxctl`, `0.1.0-dev` | Caller-selected CLI; owners remain separate exact selections. |
| `secure-identity-access-governance` | `symphony-ssiag`, `0.1.0-dev` | Existing `package install --prefix ABSOLUTE --version COMPILED_VERSION` seals immutable receipt-v2. Enrollment, supervision and provider binding are separate. |
| `stav-append-authority` | `symphony-stav-append-authority`, `0.2.1` | Existing scope installer creates versioned receipt-v2. Verify private staging mechanics before packaging; do not enroll/start a TOPS to create a binary artifact. |
| `accordare-stav-producer` | `symphony-accordare-stav-producer`, `0.1.0-dev` | Existing `install --prefix ABSOLUTE --version 0.1.0-dev`; grants, enrollment and supervision are separate. |
| `sov-stdio-adapter` | `sov-stdio-adapter`, `0.1.0-dev` | Explicit executable/digest and target configuration; no automatic service or provider activation. |
| `sov-ssh-adapter` | `sov-ssh-adapter`, `0.1.0-dev` | Ordinary SSH v1 selection and a protected-profile carrier wrapper are distinct contracts. |
| `sov-transport-profile-admin` | `sov-transport-profile-admin`, `0.1.0-dev` | Exact protected versioned executable, root-owned ancestry and reviewed host policy; packaging does not provision sudo/SSIAG/STAV scope or connect a user. |

SSIAG/STAV source literals `dev` and their selected compiled command identities are intentionally distinct. The existing candidate builder supplies the catalog's supported linker version symbols and verifies output. Do not make all modules share a beta version. D01 must approve any change to package identity before it is implemented and verified.

## W06 closeout: select before constructing

W06 is complete as a **proposal** when these two files and the inventory are reviewable. W06 is complete as a **release selection** only when D01/D02/D03/D04 decisions record the exact selected units, operations, profiles, versions, packaging/trust choices and unresolved release blockers with named owners. No approval is assumed by checking in this dossier.

| Input | Concrete decision/material required | What it unblocks |
|---|---|---|
| `INPUT-PKG-01` / D01 | Exact units and operations per profile, including independently visible exclusions/conflicts. | Selected package graph. |
| `INPUT-PKG-02` / D02 | Four core target decisions; exact Intel macOS supplemental OS/deployment/host profile. | Native host allocation and qualification matrix. |
| `INPUT-PKG-03` / D01 | Independent release identities; delivery-set naming policy. | Immutable stage names, side-by-side lifecycle cases. |
| `INPUT-PKG-04` / D01 | Shipped versus caller-supplied runtime choices and exact operation consequences. | Loader/runtime and notice closure. |
| `INPUT-PKG-05` / D03 | Retained-promise resolution for genuinely missing implementation, without inferring a scope removal or new-feature authorization. | Honest candidate capability claims. |
| `INPUT-PKG-06` / D01 | Go/Swift wrapper mechanics and complete delivery-index design. | Multi-artifact verification/install recipe. |
| `INPUT-PKG-07` / D01 | Matching source/build/install delivery and reviewed private/vendor exclusions. | Corresponding-source package. |
| `INPUT-PKG-08` / D05 | Named builder/reviewer, native hosts, exact toolchain/dependency inputs, manual execution approval. | W07 execution. |
| `INPUT-PKG-09` / D06 | Actual candidate/release destinations and immutable withdrawal/replacement policy. | Download qualification later; no URL is filled now. |
| `INPUT-TRUST-01` / D04 | Publisher signer identity/fingerprint and independently distributed `allowed_signers` policy. | Authentic publisher signing and verification. |
| `INPUT-TRUST-02` / D04 | Applicable Apple signing/team/bundle/requirement/entitlement/notarization policy. | Actual macOS trust qualification. |
| `INPUT-TRUST-03..05` / D04 | Selected vendor authorization, matching source review, every runtime/TLS/notice dependency. | Publisher approval of exact distributed bytes. |
| `INPUT-TRUST-06` / D05 | Named trust reviewer, release approver and rollback/security owner. | Accountability and recorded release decision. |

The signer policy must be independently trusted before download. A public key arriving only beside the archive is not an independent trust decision. Decisions and values stay in the owner/decision records; private signing keys, credentials, paid data and proprietary SDK bytes do not belong in this public dossier.

## W07 build and seal after approval

Execute in bounded increments, preserving failed work and exact evidence. The current task creates the plan only. Each increment has a concrete stop condition; no missing input becomes a fabricated pass. A successful first profile may enable useful follow-on work, but does not qualify the remaining required profiles.

| Increment | Work and retained output | Exit / failure condition |
|---|---|---|
| W07.1 — source and host | Select clean committed source and approved profile; record commit/tree, map hashes, toolchains, host image/architecture/ABI/filesystem and input provenance. Verify map and exact selected graph. | All records bind the same actual source/profile. Dirty source, unresolved version/profile or missing required inputs stop this profile. |
| W07.2 — dependency and build proof | Obtain approved pinned external inputs; run the selected installed-dependency groups in fresh qualification work and the exact Go identity recipes. Retain full command logs, imported-SDK proof, receipts, identities, selected consumer checks and any skips. | Selected graph builds and affected boundary checks pass without required skips. A private absolute-RPATH qualification prefix is not the final stage. |
| W07.3 — final stage | Build a new packaging graph with reviewed loader/runtime closure and finalizer. Preserve complete units, SDK exports, all notices and matching source material. Finalize bytes before sealing receipt ownership. | Inventory contains exactly selected regular files; every runtime dependency is resolved; no source/build paths or ambient developer cache needed. Interruptions leave no accepted sealed artifact; retry in a fresh stage. |
| W07.4 — complete delivery | Package native closure with existing helper; build and validate reviewed Go/Swift wrappers and delivery index. Keep separately prepared external runtimes separate until exact admitted closure is reviewed. | Every promised selected unit has a concrete artifact, install/removal recipe and provenance. Missing wrapper/index is a packaging gap, not finished distribution. |
| W07.5 — trust and private candidate | Named reviewers approve payload/notices/source and signer policy; use actual approved signing identities. Sign exact final inventory/archive bindings. | Native signature, platform policy and independent publisher verification pass as applicable. No disposable CI key or ad-hoc signature substitutes for approved identity. |
| W07.6 — handoff to W08/W09 | Retain immutable candidate artifacts, source/notices, signatures, all inventories, build evidence, reviewer decisions and pending download/live inputs. | Candidate is ready for separately authorized final-download and fresh-host testing. No claim of release acceptance, live acceptance or public availability follows from sealing. |

If the build selects a later source commit after documentation merges or stabilization, regenerate the inventory and bind all new evidence to that actual commit/tree. The dossier's `887e074...` review baseline must never be stamped onto bytes from a different checkout.

### Native build and final stage

[tests/mvp-dependencies/README.md](../../tests/mvp-dependencies/README.md) specifies the existing isolated graph driver. It builds in mapped dependency order, applies each component's installed-only switches, disables package registries, checks exact SDK origins and versions, installs complete package targets, and runs installed consumers. Groups are `bootstrap`, `knowledge`, `cloud`, `quantitative`, `duckdb`, `build_support`. The quantitative group requires exact curl 8.7.1/zstd 1.5.7 development inputs; DuckDB requires its exact dependency configuration. `build_support` creates no runtime package.

Later execution examples, from the approved clean source, with new absolute work directories:

```sh
cmake -P tests/mvp-dependencies/verify.cmake
CXX=clang++ sh tests/mvp-dependencies/run.sh knowledge /absolute/new-knowledge-work
CXX=clang++ sh tests/mvp-dependencies/run.sh quantitative /absolute/new-quantitative-work \
  -DCMAKE_PREFIX_PATH=/absolute/approved-external-prefix
```

These examples establish graph qualification only. The driver's installed libraries deliberately retain absolute private dependency RPATHs. Copying that prefix into an archive does not make it relocatable. For the final stage, use the map's exact native DAG and installed switches with separate fresh build directories and a reviewed loader layout. Include complete runtime dependency units; do not copy arbitrary build outputs or omit SDK headers/exports from a sealed receipt.

The existing [PACKAGE-HANDOFF](../../cmake/PACKAGE-HANDOFF.md) foundation example provides the finalization contract:

```sh
cmake -S libraries/knowledge-vector-engine-cpp -B /absolute/new-package-build \
  -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/absolute/new-private-stage \
  -DSYMPHONY_PACKAGE_FINALIZER=/absolute/reviewed-source/cmake/release/FinalizeNativePayload.cmake
cmake --build /absolute/new-package-build
cmake -DSYMPHONY_PACKAGE_STAGE_ROOT=/absolute/new-private-stage \
  -P /absolute/new-package-build/cmake_install.cmake
```

For dependents, add their exact map `installed_options` and approved `CMAKE_PREFIX_PATH`, plus reviewed platform loader options; record every effective argument. The finalizer strips eligible executables/shared libraries and can codesign on macOS with an explicitly selected identity. It does not solve arbitrary runtime relocation or perform notarization. Changes to loader references, signing, stapling or any other receipt-owned bytes must finish **before** final receipt hashing. Review finalizer source and its configured hash. Do not reseal an old receipt after editing its files.

On the final artifacts, run copied SDK consumers outside the source tree, using [native-sdk-distribution](../../tests/native-sdk-distribution/README.md) for the selected complete receipt units and its required upstream campaign evidence. Preserve exact SDK/C ABI differences; a C++ export does not promise cross-compiler ABI compatibility. Exercise relocatability, no-build-path execution, side-by-side installation, changed-file refusal, interrupted removal/retry and unrelated-data preservation. The existing package-release fixture demonstrates mechanics; its tiny probes are not this product qualification.

### Go commands and Apple metadata provider

The existing exact-source Go recipes are:

```sh
sh tools/qxctl/tests/reproducible_build.sh /absolute/new-qxctl-work
sh .github/scripts/build-mvp-go-identities.sh /absolute/new-go-candidate-work
```

They require exact Go 1.26.5, a clean committed source checkout and fresh work outside it. qxctl has two isolated builds and equality evidence. The second recipe builds six other commands, once each, from the seven-command identity catalog; it explicitly records reproducibility as `not_assessed`. Retain binary buildinfo and identity sidecars. Any final stripping/signing changes require new final-byte hashes; earlier build hashes remain provenance, not final artifact identities.

Use each Go command's existing install contract. The source includes receipt-v2 installers for some commands and explicitly selected executables for others. Test the chosen private staging and immutable wrapper rather than assuming the native CMake packer or an ordinary source build has supplied the complete Go distribution contract. No packaging operation should implicitly enroll a TOPS, start services, write grants or select credentials.

For the Apple provider, bare `swift build -c release` is a development metadata artifact. The actual [production bundle script](../../modules/ssiag-provider-macos-keychain/scripts/build-production-bundle.sh) requires `--output`, `--bundle-id`, `--signing-identity`, `--policy` and `--notary-profile`, with optional entitlements/provisioning inputs. It runs real codesigning, notary submission and stapling. Plan this only after approved identities and explicit execution authorization; then use the existing bundle-aware installer to create receipt ownership of the finished bundle. Readiness remains metadata-only regardless of successful platform signing.

### Cloudflare Intel supplement

The [adapter installation contract](../../modules/sov-cloudflare-adapter/INSTALL.md) installs two runtime entrypoints: `symphony-sov-cloudflare-adapter` and `symphony-cloudflare-access`, plus owner declarations, schemas/templates and docs/notices. `cloudflare-generate` and `cloudflare-prepare_bundle` are build-tree tools. The foundation and `SymphonySSIAGPrivateStore 0.1.0 EXACT` must be installed before the adapter's installed-only build.

The adapter does not download Node, install npm packages, run dependency lifecycle scripts, create credentials or activate a protected profile. Its cf/Node bundle is separate. The existing preparer consumes an already verified official Node distribution and an exact npm tree prepared with scripts disabled:

```sh
/absolute/approved-adapter-build/cloudflare-prepare_bundle \
  --node /absolute/verified-node --cf /absolute/script-disabled-cf \
  --output /absolute/new-bundle
```

Review complete transitive files/licenses and source provenance. The selected versions remain cf `1.0.0-beta.28`, Node `26.12.0`, cloudflared `2026.10.7`; source-declared expected digests are checks against later supplied bytes, not observations that a candidate exists. Only owner-admitted bundles acquire execution eligibility. The core four profiles do not gain Intel vendor admission through successful native fixture builds.

For an approved Access payload, the CMake inputs `SYMPHONY_ACCESS_CLOUDFLARED` and `SYMPHONY_ACCESS_RUNTIME_CONFIG` must both be absolute. They create the receipt-owned `cloudflared` and `access-runtime.json` siblings. The runtime config binds exact cloudflared, carrier and OpenSSH digests. None can be filled until real selected files exist. Preserve separate carrier/SSH/proxy receipt and package-manifest roles with a qualification file, as required by [protected installation](../../tools/sov-transport-profile-admin/INSTALL.md). Actual protected root ancestry, SSIAG/STAV grants, administrator policy and ordinary-user connection behavior are installed-host qualification, not consequences of copying binaries.

The implemented private-store route is opt-in and Cloudflare-specific; it does not close general production credential delivery. Current rotation and I7 execution limitations remain explicit in the capability matrix. Intel packaging does not turn an inspect/plan operation into prepare/build/deploy/status/reconcile/migrate. Live Cloudflare/Access acceptance remains user-led after packaging.

### Runtime, notices and corresponding source

The native recipe `external_dependency_refs` mirror the map's named direct/declared consumers and inherit through the exact dependency recipes. In particular, `native-source-support-cpp` and `sqav-databento-dbn-cpp` directly require CURL 8.7.1. The per-recipe `runtime_inputs` are review reminders, not an exhaustive runtime inventory; final linkage and OS/transitive dependencies still need observation.

The [publisher review checklist](../../cmake/notices/PUBLISHER-REVIEW.md) is preparation, not legal or redistribution approval. For each actual artifact classify embedded/static material, shipped dynamic libraries, system dependencies, optional caller-supplied runtime and test-only inputs. Capture recursive loader dependencies; direct `libcurl` detection is insufficient. Resolve TLS/backend/ABI/version provenance and exact notice/source material. Never repair a fresh-host failure by silently drawing from a developer cache.

| Material | Exact planning boundary |
|---|---|
| curl 8.7.1 / zstd 1.5.7 | Preserve the selected curl `COPYING` and zstd BSD `LICENSE`; inspect actual shared curl/TLS and static zstd consumer closure. macOS's system framework remains a system dependency unless separately approved for redistribution. |
| DuckDB 1.5.5 | Preserve connector provenance, exact platform runtime and both MIT notice and full 20-entry third-party appendix. Existing frozen embedded SDK identities are intentional; do not normalize them to the newest SDK. |
| PDFium | No runtime version selected here. Dependency-map historical example `153.0.7999.0` and SHV release qualification `158.0.8125.0` are different source records, not a universal pin. If shipped, explicitly select the exact tested binary/provenance and retain `LICENSE`, `licenses/*`, `VERSION`, `args.gn` and matching source/build record. |
| nlohmann/json 3.12.0 | Retain vendored MIT notice even when the payload contains only statically linked consumers. |
| Go 1.26.5 and modules | Retain exact runtime/toolchain LICENSE/PATENTS/vendor material and actual linked-module notices/buildinfo. Resolved test/tool modules are not automatically linked payload. |
| Rithmic R API+ 13.7.0.0 | Normal shared qualification disables it. SDK-enabled build is an explicit variant needing authorized headers/platform archive closure, source/redistribution review and actual qualification. SDK-free package cannot substantiate a working vendor bridge. The supplied SDK-linked source choices cover macOS Intel/ARM and Linux x86_64 lanes; no supplied SDK-linked Linux ARM64 lane is established. |
| IBKR TWS SDK 10.45 | Separately authorized external object-conformance/host input, not a dependency silently bundled by the normal adapter build. |
| cf/Node/cloudflared, OpenSSH and carrier | Exact selected upstream and transitive closure; distinct owner admission, protected-package qualification and publisher redistribution/source decisions. |
| User provider/feedback plugins and data | Caller-selected inputs and licensed/private data stay outside a default publisher archive; inclusion needs explicit reviewed scope. Public fixture licensing is not a market-data entitlement. |

The existing collector accepts exactly four arguments, including optional positions:

```sh
sh tests/dependency-notices/run.sh /absolute/final-native-stage \
  /absolute/new-notice-evidence /absolute/approved-external-prefix \
  /absolute/selected-go-command
```

Use an empty string for an unavailable optional argument and keep the resulting missing observation open. It does not download native dependencies or qualify a provider. Its wrapper verifies the qxctl module cache; retain each other command's own exact module verification and inspect the collector's actual buildinfo-derived linked notices before approving that command. Run with the exact selected Go cache/toolchain and new evidence directories. Collector `release_ready=false`, `publication_authorized=false` and publisher-pending status are deliberate.

Produce matching source/build/install material for the actual selected source commit and distributed dependencies. A source archive needs a reviewed file manifest and provenance; a generic link to moving main is insufficient. The build map or license collector does not settle corresponding-source completeness or redistribution rights. The publisher signs off the actual selected closure, not this list.

### Native candidate and trust sequence

Once a complete final native stage and real source/profile/build observations exist, the existing packer can create its candidate:

```sh
cmake -DRELEASE_STAGE=/absolute/final-native-stage \
  -DRELEASE_OUTPUT=/absolute/new-native-candidate \
  -DRELEASE_REPOSITORY=/absolute/approved-clean-source \
  -DRELEASE_PLATFORM_EVIDENCE=/absolute/evidence/platform.json \
  -DRELEASE_BUILD_EVIDENCE=/absolute/evidence/build.json \
  -P cmake/release/PrepareCandidate.cmake
```

The platform observation must use `symphony.platform-observation.v1`, `qualification_status=evidence_only`, actual source commit/tree and exact profile/OS/architecture. Build evidence must bind the same commit/tree/profile and retain the actual recipe/log/input observations. Merely filling those fields does not attest a build. The packer rejects unowned files, mixed/incorrect receipt targets, drift during archive preparation and dirty source. It emits `candidate.tar`, `candidate.json`, platform/build evidence, dependency/identity maps and checksums, with publication still false. Native archive creation does not establish cross-host byte reproducibility.

After payload review, use the publisher's existing approved key; do not generate a disposable key and relabel it production:

```sh
cmake -DRELEASE_CANDIDATE=/absolute/new-native-candidate \
  -DRELEASE_SIGN_KEY=/absolute/private-approved-publisher-key \
  -P cmake/release/SignCandidate.cmake
```

This uses SSHSIG namespace `symphony-release` over exact `candidate.json`, whose contents bind the archive and evidence digests. Signer identity/fingerprint and `allowed_signers` must come through the independent approved policy. Apple Developer ID/notarization/stapling/Gatekeeper evidence is separate, as applicable to the approved macOS payload. No payload changes follow receipt sealing; no archive/inventory changes follow its final signing without creating and reviewing a new candidate identity.

Later W08 retrieval records the real original/final URL, time, size and digest. The existing verifier first checks downloaded signed bytes, then the reviewed `candidate-host` helper performs bounded safe extraction and receipt/file verification. Its output deliberately says `verified_extraction_only`, with fresh-host, provider, release-acceptance and publication flags false. Run the selected actual lifecycle/native/SDK scenarios afterwards on fresh hosts, outside the source checkout, preserving unrelated data. The current 16 GiB archive/payload, 16 MiB inventory and 65,536-file ceilings are real packaging limits; measure this payload before selecting the format.

Revocation/rollback planning identifies exact affected digests, stops distribution through the chosen channel, independently updates trust/revocation policy, preserves evidence and issues a new immutable replacement identity. Never overwrite a signed candidate in place or infer that uninstalling an administrator/helper revokes existing sessions or resolves uncertain operations.

## Remaining gaps and evidence handoff

The bounded next increment is **W06 decision review, then W07.1–W07.3 on an explicitly approved native profile**, while assembling the other target inputs in parallel. Its concrete output is a real selected stage and build/runtime/source/notice evidence; later sealing, download qualification and live acceptance remain separately visible.

Missing inputs include approved selections, actual native hosts/dependencies, signer/platform identity, redistribution/source decisions and real distribution destinations. Missing packaging implementation/qualification includes exact relocatable production stages, any necessary Go/Swift wrappers and delivery index, full artifact runtime/notice closure, and complete final-host/lifecycle checks. Missing product implementation remains in the capability matrix and D03, including general credential operations and explicitly unavailable provider/engine paths. None of those classes can substitute for another.

Return to the implementation plan with: exact candidate/source identities; final per-file inventories and hashes; package/SDK/Go/runtime identities; build and consumer evidence; independent trust decisions; notice/source review; per-profile qualification and skips; unresolved capability promises; and actual retrieval destinations only when assigned. W07 can proceed without live provider profiles. The later user-led provider results must still bind the exact final artifact/profile and cannot be replaced by source CI or local synthetic evidence.
