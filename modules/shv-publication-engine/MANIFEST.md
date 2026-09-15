# MANIFEST

Module shv-publication-engine, engine symphony-shv-publication, vector SHV, version 0.2.0-dev. Receipt v2 owns executable, documents, schema and templates.

See PUBLICATION.md for the publication contract and limits.

## Exact interface metadata — 0.3.0-dev

OWNER-INTERFACE.json owns mechanical descriptor metadata, supported releases and package inventory. `tools/shv-interface-codegen/generate.py --owner shv-publication-engine` generates src/interface.generated.hpp, the owner Go admission file and the CMake inventory. tests/fixtures/interface-history.v1.json retains original installed descriptors; the shared history lock and CTest protect exact parity. Semantic handlers and independent qxctl replay are handwritten. The package installs its receipt-owned declaration; qxctl checks it against compiled admission. All existing command routes and defaults remain unchanged.

Publication 0.1/0.2 remain supported. Publication 0.3 preserves 0.2 semantics and store-writer admission 0.1/0.2/0.3. The embedded partition reader remains exactly 0.2, and source-member admission remains exactly 0.1. New source 0.2 is not silently substituted into publication. Retained publication intents replay under their original owner.

## Explicit composition release 0.4.0-dev

Publication 0.4.0-dev explicitly embeds partition 0.4, admits selected partition 0.2/0.3/0.4, source 0.1/0.2, kernel 0.1/0.2/0.3/0.4 and storage writers 0.1/0.2/0.3/0.4. A partition selected as 0.2 or 0.3 cannot carry source 0.2 or kernel 0.4 dependencies. Historical publishers retain their prior admissions. Retained state transitions replay against each original publisher; aggregate history validation admits the supported union without changing those owners.

Use a fresh install prefix and explicit qxctl version selection. Version 0.3 declarations remain checked against their original compiled digest. No command or default changes; metadata is not semantic authorization and future versions are not automatically admitted.
