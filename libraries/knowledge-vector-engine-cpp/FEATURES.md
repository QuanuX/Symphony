# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "libraries/knowledge-vector-engine-cpp/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "not_applicable",
          "reason": "The library has no runtime identity, installation activation, or receptor to persist in Maestro.",
          "reference": null,
          "vector": "maestro"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to the foundation contract and implementation.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the foundation contracts, build surface, and dependency evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of this currently unpublished development package.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The root capability governs the whole modular platform; this feature implements only reusable authority-free C++ mechanics.",
          "target_feature_id": "ssfv:symphony:platform"
        }
      ],
      "evidence": [
        "libraries/knowledge-vector-engine-cpp/tests/foundation_test.cpp exercises digest goldens, strict JSON rejection, process envelopes, deadlines, safe paths, no-follow reads, deterministic snapshots, canonical schema identities, and valid/invalid Gregorian and UTC precision cases.",
        "libraries/knowledge-vector-engine-cpp/CMakeLists.txt builds, tests, packages, receipts, and uninstalls the versioned static library.",
        "libraries/knowledge-vector-engine-cpp/third_party/README.md binds the exact vendored nlohmann/json source, checksum, license, and static-use boundary.",
        "libraries/knowledge-vector-engine-cpp/SPEC.md defines the exact implemented limits and non-authorization boundary."
      ],
      "feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
      "how": "A statically linked C++26 library performs bounded JSON parsing, exact process framing, stable errors, SHA-256 digests, safe relative-path validation, POSIX no-follow regular-file reads, deterministic file snapshots, and canonical STSC Gregorian/UTC representation validation under common limits.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded parsing, protocol framing, digests, path safety, no-follow reads, snapshots, temporal representation validation, and deterministic response mechanics."
        },
        {
          "language": "CMake",
          "role": "Builds and installs the exact versioned static-library package, CMake target, receipt, tests, and receipt-owned uninstall surface."
        }
      ],
      "implementation_paths": [
        "libraries/knowledge-vector-engine-cpp/CMakeLists.txt",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/digest.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/error.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/json.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/limits.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/path.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/protocol.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp",
        "libraries/knowledge-vector-engine-cpp/src/digest.cpp",
        "libraries/knowledge-vector-engine-cpp/src/error.cpp",
        "libraries/knowledge-vector-engine-cpp/src/path.cpp",
        "libraries/knowledge-vector-engine-cpp/src/protocol.cpp",
        "libraries/knowledge-vector-engine-cpp/src/temporal.cpp",
        "libraries/knowledge-vector-engine-cpp/tests/foundation_test.cpp",
        "libraries/knowledge-vector-engine-cpp/third_party/README.md"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not own vector semantics, architectural meaning, feature-worthiness, compatibility, publication, permission, or ratification.",
        "Does not expose an executable, network listener, dynamic plugin loader, provider client, canonical apply route, Maestro integration, SSIAG credential, or STAV writer.",
        "Does not claim a published package release or runtime shared-library dependency."
      ],
      "owner_contract": "libraries/knowledge-vector-engine-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [],
      "source_scope": "libraries/knowledge-vector-engine-cpp",
      "status": "experimental",
      "title": "Authority-free knowledge-vector engine foundation",
      "what": "Provides shared, bounded, deterministic mechanics used by independently installed Symphony knowledge-vector processes without absorbing any vector's domain authority.",
      "when": "Used at build time through static linkage and at process execution time whenever a consuming engine parses a request, validates canonical temporal text, reads bounded repository evidence, or emits a digest-bound response.",
      "where": "Owned by the versioned library under libraries/knowledge-vector-engine-cpp and embedded into consuming administrative cold/freezing-path processes.",
      "who": "Symphony knowledge-vector engines, the knowledge-session coordinator, their maintainers, packagers, tests, and callers that rely on consistent process and file-safety behavior.",
      "why": "Centralizes security-sensitive, domain-neutral mechanics so each vector process can remain independently versioned while sharing one tested protocol and evidence foundation."
    }
  ],
  "source_scope": "libraries/knowledge-vector-engine-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
