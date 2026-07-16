# QuanuX Symphony

## Identity
- **QuanuX**: The brand, ecosystem, and stewardship identity.
- **Symphony**: The open-source platform.

## Architecture
Symphony is a monorepo built on the principle of modular sovereignty. 
Every module within Symphony is first-class and individually installable. There are no monolithic, platform-wide infrastructure assumptions, nor any platform-wide domain dogma (such as mandatory market-data or order-flow dependencies).

## First Runtime Set
The first runtime set is currently privately staged, not canonically imported:
- `node-troll`
- `bus-troll`
- `hotpath-runtime`

## Additional Canonical Modules

- `secure-identity-access-governance`: independently installable Go-only SSIAG foundation with per-TOPS enrollment.
- `ssiag-provider-macos-keychain`: optional, independently installable Swift adapter scaffold for the future macOS Keychain boundary.
- `stav-append-authority`: independently installable Go namespace and lifecycle scaffold for the future per-TOPS serialized STAV writer.

Their canonical protocol surfaces are `knowledge/ssiag/` and `knowledge/stav/`. `knowledge/sacv/` governs API contracts and targets OpenAPI 3.2.0 without authorizing a remote API or public endpoint. The Keychain adapter is metadata-only. The STAV module installs only its executable and resolves ratified paths; the authority-free protocol codec is a shared library, while no operational credential access, STAV listener, candidate ingestion, or ledger writer is enabled.

## First-Party Libraries

`libraries/` contains shared, independently testable Go implementation code for canonical contracts. Libraries are build-time dependencies rather than installable modules: they have no binary, resident, service identity, socket, state, or runtime authority. `libraries/stav-protocol-go` implements STAV v1 serialization and validation rules owned by `knowledge/stav/`.

## Root-Level Governance Role
The root repository establishes platform invariants and guarantees modular sovereignty. It does not dictate implementation code. Implemented modules remain subordinate to their canonical contracts and independently installable.

## Doctrine
- trolls are the local residents.
- A troll is a bounded local resident of a Symphony runtime domain.
- A troll is not an AI agent.

- `node-troll` represents the node.
- `bus-troll` manages bus residency and bus compatibility.
- `hotpath-runtime` owns the native hot path.
- `hotpath-runtime` is not a troll.

- `bus-troll` is first-class and individually installable.
- `bus-troll` is required only for deployments that use a managed bus boundary.
- Bus bypass remains valid when declared by deployment constraints.
- The existence of `bus-troll` does not make bus traversal mandatory.

## Python Doctrine
Python must not be required for remote native hot-path execution or the administrative spine.
Optional isolated Python habitats may exist only when explicitly declared by a module or tool.

## License

Symphony is licensed under the GNU Affero General Public License v3.0 only.
SPDX-License-Identifier: AGPL-3.0-only.
Commercial licensing options are being prepared for agents, professional users, institutions, and funds.
Cloud providers or infrastructure vendors wishing to offer Symphony as closed-source SaaS/PaaS should contact QuanuX directly at licensing@quanux.org for custom agreements.
Without a separate written agreement, use, modification, distribution, and network deployment are governed by AGPL-3.0-only.
