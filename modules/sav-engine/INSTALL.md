# SAV Engine Installation

Configure and build with CMake 3.25+ and a C++26-capable compiler. Installation uses the requested prefix and creates `share/symphony/receipts/sav-engine/0.1.0-dev/install-receipt.json`. It neither selects nor docks the installed version. Use the generated `uninstall-sav-engine` target to remove only verified receipt-owned files.

Installed dependency mode (`-DSYMPHONY_KVE_USE_INSTALLED=ON`) requires the exact `SymphonyKnowledgeVectorEngine` CMake package `0.2.0` from the independently installed `knowledge-vector-engine-cpp` package `0.2.0-dev`. Select its explicit configuration directory with `SymphonyKnowledgeVectorEngine_DIR`; a missing or different version fails configuration without adding foundation source. The module's own release and protocol versions are unchanged.
