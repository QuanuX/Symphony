# SKVI Engine Manifest

## Canonical Surfaces

- `modules/skvi-engine/FEATURES.md`
- `modules/skvi-engine/INSTALL.md`
- `modules/skvi-engine/INTENT.md`
- `modules/skvi-engine/MANIFEST.md`
- `modules/skvi-engine/SKILL.md`
- `modules/skvi-engine/SPEC.md`

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
- install receipt: `symphony.knowledge.install-receipt.v2`

## Implemented Operations

| Operation | State | Canonical mutation |
|---|---|---|
| `inspect` | implemented | no |
| `check` | implemented | no |
| `propose` | implemented | no |
| `project` | implemented | no |
| `apply` | disabled | prohibited |

## Read and Write Boundaries

The engine reads the repository-relative SKVI index, the fixed root bootstrap, owner manifests explicitly delegated by `knowledge/MANIFEST.md`, and indexed regular files. The shared authority-free parser supplies required canonical-surface closure; the engine contains no hard-coded owner list. It proposes only typed operations targeting `knowledge/skvi/INDEX.md`. It has no filesystem write route; the prospective write set exists only inside an immutable proposal.

## Installability

The executable, contracts, receipt, and licenses install beneath module-and-version-specific paths. Installation is initially observed as `installed_undocked`, creates no global alias, selects no receptor, changes no active version, and contacts no service. Mutable lifecycle state remains outside the receipt. Uninstall removes only receipt-owned files.

## Dependencies and Boundaries

The engine statically links `knowledge-vector-engine-cpp` and inherits its pinned nlohmann/json source. It has no runtime shared-library, network, Go, Python, cgo, provider, credential, SSIAG, STAV, qxctl, or Maestro dependency. It never decides membership, mutates canonical truth, or publishes a projection.
