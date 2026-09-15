# INSTALL

Configure CMake with an explicit install prefix; build and install. The receipt-v2 preflight prevents replacing another installation. uninstall-shv-publication-engine removes only unchanged receipt-owned files. Foundation 0.1.0 can be selected with SYMPHONY_KVE_USE_INSTALLED=ON.

See PUBLICATION.md for the publication contract and limits.

## Exact interface metadata — 0.3.0-dev

OWNER-INTERFACE.json owns mechanical descriptor metadata, supported releases and package inventory. `tools/shv-interface-codegen/generate.py --owner shv-publication-engine` generates src/interface.generated.hpp, the owner Go admission file and the CMake inventory. tests/fixtures/interface-history.v1.json retains original installed descriptors; the shared history lock and CTest protect exact parity. Semantic handlers and independent qxctl replay are handwritten. The package installs its receipt-owned declaration; qxctl checks it against compiled admission. All existing command routes and defaults remain unchanged.

Publication 0.1/0.2 remain supported. Publication 0.3 preserves 0.2 semantics and store-writer admission 0.1/0.2/0.3. The embedded partition reader remains exactly 0.2, and source-member admission remains exactly 0.1. New source 0.2 is not silently substituted into publication. Retained publication intents replay under their original owner.
