# SCIV engine manifest

Package `sciv-engine`, engine `symphony-sciv`, semantic release `0.1.0-dev`, CMake package `SymphonySciv` 0.1.0, static target `Symphony::Sciv`. Proprietary implementation is C++26 with extensions disabled. Shared dependencies are exact `SymphonySnvCommon` 0.1.0 and `SymphonyKnowledgeVectorEngine` 0.2.0.

The installed versioned executable, library, public header, schemas, templates, Contract Quad and exact CMake exports are receipt-v2 owned files. Parent installation, a bus client, network access, provider accounts, Maestro and an observer are unnecessary for supplied-evidence use. Root metadata registration and qxctl bind these advertised native operations without implementing graph semantics in Go.

| Operation | Input | Output | Effect |
|---|---|---|---|
| `sciv_validate` | `symphony.snv.sciv.evidence.v1` | `symphony.snv.sciv.result.v1` | Pure bounded interpretation |
| `sciv_transition` | `symphony.snv.sciv.transition.v1` | `symphony.snv.sciv.transition-result.v1` | Digest-bound candidate; no persistence |

`source_digest` binds the exact input's deterministic nlohmann JSON `dump()` encoding. It is evidence correspondence, never physical proof. Results preserve original source-qualified relationship history, the exact owner release and distinct subject references. Source identity, revision and method version are not inferred from record IDs.

Native tests cover independently expected cluster conditions, directed/transitive profiles, contradictory and incomplete evidence, interrupted connectivity, history, malformed input, capacity and candidate replay. Platform/release integration evidence belongs to the execution packet and is recorded at its actual gate.

## Canonical Surfaces

- `modules/sciv-engine/CMakeLists.txt`
- `modules/sciv-engine/FEATURES.md`
- `modules/sciv-engine/INSTALL.md`
- `modules/sciv-engine/INTENT.md`
- `modules/sciv-engine/INTERFACE-GENERATOR.json`
- `modules/sciv-engine/MANIFEST.md`
- `modules/sciv-engine/OWNER-INTERFACE.json`
- `modules/sciv-engine/SKILL.md`
- `modules/sciv-engine/SPEC.md`
- `modules/sciv-engine/include/symphony/snv/sciv.hpp`
- `modules/sciv-engine/schemas/v1/sciv-evidence.schema.json`
- `modules/sciv-engine/schemas/v1/sciv-evidence.template.json`
- `modules/sciv-engine/schemas/v1/sciv-result.schema.json`
- `modules/sciv-engine/schemas/v1/sciv-transition-result.schema.json`
- `modules/sciv-engine/schemas/v1/sciv-transition.schema.json`
- `modules/sciv-engine/schemas/v1/sciv-transition.template.json`
- `modules/sciv-engine/src/interface.generated.hpp`
- `modules/sciv-engine/src/sciv.cpp`
- `modules/sciv-engine/tests/fixtures/interface-history.v1.json`
- `modules/sciv-engine/tests/process.cpp`
- `modules/sciv-engine/tests/sciv_test.cpp`
