# SNRV engine manifest

Module `snrv-engine`, engine `symphony-snrv`, exact development release `0.1.0-dev`; process contract `symphony.knowledge.engine-process.v1`.

Canonical domain: `knowledge/snv/snrv/`. Module Contract Quad, `OWNER-INTERFACE.json`, `schemas/v1/admin.schema.json`, `schemas/v1/admin.templates.json`, native header `include/symphony/snv/snrv.hpp`, source `src/snrv.cpp`, and independent tests define this release.

The static SDK exports `Symphony::Snrv` through exact CMake package `SymphonySnrv 0.1.0`. Dependencies are exact `SymphonyKnowledgeVectorEngine 0.2.0` and `SymphonySnvCommon 0.1.0`; a source dependency build or separately installed exact SDKs are supported. Other SNV engines, parent installation, cloud account, bus client and resident service are unnecessary for supplied evidence reduction.

The installed executable, SDK, schemas, templates, metadata and Contract Quad are enumerated in the versioned receipt-v2 owned file set. Uninstall follows that set; runtime evidence is separately owned and retained.

## Canonical Surfaces

- `modules/snrv-engine/CMakeLists.txt`
- `modules/snrv-engine/FEATURES.md`
- `modules/snrv-engine/INSTALL.md`
- `modules/snrv-engine/INTENT.md`
- `modules/snrv-engine/INTERFACE-GENERATOR.json`
- `modules/snrv-engine/MANIFEST.md`
- `modules/snrv-engine/OWNER-INTERFACE.json`
- `modules/snrv-engine/SKILL.md`
- `modules/snrv-engine/SPEC.md`
- `modules/snrv-engine/include/symphony/snv/snrv.hpp`
- `modules/snrv-engine/schemas/v1/admin.schema.json`
- `modules/snrv-engine/schemas/v1/admin.templates.json`
- `modules/snrv-engine/src/interface.generated.hpp`
- `modules/snrv-engine/src/snrv.cpp`
- `modules/snrv-engine/src/validation.hpp`
- `modules/snrv-engine/tests/fixture.hpp`
- `modules/snrv-engine/tests/fixtures/interface-history.v1.json`
- `modules/snrv-engine/tests/fixtures/resources_validate.json`
- `modules/snrv-engine/tests/process.cpp`
- `modules/snrv-engine/tests/snrv_test.cpp`
