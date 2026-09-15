# QuanuX Symphony

> [!IMPORTANT]
> Symphony is in active development. Phase 1 contains substantial working foundations; the emerging vector architecture remains deliberately separated from runtime claims that have not yet been implemented.

Symphony is an agentic-first cockpit for quantitative developers, researchers, traders, and the other participants a user chooses to involve. Its long-range purpose is to provide one coherent, independently installable suite for researching, constructing, testing, deploying, operating, and eventually observing quantitative trading systems—without taking ownership of the user's strategy logic or forcing every installation into one infrastructure model.

## Identity

- **QuanuX** is the brand, ecosystem, application owner, and steward of official source truth.
- **Symphony** is the open-source platform.
- A user's Symphony or TOPS installation is private to that user. Its declared and observed truth may guide its own bespoke evolution without becoming official QuanuX source truth.

## Architecture

Symphony is a monorepo for inspectability, not a monolithic runtime. Canonical knowledge, implementation, integration boundaries, and validation evidence live together so people and agents can reason from the same source, while runtime modules preserve independent identity, version, configuration, state, installation, update, and removal boundaries.

Phases are delivery chronology. Vectors are enduring semantic owners. A component may first be researched in one phase without making that phase its permanent architectural owner.

The platform separates four concerns that must not be collapsed:

- owner-controlled canonical meaning;
- exact evidence about physical and software reality;
- user-selected composition and operational action;
- derived projections that can always be traced back to their owners.

Symphony reports exact compatibility, incompatibility, unresolved facts, and consequences. It does not rank providers, prescribe a strategy, infer authority from caller type, or quietly widen a user's environment.

### Node, Habitat, and Nest

A **Node** is the target physical computer resource with specifically identified resources. Disconnecting, reconnecting, restarting, or reflashing only software does not create a new physical Node. Destruction, physical replacement, or a material local hardware change does. A Node incarnation is separate evidence created when that Node is attached to a bus as a participant in a trading system, research cluster, or another Symphony-built system.

A **Habitat** is the exact operating system and package conditioning delivered to a Node for the work selected by the user. Its default is the required software surface only: no unrelated packages, hidden runtimes, containers, Kubernetes, or implicit virtualization. A broad general-purpose Habitat remains a valid explicit user choice.

A **Nest** serves a purpose chosen by the user. Live strategy execution is expected to be common, and one strategy may contain many modular code sections, but Symphony does not define the Nest's purpose or equate it with one source file, process, algorithm, broker API, or data interface. Strategy logic remains the user's domain.

Multiple Nodes form a cluster only when connected to one another through one or more buses. Multiple bus fabrics may coexist inside one cluster, and a Node-in-cluster relationship may identify the relevant bus without changing the physical Node's identity.

## Emerging Vector Architecture

The detailed, ratified baseline is in [Symphony Emerging Vector Architecture](knowledge/ARCHITECTURE.md). Its terms are routed through [SLANG](knowledge/SLANG.md), while universal identity-family ownership and retired identifiers are governed by [NAMESPACES](knowledge/NAMESPACES.md).

