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
./build/symphony-validator check --repo /path/to/symphony --json
./build/symphony-validator root-summary --repo /path/to/symphony --json
```

The complete check includes implemented-module admission and invariant-ownership assurance. A discovered regression name is traceability evidence, not proof that its suite ran; use the build and test commands below to execute the applicable validator tests. The executable is deliberately read-only: `symphony-validator apply` returns stable invalid-usage status and no validation or projection command writes repository content.

## Smoke Tests

```bash
cd tools/symphony-validator
./tests/smoke.sh
```

## Exact Versioned Installation

Configure and install into any administrator-selected prefix. Versions coexist without an unversioned alias:

```bash
cmake -S tools/symphony-validator -B tools/symphony-validator/build \
  -DCMAKE_INSTALL_PREFIX=/chosen/prefix
cmake --build tools/symphony-validator/build
cmake --install tools/symphony-validator/build
```

The exact executable is `/chosen/prefix/libexec/symphony/symphony-validator/0.1.0-dev/symphony-validator`. Its immutable nine-file receipt is beneath `/chosen/prefix/share/symphony/receipts/`. Installation creates no service, listener, login hook, Maestro presence, or active binding.

The configured build tree retains the exact prefix. Uninstall only receipt-owned files:

```bash
cmake --build tools/symphony-validator/build --target uninstall-symphony-validator
```

Protected qxctl validation profiles and baselines are installation-external state and survive executable uninstall.

The receipt is immutable for one exact module/version path. A repeated install fails before any owned-file install rule; install another version side by side or run the receipt-verified uninstaller first. Uninstall validates the configured ownership set and every remaining file's recorded size and SHA-256, removes the receipt last, and treats already-missing owned files only as idempotent retry evidence.

## Development Posture
Invoked directly during development or through `qxctl validate` after exact receipt validation.

## Intended Production Posture
The administrative spine invokes the same deterministic checker. CI integration remains separate.

## Python Doctrine
Python must not be required for remote native hot-path execution or the administrative spine.
Optional isolated Python habitats may exist only when explicitly declared by a module or tool.
Choosing C++ for the validator does not ban optional isolated Python habitats.
It prevents Python from becoming required validator infrastructure for the administrative spine.
