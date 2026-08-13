# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/ssfv-engine/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
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
          "distinction": "The engine implements SSFV catalog, comparison, proposal, and graph semantics; qxctl selects and invokes the exact installed process without deciding feature-worthiness or owning canonical apply.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        }
      ],
      "evidence": [
        "modules/ssfv-engine tests exercise the implemented operation set, deterministic results, bounds, invalid input, and disabled apply.",
        "The module CMake and receipt-v2 surfaces prove independent versioned installation and conservative uninstall."
      ],
      "feature_id": "ssfv:symphony:ssfv-engine",
      "how": "The C++ process uses the shared bounded engine envelope, no-follow reads, canonical digests, deterministic ordering, strict schemas, explicit resource ceilings, and disabled canonical apply.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded SSFV inspection, validation, evidence, proposal, and contract-defined noncanonical projection operations without canonical writes."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and uninstalls the exact inactive-undocked engine package."
        }
      ],
      "implementation_paths": [
        "modules/ssfv-engine/CMakeLists.txt",
        "modules/ssfv-engine/src/main.cpp",
        "modules/ssfv-engine/src/ssfv.cpp",
        "modules/ssfv-engine/tests/process_smoke.sh",
        "modules/ssfv-engine/tests/ssfv_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not mutate canonical knowledge, decide semantic membership or ratification, start a listener, or infer permission from caller type.",
        "Does not activate or dock itself, publish artifacts, or claim an overall production release."
      ],
      "owner_contract": "modules/ssfv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl administers selection and bounded invocation of the exact installed SSFV engine while the engine retains operation semantics and the vector retains canonical truth.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "The engine statically links the common bounded process, digest, path, temporal, and snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/ssfv-engine",
      "status": "experimental",
      "title": "SSFV independently installed knowledge engine",
      "what": "Provides the independently installable freezing-path engine that validates canonical semantic-feature catalog integrity, emits digest-bound snapshots, compares semantic freshness, renders bounded catalog proposals, and builds contract-defined noncanonical feature graphs.",
      "when": "Runs only on explicit qxctl or exact local process invocation under a bounded deadline and against an exact repository snapshot.",
      "where": "Executes from an inactive-undocked versioned installation and reads repository truth owned by knowledge/ssfv/ without a network listener.",
      "who": "Any host-authorized caller using qxctl, vector maintainers, reviewers, integration tooling, and tests.",
      "why": "Gives SSFV application-owned programmatic behavior without transferring semantic, ratification, mutation, feature-worthiness, or publication authority to tooling."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed catalog-proposal changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI supplies route and owner evidence and may be one target in a bounded multi-file proposal.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future release of the catalog-proposal implementation.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine validates caller-declared semantic content and renders proposal evidence; qxctl transports it without deciding feature-worthiness, ratification, or apply.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "A catalog proposal describes a bounded prospective canonical write set; graph projection represents only current canonical relationships and never proposes edits.",
          "target_feature_id": "ssfv:symphony:ssfv-engine.semantic-graph-projection"
        }
      ],
      "evidence": [
        "modules/ssfv-engine/src/ssfv.cpp implements allocate_namespace, add_feature, update_feature, move_feature, deprecate_feature, and retire_feature proposals with exact prospective hierarchy and target-set validation.",
        "modules/ssfv-engine/tests/ssfv_test.cpp verifies deterministic proposals, new and existing owner files, atomic moves, stale feature digests, unsafe paths, typed absence, false domain authority, and disabled apply."
      ],
      "feature_id": "ssfv:symphony:ssfv-engine.catalog-change-proposal",
      "how": "Validates the repository/session envelope, contract and registry digests, caller-declared semantic rationale and evidence, namespace allocation, prospective record hierarchy, route ownership, owner and implementation evidence, lifecycle transitions, typed absence, and the exact bounded multi-file target set before rendering a content-addressed proposal.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements prospective catalog validation, semantic-declaration bounds, record lifecycle checks, exact multi-file write-set rendering, and immutable proposal digests."
        }
      ],
      "implementation_paths": [
        "modules/ssfv-engine/src/ssfv.cpp",
        "modules/ssfv-engine/tests/ssfv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not decide that a proposed capability is feature-worthy, allocate a namespace, create or edit FEATURES.md, update SKVI or the registry, ratify the proposal, or apply its write set.",
        "qxctl does not own SSFV truth, feature-worthiness, publication authority, or canonical apply; it only administers exact installed-engine invocation and validates the proposal envelope."
      ],
      "owner_contract": "modules/ssfv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssfv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl invokes the exact installed SSFV engine for explicit propose operations and validates that feature-worthiness and canonical apply remain disabled.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Proposal construction relies on shared repository identity, bounded snapshots, no-follow reads, tagged digests, strict time, and deterministic process framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/ssfv-engine",
      "status": "experimental",
      "title": "Content-addressed semantic catalog proposals",
      "what": "Produces deterministic non-ratified proposals for namespace allocation or one add, update, move, deprecate, or retire feature operation with an exact bounded multi-file write set.",
      "when": "Runs only on explicit propose invocation with exact current-state digests, an unexpired envelope, and caller-declared semantic evidence.",
      "where": "Executes read-only in the SSFV process and emits proposal evidence; canonical namespace, registry, SKVI, and distributed owner files remain untouched.",
      "who": "Host-authorized callers, feature owners, authorized feature governance, reviewers, qxctl workflows, and agents preparing an auditable catalog change.",
      "why": "Makes distributed semantic changes reviewable as one deterministic precondition-bound unit while preserving caller-neutral governance and denying autonomous feature creation."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed catalog-validation and snapshot changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes every registered feature owner, implementation surface, schema, and engine test.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future release of the catalog-integrity implementation.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine validates and snapshots SSFV truth; qxctl administers exact invocation and result validation without deciding feature-worthiness or coverage completeness.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "Catalog integrity validates current canonical structure and emits its snapshot; semantic-freshness comparison evaluates change between such snapshots without deciding semantic meaning.",
          "target_feature_id": "ssfv:symphony:ssfv-engine.semantic-freshness-comparison"
        }
      ],
      "evidence": [
        "modules/ssfv-engine/src/ssfv.cpp validates namespace and registry identity, owner files, record schemas, paths, hierarchy, relationships, cross-vector evidence, SKVI coverage, and bounded source evidence before building a semantic snapshot.",
        "modules/ssfv-engine/tests/ssfv_test.cpp verifies canonical catalog counts, hierarchy, malformed records, no-follow reads, evidence bounds, expected digests, deterministic snapshots, and partial-coverage preservation."
      ],
      "feature_id": "ssfv:symphony:ssfv-engine.catalog-integrity-snapshot",
      "how": "Loads bounded no-follow SSFV contracts, namespace and feature registries, SKVI routes, distributed owner files, owner contracts, and implementation evidence; validates all structural relationships; then emits deterministic file, record, source, contract, and snapshot digests.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded catalog discovery, structural validation, hierarchy and evidence checks, deterministic findings, and semantic-snapshot serialization."
        }
      ],
      "implementation_paths": [
        "modules/ssfv-engine/src/ssfv.cpp",
        "modules/ssfv-engine/tests/ssfv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not infer feature-worthiness, declare partial coverage complete, create records, repair routes, or turn a successful structural check into semantic ratification.",
        "qxctl does not own SSFV truth, feature-worthiness, publication authority, or canonical apply; it only administers exact installed-engine invocation and validates the bounded result."
      ],
      "owner_contract": "modules/ssfv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssfv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl selects and revalidates the exact installed SSFV engine for inspect and check operations and validates its declared result protocol.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Catalog integrity relies on shared bounded snapshots, no-follow paths, tagged digests, JSON limits, deadlines, time conformance, and deterministic framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/ssfv-engine",
      "status": "experimental",
      "title": "Semantic catalog integrity snapshots",
      "what": "Validates canonical SSFV namespace, registry, distributed record, evidence, hierarchy, relationship, and route integrity and emits a content-addressed semantic snapshot.",
      "when": "Runs on every explicit SSFV check and as a mandatory current-state precondition for diff, propose, and graph operations.",
      "where": "Executes read-only in the inactive-undocked SSFV process across knowledge/ssfv, knowledge/skvi, registered FEATURES.md owners, owner contracts, and cited implementation evidence.",
      "who": "Host-authorized callers, feature owners, reviewers, qxctl workflows, documentation tooling, integration tests, and agents locating precise application truth.",
      "why": "Makes a distributed semantic catalog mechanically auditable and content-addressable without centralizing ownership or confusing structural validity with semantic completeness."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed semantic-comparison changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the owner and implementation surfaces whose content digests participate in freshness evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future release of the freshness implementation.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine computes bounded comparison evidence; qxctl selects the exact engine and administers report or require flags without deciding whether a candidate is semantically meaningful.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "Freshness comparison reports evidence that cited implementation or contract content changed; catalog integrity determines current structural validity without claiming semantic change.",
          "target_feature_id": "ssfv:symphony:ssfv-engine.catalog-integrity-snapshot"
        }
      ],
      "evidence": [
        "modules/ssfv-engine/src/ssfv.cpp verifies supplied snapshot identity and hierarchy, compares current and baseline file and record digests, scopes feature IDs, reports uncovered paths, and classifies unresolved semantic candidates.",
        "modules/ssfv-engine/tests/ssfv_test.cpp verifies disabled, report, and require modes; deterministic unchanged snapshots; record, owner-contract, implementation-path, scope, stale-digest, malformed-baseline, deadline, and size boundaries."
      ],
      "feature_id": "ssfv:symphony:ssfv-engine.semantic-freshness-comparison",
      "how": "Validates a bounded caller-supplied semantic snapshot by its digest and structure, builds current catalog state, compares file and record evidence within an optional feature scope, and emits deterministic changed, added, removed, uncovered-path, and review-required evidence under disabled, report, or require policy.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements semantic-snapshot validation, bounded digest comparison, scoped change evidence, freshness policy outcomes, and deterministic result serialization."
        }
      ],
      "implementation_paths": [
        "modules/ssfv-engine/src/ssfv.cpp",
        "modules/ssfv-engine/tests/ssfv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not infer that changed source evidence changed feature semantics, automatically edit a record, ratify a candidate, or persist a baseline.",
        "qxctl does not own SSFV truth, feature-worthiness, publication authority, or canonical apply; it only administers the exact installed engine and caller-selected freshness posture."
      ],
      "owner_contract": "modules/ssfv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssfv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl invokes the exact installed SSFV engine for explicit check and diff operations, supplies bounded freshness inputs, and validates result identity.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Comparison relies on shared bounded snapshots, no-follow paths, tagged digests, JSON limits, deadlines, and deterministic process framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/ssfv-engine",
      "status": "experimental",
      "title": "Bounded semantic freshness comparison",
      "what": "Compares a content-addressed semantic baseline with live canonical catalog evidence and reports structural changes and unresolved semantic-review candidates.",
      "when": "Runs on explicit diff invocation or on check when freshness is report or require and a bounded baseline is supplied.",
      "where": "Executes transiently in the SSFV process against live repository truth and caller-supplied baseline evidence; persistent maintenance baselines remain a separate coordinator concern.",
      "who": "Host-authorized callers, feature owners, reviewers, qxctl maintenance workflows, agents, CI, and debugging tools assessing catalog freshness.",
      "why": "Surfaces small but consequential implementation or contract changes that could otherwise leave distributed feature truth stale, while retaining semantic judgment for authorized governance."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed graph-projection changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI supplies canonical owner and relationship routing represented by graph evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future release of the graph implementation.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine builds the contract-defined graph; qxctl only invokes an exact installed process and validates its response without owning graph or feature truth.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "The graph projects current canonical feature relationships; catalog proposals describe possible future changes and do not enter the graph until separately ratified and applied.",
          "target_feature_id": "ssfv:symphony:ssfv-engine.catalog-change-proposal"
        }
      ],
      "evidence": [
        "modules/ssfv-engine/src/ssfv.cpp emits bounded nodes, primary-parent edges, and typed crosslinks in deterministic order with complete node and edge counts and a projection digest.",
        "modules/ssfv-engine/tests/ssfv_test.cpp verifies deterministic topology, relationship integrity, duplicate and missing targets, graph bounds, canonical repository counts, and noncanonical rebuildable markers."
      ],
      "feature_id": "ssfv:symphony:ssfv-engine.semantic-graph-projection",
      "how": "Validates the complete bounded canonical catalog, serializes each record as one node, emits one primary-parent edge for every non-root record plus typed cross-feature edges, sorts the result deterministically, and marks it noncanonical and rebuildable with a tagged digest.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded graph-node and edge construction, relationship normalization, deterministic ordering, and contract-defined noncanonical projection serialization."
        }
      ],
      "implementation_paths": [
        "modules/ssfv-engine/src/ssfv.cpp",
        "modules/ssfv-engine/tests/ssfv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not persist a graph, authorize a graph database, infer unrecorded relationships, decide feature-worthiness, or make derived topology canonical.",
        "qxctl does not own SSFV truth, graph truth, feature-worthiness, publication authority, or canonical apply; it only administers exact installed-engine invocation."
      ],
      "owner_contract": "modules/ssfv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssfv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl selects and invokes the exact installed SSFV engine for explicit graph operations and validates the returned projection identity and counts.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Graph projection relies on shared bounded snapshots, no-follow paths, tagged digests, deadlines, JSON bounds, and deterministic framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/ssfv-engine",
      "status": "experimental",
      "title": "Portable semantic feature graph projection",
      "what": "Projects the complete valid SSFV catalog into deterministic feature nodes, primary-parent edges, and typed cross-feature relationships suitable for portable or graph-database consumption.",
      "when": "Runs only on explicit JSON graph invocation after the live canonical catalog passes all structural validation.",
      "where": "Exists transiently in the SSFV response boundary outside hot and warm paths; canonical semantics remain distributed across registered FEATURES.md records and SSFV contracts.",
      "who": "Host-authorized callers, qxctl, documentation and discovery tools, agents, reviewers, graph importers, and tests that need a complete relationship view.",
      "why": "Makes distributed application DNA navigable and graph-ready without making a projection store, query tool, or administrative client the source of semantic truth."
    }
  ],
  "source_scope": "modules/ssfv-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
