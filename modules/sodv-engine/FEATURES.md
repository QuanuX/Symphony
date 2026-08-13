# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sodv-engine/SPEC.md",
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
          "distinction": "The engine implements SODV observation, verification, recovery, and projection semantics; qxctl selects and invokes the exact installed process without owning release truth or publication authority.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        }
      ],
      "evidence": [
        "modules/sodv-engine tests exercise the implemented operation set, deterministic results, bounds, invalid input, and disabled apply.",
        "The module CMake and receipt-v2 surfaces prove independent versioned installation and conservative uninstall."
      ],
      "feature_id": "ssfv:symphony:sodv-engine",
      "how": "The C++ process uses the shared bounded engine envelope, no-follow reads, canonical digests, deterministic ordering, strict schemas, explicit resource ceilings, and disabled canonical apply.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded SODV inspection, validation, caller-observation verification, proposal, recovery, and contract-defined noncanonical projection operations without canonical writes."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and uninstalls the exact inactive-undocked engine package."
        }
      ],
      "implementation_paths": [
        "modules/sodv-engine/CMakeLists.txt",
        "modules/sodv-engine/src/main.cpp",
        "modules/sodv-engine/src/sodv.cpp",
        "modules/sodv-engine/tests/process_smoke.sh",
        "modules/sodv-engine/tests/sodv_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not mutate canonical knowledge, decide semantic membership or ratification, start a listener, or infer permission from caller type.",
        "Does not activate or dock itself, publish artifacts, or claim an overall production release."
      ],
      "owner_contract": "modules/sodv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl administers selection and bounded invocation of the exact installed SODV engine while the engine retains operation semantics and the vector retains canonical truth.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "The engine statically links the common bounded process, digest, path, temporal, and snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sodv-engine",
      "status": "experimental",
      "title": "SODV independently installed knowledge engine",
      "what": "Provides the independently installable freezing-path engine that validates canonical SODV release lineage, compares bounded caller-observed publication state, renders forward-only record proposals, reconciles interrupted publication journals, and emits contract-defined noncanonical transaction projections.",
      "when": "Runs only on explicit qxctl or exact local process invocation under a bounded deadline and against an exact repository snapshot.",
      "where": "Executes from an inactive-undocked versioned installation and reads repository truth owned by knowledge/sodv/ without a network listener.",
      "who": "Any host-authorized caller using qxctl, vector maintainers, reviewers, integration tooling, and tests.",
      "why": "Gives SODV application-owned programmatic behavior without transferring semantic, ratification, mutation, or publication authority to tooling."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed release-proposal changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the SODV ledger, schemas, implementation, tests, and feature owner.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV owns forward-only release record types, lineage, and proposal preconditions.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine validates and renders proposal evidence; qxctl transports exact inputs and results without ratifying, appending, tagging, or publishing.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "Forward proposals render one possible next canonical record; interrupted-publication reconciliation decides only the next recovery action from a noncanonical journal and current observation.",
          "target_feature_id": "ssfv:symphony:sodv-engine.interrupted-publication-reconciliation"
        }
      ],
      "evidence": [
        "modules/sodv-engine/src/sodv.cpp implements digest-bound authorization, correction, completion, and failure proposals with lineage and caller-observation validation.",
        "modules/sodv-engine/tests/sodv_test.cpp verifies deterministic v2 proposals, stale-state refusal, completion-evidence gates, disabled apply, and publication-operation rejection."
      ],
      "feature_id": "ssfv:symphony:sodv-engine.forward-release-record-proposal",
      "how": "Validates the repository/session envelope, exact ledger digest, strict UTC expiry, prospective record ordering and lineage, and required caller-observed evidence before rendering one content-addressed append proposal targeting only knowledge/sodv/RELEASES.md.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements forward-record validation, observation binding, deterministic Markdown rendering, proposal digests, and ledger-only write-set evidence."
        }
      ],
      "implementation_paths": [
        "modules/sodv-engine/src/sodv.cpp",
        "modules/sodv-engine/tests/sodv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not append the release ledger, ratify a record, create or move a tag, upload an artifact, or declare publication complete independently.",
        "qxctl does not own SODV truth, publication authority, or canonical apply; it only administers exact installed-engine invocation and validates the proposal envelope."
      ],
      "owner_contract": "modules/sodv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sodv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl invokes the exact installed SODV engine for explicit propose operations and validates that canonical apply remains disabled.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Proposal construction relies on shared repository identity, bounded snapshots, no-follow reads, digests, strict time, and deterministic framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sodv-engine",
      "status": "experimental",
      "title": "Forward-only release record proposals",
      "what": "Produces deterministic non-ratified proposals for one authorization, authorization correction, completion, or failure record that extends existing canonical release lineage.",
      "when": "Runs only on explicit propose invocation with exact expected ledger state, an unexpired envelope, and caller observation whenever the record is not an authorization.",
      "where": "Executes read-only in the inactive-undocked SODV process; the proposed append remains response evidence outside the canonical ledger.",
      "who": "Host-authorized callers, release maintainers, reviewers, and recovery workflows preparing a separately authorized release-ledger append.",
      "why": "Preserves append-only release history and exact evidence preconditions while separating preparation from ratification, publication, and canonical mutation."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed recovery-operation changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes SODV recovery contracts, schemas, implementation, tests, and feature ownership.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV owns interrupted-publication journal reconciliation and forward-only repair semantics.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine derives recovery evidence; qxctl supplies the exact bounded journal and observation and cannot delete, mutate, or declare them canonical through the binding layer.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "Reconciliation binds a noncanonical journal to current observed state and recommends a next action; verification compares observed state with canonical release truth without journal context.",
          "target_feature_id": "ssfv:symphony:sodv-engine.observed-publication-verification"
        }
      ],
      "evidence": [
        "modules/sodv-engine/src/sodv.cpp verifies journal digests, canonical authorization and tag sets, caller observations, completion lineage, and bounded recovery actions without mutating the journal.",
        "modules/sodv-engine/tests/sodv_test.cpp verifies resume, wait, completion-proposal, completed no-op, stale-state, and fail-closed recovery paths."
      ],
      "feature_id": "ssfv:symphony:sodv-engine.interrupted-publication-reconciliation",
      "how": "Validates a digest-bound v1 journal and its canonical authorization, compares its intended tags with bounded caller-observed publication state, then returns a deterministic resume, wait, propose-forward-completion, completed no-op, or fail-closed-review action.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements journal validation, authorization and observation binding, deterministic recovery classification, and optional forward-completion proposal composition."
        }
      ],
      "implementation_paths": [
        "modules/sodv-engine/src/sodv.cpp",
        "modules/sodv-engine/tests/sodv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not edit or delete the journal, resume a publisher, create tags, query public providers, append completion, or conceal unresolved mismatches.",
        "qxctl does not own recovery truth, release publication authority, or canonical apply; it only administers exact installed-engine invocation and validates the result envelope."
      ],
      "owner_contract": "modules/sodv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sodv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl invokes the exact installed SODV engine for explicit recover operations and validates the returned recovery identity and disabled apply state.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Recovery relies on shared bounded JSON, tagged digests, strict time, no-follow ledger reads, deadlines, and deterministic framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sodv-engine",
      "status": "experimental",
      "title": "Interrupted publication reconciliation",
      "what": "Reconciles a digest-bound noncanonical publication journal with canonical authorization and current caller-observed state to determine the safe forward action.",
      "when": "Runs only on explicit recover invocation after a publication workflow was interrupted or its durable local state requires re-evaluation.",
      "where": "Executes read-only in the SODV process against knowledge/sodv/RELEASES.md plus caller-supplied journal and observation evidence.",
      "who": "Host-authorized callers, release operators, automation, and reviewers recovering a partially completed or uncertain publication transaction.",
      "why": "Allows out-of-sequence publication work to resume or stop safely without rewriting history, trusting a warm cache, or carrying an unresolved error into permanent release truth."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed publication-verification changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the observation schema, SODV contracts, implementation, tests, and feature owner.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV owns canonical authorization, correction, completion, and external-observation comparison semantics.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine compares observation with canonical truth; qxctl supplies and transports evidence without contacting providers or declaring release completion.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "Verification evaluates bounded caller-observed external state; ledger validation evaluates only canonical release-record relationships and shapes.",
          "target_feature_id": "ssfv:symphony:sodv-engine.release-ledger-validation"
        }
      ],
      "evidence": [
        "modules/sodv-engine/src/sodv.cpp validates provider-neutral caller observations and classifies authorized-unpublished, waiting, completion-candidate, verified-completed, and mismatch states.",
        "modules/sodv-engine/tests/sodv_test.cpp verifies unresolved and resolved observations, exact tag and digest matching, caller-supplied evidence boundaries, and noncanonical output."
      ],
      "feature_id": "ssfv:symphony:sodv-engine.observed-publication-verification",
      "how": "Validates bounded provider-neutral observation fields and evidence digests, matches each unit to canonical authorization and latest correction or completion, and emits a content-addressed read-only classification without network access.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded observation parsing, canonical-lineage matching, publication-state classification, and deterministic result digests."
        }
      ],
      "implementation_paths": [
        "modules/sodv-engine/src/sodv.cpp",
        "modules/sodv-engine/tests/sodv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not collect Git or public-package evidence, perform a network request, trust a provider cache, or independently declare publication completion.",
        "qxctl does not own SODV truth, provider evidence, publication authority, or canonical apply; it only administers exact installed-engine invocation."
      ],
      "owner_contract": "modules/sodv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sodv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl selects and invokes the exact installed SODV engine for explicit verify operations and validates the returned protocol identity.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Verification relies on shared bounded JSON, tagged digests, strict time, no-follow ledger reads, deadlines, and deterministic process framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sodv-engine",
      "status": "experimental",
      "title": "Caller-observed publication verification",
      "what": "Compares bounded provider-neutral publication observations supplied by the caller with canonical authorization, correction, and completion evidence.",
      "when": "Runs on explicit verify, proposal, or recovery paths whenever current external publication state must be compared with canonical release lineage.",
      "where": "Executes locally in the SODV process; external state enters only as bounded request evidence and no provider is contacted by the engine.",
      "who": "Host-authorized callers, provider adapters, release automation, recovery workflows, reviewers, and tests supplying independently gathered evidence.",
      "why": "Separates evidence collection from deterministic release-state comparison so GitHub, GitLab, proprietary, and air-gapped workflows can use the same canonical semantics."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed release-ledger validation changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the canonical ledger, SODV contracts, schemas, implementation, tests, and feature owner.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV owns append-only release-lineage and publication-unit rules.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine applies SODV ledger rules; qxctl administers the exact installed process and cannot reinterpret a finding or make release truth canonical.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "Ledger validation proves canonical record structure and lineage; transaction projection derives a disposable view only after that truth is valid.",
          "target_feature_id": "ssfv:symphony:sodv-engine.release-transaction-projection"
        }
      ],
      "evidence": [
        "modules/sodv-engine/src/sodv.cpp parses legacy-v1 and provider-neutral-v2 records and validates IDs, time order, statuses, subjects, unit identity, correction evidence, and unique completion lineage.",
        "modules/sodv-engine/tests/sodv_test.cpp verifies the canonical repository, invalid expected state, no-follow ledger access, bounded schemas, and fail-closed operation boundaries."
      ],
      "feature_id": "ssfv:symphony:sodv-engine.release-ledger-validation",
      "how": "Reads the bounded no-follow canonical ledger, parses supported record versions, computes record digests, validates chronological append-only relationships and publication-unit invariants, and returns deterministic findings and transaction counts.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded ledger parsing, v1/v2 record normalization, lineage validation, digesting, and deterministic evidence."
        }
      ],
      "implementation_paths": [
        "modules/sodv-engine/src/sodv.cpp",
        "modules/sodv-engine/tests/sodv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not append, reorder, repair, or delete canonical records, gather public evidence, or infer that an authorization has been published.",
        "qxctl does not own release truth, publication authority, or canonical apply; it only administers exact installed-engine invocation and validates the bounded check result."
      ],
      "owner_contract": "modules/sodv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sodv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl selects and invokes the exact installed SODV engine for inspect and check operations and validates its result protocol and identity.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Ledger validation relies on shared bounded no-follow reads, snapshots, tagged digests, temporal conformance, deadlines, and process framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sodv-engine",
      "status": "experimental",
      "title": "Append-only release ledger validation",
      "what": "Validates canonical legacy-v1 and provider-neutral-v2 SODV records, publication units, ordering, and authorization-correction-completion lineage.",
      "when": "Runs on explicit check and as a mandatory precondition for verify, propose, recover, and project operations.",
      "where": "Executes read-only against knowledge/sodv/RELEASES.md and its bounded contract snapshot in the inactive-undocked SODV process.",
      "who": "Host-authorized callers, release maintainers, reviewers, proposal and recovery workflows, integration tooling, and tests.",
      "why": "Ensures all later publication evidence and recovery decisions rest on coherent immutable release history rather than a mutable or provider-specific log."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed release-projection changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes canonical SODV records, schemas, implementation, tests, and feature ownership.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV owns the release-transaction relationships represented by the projection.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "The engine builds the contract-defined projection; qxctl only invokes an exact installed engine and validates its response identity.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        },
        {
          "distinction": "The projection is a rebuildable transaction view; ledger validation is the canonical relationship check from which that view is derived.",
          "target_feature_id": "ssfv:symphony:sodv-engine.release-ledger-validation"
        }
      ],
      "evidence": [
        "modules/sodv-engine/src/sodv.cpp emits normalized records and authorization transactions with latest correction, completion, state, counts, canonical-ledger evidence, and a projection digest.",
        "modules/sodv-engine/tests/sodv_test.cpp verifies deterministic JSON projection, transaction counts, noncanonical and rebuildable markers, and reserved format boundaries."
      ],
      "feature_id": "ssfv:symphony:sodv-engine.release-transaction-projection",
      "how": "Validates canonical ledger truth, normalizes each record, groups authorization lineages with their latest correction and completion, orders the bounded output deterministically, and emits a tagged digest with noncanonical and rebuildable markers.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements deterministic release-record normalization, transaction grouping, and contract-defined noncanonical projection serialization."
        }
      ],
      "implementation_paths": [
        "modules/sodv-engine/src/sodv.cpp",
        "modules/sodv-engine/tests/sodv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not persist the projection, alter release records, observe providers, publish artifacts, or convert derived transaction state into canonical truth.",
        "qxctl does not own the projection's truth, release publication authority, or canonical apply; it only administers exact installed-engine invocation."
      ],
      "owner_contract": "modules/sodv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sodv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl selects and invokes the exact installed SODV engine for explicit project operations and validates the returned projection envelope.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Projection construction relies on shared bounded snapshots, no-follow paths, digests, deadlines, and deterministic process framing.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sodv-engine",
      "status": "experimental",
      "title": "Rebuildable release transaction projection",
      "what": "Projects valid canonical release records into a deterministic portable inventory of authorization transactions, latest corrections, completions, states, and publication units.",
      "when": "Runs only on explicit JSON project invocation after the canonical ledger passes all validation checks.",
      "where": "Exists transiently in the SODV response boundary outside hot and warm paths; canonical release truth remains in knowledge/sodv/RELEASES.md.",
      "who": "Host-authorized callers, qxctl, release tooling, reviewers, documentation systems, and tests that need a consistent transaction inventory.",
      "why": "Provides a portable graph-ready view of release lineage without treating a cache, database, administrative client, or generated artifact as canonical history."
    }
  ],
  "source_scope": "modules/sodv-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
