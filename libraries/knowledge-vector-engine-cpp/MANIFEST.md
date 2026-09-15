# Knowledge Vector Engine C++ Foundation Manifest

## Canonical Surfaces

- `libraries/knowledge-vector-engine-cpp/tests/native_process_test.cpp`

- `cmake/SymphonyNativeTests.cmake`

- `libraries/knowledge-vector-engine-cpp/tests/support/native_test.hpp`

- `libraries/knowledge-vector-engine-cpp/FEATURES.md`
- `libraries/knowledge-vector-engine-cpp/INSTALL.md`
- `libraries/knowledge-vector-engine-cpp/INTENT.md`
- `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- `libraries/knowledge-vector-engine-cpp/SKILL.md`
- `libraries/knowledge-vector-engine-cpp/SPEC.md`

## Identity

- module ID: `knowledge-vector-engine-cpp`
- source path: `libraries/knowledge-vector-engine-cpp/`
- language: C++26
- development version: `0.1.0-dev`
- executable: none
- CMake target: `Symphony::KnowledgeVectorEngine`

## Implemented Components

- strict `symphony.knowledge.engine-process.v1` envelope mechanics;
- bounded UTF-8 JSON parsing with duplicate-key, trailing-data, floating-point, depth, count, and size rejection;
- stable error codes and single-response serialization;
- first-party SHA-256 and `sha256:` tagged digests;
- POSIX no-follow component traversal and bounded regular-file reads;
- deterministic sorted file snapshots;
- deterministic, no-follow parsing of the exact owner-manifest canonical-surface declaration grammar;
- canonical Gregorian civil-date, whole-second UTC, and exact-nanosecond UTC validation;
- versioned static-library, header, CMake-package, receipt, and uninstall surfaces.

## Dependency

The only third-party source is the vendored nlohmann/json `v3.12.0` single header under `third_party/nlohmann/`. The official release SHA-256 is recorded and verified in `third_party/README.md`. It is statically consumed, has no runtime download, and is not inherited by the independent `symphony-validator`.

## Installability

The library installs into versioned library and header roots. Multiple versions can coexist. Its receipt lists only package-owned files, and the generated uninstall procedure removes only those files. Engines normally statically link this foundation and do not require it as a runtime shared-library dependency.

## Boundaries

This package owns common mechanics only. `knowledge/TIME.md` owns temporal meaning; the library only implements its normalized validation profiles. It cannot create canonical Markdown, open a network listener, load arbitrary plugins, infer caller class, grant permission, perform session mutation, apply proposals, dock with Maestro, or emit STAV events.

## Native Source and Acceptance Support

The declared test support header supplies process mechanics for native acceptance programs and source-authoring tools: descriptor isolation, explicit environments, captured exit/output, deadlines, descendant cleanup and receipt checks. Its C++ regression probes are source tests. These support files do not become installed runtime entry points or semantic engines.
