# SACV Engine Manifest

## Identity

- module ID: `sacv-engine`
- source path: `modules/sacv-engine/`
- executable and engine ID: `symphony-sacv`
- vector ID: `sacv`
- language: C++26
- development version: `0.1.0-dev`
- thermal placement: administrative freezing path

## Operations

`inspect`, `check`, `diff`, `propose`, and `project` are implemented without canonical mutation. `apply` is disabled.

## Package Boundary

The binary, five module documents, exact install receipt, AGPL license, and nlohmann/json license install beneath module-and-version-specific paths. Installation is inactive and undocked. Uninstall removes only receipt-owned regular files.

## Dependencies

The engine statically links the common knowledge-vector foundation. It has no network, service, runtime shared-library, Go, Python, cgo, SSIAG, STAV, Mintlify, SDK, SODV, or Maestro dependency.
