# SNIV engine manifest

Module `sniv-engine`, engine `symphony-sniv`, exact development release `0.1.0-dev`; process contract `symphony.knowledge.engine-process.v1`.

Canonical domain: `knowledge/snv/sniv/`. Module Contract Quad, `OWNER-INTERFACE.json`, `schemas/v1/admin.schema.json`, `schemas/v1/admin.templates.json`, native header `include/symphony/snv/sniv.hpp`, source `src/sniv.cpp`, and independent tests define this release.

The static SDK exports `Symphony::Sniv` through exact CMake package `SymphonySniv 0.1.0`. Dependencies are exact `SymphonyKnowledgeVectorEngine 0.2.0` and `SymphonySnvCommon 0.1.0`; a source dependency build or separately installed exact SDKs are supported. Other SNV engines, parent installation, cloud account, bus client and resident service are unnecessary for supplied evidence reduction.

The installed executable, SDK, schemas, templates, metadata and Contract Quad are enumerated in the versioned receipt-v2 owned file set. Uninstall follows that set; runtime evidence is separately owned and retained.

## Canonical Surfaces

- `modules/sniv-engine/CMakeLists.txt`
- `modules/sniv-engine/FEATURES.md`
- `modules/sniv-engine/INSTALL.md`
- `modules/sniv-engine/INTENT.md`
- `modules/sniv-engine/INTERFACE-GENERATOR.json`
- `modules/sniv-engine/MANIFEST.md`
- `modules/sniv-engine/OWNER-INTERFACE.json`
- `modules/sniv-engine/SKILL.md`
- `modules/sniv-engine/SPEC.md`
- `modules/sniv-engine/include/symphony/snv/sniv.hpp`
- `modules/sniv-engine/schemas/v1/admin.schema.json`
- `modules/sniv-engine/schemas/v1/admin.templates.json`
- `modules/sniv-engine/src/interface.generated.hpp`
- `modules/sniv-engine/src/sniv.cpp`
- `modules/sniv-engine/src/validation.hpp`
- `modules/sniv-engine/tests/fixture.hpp`
- `modules/sniv-engine/tests/fixtures/identity_validate.json`
- `modules/sniv-engine/tests/fixtures/interface-history.v1.json`
- `modules/sniv-engine/tests/process.cpp`
- `modules/sniv-engine/tests/sniv_test.cpp`
