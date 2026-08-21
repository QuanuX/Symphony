# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/accordare-stav-producer/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SKVI routes the owning SAV Named Version audit contract.",
          "reference": "knowledge/sav/STAV.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns authorization decisions.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns the event envelope and vocabulary.",
          "reference": "knowledge/ACCORD-AUDIT.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Derives safe Accordare candidates; the append authority alone assigns ledger identity, order, time, producer, and integrity.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.serialized-append"
        }
      ],
      "evidence": [
        "Protocol, outbox, producer, package, and qxctl grant regressions cover closed mapping, tamper rejection, crash durability, replay, and exact grant state."
      ],
      "feature_id": "ssfv:symphony:accordare-stav-producer",
      "how": "Authenticates qxctl by kernel credentials, verifies exact SSIAG and coordinator evidence, derives a closed safe candidate, fsyncs it to a private outbox, and submits it over authenticated STAV IPC.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free freezing-path producer, client, outbox, IPC, package, and administration bindings."
        }
      ],
      "implementation_paths": [
        "modules/accordare-stav-producer/client/client.go",
        "modules/accordare-stav-producer/internal/outbox/outbox.go",
        "modules/accordare-stav-producer/internal/protocol/validate.go",
        "modules/accordare-stav-producer/internal/server/server.go",
        "tools/qxctl/cmd/qxctl/stav_accordare.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not grant itself, append arbitrary events, own SAV/SSIAG/STAV truth, edit a ledger, or operate on hot or warm paths."
      ],
      "owner_contract": "modules/accordare-stav-producer/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "All commits pass through the sole per-TOPS append authority.",
          "target_feature_id": "ssfv:symphony:stav-append-authority",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/accordare-stav-producer",
      "status": "experimental",
      "title": "Durable Accordare Named Version audit producer",
      "what": "Converts four terminal SAV Named Version lifecycle outcomes into safe, durable, installation-granted STAV events.",
      "when": "After a configured qxctl Named Version mutation and during startup or explicit pending reconciliation.",
      "where": "On the Maestro/freezing-path TOPS node between qxctl/coordinator IPC and the local STAV authority.",
      "who": "Any host-authorized caller using qxctl, SSIAG, the coordinator, the isolated producer service, STAV, and TOPS administrators.",
      "why": "Powerful composition changes require durable audit truth without making the orchestration CLI or persistence coordinator a ledger writer."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SSIAG authorizes qxctl grant mutation.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns exact producer grants.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Administers one installation grant but never provides an arbitrary qxctl append command.",
          "target_feature_id": "ssfv:symphony:qxctl.stav-administration"
        }
      ],
      "evidence": [
        "Grant tests cover exact permissions, idempotency, explicit replacement, removal, and recovery from either side of file replacement."
      ],
      "feature_id": "ssfv:symphony:accordare-stav-producer.grant-administration",
      "how": "qxctl binds SSIAG authorization, producer enrollment identity, stopped-authority state, expected config digest, exact four permissions, atomic replacement, and an old/new recovery marker.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements Cobra administration, CAS, durable recovery, and strict STAV configuration validation."
        }
      ],
      "implementation_paths": [
        "tools/qxctl/cmd/qxctl/stav_grant.go",
        "tools/qxctl/cmd/qxctl/stav_grant_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not infer grants from installation, enrollment, caller class, vocabulary validity, or socket access."
      ],
      "owner_contract": "modules/accordare-stav-producer/SPEC.md",
      "parent_feature_id": "ssfv:symphony:accordare-stav-producer",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Extends the existing qxctl STAV administrative surface with two stable exact commands.",
          "target_feature_id": "ssfv:symphony:qxctl.stav-administration",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/accordare-stav-producer",
      "status": "experimental",
      "title": "Installation-specific Accordare STAV grant administration",
      "what": "Installs or removes only the ratified Accordare producer grant through protected qxctl expected-state administration.",
      "when": "While the selected STAV authority is stopped during explicit installation, replacement, or removal.",
      "where": "At the per-TOPS STAV append-authority configuration boundary.",
      "who": "Any caller holding the exact host/SSIAG permission through headless qxctl.",
      "why": "Module presence must never silently become append authority, while upgrades and interrupted administration must recover predictably."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SKVI routes SAV and coordinator command/result truth.",
          "reference": "knowledge/sav/STAV.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns terminal audit vocabulary and append truth.",
          "reference": "knowledge/ACCORD-AUDIT.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Persists authenticated intent before coordinator mutation; the candidate outbox begins only after terminal evidence exists.",
          "target_feature_id": "ssfv:symphony:accordare-stav-producer"
        }
      ],
      "evidence": [
        "Intent-store, protocol, producer, and real STAV acceptance regressions cover exact retry, conflict rejection, separate pending classes, and committed append."
      ],
      "feature_id": "ssfv:symphony:accordare-stav-producer.intent-durability",
      "how": "qxctl prepares an exact authenticated command intent before coordinator invocation. The producer fsyncs a deterministic intent identity, accepts completion only for identical peer, command, coordinator, operation, and TOPS evidence, then persists and appends the derived terminal candidate. Exact command retry repairs any interruption.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements pre-mutation intent storage, two-phase local IPC, terminal derivation, and exact retry recovery."
        }
      ],
      "implementation_paths": [
        "modules/accordare-stav-producer/internal/intent/store.go",
        "modules/accordare-stav-producer/internal/protocol/validate.go",
        "modules/accordare-stav-producer/internal/server/server.go",
        "tools/qxctl/cmd/qxctl/named_version.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not infer whether an interrupted coordinator mutated, erase unresolved intent, use timestamps for ordering, or fabricate terminal evidence."
      ],
      "owner_contract": "modules/accordare-stav-producer/SPEC.md",
      "parent_feature_id": "ssfv:symphony:accordare-stav-producer",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Closes the mutation-to-audit crash window through durable preparation and exact replay.",
          "target_feature_id": "ssfv:symphony:knowledge-session-coordinator.named-version-durability",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/accordare-stav-producer",
      "status": "experimental",
      "title": "Two-phase Accordare audit intent durability",
      "what": "Makes configured Named Version mutations recoverably auditable across process interruption before terminal candidate creation.",
      "when": "Before every configured mutation and again when the exact operation completes or is retried.",
      "where": "In a protected per-TOPS intent store distinct from the STAV candidate outbox.",
      "who": "Any host-authorized qxctl caller, the coordinator, the authenticated Accordare producer, and administrators inspecting recovery state.",
      "why": "A coordinator commit must not become permanently unauditable because orchestration stopped before producer submission."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "Module lifecycle remains separate from ledger grants and SSIAG permission.",
          "reference": "knowledge/ACCORD-AUDIT.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Owns only producer liveness; STAV and SSIAG retain independent supervisors and no startup dependency is introduced.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.native-supervision"
        }
      ],
      "evidence": [
        "Descriptor tests verify exact binary/TOPS binding, bounded restart and shutdown, caller identity, and absence of cross-service dependencies."
      ],
      "feature_id": "ssfv:symphony:accordare-stav-producer.native-supervision",
      "how": "The module renders per-TOPS launchd or systemd descriptors that pin the exact receipted executable and enrolled service identity. qxctl verifies the receipt before invoking install or uninstall; native managers own liveness only.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements cgo-free native descriptors, manager invocation, exact receipt verification, and qxctl lifecycle administration."
        }
      ],
      "implementation_paths": [
        "modules/accordare-stav-producer/cmd/symphony-accordare-stav-producer/main.go",
        "modules/accordare-stav-producer/internal/supervision/supervision.go",
        "tools/qxctl/cmd/qxctl/stav_accordare.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not grant STAV append authority, create identities, couple service startup, support native Windows, or supervise hot/warm-path work."
      ],
      "owner_contract": "modules/accordare-stav-producer/SPEC.md",
      "parent_feature_id": "ssfv:symphony:accordare-stav-producer",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Uses the same independent per-TOPS liveness posture without coupling startup.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.native-supervision",
          "type": "composes_with"
        }
      ],
      "source_scope": "modules/accordare-stav-producer",
      "status": "experimental",
      "title": "Per-TOPS native Accordare producer supervision",
      "what": "Installs or removes one independently supervised Accordare producer receptor for an enrolled TOPS.",
      "when": "After exact package installation and enrollment, before ordinary headless audit production.",
      "where": "In user or system launchd/systemd domains on macOS or Linux.",
      "who": "Host-authorized qxctl callers, target-host administrators, native supervisors, and owner-provided equivalent supervisors.",
      "why": "A durable audit producer must restart predictably without making qxctl, SSIAG, or STAV its parent process."
    }
  ],
  "source_scope": "modules/accordare-stav-producer"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
