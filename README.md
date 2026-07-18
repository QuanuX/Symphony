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

- [`qxctl`](tools/qxctl/) is the Go-based Cobra/Viper administrative and query CLI. It implements repository inspection, contract inventory, authenticated SSIAG metadata queries, and authenticated read-only STAV status, verification, query, and doctor commands.
- [Symphony Secure Identity and Access Governance](modules/secure-identity-access-governance/) is an independently installable, cgo-free Go foundation with per-TOPS enrollment, exact local peer and endpoint trust, a metadata-only Unix-socket API, typed safe-metadata STAV production, and native launchd/systemd supervision. Credential use, policy mutation, provider execution, and secret delivery are not enabled.
- [STAV Append Authority](modules/stav-append-authority/) is an independently installable Go service with per-TOPS durable append-only ledgers, mutually authenticated local IPC, exact producer and reader grants, fsync-before-receipt durability, bounded read projections, startup verification and tail recovery, and native launchd/systemd supervision.
- [STAV Protocol for Go](libraries/stav-protocol-go/) is an authority-free Go library implementing the canonical STAV v1 codec, validation, digest, framing, and conformance rules.
- [SSIAG macOS Keychain Provider](modules/ssiag-provider-macos-keychain/) is an independently buildable Swift metadata adapter implementing bounded `hello`, `status`, and `capabilities` operations. Operational Keychain access is deliberately disabled.
- [Symphony Validator](tools/symphony-validator/) is a deterministic, read-only C++26 repository checker with a CMake build, line-oriented evidence, stable exit behavior, and extensive smoke fixtures. Structured projectors, qxctl mediation, CI wiring, and portable installation packaging remain deferred.
- [`knowledge/`](knowledge/) contains the canonical SKV surfaces currently established for source routing (SKVI), change truth (SCLV), API governance (SACV), publication governance (SODV), SSIAG, and STAV. Canonical knowledge governs implementations; tools do not own canonical schemas.

## First Runtime Set

The repository contains proposal-only Contract Quad seeds for `node-troll`, `bus-troll`, and `hotpath-runtime`. No executable implementation, installation readiness, or operational runtime capability is claimed for those modules.

## Current Integration Boundary

SSIAG submits only typed, security-relevant safe metadata to the STAV append authority and never writes ledger files. qxctl authenticates the exact configured SSIAG and STAV endpoints before application exchange and remains read-only. The macOS provider reports metadata only. SACV targets OpenAPI 3.2.0 governance, but no remote HTTP API, SDK, live playground, or published OpenAPI description is currently claimed.

## Releases and Documentation

Symphony releases will roll out module by module rather than waiting for a monolithic platform release. Each published module will carry its own version, compatibility boundary, and evidence; only artifacts actually published from the repository are releases.

Repository contracts and implementation notes document the current development state. Robust operator, security, API, integration, and module documentation will accompany the official launch.

## Root-Level Governance Role

The repository root establishes platform invariants and guarantees modular sovereignty. Implementations remain subordinate to their canonical contracts, and separately installable modules retain their own runtime authority and lifecycle.

## Doctrine

- A troll is a bounded local resident of a Symphony runtime domain; it is not an AI agent.
- `node-troll` represents the node and `bus-troll` manages an optional managed-bus residency boundary at the contract level only today.
- `hotpath-runtime` is the proposed native hot-path runtime substrate and is not a troll.
- Bus bypass remains valid when declared by deployment constraints; the presence of a bus contract does not make bus traversal mandatory.

These statements describe the current canonical contract seeds, not implemented runtime capability.

## Python Doctrine

Python is not required for remote native hot-path execution or the administrative spine. Optional isolated Python habitats may exist only when explicitly declared by a module or tool.

## License

Symphony is licensed under the GNU Affero General Public License v3.0 only (`AGPL-3.0-only`). Without a separate written agreement, use, modification, distribution, and network deployment are governed by that license. For commercial licensing inquiries, contact `licensing@quanux.org`.
