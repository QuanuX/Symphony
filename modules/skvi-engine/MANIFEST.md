# SKVI Engine Manifest

## Identity

- module ID: `skvi-engine`
- source path: `modules/skvi-engine/`
- executable: `symphony-skvi`
- engine ID: `symphony-skvi`
- vector ID: `skvi`
- language: C++26
- development version: `0.1.0-dev`
- thermal placement: administrative freezing path

## Protocols

- process: `symphony.knowledge.engine-process.v1`
- descriptor: `symphony.knowledge.engine-descriptor.v1`
- proposal: `symphony.knowledge.proposal.v1`
- check result: `symphony.skvi.check-result.v1`
- projection: `symphony.skvi.projection.v1`
- install receipt: `symphony.knowledge.install-receipt.v1`

## Implemented Operations

| Operation | State | Canonical mutation |
|---|---|---|
| `inspect` | implemented | no |
| `check` | implemented | no |
| `propose` | implemented | no |
| `project` | implemented | no |
| `apply` | disabled | prohibited |

## Read and Write Boundaries

The engine reads the repository-relative SKVI index, SKVI contracts, and indexed regular files. It proposes only typed operations targeting `knowledge/skvi/INDEX.md`. It has no filesystem write route; the prospective write set exists only inside an immutable proposal.

## Installability

The executable, contracts, receipt, and licenses install beneath module-and-version-specific paths. Installation reports `installed_undocked`, creates no global alias, selects no receptor, changes no active version, and contacts no service. Uninstall removes only receipt-owned files.

## Dependencies and Boundaries

The engine statically links `knowledge-vector-engine-cpp` and inherits its pinned nlohmann/json source. It has no runtime shared-library, network, Go, Python, cgo, provider, credential, SSIAG, STAV, qxctl, or Maestro dependency. It never decides membership, mutates canonical truth, or publishes a projection.
