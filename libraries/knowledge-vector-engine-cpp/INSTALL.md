# Knowledge Vector Engine C++ Foundation Installation

## Requirements

- CMake 3.25 or newer
- a C++26-capable compiler
- a single-configuration CMake generator, such as Unix Makefiles or Ninja
- POSIX file-descriptor APIs on the supported Linux-first path or macOS development path

No network access is required after source checkout. The JSON dependency is vendored and checksum-recorded.
Multi-configuration generators are rejected in this development version because the deterministic package receipt must name the exact configuration-specific CMake export file.

## Build and Test

```bash
cmake -S libraries/knowledge-vector-engine-cpp -B build/knowledge-vector-engine-cpp -DBUILD_TESTING=ON
cmake --build build/knowledge-vector-engine-cpp
ctest --test-dir build/knowledge-vector-engine-cpp --output-on-failure
```

## Install

Use a caller-selected prefix. Installation does not activate an engine, edit a repository, install hooks, or contact Maestro.

```bash
cmake --install build/knowledge-vector-engine-cpp --prefix /chosen/prefix
```

Headers, the static archive, CMake package metadata, licenses, contracts, and immutable content-addressed `symphony.knowledge.install-receipt.v2` receipt are installed under versioned paths. The receipt is generated last from the installed regular-file content and excludes itself from ownership hashing.

## Use from Another Build

```bash
cmake -S modules/knowledge-session-coordinator \
  -B build/knowledge-session-coordinator \
  -DSYMPHONY_KVE_USE_INSTALLED=ON \
  -DCMAKE_PREFIX_PATH=/chosen/prefix
```

## Uninstall

The build-local uninstall target removes only the exact files in this package's versioned receipt model:

```bash
cmake -DINSTALL_PREFIX=/chosen/prefix \
  -P build/knowledge-vector-engine-cpp/uninstall.cmake
```

It does not remove canonical knowledge, engine state, other versions, or directories it does not own. The protected qxctl lifecycle circuit is the separate operational administration path for staged receipt-v2 package convergence.

The receipt is immutable for one exact module/version path. A repeated install fails before any owned-file install rule; install another version side by side or run this receipt-verified uninstaller first. Uninstall verifies the configured ownership set plus every remaining file's recorded size and SHA-256 before removal, removes the receipt last, and accepts already-missing owned files only as idempotent retry evidence.
