# QuanuX Symphony

**A modular workbench for quantitative research and system administration.**

Symphony brings quantitative data, replay, hardware and infrastructure knowledge, and explicit administrative control into one inspectable platform. Its native engines and SDKs do the domain work; `qxctl` gives people and agents a common way to inspect, validate, and administer separately installed components.

Build the environment your work needs. Keep your strategy logic, provider choices, hardware, and deployment topology under your control.

[Start from source](.github/FIRST-RUN.md) · [Explore the components](#architecture) · [Platform targets](#platform-targets) · [Release gates](.github/RELEASE-GATES.md) · [Report a problem](#support-and-contributing)

> **Development status:** Symphony has working native libraries, engines, administrative services, and cross-platform qualification. It is preparing for its first MVP binary distribution. Final downloadable packages are being prepared; users will perform live-provider qualification after packaging. Native credential/provider implementation limits remain explicit below. The tables below distinguish working source capabilities from the remaining release work.

## Identity

QuanuX is the ecosystem and steward of official source truth. Symphony is its open-source platform. A user’s private Symphony/TOPS installation retains its own declared and observed state; that state does not become official QuanuX source truth.

## What you can do today

| Workflow | Available in the repository | Where to start |
| --- | --- | --- |
| Inspect and administer an installation | Query contracts, versions, command identities, installed engines, lifecycle state, and protected selections through a Go CLI with structured output. | [qxctl](tools/qxctl/README.md) |
| Work with quantitative data | Use C++ SDKs for capture, metadata, flow, integer conversion, retained stores, replay and checkpoints; inspect retained state through installed owner engines. | [SQV contracts](knowledge/sqv/) · [Installed dependency recipe](tests/mvp-dependencies/README.md) |
| Develop a dedicated Rithmic integration | Inspect exact SDK/adapter bindings and maintenance impacts; build the non-Windows native SRRAPIV runtime with an explicitly supplied SDK. Native mapping, runtime tests and live vendor qualification remain separate evidence. | [SRRAPIV contracts](knowledge/sqv/srrapiv/SPEC.md) · [Native runtime](modules/srrapiv-runtime-cpp/INSTALL.md) |
| Run bounded backtest research | Build a signal census, run native evaluations, retain results, and exercise cancellation and recovery through SBV and the optional Go adapter. | [Installed SBV scenario](tests/sbv-launch-boundary/README.md) · [Go adapter](libraries/sbv-go/README.md) |
| Build traceable hardware knowledge | Retain source captures, replay a selected catalogue, evaluate requirements, query a graph, and preview publication with explicit provenance. | [Hardware starter](examples/mvp-hardware/README.md) |
| Inspect cloud and Node evidence | Use cloud/provider source-knowledge engines and five Node engines for supplied identity, resources, cluster relationships, names, and composition. | [SCV source knowledge](knowledge/scv/SOURCE-KNOWLEDGE.md) · [SNV qualification](knowledge/snv/VALIDATION.md) |
| Add relational graph indexing | Use the optional native DuckDB connectors for retained SCV and SHV graphs. Both connectors have installed qualification on all four MVP targets. | [Connector workflow](.github/workflows/duckdb-connectors.yml) · [Exact dependency map](cmake/mvp-dependencies.json) |
| Govern local administration | Apply exact authorization policy, maintain append-only audit evidence, inspect lifecycle plans, and recover bounded administrative operations. | [SSIAG](modules/secure-identity-access-governance/README.md) · [STAV](modules/stav-append-authority/) · [Operator runbooks](tests/launch-operations/README.md) |

The SBV launch scenario uses deliberately synthetic market events, a small CPU workload, and same-host process transport. It verifies that path, including uncertainty and recovery; it does not establish strategy performance, live trading readiness, or distributed performance. Cloud knowledge does not provision a cloud account, and a hardware catalogue does not install a driver.

## Platform targets

The required MVP profiles are:

| Target | Architecture | Current qualification boundary |
| --- | --- | --- |
| macOS 15 | Apple Silicon / ARM64 | Native installed SDKs, Go modules, retained inspection, package lifecycle, and DuckDB connectors; Apple provider contract tests remain distinct from operational Keychain access. |
| Ubuntu 24.04 LTS | x86_64 / amd64 | Native installed SDKs, Go modules, retained inspection, package lifecycle, and DuckDB connectors. |
| Ubuntu 24.04 LTS | ARM64 / aarch64 | The same selected Linux qualification; accepted as the ARM Linux target for this MVP sprint. |
| Ubuntu 26.04 LTS | x86_64 / amd64 | Native installed SDKs, Go modules, retained inspection, package lifecycle, and DuckDB connectors. |

[PR #184](https://github.com/QuanuX/Symphony/pull/184) completed all eight workflows for its exact source revision, including the four-target installed and DuckDB matrices. This is source/build qualification. Every final downloadable candidate still needs fresh-host acceptance on its selected profile.

**DGX Spark:** Ubuntu ARM64 is the current qualification substitute. Physical DGX Spark, its DGX OS image, NVIDIA drivers, and GPU behavior are not yet qualified. Intel macOS checks provide supplemental development evidence. Native Windows is unsupported, and no native Windows port is planned; WSL and remote Linux access are separate Linux environments.

## Get started from source

Start with the CLI to explore the repository without enrolling a service or contacting a provider. Use a reviewed source revision and the exact Go toolchain declared in [`tools/qxctl/go.mod`](tools/qxctl/go.mod), currently **Go 1.26.5**.

From the repository root:

```sh
repo=$(pwd -P)
umask 077
work=$(mktemp -d "${TMPDIR:-/tmp}/symphony-first-run.XXXXXX")
(cd tools/qxctl && GOWORK=off GOTOOLCHAIN=local GOENV=off \
  GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org \
  go build -mod=readonly -trimpath -o "$work/qxctl" ./cmd/qxctl)
"$work/qxctl" version --json
sh tests/launch-acceptance/source-smoke.sh "$work/qxctl" "$repo" "$work/evidence"
```

This builds a development CLI, reports its executable identity, reads the invariant registry, and checks refusal of a missing repository. It does not establish repository semantic validity; that requires the separately installed validator. An ordinary local build reports `0.1.0-dev` and unavailable source-stamp fields. A release build must bind its exact source, dependency map, and build recipe separately.

Next, choose the path you need:

- **Native foundations:** use the [`bootstrap` installed recipe](tests/mvp-dependencies/README.md), then the selected module's `INSTALL.md`.
- **Quantitative research:** build the `quantitative` dependency group and run the [bounded SBV scenario](tests/sbv-launch-boundary/README.md).
- **Hardware knowledge:** follow the [installed hardware starter](examples/mvp-hardware/README.md), which reuses the pinned documentary-source campaign.
- **Operating an installation:** read the [source first-run and removal guide](.github/FIRST-RUN.md) and [retention and authorization runbooks](tests/launch-operations/README.md) before changing protected state.

Full native qualification uses CMake **3.31+**, a profile-selected C++26 compiler and platform SDK, and exact external dependencies. The installed CI recipe selects Xcode 26.3’s Apple compiler on macOS, Clang 18 on Ubuntu 24.04, and Clang 21 on Ubuntu 26.04; exact compiler/runtime versions are recorded per run. Individual projects may have lower CMake minimums. Go, native package, SDK, schema, and owner-interface versions are independent; the [dependency map](cmake/mvp-dependencies.json) records their exact relationships.

Modules install independently into explicit prefixes with owned-file receipts and guarded removal. Keep old and new versions side by side until compatibility is verified. Installation does not activate a provider or authorize an operation. Anaconda packages and download mirrors are planned distribution work; no production channel or installation command is claimed here yet.

## Provider and credential status

Provider integration has several layers: a parser or SDK, a bounded session/acquisition path, credential authority, an actual provider connection, and operational acceptance. A passing fixture test proves only the layer it exercises.

| Integration | Implemented source capability | Required before an operational launch promise |
| --- | --- | --- |
| **SSIAG credentials** | Internal one-shot guard, policy-bound dispatch, durable exact-owner admission snapshots, intent/outcome journal, STAV commitment and reconciliation. | Permission-backed lease lifecycle/audit wiring, native recipient/provider admission and protected delivery, and the actual native consumer bridge. No operational secret-delivery route is enabled. |
| **Apple Keychain** | Independently installed Swift metadata adapter, native peer/code checks, signed-bundle and session-readiness observations. | Data-protection Keychain item/access-control operations, deployment identity and login-context qualification, and protected credential use. Metadata readiness does not open the Keychain. |
| **Linux credentials** | SSIAG's portable authorization and journal foundations. | The separately identified Linux Secret Service provider and native delivery integration. Headless identities need their own explicit profile. |
| **Databento** | DBN decoding and replay, reference/quote support, and quote/request/attempt-bound historical acquisition composition. | Actual SSIAG bridge, authorized account and entitlements, bounded request/spend approval, and real success/refusal/recovery evidence. |
| **FRED / ALFRED** | Bounded native request and response handling with series/vintage/page identity and caller-supplied credential-use integration. | Operational credential bridge and authorized installed-provider qualification, including partial pages and refusal paths. |
| **IBKR Client Portal** | Read-only session admission, authenticated-status/account checks, bounded data reads and expiry/refusal handling through an explicit Gateway transport. | An externally authenticated Gateway and authorized profile; actual installed session and transport qualification. No login automation or order entry is implied. |
| **IBKR TWS** | Serialized host-session coordinator and read-only callback collection with generation/request binding, pacing, cancellation and disconnect handling. | Authorized SDK 10.45 provenance, concrete vendor socket/EReader/EWrapper/Decimal bridge and actual TWS/Gateway host qualification. Fixture sessions are not vendor SDK conformance. |
| **News** | Vendor-neutral article ingress, immutable revision/correction binding and capture. | An authorized caller/content source and installed retention qualification. No news vendor, scraper, feed subscription or fetching of article URLs is selected. |

The [credential integration handoff](modules/secure-identity-access-governance/CREDENTIAL-QUALIFICATION.md) identifies remaining **implementation**, deployment and qualification work. Missing native implementation must be completed before its operations are advertised as supported. Users will qualify supported live-provider paths after packaging; unavailable live test profiles do not block package preparation. Credentials and private provider payloads must remain outside command arguments, logs, public issues, and qualification artifacts.

## Architecture

Symphony is a monorepo for inspectability, with independently versioned and installable runtime components. A user can select, update, diagnose and remove engines individually. Shared native mechanics do not make the system a single resident process.

| Area | Components and responsibility |
| --- | --- |
| Administrative interface | [qxctl](tools/qxctl/) owns CLI grammar and presentation. It invokes exact installed owners and preserves their response and authorization boundaries. |
| Knowledge foundation | [C++ foundation](libraries/knowledge-vector-engine-cpp/) provides bounded framing, JSON, hashes, manifests and receipt mechanics. [Validator](tools/symphony-validator/) performs deterministic read-only checks. |
| Coordination | [Knowledge Session Coordinator](modules/knowledge-session-coordinator/) manages bounded sessions, compatibility and lifecycle plans/journals. [Maestro](modules/maestro/) records authenticated engine presence; it does not schedule or execute engines. |
| Identity and audit | [SSIAG](modules/secure-identity-access-governance/) owns authentication/authorization and provider bindings. [STAV](modules/stav-append-authority/) owns append authority; [Go protocol](libraries/stav-protocol-go/) and [Accordare producer](modules/accordare-stav-producer/) retain separate roles. |
| Knowledge governance | [SKVI](modules/skvi-engine/) indexes declared surfaces; [SCLV](modules/sclv-engine/) checks change history; [SACV](modules/sacv-engine/) checks API contracts; [SODV](modules/sodv-engine/) checks release-publication evidence; [SSFV](modules/ssfv-engine/) checks feature truth. |
| Composition and evolution | [SAV / Accordare](modules/sav-engine/) resolves and evaluates explicit composition; [SEV](modules/sev-engine/) prepares evolution plans and evidence. Neither silently applies canonical changes. |
| Cloud, hardware and Nodes | [SCV](knowledge/scv/), [SHV](knowledge/shv/) and [SNV](knowledge/snv/) own their respective source evidence, interpretation and supplied state. Optional DuckDB connectors add retained relational graph indexes. |
| Quantitative work | [SQV](knowledge/sqv/) includes capture, metadata, flow, conversion, retention, replay and source adapters. [SBV](modules/sbv-engine/) owns native backtest research, with an optional [Go client](libraries/sbv-go/). |
| Connection classes | SQV's peer owners are [SOOV](knowledge/sqv/soov/) for FIX, [SCABV](knowledge/sqv/scabv/) for component-scoped broker/API bindings, and sovereign [SRRAPIV](knowledge/sqv/srrapiv/) for dedicated R API+ C++ integration. Exact module contracts delimit implementation and qualification. |
| Operations | [SOV](knowledge/sov/) provides bounded local handoff and an optional scoped SSH carrier. These mechanisms do not establish general cloud provisioning or cross-host deployment acceptance. |

The machine-readable [component/version map](cmake/mvp-dependencies.json) and [functional qualification manifest](tests/release-qualification/manifest.json) enumerate current roots and test selections. A component in that inventory is not automatically selected for the first binary release. Each module's Contract Quad (`INTENT.md`, `MANIFEST.md`, `SKILL.md`, `SPEC.md`) and `INSTALL.md` define its actual boundary.

## Doctrine

- **Explicit ownership:** domain vectors own meaning; qxctl presents their operations. Derived views retain provenance to their source owners.
- **Evidence before action:** exact versions, receipt identities, expected state, bounded requests, and explicit authority constrain supported mutations. An uncertain outcome is reconciled rather than silently retried.
- **Independent deployment:** modules retain their own configuration, state, installation, update and removal boundaries. There is no implicit newest-version selection.
- **User-controlled composition:** strategy logic, provider selection, bus topology and supported adapters remain the user's choices. Human or agent identity alone grants no authority.
- **A protected hot path:** administration and knowledge work are cold or freezing-path capabilities. qxctl is not a mandatory resident trading-loop dependency. Explicitly installed SSIAG/STAV services retain their documented supervision roles.

A **Node** identifies physical computer resources. A **Habitat** is the exact OS/package conditioning selected for a Node. A **Nest** serves the user's chosen purpose; it is not constrained to one strategy, file, process or broker interface. Multiple buses may coexist, and their choice remains explicit. See the [architecture](knowledge/ARCHITECTURE.md), [terminology](knowledge/SLANG.md), [identity namespaces](knowledge/NAMESPACES.md), and [platform governance](knowledge/platform/INTENT.md) for precise definitions.

## Emerging Vector Architecture

SIV (Symphony Intelligence Vector) remains an emerging domain. Its optional SMCV Markdown mediation component has no implemented translator or command; SAIV is reserved. **SAV continues to mean Symphony Accordare Vector.** Prima Parte is a ratified optional Node-local witness concept, not an executable. `hotpath-runtime` is a proposal-only contract seed. The retired `node-troll` and `bus-troll` names are reserved tombstones.

General provider provisioning, Terraform apply, Habitat construction, Nest delivery, bus installation and canonical knowledge apply remain outside the implemented runtime promises. The Linux systemd boot receptor performs report-only lifecycle planning; it does not apply the plan. SACV currently supports bounded OpenAPI 3.2.0 JSON with internal references; YAML and external references return explicit unsupported responses. See its [shipped example](modules/sacv-engine/tests/fixtures/README.md).

## Root-Level Governance Role

The [platform governance contract](knowledge/platform/INTENT.md) defines shared invariants and modular sovereignty within the knowledge architecture. It does not absorb a vector’s meaning or a module’s runtime authority. Current contracts describe current truth; immutable change and publication records retain historical evidence. Phases describe delivery chronology, while vectors remain the enduring domain owners.

## Verification and release readiness

Symphony's checks cover distinct questions:

| Evidence | What it establishes |
| --- | --- |
| [Dependency map verification](tests/mvp-dependencies/README.md) | Source/version consistency, explicit dependency ordering and installed SDK consumption. |
| [Functional qualification](tests/release-qualification/README.md) | Selected native tests, Go race suites, Swift contracts, generated interfaces and installed scenarios on named profiles. |
| [Package qualification](tests/package-release/README.md) | Candidate verification and lifecycle mechanics using controlled fixtures. |
| [SDK distribution and removal](tests/native-sdk-distribution/README.md) | Receipt-owned headers/exports, installed consumers, refusal and resumable removal preserving unrelated files. |
| [Dependency notice inventory](tests/dependency-notices/README.md) | Collected identities and notice material; publisher review and recursive runtime closure remain separate. |
| [CodeQL](.github/codeql/README.md) | Static-analysis baseline. C++ uses source-only analysis with documented C++26 coverage limits; a clean scan is not complete security or runtime qualification. |
| [Final-download acceptance](tests/launch-acceptance/README.md) | Still pending: actual approved candidate bytes, independent signer trust, fresh hosts and all required scenarios, including promised provider operations. |

Run the source checks with CMake from the repository root:

```sh
cmake -P tests/mvp-dependencies/verify.cmake
cmake -P tests/release-qualification/verify.cmake
```

The [finish-line gate checklist](.github/RELEASE-GATES.md) consolidates the remaining capability selection, credential/provider implementation, redistribution review, candidate trust, acceptance and operational ownership decisions. [Launch ownership](.github/LAUNCH.md) names @quantDIY as release approver and rollback primary. Completing source tasks does not itself authorize publication.

Symphony releases module by module. Existing SODV source publications below are public Go source-module versions, not a platform binary launch. Independent versions and compatibility boundaries remain visible through release preparation.

## Support and contributing

For non-sensitive defects, [open a bug report](https://github.com/QuanuX/Symphony/issues/new?template=bug.yml) with the exact component version/source, OS/architecture, reproduction, expected outcome and redacted evidence. For vulnerabilities, use the support-ticket route in [SECURITY.md](SECURITY.md); do not disclose credentials or exploit details in a public issue. Response ownership and intake qualification remain tracked release work; no response-time guarantee is stated.

When changing a component, start with its owning contracts and installation guide. Keep proposals labelled, preserve independent versions and receipt ownership, update the dependency/test maps when their inputs change, and run the affected focused and installed checks. Canonical change recording follows [SCLV guidance](knowledge/sclv/SKILL.md); historical records are append-only.

## Python Doctrine

First-party source, generators, schema authoring, tests and acceptance helpers have no Python implementation or dependency. User-authored programs and explicitly selected isolated Habitats are not restricted by that repository policy.

## License

Symphony is licensed under the [GNU Affero General Public License v3.0 only](LICENSE) (`AGPL-3.0-only`). Without a separate written agreement, use, modification, distribution and network deployment are governed by that license. Commercial licensing inquiries: `licensing@quanux.org`.

## Repository facts

The following generated snapshot is validated against the owning source contracts. Its feature catalog is intentionally partial; counts are not a claim of complete platform or host coverage.

<details>
<summary>Machine-checked feature, command and source-publication inventory</summary>

<!-- symphony:root-summary:v1:begin -->
## Machine-Checked Repository Snapshot

This bounded summary is derived from canonical SSFV coverage and routing, the feature-administration profile, the qxctl command registry, and completed SODV publication records. Edit its source contracts, then regenerate; do not hand-edit the values below.

- SSFV catalog state: `partial`; registered features: **151**; registered owner scopes: **67**; ratified nested features: **79**.
- Feature-administration expectations: **414** reviewed surfaces; **403** required, **29** evidence-backed exemptions, **9** prohibitions, **0** unreviewed.
- qxctl stable command identities: **487**.
- Registered owner capabilities:
  - `ssfv:symphony:accordare-stav-producer`
  - `ssfv:symphony:knowledge-session-coordinator`
  - `ssfv:symphony:knowledge-vector-engine-foundation`
  - `ssfv:symphony:maestro-presence-authority`
  - `ssfv:symphony:native-source-support-cpp`
  - `ssfv:symphony:platform`
  - `ssfv:symphony:qxctl`
  - `ssfv:symphony:sacv-engine`
  - `ssfv:symphony:sav-engine`
  - `ssfv:symphony:sbv-engine`
  - `ssfv:symphony:sbv-go-adapter`
  - `ssfv:symphony:scabv-ibkr-client-portal-cpp`
  - `ssfv:symphony:scabv-ibkr-tws-cpp`
  - `ssfv:symphony:scev-cf-engine`
  - `ssfv:symphony:scev-engine`
  - `ssfv:symphony:schv-aws-engine`
  - `ssfv:symphony:schv-azure-engine`
  - `ssfv:symphony:schv-do-engine`
  - `ssfv:symphony:schv-engine`
  - `ssfv:symphony:schv-gcp-engine`
  - `ssfv:symphony:sciv-engine`
  - `ssfv:symphony:sclv-engine`
  - `ssfv:symphony:scnv-engine`
  - `ssfv:symphony:scv-engine`
  - `ssfv:symphony:scv-graph-duckdb-connector`
  - `ssfv:symphony:sev-engine`
  - `ssfv:symphony:shv-engine`
  - `ssfv:symphony:shv-graph-adapter`
  - `ssfv:symphony:shv-graph-duckdb-connector`
  - `ssfv:symphony:shv-partition-engine`
  - `ssfv:symphony:shv-pdf-adapter`
  - `ssfv:symphony:shv-profile-engine`
  - `ssfv:symphony:shv-publication-engine`
  - `ssfv:symphony:shv-source-engine`
  - `ssfv:symphony:skvi-engine`
  - `ssfv:symphony:sniv-engine`
  - `ssfv:symphony:snrv-engine`
  - `ssfv:symphony:snv-common-cpp`
  - `ssfv:symphony:snv-engine`
  - `ssfv:symphony:snv-local-observer`
  - `ssfv:symphony:sodv-engine`
  - `ssfv:symphony:sov-cloudflare-adapter`
  - `ssfv:symphony:sov-stdio-adapter`
  - `ssfv:symphony:sqav-attempt-engine`
  - `ssfv:symphony:sqav-capture-cpp`
  - `ssfv:symphony:sqav-databento-dbn-cpp`
  - `ssfv:symphony:sqav-databento-reference-cpp`
  - `ssfv:symphony:sqav-fred-cpp`
  - `ssfv:symphony:sqav-news-api-cpp`
  - `ssfv:symphony:sqav-request-engine`
  - `ssfv:symphony:sqdv-checkpoint-engine`
  - `ssfv:symphony:sqdv-delivery-cpp`
  - `ssfv:symphony:sqfv-batch-cpp`
  - `ssfv:symphony:sqfv-flow-engine`
  - `ssfv:symphony:sqmv-metadata-cpp`
  - `ssfv:symphony:sqmv-metadata-engine`
  - `ssfv:symphony:sqpv-inspection-engine`
  - `ssfv:symphony:sqpv-local-store-cpp`
  - `ssfv:symphony:sqtv-conversion-engine`
  - `ssfv:symphony:sqtv-integer-conversion-cpp`
  - `ssfv:symphony:srrapiv-runtime-cpp`
  - `ssfv:symphony:ssfv-engine`
  - `ssfv:symphony:ssiag-foundation`
  - `ssfv:symphony:ssiag.macos-keychain-metadata`
  - `ssfv:symphony:stav-append-authority`
  - `ssfv:symphony:stav-protocol-kernel`
  - `ssfv:symphony:symphony-validator`
- Completed SODV source publications:
  - `github.com/QuanuX/Symphony/libraries/stav-protocol-go` `v0.2.0` (tag `libraries/stav-protocol-go/v0.2.0`, source `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`)
  - `github.com/QuanuX/Symphony/modules/accordare-stav-producer` `v0.1.0-dev` (tag `modules/accordare-stav-producer/v0.1.0-dev`, source `f36c3372ee5fb2ab9d1eb059eac30b543185db2d`)
  - `github.com/QuanuX/Symphony/modules/stav-append-authority` `v0.1.0` (tag `modules/stav-append-authority/v0.1.0`, source `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`)
  - `github.com/QuanuX/Symphony/modules/stav-append-authority` `v0.2.0` (tag `modules/stav-append-authority/v0.2.0`, source `ed7484d70607aa96e64916dd4e59d3972a61980b`)
  - `github.com/QuanuX/Symphony/modules/stav-append-authority` `v0.2.1` (tag `modules/stav-append-authority/v0.2.1`, source `967c005e21a8f153ed453c79baa09fe2ab549f01`)
  - `github.com/QuanuX/Symphony/tools/sov-ssh-adapter` `v0.1.0-dev` (tag `tools/sov-ssh-adapter/v0.1.0-dev`, source `967c005e21a8f153ed453c79baa09fe2ab549f01`)
- Snapshot digest: `sha256:440f2e5dd8c1c1878b285cc79dd9d8c3d77b524ba0915f23ae905b74cb402bd9`
<!-- symphony:root-summary:v1:end -->

</details>
