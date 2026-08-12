# SSFV Engine Installation

## Requirements

- CMake 3.25 or newer
- a C++26-capable compiler
- the Symphony knowledge-vector C++ foundation, embedded from the monorepo or installed as an exact compatible package
- a caller-selected installation prefix owned by the invoking administrator or root and not writable by group or other

## Build, Test, and Install

```sh
cmake -S modules/ssfv-engine -B /tmp/ssfv-build \
  -DBUILD_TESTING=ON -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_INSTALL_PREFIX=/chosen/prefix
cmake --build /tmp/ssfv-build
ctest --test-dir /tmp/ssfv-build --output-on-failure
cmake --install /tmp/ssfv-build
```

## Lifecycle

Installation is `installed_undocked`, creates no global alias, selects no active version, contacts no service, and may coexist with other versions. Invoke the exact version through `qxctl ssfv ... --prefix /chosen/prefix`.

```sh
cmake --build /tmp/ssfv-build --target uninstall-ssfv-engine
```

Uninstall removes only the nine paths named by the exact receipt. It preserves canonical SSFV truth, local noncanonical evidence, other versions, and unrelated prefix content. Custom `libexec` or `share` directory names are rejected because qxctl deliberately resolves one receipt layout.

The receipt is immutable for one exact module/version path. A repeated install fails before any owned-file install rule; install another version side by side or run the receipt-verified uninstaller first. Uninstall validates the configured ownership set and every remaining file's recorded size and SHA-256, removes the receipt last, and treats already-missing owned files only as idempotent retry evidence.
