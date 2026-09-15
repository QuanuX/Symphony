# SHV profile engine specification

Owner contract: `knowledge/shv/PROFILES.md` v1. Independently installable C++26
`symphony-shv-profile` 0.3.0-dev, process protocol v1 and administration descriptor v2.
Seven read-only operations: inspect, profile_compile, mapping_diagnose,
universe_build, universe_bind, extraction_diagnose and references_analyze. Native schemas and templates are receipt-owned.

The compiled kernel reader is exactly 0.3.0-dev, with PDF adapter semantics 0.2.0-dev.
A static assertion requires review before a kernel version change. This owner does
not mutate or release those existing packages. qxctl independently validates the
profile result and original-source correspondence for binding.

Mapping diagnostics cover declarations only. Universe build checks a portable
recipe; bind additionally reads the caller's exact local evidence. Neither operation
publishes a catalogue or graph, fetches a URL, grants permission, authenticates a
publisher, deletes history or asserts physical hardware compatibility.

Source diagnostics preserve individual field failures; reference analysis validates
explicit document pointers and reports supplied-graph reachability. Neither reports
a complete dependency universe or grants deletion authority. See PROFILES.md.

## Mechanical interface declaration

`OWNER-INTERFACE.json` declares the exact 0.1/0.2/0.3 release operation sets,
input/output protocols, descriptor interactions, embedded reader identity and
schema/companion inventory. `tools/generate_interface.py` produces checked-in
native descriptor metadata, Go admission/output metadata and CMake inventory.
Its `--check` mode rejects drift. Generation is an authoring step, never a runtime
Python dependency. Frozen installed 0.1/0.2 descriptors are retained under
`tests/fixtures/interface-history.v1.json`; changing their metadata fails generation.
Semantic handlers and independent Go result checks remain hand-authored.

The 0.3 package owns `share/symphony/contracts/shv-profile-engine/0.3.0-dev/OWNER-INTERFACE.json`.
Every qxctl installation inspection for 0.3 verifies that regular receipt-owned file,
its exact byte length/digest and its canonical definition against compiled admission.
Even a self-consistently resealed replacement cannot broaden support. Existing
profile inspect/schema/template and operation commands exercise this check; no
competing interface command is introduced. Earlier packages need no new file and
retain their existing operations and CLI defaults. Seven native operations and
all result schemas retain their meaning. The descriptor engine version changes;
its remaining content is checked against the frozen 0.2 descriptor.

This bounded declaration is not a third-party extension loader or a hardware-policy
schema. Caller metric extensions, optional/retired fields and source choices remain
owned by PROFILES.md. Extending declaration grammar requires an explicit authoring
contract change; that does not restrict caller-authored profiles.