| Vector | Bounded purpose | Current posture |
|---|---|---|
| **SKV — Symphony Knowledge Vector** | Canonical knowledge architecture and the common contracts that let independently owned vectors remain coherent and agent-readable. | Phase 1 foundation is implemented; current and emerging surfaces are manifest-declared and SKVI-indexed. |
| **SKVI / SCLV / SACV / SODV / SSFV / SAV / SEV** | Source routing, change truth, API governance, official documentation and release-publication governance, semantic feature truth, Accordare composition, and governed evolution. | Contracted Phase 1 domains with bounded engines and administrative integrations where their individual contracts say so. SODV governs official projection; it is not the publisher. |
| **SOV — Symphony Ops Vector** | qxctl-administered provisioning, Habitat conditioning, Nest delivery, bus-adapter setup, and optional remote-Node operations. | Emerging operations domain; provider, Terraform, Habitat, Nest, bus, and remote-operation protocols remain to be designed and implemented. |
| **SCV — Symphony Cloud Vector** | Private knowledge of offsite provider resources, offerings, regions, constraints, observations, and hybrid possibilities. | Eight independently installed C++ engines provide bounded source/corpus knowledge, portable provider packages and finite caller-directed composition. qxctl supports retained workflows and protected graph selection; an optional DuckDB connector adds retained relational graph indexing through qxctl. Comprehensive provider coverage, dedicated graph traversal and operational adapters remain separate work. |
| **SNV — Symphony Node Vector** | Records and relates Node identity, resources, cluster relationships, and names without dictating them. | Emerging composition of **SNIV** (identity), **SNRV** (resources), **SCIV** (cluster identity/connectivity), and SNV-bounded **SCNV** (consolidated naming). Record schemas and engines remain deferred. |
| **SQV — Symphony Quantitative Vector** | Reusable quantitative and trading-system framework contracts without acquiring user strategy logic. | Emerging domain. **SOOV — Symphony Orchestra Omega Vector** is its first named subvector: a future high-performance, C++-only FIX architecture informed by historic QuanuX work. Detailed FIX behavior is not yet canonical. |
| **SHV — Symphony Hardware Vector** | Hardware-capability knowledge for processors, CPU topology and execution-unit designs, caches, GPUs, NICs and fibre interfaces, motherboards, RAM and NVMe; default curation prioritizes original components and evidenced variants. | Initial C++ kernel: caller-defined coverage, retained-source catalogue replay, requirement evaluation and graph projection. Source revision/capture provenance engine and generic graph exchange are implemented; protected source activation, broad component mappings, durable vendor drivers and Composer integration remain open. |
| **SIV — Symphony Intelligence Vector** | Future local and remote agent collaboration, extended context, structured long-term logic, communication, and governed Symphony interaction. | Emerging domain. **SMCV** is its optional Markdown conversion component; **SAIV** is reserved for a later integration subvector and currently has no behavior. |

SAV continues to mean **Symphony Accordare Vector**. It is not the intelligence vector.

SMCV is planned as an optional, qxctl-configurable translator above an enabled IPC connection. It may preserve precise Markdown for an agent while mapping to the exact versioned JSON or other machine format owned by the receiving contract; it cannot invent missing meaning, repair invalid input, or authorize execution. No SMCV command or translator is implemented yet.

Every new vector and subvector inherits the Phase 1 method without surrendering its domain:

1. the semantic owner receives a Contract Quad—`INTENT.md`, `MANIFEST.md`, `SKILL.md`, and `SPEC.md`—plus only the focused companion surfaces it actually needs;
2. owner manifests declare their canonical surfaces and delegation graph;
3. SKVI maps each current surface exactly once;
4. SLANG and NAMESPACES route terminology and identity families without copying owner doctrine;
5. SCLV, SEV, SSFV, SACV, SAV, qxctl, lifecycle, SSIAG, STAV, Maestro, validation, and SODV are engaged only when their own applicability boundaries are met; and
6. obsolete companion surfaces are superseded, retired, or removed through the same governed evolution path instead of lingering as contradictory agent context.

This is evidence-based integration, not ceremonial duplication. A deferred engine has no fabricated receipt, a proposed API has no invented registry entry, and an unpublished contract has no SODV completion claim.

### Operations, provider, and bus posture

The user chooses provider, region, owned hardware, topology, bus, software, and per-Node administration arrangement. The initial SCV comparison universe includes AWS, Azure, Google Cloud, DigitalOcean, Cloudflare, and user-owned hardware, including mixed-provider or region-constrained systems. Each provider's realities remain visible without treating any provider as the baseline.

SOV is the control plane through which authorized qxctl operations may act on SCV evidence. SCV owns provider meaning; SOV does not rewrite it, and SCV does not perform provider operations.

Bus choice is also user-controlled. NATS JetStream, ZeroMQ, and other fabrics are adapters rather than doctrine. Heavy transfer, control, research, and execution exhaust do not have to share one bus. A topology may use multiple buses or bypass a bus for a declared path, especially where shared traffic could add jitter. Symphony does not prescribe the contents of strategy exhaust.

qxctl and the knowledge plane are cold or freezing-path administration. They may explicitly act on a Node that also hosts live work, but they must not become continuous hot-path dependencies, hidden watchers, resident interpreters, or unsolicited background package managers.

### Remote qxctl and Prima Parte

Phase 2 research preserves three optional remote-administration arrangements: controller-side qxctl, qxctl installed on an enabled target Node, and qxctl staged for one bounded target operation. The cluster creator chooses the arrangement, globally or per Node. IPC is preferred where the selected topology supports it; a scoped remote CLI or shell-mediated invocation may be a fallback. The controller/target work split and exact transport remain open design questions.

