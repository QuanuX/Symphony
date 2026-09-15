# SHV interface authoring manifest

Canonical owner: tools/shv-interface-codegen/SPEC.md. Owned surfaces: INTENT.md, MANIFEST.md, SPEC.md, SKILL.md, generate.cpp, history-lock.json and tests/interface_test.cpp. The tool is repository authoring support, not an independently installed runtime engine or user capability. Module declarations and generated files remain under their respective module contracts.

## Canonical Surfaces

- `tools/shv-interface-codegen/INTENT.md`
- `tools/shv-interface-codegen/MANIFEST.md`
- `tools/shv-interface-codegen/SPEC.md`
- `tools/shv-interface-codegen/SKILL.md`
- `tools/shv-interface-codegen/generate.cpp`
- `tools/shv-interface-codegen/history-lock.json`
- `tools/shv-interface-codegen/tests/interface_test.cpp`

- `tools/shv-interface-codegen/EXTENDING.md`
- `tools/shv-interface-codegen/owner_codegen.cpp`
- `tools/shv-interface-codegen/tests/owner_test.cpp`

- `tools/authoring-cpp/CMakeLists.txt`
- `tools/authoring-cpp/authoring.hpp`
- `tools/authoring-cpp/scv_interface.hpp`
- `tools/authoring-cpp/shv_interface.hpp`
- `tools/authoring-cpp/schemas.hpp`
- `tools/authoring-cpp/test_support.hpp`
- `tools/authoring-cpp/schema_test.cpp`

## Shared Native Authoring Support

This owner declares the reusable C++ authoring build and headers used by the SCV, SHV and qxctl schema frontends. These provide mechanical generation and test support; each owner retains its input schema and semantic contract.
