# SHV PDF adapter manifest

## Canonical Surfaces

- `modules/shv-pdf-adapter/tests/installed_integration.cpp`

- `modules/shv-pdf-adapter/FEATURES.md`
- `modules/shv-pdf-adapter/INSTALL.md`
- `modules/shv-pdf-adapter/INTENT.md`
- `modules/shv-pdf-adapter/INTERFACE-GENERATOR.json`
- `modules/shv-pdf-adapter/MANIFEST.md`
- `modules/shv-pdf-adapter/OWNER-INTERFACE.json`
- `modules/shv-pdf-adapter/SKILL.md`
- `modules/shv-pdf-adapter/SPEC.md`
- `modules/shv-pdf-adapter/schemas/v1/pdf.schema.json`
- `modules/shv-pdf-adapter/schemas/v1/pdf.templates.json`

Module `shv-pdf-adapter`, engine `symphony-shv-pdf`, vector `shv`, version `0.3.0-dev`, kind `adapter`. C++26 native executable, statically linked engine foundation, runtime-selected external PDFium C API. Receipt-v2 installs the executable, six owner documents, PDF-ADAPTER.md, schema/templates and licenses. No PDFium binary is bundled or installed.

Version 0.2.0 adds native `graph_project` and `graph_validate`, administered by `qxctl shv pdf graph project|validate|roundtrip`. Project takes an extraction request; validate and roundtrip take `{request,graph}`. Roundtrip additionally selects `--adapter-prefix` and `--adapter-version`. See PDF-ADAPTER.md for exact provenance semantics. Retained 0.1.0 installations remain explicitly selectable for their original operations.

## Mechanical interface release 0.3.0-dev

OWNER-INTERFACE.json declares exact metadata. INTERFACE-GENERATOR.json selects generation paths and frozen history; the shared registration-driven generator produces src/interface.generated.hpp, Go admission and CMake inventory. qxctl verifies the installed declaration through receipt-v2 and compiled admission. Existing command identities and defaults remain unchanged. Old packages remain independently selectable; artifacts and retained writers keep their exact original identity. Metadata generation is separate from native semantic handlers and independent Go result replay.

This inventory is not exhaustive. New owners can use `tools/shv-interface-codegen/EXTENDING.md` without extending a fixed generator whitelist. They must supply their own semantic contracts and qxctl integration.
