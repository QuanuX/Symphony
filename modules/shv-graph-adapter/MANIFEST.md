# SHV Generic Graph Adapter Manifest

## Canonical Surfaces

- `modules/shv-graph-adapter/INTENT.md`
- `modules/shv-graph-adapter/MANIFEST.md`
- `modules/shv-graph-adapter/SPEC.md`
- `modules/shv-graph-adapter/SKILL.md`
- `modules/shv-graph-adapter/INSTALL.md`
- `modules/shv-graph-adapter/FEATURES.md`
- `modules/shv-graph-adapter/schemas/v1/graph-adapter.schema.json`
- `modules/shv-graph-adapter/schemas/v1/graph-adapter.templates.json`
- `modules/shv-graph-adapter/include/symphony/graph/adapter.hpp`

## Mechanical interface release 0.2.0-dev

OWNER-INTERFACE.json declares exact metadata. INTERFACE-GENERATOR.json selects generation paths and frozen history; the shared registration-driven generator produces src/interface.generated.hpp, Go admission and CMake inventory. qxctl verifies the installed declaration through receipt-v2 and compiled admission. Existing command identities and defaults remain unchanged. Old packages remain independently selectable; artifacts and retained writers keep their exact original identity. Metadata generation is separate from native semantic handlers and independent Go result replay.

This inventory is not exhaustive. New owners can use `tools/shv-interface-codegen/EXTENDING.md` without extending a fixed generator whitelist. They must supply their own semantic contracts and qxctl integration.
