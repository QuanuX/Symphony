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
        "Does not claim a published package release or runtime shared-library dependency.",
        "qxctl administers exact consuming engine processes and does not bind, invoke, or administer this static foundation package as an engine."
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
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to the shared process-envelope implementation and its limits.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the process schemas, public foundation interfaces, implementation, and conformance tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of the package containing this protocol implementation.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "This subfeature bounds one-shot process communication; no-follow evidence snapshots govern repository file acquisition rather than request and response envelopes.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.content-addressed-evidence-snapshots"
        }
      ],
      "evidence": [
        "libraries/knowledge-vector-engine-cpp/tests/foundation_test.cpp verifies strict JSON limits, duplicate-key and invalid-value rejection, target and deadline checks, stable errors, response digests, and newline-terminated compact output.",
        "knowledge/schemas/v1/engine-process-request.schema.json and knowledge/schemas/v1/engine-process-response.schema.json define the canonical envelopes implemented by the foundation."
      ],
      "feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
      "how": "Parses one size-bounded strict JSON request, rejects unknown or duplicate fields and unsafe scalar forms, validates target identity and a caller-supplied deadline, and emits one bounded compact success or stable-error response whose digest covers the canonical response before digest insertion.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements strict bounded JSON parsing, process-envelope validation, stable errors, deterministic response construction, and digest-bound serialization."
        }
      ],
      "implementation_paths": [
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/error.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/json.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/limits.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/protocol.hpp",
        "libraries/knowledge-vector-engine-cpp/src/error.cpp",
        "libraries/knowledge-vector-engine-cpp/src/protocol.cpp",
        "libraries/knowledge-vector-engine-cpp/tests/foundation_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not authenticate a caller, authorize an operation, define vector payload semantics, enforce a child-process lifetime from inside the child, or provide a listener.",
        "Does not grant qxctl or a consuming engine semantic, ratification, canonical-apply, publication, or docking authority."
      ],
      "owner_contract": "libraries/knowledge-vector-engine-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The common bounded process protocol enables the SCLV engine to expose deterministic local operations under the same framing and resource contract.",
          "target_feature_id": "ssfv:symphony:sclv-engine",
          "type": "enables"
        },
        {
          "rationale": "The common bounded process protocol enables the SKVI engine to expose deterministic local operations under the same framing and resource contract.",
          "target_feature_id": "ssfv:symphony:skvi-engine",
          "type": "enables"
        }
      ],
      "source_scope": "libraries/knowledge-vector-engine-cpp",
      "status": "experimental",
      "title": "Bounded deterministic engine-process protocol",
      "what": "Provides a common one-request/one-response protocol that gives independently installed knowledge processes strict resource, identity, deadline, error, and response-integrity behavior.",
      "when": "For every supported direct or qxctl-mediated invocation of a consuming coordinator or vector engine, before and after the operation-specific handler runs.",
      "where": "Statically linked into consuming C++ knowledge processes and exercised over their standard-input and standard-output boundary outside hot and warm paths.",
      "who": "Independently installed knowledge engines, the knowledge-session coordinator, qxctl's bounded process client, conformance tests, maintainers, and other exact local callers.",
      "why": "A uniform fail-closed envelope prevents each vector process from inventing incompatible parsing, framing, deadline, or evidence-integrity behavior."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to repository evidence safety and digest behavior.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the path, digest, and snapshot APIs, implementations, and conformance evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of the package containing these evidence mechanics.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "This subfeature acquires and binds repository file evidence; the bounded process protocol governs invocation envelopes and response framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol"
        }
      ],
      "evidence": [
        "libraries/knowledge-vector-engine-cpp/tests/foundation_test.cpp verifies normalized safe paths, sorted unique snapshots, content and snapshot digests, symlink refusal, special-file refusal, traversal refusal, file bounds, and deadline checks.",
        "libraries/knowledge-vector-engine-cpp/SPEC.md defines the file-descriptor-relative no-follow traversal and length-delimited snapshot-digest contract."
      ],
      "feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.content-addressed-evidence-snapshots",
      "how": "Validates normalized portable relative paths, opens the root and every path component through no-follow file-descriptor operations, accepts only regular files, performs bounded deadline-aware reads, sorts unique paths, and derives per-file and length-delimited aggregate SHA-256 evidence.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements safe relative-path validation, POSIX no-follow traversal, bounded regular-file reads, SHA-256, and deterministic snapshot construction."
        }
      ],
      "implementation_paths": [
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/digest.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/limits.hpp",
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/path.hpp",
        "libraries/knowledge-vector-engine-cpp/src/digest.cpp",
        "libraries/knowledge-vector-engine-cpp/src/path.cpp",
        "libraries/knowledge-vector-engine-cpp/tests/foundation_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not watch files, repair repository state, interpret file semantics, authorize a read, or make a snapshot canonical truth.",
        "Does not claim that a digest proves permission, ratification, publication, causality, or freshness."
      ],
      "owner_contract": "libraries/knowledge-vector-engine-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Safe content-addressed repository acquisition composes with the common bounded request and response boundary.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
          "type": "composes_with"
        },
        {
          "rationale": "Exact no-follow snapshots enable the SCLV engine to bind ledger checks and proposals to repository evidence.",
          "target_feature_id": "ssfv:symphony:sclv-engine.append-only-ledger-assurance",
          "type": "enables"
        },
        {
          "rationale": "Exact no-follow snapshots enable the SKVI engine to verify and bind canonical index targets.",
          "target_feature_id": "ssfv:symphony:skvi-engine.structural-index-assurance",
          "type": "enables"
        }
      ],
      "source_scope": "libraries/knowledge-vector-engine-cpp",
      "status": "experimental",
      "title": "No-follow content-addressed evidence snapshots",
      "what": "Produces deterministic bounded evidence for an exact set of repository regular files while failing closed on path traversal, symlinks, special files, duplicate paths, and changed or excessive inputs.",
      "when": "Whenever a consuming knowledge process must bind inspection, validation, proposal, recovery, or projection output to exact repository inputs.",
      "where": "Inside the statically linked foundation on supported POSIX development and TOPS environments, rooted at a caller-selected repository directory.",
      "who": "Knowledge-vector engines, the knowledge-session coordinator, their tests and maintainers, and callers consuming content-addressed evidence.",
      "why": "Repository truth must be acquired without symlink substitution or ambiguous path interpretation and remain reproducibly tied to the exact bytes examined."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to temporal representation conformance in the foundation.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the canonical temporal contract, public validators, implementation, and conformance tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of the package implementing these representation validators.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "This subfeature validates canonical timestamp text only; the bounded process protocol checks whether a caller-supplied deadline is currently acceptable.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol"
        }
      ],
      "evidence": [
        "libraries/knowledge-vector-engine-cpp/tests/foundation_test.cpp verifies leap years, month lengths, year bounds, exact UTC seconds and nanoseconds, and rejection of offsets, variable precision, impossible dates, leap seconds, and year zero.",
        "knowledge/TIME.md owns the canonical civil-date and UTC representation profiles implemented by this authority-free utility."
      ],
      "feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.temporal-representation-conformance",
      "how": "Parses fixed-width ASCII fields, applies real proleptic-Gregorian leap-year and month-length rules, and accepts only canonical civil dates, whole-second UTC, or exactly nine-digit nanosecond UTC representations defined by STSC.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements allocation-free Gregorian and exact UTC representation validation for statically linked consumers."
        }
      ],
      "implementation_paths": [
        "libraries/knowledge-vector-engine-cpp/include/symphony/knowledge/engine/temporal.hpp",
        "libraries/knowledge-vector-engine-cpp/src/temporal.cpp",
        "libraries/knowledge-vector-engine-cpp/tests/foundation_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not read, set, synchronize, or attest a clock; select a timezone; establish freshness or causality; or define domain lifetime, sequence, or recovery semantics.",
        "Does not create a time service, temporal engine, trading-node clock doctrine, or timestamp authority."
      ],
      "owner_contract": "libraries/knowledge-vector-engine-cpp/SPEC.md",
      "parent_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Canonical temporal representation checks compose with bounded process parsing wherever an operation accepts timestamp text.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
          "type": "composes_with"
        }
      ],
      "source_scope": "libraries/knowledge-vector-engine-cpp",
      "status": "experimental",
      "title": "Canonical temporal representation conformance",
      "what": "Determines whether civil-date and UTC timestamp text conforms exactly to Symphony's canonical cross-language temporal representation profiles.",
      "when": "When consuming knowledge processes validate caller-declared, contract-derived, journaled, proposal, expiry, or recorded timestamp fields.",
      "where": "Inside the statically linked C++ foundation under the authority of knowledge/TIME.md, independent of machine locale and timezone presentation.",
      "who": "Knowledge-vector engines, the knowledge-session coordinator, conformance tests, maintainers, and callers relying on stable temporal text across languages.",
      "why": "Exact shared representation rules prevent impossible dates, precision drift, timezone ambiguity, and cross-engine disagreement without centralizing domain time authority."
    }
  ],
  "source_scope": "libraries/knowledge-vector-engine-cpp"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