**Prima Parte** is a ratified Phase 3 concept, not a current executable. It is an optional, extremely lightweight C++ Node-local state witness explicitly invoked by qxctl after a material operation. It has no daemon, listener, timer, watcher, polling loop, or resident bus connection. Its bounded durable vocabulary includes `current`, `previous`, and `lastTransmitted`; transmitted means sent by the remote Node after the associated command executed, not confirmed received. Prima Parte records only its own Node, never becomes a smaller Maestro, and exits after its bounded work.

The retired `node-troll` and `bus-troll` module identities remain reserved tombstones. “Troll” may still describe an optional user-programmed resident living at a connection point, but Symphony assigns it no required Node, bus, supervision, compatibility, or messaging role. The concept may disappear entirely if the architecture makes it unnecessary.

## Delivery

Seven delivery sprints remain. Their detailed sequence and contents will be presented when their architecture and public boundaries are ready.

## Implemented Foundations

- [`qxctl`](tools/qxctl/) is Symphony's Go-based, agentic-first administrative and query CLI. Its checked-in registry binds **369** executable command leaves to stable machine identities and reviewed feature-administration evidence. It implements repository and contract inspection; validator, warning, invariant, feature, lifecycle, session, reconciliation, binding, Maestro, SSIAG, STAV, Accordare, and implemented vector-engine administration. Exact installation verification, bounded subprocesses, hard deadlines, response identity/digest checks, expected-state transactions, and durable recovery are used where the owning contract requires them. Engine-binding registry v2 supports the eight established roles and bounded future role identities; legacy v1 state is dual-read and requires explicit digest-bound migration before mutation.
- [Symphony Secure Identity and Access Governance](modules/secure-identity-access-governance/) is an independently installable, cgo-free Go foundation for exact caller-neutral authorization, per-TOPS enrollment, local endpoint trust, protected policy lifecycle, provider-installation and binding lifecycle, safe audit metadata, and native launchd/systemd supervision. Ordinary foundational mutation fails closed pending its required audit route. Operational credential use, canonical knowledge apply, and secret delivery remain disabled.
- [STAV Append Authority](modules/stav-append-authority/) is an independently installable Go service for per-TOPS append-only audit ledgers, mutually authenticated local IPC, exact producer/reader grants, fsync-before-receipt durability, bounded reads, verification, recovery, enrollment, and native supervision. qxctl never receives raw append authority.
- [STAV Protocol for Go](libraries/stav-protocol-go/) is an authority-free Go library implementing the canonical STAV v1 codec, strict validation, digests, framing, conformance rules, and closed producer vocabulary.
- [Accordare STAV Producer](modules/accordare-stav-producer/) is an independently installable, freezing-path Go producer for the bounded SAV Named Version audit circuit. It keeps pre-mutation intent and append recovery durable, authenticates its local peers, and submits only the exact safe candidate admitted by its closed vocabulary; it is not STAV append authority.
- [SSIAG macOS Keychain Provider](modules/ssiag-provider-macos-keychain/) is an independently installable Swift metadata adapter with a bounded mutually verified handshake and signed-bundle/session-readiness observation. Operational Keychain access remains deliberately disabled.
- [Symphony Validator](tools/symphony-validator/) is an independent, deterministic, read-only C++26 repository checker. It reports line or structured JSON evidence for contract shape, current source closure, invariant ownership, feature administration, releases, and the machine-owned root summary; it cannot repair or publish the repository.
- [Knowledge Vector Engine C++ Foundation](libraries/knowledge-vector-engine-cpp/) supplies authority-free process framing, bounded JSON, SHA-256, no-follow reads, deterministic snapshots, temporal validation, exact owner-manifest discovery, versioned packaging, receipts, and receipt-owned uninstall mechanics. Manifest discovery begins from a fixed bootstrap set and traverses only explicit subordinate declarations—never an ambient repository crawl.
- [Knowledge Session Coordinator](modules/knowledge-session-coordinator/) is an independently installable C++26 process for bounded compatibility, worktree reconciliation, authenticated noncanonical sessions, SSFV maintenance, report-only lifecycle planning, and separately authorized apply-capable journals with prepared attempts, compare-and-swap, re-observation, dynamic replanning, and recovery. It does not execute vector semantics or host actions by itself.
- [Maestro](modules/maestro/) is an independently installable freezing-path C++26 presence authority. It records exact authenticated vector-engine docking relationships per TOPS and receptor and derives a complete read-only inventory. It does not start, schedule, supervise, or invoke engines.
- [SKVI Engine](modules/skvi-engine/) implements deterministic inspect/check, caller-declared immutable proposals, and disposable digest-bound projections. It now verifies that every manifest-declared required surface is indexed exactly once, but it cannot decide membership or write canonical truth.
- [SCLV Engine](modules/sclv-engine/) implements deterministic append-only ledger checks, provider-neutral v3 proposals, non-mutating recovery, and disposable projections. Admission binds current indexed evidence; later legitimate retirement preserves immutable historical provenance without forcing deleted current prose back into SKVI.
- [SACV Engine](modules/sacv-engine/) implements bounded OpenAPI 3.2.0 JSON checks, deterministic compatibility diffs, caller-declared registry proposals, and disposable inventories. No endpoint, SDK, publication, generated binding, or canonical apply is implied.
- [SODV Engine](modules/sodv-engine/) implements local append-only release-ledger checks, caller-supplied observation verification, provider-neutral release proposals, non-mutating recovery, and disposable release inventories. It has no network access, publishes nothing, creates no tags, and cannot declare a release complete.
- [SSFV Engine](modules/ssfv-engine/) implements structural and freshness-aware feature checks, content-addressed diffs, caller-declared proposals, deterministic disposable graphs, and feature-administration assurance. Its catalog is explicitly partial; it does not decide feature-worthiness, invent identities, or claim repository/host completeness.
- [Symphony Accordare Vector Engine](modules/sav-engine/) is a freezing-path C++26 engine for deterministic Accord reference resolution, immutable derived CURRENT snapshots, three-axis evaluation, comparison, explanation, graphs, Named Version validation/diff, Extension Capsule checks, Installation Blueprint planning, and compatibility negotiation. It produces evidence and proposals, not canonical mutations.
- [Symphony Evolution Vector Engine](modules/sev-engine/) is a freezing-path C++26 engine for deterministic evolution cases, impact/disposition planning, dependency-ready-set recalculation, transition verification, recovery advice, SCSEV assessment, novelty/watch checks, trigger coalescing, lifecycle-session binding, and graph projection. It neither watches a host nor applies a transition.
- [SCV source-knowledge engines](knowledge/scv/SOURCE-KNOWLEDGE.md) provide eight independently packaged C++26 SCV, hyperscaler, edge and provider domains. They preserve explicit source revisions and bounded captures, retain qualified claims and native document structure, and produce reproducible graph/query evidence. qxctl separately administers audited local source changes and coherent graph selection; provider operations and complete cloud catalogs remain outside this increment.
- [`knowledge/`](knowledge/) contains the current canonical SKV corpus: vector Contract Quads, schemas, registries, profiles, ledgers, companion surfaces, and the emerging Phase 2–8 architecture. Implementations remain subordinate to their semantic owners.

