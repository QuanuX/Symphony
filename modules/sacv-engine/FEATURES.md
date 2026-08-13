# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sacv-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SACV owns OpenAPI contract and compatibility semantics implemented by this engine.",
          "reference": "knowledge/sacv/SPEC.md",
          "vector": "sacv"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed engine and vector changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes engine contracts, implementation, and feature ownership.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future engine release.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine interprets one vector read-only; the coordinator persists cross-session noncanonical administrative evidence and never decides vector truth.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator"
        },
        {
          "distinction": "The engine implements SACV operation semantics; qxctl selects, revalidates, invokes, and validates the exact installed process without owning SACV truth or canonical apply.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        }
      ],
      "evidence": [
        "modules/sacv-engine tests exercise the implemented operation set, deterministic results, bounds, invalid input, and disabled apply.",
        "The module CMake and receipt-v2 surfaces prove independent versioned installation and conservative uninstall."
      ],
      "feature_id": "ssfv:symphony:sacv-engine",
      "how": "The C++ process uses the shared bounded engine envelope, no-follow reads, canonical digests, deterministic ordering, strict schemas, explicit resource ceilings, and disabled canonical apply.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded SACV inspection, validation, evidence, proposal, and contract-defined noncanonical projection operations without canonical writes."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and uninstalls the exact inactive-undocked engine package."
        }
      ],
      "implementation_paths": [
        "modules/sacv-engine/CMakeLists.txt",
        "modules/sacv-engine/src/main.cpp",
        "modules/sacv-engine/src/sacv.cpp",
        "modules/sacv-engine/tests/process_smoke.sh",
        "modules/sacv-engine/tests/sacv_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not mutate canonical knowledge, decide semantic membership or ratification, start a listener, or infer permission from caller type.",
        "Does not activate or dock itself, publish artifacts, or claim an overall production release."
      ],
      "owner_contract": "modules/sacv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl administers selection and bounded invocation of the exact installed SACV engine while the engine retains operation semantics and the vector retains canonical truth.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "The engine statically links the common bounded process, digest, path, temporal, and snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sacv-engine",
      "status": "experimental",
      "title": "SACV independently installed knowledge engine",
      "what": "Provides the independently installable freezing-path engine that reads canonical SACV truth, validates registered OpenAPI contracts, compares bounded caller-designated contract evidence, and emits deterministic proposals and contract-defined noncanonical inventories.",
      "when": "Runs only on explicit qxctl or exact local process invocation under a bounded deadline and against an exact repository snapshot.",
      "where": "Executes from an inactive-undocked versioned installation and reads repository truth owned by knowledge/sacv/ without a network listener.",
      "who": "Any host-authorized caller using qxctl, vector maintainers, reviewers, integration tooling, and tests.",
      "why": "Gives SACV application-owned programmatic behavior without transferring semantic, ratification, mutation, or publication authority to tooling."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SACV owns OpenAPI 3.2.0 structural, security, reference, and example-safety requirements.",
          "reference": "knowledge/sacv/SPEC.md",
          "vector": "sacv"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed OpenAPI-conformance changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI supplies canonical routing evidence for registered API documents and security profiles.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future release of the conformance implementation.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine owns deterministic conformance mechanics; qxctl administers exact process selection and invocation without owning the rules or result truth.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "Conformance evaluates canonical registry and document truth; compatibility evidence compares two caller-designated revisions without registering either one.",
          "target_feature_id": "ssfv:symphony:sacv-engine.openapi-compatibility-evidence"
        }
      ],
      "evidence": [
        "modules/sacv-engine/src/sacv.cpp validates registry grammar, uniqueness, owner placement, SKVI coverage, OpenAPI 3.2.0 structure, security, local references, examples, servers, operation identifiers, and responses.",
        "modules/sacv-engine/tests/sacv_test.cpp verifies valid JSON, fail-closed YAML, unsafe profiles, symlink refusal, schema identity, bounds, and empty-registry validity."
      ],
      "feature_id": "ssfv:symphony:sacv-engine.api-contract-conformance",
      "how": "Loads the bounded canonical registry and registered no-follow JSON documents, validates identity and ownership, applies OpenAPI 3.2.0 structural and security rules, resolves only repository-local fragment references, and emits ordered findings and counts.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements fail-closed registry parsing, JSON OpenAPI validation, local-reference checks, profile enforcement, and deterministic findings."
        }
      ],
      "implementation_paths": [
        "modules/sacv-engine/src/sacv.cpp",
        "modules/sacv-engine/tests/sacv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not claim YAML conformance, resolve remote or external-file references, authenticate callers, generate endpoints, or decide registry ownership.",
        "qxctl does not own OpenAPI truth, publication authority, or canonical apply; it only administers exact installed-engine invocation and validates the bounded result envelope."
      ],
      "owner_contract": "modules/sacv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sacv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl selects and revalidates the exact installed SACV engine for inspect and check operations and validates its declared result protocol.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Conformance relies on shared bounded no-follow reads, snapshots, digests, deadlines, JSON bounds, and process framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sacv-engine",
      "status": "experimental",
      "title": "Registered OpenAPI contract conformance",
      "what": "Validates the canonical SACV registry and its registered JSON OpenAPI 3.2.0 contracts against bounded governance, structural, security, reference, and example-safety rules.",
      "when": "Runs on an explicit check invocation, optionally guarded by the caller's expected registry digest.",
      "where": "Executes read-only in the inactive-undocked SACV process against repository-local knowledge/sacv truth and registered owner documents.",
      "who": "Host-authorized callers, API owners, reviewers, documentation pipelines, integration tooling, and tests that need deterministic conformance evidence.",
      "why": "Prevents API contract drift and unsafe publication inputs from silently crossing the declarative source-of-truth boundary."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SACV owns the canonical API registry and inventory projection contract.",
          "reference": "knowledge/sacv/SPEC.md",
          "vector": "sacv"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed inventory-projection changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes every registered SACV contract and its owner evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future release of the inventory implementation.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine builds the contract-defined inventory; qxctl only invokes an exact installed version and validates its result identity.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "The inventory normalizes already registered contracts; registration proposals describe a possible future registry change without applying it.",
          "target_feature_id": "ssfv:symphony:sacv-engine.contract-registration-proposal"
        }
      ],
      "evidence": [
        "modules/sacv-engine/src/sacv.cpp implements the deterministic registry-conformance-inventory projection with explicit noncanonical and rebuildable markers.",
        "modules/sacv-engine/tests/sacv_test.cpp verifies empty and populated inventory counts, deterministic ordering, bounded format selection, and noncanonical authority."
      ],
      "feature_id": "ssfv:symphony:sacv-engine.contract-inventory-projection",
      "how": "Loads and validates the bounded SACV registry, normalizes registered metadata and conformance results into deterministic order, and emits a JSON inventory marked noncanonical and rebuildable without embedding raw contracts.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded registry loading, deterministic normalization, and contract-defined noncanonical inventory serialization."
        }
      ],
      "implementation_paths": [
        "modules/sacv-engine/src/sacv.cpp",
        "modules/sacv-engine/tests/sacv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not persist the inventory, bundle raw OpenAPI documents, generate code or documentation, or create registry entries.",
        "qxctl does not own the inventory's vector truth, publication authority, or canonical apply; it only administers exact installed-engine invocation."
      ],
      "owner_contract": "modules/sacv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sacv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl selects and invokes the exact installed SACV engine for explicit project operations and validates the returned projection envelope.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Inventory construction relies on shared bounded snapshots, no-follow paths, digests, deadlines, and process framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sacv-engine",
      "status": "experimental",
      "title": "Rebuildable API contract inventory",
      "what": "Projects the canonical SACV registry and conformance metadata into a portable deterministic contract inventory.",
      "when": "Runs only on an explicit JSON project invocation against the current bounded repository snapshot.",
      "where": "Exists transiently in the SACV response boundary outside hot and warm paths; canonical API truth remains in knowledge/sacv and registered owner documents.",
      "who": "Host-authorized callers, qxctl, documentation and integration tooling, reviewers, and tests that need a consistent API-contract inventory.",
      "why": "Provides a fast rebuildable view of registered API surfaces without turning a cache, generated document, or administrative client into canonical truth."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SACV owns registry-entry semantics and the only canonical target described by these proposals.",
          "reference": "knowledge/sacv/SPEC.md",
          "vector": "sacv"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed proposal-operation changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI provides owner and path routing evidence required before a contract can be proposed for registration.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future release of the proposal implementation.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine validates and renders proposal evidence; qxctl transports exact input and output without ratifying or applying the proposed change.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "A registration proposal describes a bounded prospective registry change; the contract inventory reflects only current canonical entries.",
          "target_feature_id": "ssfv:symphony:sacv-engine.contract-inventory-projection"
        }
      ],
      "evidence": [
        "modules/sacv-engine/src/sacv.cpp implements deterministic register_contract and replace_contract proposals bound to repository, session, context, time, expected digests, and one registry-only write set.",
        "modules/sacv-engine/tests/sacv_test.cpp verifies deterministic output, owner enforcement, registry target confinement, false ratification, disabled apply, and reserved-operation rejection."
      ],
      "feature_id": "ssfv:symphony:sacv-engine.contract-registration-proposal",
      "how": "Validates the common proposal envelope, current registry digest, optional prior-entry digest, candidate ownership and OpenAPI conformance, then renders an immutable content-addressed register-or-replace proposal targeting only knowledge/sacv/REGISTRY.md.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded proposal validation, candidate conformance, digest preconditions, deterministic rendering, and registry-only write-set evidence."
        }
      ],
      "implementation_paths": [
        "modules/sacv-engine/src/sacv.cpp",
        "modules/sacv-engine/tests/sacv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not create or edit an OpenAPI document, ratify ownership, register a contract, write the registry, publish documentation, or generate an SDK.",
        "qxctl does not own SACV truth, publication authority, or canonical apply; it only administers exact installed-engine invocation and validates the proposal envelope."
      ],
      "owner_contract": "modules/sacv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sacv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl invokes the exact installed SACV engine for explicit propose operations and validates that canonical apply remains disabled.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Proposal construction relies on shared repository identity, bounded snapshots, digests, temporal validation, and deterministic process framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sacv-engine",
      "status": "experimental",
      "title": "Content-addressed API registration proposals",
      "what": "Produces deterministic non-ratified proposals to register or replace one SACV contract entry after validating its existing owner document and exact registry preconditions.",
      "when": "Runs only on an explicit propose invocation with a bounded repository/session envelope, exact expected digests, and unexpired UTC timestamps.",
      "where": "Executes read-only in the SACV process and emits a proposal response; canonical state remains untouched in knowledge/sacv/REGISTRY.md.",
      "who": "Host-authorized callers, API owners, reviewers, and administrative workflows preparing a separately reviewed registry change.",
      "why": "Separates deterministic change preparation from human- or system-authorized ratification and application while preserving exact precondition evidence."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SACV owns compatibility vocabulary and evidence requirements for OpenAPI contract revisions.",
          "reference": "knowledge/sacv/SPEC.md",
          "vector": "sacv"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed compatibility-engine changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the SACV contracts, schemas, implementation, tests, and feature owner.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future release of the compatibility engine.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine classifies bounded differences; qxctl administers exact invocation and validates the result envelope without accepting or rejecting a contract change.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "Compatibility evidence compares two caller-designated contract revisions; OpenAPI conformance evaluates one document and its registered governance context.",
          "target_feature_id": "ssfv:symphony:sacv-engine.api-contract-conformance"
        }
      ],
      "evidence": [
        "modules/sacv-engine/src/sacv.cpp implements digest-bound baseline and candidate reads plus deterministic identical, compatible_additive, breaking, and review_required classification.",
        "modules/sacv-engine/tests/sacv_test.cpp verifies additive and breaking classifications, malformed contracts, external-reference refusal, and deterministic bounded results."
      ],
      "feature_id": "ssfv:symphony:sacv-engine.openapi-compatibility-evidence",
      "how": "Reads two bounded no-follow caller-designated JSON OpenAPI documents, verifies their tagged digests, validates each contract, compares normalized operations and schemas, and emits deterministic change evidence without deciding acceptance.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded document validation, normalized comparison, compatibility classification, and deterministic evidence serialization."
        }
      ],
      "implementation_paths": [
        "modules/sacv-engine/src/sacv.cpp",
        "modules/sacv-engine/tests/sacv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not decide whether a breaking or review-required change is authorized, safe to deploy, or eligible for publication.",
        "qxctl does not own compatibility truth, publication authority, or canonical apply; it only selects, invokes, and validates the exact installed engine."
      ],
      "owner_contract": "modules/sacv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sacv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl binds and invokes the exact installed SACV engine for explicit diff operations and validates the returned protocol identity.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "The comparison relies on shared bounded paths, no-follow reads, tagged digests, deadlines, and deterministic process framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sacv-engine",
      "status": "experimental",
      "title": "Bounded OpenAPI compatibility evidence",
      "what": "Classifies bounded caller-designated OpenAPI revision evidence as identical, additive-compatible, breaking, or requiring review.",
      "when": "Runs only on an explicit diff invocation with exact baseline and candidate paths and digests under the request deadline.",
      "where": "Executes in the inactive-undocked SACV process against repository-local JSON documents; it performs no network or external-reference resolution.",
      "who": "Host-authorized callers, API maintainers, reviewers, integration tooling, and release preparation workflows that need deterministic compatibility evidence.",
      "why": "Makes contract-change consequences mechanically visible before any separately authorized acceptance, publication, or deployment decision."
    }
  ],
  "source_scope": "modules/sacv-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
