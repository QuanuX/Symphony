# SHV partition engine — SPEC.md

Canonical semantics: `knowledge/shv/PARTITIONS.md`, installed alongside this document as PARTITIONS.md. Independent C++26 package shv-partition-engine0.2.0-dev, receipt-v2 entry point symphony-shv-partition. Four pure bounded operations; qxctl administers every operation and receipt-owned discovery. Direct inputs are caller-declared dependency references, not authenticated hardware evidence.

Build: `cmake -S modules/shv-partition-engine -B <build> -DCMAKE_INSTALL_PREFIX=<absolute-prefix>`, then `cmake --build <build>` and `cmake --install <build>`. Guarded uninstall target: uninstall-shv-partition-engine. Source ownership: src/, schemas/v1/, tests/, CMakeLists.txt, cmake/ and these contract documents. No network, registry publication, hardware mutation, graph vendor or dynamic loader is installed.

The consolidated SHV gate adds provenance-bearing PDF ingestion (kernel 0.3.0-dev) and explicit kernel dependency admission (partition 0.2.0-dev), described in knowledge/shv/DOCUMENT-INGESTION.md. Earlier installations remain explicitly selectable.

## Mechanical interface release 0.3.0-dev

OWNER-INTERFACE.json declares exact metadata. INTERFACE-GENERATOR.json selects generation paths and frozen history; the shared registration-driven generator produces src/interface.generated.hpp, Go admission and CMake inventory. qxctl verifies the installed declaration through receipt-v2 and compiled admission. Existing command identities and defaults remain unchanged. Old packages remain independently selectable; artifacts and retained writers keep their exact original identity. Metadata generation is separate from native semantic handlers and independent Go result replay.

This inventory is not exhaustive. New owners can use `tools/shv-interface-codegen/EXTENDING.md` without extending a fixed generator whitelist. They must supply their own semantic contracts and qxctl integration.

Dependency admission remains source 0.1 and kernel 0.1/0.2/0.3. The new standalone partition identity does not silently widen those inputs. Publication retains its explicit embedded partition 0.2 reader.
