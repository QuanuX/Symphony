# SHV profile engine specification

Owner contract: `knowledge/shv/PROFILES.md` v1. Independently installable C++26
`symphony-shv-profile` 0.2.0-dev, process protocol v1 and administration descriptor v2.
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
