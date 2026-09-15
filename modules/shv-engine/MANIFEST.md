# SHV engine manifest

Canonical owner: `modules/shv-engine/`. Semantic vector: `knowledge/shv/`.

Owned surfaces: INTENT.md, SPEC.md, MANIFEST.md, SKILL.md, INSTALL.md, FEATURES.md, schemas/v1/shv.schema.json, schemas/v1/shv.templates.json, src/ and tests/.

Package `shv-engine` 0.3.0-dev installs `symphony-shv` and exact receipt-owned schemas, templates, companions and licenses. C++26 owns source consumption, hardware semantics, coverage and projection; qxctl administers it. Generic graph exchange preserves this ownership. No mutable selection or daemon is installed.

The consolidated SHV gate adds provenance-bearing PDF ingestion (kernel 0.3.0-dev) and explicit kernel dependency admission (partition 0.2.0-dev), described in knowledge/shv/DOCUMENT-INGESTION.md. Earlier installations remain explicitly selectable.

## Mechanical interface release 0.4.0-dev

OWNER-INTERFACE.json declares exact metadata. INTERFACE-GENERATOR.json selects generation paths and frozen history; the shared registration-driven generator produces src/interface.generated.hpp, Go admission and CMake inventory. qxctl verifies the installed declaration through receipt-v2 and compiled admission. Existing command identities and defaults remain unchanged. Old packages remain independently selectable; artifacts and retained writers keep their exact original identity. Metadata generation is separate from native semantic handlers and independent Go result replay.

This inventory is not exhaustive. New owners can use `tools/shv-interface-codegen/EXTENDING.md` without extending a fixed generator whitelist. They must supply their own semantic contracts and qxctl integration.

The kernel retains its exact embedded PDF 0.2 reader. PDF 0.3 is independently selectable for PDF operations; it is not substituted into kernel provenance.
