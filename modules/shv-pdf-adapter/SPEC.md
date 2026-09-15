# SHV PDF adapter specification

The owner contract is [PDF-ADAPTER.md](../../knowledge/shv/PDF-ADAPTER.md). Native operations are `inspect` and `extract`; qxctl adds original-byte replay via `verify` and receipt-owned `schema`/`template` discovery. Exact version and install prefix are required. Only `amd-69290-table8.v1` is implemented. The result is a derivation of one source, not a second documentary lineage or a kernel assertion bundle.

Version 0.2.0 adds native `graph_project` and `graph_validate`, administered by `qxctl shv pdf graph project|validate|roundtrip`. Project takes an extraction request; validate and roundtrip take `{request,graph}`. Roundtrip additionally selects `--adapter-prefix` and `--adapter-version`. See PDF-ADAPTER.md for exact provenance semantics. Retained 0.1.0 installations remain explicitly selectable for their original operations.

## Mechanical interface release 0.3.0-dev

OWNER-INTERFACE.json declares exact metadata. INTERFACE-GENERATOR.json selects generation paths and frozen history; the shared registration-driven generator produces src/interface.generated.hpp, Go admission and CMake inventory. qxctl verifies the installed declaration through receipt-v2 and compiled admission. Existing command identities and defaults remain unchanged. Old packages remain independently selectable; artifacts and retained writers keep their exact original identity. Metadata generation is separate from native semantic handlers and independent Go result replay.

This inventory is not exhaustive. New owners can use `tools/shv-interface-codegen/EXTENDING.md` without extending a fixed generator whitelist. They must supply their own semantic contracts and qxctl integration.
