# QuanuX Symphony

> [!IMPORTANT]
> Symphony is in active development. The repository contains operational foundations, bounded integrations, and proposal-only contract seeds; it is not an overall production release.

## Identity

- **QuanuX** is the brand, ecosystem, and stewardship identity.
- **Symphony** is the open-source platform.

## Architecture

Symphony is intentionally organized as a monorepo so maintainers and agentic tools can inspect canonical knowledge, implementation, integration boundaries, and validation evidence together. Deployment remains modular: runtime modules preserve independent lifecycle, identity, configuration, state, and version boundaries.

Root governance establishes shared invariants without turning the repository into one monolithic runtime or imposing platform-wide infrastructure, market-data, or order-flow assumptions.

## Implemented Foundations

- [`qxctl`](tools/qxctl/) is the Go-based Cobra/Viper administrative and query CLI. It implements repository inspection, contract and invariant inventory, governed validator profiles, baselines, protected warning lifecycle, invariant assurance, and root-summary projection, authenticated SSIAG metadata, provider-trust, policy, exact provider-installation and binding-lifecycle administration, authenticated read-only STAV operations, exact SSIAG/STAV enrollment and native-supervision lifecycle routes, exact engine binding and reconciliation, explicit idempotent login/refresh/logout knowledge-session transitions, persistent SSFV session-maintenance administration, complete read-only Maestro inventory, and exact-installation SKVI, SCLV, SACV, SODV, SSFV, SAV, SEV, and Symphony Validator invocation with hard process deadlines and response verification. Its checked-in registry currently binds 185 executable commands to stable machine identities and reviewed feature-administration evidence.
- [Symphony Secure Identity and Access Governance](modules/secure-identity-access-governance/) is an independently installable, cgo-free Go foundation with receipt-v2 side-by-side installation, exact per-TOPS enrollment and native-supervision observe/plan/apply/status/recovery transactions, exact local peer and endpoint trust, a bounded Unix-socket API, exact caller-neutral allow/deny decisions, protected local policy proposal/apply/recovery, exact receipt-backed provider trust inspection and permission-backed fresh metadata verification, complete receipt-owned provider-bundle staging, three-layer signed-bundle/policy/session readiness observation, opaque exact provider-installation inventory, two-way provider-binding plan/apply/status/recovery with crash-safe STAV-before-commit durability, typed safe-metadata STAV production, and native launchd/systemd supervision. Ordinary foundational mutation fails closed pending the closed lifecycle audit-receipt route; explicit audit-deferred mutation remains durable and reconciliation-required. Operational credential use, Keychain operations, canonical knowledge apply, and secret delivery are not enabled.
- [STAV Append Authority](modules/stav-append-authority/) is an independently installable Go service with receipt-v2 side-by-side installation, exact per-TOPS enrollment and native-supervision observe/plan/apply/status/recovery transactions, per-TOPS durable append-only ledgers, mutually authenticated local IPC, exact producer and reader grants, fsync-before-receipt durability, bounded read projections, startup verification and tail recovery, and native launchd/systemd supervision. qxctl never receives raw append authority.
- [STAV Protocol for Go](libraries/stav-protocol-go/) is an authority-free Go library implementing the canonical STAV v1 codec, validation, digest, framing, and conformance rules.
- [SSIAG macOS Keychain Provider](modules/ssiag-provider-macos-keychain/) is an independently installable Swift metadata adapter implementing one bounded, mutually verified provider handshake plus a separate Phase 10B signed-bundle and security-session readiness observation. Production-bundle construction, exact receipt ownership, native code-requirement evaluation, and safe session evidence are implemented; no signed release artifact is claimed, and operational Keychain access remains deliberately disabled.
- [Symphony Validator](tools/symphony-validator/) is a deterministic, read-only C++26 repository checker with line and structured JSON evidence, common-invariant ownership and source-module admission assurance, bounded digest-bearing root-summary JSON/Markdown projection and freshness assurance, stable exit behavior, exact versioned installation/uninstallation, qxctl mediation, and extensive smoke fixtures. Its module census closes source-level declaration omissions; a complete receipt-bound installed-host inventory remains a separately gated protocol. CI wiring and general documentation projection remain deferred.
- [Knowledge Vector Engine C++ Foundation](libraries/knowledge-vector-engine-cpp/) implements authority-free bounded JSON process framing, SHA-256 digests, no-follow repository reads, deterministic snapshots, versioned packaging, receipts, and receipt-owned uninstall mechanics.
- [Knowledge Session Coordinator](modules/knowledge-session-coordinator/) is an independently installable C++26 process implementing bounded inspection, snapshot checks, explicit compatibility negotiation, durable per-worktree reconciliation, SSIAG-authorized noncanonical authority epochs, persistent SSFV baseline/review maintenance, protected report-only lifecycle journals, and separate apply-capable journals with prepared attempts, exact compare-and-swap, content-addressed applied evidence, dynamic replanning, and evidence-based recovery. It may prepare and verify an external Maestro action; vector invocation, host action execution inside the coordinator, observers, and canonical apply remain disabled.
- [Maestro](modules/maestro/) is an independently installable freezing-path C++26 presence authority. It records exact authenticated vector-engine docking/undocking relationships per TOPS and receptor with dual-slot durability and forward recovery, and derives a complete stable read-only inventory from those authoritative registries. It does not start, schedule, supervise, or invoke docked engines.
- [SKVI Engine](modules/skvi-engine/) is an independently installable C++26 structural knowledge engine implementing deterministic inspect/check, caller-declared immutable proposals, and disposable digest-bound JSON projections. It cannot decide index membership or write canonical knowledge.
- [SCLV Engine](modules/sclv-engine/) is an independently installable C++26 change-truth engine implementing deterministic ledger checks, provider-neutral v3 proposals, non-mutating closure recovery, disposable projections, and bounded local-Git and air-gapped evidence adapters. It cannot ratify, append, commit, or delete recovery journals.
- [SACV Engine](modules/sacv-engine/) is an independently installable C++26 API-contract governance engine implementing bounded OpenAPI 3.2.0 JSON checks, deterministic compatibility diffs, caller-declared registry proposals, and disposable registry inventories. YAML entry documents fail closed until the separate parser gate; no endpoint, SDK, publication, generated binding, or canonical apply is implemented.
- [SODV Engine](modules/sodv-engine/) is an independently installable C++26 release-publication governance engine implementing local append-only ledger checks, caller-supplied observation verification, provider-neutral release-record proposals, non-mutating interrupted-session recovery, and disposable release inventories. It performs no network access, creates no tags, declares no release complete, and exposes no canonical apply.
- [SSFV Engine](modules/ssfv-engine/) is an independently installable C++26 semantic-feature engine implementing structural and freshness-aware checks, content-addressed diffs, caller-declared proposals, disposable deterministic graphs, and repository-independent feature-administration assurance. Its explicitly partial catalog records seventy-eight ratified experimental records across the root and fourteen implemented owner scopes, including fifty-nine exact nested features. All 156 registered administration expectations have reviewed routes or evidence-backed dispositions; the engine does not decide feature-worthiness, invent command identities, claim repository-wide or installed-host completeness, or write canonical truth.
- [Symphony Accord Vector](modules/sav-engine/) is an independently installable freezing-path C++26 engine for deterministic, read-only Accord reference resolution, immutable derived CURRENT snapshots, three-axis evaluation, comparison, explanation, disposable graphs, Named Version validation and diff, Extension Capsule checks, Installation Blueprint planning, and explicit compatibility negotiation. Its outputs are evidence and proposals only; it does not write canonical knowledge, seal versions, install components, or dock engines.
- [Symphony Evolution Vector](modules/sev-engine/) is an independently installable freezing-path C++26 engine for deterministic, read-only evolution cases, impact and disposition planning, dependency-ready-set recalculation, transition verification, recovery advice, SCSEV command-surface assessment, novelty and watch-policy checks, trigger coalescing, and lifecycle-session binding. It reuses the shared lifecycle journal rather than creating a second mutation authority, and it neither watches a host nor applies a transition itself.
- [`knowledge/`](knowledge/) contains the canonical SKV surfaces currently established for source routing (SKVI), change truth (SCLV), API governance (SACV), publication governance (SODV), semantic feature truth (SSFV), composition and evaluation (SAV), evolution and command-surface assessment (SEV/SCSEV), SSIAG, STAV, and common temporal, validation, feature-administration, foundational-lifecycle, and cross-vector desired/observed/plan/applied/boot contracts. The coordinator implements report-only dependency-driven two-way planning over supplied evidence, so compatible component actions can be replanned around localized blockers without changing ordered safety phases. Canonical knowledge governs implementations; tools do not own canonical schemas.

