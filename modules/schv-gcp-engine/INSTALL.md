# Symphony Cloud Hyperscalers Vector — Google Cloud Engine Installation

Requires CMake 3.25+, a C++26 compiler and the Linux-first or supported macOS POSIX development path. Each module can build from the monorepo and links the authority-free `Symphony::KnowledgeVectorEngine` mechanics.

```bash
cmake -S modules/schv-gcp-engine -B build/schv-gcp-engine -DBUILD_TESTING=ON
cmake --build build/schv-gcp-engine
ctest --test-dir build/schv-gcp-engine --output-on-failure
cmake --install build/schv-gcp-engine --prefix /chosen/prefix
```

The exact executable is `libexec/symphony/schv-gcp-engine/0.4.0-dev/symphony-schv-gcp` beneath the selected prefix. Install receipts and contract documents use the common exact `share` layout. Installation is inactive `installed_undocked`; it neither selects a binding nor installs another domain. Use an explicit prefix, exact version and selected domain through the admitted qxctl command surface. Direct `--descriptor` returns the supported operations.

Only receipt-owned files may be removed by the module's receipt-verified uninstaller. Source stores, captured evidence, selected graphs, other versions and containing directories are not package files. An immutable version cannot be overwritten in place. No remote acquisition, provider credential, network activation or publication is part of installation.

The additive `0.3.0-dev` package can coexist with retained exact `0.1.0-dev` and `0.2.0-dev` installations. Do not rebuild changed source under the old version identity, overwrite an existing receipt, or install a latest alias. Corpus operations accept exact `0.2.0-dev` or `0.3.0-dev`; provider interpretation and connection operations require exact `0.3.0-dev`. Existing command defaults remain unchanged until the caller selects another admitted exact version.

The additive `0.4.0-dev` installation includes receipt-owned SCV schema discovery documents and templates. Use explicit `.4` for profile preparation and schema discovery. Existing operations may explicitly select `.4`; their earlier defaults and installed versions are preserved. Do not overwrite an existing exact-version receipt with rebuilt bytes.
