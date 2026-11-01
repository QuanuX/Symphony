# SEV Engine Installation

Build with CMake 3.25+ and a C++26-capable compiler. Installation creates an exact receipt beneath `share/symphony/receipts/sev-engine/0.1.0-dev/` and leaves the version inactive and undocked. The generated uninstall target removes only receipt-owned files after common receipt verification.

Installed dependency mode (`-DSYMPHONY_KVE_USE_INSTALLED=ON`) requires the exact `SymphonyKnowledgeVectorEngine` CMake package `0.2.0` from the independently installed `knowledge-vector-engine-cpp` package `0.2.0-dev`. Select its explicit configuration directory with `SymphonyKnowledgeVectorEngine_DIR`; a missing or different version fails configuration without adding foundation source. The module's own release and protocol versions are unchanged.