`hotpath-runtime` remains a proposal-only contract seed awaiting its own architectural review. It has no executable implementation or installation-readiness claim.

## Current Integration Boundary

The implemented qxctl and Phase 1 engines are local administrative/cold-or-freezing-path capabilities. qxctl invokes an exact receipt-validated installation and preserves the operation owner's authority and response contract. The current generic apply adapters are deliberately finite: staged receipt-v2 package installation/reclamation, protected selection/activation state, side-by-side coordinator handoff, and authenticated Maestro presence. Package download, receipt-v1 mutation, arbitrary entry-point execution, live-service activation, in-place coordinator self-replacement, hidden watchers, native Windows host integration, Maestro engine execution, and canonical knowledge apply are not implemented.

qxctl also provides a Linux-only systemd receptor for report-only lifecycle planning at host boot. It is independently installable, disableable, reconcilable, and removable per TOPS/profile; it never invokes lifecycle apply or component execution. Windows users currently use WSL or administer a remote Linux TOPS Node.

The emerging SOV remote-operation and deployment contracts do not make provider provisioning, Terraform apply, Habitat construction, Nest delivery, bus installation, remote command transport, or Prima Parte operational today.

<!-- symphony:root-summary:v1:begin -->
## Machine-Checked Repository Snapshot

This bounded summary is derived from canonical SSFV coverage and routing, the feature-administration profile, the qxctl command registry, and completed SODV publication records. Edit its source contracts, then regenerate; do not hand-edit the values below.

