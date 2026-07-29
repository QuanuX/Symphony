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
