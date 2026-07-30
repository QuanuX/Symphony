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
          "applicability": "not_applicable",
          "reason": "Reconciliation is caller-neutral and noncanonical; it does not authenticate a caller, request an SSIAG decision, or establish an authority epoch.",
          "reference": null,
          "vector": "ssiag"
        },
        {
          "applicability": "not_applicable",
          "reason": "The reconciliation slice emits no STAV event and has no append-authority producer grant.",
          "reference": null,
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
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp verifies lifecycle mutations, exact-state rejection, idempotent replay, compatible unordered engine-version evidence, capability downgrade, content drift, close, damaged-head discovery recovery, stale-temporary cleanup, extension preservation, critical-extension refusal, lock contention, symlink rejection, and worktree isolation.",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh verifies deterministic process responses, explicit disabled authority, bounded check output, duplicate-key rejection, and disabled canonical apply.",
        "modules/knowledge-session-coordinator/CMakeLists.txt builds, tests, installs, receipts, and uninstalls the exact versioned process.",
        "modules/knowledge-session-coordinator/SPEC.md defines inspect/check plus durable reconciliation, two-way procedural compatibility, evidence-preserving recovery, and the deferred authenticated-session boundary.",
        "tools/qxctl/cmd/qxctl/main.go and tools/qxctl/internal/knowledgeengine/client.go verify and invoke the exact bound coordinator while recording a role-sorted engine inventory."
      ],
      "feature_id": "ssfv:symphony:knowledge-session-coordinator",
      "how": "The C++26 symphony-knowledge-session process statically links the shared foundation, accepts the common bounded process envelope, hashes caller-supplied safe relative files without following symlinks, serializes each worktree through a private persistent lock, and commits content-addressed checkpoints through an fsync-backed dual-slot journal plus atomic head. Go qxctl revalidates its immutable binding snapshot, negotiates protocol/version/capabilities, supplies stable operation IDs and compare-and-swap state, and records the exact bound-engine inventory.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the independently installed process, descriptor, bounded snapshot check, compatibility negotiation, journal durability, idempotent reconciliation lifecycle, and evidence-preserving recovery."
        },
        {
          "language": "CMake",
          "role": "Builds and installs the exact versioned executable, module contracts, receipt, tests, and receipt-owned uninstall surface."
        },
        {
          "language": "Go",
          "role": "Implements qxctl binding revalidation, exact coordinator invocation, reconciliation command grammar, operation identity, expected-state input, and bound-engine inventory delivery."
        }
      ],
      "implementation_paths": [
        "modules/knowledge-session-coordinator/CMakeLists.txt",
        "modules/knowledge-session-coordinator/src/coordinator.cpp",
        "modules/knowledge-session-coordinator/src/coordinator.hpp",
        "modules/knowledge-session-coordinator/src/main.cpp",
        "modules/knowledge-session-coordinator/src/reconciliation.cpp",
        "modules/knowledge-session-coordinator/src/reconciliation.hpp",
        "modules/knowledge-session-coordinator/tests/coordinator_test.cpp",
        "modules/knowledge-session-coordinator/tests/process_smoke.sh",
        "tools/qxctl/cmd/qxctl/commands.go",
        "tools/qxctl/cmd/qxctl/main.go",
        "tools/qxctl/internal/knowledgeengine/client.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not authenticate a caller, establish or recover an authenticated session or authority epoch, run a watcher, or invoke a vector engine.",
        "Does not implement canonical apply; it mutates only protected noncanonical user-scope reconciliation state.",
        "Does not integrate SSIAG, STAV, or Maestro and does not mutate canonical repository state.",
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
      "title": "Durable knowledge reconciliation coordinator",
      "what": "Provides an independently installable administrative process that reports exact capabilities, computes deterministic snapshots, and maintains a recoverable noncanonical reconciliation context for each canonical worktree.",
      "when": "Runs only on explicit user-scope qxctl or exact process invocation under a bounded deadline. A context begins with an explicit inventory, checkpoints or closes under exact expected state, and repairs only through an explicit recovery operation.",
      "where": "Executes as the inactive, installed-undocked symphony-knowledge-session C++ process in the administrative freezing path and roots checks at its current repository working directory.",
      "who": "Any host-authorized caller using qxctl, plus maintainers and tests that need a caller-neutral, domain-neutral reconciliation boundary across independently versioned SKV engines.",
      "why": "Preserves coherent worktree and engine-version evidence when upgrades, retries, or crashes occur out of sequence, while providing a safe independently versioned landing zone for future authenticated session coordination."
    }
  ],
  "source_scope": "modules/knowledge-session-coordinator"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
