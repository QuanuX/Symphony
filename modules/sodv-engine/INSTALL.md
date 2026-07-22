# SODV Engine Installation

## Requirements

- CMake 3.25 or newer
- a C++26-capable compiler
- the Symphony knowledge-vector C++ foundation, embedded from the monorepo or installed as a package

## Build, Test, and Install

```sh
cmake -S modules/sodv-engine -B /tmp/sodv-build \
  -DBUILD_TESTING=ON -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_INSTALL_PREFIX=/chosen/prefix
cmake --build /tmp/sodv-build
ctest --test-dir /tmp/sodv-build --output-on-failure
cmake --install /tmp/sodv-build
```

## Install and Uninstall

The installation is `installed_undocked`, creates no global command alias, selects no active version, contacts no service, and may coexist with other installed engine versions. Invoke the exact version through `qxctl sodv ... --prefix /chosen/prefix`.

```sh
cmake --build /tmp/sodv-build --target uninstall-sodv-engine
```

The uninstall target removes only the nine paths named by the exact receipt and preserves `knowledge/sodv/RELEASES.md`, local session/recovery evidence, other versions, and unrelated prefix content. Custom `libexec` or `share` directory names are rejected because qxctl deliberately resolves one receipt layout.
