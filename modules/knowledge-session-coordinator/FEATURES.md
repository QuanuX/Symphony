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
          "applicability": "applicable",
          "reason": "The coordinator derives and serializes exact docking actions while the separately installed Maestro presence authority owns their authenticated operational state.",
          "reference": "modules/maestro/SPEC.md",
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
          "applicability": "applicable",
          "reason": "Every authenticated-session and qxctl lifecycle administration operation consumes a fresh exact caller-neutral SSIAG decision derived from kernel peer identity. Stable TOPS/profile policy resources avoid artifact-driven grant churn, while qxctl and the coordinator independently bind exact profile, receipt, observation, journal, applied-state, and registry evidence.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG releases no authorization decision used by the session slice until its safe policy-decision event is committed by the STAV append authority; the coordinator itself is not a STAV producer.",
          "reference": "knowledge/stav/SPEC.md",
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
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp verifies reconciliation and authenticated-session lifecycle mutations, exact-state rejection, idempotent replay, context attachment, linked epochs, decision/capability binding, expiry, capability downgrade, damaged-head recovery, extension preservation, critical-state refusal, lock contention, symlink rejection, and isolation.",
        "modules/knowledge-session-coordinator/tests/lifecycle_test.cpp verifies normalized deterministic plans, receipt-v1/v2 capability negotiation, forward and inverse exact-version selection, dependency-ready-set healing, isolated staged-install blocker resolution without dependency bypass, exact report/apply journal binding, prepared-attempt durability, direct outcome verification, prior-applied evidence across transactions, idempotent replay, applied-state closure, stale compare-and-swap rejection, active-action recovery, critical/noncritical dependency behavior, component-capability blocking, cycle isolation, compatibility and integrity failure, exact receptor replacement, safe undock/deactivate/select/activate/dock ordering, and timestamp-neutral transaction identity.",
        "modules/knowledge-session-coordinator/tests/ssfv_maintenance_test.cpp verifies persistent SSFV baseline capture, idempotent replay, exact compare-and-swap rejection, separate baseline/current engine lineage across upgrade order, read-only status, corrupted-head detection, and explicit digest-linked forward recovery.",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh verifies deterministic process responses, implemented session capability truth, bounded check output, duplicate-key rejection, and disabled canonical apply.",
        "modules/knowledge-session-coordinator/CMakeLists.txt builds, tests, installs, receipts, and uninstalls the exact versioned process.",
        "modules/knowledge-session-coordinator/SPEC.md defines inspect/check plus separate durable reconciliation and authenticated-session journals, two-way procedural compatibility, evidence-preserving recovery, and the canonical-apply boundary.",
        "tools/qxctl/cmd/qxctl/session_test.go verifies explicit login, refresh, logout, retry, bounded recovery, reauthentication, and interrupted-close resumption over the coordinator primitives.",
        "tools/qxctl/internal/knowledgelifecycle/profile_test.go verifies protected desired-profile compare-and-swap, semantic retry, linked generations, fixed-layout v1/v2 receipt observation, content drift, unknown-package preservation, and stable inventory identity.",
        "tools/qxctl/cmd/qxctl/lifecycle_test.go verifies report-plan safety, dynamic scheduler invariants, staged-install exception isolation, strict v1/v2 lifecycle-journal IPC validation, exact apply result validation, critical-extension refusal, and lifecycle Cobra grammar.",
        "tools/qxctl/cmd/qxctl/ssfv_maintenance_test.go verifies the cross-vector Cobra grammar, fully bound authorization resource, recursively canonical evidence digest, canonical-apply refusal, and journal-digest enforcement.",
        "tools/qxctl/internal/knowledgelifecycle/executor_test.go verifies protected runtime compare-and-swap/idempotency, prepared observation-drift refusal, staged receipt-v2 install/uninstall and interruption replay, separate rollback proof, conflicting administrator-file protection, and exact Maestro docking adapter evidence.",
        "tools/qxctl/cmd/qxctl/lifecycle_binding_test.go verifies exact receipt-v2 established-role selection, forward upgrade, inverse rollback, and binding-registry predecessor evidence; tools/qxctl/internal/knowledgeengine/client_test.go verifies receipt-v1/v2 dual-read behavior, compatible forward growth of the v2 owned-file set, stable role semantics, and receipt-v2 content/digest tamper rejection.",
        "modules/maestro/tests/maestro_test.cpp and tools/qxctl/internal/maestroclient/client_test.go verify exact receptor presence, authorization binding, receipt identity, mutation retry, and forward recovery without engine execution.",
        "tools/qxctl/cmd/qxctl/main.go, tools/qxctl/cmd/qxctl/lifecycle.go, tools/qxctl/cmd/qxctl/lifecycle_apply.go, tools/qxctl/internal/knowledgelifecycle/executor.go, tools/qxctl/internal/ssiagclient/client.go, and tools/qxctl/internal/knowledgeengine/client.go authenticate SSIAG, validate safe authorization evidence, invoke the exact bound coordinator, collect lifecycle observations, serialize external actions, and re-observe before finalization."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "how": "The C++26 symphony-knowledge-session process statically links the shared foundation, accepts the common bounded process envelope, and keeps reconciliation contexts, authenticated authority epochs, SSFV maintenance contexts, report lifecycle streams, and apply lifecycle streams separate. Each durable stream uses a private no-follow lock, fsync-backed dual slots, an atomic head, content digests, stable operation IDs, exact compare-and-swap state, opaque-extension preservation, and evidence-based forward recovery. SSFV maintenance preserves the initial semantic snapshot and baseline engine separately from current engine evidence, so compatible upgrades and rollbacks cannot reinterpret the baseline; qxctl supplies exact read-only SSFV diff evidence and explicit complete or not-configured Maestro inventory evidence under fresh SSIAG authorization. The lifecycle planner validates complete caller-supplied desired and observed evidence and derives stable forward/inverse actions from a dependency-ready set. Report boot persists non-executable source evidence in v1. Separately authorized v2 prepare records one active attempt before external action; finalize accepts complete re-observation only when it proves the target; close selects immutable content-addressed applied evidence. Go qxctl revalidates legacy and immutable receipt formats, obtains fresh SSIAG decisions per phase, maintains desired and generic runtime state, scans fixed receipt layouts, executes only reviewed adapters, and re-observes before finalization. Established roles select through binding-registry compare-and-swap. Coordinator changes install side by side, require the candidate to reproduce the exact prepared journal, retain the invoking coordinator for uninterrupted finalization, and recover after a switch through the same durable active action. The planner isolates cycles and blockers, reports noncritical advisories, binds exact receptors, and safely sequences package and receptor changes without the coordinator discovering packages, writing Maestro state, deciding feature-worthiness, or performing host mutation.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the independently installed process, descriptor, bounded snapshot check, compatibility negotiation, separate reconciliation/session/report/apply journal durability, SSIAG evidence validation, dependency-driven lifecycle planning, prepared attempts, verified applied-state commitment, idempotent lifecycles, and evidence-preserving recovery."
        },
        {
          "language": "CMake",
          "role": "Builds and installs the exact versioned executable, module contracts, immutable receipt-v2 generated last from installed content, tests, and receipt-owned uninstall surface."
        },
        {
          "language": "Go",
          "role": "Implements qxctl receipt-v1/v2 binding revalidation, stable-resource kernel-authenticated SSIAG authorization requests, proposal-only lifecycle grant generation, decision-boundary validation, exact coordinator invocation, candidate-verified coordinator handoff, reconciliation/session/lifecycle command grammar, protected lifecycle desired/runtime state, fixed-layout receipt observation, staged receipt-v2 install/uninstall, established binding and generic selection/activation adapters, explicit login/refresh/logout transition composition, operation identity, expected-state input, and bound-engine inventory delivery."
        }
      ],
      "implementation_paths": [
        "cmake/SymphonyInstallReceiptV2.cmake",
        "cmake/SymphonyInstallReceiptV2.cmake.in",
        "modules/knowledge-session-coordinator/CMakeLists.txt",
        "modules/knowledge-session-coordinator/src/authority_session.cpp",
        "modules/knowledge-session-coordinator/src/authority_session.hpp",
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/src/coordinator.hpp",
        "modules/knowledge-session-coordinator/src/lifecycle.cpp",
        "modules/knowledge-session-coordinator/src/lifecycle.hpp",
        "modules/knowledge-session-coordinator/src/lifecycle_journal.cpp",
        "modules/knowledge-session-coordinator/src/lifecycle_journal.hpp",
        "modules/knowledge-session-coordinator/src/main.cpp",
        "modules/knowledge-session-coordinator/src/reconciliation.cpp",
        "modules/knowledge-session-coordinator/src/reconciliation.hpp",
        "modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp",
        "modules/knowledge-session-coordinator/src/ssfv_maintenance.hpp",
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp",
        "modules/knowledge-session-coordinator/tests/lifecycle_test.cpp",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh",
        "modules/knowledge-session-coordinator/tests/ssfv_maintenance_test.cpp",
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/lifecycle.go",
        "tools/qxctl/cmd/qxctl/lifecycle_apply.go",
        "tools/qxctl/cmd/qxctl/lifecycle_binding_test.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/cmd/qxctl/ssfv_maintenance.go",
        "tools/qxctl/internal/knowledgeengine/client.go",
        "tools/qxctl/internal/knowledgelifecycle/executor.go",
        "tools/qxctl/internal/knowledgelifecycle/install_unix.go",
        "tools/qxctl/internal/knowledgelifecycle/observation.go",
        "tools/qxctl/internal/knowledgelifecycle/profile.go",
        "tools/qxctl/internal/knowledgelifecycle/runtime.go",
        "tools/qxctl/internal/knowledgelifecycle/scan_unix.go",
        "tools/qxctl/internal/knowledgelifecycle/state_unix.go",
        "tools/qxctl/internal/maestroclient/client.go",
        "tools/qxctl/internal/ssiagclient/client.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not independently authenticate the operating-system caller or decide permission; SSIAG owns those decisions, and the coordinator validates only the supplied exact safe evidence.",
        "Does not implement canonical apply; it mutates only protected noncanonical reconciliation, authenticated-session, lifecycle-journal, runtime-state, receipt-v2 package, and applied-state surfaces within the explicit lifecycle contract.",
        "Does not call SSIAG or STAV directly, consume credentials or provider secrets, write Maestro presence, or mutate canonical repository state.",
        "The coordinator does not collect lifecycle observations, administer desired/runtime profiles, or execute host actions; qxctl performs those functions while the coordinator serializes attempts and verifies applied evidence.",
        "Does not mutate receipt-v1 packages, download packages, execute arbitrary receipt entry points, activate live services/processes, replace itself in place, rewrite engine bindings directly, or execute a docked engine. qxctl may externally select an exact receipt-v2 candidate only after that candidate reproduces the prepared journal.",
        "Does not claim system or TOPS provisioning, active docking, or a published module release."
      ],
      "owner_contract": "modules/knowledge-session-coordinator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The coordinator derives, serializes, and verifies docking actions while Maestro alone persists the authenticated receptor presence outcome.",
          "target_feature_id": "ssfv:symphony:maestro-presence-authority",
          "type": "composes_with"
        },
        {
          "rationale": "The coordinator statically links and relies on the shared process, digest, path, and snapshot mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/knowledge-session-coordinator",
      "status": "experimental",
      "title": "Durable knowledge session, reconciliation, and lifecycle coordinator",
      "what": "Provides an independently installable receipt-v2 administrative process that reports exact capabilities, computes deterministic snapshots, maintains recoverable noncanonical reconciliation, authority-epoch, and persistent SSFV semantic-baseline state, converts complete desired/observed lifecycle evidence into a deterministic dependency plan, preserves report-only boot evidence, and separately coordinates prepared actions, established-role version handoff, verified completion, applied-state commitment, and recovery per TOPS and profile.",
      "when": "Runs only on explicit user-scope qxctl or exact process invocation under a bounded deadline. Reconciliation begins with an explicit inventory. Session and lifecycle operations each require fresh exact SSIAG evidence. Report/boot re-read protected profiles and re-observe configured roots. Explicit apply is eligible only for apply-compatible profiles with exact source/journal/applied compare-and-swap; every external action is prepared durably and followed by complete re-observation. Status is read-only and recovery is explicit. No operation acquires canonical apply authority.",
      "where": "Executes as the inactive, installed-undocked symphony-knowledge-session C++ process in the administrative freezing path and roots checks at its current repository working directory.",
      "who": "Any host-authorized caller using qxctl, plus maintainers and tests that need caller-neutral, domain-neutral authority-epoch and reconciliation boundaries across independently versioned SKV engines.",
      "why": "Preserves coherent authority-epoch, worktree, and engine-version evidence when upgrades, retries, logouts, or crashes occur out of sequence, and allows compatible lifecycle work to reorder around localized blockers while separating SSIAG authorization, session state, reconciliation state, lifecycle reports, vector semantics, and apply authority."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed authenticated-session contracts and implementation.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes authority-session schemas, implementation, tests, and qxctl grammar.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns exact host identity and granted-permission decisions.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "The authorization decision is released only after its safe event is durably appended through STAV.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "An authority epoch records SSIAG-bound permission and expiry; reconciliation records review context without authentication semantics.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.reconciliation"
        }
      ],
      "evidence": [
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp verifies linked epochs, decision/capability binding, expiry, downgrade refusal, idempotency, and bounded recovery.",
        "tools/qxctl/cmd/qxctl/session_test.go verifies explicit login, refresh, logout, reauthentication, and interrupted-close recovery composition."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator.authority-epochs",
      "how": "The coordinator validates exact decision and capability evidence, TOPS/subject/resource binding, expiry, policy and configuration digests, expected journal state, operation fingerprints, linked generations, and unique forward recovery.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements exact SSIAG evidence validation, linked authority epochs, expiry, checkpointing, closure, compatibility, and forward recovery."
        },
        {
          "language": "Go",
          "role": "Supplies fresh authenticated decisions and explicit session transition composition through qxctl."
        }
      ],
      "implementation_paths": [
        "modules/knowledge-session-coordinator/src/authority_session.cpp",
        "modules/knowledge-session-coordinator/src/authority_session.hpp",
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp",
        "tools/qxctl/cmd/qxctl/session_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not authenticate the operating-system caller, call SSIAG or STAV directly, or convert evidence into a bearer token.",
        "Does not grant canonical apply, mutate policy, or make caller class an authorization input."
      ],
      "owner_contract": "modules/knowledge-session-coordinator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl authenticates the endpoint and obtains decisions while the coordinator owns epoch durability and recovery.",
          "target_feature_id": "ssfv:symphony:qxctl.authenticated-sessions",
          "type": "composes_with"
        },
        {
          "rationale": "SSIAG owns caller authentication and permission decisions; the coordinator validates only supplied exact safe evidence.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/knowledge-session-coordinator",
      "status": "experimental",
      "title": "Durable authenticated authority epochs",
      "what": "Persists one caller-neutral authenticated knowledge authority epoch from explicit login/begin through refresh/checkpoint and logout/close or expiry.",
      "when": "Begins on explicit authenticated session creation, advances on explicit checkpoint or refresh, and closes on logout, expiry, policy change, configuration change, or explicit recovery outcome.",
      "where": "Stores protected noncanonical session journals keyed by exact scope, TOPS, subject, and repository identity beneath the coordinator state root.",
      "who": "Any caller whose target-host identity and granted permission were authenticated by SSIAG and transported through qxctl.",
      "why": "Makes authority continuity explicit, expiring, non-transferable, replay-safe, and recoverable without privileging a human or AI caller class."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Dock and undock adapters commit exact authenticated Maestro receptor presence outside the coordinator.",
          "reference": "modules/maestro/SPEC.md",
          "vector": "maestro"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed apply journal, adapter, and recovery changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes apply schemas, coordinator source, qxctl adapters, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Every status, prepare, finalize, close, and recovery invocation carries a fresh exact decision.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG releases lifecycle authorization evidence only after safe audit commitment through STAV.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Apply coordination durably records authorized attempts and verified outcomes; planning alone is stateless, report-only, and never mutation authority.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.lifecycle-planning"
        }
      ],
      "evidence": [
        "modules/knowledge-session-coordinator/tests/lifecycle_test.cpp verifies prepare-before-mutation, exact finalize proof, dynamic replanning, interruption recovery, applied-state closure, upgrade-order compatibility, and failed-attempt preservation.",
        "tools/qxctl/internal/knowledgelifecycle/executor_test.go verifies reviewed adapters, prepared observation-drift refusal, staged install/uninstall, rollback proof, and exact Maestro evidence."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator.lifecycle-apply-coordination",
      "how": "Prepare records an active attempt before mutation; qxctl executes a closed adapter vocabulary; finalize binds the same operation, journal, profile, artifacts, and observations; the coordinator replans dynamically, preserves failures, and commits content-addressed applied state only at verified convergence.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Serializes prepared attempts, validates exact authorization and evidence, replans from verified outcomes, and commits immutable applied-state selection."
        },
        {
          "language": "Go",
          "role": "Obtains fresh authorization, executes the bounded reviewed external adapters, re-observes host state, and submits exact finalize evidence."
        }
      ],
      "implementation_paths": [
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/src/lifecycle.cpp",
        "modules/knowledge-session-coordinator/src/lifecycle.hpp",
        "modules/knowledge-session-coordinator/src/lifecycle_journal.cpp",
        "modules/knowledge-session-coordinator/src/lifecycle_journal.hpp",
        "modules/knowledge-session-coordinator/tests/lifecycle_test.cpp",
        "tools/qxctl/cmd/qxctl/lifecycle_apply.go",
        "tools/qxctl/internal/knowledgelifecycle/executor.go",
        "tools/qxctl/internal/knowledgelifecycle/executor_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not perform host or Maestro mutation inside the coordinator, replace itself in place, or write qxctl binding/runtime state.",
        "Does not download, mutate receipt v1, execute arbitrary entry points, activate live services, run docked engines, or mutate canonical knowledge."
      ],
      "owner_contract": "modules/knowledge-session-coordinator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Each prepare and finalize recomputes the deterministic plan from complete current evidence.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.lifecycle-planning",
          "type": "composes_with"
        },
        {
          "rationale": "qxctl owns authorization exchange and external adapters while the coordinator owns serialization, verification, and recovery.",
          "target_feature_id": "ssfv:symphony:qxctl.lifecycle-convergence",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/knowledge-session-coordinator",
      "status": "experimental",
      "title": "Durable lifecycle apply coordination",
      "what": "Coordinates one durable, separately authorized lifecycle action at a time and advances applied evidence only after complete re-observation proves the target.",
      "when": "Runs only on explicit apply, apply-status, or apply-recover commands after a compatible report journal exists and fresh authorization is available for every phase.",
      "where": "Stores a separate protected v2 apply stream and immutable applied evidence beside, never over, the report-only v1 lifecycle journal.",
      "who": "A target-host-authorized lifecycle administrator, automation, or agentic owner invoking qxctl with exact expected state and trusted staged evidence.",
      "why": "Makes host mutation interruption-safe, retryable, reversible where contracted, and tolerant of unplanned component upgrade order without fabricating completion or discarding failed evidence."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Desired and observed docking state uses explicit Maestro receptor identities while the planner never contacts Maestro.",
          "reference": "modules/maestro/SPEC.md",
          "vector": "maestro"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed lifecycle planning and compatibility changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes lifecycle schemas, planner source, qxctl integration, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "Planning emits disposable report-only evidence with apply_authorized false; apply coordination separately records and verifies one authorized external action at a time.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.lifecycle-apply-coordination"
        }
      ],
      "evidence": [
        "modules/knowledge-session-coordinator/tests/lifecycle_test.cpp verifies deterministic plans, forward/inverse selection, dependency-ready healing, blocker and cycle isolation, receptor replacement, and timestamp-neutral identity.",
        "tools/qxctl/cmd/qxctl/lifecycle_test.go verifies complete observation transport, result validation, compatibility blocking, and report-only command behavior."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator.lifecycle-planning",
      "how": "The coordinator validates digest-bound complete evidence, negotiates explicit versions and capabilities, builds a stable dependency-ready graph, isolates cycles and localized blockers, and recomputes order from each changed verified observation.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Validates complete desired and observed evidence and derives deterministic forward/inverse dependency-ready plans and localized blockers."
        },
        {
          "language": "Go",
          "role": "Collects fixed-layout observations, invokes the exact planner, and validates bounded noncanonical results through qxctl."
        }
      ],
      "implementation_paths": [
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/src/lifecycle.cpp",
        "modules/knowledge-session-coordinator/src/lifecycle.hpp",
        "modules/knowledge-session-coordinator/tests/lifecycle_test.cpp",
        "tools/qxctl/cmd/qxctl/lifecycle_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not acquire mutation locks, request authorization, install or remove packages, activate processes, contact receptors, or write state.",
        "Does not infer newest versions, ignore blockers, participate in hot or warm paths, or grant apply authority."
      ],
      "owner_contract": "modules/knowledge-session-coordinator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl supplies complete evidence and owns external administration while the coordinator derives deterministic non-mutating plans.",
          "target_feature_id": "ssfv:symphony:qxctl.lifecycle-convergence",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/knowledge-session-coordinator",
      "status": "experimental",
      "title": "Deterministic two-way lifecycle planning",
      "what": "Produces a deterministic report-only plan for installing, uninstalling, selecting, activating, docking, undocking, preserving, or verifying independently versioned components.",
      "when": "Runs on explicit lifecycle report or boot planning and before each separately authorized apply prepare/finalize step.",
      "where": "Executes as a freezing-path local process and reads no host files, receipts, profiles, registries, or Maestro state directly.",
      "who": "Target-host-authorized administrators, automation, or agentic owners using qxctl to compare explicit desired and observed module state.",
      "why": "Makes upgrade and rollback equally ordinary convergence and lets an unplanned sequence self-heal by changing safe ready order without bypassing fixed thermal or dependency restrictions."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed Named Version durability and qxctl administration changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the common persistence schemas, coordinator implementation, qxctl grammar, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV owns any later publication linkage while Named Version persistence leaves that reference null until publication exists.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "Every proposal, seal, alias, lookup, status, and recovery request carries a fresh exact caller-neutral decision.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "Safe Accordare audit vocabulary remains separately gated; results explicitly keep direct STAV append disabled until that vocabulary is ratified.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "SAV validates composition semantics but remains stateless; this coordinator feature preserves only protected immutable bytes, proposals, and noncanonical selector state.",
          "target_feature_id": "ssfv:symphony:sav-engine.named-version"
        }
      ],
      "evidence": [
        "knowledge/schemas/v1/named-version-command.schema.json, named-version-proposal.schema.json, named-version-registry.schema.json, named-version-head.schema.json, and named-version-result.schema.json close the complete shared persistence boundary.",
        "modules/knowledge-session-coordinator/tests/named_versions_test.cpp verifies protected preparation, immutable seal, post-write exact-byte lookup, alias selection, idempotent replay, stale compare-and-swap refusal, damaged-head recovery, and symlink rejection.",
        "tools/qxctl/cmd/qxctl/named_version_test.go verifies strict result identity, resource binding, digest validation, and the six headless command routes."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator.named-version-durability",
      "how": "qxctl invokes the exact protected SAV binding before preparation, obtains a fresh SSIAG decision for each digest-bound operation, and invokes the exact coordinator binding. The C++ coordinator stores idempotent prepared proposals and immutable digest-named objects, serializes registry and alias changes through a no-follow lock, fsync-backed alternating slots, and an atomic head, rejects stale state and operation reuse, and recovers only one unique linked registry. qxctl re-reads and revalidates every returned artifact through the exact SAV binding.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements protected proposal and object persistence, exact-state registry mutation, alias selection, lookup, compatibility, and forward recovery."
        },
        {
          "language": "Go",
          "role": "Implements headless qxctl orchestration, exact SAV validation and revalidation, SSIAG authorization, coordinator invocation, and strict result validation."
        }
      ],
      "implementation_paths": [
        "knowledge/schemas/v1/named-version-command.schema.json",
        "knowledge/schemas/v1/named-version-head.schema.json",
        "knowledge/schemas/v1/named-version-proposal.schema.json",
        "knowledge/schemas/v1/named-version-registry.schema.json",
        "knowledge/schemas/v1/named-version-result.schema.json",
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/src/named_versions.cpp",
        "modules/knowledge-session-coordinator/src/named_versions.hpp",
        "modules/knowledge-session-coordinator/tests/named_versions_test.cpp",
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/named_version.go",
        "tools/qxctl/cmd/qxctl/named_version_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not decide composition truth, grant permission, publish or install a package, activate or dock a composition, or mutate canonical repository knowledge.",
        "Does not infer newest versions, use aliases as identity, rewrite sealed objects, persist a canonical database, or append STAV events before vocabulary ratification."
      ],
      "owner_contract": "modules/knowledge-session-coordinator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl owns exact binding resolution, authorization exchange, orchestration, and post-read revalidation.",
          "target_feature_id": "ssfv:symphony:qxctl",
          "type": "composes_with"
        },
        {
          "rationale": "SAV supplies stateless semantic validation while the coordinator owns protected noncanonical durability.",
          "target_feature_id": "ssfv:symphony:sav-engine.named-version",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/knowledge-session-coordinator",
      "status": "experimental",
      "title": "Protected immutable Named Version durability",
      "what": "Persists SAV-validated immutable Named Version objects, durable preparation evidence, and a recoverable noncanonical identity and alias registry per TOPS.",
      "when": "Runs only on explicit qxctl proposal, seal, alias, lookup, status, or recovery commands with fresh exact SSIAG authorization; status and lookup remain read-only and recovery remains explicit.",
      "where": "Stores protected freezing-path state beneath the selected coordinator state root, separated by opaque TOPS key, with immutable objects distinct from the dual-slot selector registry.",
      "who": "Any target-host-authorized owner, administrator, automation, or agentic caller operating through qxctl with effective permission.",
      "why": "Makes reusable Symphony compositions interruption-safe, content-addressed, rollback-selectable, upgrade-order tolerant, and independently verifiable without turning a mutable alias or database projection into identity."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed reconciliation durability and recovery changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the coordinator reconciliation contracts, source, tests, and install surface.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        }
      ],
      "distinctions": [
        {
          "distinction": "Reconciliation contexts preserve bounded review continuity but contain no authentication or permission evidence; authority epochs require exact SSIAG-bound state.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.authority-epochs"
        }
      ],
      "evidence": [
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp verifies exact-state rejection, idempotent replay, damaged-head recovery, extension preservation, lock contention, symlink rejection, and isolation.",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh verifies bounded deterministic process responses and duplicate-key rejection."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator.reconciliation",
      "how": "The coordinator uses a private no-follow lock, synchronized alternating slots, an atomic head, linked digests, exact operation fingerprints, extension preservation, and compare-and-swap mutation.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements bounded reconciliation operations, protected linked journals, exact-state mutation, idempotency, and forward recovery."
        }
      ],
      "implementation_paths": [
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/src/reconciliation.cpp",
        "modules/knowledge-session-coordinator/src/reconciliation.hpp",
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not authenticate callers, decide permission, inspect Git providers, or mark a review complete autonomously.",
        "Does not mutate repository files, canonical knowledge, SSIAG policy, STAV ledgers, or Maestro state."
      ],
      "owner_contract": "modules/knowledge-session-coordinator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "qxctl resolves and revalidates the exact coordinator installation before invoking its reconciliation operations.",
          "target_feature_id": "ssfv:symphony:qxctl.engine-bindings",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/knowledge-session-coordinator",
      "status": "experimental",
      "title": "Durable worktree reconciliation contexts",
      "what": "Persists begin, status, checkpoint, close, and recovery evidence for one repository reconciliation context.",
      "when": "Runs only when qxctl explicitly invokes a reconciliation operation; status remains read-only and recovery is explicit.",
      "where": "Stores protected noncanonical user-scope context state beneath the selected coordinator state root.",
      "who": "qxctl and target-host-authorized maintainers or agentic tools coordinating an explicitly bounded repository review context.",
      "why": "Preserves review continuity and makes interrupted or overlapping reconciliation visible without treating repository history, timestamps, or process order as proof of completion."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Maestro owns the complete read-only derived receptor inventory supplied as lineage evidence.",
          "reference": "modules/maestro/SPEC.md",
          "vector": "maestro"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSFV maintenance and inventory composition changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes SSFV maintenance contracts, implementation, qxctl grammar, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Each mutation consumes an active session and fresh exact authorization decision.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "The SSIAG authorization evidence is audit-before-release through STAV.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Semantic maintenance persists exact SSFV baseline/diff and engine lineage; reconciliation records bounded worktree review contexts without feature semantics.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.reconciliation"
        }
      ],
      "evidence": [
        "modules/knowledge-session-coordinator/tests/ssfv_maintenance_test.cpp verifies baseline persistence, replay, compare-and-swap rejection, separate engine lineage, corrupted-head detection, and digest-linked recovery.",
        "tools/qxctl/cmd/qxctl/ssfv_maintenance_test.go verifies bound authorization, canonical evidence digests, disabled canonical apply, and journal-digest enforcement."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator.semantic-maintenance",
      "how": "Each operation binds an active authority epoch, exact engine receipt and snapshot digests, immutable baseline, current engine, Maestro evidence, expected journal state, operation identity, dual-slot durability, and explicit forward recovery.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements durable baseline/current semantic lineage, exact journal mutation, compatibility, drift state, and recovery."
        },
        {
          "language": "Go",
          "role": "Collects validated SSFV snapshot/diff and complete Maestro inventory evidence under fresh authorization and invokes the coordinator."
        }
      ],
      "implementation_paths": [
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/src/ssfv_maintenance.cpp",
        "modules/knowledge-session-coordinator/src/ssfv_maintenance.hpp",
        "modules/knowledge-session-coordinator/tests/ssfv_maintenance_test.cpp",
        "tools/qxctl/cmd/qxctl/ssfv_maintenance.go",
        "tools/qxctl/cmd/qxctl/ssfv_maintenance_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not decide feature-worthiness, ratify semantics, create or edit FEATURES.md, or apply an SSFV proposal.",
        "Does not make Maestro inventory canonical, mutate Maestro state, or reinterpret an immutable baseline with a later engine."
      ],
      "owner_contract": "modules/knowledge-session-coordinator/SPEC.md",
      "parent_feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Maintenance captures complete derived receptor inventory or an explicit not-configured reason as lineage evidence.",
          "target_feature_id": "ssfv:symphony:maestro-presence-authority.complete-inventory",
          "type": "composes_with"
        },
        {
          "rationale": "qxctl supplies validated read-only SSFV snapshots and diffs while the coordinator preserves their session lineage.",
          "target_feature_id": "ssfv:symphony:ssfv-engine",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/knowledge-session-coordinator",
      "status": "experimental",
      "title": "Persistent SSFV semantic-maintenance lineage",
      "what": "Persists the initial SSFV semantic snapshot, baseline engine identity, later current-engine evidence, exact diffs, and complete or explicitly not-configured Maestro inventory lineage.",
      "when": "Begins on explicit semantic-maintenance administration, checkpoints or closes after a validated live diff, and recovers only on explicit unambiguous evidence.",
      "where": "Stores protected noncanonical context state keyed by TOPS, SSIAG subject, and repository below the coordinator state root.",
      "who": "An authenticated target-host-authorized caller maintaining semantic review evidence through qxctl.",
      "why": "Keeps semantic review stable across compatible engine upgrade and rollback order without reinterpreting the baseline or turning transient observations into canonical feature truth."
    }
  ],
  "source_scope": "modules/knowledge-session-coordinator"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
