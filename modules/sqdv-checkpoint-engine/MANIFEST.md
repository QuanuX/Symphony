# SQDV checkpoint administration manifest

Module/package `sqdv-checkpoint-engine`; version `0.1.0-dev`.

## Canonical surfaces

- `modules/sqdv-checkpoint-engine/CMakeLists.txt`
- `modules/sqdv-checkpoint-engine/FEATURES.md`
- `modules/sqdv-checkpoint-engine/INSTALL.md`
- `modules/sqdv-checkpoint-engine/INTENT.md`
- `modules/sqdv-checkpoint-engine/INTERFACE-GENERATOR.json`
- `modules/sqdv-checkpoint-engine/MANIFEST.md`
- `modules/sqdv-checkpoint-engine/OWNER-INTERFACE.json`
- `modules/sqdv-checkpoint-engine/SKILL.md`
- `modules/sqdv-checkpoint-engine/SPEC.md`
- `modules/sqdv-checkpoint-engine/cmake/uninstall.cmake.in`
- `modules/sqdv-checkpoint-engine/schemas/v1/admin.schema.json`
- `modules/sqdv-checkpoint-engine/schemas/v1/admin.templates.json`
- `modules/sqdv-checkpoint-engine/src/admin.cpp`
- `modules/sqdv-checkpoint-engine/src/interface.generated.hpp`
- `modules/sqdv-checkpoint-engine/tests/fixtures/interface-history.v1.json`

## Shared implementation

- `cmake/SymphonySqvAdministration.cmake`
- `tools/sqv-administration-cpp/MANIFEST.md`
- `tests/sqv-administration/README.md`
- `modules/sqpv-inspection-engine/src/store_reader.cpp`
- `modules/sqpv-inspection-engine/src/store_reader.hpp`

- `modules/sqdv-checkpoint-engine/tests/process.cpp`

## Platform scope

The shared SQPV reader supports macOS local APFS and Ubuntu LTS Linux x86_64/ARM64, with the Linux filesystem selected by the caller and no filesystem-name allowlist. DGX Spark uses that ARM64 code path; physical Spark/DGX OS qualification remains separate and pending. Native Windows is unsupported. SPEC owns the admission and observation contract; INSTALL describes per-deployment verification. The package version, process protocol and retained formats are unchanged by this portability revision.
