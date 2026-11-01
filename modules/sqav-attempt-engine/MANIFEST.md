# SQAV attempt administration manifest

Module/package `sqav-attempt-engine`; version `0.1.0-dev`.

## Canonical surfaces

- `modules/sqav-attempt-engine/CMakeLists.txt`
- `modules/sqav-attempt-engine/FEATURES.md`
- `modules/sqav-attempt-engine/INSTALL.md`
- `modules/sqav-attempt-engine/INTENT.md`
- `modules/sqav-attempt-engine/INTERFACE-GENERATOR.json`
- `modules/sqav-attempt-engine/MANIFEST.md`
- `modules/sqav-attempt-engine/OWNER-INTERFACE.json`
- `modules/sqav-attempt-engine/SKILL.md`
- `modules/sqav-attempt-engine/SPEC.md`
- `modules/sqav-attempt-engine/cmake/uninstall.cmake.in`
- `modules/sqav-attempt-engine/schemas/v1/admin.schema.json`
- `modules/sqav-attempt-engine/schemas/v1/admin.templates.json`
- `modules/sqav-attempt-engine/src/admin.cpp`
- `modules/sqav-attempt-engine/src/interface.generated.hpp`
- `modules/sqav-attempt-engine/tests/fixtures/interface-history.v1.json`

## Shared implementation

- `cmake/SymphonySqvAdministration.cmake`
- `tools/sqv-administration-cpp/MANIFEST.md`
- `tests/sqv-administration/README.md`
- `modules/sqpv-inspection-engine/src/store_reader.cpp`
- `modules/sqpv-inspection-engine/src/store_reader.hpp`

- `modules/sqav-attempt-engine/tests/process.cpp`

## Platform scope

The shared SQPV reader supports macOS local APFS and Ubuntu LTS Linux x86_64/ARM64, with the Linux filesystem selected by the caller and no filesystem-name allowlist. DGX Spark uses that ARM64 code path; physical Spark/DGX OS qualification remains separate and pending. Native Windows is unsupported. SPEC owns the admission and observation contract; INSTALL describes per-deployment verification. The package version, process protocol and retained formats are unchanged by this portability revision.
