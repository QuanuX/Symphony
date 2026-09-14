# SHV partition engine — INTENT.md

Canonical semantics: `knowledge/shv/PARTITIONS.md`, installed alongside this document as PARTITIONS.md. Independent C++26 package shv-partition-engine0.2.0-dev, receipt-v2 entry point symphony-shv-partition. Four pure bounded operations; qxctl administers every operation and receipt-owned discovery. Direct inputs are caller-declared dependency references, not authenticated hardware evidence.

Build: `cmake -S modules/shv-partition-engine -B <build> -DCMAKE_INSTALL_PREFIX=<absolute-prefix>`, then `cmake --build <build>` and `cmake --install <build>`. Guarded uninstall target: uninstall-shv-partition-engine. Source ownership: src/, schemas/v1/, tests/, CMakeLists.txt, cmake/ and these contract documents. No network, registry publication, hardware mutation, graph vendor or dynamic loader is installed.

The consolidated SHV gate adds provenance-bearing PDF ingestion (kernel 0.3.0-dev) and explicit kernel dependency admission (partition 0.2.0-dev), described in knowledge/shv/DOCUMENT-INGESTION.md. Earlier installations remain explicitly selectable.
