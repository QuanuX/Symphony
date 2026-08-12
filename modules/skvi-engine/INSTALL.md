# SKVI Engine Installation

## Requirements

- CMake 3.25 or newer
- a C++26-capable compiler
- a single-configuration CMake generator when building the foundation from the monorepo
- POSIX file-descriptor APIs on the Linux-first path or macOS development path
- the exact `libexec` and `share` GNU installation-directory layout required by the current qxctl receipt resolver

## Build and Test from the Monorepo

```bash
cmake -S modules/skvi-engine -B build/skvi-engine -DBUILD_TESTING=ON
cmake --build build/skvi-engine
ctest --test-dir build/skvi-engine --output-on-failure
```

## Build Against an Installed Foundation

```bash
cmake -S modules/skvi-engine \
  -B build/skvi-engine-installed \
  -DBUILD_TESTING=ON \
  -DSYMPHONY_KVE_USE_INSTALLED=ON \
  -DCMAKE_PREFIX_PATH=/foundation/prefix
```

## Install

```bash
cmake --install build/skvi-engine --prefix /chosen/prefix
```

The versioned executable is installed at `libexec/symphony/skvi-engine/0.1.0-dev/symphony-skvi`. No unversioned alias is created. Installation is initially observed as inactive and undocked; mutable selection and receptor state remain outside the receipt.

## qxctl Development Invocation

The implemented qxctl SKVI commands require the exact installation prefix and version so they can validate the receipt and every package-owned file before execution:

```bash
qxctl skvi inspect --prefix /chosen/prefix --version 0.1.0-dev
qxctl skvi check --prefix /chosen/prefix --version 0.1.0-dev
```

This is exact-version invocation, not lifecycle activation or a default-prefix policy.

## Uninstall

```bash
cmake -DINSTALL_PREFIX=/chosen/prefix -P build/skvi-engine/uninstall.cmake
```

Only receipt-owned files are removed. Canonical knowledge, projections, proposals, other versions, and containing directories are preserved.

The receipt is immutable for one exact module/version path. A repeated install fails before any owned-file install rule; install another version side by side or run the receipt-verified uninstaller first. Uninstall validates the configured ownership set and every remaining file's recorded size and SHA-256, removes the receipt last, and treats already-missing owned files only as idempotent retry evidence.
