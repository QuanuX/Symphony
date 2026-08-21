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
      "what": "Converts four successful SAV Named Version lifecycle results into safe, durable, installation-granted STAV events.",
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
    }
  ],
  "source_scope": "modules/accordare-stav-producer"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
