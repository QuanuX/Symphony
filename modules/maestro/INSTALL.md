# Maestro Install

## Requirements

- CMake 3.25+
- a C++26-capable compiler
- Linux-first; macOS remains a development path
- no Python, cgo, container, cloud, or network requirement

## Build and Test

```bash
cmake -S modules/maestro -B build/maestro \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_INSTALL_PREFIX=/absolute/prefix
cmake --build build/maestro
ctest --test-dir build/maestro --output-on-failure
```

## Install

```bash
cmake --install build/maestro
```

Installation creates no receptor state and changes no engine binding or lifecycle profile.

## Uninstall

Use the generated `uninstall-maestro` target. It removes only receipt-owned files and preserves per-TOPS docking presence evidence.

```bash
cmake --build build/maestro --target uninstall-maestro
```

If installation used a one-off `cmake --install ... --prefix` override instead of the configured prefix, pass that exact prefix to the generated uninstall program:

```bash
cmake -DINSTALL_PREFIX=/absolute/prefix -P build/maestro/uninstall.cmake
```
