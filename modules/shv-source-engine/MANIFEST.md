# SHV source engine manifest

## Canonical Surfaces

- `modules/shv-source-engine/FEATURES.md`
- `modules/shv-source-engine/INSTALL.md`
- `modules/shv-source-engine/INTENT.md`
- `modules/shv-source-engine/MANIFEST.md`
- `modules/shv-source-engine/OWNER-INTERFACE.json`
- `modules/shv-source-engine/SKILL.md`
- `modules/shv-source-engine/SPEC.md`
- `modules/shv-source-engine/schemas/v1/source.schema.json`
- `modules/shv-source-engine/schemas/v1/source.templates.json`

Canonical owner: modules/shv-source-engine/SPEC.md. Semantic owner: knowledge/shv/SOURCES.md. Package shv-source-engine 0.2.0-dev, vector shv, engine symphony-shv-source, independently installed C++26 executable with exact receipt-v2 entry point. Owned surfaces: src/, tests/, scripts/, CMakeLists.txt, cmake/, schemas/v1/source.schema.json and source.templates.json, INTENT.md, MANIFEST.md, SPEC.md, SKILL.md, INSTALL.md, FEATURES.md. The source owner SOURCES.md companion is installed with the package. No source selection, permissions, acquisition network client or durable graph store is installed.

## Exact interface metadata — 0.2.0-dev

OWNER-INTERFACE.json owns mechanical descriptor metadata, supported releases and package inventory. `tools/shv-interface-codegen/generate.py --owner shv-source-engine` generates src/interface.generated.hpp, the owner Go admission file and the CMake inventory. tests/fixtures/interface-history.v1.json retains original installed descriptors; the shared history lock and CTest protect exact parity. Semantic handlers and independent qxctl replay are handwritten. The package installs its receipt-owned declaration; qxctl checks it against compiled admission. All existing command routes and defaults remain unchanged.

Source 0.1 remains supported. Source 0.2 stamps its own graph identity; validation requires the original graph owner. qxctl source activation retains and replays the exact selected owner version. At this source 0.2 release, publication 0.1/0.2/0.3 retained source 0.1 admission. Publication 0.4 subsequently admits source 0.1/0.2 through its explicit composition contract; retained publisher releases keep their original admissions. See `modules/shv-publication-engine/SPEC.md` for the exact selected publisher.
