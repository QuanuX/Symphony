# QuanuX Symphony

> [!IMPORTANT]
> Symphony is in active development. The repository contains operational foundations, metadata-only integrations, and proposal-only contract seeds; it is not an overall production release.

## Identity

- **QuanuX** is the brand, ecosystem, and stewardship identity.
- **Symphony** is the open-source platform.

## Architecture

Symphony is intentionally organized as a monorepo so maintainers and agentic tools can inspect canonical knowledge, implementation, integration boundaries, and validation evidence together. Deployment remains modular: runtime modules preserve independent lifecycle, identity, configuration, state, and version boundaries.

Root governance establishes shared invariants without turning the repository into one monolithic runtime or imposing platform-wide infrastructure, market-data, or order-flow assumptions.

## Implemented Foundations

- [`qxctl`](tools/qxctl/) is the Go-based Cobra/Viper administrative and query CLI. It implements repository inspection, contract inventory, authenticated SSIAG metadata queries, authenticated read-only STAV operations, and exact-installation SKVI, SCLV, SACV, and SODV engine invocation with hard process deadlines and response verification.
- [Symphony Secure Identity and Access Governance](modules/secure-identity-access-governance/) is an independently installable, cgo-free Go foundation with per-TOPS enrollment, exact local peer and endpoint trust, a metadata-only Unix-socket API, typed safe-metadata STAV production, and native launchd/systemd supervision. Credential use, policy mutation, provider execution, and secret delivery are not enabled.
- [STAV Append Authority](modules/stav-append-authority/) is an independently installable Go service with per-TOPS durable append-only ledgers, mutually authenticated local IPC, exact producer and reader grants, fsync-before-receipt durability, bounded read projections, startup verification and tail recovery, and native launchd/systemd supervision.
- [STAV Protocol for Go](libraries/stav-protocol-go/) is an authority-free Go library implementing the canonical STAV v1 codec, validation, digest, framing, and conformance rules.
- [SSIAG macOS Keychain Provider](modules/ssiag-provider-macos-keychain/) is an independently buildable Swift metadata adapter implementing bounded `hello`, `status`, and `capabilities` operations. Operational Keychain access is deliberately disabled.
- [Symphony Validator](tools/symphony-validator/) is a deterministic, read-only C++26 repository checker with a CMake build, line-oriented evidence, stable exit behavior, and extensive smoke fixtures. Structured projectors, qxctl mediation, CI wiring, and portable installation packaging remain deferred.
- [Knowledge Vector Engine C++ Foundation](libraries/knowledge-vector-engine-cpp/) implements authority-free bounded JSON process framing, SHA-256 digests, no-follow repository reads, deterministic snapshots, versioned packaging, receipts, and receipt-owned uninstall mechanics.
- [Knowledge Session Coordinator](modules/knowledge-session-coordinator/) is an independently installable C++26 process implementing bounded inspect and read-only snapshot checks. Authenticated session mutation, journals, locks, vector coordination, and apply remain disabled.
- [SKVI Engine](modules/skvi-engine/) is an independently installable C++26 structural knowledge engine implementing deterministic inspect/check, caller-declared immutable proposals, and disposable digest-bound JSON projections. It cannot decide index membership or write canonical knowledge.
- [SCLV Engine](modules/sclv-engine/) is an independently installable C++26 change-truth engine implementing deterministic ledger checks, provider-neutral v3 proposals, non-mutating closure recovery, disposable projections, and bounded local-Git and air-gapped evidence adapters. It cannot ratify, append, commit, or delete recovery journals.
- [SACV Engine](modules/sacv-engine/) is an independently installable C++26 API-contract governance engine implementing bounded OpenAPI 3.2.0 JSON checks, deterministic compatibility diffs, caller-declared registry proposals, and disposable registry inventories. YAML entry documents fail closed until the separate parser gate; no endpoint, SDK, publication, generated binding, or canonical apply is implemented.
- [SODV Engine](modules/sodv-engine/) is an independently installable C++26 release-publication governance engine implementing local append-only ledger checks, caller-supplied observation verification, provider-neutral release-record proposals, non-mutating interrupted-session recovery, and disposable release inventories. It performs no network access, creates no tags, declares no release complete, and exposes no canonical apply.
- [`knowledge/`](knowledge/) contains the canonical SKV surfaces currently established for source routing (SKVI), change truth (SCLV), API governance (SACV), publication governance (SODV), SSIAG, and STAV. Canonical knowledge governs implementations; tools do not own canonical schemas.

## First Runtime Set

The repository contains proposal-only Contract Quad seeds for `node-troll`, `bus-troll`, and `hotpath-runtime`. No executable implementation, installation readiness, or operational runtime capability is claimed for those modules.

## Current Integration Boundary

SSIAG submits only typed, security-relevant safe metadata to the STAV append authority and never writes ledger files. qxctl authenticates the exact configured SSIAG and STAV endpoints before application exchange and performs no canonical mutation. For SKVI, SCLV, SACV, and SODV, qxctl validates an exact inactive-undocked installation before invoking its bounded local process; lifecycle selection, docking, and apply are not implemented. The macOS provider reports metadata only. SACV's canonical registry remains empty: no remote HTTP API, SDK, live playground, or published OpenAPI description is currently claimed. SODV release observation remains caller-supplied: the engine does not contact Git hosts or package providers.

## Releases and Documentation

Symphony releases will roll out module by module rather than waiting for a monolithic platform release. Each published module will carry its own version, compatibility boundary, and evidence; only artifacts actually published from the repository are releases.

The currently published source-module set is intentionally narrow:

- STAV Protocol for Go `v0.2.0` is the current public protocol-kernel source module;
- STAV Append Authority `v0.2.0` is the current public supervised service source module;
- STAV Append Authority `v0.1.0` remains immutable historical release evidence for the pre-supervision boundary.

These are public Go source-module versions, not GitHub binary releases or a platform launch. qxctl, SSIAG, provider adapters, proposal-only runtime modules, SDKs, containers, and documentation sites are not claimed as released.

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
