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

Use `-DSYMPHONY_KVE_USE_INSTALLED=ON -DCMAKE_PREFIX_PATH=/foundation/prefix` to test against an installed foundation package.

## Install

```bash
cmake --install build/sclv-engine --prefix /chosen/prefix
```

The executables install below `libexec/symphony/sclv-engine/0.1.0-dev/`. Installation is initially observed as inactive and undocked; mutable selection and receptor state remain outside the receipt. No default prefix, active version, hook, journal, or Maestro receptor is selected.

qxctl invokes only an exact installed engine or adapter after validating the full receipt-owned package:

```bash
qxctl sclv inspect --prefix /chosen/prefix --version 0.1.0-dev
qxctl sclv check --prefix /chosen/prefix --version 0.1.0-dev
qxctl sclv project --prefix /chosen/prefix --version 0.1.0-dev
qxctl sclv evidence local-git --prefix /chosen/prefix --version 0.1.0-dev --input /path/to/local-git-input.json
qxctl sclv evidence airgap --prefix /chosen/prefix --version 0.1.0-dev --input /path/to/airgap-input.json
```

The evidence commands require receipt v2 and the exact typed adapter entry point. qxctl validates the normalized envelope and its digest but does not treat successful normalization as truth, permission, ratification, or canonical acceptance.

## Uninstall

```bash
cmake -DINSTALL_PREFIX=/chosen/prefix -P build/sclv-engine/uninstall.cmake
```

Only the files named by the immutable package receipt are removed.

The receipt is immutable for one exact module/version path. A repeated install fails before any owned-file install rule; install another version side by side or run the receipt-verified uninstaller first. Uninstall validates the configured ownership set and every remaining file's recorded size and SHA-256, removes the receipt last, and treats already-missing owned files only as idempotent retry evidence.
