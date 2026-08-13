# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/sclv-engine/SPEC.md",
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
        }
      ],
      "evidence": [
        "modules/sclv-engine tests exercise the implemented operation set, deterministic results, bounds, invalid input, and disabled apply.",
        "The module CMake and receipt-v2 surfaces prove independent versioned installation and conservative uninstall."
      ],
      "feature_id": "ssfv:symphony:sclv-engine",
      "how": "The C++ process uses the shared bounded engine envelope, no-follow reads, canonical digests, deterministic ordering, strict schemas, explicit resource ceilings, and disabled canonical apply.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded SCLV inspection, validation, evidence, proposal, recovery, and contract-defined noncanonical projection operations without canonical writes."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and uninstalls the exact inactive-undocked engine package."
        }
      ],
      "implementation_paths": [
        "modules/sclv-engine/CMakeLists.txt",
        "modules/sclv-engine/src/main.cpp",
        "modules/sclv-engine/src/sclv.cpp",
        "modules/sclv-engine/tests/process_smoke.sh",
        "modules/sclv-engine/tests/sclv_test.cpp"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not mutate canonical knowledge, decide semantic membership or ratification, start a listener, or infer permission from caller type.",
        "Does not activate or dock itself, publish artifacts, or claim an overall production release."
      ],
      "owner_contract": "modules/sclv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The engine statically links the common bounded process, digest, path, temporal, and snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sclv-engine",
      "status": "experimental",
      "title": "SCLV independently installed knowledge engine",
      "what": "Provides the independently installable freezing-path engine that reads canonical SCLV truth and emits deterministic bounded evidence, proposals, and disposable projections under that vector's contract.",
      "when": "Runs only on explicit qxctl or exact local process invocation under a bounded deadline and against an exact repository snapshot.",
      "where": "Executes from an inactive-undocked versioned installation and reads repository truth owned by knowledge/sclv/ without a network listener.",
      "who": "Any host-authorized caller using qxctl, vector maintainers, reviewers, integration tooling, and tests.",
      "why": "Gives SCLV application-owned programmatic behavior without transferring semantic, ratification, mutation, or publication authority to tooling."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV owns the canonical append-only ledger shape, lineage, and reviewed-change semantics checked by this operation.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the ledger, SCLV contracts, engine implementation, and indexed references checked during proposal preparation.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of the engine implementing ledger assurance.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "Evaluates the current immutable ledger and its lineage; evidence-bound append proposals describe one prospective v3 addition without mutating that ledger.",
          "target_feature_id": "ssfv:symphony:sclv-engine.evidence-bound-append-proposals"
        },
        {
          "distinction": "This is detailed vector-owned SCLV compatibility assurance; Symphony Validator independently checks broader repository and cross-vector conformance.",
          "target_feature_id": "ssfv:symphony:symphony-validator"
        }
      ],
      "evidence": [
        "modules/sclv-engine/tests/sclv_test.cpp verifies the live ledger, v1/v2/v3 compatibility, exact v3 shape, unique identity and revision evidence, chronology, expected-digest comparison, and invalid-ledger findings.",
        "modules/sclv-engine/tests/process_smoke.sh exercises installed-style bounded ledger checking against repository truth."
      ],
      "feature_id": "ssfv:symphony:sclv-engine.append-only-ledger-assurance",
      "how": "Reads the canonical ledger and contract snapshot through bounded no-follow operations, parses legacy and exact v3 records, validates required fields, stable identities, provider-neutral change-request and ratification evidence, unique revisions, nondecreasing recorded time, safe affected surfaces, SKVI references, and optional expected-ledger state, then emits deterministic findings without repair.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded append-only ledger parsing, multi-version shape and chronology checks, deterministic findings, and expected-state comparison."
        }
      ],
      "implementation_paths": [
        "modules/sclv-engine/src/sclv.cpp",
        "modules/sclv-engine/src/sclv.hpp",
        "modules/sclv-engine/tests/process_smoke.sh",
        "modules/sclv-engine/tests/sclv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not prove permission or ratification, append or rewrite a record, repair chronology, modify provider state, or convert legacy evidence into v3 truth.",
        "qxctl selects and validates the exact installed process and its response; it does not own SCLV ledger semantics, findings, or canonical mutation."
      ],
      "owner_contract": "modules/sclv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sclv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl resolves and revalidates one exact installed SCLV engine before invoking its check operation, without becoming the ledger's source of truth.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Ledger request parsing, deadlines, stable findings responses, and digest framing use the shared bounded process contract.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
          "type": "depends_on"
        },
        {
          "rationale": "Canonical ledger and contract evidence use the shared no-follow content-addressed snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.content-addressed-evidence-snapshots",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sclv-engine",
      "status": "experimental",
      "title": "Append-only change-ledger assurance",
      "what": "Produces deterministic bounded evidence that the canonical reviewed-change ledger retains valid multi-version record shape, unique lineage, temporal order, provider-neutral v3 semantics, and optional expected-state continuity.",
      "when": "On explicit `qxctl sclv check`, exact local check invocation, and internally before proposal or projection operations that require a clean ledger.",
      "where": "Inside one exact inactive-undocked SCLV engine installation reading knowledge/sclv/CHANGELOG.md and canonical SCLV contracts without a listener.",
      "who": "Any permission-backed caller using qxctl or the exact process, SCLV maintainers, reviewers, release gates, integration tooling, and tests.",
      "why": "Reviewed-change history must expose gaps, ambiguity, duplicate lineage, or temporal regression before new evidence is proposed or derived history is consumed."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV owns the canonical change-history semantics normalized by this disposable projection.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the canonical ledger, contracts, engine implementation, and projection schema.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of the engine or derived projection format.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "Normalizes already-recorded history for bounded consumption; append proposals describe prospective evidence that is not yet canonical.",
          "target_feature_id": "ssfv:symphony:sclv-engine.evidence-bound-append-proposals"
        }
      ],
      "evidence": [
        "modules/sclv-engine/tests/sclv_test.cpp verifies deterministic provider-neutral history, legacy normalization markers, record and projection digests, clean-ledger gating, and noncanonical rebuildability.",
        "modules/sclv-engine/SPEC.md limits projection to the portable JSON format and forbids engine writes."
      ],
      "feature_id": "ssfv:symphony:sclv-engine.disposable-provider-neutral-history",
      "how": "Requires a clean ledger, normalizes v3 records into explicit repository, change-request, ratification, and affected-surface fields, marks bounded legacy normalization rather than inventing missing evidence, binds each record and the full result to digests, and returns noncanonical rebuildable JSON.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements multi-version record normalization, deterministic ordering, legacy evidence marking, per-record digests, and aggregate projection binding."
        }
      ],
      "implementation_paths": [
        "modules/sclv-engine/src/sclv.cpp",
        "modules/sclv-engine/src/sclv.hpp",
        "modules/sclv-engine/tests/process_smoke.sh",
        "modules/sclv-engine/tests/sclv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not upgrade legacy canonical records, fill unavailable evidence, write an artifact, create a database, publish history, or make the projection authoritative.",
        "qxctl mediates exact process selection and response validation only; it does not own projection semantics or convert derived history into canonical SCLV truth."
      ],
      "owner_contract": "modules/sclv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sclv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl revalidates one exact installed SCLV engine and invokes project without absorbing SCLV or projection authority.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Projection parsing, limits, deadlines, errors, and response binding use the shared bounded process contract.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
          "type": "depends_on"
        },
        {
          "rationale": "Projection input evidence uses the shared no-follow content-addressed snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.content-addressed-evidence-snapshots",
          "type": "depends_on"
        },
        {
          "rationale": "History projection is available only when the canonical ledger passes bounded append-only assurance.",
          "target_feature_id": "ssfv:symphony:sclv-engine.append-only-ledger-assurance",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sclv-engine",
      "status": "experimental",
      "title": "Disposable provider-neutral change-history projection",
      "what": "Emits a portable content-addressed JSON history that gives canonical SCLV records a bounded provider-neutral consumption shape while preserving explicit legacy gaps.",
      "when": "On explicit `qxctl sclv project` or exact local process invocation after the canonical ledger passes bounded assurance.",
      "where": "Returned in-process from one exact inactive-undocked SCLV engine installation and never persisted by the engine.",
      "who": "Any permission-backed caller using qxctl or the exact process, reviewers, release tooling, audit consumers, integration tooling, and tests.",
      "why": "Downstream consumers need stable reviewed-change history without depending on one forge format or mistaking a generated view for the append-only source record."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV owns the exact v3 record and proposal semantics implemented by this capability.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the affected and indexed reference paths bound into the proposal read set.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs publication evidence that may later consume a merged and canonically recorded closure.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "Binds one caller-declared v3 append candidate to normalized evidence; provider normalization creates the input envelopes but does not propose a ledger record.",
          "target_feature_id": "ssfv:symphony:sclv-engine.provider-neutral-evidence-normalization"
        }
      ],
      "evidence": [
        "modules/sclv-engine/tests/sclv_test.cpp verifies deterministic rendered v3 proposals, repository and provider-evidence binding, duplicate refusal, nondecreasing recording time, exact read and write sets, false ratification, and disabled apply.",
        "knowledge/sclv/schemas/v3/proposal-input.schema.json defines the bounded caller input accepted by the engine."
      ],
      "feature_id": "ssfv:symphony:sclv-engine.evidence-bound-append-proposals",
      "how": "Requires a clean ledger and one exact v3 record, matches normalized evidence to its repository revision and tree, change request, and asserted ratification claims, rejects duplicate identity and backward recording order, reads affected and indexed references without following symlinks, renders deterministic Markdown, and returns one immutable append proposal with exact prior-state and desired-change digests.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements exact v3 validation, provider-evidence matching, duplicate and chronology refusal, reference acquisition, deterministic Markdown rendering, and content-addressed proposal construction."
        }
      ],
      "implementation_paths": [
        "modules/sclv-engine/src/provider.cpp",
        "modules/sclv-engine/src/provider.hpp",
        "modules/sclv-engine/src/sclv.cpp",
        "modules/sclv-engine/src/sclv.hpp",
        "modules/sclv-engine/tests/sclv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not authenticate a subject, prove effective permission, ratify the record, append Markdown, commit Git state, close a request, or apply canonical content.",
        "qxctl validates the selected installation, invokes propose, and checks bounded result safety; it does not own SCLV truth, evidence meaning, or append authority."
      ],
      "owner_contract": "modules/sclv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sclv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl resolves and revalidates one exact installed SCLV engine before invoking propose, without gaining ratification or canonical-apply authority.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Proposal parsing, limits, deadlines, errors, and deterministic response binding use the shared bounded process contract.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
          "type": "depends_on"
        },
        {
          "rationale": "Proposal read sets and affected/indexed path evidence use the shared no-follow content-addressed snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.content-addressed-evidence-snapshots",
          "type": "depends_on"
        },
        {
          "rationale": "A prospective append is constructed only after current canonical history passes bounded ledger assurance.",
          "target_feature_id": "ssfv:symphony:sclv-engine.append-only-ledger-assurance",
          "type": "depends_on"
        },
        {
          "rationale": "Repository, change-request, and ratification claims must match one or more normalized provider-evidence envelopes.",
          "target_feature_id": "ssfv:symphony:sclv-engine.provider-neutral-evidence-normalization",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sclv-engine",
      "status": "experimental",
      "title": "Evidence-bound SCLV append proposals",
      "what": "Produces one deterministic immutable proposal for appending a caller-declared v3 reviewed-change record to exact current ledger state.",
      "when": "On explicit `qxctl sclv propose` or exact local process invocation after normalized evidence and the desired record are supplied and the current ledger is clean.",
      "where": "Inside one exact inactive-undocked SCLV engine installation, reading canonical SCLV and SKVI evidence and returning a non-applying local process result.",
      "who": "Any permission-backed caller using qxctl or the exact process, architects, maintainers, reviewers, closure tooling, and tests.",
      "why": "A reviewed-change record must be reproducible, provider-neutral, and stale-state-safe before ordinary review may append it to canonical history."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV owns interrupted-closure and late-recovery semantics implemented by this reconciliation operation.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the recovery schemas, SCLV engine implementation, and any exact references bound by a late proposal.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV may consume completed closure evidence but does not own SCLV recovery decisions.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "Reconciles a caller-supplied ephemeral closure journal into one safe forward action; append proposals validate a fully declared prospective record against the canonical ledger.",
          "target_feature_id": "ssfv:symphony:sclv-engine.evidence-bound-append-proposals"
        }
      ],
      "evidence": [
        "modules/sclv-engine/tests/sclv_test.cpp verifies exact journal digests, resume, abandon, no-op, late-recovery proposal, indeterminate refusal, state-specific proposal rules, and non-mutation markers.",
        "knowledge/sclv/schemas/v3/recovery-input.schema.json defines the bounded journal and observed-state evidence accepted by recovery."
      ],
      "feature_id": "ssfv:symphony:sclv-engine.forward-only-closure-recovery",
      "how": "Validates an exact digest-bound ephemeral journal and caller-observed closure state, refuses indeterminate evidence, maps still-open, closed-without-merge, and merged-already-recorded states to resume, abandon, or no-op, and permits merged-unrecorded state only to produce a fully evidence-bound late-recovery append proposal carrying the supplied reason.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements journal-shape and digest validation, deterministic observed-state reconciliation, ambiguity refusal, and nested late-recovery proposal construction."
        }
      ],
      "implementation_paths": [
        "modules/sclv-engine/src/provider.cpp",
        "modules/sclv-engine/src/provider.hpp",
        "modules/sclv-engine/src/sclv.cpp",
        "modules/sclv-engine/src/sclv.hpp",
        "modules/sclv-engine/tests/sclv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not discover remote merge state, mutate or delete a journal, append a late record, repair ambiguity, reopen a request, or roll canonical history backward.",
        "qxctl transports explicit recovery evidence and validates the exact installed response; it does not select domain truth or turn a recommendation into canonical action."
      ],
      "owner_contract": "modules/sclv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sclv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl revalidates one exact installed SCLV engine and invokes explicit recovery without gaining journal, ledger, or semantic authority.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Recovery composes with current append-only ledger assurance when classifying already-recorded and unrecorded closure evidence.",
          "target_feature_id": "ssfv:symphony:sclv-engine.append-only-ledger-assurance",
          "type": "composes_with"
        },
        {
          "rationale": "The merged-unrecorded path reuses the same fully evidence-bound append-proposal capability with an explicit late-recovery disposition.",
          "target_feature_id": "ssfv:symphony:sclv-engine.evidence-bound-append-proposals",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/sclv-engine",
      "status": "experimental",
      "title": "Forward-only interrupted-closure recovery",
      "what": "Produces one deterministic non-mutating recovery disposition for exact interrupted SCLV closure evidence, including a late-record proposal only when a merge is proven unrecorded.",
      "when": "On explicit `qxctl sclv recover` or exact local process invocation after a previous closure attempt leaves a bounded journal requiring reconciliation.",
      "where": "Inside one exact inactive-undocked SCLV engine installation using caller-supplied local journal and observed-state evidence without network or provider access.",
      "who": "Any permission-backed caller using qxctl or the exact process, architects, maintainers, recovery operators, reviewers, and tests.",
      "why": "Interrupted or out-of-order review workflows need a retry-safe path that moves only toward truthful closure and cannot hide ambiguity or rewrite permanent history."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV owns the provider-neutral repository, change-request, and ratification evidence contract implemented by the adapters.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the adapter executables, provider implementation, schemas, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future publication of the installed package containing both adapter executables.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        }
      ],
      "distinctions": [
        {
          "distinction": "Creates bounded common evidence envelopes from local Git or caller-declared air-gap inputs; append proposals decide only whether supplied envelopes match one desired record.",
          "target_feature_id": "ssfv:symphony:sclv-engine.evidence-bound-append-proposals"
        }
      ],
      "evidence": [
        "modules/sclv-engine/tests/sclv_test.cpp verifies local-Git SHA-1 and SHA-256 normalization, recursive tree binding, air-gap combined evidence, strict fields, digests, limits, and non-asserted versus asserted ratification behavior.",
        "modules/sclv-engine/tests/process_smoke.sh exercises both separately discoverable adapter processes through the common bounded protocol."
      ],
      "feature_id": "ssfv:symphony:sclv-engine.provider-neutral-evidence-normalization",
      "how": "The local-Git adapter executes fixed shell-free `/usr/bin/git` commit and recursive-tree commands under a bounded environment, output ceiling, and deadline; the air-gap adapter validates caller-declared repository, change-request, and asserted-ratification evidence; both emit the same strict digest-bound provider-evidence envelope.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements common evidence validation and digests, bounded fixed local-Git collection, air-gap normalization, and separate process entry points."
        }
      ],
      "implementation_paths": [
        "modules/sclv-engine/src/airgap_main.cpp",
        "modules/sclv-engine/src/local_git.cpp",
        "modules/sclv-engine/src/local_git.hpp",
        "modules/sclv-engine/src/local_git_main.cpp",
        "modules/sclv-engine/src/provider.cpp",
        "modules/sclv-engine/src/provider.hpp",
        "modules/sclv-engine/tests/process_smoke.sh",
        "modules/sclv-engine/tests/sclv_test.cpp"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Well-formed normalized evidence is not proof of identity, permission, legal authority, ratification, change-request state, or canonical acceptance.",
        "qxctl receipt-validates and can invoke either exact adapter process, but it treats normalized output only as caller-controlled evidence and does not convert it into truth, permission, ratification, or canonical acceptance.",
        "Neither adapter uses a network, shell, forge-specific API, provider mutation, credential, ledger append, or canonical apply path."
      ],
      "owner_contract": "modules/sclv-engine/SPEC.md",
      "parent_feature_id": "ssfv:symphony:sclv-engine",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl's binding validation closes over both exact receipt-v2 adapter entry points and its bounded evidence commands validate their full output and digest without gaining provider-evidence authority.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        },
        {
          "rationale": "Both adapter processes use the shared bounded request, deadline, stable-error, and deterministic response contract.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.bounded-process-protocol",
          "type": "depends_on"
        },
        {
          "rationale": "Both adapters use the shared strict temporal representation validators for observed evidence timestamps.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation.temporal-representation-conformance",
          "type": "depends_on"
        },
        {
          "rationale": "Normalized repository, change-request, and ratification envelopes enable evidence-bound SCLV append proposals.",
          "target_feature_id": "ssfv:symphony:sclv-engine.evidence-bound-append-proposals",
          "type": "enables"
        }
      ],
      "source_scope": "modules/sclv-engine",
      "status": "experimental",
      "title": "Provider-neutral change evidence normalization",
      "what": "Normalizes local Git revision evidence and explicitly declared air-gapped repository, change-request, and ratification evidence into one bounded provider-independent envelope.",
      "when": "Before an SCLV append proposal when a caller needs reproducible repository evidence or a common envelope for an environment without a forge dependency.",
      "where": "In two separately discoverable local executables shipped within one exact inactive-undocked SCLV engine installation; neither exposes a listener.",
      "who": "Any permission-backed caller invoking an exact adapter, offline operators, local-Git users, closure tooling, reviewers, and tests.",
      "why": "SCLV closure must remain usable with GitHub, GitLab, proprietary systems, or air-gapped workflows without making any one external provider a source of Symphony authority."
    }
  ],
  "source_scope": "modules/sclv-engine"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
