# SQPV inspection administration manifest

Module/package `sqpv-inspection-engine`; version `0.1.0-dev`.

## Canonical surfaces

- `modules/sqpv-inspection-engine/CMakeLists.txt`
- `modules/sqpv-inspection-engine/FEATURES.md`
- `modules/sqpv-inspection-engine/INSTALL.md`
- `modules/sqpv-inspection-engine/INTENT.md`
- `modules/sqpv-inspection-engine/INTERFACE-GENERATOR.json`
- `modules/sqpv-inspection-engine/MANIFEST.md`
- `modules/sqpv-inspection-engine/OWNER-INTERFACE.json`
- `modules/sqpv-inspection-engine/SKILL.md`
- `modules/sqpv-inspection-engine/SPEC.md`
- `modules/sqpv-inspection-engine/cmake/uninstall.cmake.in`
- `modules/sqpv-inspection-engine/schemas/v1/admin.schema.json`
- `modules/sqpv-inspection-engine/schemas/v1/admin.templates.json`
- `modules/sqpv-inspection-engine/src/admin.cpp`
- `modules/sqpv-inspection-engine/src/interface.generated.hpp`
- `modules/sqpv-inspection-engine/src/store_reader.cpp`
- `modules/sqpv-inspection-engine/src/store_reader.hpp`
- `modules/sqpv-inspection-engine/tests/fixtures/interface-history.v1.json`

## Shared implementation

- `cmake/SymphonySqvAdministration.cmake`
- `tools/sqv-administration-cpp/MANIFEST.md`
- `tests/sqv-administration/README.md`

- `modules/sqpv-inspection-engine/tests/process.cpp`

## Platform scope

The shared SQPV reader supports macOS local APFS and Ubuntu LTS Linux x86_64/ARM64, with the Linux filesystem selected by the caller and no filesystem-name allowlist. DGX Spark uses that ARM64 code path; physical Spark/DGX OS qualification remains separate and pending. Native Windows is unsupported. SPEC owns the admission and observation contract; INSTALL describes per-deployment verification. The package version, process protocol and retained formats are unchanged by this portability revision.
