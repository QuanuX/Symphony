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
          "reason": "The installed-undocked coordinator can report an exact desired receptor and rejects dock/undock execution; its bindings, report/apply journals, and applied evidence remain outside Maestro.",
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
          "applicability": "applicable",
          "reason": "Every authenticated-session and qxctl lifecycle administration operation consumes a fresh exact caller-neutral SSIAG decision derived from kernel peer identity; the coordinator validates session capability evidence without deciding authority, while qxctl binds lifecycle decisions to exact profile or observation evidence.",
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
        "modules/knowledge-session-coordinator/tests/process_smoke.sh verifies deterministic process responses, implemented session capability truth, bounded check output, duplicate-key rejection, and disabled canonical apply.",
        "modules/knowledge-session-coordinator/CMakeLists.txt builds, tests, installs, receipts, and uninstalls the exact versioned process.",
        "modules/knowledge-session-coordinator/SPEC.md defines inspect/check plus separate durable reconciliation and authenticated-session journals, two-way procedural compatibility, evidence-preserving recovery, and the canonical-apply boundary.",
        "tools/qxctl/cmd/qxctl/session_test.go verifies explicit login, refresh, logout, retry, bounded recovery, reauthentication, and interrupted-close resumption over the coordinator primitives.",
        "tools/qxctl/internal/knowledgelifecycle/profile_test.go verifies protected desired-profile compare-and-swap, semantic retry, linked generations, fixed-layout v1/v2 receipt observation, content drift, unknown-package preservation, and stable inventory identity.",
        "tools/qxctl/cmd/qxctl/lifecycle_test.go verifies report-plan safety, dynamic scheduler invariants, staged-install exception isolation, strict v1/v2 lifecycle-journal IPC validation, exact apply result validation, critical-extension refusal, and lifecycle Cobra grammar.",
        "tools/qxctl/internal/knowledgelifecycle/executor_test.go verifies protected runtime compare-and-swap/idempotency, prepared observation-drift refusal, staged receipt-v2 install/uninstall and interruption replay, separate rollback proof, and conflicting administrator-file protection.",
        "tools/qxctl/cmd/qxctl/main.go, tools/qxctl/cmd/qxctl/lifecycle.go, tools/qxctl/cmd/qxctl/lifecycle_apply.go, tools/qxctl/internal/knowledgelifecycle/executor.go, tools/qxctl/internal/ssiagclient/client.go, and tools/qxctl/internal/knowledgeengine/client.go authenticate SSIAG, validate safe authorization evidence, invoke the exact bound coordinator, collect lifecycle observations, serialize external actions, and re-observe before finalization."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "how": "The C++26 symphony-knowledge-session process statically links the shared foundation, accepts the common bounded process envelope, and keeps reconciliation contexts, authenticated authority epochs, report lifecycle streams, and apply lifecycle streams separate. Each durable stream uses a private no-follow lock, fsync-backed dual slots, an atomic head, content digests, stable operation IDs, exact compare-and-swap state, opaque-extension preservation, and evidence-based forward recovery. The lifecycle planner validates complete caller-supplied desired and observed evidence and derives stable forward/inverse actions from a dependency-ready set. Report boot persists non-executable source evidence in v1. Separately authorized v2 prepare records one active attempt before external action; finalize accepts complete re-observation only when it proves the target; close selects immutable content-addressed applied evidence. Stable inventory excludes only collection-time document fields. Go qxctl revalidates the exact coordinator, obtains fresh SSIAG decisions per phase, maintains desired and generic runtime state, scans fixed receipt layouts, executes only reviewed staged receipt-v2 or selection/activation adapters, and re-observes before finalization. The planner isolates cycles and blockers, reports noncritical advisories, binds exact receptors, and safely sequences package changes without the coordinator discovering packages or performing host mutation. Explicit session transitions retain their independent stable-event recovery behavior.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the independently installed process, descriptor, bounded snapshot check, compatibility negotiation, separate reconciliation/session/report/apply journal durability, SSIAG evidence validation, dependency-driven lifecycle planning, prepared attempts, verified applied-state commitment, idempotent lifecycles, and evidence-preserving recovery."
        },
        {
          "language": "CMake",
          "role": "Builds and installs the exact versioned executable, module contracts, receipt, tests, and receipt-owned uninstall surface."
        },
        {
          "language": "Go",
          "role": "Implements qxctl binding revalidation, kernel-authenticated SSIAG authorization requests, decision-boundary validation, exact coordinator invocation, reconciliation/session/lifecycle command grammar, protected lifecycle desired/runtime state, fixed-layout receipt observation, staged receipt-v2 install/uninstall, selection/activation adapters, explicit login/refresh/logout transition composition, operation identity, expected-state input, and bound-engine inventory delivery."
        }
      ],
      "implementation_paths": [
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
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp",
        "modules/knowledge-session-coordinator/tests/lifecycle_test.cpp",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh",
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/lifecycle.go",
        "tools/qxctl/cmd/qxctl/lifecycle_apply.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/internal/knowledgeengine/client.go",
        "tools/qxctl/internal/knowledgelifecycle/executor.go",
        "tools/qxctl/internal/knowledgelifecycle/install_unix.go",
        "tools/qxctl/internal/knowledgelifecycle/observation.go",
        "tools/qxctl/internal/knowledgelifecycle/profile.go",
        "tools/qxctl/internal/knowledgelifecycle/runtime.go",
        "tools/qxctl/internal/knowledgelifecycle/scan_unix.go",
        "tools/qxctl/internal/knowledgelifecycle/state_unix.go",
        "tools/qxctl/internal/ssiagclient/client.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not independently authenticate the operating-system caller or decide permission; SSIAG owns those decisions, and the coordinator validates only the supplied exact safe evidence.",
        "Does not implement canonical apply; it mutates only protected noncanonical reconciliation, authenticated-session, lifecycle-journal, runtime-state, receipt-v2 package, and applied-state surfaces within the explicit lifecycle contract.",
        "Does not call SSIAG or STAV directly, consume credentials or provider secrets, integrate Maestro, or mutate canonical repository state.",
        "The coordinator does not collect lifecycle observations, administer desired/runtime profiles, or execute host actions; qxctl performs those functions while the coordinator serializes attempts and verifies applied evidence.",
        "Does not mutate receipt-v1 packages, download packages, execute arbitrary receipt entry points, activate live services/processes, replace the active coordinator, rewrite engine bindings, or dock with Maestro.",
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
      "title": "Durable knowledge session, reconciliation, and lifecycle coordinator",
      "what": "Provides an independently installable administrative process that reports exact capabilities, computes deterministic snapshots, maintains recoverable noncanonical reconciliation and authority-epoch state, converts complete desired/observed lifecycle evidence into a deterministic dependency plan, preserves report-only boot evidence, and separately coordinates prepared actions, verified completion, applied-state commitment, and recovery per TOPS and profile.",
      "when": "Runs only on explicit user-scope qxctl or exact process invocation under a bounded deadline. Reconciliation begins with an explicit inventory. Session and lifecycle operations each require fresh exact SSIAG evidence. Report/boot re-read protected profiles and re-observe configured roots. Explicit apply is eligible only for apply-compatible profiles with exact source/journal/applied compare-and-swap; every external action is prepared durably and followed by complete re-observation. Status is read-only and recovery is explicit. No operation acquires canonical apply authority.",
      "where": "Executes as the inactive, installed-undocked symphony-knowledge-session C++ process in the administrative freezing path and roots checks at its current repository working directory.",
      "who": "Any host-authorized caller using qxctl, plus maintainers and tests that need caller-neutral, domain-neutral authority-epoch and reconciliation boundaries across independently versioned SKV engines.",
      "why": "Preserves coherent authority-epoch, worktree, and engine-version evidence when upgrades, retries, logouts, or crashes occur out of sequence, and allows compatible lifecycle work to reorder around localized blockers while separating SSIAG authorization, session state, reconciliation state, lifecycle reports, vector semantics, and apply authority."
    }
  ],
  "source_scope": "modules/knowledge-session-coordinator"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
