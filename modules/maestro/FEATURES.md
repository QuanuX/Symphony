# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/maestro/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Maestro owns protected operational docking presence while this SSFV record owns the application-level meaning of that implemented capability.",
          "reference": "modules/maestro/SPEC.md",
          "vector": "maestro"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed changes to the Maestro presence authority and lifecycle integration.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the Maestro contracts, implementation, qxctl client, and this distributed feature record without owning its semantics.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Every status, docking, undocking, and recovery operation consumes a fresh exact caller-neutral SSIAG decision derived from target-host authority.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG releases no Maestro authorization decision until the corresponding safe STAV policy-decision event commits.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "The knowledge-session coordinator derives, serializes, and verifies lifecycle actions; Maestro alone persists exact docking presence and neither component executes the recorded engine.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator"
        },
        {
          "distinction": "The shared foundation provides statically linked authority-free mechanics; Maestro is an independently installed authenticated state authority with its own durable receptor streams.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation"
        }
      ],
      "evidence": [
        "modules/maestro/tests/maestro_test.cpp verifies exact receptor identity, authorization binding, semantic retry, compare-and-swap generations, replacement refusal, recovery, symlink rejection, lock contention, unknown-state preservation, complete sorted derived inventory, stable inventory digests, and capability refusal.",
        "modules/maestro/tests/process_smoke.sh verifies the bounded local process envelope and descriptor surface.",
        "modules/maestro/src/maestro.cpp implements authenticated inspect, status, dock, undock, and recovery over protected dual-slot receptor registries.",
        "tools/qxctl/cmd/qxctl/maestro.go and tools/qxctl/internal/maestroclient/client.go validate exact installed Maestro identity, obtain SSIAG evidence, invoke bounded operations, and verify response digests.",
        "tools/qxctl/cmd/qxctl/lifecycle_apply.go and tools/qxctl/internal/knowledgelifecycle/executor.go connect dependency-planned dock/undock actions to exhaustive authenticated receptor observation and verified re-observation.",
        "modules/maestro/CMakeLists.txt builds, tests, installs, receipts, and uninstalls the exact versioned process."
      ],
      "feature_id": "ssfv:symphony:maestro-presence-authority",
      "how": "The independently installed C++26 symphony-maestro process accepts bounded local process requests and maintains one protected per-TOPS/per-receptor registry stream. Each stream uses a private no-follow lock, alternating synchronized slots, an atomic head, linked generations, exact compare-and-swap state, content digests, semantic operation identities, and unique forward recovery. Its authenticated inventory operation locks and validates every existing receptor stream, fails closed rather than returning a partial view, sorts exact receptor/component evidence, computes a timestamp-independent stable inventory digest, and wraps it in separately timestamped observation evidence without persisting a second registry. qxctl validates the exact installation, obtains fresh SSIAG evidence for each protected operation, and verifies all response digests. Lifecycle apply supplies an exhaustive receptor set, discovers existing presence before inverse work, serializes dock or undock through the coordinator, commits presence through Maestro, and re-observes before applied evidence can advance.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Implements the bounded Maestro process, exact receptor descriptors, authenticated presence mutation, durable dual-slot registries, status, idempotency, and evidence-preserving recovery."
        },
        {
          "language": "CMake",
          "role": "Builds, tests, installs, receipts, and uninstalls the exact versioned Maestro package."
        },
        {
          "language": "Go",
          "role": "Implements qxctl Maestro administration, exact installation validation, caller-neutral SSIAG exchange, exhaustive lifecycle receptor observation, docking adapters, and response verification."
        }
      ],
      "implementation_paths": [
        "modules/maestro/CMakeLists.txt",
        "modules/maestro/src/maestro.cpp",
        "modules/maestro/src/maestro.hpp",
        "modules/maestro/src/main.cpp",
        "modules/maestro/tests/maestro_test.cpp",
        "modules/maestro/tests/process_smoke.sh",
        "tools/qxctl/cmd/qxctl/lifecycle_apply.go",
        "tools/qxctl/cmd/qxctl/maestro.go",
        "tools/qxctl/internal/knowledgeengine/client.go",
        "tools/qxctl/internal/knowledgelifecycle/executor.go",
        "tools/qxctl/internal/knowledgelifecycle/observation.go",
        "tools/qxctl/internal/maestroclient/client.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not start, invoke, schedule, supervise, patch, or assign compute resources to a docked engine.",
        "Does not own vector semantics, desired-state policy, package truth, authentication policy, canonical knowledge, or feature truth outside this SSFV record.",
        "Does not mutate installation receipts, select or activate versions, download packages, run entry points, or replace the lifecycle coordinator.",
        "Does not create a network listener, remote control plane, canonical apply route, or trading hot/warm-path dependency.",
        "Does not infer permission from caller type; every protected operation is governed by target-host identity and granted authority."
      ],
      "owner_contract": "modules/maestro/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The lifecycle coordinator derives and serializes docking actions while Maestro persists their exact authenticated presence outcome.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator",
          "type": "composes_with"
        },
        {
          "rationale": "Maestro statically links the shared bounded process, digest, path, and temporal mechanics.",
          "target_feature_id": "ssfv:symphony:knowledge-vector-engine-foundation",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/maestro",
      "status": "experimental",
      "title": "Authenticated durable Maestro docking presence",
      "what": "Provides the independently installable freezing-path authority that records which exact compatible vector-engine installation is docked to which Maestro receptor for one TOPS, with safe inspection, complete derived inventory, mutation, retry, and recovery.",
      "when": "Runs only on explicit qxctl inspection, status, lifecycle docking/undocking, or recovery under a bounded deadline and fresh SSIAG authorization. It is not continuously resident and does not execute when recorded engines perform their own work.",
      "where": "Executes locally on the Maestro-hosting TOPS node and stores protected noncanonical state under the selected per-TOPS Maestro state boundary. Its work remains outside hot and warm trading paths.",
      "who": "Any target-host-authorized caller using qxctl, lifecycle administrators, the knowledge-session coordination circuit, maintainers, and tests that need exact engine-to-receptor presence evidence.",
      "why": "Gives independently versioned and independently installed vector engines a durable receptor docking record without turning installation, selection, activation, semantic truth, or process execution into the same state transition."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Maestro owns receptor discovery, validation, stable inventory, and observation semantics.",
          "reference": "modules/maestro/SPEC.md",
          "vector": "maestro"
        },
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed complete inventory and client integration changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes Maestro inventory schemas, implementation, qxctl client, and tests.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "Inventory requires a fresh exact caller-neutral SSIAG decision.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG releases inventory authorization only after its safe audit event is committed through STAV.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "The parent feature owns durable status/dock/undock/recovery presence; this subfeature is a complete derived read-only TOPS-wide projection and never a second registry.",
          "target_feature_id": "ssfv:symphony:maestro-presence-authority"
        }
      ],
      "evidence": [
        "modules/maestro/tests/maestro_test.cpp verifies complete sorted inventory, shared-lock behavior, stable/timestamped digest separation, unsafe-state refusal, and no partial result.",
        "tools/qxctl/internal/maestroclient/client_test.go verifies exact authorization binding, identity, completeness, inventory digest, and observation validation."
      ],
      "feature_id": "ssfv:symphony:maestro-presence-authority.complete-inventory",
      "how": "Maestro validates directory names, ownership, modes, links, shared locks, heads, slots, digest chains, TOPS and receptor identities, then derives a stable inventory digest separately from the STSC whole-second UTC observation envelope.",
      "implementation_languages": [
        {
          "language": "C++26",
          "role": "Discovers and validates every protected receptor stream and derives one sorted stable inventory plus timestamped observation evidence."
        },
        {
          "language": "Go",
          "role": "Obtains fresh authorization, invokes the exact installed Maestro process, and validates complete inventory results through qxctl."
        }
      ],
      "implementation_paths": [
        "modules/maestro/src/maestro.cpp",
        "modules/maestro/src/maestro.hpp",
        "modules/maestro/src/main.cpp",
        "modules/maestro/tests/maestro_test.cpp",
        "tools/qxctl/cmd/qxctl/maestro.go",
        "tools/qxctl/internal/maestroclient/client.go",
        "tools/qxctl/internal/maestroclient/client_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not create or repair receptor state, persist a second inventory, or return a partial view as complete or empty.",
        "Does not execute, schedule, supervise, load, or health-check docked engines and does not make the projection canonical."
      ],
      "owner_contract": "modules/maestro/SPEC.md",
      "parent_feature_id": "ssfv:symphony:maestro-presence-authority",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Semantic maintenance consumes complete inventory or an explicit not-configured reason as noncanonical lineage evidence.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.semantic-maintenance",
          "type": "composes_with"
        },
        {
          "rationale": "qxctl supplies exact authorization and validates the inventory result without owning Maestro state.",
          "target_feature_id": "ssfv:symphony:qxctl.maestro-administration",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/maestro",
      "status": "experimental",
      "title": "Complete derived Maestro receptor inventory",
      "what": "Returns one complete sorted read-only inventory of every valid Maestro receptor and its exact docked component identity for a TOPS.",
      "when": "Runs only on explicit authenticated inventory requests or when qxctl collects complete evidence for lifecycle or SSFV maintenance.",
      "where": "Reads protected per-TOPS Maestro receptor registries on the local Maestro-hosting node without creating, repairing, or persisting a second inventory.",
      "who": "Any target-host-authorized caller, lifecycle administrator, or semantic-maintenance circuit requiring exact TOPS-wide receptor evidence.",
      "why": "Prevents partial, busy, damaged, or ambiguous receptor state from being mistaken for absence while giving other freezing-path circuits stable evidence across observation times."
    }
  ],
  "source_scope": "modules/maestro"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
