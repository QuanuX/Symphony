# SCNV engine manifest

Version: `0.1.0-dev`. Engine: `symphony-scnv`. Vector: `scnv`. Package: `scnv-engine`.

Canonical domain ownership is `knowledge/snv/scnv/`. This module owns the exact v1 executable and SDK contracts described in `SPEC.md`, the public header `include/symphony/snv/scnv.hpp`, pure reducers, installed process, schemas, templates and owner regression tests.

Native operations are `names_validate` and `names_resolve`; their exact input/output protocols are `scnv.names.v1`/`scnv.names-result.v1` and `scnv.resolve.v1`/`scnv.resolve-result.v1`. Applicable qxctl routes are `snv names validate` and `snv names resolve`; canonical registry admission is managed by the SNV integration owner.

Dependencies: `SymphonyKnowledgeVectorEngine` 0.2.0 and neutral `SymphonySnvCommon` 0.1.0. Both are statically compiled for process execution. The independently installed SDK target is `Symphony::Scnv`; installed SDK consumers declare those exact dependencies. No independently installed semantic child, SNV parent, resident service, provider adapter or Maestro is required.

Private Node records are caller evidence, never public source-contract inventory. The module performs no file/network writes. Transition output is an explicit proposal for separately governed retention/selection.

## Canonical Surfaces

- `modules/scnv-engine/CMakeLists.txt`
- `modules/scnv-engine/FEATURES.md`
- `modules/scnv-engine/INSTALL.md`
- `modules/scnv-engine/INTENT.md`
- `modules/scnv-engine/INTERFACE-GENERATOR.json`
- `modules/scnv-engine/MANIFEST.md`
- `modules/scnv-engine/OWNER-INTERFACE.json`
- `modules/scnv-engine/SKILL.md`
- `modules/scnv-engine/SPEC.md`
- `modules/scnv-engine/include/symphony/snv/scnv.hpp`
- `modules/scnv-engine/schemas/v1/admin.templates.json`
- `modules/scnv-engine/schemas/v1/names-result.schema.json`
- `modules/scnv-engine/schemas/v1/names.schema.json`
- `modules/scnv-engine/schemas/v1/resolve-result.schema.json`
- `modules/scnv-engine/schemas/v1/resolve.schema.json`
- `modules/scnv-engine/src/interface.generated.hpp`
- `modules/scnv-engine/src/scnv.cpp`
- `modules/scnv-engine/tests/fixtures/interface-history.v1.json`
- `modules/scnv-engine/tests/fixtures/names_assign.json`
- `modules/scnv-engine/tests/fixtures/names_resolve.json`
- `modules/scnv-engine/tests/fixtures/names_validate.json`
- `modules/scnv-engine/tests/process.cpp`
- `modules/scnv-engine/tests/scnv_test.cpp`
