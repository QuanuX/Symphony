# SHV source engine specification 0.2.0-dev

The adopted bounded wire contract is knowledge/shv/SOURCES.md. Eight pure process operations: inspect, source_plan, source_reduce, source_status, capture_import, capture_compare, graph_project and graph_validate. C++ owns exact validation and derived artifacts; qxctl independently checks source/transition/byte correspondence and exposes every delivered operation through exact installed receipt selection.

Plans and reductions operate on caller-supplied immutable values. A digest or successful reduction is not approval, publisher authentication or proof of a protected registry commit. State activation with human or permitted-agent confirmation requires a separate qxctl storage/SSIAG adapter. The native engine does not perform activation; the existing qxctl source activation adapter supplies that separate protected workflow. Captures are explicit offline imports of verified regular-file bytes; no HTTP request, browser execution or hardware observation occurs.

Source location, source revision, byte identity, interpretation, component identity and installed Node state remain distinct. Comparison reports evidence differences without inferring changed hardware capability. Graph nodes preserve full provenance and exact source revisions, and all graph validation replays retained files. The existing portable graph adapter transports these artifacts without interpreting SHV semantics.

Strict fields, count/byte bounds, controlled URI grammar, no-follow file reads and finite history make failure explicit. Eight captures may total at most4MiB, each at most1MiB; history max32 revisions; scope max32 subjects and16 locators per revision; redirects max8. The foundation's one MiB request/four MiB response and JSON limits remain independent. Unsupported syntax requires a new explicit profile, not silent normalization or execution. Original kernel and adapter versions are unchanged.

## Exact interface metadata — 0.2.0-dev

OWNER-INTERFACE.json owns mechanical descriptor metadata, supported releases and package inventory. `tools/shv-interface-codegen/generate.py --owner shv-source-engine` generates src/interface.generated.hpp, the owner Go admission file and the CMake inventory. tests/fixtures/interface-history.v1.json retains original installed descriptors; the shared history lock and CTest protect exact parity. Semantic handlers and independent qxctl replay are handwritten. The package installs its receipt-owned declaration; qxctl checks it against compiled admission. All existing command routes and defaults remain unchanged.

Source 0.1 remains supported. Source 0.2 stamps its own graph identity; validation requires the original graph owner. qxctl source activation retains and replays the exact selected owner version. Downstream publication source-member admission remains 0.1; this release does not broaden that composition.
