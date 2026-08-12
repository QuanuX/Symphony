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
    }
  ],
  "source_scope": "tools/qxctl"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
