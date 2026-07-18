# Symphony Validator Installation

****

## Requirements

The Symphony Validator requires:
* C++26 compatible toolchain
* CMake 3.25+
* On macOS: Xcode Command Line Tools or full Xcode with a C++26-capable compiler

## macOS Build Instructions

The current source has been built and smoke-tested with AppleClang 21 from Xcode. Use the native toolchain first:

```bash
cmake -S tools/symphony-validator -B tools/symphony-validator/build
cmake --build tools/symphony-validator/build
```

If an older installed AppleClang cannot satisfy the C++26 build contract, install a current Homebrew LLVM, LLD, and CMake toolchain:

```bash
brew install llvm lld cmake
```

Then configure and build using that compiler and the explicit LLD Mach-O linker:

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
