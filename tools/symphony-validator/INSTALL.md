# Symphony Validator Installation

****

## Requirements

The Symphony Validator requires:
* C++26 compatible toolchain
* CMake 3.20+
* On macOS: Xcode Command Line Tools (or full Xcode) with Homebrew LLVM 18+

## macOS Build Instructions

Due to limited C++26 support in AppleClang, building on macOS requires a specific LLVM toolchain from Homebrew:

```bash
brew install llvm lld cmake
```

Then configure and build using the Homebrew LLVM compiler and the explicit LLD Mach-O linker:

```bash
cd tools/symphony-validator
rm -rf build
SDKROOT="$(xcrun --show-sdk-path)" \
CXX=/usr/local/opt/llvm/bin/clang++ \
LDFLAGS="-fuse-ld=$(brew --prefix lld)/bin/ld64.lld" \
cmake -S . -B build
cmake --build build
```

## Running the Validator

```bash
./build/symphony-validator check --repo /path/to/symphony
```

## Smoke Tests

```bash
cd tools/symphony-validator
./tests/smoke.sh
```

## Installation Packaging
The validator is locally buildable as a native tool through CMake. A portable host installer and uninstall manifest remain to be implemented before distribution as an independently packaged tool.

## Development Posture
Invoked as a standalone tool during local development and preflight checks.

## Intended Production Posture
Future CI and administrative-spine integrations may invoke the same deterministic checker for structural and doctrinal verification after those integrations are separately authorized.

## Python Doctrine
Python must not be required for remote native hot-path execution or the administrative spine.
Optional isolated Python habitats may exist only when explicitly declared by a module or tool.
Choosing C++ for the validator does not ban optional isolated Python habitats.
It prevents Python from becoming required validator infrastructure for the administrative spine.
