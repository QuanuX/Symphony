# SHV partition engine — INTENT.md

Canonical semantics: `knowledge/shv/PARTITIONS.md`, installed alongside this document as PARTITIONS.md. Independent C++26 package shv-partition-engine0.2.0-dev, receipt-v2 entry point symphony-shv-partition. Four pure bounded operations; qxctl administers every operation and receipt-owned discovery. Direct inputs are caller-declared dependency references, not authenticated hardware evidence.

Build: `cmake -S modules/shv-partition-engine -B <build> -DCMAKE_INSTALL_PREFIX=<absolute-prefix>`, then `cmake --build <build>` and `cmake --install <build>`. Guarded uninstall target: uninstall-shv-partition-engine. Source ownership: src/, schemas/v1/, tests/, CMakeLists.txt, cmake/ and these contract documents. No network, registry publication, hardware mutation, graph vendor or dynamic loader is installed.

The consolidated SHV gate adds provenance-bearing PDF ingestion (kernel 0.3.0-dev) and explicit kernel dependency admission (partition 0.2.0-dev), described in knowledge/shv/DOCUMENT-INGESTION.md. Earlier installations remain explicitly selectable.

## Explicit composition release 0.4.0-dev

Partition 0.4.0-dev explicitly admits source 0.1/0.2 and kernel 0.1/0.2/0.3/0.4. Historical partition 0.1/0.2/0.3 retain their prior source/kernel admission. qxctl passes the selected partition owner through build, manifest, query, materialization checkpoint and resolution replay. The embedded previous-version macro retains the partition 0.2 semantics.

Use a fresh install prefix and explicit qxctl version selection. Version 0.3 declarations remain checked against their original compiled digest. No command or default changes; metadata is not semantic authorization and future versions are not automatically admitted.
