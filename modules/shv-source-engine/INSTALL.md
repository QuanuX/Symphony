# Independent installation

Configure modules/shv-source-engine with CMake3.25+ and C++26. SYMPHONY_KVE_USE_INSTALLED=ON uses the installed foundation; otherwise the existing repository foundation is linked. No dependencies are downloaded. Install into an explicit prefix; executable libexec/symphony/shv-source-engine/0.1.0-dev/symphony-shv-source, resources share/symphony/schemas/shv-source-engine/0.1.0-dev/source.schema.json and source.templates.json. Receipt is written last and owned mismatched bytes are not overwritten.

uninstall-shv-source-engine uses the guarded receipt-v2 remover and preserves unrelated user data. Exact platform scope is the built receipt. No native graph database, active source head, system service or credentials are selected. Caller files remain outside package ownership.

## Exact interface metadata — 0.2.0-dev

OWNER-INTERFACE.json owns mechanical descriptor metadata, supported releases and package inventory. `tools/shv-interface-codegen/generate.cpp --owner shv-source-engine` generates src/interface.generated.hpp, the owner Go admission file and the CMake inventory. tests/fixtures/interface-history.v1.json retains original installed descriptors; the shared history lock and CTest protect exact parity. Semantic handlers and independent qxctl replay are handwritten. The package installs its receipt-owned declaration; qxctl checks it against compiled admission. All existing command routes and defaults remain unchanged.

Source 0.1 remains supported. Source 0.2 stamps its own graph identity; validation requires the original graph owner. qxctl source activation retains and replays the exact selected owner version. At this source 0.2 release, publication 0.1/0.2/0.3 retained source 0.1 admission. Publication 0.4 subsequently admits source 0.1/0.2 through its explicit composition contract; retained publisher releases keep their original admissions. See `modules/shv-publication-engine/SPEC.md` for the exact selected publisher.
