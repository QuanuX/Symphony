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
          "reason": "The installed-undocked coordinator creates no Maestro receptor, activation, or persisted deployment state; its user-scope binding and reconciliation journal remain outside Maestro.",
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
          "reason": "Every authenticated-session operation consumes a fresh exact caller-neutral SSIAG decision derived from kernel peer identity; the coordinator validates its non-transferable capability evidence without deciding authority.",
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
        "modules/knowledge-session-coordinator/tests/process_smoke.sh verifies deterministic process responses, implemented session capability truth, bounded check output, duplicate-key rejection, and disabled canonical apply.",
        "modules/knowledge-session-coordinator/CMakeLists.txt builds, tests, installs, receipts, and uninstalls the exact versioned process.",
        "modules/knowledge-session-coordinator/SPEC.md defines inspect/check plus separate durable reconciliation and authenticated-session journals, two-way procedural compatibility, evidence-preserving recovery, and the canonical-apply boundary.",
        "tools/qxctl/cmd/qxctl/main.go, tools/qxctl/internal/ssiagclient/client.go, and tools/qxctl/internal/knowledgeengine/client.go authenticate SSIAG, validate safe authorization evidence, invoke the exact bound coordinator, and record the role-sorted engine inventory for reconciliation."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "how": "The C++26 symphony-knowledge-session process statically links the shared foundation, accepts the common bounded process envelope, and keeps reconciliation contexts separate from authenticated authority epochs. Each state stream uses a private no-follow lock, fsync-backed dual slots, an atomic head, content digests, stable operation IDs, exact compare-and-swap state, opaque-extension preservation, and evidence-based forward recovery. Go qxctl revalidates its immutable coordinator binding, authenticates the TOPS-scoped SSIAG endpoint, obtains and validates one exact audited decision per session operation, negotiates protocol/version/capabilities, invokes that exact coordinator, and supplies a role-sorted bound-engine inventory only to reconciliation operations.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the independently installed process, descriptor, bounded snapshot check, compatibility negotiation, separate reconciliation/session journal durability, SSIAG evidence validation, idempotent lifecycles, and evidence-preserving recovery."
        },
        {
          "language": "CMake",
          "role": "Builds and installs the exact versioned executable, module contracts, receipt, tests, and receipt-owned uninstall surface."
        },
        {
          "language": "Go",
          "role": "Implements qxctl binding revalidation, kernel-authenticated SSIAG authorization requests, decision-boundary validation, exact coordinator invocation, reconciliation/session command grammar, operation identity, expected-state input, and bound-engine inventory delivery."
        }
      ],
      "implementation_paths": [
        "modules/knowledge-session-coordinator/CMakeLists.txt",
        "modules/knowledge-session-coordinator/src/authority_session.cpp",
        "modules/knowledge-session-coordinator/src/authority_session.hpp",
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/src/coordinator.hpp",
        "modules/knowledge-session-coordinator/src/main.cpp",
        "modules/knowledge-session-coordinator/src/reconciliation.cpp",
        "modules/knowledge-session-coordinator/src/reconciliation.hpp",
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh",
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/internal/knowledgeengine/client.go",
        "tools/qxctl/internal/ssiagclient/client.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not independently authenticate the operating-system caller or decide permission; SSIAG owns those decisions, and the coordinator validates only the supplied exact safe evidence.",
        "Does not implement canonical apply; it mutates only protected noncanonical user-scope reconciliation and authenticated-session state.",
        "Does not call SSIAG or STAV directly, consume credentials or provider secrets, integrate Maestro, or mutate canonical repository state.",
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
      "title": "Durable knowledge session and reconciliation coordinator",
      "what": "Provides an independently installable administrative process that reports exact capabilities, computes deterministic snapshots, maintains one recoverable noncanonical reconciliation context per canonical worktree, and maintains separately authorized noncanonical authority epochs per TOPS, subject, and repository.",
      "when": "Runs only on explicit user-scope qxctl or exact process invocation under a bounded deadline. Reconciliation begins with an explicit inventory. Session operations each require fresh exact SSIAG evidence. Both lifecycles checkpoint or close under exact expected state and repair only through explicit evidence-based recovery.",
      "where": "Executes as the inactive, installed-undocked symphony-knowledge-session C++ process in the administrative freezing path and roots checks at its current repository working directory.",
      "who": "Any host-authorized caller using qxctl, plus maintainers and tests that need caller-neutral, domain-neutral authority-epoch and reconciliation boundaries across independently versioned SKV engines.",
      "why": "Preserves coherent authority-epoch, worktree, and engine-version evidence when upgrades, retries, logouts, or crashes occur out of sequence, while separating SSIAG authorization, session state, reconciliation state, vector semantics, and canonical apply authority."
    }
  ],
  "source_scope": "modules/knowledge-session-coordinator"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
