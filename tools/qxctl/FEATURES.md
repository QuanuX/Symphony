# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "tools/qxctl/MANIFEST.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "qxctl invokes exact installed SACV engine operations.",
          "reference": "knowledge/sacv/SPEC.md",
          "vector": "sacv"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed qxctl changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes qxctl commands and clients.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "qxctl invokes SODV evidence operations and SODV governs release truth.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns the authorization decisions and protected local policy contracts consumed by qxctl.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns the audit ledgers qxctl verifies and queries without writing.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "The platform record defines the whole modular boundary; qxctl is its administrative CLI rather than the platform itself.",
          "target_feature_id": "ssfv:symphony:platform"
        }
      ],
      "evidence": [
        "The qxctl Go suites exercise command grammar, exact installation resolution, endpoint trust, digest validation, state durability, replay, recovery, and lifecycle compatibility.",
        "Lifecycle host tests prove descriptor compare-and-swap, report-only systemd grammar, content-addressed fallback promotion, and resumable marker-owned uninstall.",
        "The root command exposes implemented administrative surfaces while namespace-reserved or unauthorized mutation leaves fail closed.",
        "SSIAG client and CLI tests cover bounded no-follow policy input, closed proposal/apply/recovery protocols, TOPS binding, and caller-neutral result checks."
      ],
      "feature_id": "ssfv:symphony:qxctl",
      "how": "Cobra/Viper grammar invokes exact installed components, authenticates local endpoints, requests fresh operation-bound SSIAG decisions, administers server-owned policy protocols from bounded files, serializes shared-root multi-profile receipt claims with receipt-layout old-client fencing and conservative reclamation, installs and reconciles an explicit content-addressed Linux report-only boot receptor, validates nested evidence and digests, preserves compare-and-swap state, and presents bounded human or JSON output.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the Cobra/Viper CLI, exact local clients, protected administrative state, lifecycle orchestration, response validation, and caller-neutral command grammar."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/lifecycle_host.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/cmd/qxctl/named_version.go",
        "tools/qxctl/cmd/qxctl/named_version_test.go",
        "tools/qxctl/internal/knowledgeengine/client.go",
        "tools/qxctl/internal/knowledgelifecycle/executor.go",
        "tools/qxctl/internal/knowledgelifecycle/host.go",
        "tools/qxctl/internal/knowledgelifecycle/host_admin_unix.go",
        "tools/qxctl/internal/knowledgelifecycle/ownership.go",
        "tools/qxctl/internal/ssiagclient/client.go",
        "tools/qxctl/internal/stavclient/client.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not own canonical vector schemas, grant host permission, classify callers, write STAV ledgers, or autonomously ratify proposals.",
        "Does not install hidden host hooks, login/session hooks, or native Windows receptors; download packages; execute arbitrary receipt entry points; or participate in hot/warm trading paths."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl collects and authorizes evidence while the coordinator serializes and verifies durable administrative transitions.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator",
          "type": "composes_with"
        },
        {
          "rationale": "qxctl administers exact authenticated Maestro presence and inventory operations.",
          "target_feature_id": "ssfv:symphony:maestro-presence-authority",
          "type": "composes_with"
        },
        {
          "rationale": "qxctl authenticates the SSIAG endpoint and consumes exact caller-neutral decisions without owning policy truth.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "Caller-neutral Symphony administrative CLI",
      "what": "Provides Symphony's canonical command-line administration and query surface across repository inspection, validation, SSIAG decisions and local policy administration, STAV, knowledge engines, sessions, lifecycle convergence, the explicit Linux report-only boot receptor, and Maestro presence.",
      "when": "Runs on explicit invocation or through the reviewed Linux report-only systemd receptor; no command becomes a hidden daemon or hot-path dependency.",
      "where": "Executes on a supported administrative node and may administer local or later explicitly contracted remote TOPS nodes while remaining outside trading hot and warm paths.",
      "who": "Target-host owners, administrators, maintainers, agentic tools, automation, and any caller operating within effective host permission.",
      "why": "Gives all authorized callers one stable, scriptable, provider-neutral administrative voice without making the CLI schema owner or runtime authority."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed session and authorization-boundary changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes session grammar, clients, tests, and coordinator contracts.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns caller authentication and exact permission decisions.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG releases a usable security decision only after its safe audit event is committed through STAV.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Session administration consumes fresh SSIAG decisions for coordinator operations; SSIAG administration manages and inspects SSIAG policy and provider surfaces themselves.",
          "target_feature_id": "ssfv:symphony:qxctl.ssiag-administration"
        }
      ],
      "evidence": [
        "tools/qxctl/cmd/qxctl/session_test.go verifies explicit login, refresh, logout, retry, bounded recovery, reauthentication, and interrupted-close resumption.",
        "tools/qxctl/internal/ssiagclient/client_test.go verifies endpoint trust, caller-neutral authorization results, resource binding, and bounded response validation."
      ],
      "feature_id": "ssfv:symphony:qxctl.authenticated-sessions",
      "how": "qxctl resolves one exact coordinator, authenticates the configured SSIAG endpoint, obtains a fresh operation-bound decision, validates its complete safe evidence, and transports it with exact expected-state and stable operation identity.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements explicit session commands, host-event transition composition, exact SSIAG exchange, decision validation, and coordinator invocation."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/cmd/qxctl/session_test.go",
        "tools/qxctl/internal/knowledgeengine/client.go",
        "tools/qxctl/internal/ssiagclient/client.go",
        "tools/qxctl/internal/ssiagclient/client_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not install login hooks, infer a session from process lifetime, or make SSIAG decisions transferable bearer authority.",
        "Does not own policy truth, coordinator journals, or canonical knowledge mutation."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl obtains authorization and composes commands while the coordinator owns epoch validation, durability, and recovery.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.authority-epochs",
          "type": "composes_with"
        },
        {
          "rationale": "SSIAG authenticates the host subject and issues the exact non-transferable decision evidence consumed by qxctl and the coordinator.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "Authenticated knowledge-session administration",
      "what": "Administers explicit begin, status, checkpoint, close, recovery, and login/refresh/logout composition for durable knowledge authority epochs.",
      "when": "Runs on explicit session commands or explicit host-event transition invocations; no hidden login hook or watcher is installed.",
      "where": "Executes on the administrative TOPS node against local protected SSIAG and coordinator process boundaries.",
      "who": "Any target-host-authorized caller, including administrators, automation, and agentic owners or delegates, operating through qxctl.",
      "why": "Connects host permission to durable caller-neutral session evidence without treating caller class, a token, or the CLI itself as authority."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed command-identity and registry-contract changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the qxctl registry implementation, checked-in projection, documentation, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "The command registry declares stable qxctl grammar and semantic bindings; the SSFV engine independently evaluates whether registered features and engine operations have complete administration coverage.",
          "target_feature_id": "ssfv:symphony:ssfv-engine.administration-assurance"
        }
      ],
      "evidence": [
        "tools/qxctl/internal/commandregistry/registry_test.go verifies stable identity, complete Cobra parity, duplicate rejection, explicit structural and prohibited roles, canonical digests, and observed executable binding.",
        "tools/qxctl/cmd/qxctl/command_manifest_test.go verifies expected/observed registry generation, checked-in projection parity, strict bounded verification, exact feature bindings, and known JSON-output debt.",
        "tools/qxctl/cmd/qxctl/ssfv_test.go verifies the bounded headless administration-check route, exact payload transport, canonical result validation, and digest tamper rejection."
      ],
      "feature_id": "ssfv:symphony:qxctl.command-registry",
      "how": "Every public or hidden executable Cobra leaf carries one attached CommandSpec with a stable qxcmd identity and explicit feature interaction, backend operation, mutability, authority, scope, protocol, validation, noninteractive, and JSON-output evidence. One parity walk rejects unclassified or duplicate grammar, derives the machine projection from the live tree, emits a client-independent expected registry or executable-bound observed registry, and verifies checked-in expected evidence with canonical self-digests. Hidden prohibited leaves remain visible registry evidence but cannot satisfy required administration coverage.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements single-source command metadata attachment, full Cobra-tree parity, canonical expected and observed registry projection, strict bounded verification, and administration-check transport."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/COMMANDS.json",
        "tools/qxctl/COMMANDS.md",
        "tools/qxctl/cmd/qxctl/command_manifest.go",
        "tools/qxctl/cmd/qxctl/command_manifest_test.go",
        "tools/qxctl/cmd/qxctl/command_specs.go",
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/ssfv.go",
        "tools/qxctl/cmd/qxctl/ssfv_test.go",
        "tools/qxctl/internal/commandregistry/registry.go",
        "tools/qxctl/internal/commandregistry/registry_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not dynamically inject commands, infer or allocate stable command identities, invent feature names, or use an external suggestion service at runtime.",
        "Does not decide semantic worthiness, administration requirement, authorization, ratification, engine truth, runtime availability, or canonical mutation."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The registry supplies exact expected and observed qxctl evidence while the engine owns deterministic coverage and module-admission evaluation.",
          "target_feature_id": "ssfv:symphony:ssfv-engine.administration-assurance",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "Stable qxctl command identity and coverage registry",
      "what": "Defines one stable machine identity and explicit administration contract for every public or hidden executable qxctl leaf, with checked-in expected evidence and runtime observed evidence generated from the same Cobra tree.",
      "when": "Parity is validated whenever qxctl constructs its command tree; expected evidence is generated or verified during development and repository validation, and observed evidence is emitted only on explicit invocation.",
      "where": "Runs entirely inside the local qxctl process and leaves a repository-owned expected registry that the SSFV engine can evaluate when no qxctl executable is present.",
      "who": "Any target-host-authorized caller or reviewed development workflow consuming the published protocol without receiving extra authority from caller class.",
      "why": "Makes a newly added module or engine operation visibly uncovered until a reviewed qxctl command, explicit exception, or prohibition is ratified, preventing administrative surfaces from disappearing behind implementation details."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed binding and compatibility changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the binding registry, engine client, commands, tests, and contracts.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "The parent feature covers the complete administrative CLI; this nested feature only selects and revalidates exact installed processes for bounded invocation.",
          "target_feature_id": "ssfv:symphony:qxctl"
        }
      ],
      "evidence": [
        "tools/qxctl/internal/knowledgebinding/registry_test.go verifies protected compare-and-swap binding state and exact installation identity.",
        "tools/qxctl/internal/knowledgeengine/client_test.go verifies receipt-v1/v2 dual-read behavior, role closure, content/digest tamper rejection, and exact receipt-v2 typed SCLV evidence-adapter resolution.",
        "tools/qxctl/cmd/qxctl/sclv_test.go verifies adapter identity, exact normalized result fields, nested evidence semantics, evidence digests, and rejection of ratification escalation."
      ],
      "feature_id": "ssfv:symphony:qxctl.engine-bindings",
      "how": "qxctl validates immutable install receipts and executable digests, stores a no-follow owner-only compare-and-swap registry, preserves receipt-v1/v2 compatibility for main engines, and invokes only the exact recorded local process through the bounded engine protocol. SCLV evidence normalization additionally requires an exact receipt-v2 typed adapter entry point, revalidates the complete package, and validates the adapter response identity, exact field set, nested semantics, whole-second UTC timestamp, and content digest before presentation.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements protected binding profiles, receipt and executable revalidation, exact process invocation, compare-and-swap state, and command grammar."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/internal/knowledgebinding/registry.go",
        "tools/qxctl/internal/knowledgebinding/registry_test.go",
        "tools/qxctl/internal/knowledgebinding/state_unix.go",
        "tools/qxctl/internal/knowledgeengine/client.go",
        "tools/qxctl/internal/knowledgeengine/client_test.go",
        "tools/qxctl/internal/knowledgeengine/open_relative_unix.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not install, uninstall, activate, dock, download, or infer the newest engine version.",
        "Does not grant permission, establish an authenticated session, decide truth or ratification, mutate canonical knowledge, or apply an adapter result."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The binding client invokes engines that implement the common bounded process and receipt contracts.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "composes_with"
        },
        {
          "rationale": "qxctl invokes the exact receipt-owned SCLV adapter and validates its bounded normalization result without converting source evidence into truth, permission, ratification, or canonical apply authority.",
          "target_feature_id": "ssfv:symphony:sclv-engine.provider-neutral-evidence-normalization",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "Exact installed knowledge-engine bindings",
      "what": "Binds each closed knowledge-engine role to one exact inactive-undocked installed version, revalidates that installation before every invocation, and exposes exact receipt-owned SCLV evidence normalization adapters without adding them as binding roles.",
      "when": "Runs only on explicit binding list, show, set, remove, doctor, an operation that resolves a bound engine, or explicit SCLV local-Git or air-gap evidence normalization.",
      "where": "Stores noncanonical user-scope state below the configured qxctl state root and executes the selected local installation outside hot and warm paths.",
      "who": "Any caller holding effective target-host permission and using qxctl to administer its protected user-default engine bindings.",
      "why": "Lets independently installed engine versions be selected and swapped without mutable aliases, version-recency inference, or implementation absorption into qxctl."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed validation detection and administration changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI is an indexed surface whose deterministic coverage is validated and routed by the validator contracts.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "Validation evaluates repository evidence and warning policy; lifecycle convergence mutates only protected noncanonical installation and presence state through separate authorization and journals.",
          "target_feature_id": "ssfv:symphony:qxctl.lifecycle-convergence"
        }
      ],
      "evidence": [
        "tools/qxctl/internal/validation/validation_test.go verifies exact process output, finding identities, immutable detection, subject and occurrence history, classifications, expiry, supersession, presentation-only mute, compare-and-swap state, root-summary exit 25, and filters that never narrow scanning.",
        "tools/qxctl/cmd/qxctl/validation.go implements scan, debug, profile, baseline, warning-lifecycle, and root-summary grammar over the bounded client and protected state."
      ],
      "feature_id": "ssfv:symphony:qxctl.governed-validation",
      "how": "qxctl validates the validator receipt, invokes complete validation or the distinct root-summary projection with an empty environment and deadline, verifies exact identities and nested digests, then applies protected record/review/require policy and a side-by-side subject-aware warning lifecycle after detection.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements exact validator invocation, nested evidence and root-summary validation, immutable detection evaluation, protected warning profiles/baselines/subject lifecycle, and actionable/debug presentation."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/cmd/qxctl/validation.go",
        "tools/qxctl/internal/validation/client.go",
        "tools/qxctl/internal/validation/digest.go",
        "tools/qxctl/internal/validation/evaluate.go",
        "tools/qxctl/internal/validation/policy.go",
        "tools/qxctl/internal/validation/state_unix.go",
        "tools/qxctl/internal/validation/validation_test.go",
        "tools/qxctl/internal/validation/warnings.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not suppress, rewrite, delete, administratively declare resolution of, or ratify raw findings and does not change detector execution sensitivity.",
        "Does not make a baseline or warning classification canonical truth, grant permission, mutate repository source, or write README."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The validator owns deterministic raw findings while qxctl validates and governs their downstream disposition and presentation.",
          "target_feature_id": "ssfv:symphony:symphony-validator",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "Governed repository validation evidence",
      "what": "Runs the exact installed Symphony Validator, inspects its root-summary projection, and administers warning disposition, actionable subject lifecycle, occurrence history, and presentation without changing raw detector truth.",
      "when": "Runs on explicit validate scan, debug, root-summary, profile, baseline, or warning commands and during reviewed gates that request validator evidence.",
      "where": "Executes on an administrative node against a selected repository and protected per-TOPS validation state outside runtime trading paths.",
      "who": "Any caller with effective target-host permission using qxctl for repository review, debugging, baselining, or release evidence.",
      "why": "Separates immutable defect and drift detection from administrator-controlled sensitivity, acknowledgement, lifecycle classification, and presentation so ordinary output is actionable while complete historical evidence remains visible and diagnosable."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed invariant-administration and assurance changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the common invariant contracts, query schema, qxctl implementation, and validator evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "The command registry records executable qxctl leaves; invariant assurance exposes the separately owned rule-to-owner and regression registry without inventing commands or invariants.",
          "target_feature_id": "ssfv:symphony:qxctl.command-registry"
        }
      ],
      "evidence": [
        "tools/qxctl/internal/invariantregistry/registry_test.go verifies bounded no-follow loading, exact shape and digest checks, ordering, hostile-text refusal, and digest-bearing status/list/show projections.",
        "tools/qxctl/cmd/qxctl/knowledge_invariant_test.go verifies four stable read-only noninteractive commands and preservation of the installed validator's exact invariant-assurance exit status.",
        "knowledge/INVARIANTS.md keeps semantic ownership in the common contract while qxctl remains a replaceable consumer and administrative projection.",
        "Focused v3 producer and consumer regressions reject older-version widening, substituted owner paths and duplicate operation ownership. Test execution is recorded in the change closure."
      ],
      "feature_id": "ssfv:symphony:qxctl.invariant-assurance",
      "how": "qxctl performs a bounded consumer-side identity, shape, ordering, reference, and omit-self digest check for status/list/show, marks semantic validity as not asserted, and delegates complete assurance to one exact receipt-validated Symphony Validator invocation.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements caller-neutral Cobra grammar, bounded no-follow registry consumption, deterministic digest-bearing projections, and exact installed-validator mediation."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/knowledge_invariant.go",
        "tools/qxctl/cmd/qxctl/knowledge_invariant_test.go",
        "tools/qxctl/internal/invariantregistry/registry.go",
        "tools/qxctl/internal/invariantregistry/registry_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not create, name, modify, ratify, execute, or repair an invariant or its regression suite.",
        "Does not claim a partial query proves semantic validity, complete legacy coverage, installed-host inventory completeness, or authorization."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The validator produces complete repository assurance while qxctl provides stable headless inspection and exact installed invocation.",
          "target_feature_id": "ssfv:symphony:symphony-validator.invariant-ownership-assurance",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "Headless invariant ownership administration",
      "what": "Provides stable status, list, show, and complete-check commands for the common incremental invariant ownership registry with deterministic structured output for headless callers. Versioned v3 registry admission preserves exact earlier adapter definitions and independently named process adapters with explicit module/entrypoint ownership and unique operation declarations.",
      "when": "Runs only on explicit local qxctl invocation during development, review, integration admission, or diagnostics.",
      "where": "Executes on an administrative node against one selected repository and one exact independently installed validator, outside hot and warm paths.",
      "who": "Target-host administrators, independent module developers, automated release gates, and agentic callers using qxctl as the administrative spine.",
      "why": "Makes lowest-owner obligations and missing evidence directly inspectable without requiring AI, guessing semantic names, or turning qxctl into the rule authority."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Maestro owns exact authenticated receptor-presence outcomes.",
          "reference": "modules/maestro/SPEC.md",
          "vector": "maestro"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed lifecycle protocol and implementation changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes lifecycle contracts, schemas, implementation, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Every protected lifecycle phase consumes a fresh exact caller-neutral authorization decision.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "Security-relevant lifecycle authorization outcomes enter the governed audit circuit through SSIAG.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Lifecycle convergence changes explicit desired and observed state; an engine binding only selects an already installed exact local process for invocation.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        }
      ],
      "evidence": [
        "tools/qxctl/cmd/qxctl/lifecycle_test.go verifies report/apply grammar, dynamic scheduler invariants, journal binding, and exact result validation.",
        "tools/qxctl/internal/knowledgelifecycle/executor_test.go verifies prepared-state refusal, staged receipt-v2 actions, interruption replay, rollback proof, and Maestro adapter evidence.",
        "tools/qxctl/cmd/qxctl/lifecycle_binding_test.go verifies forward upgrade, inverse rollback, and exact predecessor evidence for established roles."
      ],
      "feature_id": "ssfv:symphony:qxctl.lifecycle-convergence",
      "how": "qxctl collects complete observations, obtains fresh SSIAG decisions, invokes the exact coordinator, executes only reviewed adapters after a durable prepared attempt, re-observes the host, and finalizes verified outcomes while supporting forward and inverse ordering.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements desired-state administration, fixed-layout observation, ownership fencing, reviewed external adapters, dynamic retries, and command orchestration."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/lifecycle.go",
        "tools/qxctl/cmd/qxctl/lifecycle_apply.go",
        "tools/qxctl/cmd/qxctl/lifecycle_binding_test.go",
        "tools/qxctl/cmd/qxctl/lifecycle_test.go",
        "tools/qxctl/internal/knowledgelifecycle/executor.go",
        "tools/qxctl/internal/knowledgelifecycle/executor_test.go",
        "tools/qxctl/internal/knowledgelifecycle/observation.go",
        "tools/qxctl/internal/knowledgelifecycle/ownership.go",
        "tools/qxctl/internal/knowledgelifecycle/profile.go",
        "tools/qxctl/internal/knowledgelifecycle/runtime.go",
        "tools/qxctl/internal/knowledgelifecycle/scan_unix.go",
        "tools/qxctl/internal/knowledgelifecycle/state_unix.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not download packages, mutate receipt v1, execute arbitrary receipt entry points, activate live component processes, or execute docked engines.",
        "Does not choose a newest version, bypass blockers, create new binding roles, or mutate canonical knowledge."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The coordinator serializes and verifies actions while qxctl owns observation, authorization exchange, and the reviewed host adapters.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.lifecycle-apply-coordination",
          "type": "composes_with"
        },
        {
          "rationale": "Dock and undock actions use Maestro's authenticated durable receptor-presence authority.",
          "target_feature_id": "ssfv:symphony:maestro-presence-authority",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "Two-way module lifecycle convergence",
      "what": "Converges independently versioned modules and vector engines toward explicit desired profiles through report-only planning and separately authorized apply-compatible execution.",
      "when": "Runs on explicit profile, observe, report, boot, apply, status, ownership, or recovery commands and after an administrator chooses to reconcile changed installations.",
      "where": "Operates on supported Linux-first administrative TOPS nodes and protected per-TOPS/profile state outside trading hot and warm paths.",
      "who": "Target-host-authorized lifecycle administrators, automation, agentic owners or delegates, and recovery operators using qxctl.",
      "why": "Allows modules to be added, removed, upgraded, rolled back, docked, and undocked in unplanned order while preserving evidence and healing localized dependency ordering failures."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed host-receptor contracts and implementation.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the receptor grammar, implementation, tests, and lifecycle ownership contract.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "The invoked report-only lifecycle operation remains independently authorized under the normal lifecycle boundary.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        }
      ],
      "distinctions": [
        {
          "distinction": "This receptor represents an explicit host boot trigger for report-only lifecycle work; it is not a user login/session hook.",
          "target_feature_id": "ssfv:symphony:qxctl.authenticated-sessions"
        }
      ],
      "evidence": [
        "tools/qxctl/internal/knowledgelifecycle/host_test.go verifies descriptor compare-and-swap, report-only systemd grammar, content-addressed fallback promotion, and resumable marker-owned uninstall."
      ],
      "feature_id": "ssfv:symphony:qxctl.linux-host-receptor",
      "how": "qxctl renders fixed grammar, validates paths and ownership, records compare-and-swap marker state, promotes content-addressed fallback definitions, and performs resumable marker-owned uninstall without accepting arbitrary unit content.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements report-only systemd descriptor generation, protected marker state, installation, comparison, removal, and recovery."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/lifecycle_host.go",
        "tools/qxctl/internal/knowledgelifecycle/host.go",
        "tools/qxctl/internal/knowledgelifecycle/host_admin_unix.go",
        "tools/qxctl/internal/knowledgelifecycle/host_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not install a login hook, hidden watcher, native Windows service, or live apply daemon.",
        "Does not expand the report-only boot invocation into package, binding, Maestro, or canonical mutation authority."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The receptor provides one explicit boot-time entry into the existing report-only lifecycle circuit.",
          "target_feature_id": "ssfv:symphony:qxctl.lifecycle-convergence",
          "type": "extends"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "Explicit Linux report-only boot receptor",
      "what": "Provides a content-addressed systemd receptor that can invoke qxctl lifecycle boot in report-only mode after host boot.",
      "when": "Runs only after explicit administrative installation and at the configured systemd boot trigger; removal and repair are explicit.",
      "where": "Applies to supported Linux hosts; Windows uses WSL or a remotely administered TOPS node and receives no native receptor.",
      "who": "An authorized Linux host administrator explicitly enabling or maintaining the qxctl report-only receptor.",
      "why": "Completes a configurable default installation point for report-only convergence while keeping host hooks visible, removable, and separate from mutation authority."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Maestro owns receptor-presence state and complete inventory semantics.",
          "reference": "modules/maestro/SPEC.md",
          "vector": "maestro"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed Maestro client and lifecycle adapter changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes Maestro contracts, client, command, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Every protected Maestro operation consumes a fresh exact caller-neutral decision.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG releases Maestro authorization only after its safe audit event is committed through STAV.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Maestro administration records receptor presence for an installed component; engine binding selects an exact process for qxctl invocation and never establishes docking.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings"
        }
      ],
      "evidence": [
        "tools/qxctl/internal/maestroclient/client_test.go verifies exact installation identity, authorization binding, inventory validation, mutation retry, and recovery evidence.",
        "tools/qxctl/cmd/qxctl/lifecycle_test.go verifies exact Maestro installation and exhaustive receptor evidence at the lifecycle boundary."
      ],
      "feature_id": "ssfv:symphony:qxctl.maestro-administration",
      "how": "qxctl validates the exact Maestro receipt and executable, obtains fresh operation-bound SSIAG evidence, invokes the bounded local process, and verifies identity, retry, inventory, mutation, and recovery results.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements exact Maestro installation validation, fresh authorization exchange, receptor command construction, response validation, and CLI grammar."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/maestro.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/internal/maestroclient/client.go",
        "tools/qxctl/internal/maestroclient/client_test.go",
        "tools/qxctl/internal/ssiagclient/client.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not execute, schedule, supervise, load, or health-check docked engines.",
        "Does not install modules, select active engine bindings, mutate canonical truth, or own Maestro state."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl is the authenticated administrative client for Maestro's durable presence authority.",
          "target_feature_id": "ssfv:symphony:maestro-presence-authority",
          "type": "composes_with"
        },
        {
          "rationale": "Lifecycle convergence uses the same exact client as its reviewed docking and undocking adapter.",
          "target_feature_id": "ssfv:symphony:qxctl.lifecycle-convergence",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "Authenticated Maestro receptor administration",
      "what": "Administers complete inventory, status, dock, undock, and recovery against one exact installed Maestro presence authority.",
      "when": "Runs on explicit Maestro commands and as the reviewed docking adapter inside separately authorized lifecycle convergence.",
      "where": "Executes locally on the Maestro-hosting TOPS node against protected per-TOPS receptor state.",
      "who": "Any caller with exact target-host Maestro permission, including lifecycle administrators, automation, and agentic owners or delegates.",
      "why": "Provides a stable administrative receptor interface for independently versioned engines without allowing qxctl or Maestro to execute, schedule, or supervise them."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Owns source/evidence meanings and exact package-to-composition workflow boundaries; qxctl coordinates without acquiring native semantic authority.",
          "reference": "knowledge/scv/COMPOSITION-WORKFLOWS.md",
          "vector": "scv"
        },
        {
          "applicability": "applicable",
          "reason": "Owns authenticated decisions and their committed audit path.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "Distinguishes committed authorization audit evidence from source selection mutation evidence.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [],
      "evidence": [
        "Receipt and result boundary tests reject domain/version drift, tampered executables, changed capture fields and oversized bodies.",
        "Source transaction tests cover preauthorization intent, absent authorization, stale compare-and-swap, operation identity collisions, interrupted writes, recovery and unsafe filesystem links.",
        "Transport tests qualify partial/failed captures, retain exact version queries and reject private targets, credential-bearing authorities and unsafe redirects.",
        "Corpus producer/consumer and storage regressions bind referenced bytes to indexes, preserve failed-refresh evidence, checkpoint jobs and verify retained old-version invocation.",
        "Profile/connection consumer and exact installed-process tests reject changed extraction evidence, resealed comparison results and changed reassessment bindings; old exact versions remain invocable.",
        "Agent workflow regressions cover replay, collisions, interrupted stages, altered evidence and machine-readable failures; installed acceptance remains bound to the recorded build.",
        "Provider coverage owner/consumer and installed-process regressions exercise exact joins, source declaration drift, gap inventories and replayed attempts without inferring complete expertise or runtime compatibility.",
        "Generated-interface tests pin historical exact operation/retention admissions, reject malformed declarations and schema drift, and exercise installed interface discovery and changed receipt-owned bytes. Independent pack/composition consumer tests remain separate from generated metadata; execution scope is recorded in the increment closure.",
        "Exact .7 retained composition workflow producer, consumer, interruption and installed-process checks preserve original pack owners and fixed caller selections. Strict typed journal roundtrip rejects omitted/null fields that could otherwise normalize under an earlier seal. Execution evidence belongs to the increment closure.",
        "Owner and independent Go consumer regressions cover exact targets, preserved caller criteria, partial/contradictory evidence and resealed-result rejection. Retained links replay original owners and verify immutable input correspondence; execution evidence belongs to the increment closure.",
        "Native and independent Go regressions cover complete closure, exact full-value identities, malformed graphs and pre-allocation expansion budgets; installed qxctl evidence belongs to the increment closure. Raw bundle Unicode spelling and exact typed retained-record identity are independently checked before decoder normalization can alter sealed meaning.",
        "Versioned coordination regressions cover exact checkpoint reconstruction, original-owner replay, observational status, malformed retained evidence and interrupted publication. Installed complete-fixture evidence is recorded in the increment closure.",
        "Graph-index consumer and installed-process regressions cover independently reconstructed rows, namespace isolation, idempotent prepare/commit recovery, exact owner replay and missing-owner observations. Execution evidence remains in the increment closure."
      ],
      "feature_id": "ssfv:symphony:qxctl.scv-administration",
      "how": "Reuses receipt-v2 and version-specific process/result validation; bounds public HTTPS retrieval; retains immutable exact corpus artifacts with no-replace publication and recoverable job intent; separately applies authenticated SSIAG authorization to protected source/graph selection. Preserves exact authored interpretation wrappers and independently checks returned evidence, comparisons and reassessment bindings at the process boundary. Coordinates owner operations with pinned inputs and receipts; replays retained evidence before accepting progress and never substitutes current selections on recovery.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Administrative dispatch, exact-version process/result validation, public HTTPS transport, immutable corpus retention and recovery, and separate protected local persistence with authenticated permission consumption."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/cli_errors.go",
        "tools/qxctl/cmd/qxctl/cli_help.go",
        "tools/qxctl/cmd/qxctl/scv.go",
        "tools/qxctl/cmd/qxctl/scv_acquire.go",
        "tools/qxctl/cmd/qxctl/scv_bundle.go",
        "tools/qxctl/cmd/qxctl/scv_bundle_obligation.go",
        "tools/qxctl/cmd/qxctl/scv_bundle_workflow.go",
        "tools/qxctl/cmd/qxctl/scv_composition.go",
        "tools/qxctl/cmd/qxctl/scv_composition_workflow.go",
        "tools/qxctl/cmd/qxctl/scv_corpus.go",
        "tools/qxctl/cmd/qxctl/scv_graph.go",
        "tools/qxctl/cmd/qxctl/scv_graph_index.go",
        "tools/qxctl/cmd/qxctl/scv_graph_index_maintenance.go",
        "tools/qxctl/cmd/qxctl/scv_graph_index_schema.go",
        "tools/qxctl/cmd/qxctl/scv_graph_index_transfer.go",
        "tools/qxctl/cmd/qxctl/scv_graph_index_transfer_commands.go",
        "tools/qxctl/cmd/qxctl/scv_interface.go",
        "tools/qxctl/cmd/qxctl/scv_logical_reference.go",
        "tools/qxctl/cmd/qxctl/scv_obligation.go",
        "tools/qxctl/cmd/qxctl/scv_profile.go",
        "tools/qxctl/cmd/qxctl/scv_schema.go",
        "tools/qxctl/cmd/qxctl/scv_workflow.go",
        "tools/qxctl/internal/knowledgeengine/scv.go",
        "tools/qxctl/internal/knowledgeengine/scv_bundle.go",
        "tools/qxctl/internal/knowledgeengine/scv_bundle_file.go",
        "tools/qxctl/internal/knowledgeengine/scv_bundle_text.go",
        "tools/qxctl/internal/knowledgeengine/scv_composition.go",
        "tools/qxctl/internal/knowledgeengine/scv_graph_index.go",
        "tools/qxctl/internal/knowledgeengine/scv_graph_index_maintenance.go",
        "tools/qxctl/internal/knowledgeengine/scv_graph_index_schema.go",
        "tools/qxctl/internal/knowledgeengine/scv_interface_generated.go",
        "tools/qxctl/internal/knowledgeengine/scv_interpretation.go",
        "tools/qxctl/internal/knowledgeengine/scv_obligation.go",
        "tools/qxctl/internal/knowledgeengine/scv_owner_interface.go",
        "tools/qxctl/internal/knowledgeengine/scv_pack.go",
        "tools/qxctl/internal/knowledgeengine/scv_profile.go",
        "tools/qxctl/internal/knowledgeengine/scv_schema.go",
        "tools/qxctl/internal/scvcorpus/rename_darwin.go",
        "tools/qxctl/internal/scvcorpus/rename_linux.go",
        "tools/qxctl/internal/scvcorpus/storage_unix.go",
        "tools/qxctl/internal/scvcorpus/store.go",
        "tools/qxctl/internal/scvstate/storage_unix.go",
        "tools/qxctl/internal/scvstate/store.go",
        "tools/qxctl/internal/scvtransfer/journal.go",
        "tools/qxctl/internal/scvtransfer/schema.go",
        "tools/qxctl/internal/scvtransfer/storage_unix.go",
        "tools/qxctl/internal/scvtransport/https.go",
        "tools/qxctl/internal/scvworkflow/composition.go",
        "tools/qxctl/internal/scvworkflow/composition_storage_unix.go",
        "tools/qxctl/internal/scvworkflow/composition_v2.go",
        "tools/qxctl/internal/scvworkflow/filesystem_unix.go",
        "tools/qxctl/internal/scvworkflow/storage_unix.go",
        "tools/qxctl/internal/scvworkflow/store.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not own provider truth, infer arbitrary prose, execute downloaded text or install a universal runtime.",
        "Does not mutate canonical knowledge registries, create policy grants, append STAV events directly, or claim a source-write receipt from the existing SSIAG policy-decision audit.",
        "Public HTTPS transport supplies no credentials or account authentication; selected-source coverage is bounded and does not establish provider completeness.",
        "Does not introduce a current/latest corpus alias, a selected corpus head or an SSIAG/STAV write receipt for immutable evidence retention."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "C++ owns provider source and knowledge semantics; qxctl owns the administrative adapter.",
          "target_feature_id": "ssfv:symphony:scv-engine",
          "type": "composes_with"
        },
        {
          "rationale": "Administers the optional relational graph index through its exact installed C++ connector and independently validates retained evidence.",
          "target_feature_id": "ssfv:symphony:scv-graph-duckdb-connector",
          "type": "composes_with"
        },
        {
          "rationale": "Consumes fresh exact authenticated permission decisions without deriving subject or granting permission.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "SCV source and knowledge administration",
      "what": "Administers selected SCV family/provider source, capture, corpus, interpretation and graph operations through exact installed C++ engines; retains immutable corpus evidence separately from permission-backed source and graph selections. Invokes reusable profile interpretation, scoped connection checks and retained-result reassessment without selecting a graph head or claiming live provider compatibility. Provides exact installed schemas and profile preparation, structured SCV failures, immutable native artifact provenance and recoverable interpretation/evaluation runs; metadata corpus query is independent of capture export. Native provider coverage accounts separately for declared inventory, exact selected corpus evidence and independently replayed profile bindings, preserving unlisted, unselected, partial and stale gaps. Invokes portable provider-pack preparation/evaluation and finite caller-selected composition exploration/reassessment, retains their exact artifacts, and exposes the receipt-owned versioned owner interface. Generated release/operation metadata reduces repeated interface tables while independent consumers continue to reject forged semantic results. Exact .7 composition workflows checkpoint original-owner package evaluations, finite exploration and optional reassessment while retaining the caller request, root/TOPS and immutable provenance records. Observational status verifies sealed linkage without native replay. Exact obligation inventory identifies native checks independently of anonymous summaries; subsequent-evidence follow-up preserves caller criteria and policy, reports explicit query-time changes, and separates criterion state from opaque provenance and runtime verification. Complete evidence bundles deduplicate repeated containers, bound graph expansion before allocation, and preserve exact caller-selected composition semantics through independent owner validation. Explicit v2 bundle workflows and immutable obligation relationships preserve original owners, caller fields and separate logical/transport/record identities through local recovery. An optional receipt-owned C++ DuckDB connector retains immutable graph snapshots and derived structural indexes. qxctl independently validates exact row/page correspondence and replays the retained validating SCV owner at explicit query time; indexing does not select a graph head. Revision-bound inventory and exact caller-selected transfer planning preserve operation history, target requirements and explicit semantic-owner blockers without executing transfer or retirement.",
      "when": "Runs on explicit cold/freezing administrative invocation, including exact operation recovery.",
      "where": "Supported local macOS/Linux user-scoped TOPS evidence stores and separately protected source/graph state; pure operations accept an explicitly selected local installation.",
      "who": "Human or agentic callers using ordinary local filesystem permissions for evidence operations; protected source/graph selection additionally requires the same authenticated SSIAG permission contract.",
      "why": "Makes native knowledge acquisition and interpretation callable through qxctl while preserving source authority, exact engine provenance and caller-selected evidence policy."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Source contract discovery and exact closure traceability.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Material source change closes append-only after verification.",
          "reference": "knowledge/sclv/SPEC.md",
          "vector": "sclv"
        }
      ],
      "distinctions": [],
      "evidence": [
        "tools/qxctl/cmd/qxctl/shv_test.go exercises all native routes and exact-version grammar.",
        "tools/qxctl/internal/knowledgeengine/shv_test.go rejects resealed changed evidence and verifies receipt-backed installed process results.",
        "SHV-03 adds version-bound scoped-table replay, complete operating-profile rows and quarter-date consumer checks, with focused production-qxctl Intel/AMD acceptance.",
        "SHV-04 adds protected source CAS/journal tests and production qxctl with isolated real SSIAG/STAV acceptance; no global source-write STAV receipt is claimed."
      ],
      "feature_id": "ssfv:symphony:qxctl.shv-administration",
      "how": "Validates exact receipts, descriptors, bounded process responses and independently rederives coverage, source-backed catalogue claims, evaluation, graph projection and adapter row correspondence.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Exact installation discovery, process consumption, independent evidence checks and command grammar."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/shv.go",
        "tools/qxctl/cmd/qxctl/shv_source.go",
        "tools/qxctl/cmd/qxctl/shv_source_test.go",
        "tools/qxctl/cmd/qxctl/shv_test.go",
        "tools/qxctl/internal/knowledgeengine/shv.go",
        "tools/qxctl/internal/knowledgeengine/shv_discovery_schema.go",
        "tools/qxctl/internal/knowledgeengine/shv_lifecycle.go",
        "tools/qxctl/internal/knowledgeengine/shv_lifecycle_descriptor.go",
        "tools/qxctl/internal/knowledgeengine/shv_lifecycle_schema.go",
        "tools/qxctl/internal/knowledgeengine/shv_lifecycle_test.go",
        "tools/qxctl/internal/knowledgeengine/shv_lifecycle_validation.go",
        "tools/qxctl/internal/knowledgeengine/shv_schema.go",
        "tools/qxctl/internal/knowledgeengine/shv_source.go",
        "tools/qxctl/internal/knowledgeengine/shv_document.go",
        "tools/qxctl/internal/knowledgeengine/shv_test.go",
        "tools/qxctl/internal/knowledgeengine/shv_validation.go",
        "tools/qxctl/internal/knowledgeengine/shv_tables.go",
        "tools/qxctl/cmd/qxctl/shv_source_activation.go",
        "tools/qxctl/cmd/qxctl/shv_activation_discovery.go",
        "tools/qxctl/internal/shvstate/store.go",
        "tools/qxctl/internal/shvstate/validation.go",
        "tools/qxctl/internal/shvstate/authorization.go",
        "tools/qxctl/internal/shvstate/storage_unix.go",
        "tools/qxctl/internal/shvstate/storage_unsupported.go",
        "tools/qxctl/cmd/qxctl/shv_pdf.go",
        "tools/qxctl/cmd/qxctl/shv_pdf_graph.go",
        "tools/qxctl/internal/knowledgeengine/shv_pdf_graph.go",
        "tools/qxctl/cmd/qxctl/shv_pdf_test.go",
        "tools/qxctl/internal/knowledgeengine/shv_pdf.go",
        "tools/qxctl/internal/knowledgeengine/shv_pdf_descriptor.go",
        "tools/qxctl/internal/knowledgeengine/shv_pdf_validation.go",
        "tools/qxctl/internal/knowledgeengine/shv_pdf_test.go",
        "tools/qxctl/cmd/qxctl/shv_dossier.go",
        "tools/qxctl/cmd/qxctl/shv_dossier_test.go",
        "tools/qxctl/cmd/qxctl/shv_dossier.schema.json",
        "tools/qxctl/cmd/qxctl/shv_inventory.go",
        "tools/qxctl/cmd/qxctl/shv_inventory_test.go",
        "tools/qxctl/cmd/qxctl/shv_inventory.schema.json",
        "tools/qxctl/cmd/qxctl/shv_refresh.go",
        "tools/qxctl/cmd/qxctl/shv_refresh_test.go",
        "tools/qxctl/cmd/qxctl/shv_refresh.schema.json",
        "tools/qxctl/scripts/build_shv_refresh_schema.py",
        "tools/qxctl/cmd/qxctl/shv_refresh_compare.go",
        "tools/qxctl/cmd/qxctl/shv_refresh_compare_test.go",
        "tools/qxctl/cmd/qxctl/shv_partition.go",
        "tools/qxctl/internal/knowledgeengine/shv_partition.go",
        "tools/qxctl/internal/knowledgeengine/shv_partition_descriptor.go",
        "tools/qxctl/internal/knowledgeengine/shv_partition_validation.go",
        "tools/qxctl/internal/knowledgeengine/shv_partition_test.go",
        "tools/qxctl/cmd/qxctl/shv_materialization.go",
        "tools/qxctl/cmd/qxctl/shv_materialization_discovery.go",
        "tools/qxctl/cmd/qxctl/shv_materialization.schema.json",
        "tools/qxctl/cmd/qxctl/shv_materialization_test.go",
        "tools/qxctl/internal/shvjob/store.go",
        "tools/qxctl/internal/shvjob/storage_unix.go",
        "tools/qxctl/internal/shvjob/storage_unsupported.go",
        "tools/qxctl/internal/shvjob/store_test.go",
        "tools/qxctl/cmd/qxctl/shv_resolution.go",
        "tools/qxctl/cmd/qxctl/shv_resolution.schema.json",
        "tools/qxctl/cmd/qxctl/shv_resolution_test.go",
        "tools/qxctl/cmd/qxctl/shv_relocation.go",
        "tools/qxctl/cmd/qxctl/shv_relocation.schema.json",
        "tools/qxctl/cmd/qxctl/shv_relocation_test.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not select hardware, requirements, providers, topology or a graph database vendor.",
        "Does not persist a graph, authenticate publisher claims, fetch sources or infer whole-system compatibility from processor fields."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "C++ owns SHV semantic rebuild and findings.",
          "target_feature_id": "ssfv:symphony:shv-engine",
          "type": "composes_with"
        },
        {
          "rationale": "Independent generic C++ adapter owns structural exchange and queries.",
          "target_feature_id": "ssfv:symphony:shv-graph-adapter",
          "type": "composes_with"
        },
        {
          "rationale": "C++ owns source revision and capture provenance; qxctl independently verifies exact correspondence.",
          "target_feature_id": "ssfv:symphony:shv-source-engine",
          "type": "depends_on"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "SHV source-backed hardware and generic graph administration",
      "what": "Administers exact SHV kernel, source lifecycle and generic graph operations, including separately authorized local source activation and recovery.",
      "when": "Only upon an explicit read-only qxctl shv invocation.",
      "where": "Local administrative process against explicit exact receipt-owned C++ installations and retained source roots.",
      "who": "Users, agents and automation selecting exact independently installed components.",
      "why": "Keeps the agentic operating interface complete while preserving user-selected hardware, coverage, source mappings and requirements."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG client and grammar changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes SSIAG contracts, clients, commands, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns canonical identity, access-governance, credential, and authorization contracts.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG policy decisions are audit-before-release operations through the STAV append authority.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "SSIAG administration manages policy and decision interfaces; authenticated-session administration consumes decisions to mutate a separate coordinator journal.",
          "target_feature_id": "ssfv:symphony:qxctl.authenticated-sessions"
        }
      ],
      "evidence": [
        "tools/qxctl/cmd/qxctl/foundation_lifecycle_test.go verifies the complete enrollment and supervisor families, rejects unratified shortcuts, and closes malformed intent before adapter invocation.",
        "tools/qxctl/cmd/qxctl/ssiag_test.go verifies provider-trust, provider-binding, decision, and policy-administration grammar, closed mutation paths, and bounded inputs.",
        "tools/qxctl/internal/foundationlifecycle/client_test.go verifies exact receipt-v2 selection, adapter identity, canonical digests, operation-specific result shape, audit disposition, and recovery evidence.",
        "tools/qxctl/internal/ssiagclient/client_test.go verifies trust, peer authentication, safe evidence, and caller-neutral result handling.",
        "tools/qxctl/internal/ssiagclient/provider_binding_test.go verifies all six closed provider lifecycle routes, exact request bodies, opaque selection, digest-bound results, and secret/operational rejection."
      ],
      "feature_id": "ssfv:symphony:qxctl.ssiag-administration",
      "how": "The cgo-free clients validate protected trust configuration and Unix-socket metadata, verify kernel peer identity before HTTP exchange, read closed digest-bound provider trust evidence, ask SSIAG for fresh permission-backed provider verification without inspecting or launching adapters, consume SSIAG-owned opaque provider-installation inventory, administer exact-state provider binding plan/apply/status/recovery, select one exact receipt-owned foundational lifecycle adapter without version recency inference, invoke it through bounded process IPC, and independently validate proposal, apply, recovery, audit, and safe decision evidence.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements local endpoint trust, kernel peer verification, opaque provider-binding administration, exact foundational lifecycle-adapter selection, bounded decision and process clients, protected policy administration, and CLI grammar."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/foundation_lifecycle.go",
        "tools/qxctl/cmd/qxctl/foundation_lifecycle_test.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/cmd/qxctl/ssiag_provider_binding.go",
        "tools/qxctl/cmd/qxctl/ssiag_test.go",
        "tools/qxctl/internal/foundationlifecycle/client.go",
        "tools/qxctl/internal/foundationlifecycle/client_test.go",
        "tools/qxctl/internal/ssiagclient/client.go",
        "tools/qxctl/internal/ssiagclient/client_test.go",
        "tools/qxctl/internal/ssiagclient/peerauth_darwin.go",
        "tools/qxctl/internal/ssiagclient/peerauth_linux.go",
        "tools/qxctl/internal/ssiagclient/provider_binding.go",
        "tools/qxctl/internal/ssiagclient/provider_binding_test.go",
        "tools/qxctl/internal/ssiagclient/trust_unix.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not own SSIAG schemas, persist credentials, implement keyring providers, open provider receipts, resolve opaque installation identities, launch provider adapters, edit STAV, or decide permission.",
        "Does not infer authority from human or AI caller class and does not create remote SSIAG access.",
        "Does not implement enrollment or supervision, render native descriptors, select a newest package version, expose purge, or bypass module recovery and audit state."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl is the administrative client; SSIAG owns authentication, policy evaluation, credential boundaries, and mutation state.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "SSIAG provider trust, binding, policy, decision, enrollment, and supervision administration",
      "what": "Queries SSIAG provider metadata and trust evidence, requests fresh SSIAG-owned trust verification, inventories opaque provider installations, administers SSIAG-owned exact binding plan/apply/status/recovery, queries decisions, administers explicit grants and protected policy mutation, and drives exact installed enrollment and native-supervision lifecycle adapters without embedding module or provider logic in qxctl.",
      "when": "Runs only on explicit SSIAG commands or when another protected qxctl operation requests a fresh exact decision.",
      "where": "Operates against the configured local SSIAG foundation endpoint for one TOPS and scope.",
      "who": "Any caller holding the applicable target-host permission, including a host owner, delegated administrator, automation, or agentic owner.",
      "why": "Provides one caller-neutral administrative surface for ownership and granted-permission policy while leaving authentication, policy truth, credentials, and provider execution with SSIAG."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed STAV client and grammar changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes STAV contracts, clients, commands, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG produces the security-relevant decision events whose safe metadata is committed through STAV.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns canonical envelope, projection, ledger, reader, and append-authority truth.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "STAV administration reads and verifies audit evidence; SSIAG administration evaluates and manages caller-neutral identity and access-governance policy.",
          "target_feature_id": "ssfv:symphony:qxctl.ssiag-administration"
        }
      ],
      "evidence": [
        "tools/qxctl/cmd/qxctl/foundation_lifecycle_test.go verifies the complete enrollment and supervisor families, rejects unratified shortcuts, and closes malformed intent before adapter invocation.",
        "tools/qxctl/cmd/qxctl/stav_accordare.go implements exact producer status and durable reconciliation over the enrolled per-TOPS Unix socket.",
        "tools/qxctl/cmd/qxctl/stav_grant_test.go verifies exact stopped-authority grant mutation, SSIAG authorization, configuration digest compare-and-swap, and recovery behavior.",
        "tools/qxctl/cmd/qxctl/stav_test.go verifies STAV command grammar, prohibited raw append, and validated response behavior.",
        "tools/qxctl/internal/foundationlifecycle/client_test.go verifies exact receipt-v2 selection, adapter identity, canonical digests, operation-specific result shape, audit disposition, and recovery evidence.",
        "tools/qxctl/internal/stavclient/paths_test.go verifies bounded per-TOPS state and socket path resolution."
      ],
      "feature_id": "ssfv:symphony:qxctl.stav-administration",
      "how": "qxctl resolves protected per-TOPS paths, verifies configured STAV and Accordare producer endpoints and peers, sends canonical local requests, performs SSIAG-authorized exact-grant compare-and-swap while STAV is stopped, selects one exact receipt-owned lifecycle adapter without version recency inference, invokes it through bounded process IPC, and validates reader classifications, projections, lifecycle transitions, audit disposition, reconciliation evidence, digests, and bounded results.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements mutually authenticated local STAV query and verification, exact lifecycle-adapter selection, bounded result validation, path resolution, and command grammar."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/foundation_lifecycle.go",
        "tools/qxctl/cmd/qxctl/foundation_lifecycle_test.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/cmd/qxctl/stav_accordare.go",
        "tools/qxctl/cmd/qxctl/stav_grant.go",
        "tools/qxctl/cmd/qxctl/stav_grant_test.go",
        "tools/qxctl/cmd/qxctl/stav_test.go",
        "tools/qxctl/internal/foundationlifecycle/client.go",
        "tools/qxctl/internal/foundationlifecycle/client_test.go",
        "tools/qxctl/internal/stavclient/client.go",
        "tools/qxctl/internal/stavclient/paths.go",
        "tools/qxctl/internal/stavclient/paths_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not expose raw qxctl append, edit ledger files, own the STAV schema, or bypass producer authorization.",
        "Does not convert query access or audit availability into SSIAG permission or canonical mutation authority.",
        "Does not render or own module internals, select a newest package version, expose purge, or bypass exact enrollment, module recovery, SSIAG authorization, and audit state."
      ],
      "owner_contract": "tools/qxctl/MANIFEST.md",
      "parent_feature_id": "ssfv:symphony:qxctl",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl administers safe query and verification operations while the append authority alone serializes validated event production.",
          "target_feature_id": "ssfv:symphony:stav-append-authority",
          "type": "composes_with"
        }
      ],
      "source_scope": "tools/qxctl",
      "status": "experimental",
      "title": "STAV audit, Accordare producer, grant, enrollment, and supervision administration",
      "what": "Provides bounded STAV status, verification, and query administration; Accordare producer status and durable reconciliation; SSIAG-authorized exact producer-grant administration; and exact installed enrollment and native-supervision lifecycle-adapter administration without exposing raw append or module internals.",
      "when": "Runs only on explicit STAV administrative or query commands and when other security flows need validated audit availability evidence.",
      "where": "Executes locally on the administrative TOPS node against the separately installed STAV append authority and per-installation ledger state.",
      "who": "Authorized target-host administrators, maintainers, auditors, automation, and agentic tools using qxctl.",
      "why": "Makes tamper-evident operational audit evidence inspectable without giving a general-purpose CLI permission to forge or edit the append-only record."
    }
  ],
  "source_scope": "tools/qxctl"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->

SCV index transfer adds exact-plan execution, journal-only inspection and recoverable replay under `knowledge/scv/INDEX-TRANSFER.md`. Native C++ prepare/commit owners remain unchanged; qxctl retains only durable copy progress and independently checks target evidence.

`scv graph-index schema` lists or returns exact native receipt-owned and separately embedded qxctl orchestration definitions. Discovery labels ownership and file digests and never substitutes a newer connector profile.
