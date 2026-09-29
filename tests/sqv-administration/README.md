# SQV administration verification

Configure this directory with CMake 3.30+ and C++26, build, then run CTest. It builds six exact owner executables and four focused native tests: metadata, flow, conversion and read-only storage boundaries. `sqv-admin-fixtures /absolute/fresh/private/directory` creates synthetic fixtures through the existing native SQPV/SQDV/SQAV writers. No network or credentials are used. The directory must not exist and its ancestors must satisfy the store path contract.

Install the six engines into a fresh prefix. With Go 1.26.5, run `TestSQVAdministrationInstalled` under tools/qxctl/internal/knowledgeengine with SQV_ADMIN_TEST_PREFIX and SQV_ADMIN_FIXTURES. The test verifies real native responses, receipts and resources, mutable-state refusal, full-width counters, dirty-state non-recovery, tampering, malformed inputs and observation scopes. Existing `TestSQAVInstalled` separately protects the original acquisition-request contract.

Production observation targets link only metadata, frame and process foundations. SQPV/SQDV/Databento writers, zstd and provider transport are fixture-only dependencies. Source and installed dependency builds are exact-version checked. Broad unrelated runtime suites are not part of this focused gate.
