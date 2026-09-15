# SHV graph store manifest

## Canonical Surfaces

- `modules/shv-graph-duckdb-connector/tests/installed_integration.cpp`

- `modules/shv-graph-duckdb-connector/DUCKDB-PROVENANCE.json`
- `modules/shv-graph-duckdb-connector/FEATURES.md`
- `modules/shv-graph-duckdb-connector/INSTALL.md`
- `modules/shv-graph-duckdb-connector/INTENT.md`
- `modules/shv-graph-duckdb-connector/INTERFACE-GENERATOR.json`
- `modules/shv-graph-duckdb-connector/MANIFEST.md`
- `modules/shv-graph-duckdb-connector/OWNER-INTERFACE.json`
- `modules/shv-graph-duckdb-connector/SKILL.md`
- `modules/shv-graph-duckdb-connector/SPEC.md`
- `modules/shv-graph-duckdb-connector/schemas/v1/graph-store.schema.json`
- `modules/shv-graph-duckdb-connector/schemas/v1/graph-store.templates.json`

## Mechanical interface release 0.4.0-dev

OWNER-INTERFACE.json declares exact metadata. INTERFACE-GENERATOR.json selects generation paths and frozen history; the shared registration-driven generator produces src/interface.generated.hpp, Go admission and CMake inventory. qxctl verifies the installed declaration through receipt-v2 and compiled admission. Existing command identities and defaults remain unchanged. Old packages remain independently selectable; artifacts and retained writers keep their exact original identity. Metadata generation is separate from native semantic handlers and independent Go result replay.

This inventory is not exhaustive. New owners can use `tools/shv-interface-codegen/EXTENDING.md` without extending a fixed generator whitelist. They must supply their own semantic contracts and qxctl integration.

Store 0.4 can inventory retained writers 0.1/0.2/0.3/0.4; other data operations remain bound to the exact writer. Transfer planning binds its selected 0.3 or 0.4 reader. Old readers do not admit new 0.4 writers, and old publication admission remains unchanged. Embedded structural adapter stays exactly 0.1. No database migration or graph database default is introduced.