## First Runtime Set

The repository contains proposal-only Contract Quad seeds for `node-troll`, `bus-troll`, and `hotpath-runtime`. No executable implementation, installation readiness, or operational runtime capability is claimed for those modules.

## Current Integration Boundary

qxctl includes an explicit Linux-only systemd receptor for report-only lifecycle planning at host boot. It is independently installable, updateable, disableable, reconcilable, and uninstallable per TOPS/profile. The receptor uses content-addressed qxctl executors, bounded accepted fallbacks, kernel boot UUID idempotency, durable compare-and-swap descriptors, and resumable removal; it never invokes lifecycle apply or component execution. Native Windows host integration is not planned—Windows users use WSL or administer a remote Linux TOPS node.


SSIAG submits only typed, security-relevant safe metadata to the STAV append authority and never writes ledger files. qxctl authenticates the exact configured SSIAG and STAV endpoints before application exchange and performs no canonical mutation. It implements exact user-default engine selection, durable reconciliation, SSIAG-authorized session operations, explicit host-event session convergence, persistent SSFV baseline/checkpoint/close/recovery administration, protected validator warning-policy and baseline administration, protected per-TOPS lifecycle profiles, root-local multi-profile receipt ownership with old-client compatibility fencing and conservative reclamation, fixed-layout receipt and Maestro-presence observation, complete read-only Maestro receptor inventory, report-only boot journals, exact SSIAG/STAV enrollment and native-supervision lifecycle administration, exact opaque provider-installation inventory and two-way provider-binding lifecycle administration, and an explicit local apply-compatible circuit through the exact bound coordinator. The current apply adapters can install exact staged receipt-v2 packages and reclaim them only after all shared-root claims release, update protected generic selection/activation state, perform verified side-by-side coordinator handoff, and commit authenticated Maestro docking presence; attempts are recorded before host mutation and applied evidence advances only after re-observation. SSIAG provider binding separately preserves the initiating safe audit identity, verifies the exact foundation/adapter pair, requires committed STAV evidence, changes state while recovery evidence remains durable, and supports receipt-bound offline recovery only under exclusive stopped-service ownership. Package download, receipt-v1 mutation, arbitrary entry-point execution, live service activation, Maestro engine execution, in-place coordinator self-replacement, login/session hook installation, hidden watchers, native Windows host integration, and canonical knowledge apply are not implemented. For SKVI, SCLV, SACV, SODV, and SSFV, qxctl validates an exact inactive-undocked installation before invoking its bounded local process. SSFV coverage is explicitly partial: registered owner scopes and ratified nested features are cataloged, but this is not a repository-completeness claim. The macOS provider reports metadata only. SACV's canonical registry remains empty: no remote HTTP API, SDK, live playground, or published OpenAPI description is currently claimed. SODV release observation remains caller-supplied: the engine does not contact Git hosts or package providers.

