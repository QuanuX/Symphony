# SCLV Engine Installation

## Requirements

- CMake 3.25 or newer
- a C++26-capable compiler
- Linux or the macOS development path with POSIX file-descriptor/process APIs
- `/usr/bin/git` for the local-Git evidence adapter
- exact GNU `libexec` and `share` installation directory names

## Build and Test

```bash
cmake -S modules/sclv-engine -B build/sclv-engine -DBUILD_TESTING=ON
cmake --build build/sclv-engine
ctest --test-dir build/sclv-engine --output-on-failure
```

Use `-DSYMPHONY_KVE_USE_INSTALLED=ON -DCMAKE_PREFIX_PATH=/foundation/prefix` to test against an installed shared foundation.

## Install

```bash
cmake --install build/sclv-engine --prefix /chosen/prefix
```

The executables install below `libexec/symphony/sclv-engine/0.1.0-dev/`. The receipt remains inactive and undocked. No default prefix, active version, hook, journal, or Maestro receptor is selected.

qxctl invokes only an exact installed engine after validating the full eleven-file receipt:

```bash
qxctl sclv inspect --prefix /chosen/prefix --version 0.1.0-dev
qxctl sclv check --prefix /chosen/prefix --version 0.1.0-dev
qxctl sclv project --prefix /chosen/prefix --version 0.1.0-dev
```

## Uninstall

```bash
cmake -DINSTALL_PREFIX=/chosen/prefix -P build/sclv-engine/uninstall.cmake
```

Only the eleven receipt-owned files are removed.
