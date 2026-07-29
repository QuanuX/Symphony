# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/knowledge-session-coordinator/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "not_applicable",
          "reason": "The current installed-undocked version creates no Maestro receptor, binding, activation, or persisted deployment state.",
          "reference": null,
          "vector": "maestro"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to the coordinator contract and implementation.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the coordinator contracts and build surface.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "not_applicable",
          "reason": "The current read-only slice does not authenticate a caller, request an SSIAG decision, or establish an authority epoch.",
          "reference": null,
          "vector": "ssiag"
        },
        {
          "applicability": "not_applicable",
          "reason": "The current read-only slice emits no STAV event and has no append-authority producer grant.",
          "reference": null,
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "The shared foundation is a statically linked authority-free library with no executable; this feature is an independently installed process with a bounded descriptor and read-only operations.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation"
        }
      ],
      "evidence": [
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp verifies descriptor truth, read-only snapshot comparison, unsafe-path rejection, invalid digest rejection, and reserved-operation failure.",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh verifies deterministic process responses, explicit disabled authority, bounded check output, duplicate-key rejection, and reserved begin failure.",
        "modules/knowledge-session-coordinator/CMakeLists.txt builds, tests, installs, receipts, and uninstalls the exact versioned process.",
        "modules/knowledge-session-coordinator/SPEC.md defines the implemented inspect/check behavior and deferred authenticated-session boundary."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "how": "The C++26 symphony-knowledge-session process statically links the shared foundation, accepts the common bounded process envelope, reports an exact descriptor, and hashes caller-supplied safe relative regular-file paths without following symlinks or returning file bodies.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the independently installed process, descriptor, inspect operation, bounded read-only snapshot check, and stable failure behavior."
        },
        {
          "language": "CMake",
          "role": "Builds and installs the exact versioned executable, module contracts, receipt, tests, and receipt-owned uninstall surface."
        }
      ],
      "implementation_paths": [
        "modules/knowledge-session-coordinator/CMakeLists.txt",
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/src/coordinator.hpp",
        "modules/knowledge-session-coordinator/src/main.cpp",
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not authenticate a caller, establish or recover a session, create a journal, take a writer lock, run a watcher, or invoke a vector engine.",
        "Does not implement begin, status, checkpoint, close, recover, or apply; those operations remain reserved or disabled.",
        "Does not integrate qxctl, SSIAG, STAV, or Maestro, and does not mutate canonical repository state.",
        "Does not claim system or TOPS provisioning, active docking, or a published module release."
      ],
      "owner_contract": "modules/knowledge-session-coordinator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The coordinator statically links and relies on the shared process, digest, path, and snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/knowledge-session-coordinator",
      "status": "experimental",
      "title": "Read-only knowledge-session coordination foundation",
      "what": "Provides an independently installable administrative process that reports its exact capability state and computes deterministic read-only snapshots over explicit repository paths.",
      "when": "Runs only on explicit user-scope process invocation under a bounded deadline. It does not currently begin or supervise an authenticated knowledge session.",
      "where": "Executes as the inactive, installed-undocked symphony-knowledge-session C++ process in the administrative freezing path and roots checks at its current repository working directory.",
      "who": "Callers, maintainers, tests, and future qxctl, SSIAG, STAV, and vector-engine integrations that need a domain-neutral knowledge-session process boundary.",
      "why": "Provides a safe, independently versioned landing zone for future authenticated session coordination while making current read-only repository-state evidence explicit and testable."
    }
  ],
  "source_scope": "modules/knowledge-session-coordinator"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
