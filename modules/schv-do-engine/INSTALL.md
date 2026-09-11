# Symphony Cloud Hyperscalers Vector — DigitalOcean Engine Installation

Requires CMake 3.25+, a C++26 compiler and the Linux-first or supported macOS POSIX development path. Each module can build from the monorepo and links the authority-free `Symphony::KnowledgeVectorEngine` mechanics.

```bash
cmake -S modules/schv-do-engine -B build/schv-do-engine -DBUILD_TESTING=ON
cmake --build build/schv-do-engine
ctest --test-dir build/schv-do-engine --output-on-failure
cmake --install build/schv-do-engine --prefix /chosen/prefix
```

The exact executable is `libexec/symphony/schv-do-engine/0.3.0-dev/symphony-schv-do` beneath the selected prefix. Install receipts and contract documents use the common exact `share` layout. Installation is inactive `installed_undocked`; it neither selects a binding nor installs another domain. Use an explicit prefix, exact version and selected domain through the admitted qxctl command surface. Direct `--descriptor` returns the supported operations.

Only receipt-owned files may be removed by the module's receipt-verified uninstaller. Source stores, captured evidence, selected graphs, other versions and containing directories are not package files. An immutable version cannot be overwritten in place. No remote acquisition, provider credential, network activation or publication is part of installation.

The additive `0.3.0-dev` package can coexist with retained exact `0.1.0-dev` and `0.2.0-dev` installations. Do not rebuild changed source under the old version identity, overwrite an existing receipt, or install a latest alias. Corpus operations accept exact `0.2.0-dev` or `0.3.0-dev`; provider interpretation and connection operations require exact `0.3.0-dev`. Existing command defaults remain unchanged until the caller selects another admitted exact version.
