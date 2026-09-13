# SHV Generic Graph Adapter SPEC

Version 0.1.0-dev uses engine-process.v1, descriptor.v2 and receipt.v2. Operations inspect, roundtrip and query are pure. Graph exchange protocol is symphony.graph.exchange.v1. The adapter verifies owner-artifact and graph self-seals, exact fields, sorted unique node/edge IDs, sorted unique labels and endpoint closure. It does not reperform the semantic owner or authenticate the artifact. Properties preserve arbitrary canonical JSON objects without hardware ontology.

Finite bounds: 1 MiB request, 4 MiB response, 64 JSON depth, 32,768 aggregate events, 65,536-byte strings; interoperable integral numbers only. Up to 1,024 nodes, 2,048 edges, 32 labels per node, 2,048 query IDs. IDs up to 512 bytes; edge labels and owner text up to 128 bytes. These are reference adapter limits, not atlas coverage restrictions. Overflow rejects; no truncated success. Duplicate query IDs reject. Query keeps caller ID order and missing-ID order; rows remain graph order; empty IDs selects all rows.

The public C++ port is include/symphony/graph/adapter.hpp. Implementations return sealed adapter-result.v1 records. PortableReference is in-memory and makes no persistence claim. Other drivers need their own versioned implementation and conformance evidence. The header depends on the separately installed exact foundation; the installed executable is self-contained.

Schema describes structural shapes; runtime checks cross-field correspondence, canonical subset and seals. Templates use null for unanswered graph/kind, deliberately invalid until caller supplies evidence. Roundtrip is lossless structural transport, not hardware validation. query rows refer to one full graph digest; no pagination or sharded atlas support is claimed.
