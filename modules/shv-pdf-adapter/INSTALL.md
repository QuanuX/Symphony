# Installing the SHV PDF adapter

Use a CMake version and C++ compiler supporting C++26:

```sh
cmake -S modules/shv-pdf-adapter -B build/shv-pdf -DCMAKE_INSTALL_PREFIX=/absolute/selected/prefix
cmake --build build/shv-pdf
ctest --test-dir build/shv-pdf --output-on-failure
cmake --install build/shv-pdf
```

`SYMPHONY_KVE_USE_INSTALLED=ON` selects an independently installed foundation through CMake package discovery. The default builds the repository foundation. The runtime layout is `libexec/symphony/shv-pdf-adapter/0.2.0-dev/symphony-shv-pdf`; receipts bind installed files. `uninstall-shv-pdf-adapter` removes only verified owned files.

Callers supply a trusted PDFium shared library with the C API listed in `src/pdf.cpp`, explicit decoder root, relative path and SHA256. This pass verifies macOS loading with PDFium 153.0.7999.0; other platforms are unverified. The adapter uses POSIX dynamic loading and makes no Windows portability claim. Decoder replacement requires an explicit new identity. The binary is executable code, not a sandbox; its transitive platform dependencies are outside the binary hash. Uninstall never removes that external library.

Version 0.2.0 adds native `graph_project` and `graph_validate`, administered by `qxctl shv pdf graph project|validate|roundtrip`. Project takes an extraction request; validate and roundtrip take `{request,graph}`. Roundtrip additionally selects `--adapter-prefix` and `--adapter-version`. See PDF-ADAPTER.md for exact provenance semantics. Retained 0.1.0 installations remain explicitly selectable for their original operations.