<!-- symphony:root-summary:v1:begin -->
## Machine-Checked Repository Snapshot

This bounded summary is derived from canonical SSFV coverage and routing, the feature-administration profile, the qxctl command registry, and completed SODV publication records. Edit its source contracts, then regenerate; do not hand-edit the values below.

- SSFV catalog state: `partial`; registered features: **78**; registered owner scopes: **14**; ratified nested features: **59**.
- Feature-administration expectations: **156** reviewed surfaces; **145** required, **13** evidence-backed exemptions, **10** prohibitions, **0** unreviewed.
- qxctl stable command identities: **185**.
- Registered owner capabilities:
  - `ssfv:symphony:knowledge-session-coordinator`
  - `ssfv:symphony:knowledge-vector-engine-foundation`
  - `ssfv:symphony:maestro-presence-authority`
  - `ssfv:symphony:qxctl`
  - `ssfv:symphony:sacv-engine`
  - `ssfv:symphony:sclv-engine`
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
- Snapshot digest: `sha256:f3763c64734edab2c202df5468e6f4c751c7719e354947d404ed9686c1ab1dd7`
<!-- symphony:root-summary:v1:end -->

## Releases and Documentation

Symphony releases will roll out module by module rather than waiting for a monolithic platform release. Each published module will carry its own version, compatibility boundary, and evidence; only artifacts actually published from the repository are releases.

The machine-checked snapshot above lists every completed SODV source publication. Those are public Go source-module versions, not GitHub binary releases or a platform launch. qxctl, SSIAG, provider adapters, proposal-only runtime modules, SDKs, containers, and documentation sites are not claimed as released.

Repository contracts and implementation notes document the current development state. Robust operator, security, API, integration, and module documentation will accompany the official launch.

## Root-Level Governance Role

The repository root establishes platform invariants and guarantees modular sovereignty. Implementations remain subordinate to their canonical contracts, and separately installable modules retain their own runtime authority and lifecycle.

## Doctrine

- A troll is a bounded local resident of a Symphony runtime domain; the term describes runtime residency, not caller identity, intelligence, or authorization.
- `node-troll` represents the node and `bus-troll` manages an optional managed-bus residency boundary at the contract level only today.
- `hotpath-runtime` is the proposed native hot-path runtime substrate and is not a troll.
- Bus bypass remains valid when declared by deployment constraints; the presence of a bus contract does not make bus traversal mandatory.

Symphony authorizes supported operations from target-host ownership or granted permission, not from whether a caller is human, AI, a service, or another actor type. The host administrator controls configurable safeguards; protocol-integrity rules remain mandatory within supported tooling.

These statements describe the current canonical contract seeds, not implemented runtime capability.

## Python Doctrine

Python is not required for remote native hot-path execution or the administrative spine. Optional isolated Python habitats may exist only when explicitly declared by a module or tool.

## License

Symphony is licensed under the GNU Affero General Public License v3.0 only (`AGPL-3.0-only`). Without a separate written agreement, use, modification, distribution, and network deployment are governed by that license. For commercial licensing inquiries, contact `licensing@quanux.org`.