- SSFV catalog state: `partial`; registered features: **110**; registered owner scopes: **35**; ratified nested features: **76**.
- Feature-administration expectations: **295** reviewed surfaces; **285** required, **13** evidence-backed exemptions, **9** prohibitions, **0** unreviewed.
- qxctl stable command identities: **369**.
- Registered owner capabilities:
  - `ssfv:symphony:accordare-stav-producer`
  - `ssfv:symphony:knowledge-session-coordinator`
  - `ssfv:symphony:knowledge-vector-engine-foundation`
  - `ssfv:symphony:maestro-presence-authority`
  - `ssfv:symphony:platform`
  - `ssfv:symphony:qxctl`
  - `ssfv:symphony:sacv-engine`
  - `ssfv:symphony:sav-engine`
  - `ssfv:symphony:scev-cf-engine`
  - `ssfv:symphony:scev-engine`
  - `ssfv:symphony:schv-aws-engine`
  - `ssfv:symphony:schv-azure-engine`
  - `ssfv:symphony:schv-do-engine`
  - `ssfv:symphony:schv-engine`
  - `ssfv:symphony:schv-gcp-engine`
  - `ssfv:symphony:sclv-engine`
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
  - `ssfv:symphony:sodv-engine`
  - `ssfv:symphony:ssfv-engine`
  - `ssfv:symphony:ssiag-foundation`
  - `ssfv:symphony:ssiag.macos-keychain-metadata`
  - `ssfv:symphony:stav-append-authority`
  - `ssfv:symphony:stav-protocol-kernel`
  - `ssfv:symphony:symphony-validator`
- Completed SODV source publications:
  - `github.com/QuanuX/Symphony/libraries/stav-protocol-go` `v0.2.0` (tag `libraries/stav-protocol-go/v0.2.0`, source `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`)
  - `github.com/QuanuX/Symphony/modules/stav-append-authority` `v0.1.0` (tag `modules/stav-append-authority/v0.1.0`, source `55f8faf26f4f85213ac23cc1de7ba897b2129a4c`)
  - `github.com/QuanuX/Symphony/modules/stav-append-authority` `v0.2.0` (tag `modules/stav-append-authority/v0.2.0`, source `ed7484d70607aa96e64916dd4e59d3972a61980b`)
- Snapshot digest: `sha256:4a2b293e9531ec1b6d06e39bb672d81d5c5883c5ef0583a07ad8b414f002cd4a`
<!-- symphony:root-summary:v1:end -->

## Releases and Documentation

Symphony releases module by module rather than waiting for a monolithic platform release. Each published unit retains its own version, compatibility boundary, and evidence. Only artifacts actually published from the repository are releases.

The machine-checked snapshot lists every completed SODV source publication. Those entries are public Go source-module versions, not GitHub binary releases or a platform launch. Contracts and implementation notes describe development truth; official public projections remain governed by SODV.

## Root-Level Governance Role

The [platform governance contract](knowledge/platform/INTENT.md) establishes platform invariants, shared interpretation rules, and modular sovereignty within SKV. It does not absorb a vector's meaning or a module's runtime authority.

The SKV evolution framework allows Symphony to add, change, supersede, deprecate, retire, and remove surfaces while preserving the distinction between present truth and historical evidence. Companion files exist only while they improve bounded agent understanding; they must not remain as stale restatements once their purpose disappears.

## Doctrine

- User strategy logic, Nest purpose, provider choice, hardware choice, bus topology, and supported adapter selection remain user-controlled.
- Symphony provides exact compatibility evidence and supported operations; it does not turn recommendations into platform law.
- The hot path is protected from administrative residency and accidental shared-bus pressure. A user's explicit architecture remains sovereign.
- qxctl owns command grammar and presentation. Domain vectors own the operations and meanings those commands expose.
- Names are recorded under their owning scope; SNV and SCNV do not dictate them, and root NAMESPACES remains broader than SNV.
- A caller's classification as human, AI, agent, service, or workload does not grant or remove authority. Target-host ownership, explicit permission, expected state, and owner-configured safeguards govern supported actions.
- Current contracts describe current truth. Immutable SCLV/SODV records preserve what was known, changed, authorized, or completed at their recorded point in history.

## Python Doctrine

First-party Symphony source, generators, schema authoring, tests, and acceptance helpers require no Python implementation or dependency. The former Python development tooling is replaced by C++. This repository policy does not restrict user-authored programs or explicitly selected isolated Habitats.

## License

Symphony is licensed under the GNU Affero General Public License v3.0 only (`AGPL-3.0-only`). Without a separate written agreement, use, modification, distribution, and network deployment are governed by that license. For commercial licensing inquiries, contact `licensing@quanux.org`.
