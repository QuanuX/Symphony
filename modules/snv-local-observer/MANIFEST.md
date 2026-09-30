# SNV local observer manifest

Module `snv-local-observer`, release `0.1.0-dev`; process `symphony-snv-local-observer`; SDK `SymphonySnvLocalObserver` and `Symphony::SnvLocalObserver`. The module receipt has null vector/engine identities; its executable speaks the finite engine-process protocol with a distinct collector target. KVE `0.2.0-dev` is the only SDK dependency. No dependency on any of the five SNV semantic engines exists.

Operation `observe` is read-only and non-idempotent: repeated invocation can observe different values. The pure request/response schema is a bound for one invocation, not a subscription. Collection is Linux only; portable builds return explicit unavailable results.

## Canonical Surfaces

- `modules/snv-local-observer/CMakeLists.txt`
- `modules/snv-local-observer/FEATURES.md`
- `modules/snv-local-observer/INSTALL.md`
- `modules/snv-local-observer/INTENT.md`
- `modules/snv-local-observer/INTERFACE-GENERATOR.json`
- `modules/snv-local-observer/MANIFEST.md`
- `modules/snv-local-observer/OWNER-INTERFACE.json`
- `modules/snv-local-observer/SKILL.md`
- `modules/snv-local-observer/SPEC.md`
- `modules/snv-local-observer/include/symphony/snv/local_observer.hpp`
- `modules/snv-local-observer/schemas/v1/admin.schema.json`
- `modules/snv-local-observer/schemas/v1/admin.templates.json`
- `modules/snv-local-observer/src/interface.generated.hpp`
- `modules/snv-local-observer/src/main.cpp`
- `modules/snv-local-observer/src/observer.cpp`
- `modules/snv-local-observer/tests/fixtures/interface-history.v1.json`
- `modules/snv-local-observer/tests/fixtures/observe.json`
- `modules/snv-local-observer/tests/observer_test.cpp`
- `modules/snv-local-observer/tests/process.cpp`
- `modules/snv-local-observer/tests/fixtures/portable-unavailable-result.v1.json`
