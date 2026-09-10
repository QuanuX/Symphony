# Symphony Cloud Hyperscalers Vector — AWS Engine Installation

Requires CMake 3.25+, a C++26 compiler and the Linux-first or supported macOS POSIX development path. Each module can build from the monorepo and links the authority-free `Symphony::KnowledgeVectorEngine` mechanics.

```bash
cmake -S modules/schv-aws-engine -B build/schv-aws-engine -DBUILD_TESTING=ON
cmake --build build/schv-aws-engine
ctest --test-dir build/schv-aws-engine --output-on-failure
cmake --install build/schv-aws-engine --prefix /chosen/prefix
```

The exact executable is `libexec/symphony/schv-aws-engine/0.1.0-dev/symphony-schv-aws` beneath the selected prefix. Install receipts and contract documents use the common exact `share` layout. Installation is inactive `installed_undocked`; it neither selects a binding nor installs another domain. Use an explicit prefix, exact version and selected domain through the admitted qxctl command surface. Direct `--descriptor` returns the supported operations.

Only receipt-owned files may be removed by the module's receipt-verified uninstaller. Source stores, captured evidence, selected graphs, other versions and containing directories are not package files. An immutable version cannot be overwritten in place. No remote acquisition, provider credential, network activation or publication is part of installation.
