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

## Root-Level Governance Role
The root repository establishes platform invariants and guarantees modular sovereignty. It does not dictate implementation code. No implementation import has occurred yet.

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
