# Symphony Semantic Features

<!-- symphony:ssfv:feature-file:v1:begin -->
```json
{
  "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
  "protocol": "symphony.ssfv.feature-file.v1",
  "records": [
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG changes.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes SSIAG contracts and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future SSIAG release.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG vector truth owns the identity, policy, capability, provider, and safeguard semantics.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns the committed safe audit evidence required before decisions are released.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "SSIAG owns identity and authorization decisions; qxctl is the caller-neutral administrative client and does not own policy truth.",
          "target_feature_id": "ssfv:symphony:qxctl"
        }
      ],
      "evidence": [
        "Peer-authentication tests cover Darwin/Linux credential extraction, exact mapping, ambiguity refusal, and endpoint mismatch.",
        "Policy, server, STAV producer, lifecycle, and supervision tests cover exact grants, non-transferable bindings, audit-before-release, and per-TOPS isolation.",
        "Policy-administration tests cover config/overlay selection, CAS, durable prepare/audit/commit stages, exact recovery evidence, reset, tamper and symlink refusal, and live evaluator activation."
      ],
      "feature_id": "ssfv:symphony:ssiag-foundation",
      "how": "Darwin and Linux kernel peer credentials map exact UID/GID identities to canonical subjects; target-host ownership or exact current grants authorize protected local policy proposals; CAS, a durable attempt journal, idempotent STAV audit, atomic state replacement, and live snapshot exchange complete apply or recovery.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free per-TOPS service, kernel peer authentication, canonical subject mapping, exact policy decisions, endpoint trust, STAV production, and supervision."
        }
      ],
      "implementation_paths": [
        "modules/secure-identity-access-governance/cmd/symphony-ssiag/main.go",
        "modules/secure-identity-access-governance/internal/peerauth/peerauth.go",
        "modules/secure-identity-access-governance/internal/policy/policy.go",
        "modules/secure-identity-access-governance/internal/policyadmin/manager.go",
        "modules/secure-identity-access-governance/internal/policyadmin/storage_unix.go",
        "modules/secure-identity-access-governance/internal/server/server.go",
        "modules/secure-identity-access-governance/internal/server/server_test.go",
        "modules/secure-identity-access-governance/internal/stavproducer/producer.go"
      ],
      "kind": "feature",
      "non_claims": [
        "Does not infer authority from whether a caller is human, AI, agentic, automated, a service, or an organization.",
        "The current record does not claim canonical knowledge apply, operational credential delivery, remote access, or operational Keychain use."
      ],
      "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
      "parent_feature_id": "ssfv:symphony:platform",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Audited authorization decisions fail closed unless the per-TOPS STAV authority commits their safe outcome evidence.",
          "target_feature_id": "ssfv:symphony:stav-append-authority",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/secure-identity-access-governance",
      "status": "experimental",
      "title": "Caller-neutral secure identity and access governance foundation",
      "what": "Provides independently installable per-TOPS local identity, authentication, exact deny-by-default authorization, protected operational policy administration, non-transferable capability evidence, safe provider metadata, and audited decision services.",
      "when": "Used for explicit administrative cold or freezing-path operations requiring fresh target-host authorization; it is not consulted inline by hot or warm trading work.",
      "where": "Runs on the target TOPS node over a protected Unix socket with isolated configuration, service identity, runtime state, and native supervision.",
      "who": "Any target-host-authorized caller, qxctl, protected Symphony administrative consumers, TOPS owners, maintainers, and provider adapters.",
      "why": "Makes host ownership and granted permission—not caller class—the uniform gate for protected Symphony administration while preventing reusable bearer authority or secret leakage."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the SSIAG contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future SSIAG release or derived documentation projection.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns exact caller-neutral policy and capability semantics.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns the committed safe audit evidence required before a decision is released.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Evaluates current policy and emits bounded evidence; policy administration changes protected effective policy.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.policy-administration"
        }
      ],
      "evidence": [
        "Policy and server tests cover exact grants, deny-by-default evaluation, capability binding, expiry, ambiguity refusal, and audit-before-release."
      ],
      "feature_id": "ssfv:symphony:ssiag-foundation.authorization-capabilities",
      "how": "Validates fresh UTC intent, evaluates a deterministic deny-by-default exact-grant policy, binds the decision to subject, TOPS, authority basis, request and correlation identity, policy and configuration digests, and expiry, and releases the result only after safe STAV commitment.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements cgo-free exact policy evaluation, request validation, safe decision responses, and non-transferable capability binding."
        }
      ],
      "implementation_paths": [
        "modules/secure-identity-access-governance/internal/model/model.go",
        "modules/secure-identity-access-governance/internal/policy/policy.go",
        "modules/secure-identity-access-governance/internal/policy/policy_test.go",
        "modules/secure-identity-access-governance/internal/server/server.go",
        "modules/secure-identity-access-governance/internal/server/server_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Capabilities are not transferable tokens, credentials, generic permission grants, canonical-apply authority, or evidence of legal capacity."
      ],
      "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The authorization subject and service endpoint must first be established from kernel-attested identity.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.kernel-peer-trust",
          "type": "depends_on"
        },
        {
          "rationale": "A decision is released only after its safe outcome has a committed audit receipt.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.safe-audit-production",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/secure-identity-access-governance",
      "status": "experimental",
      "title": "Exact caller-neutral authorization and bounded capability evidence",
      "what": "Produces exact allow or deny decisions and non-transferable, short-lived capability evidence for one operation, resource, audience, and scope tuple.",
      "when": "For each cold or freezing-path protected administrative operation requiring current target-host permission.",
      "where": "Inside the per-TOPS SSIAG service and its local authorization API.",
      "who": "Kernel-authenticated subjects, qxctl, protected administrative consumers, TOPS owners, and maintainers.",
      "why": "Gives all permission-backed callers the same operation-specific authority path without reusable bearer permission or caller-class discrimination."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the SSIAG contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future SSIAG release or derived documentation projection.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns local peer and endpoint trust semantics.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        }
      ],
      "distinctions": [
        {
          "distinction": "Establishes trustworthy identity and endpoint binding; authorization capabilities decide whether that subject may perform an operation.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.authorization-capabilities"
        }
      ],
      "evidence": [
        "Peer-authentication and client tests cover Darwin and Linux credentials, exact subject mapping, trusted configuration, socket ownership, and endpoint mismatch refusal."
      ],
      "feature_id": "ssfv:symphony:ssiag-foundation.kernel-peer-trust",
      "how": "Darwin LOCAL_PEERCRED and LOCAL_PEERPID or Linux SO_PEERCRED yields connection identity; exact UID and GID mappings resolve canonical subjects; clients validate protected configuration, pre-dial socket ownership, and post-dial service UID and GID before application bytes.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements cgo-free Darwin and Linux kernel peer authentication, subject resolution, protected trust loading, and client endpoint verification."
        }
      ],
      "implementation_paths": [
        "modules/secure-identity-access-governance/internal/client/client.go",
        "modules/secure-identity-access-governance/internal/client/socket_owner_unix.go",
        "modules/secure-identity-access-governance/internal/config/trusted.go",
        "modules/secure-identity-access-governance/internal/peerauth/credentials.go",
        "modules/secure-identity-access-governance/internal/peerauth/credentials_darwin.go",
        "modules/secure-identity-access-governance/internal/peerauth/credentials_linux.go",
        "modules/secure-identity-access-governance/internal/peerauth/peerauth.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not provide remote authentication, bearer tokens, OAuth or OIDC, mTLS, a network listener, or authority based on caller category."
      ],
      "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Exact per-TOPS and service identity are established by enrollment before the endpoint can be trusted.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.tops-enrollment",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/secure-identity-access-governance",
      "status": "experimental",
      "title": "Kernel-attested local peer and endpoint trust",
      "what": "Authenticates accepted local callers and the connected SSIAG service endpoint from kernel-attested process credentials rather than caller-supplied identity or socket-path trust.",
      "when": "At server startup, accepted connection setup, and every protected client connection before HTTP request bytes are exchanged.",
      "where": "On supported Darwin and Linux or WSL TOPS nodes over one protected per-TOPS Unix socket.",
      "who": "qxctl, SSIAG local clients, the SSIAG service, protected local consumers, and maintainers.",
      "why": "Path ownership and caller claims are insufficient identity evidence; kernel-attested endpoints prevent confused-deputy and socket-substitution failures."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the SSIAG contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future SSIAG release or derived documentation projection.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns native supervision and socket lifecycle boundaries.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        }
      ],
      "distinctions": [
        {
          "distinction": "Manages SSIAG process liveness only; the STAV sibling supervises a separately starting audit authority.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.native-supervision"
        }
      ],
      "evidence": [
        "Supervision and socket-lock tests cover exact launchd and systemd descriptors, identity binding, atomic replacement, stale-socket proof, and graceful shutdown.",
        "Foundation-lifecycle tests cover offline manager observation, manager-unavailable refusal before attempt creation, exact compare-and-swap, replay, interruption, and recovery."
      ],
      "feature_id": "ssfv:symphony:ssiag-foundation.native-supervision",
      "how": "Observes manager availability and descriptor integrity offline, binds plans and protected digest-linked attempts to exact installation and state evidence, renders identity-bound launchd or systemd descriptors that pin the verified binary, commits descriptor updates atomically, applies bounded restart and shutdown settings, recovers interrupted work by exact compare-and-swap, serializes socket ownership through a persistent lock, and removes only provably stale sockets.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements cgo-free native descriptor management, service identity checks, socket lifecycle locking, stale recovery, and graceful shutdown."
        }
      ],
      "implementation_paths": [
        "modules/secure-identity-access-governance/cmd/symphony-ssiag/main.go",
        "modules/secure-identity-access-governance/internal/foundationlifecycle/engine.go",
        "modules/secure-identity-access-governance/internal/foundationlifecycle/engine_test.go",
        "modules/secure-identity-access-governance/internal/foundationlifecycle/store_unix.go",
        "modules/secure-identity-access-governance/internal/foundationlifecycle/types.go",
        "modules/secure-identity-access-governance/internal/server/socketlock_test.go",
        "modules/secure-identity-access-governance/internal/server/socketlock_unix.go",
        "modules/secure-identity-access-governance/internal/supervision/supervision.go",
        "modules/secure-identity-access-governance/internal/supervision/supervision_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not create system principals, grant policy or provider authority, imply a STAV dependency, use supervisor socket activation, or make supervised mode an authorization credential.",
        "Ordinary foundational lifecycle mutation remains fail-closed until the separately owned STAV receipt endpoint is implemented; audit-deferred use is explicit and remains marked for forward reconciliation."
      ],
      "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "SSIAG and STAV have independent supervision and neither service starts the other.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.native-supervision",
          "type": "distinguished_from"
        }
      ],
      "source_scope": "modules/secure-identity-access-governance",
      "status": "experimental",
      "title": "Per-TOPS native SSIAG supervision",
      "what": "Installs, starts, stops, and removes one bounded per-TOPS native liveness descriptor and protects the service-owned socket lifecycle.",
      "when": "For supported production service lifecycle; direct user serve remains a development or diagnostic route.",
      "where": "In per-user or system launchd or systemd domains and the per-TOPS runtime socket namespace on macOS or Linux.",
      "who": "TOPS owners, target-host administrators, launchd or systemd, owner-provided equivalent supervisors, and maintainers.",
      "why": "Anchors foundational service liveness and restart behavior without making a generic Symphony supervisor part of the authentication bootstrap cycle."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the SSIAG contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future SSIAG release or derived documentation projection.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns protected local policy administration semantics.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns the committed audit receipt required before policy commitment.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Changes protected noncanonical effective policy; authorization capabilities evaluate the current policy and canonical knowledge remains unchanged.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.authorization-capabilities"
        }
      ],
      "evidence": [
        "Policy-administration and server tests cover proposal binding, CAS, durable prepare and audit stages, atomic generation commit, reset, tamper refusal, and exact roll-forward recovery."
      ],
      "feature_id": "ssfv:symphony:ssiag-foundation.policy-administration",
      "how": "Binds caller-neutral host-owner or granted-permission authority to exact current, configuration, and desired digests; persists issued proposals and prepared or audited attempts; requires a committed STAV receipt; atomically commits a generation-counted overlay or reset; swaps the live evaluator; and rolls forward only from exact evidence.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements cgo-free protected proposal, apply, reset, status, durable attempt storage, atomic policy generations, live exchange, and exact recovery."
        }
      ],
      "implementation_paths": [
        "modules/secure-identity-access-governance/internal/policyadmin/manager.go",
        "modules/secure-identity-access-governance/internal/policyadmin/manager_test.go",
        "modules/secure-identity-access-governance/internal/policyadmin/storage_unix.go",
        "modules/secure-identity-access-governance/internal/server/server.go",
        "modules/secure-identity-access-governance/internal/server/server_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not provide audit-deferred recovery, general safeguard administration, remote policy mutation, canonical knowledge apply, or unaudited fallback."
      ],
      "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The current policy determines whether a mapped caller may administer its successor.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.authorization-capabilities",
          "type": "depends_on"
        },
        {
          "rationale": "Every protected policy request requires kernel-attested caller and endpoint identity.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.kernel-peer-trust",
          "type": "depends_on"
        },
        {
          "rationale": "Apply and recovery require committed safe audit evidence before the generation changes.",
          "target_feature_id": "ssfv:symphony:ssiag-foundation.safe-audit-production",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/secure-identity-access-governance",
      "status": "experimental",
      "title": "Durable local SSIAG policy administration",
      "what": "Proposes, applies, resets, inspects, and uniquely recovers protected local policy overlays without rewriting enrolled configuration.",
      "when": "During explicit local administrative policy changes and recovery from an interrupted prepared or audited attempt.",
      "where": "In the private per-TOPS policy state directory and SSIAG policy API under one persistent lock.",
      "who": "Kernel-authenticated target-host owners, subjects with exact policy-administration grants, qxctl, and recovery tooling.",
      "why": "Allows operational permission state to evolve safely while older binaries retain the enrolled deny-by-default posture and interrupted work remains recoverable rather than ambiguous."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the SSIAG contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future SSIAG release or derived documentation projection.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns provider descriptor and operational-provider boundaries.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        }
      ],
      "distinctions": [
        {
          "distinction": "Reports provider descriptors already present in SSIAG configuration; the Swift adapter independently reports static metadata and is not invoked by this registry.",
          "target_feature_id": "ssfv:symphony:ssiag.macos-keychain-metadata"
        }
      ],
      "evidence": [
        "Provider registry and server tests cover deterministic sorting, duplicate refusal, bounded capability copying, and safe descriptor-only responses."
      ],
      "feature_id": "ssfv:symphony:ssiag-foundation.provider-metadata-registry",
      "how": "Validates and sorts configured descriptors, rejects duplicate provider identities, copies bounded capability lists, and exposes metadata without credential values or provider execution.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free configured provider registry and safe descriptor presentation through the local client and server."
        }
      ],
      "implementation_paths": [
        "modules/secure-identity-access-governance/internal/client/client.go",
        "modules/secure-identity-access-governance/internal/provider/registry.go",
        "modules/secure-identity-access-governance/internal/provider/registry_test.go",
        "modules/secure-identity-access-governance/internal/server/server.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not execute an adapter, validate an adapter binary, enable a provider, access a credential, or prove operational readiness."
      ],
      "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "The Go registry and Swift executable expose separate metadata sources and have no implemented invocation bridge.",
          "target_feature_id": "ssfv:symphony:ssiag.macos-keychain-metadata",
          "type": "distinguished_from"
        }
      ],
      "source_scope": "modules/secure-identity-access-governance",
      "status": "experimental",
      "title": "Safe configured provider metadata registry",
      "what": "Reports a deterministic safe inventory of configured provider names, kinds, declared capabilities, exportability, interaction requirements, and disabled or declared state.",
      "when": "During SSIAG status and provider discovery, including when operational adapters are unavailable or disabled.",
      "where": "Inside the Go foundation registry and the local providers endpoint.",
      "who": "SSIAG administrators, qxctl metadata queries, the SSIAG status API, maintainers, and future separately authorized provider bridges.",
      "why": "Administrators and consumers need truthful capability discovery that fails closed without turning provider presence into operational readiness."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the SSIAG contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future SSIAG release or derived documentation projection.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG owns the closed safe security-outcome vocabulary.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        },
        {
          "applicability": "applicable",
          "reason": "STAV owns candidate, producer, receipt, and committed ledger semantics.",
          "reference": "knowledge/stav/SPEC.md",
          "vector": "stav"
        }
      ],
      "distinctions": [
        {
          "distinction": "Constructs and submits a bounded safe candidate; the append authority authenticates the producer, assigns trusted ledger fields, serializes, and persists it.",
          "target_feature_id": "ssfv:symphony:stav-append-authority.serialized-append"
        }
      ],
      "evidence": [
        "Producer and server tests cover the closed event vocabulary, safe-field validation, endpoint authentication, committed receipt binding, and transport or rejection failure."
      ],
      "feature_id": "ssfv:symphony:ssiag-foundation.safe-audit-production",
      "how": "Maps typed event kinds and allowed outcomes to fixed event-class, operation, intent, and reason tuples; accepts only safe actor, target, and configuration references; validates the candidate through the STAV kernel; authenticates the append endpoint; and rejects transport failure, invalid response, or non-commitment.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free closed SSIAG event vocabulary, safe STAV candidate construction, endpoint-authenticated submission, and receipt validation."
        }
      ],
      "implementation_paths": [
        "modules/secure-identity-access-governance/cmd/symphony-ssiag/main.go",
        "modules/secure-identity-access-governance/internal/server/server.go",
        "modules/secure-identity-access-governance/internal/stavproducer/producer.go",
        "modules/secure-identity-access-governance/internal/stavproducer/producer_test.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not write or spool ledger data, accept arbitrary event classes, include secrets, proofs, or provider payloads, or authorize unimplemented provider, credential, or lease operations."
      ],
      "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Only the authenticated append authority can assign trusted ledger fields and issue a committed receipt.",
          "target_feature_id": "ssfv:symphony:stav-append-authority",
          "type": "depends_on"
        },
        {
          "rationale": "Candidate construction and validation use the authority-free canonical STAV protocol kernel.",
          "target_feature_id": "ssfv:symphony:stav-protocol-kernel",
          "type": "depends_on"
        }
      ],
      "source_scope": "modules/secure-identity-access-governance",
      "status": "experimental",
      "title": "Closed safe SSIAG audit production",
      "what": "Converts a closed SSIAG outcome vocabulary into safe STAV candidates and requires a committed receipt before an audit-dependent operation succeeds.",
      "when": "Before releasing an audited decision and before committing an ordinary audited SSIAG policy mutation; other closed kinds become usable only when their owning operations are separately implemented.",
      "where": "In the Go foundation STAV producer package over authenticated local IPC to the same TOPS append authority.",
      "who": "SSIAG authorization and policy operations, future separately enabled provider, credential, or lease operations, the STAV append authority, and maintainers.",
      "why": "Prevents arbitrary security-event classes, secret-bearing audit content, producer-assigned ledger fields, direct ledger mutation, and unaudited success."
    },
    {
      "cross_vector_references": [
        {
          "applicability": "applicable",
          "reason": "SCLV records reviewed SSIAG changes after merge.",
          "reference": "knowledge/sclv/CHANGELOG.md",
          "vector": "sclv"
        },
        {
          "applicability": "applicable",
          "reason": "SKVI routes the SSIAG contracts, feature file, and implementation evidence.",
          "reference": "knowledge/skvi/INDEX.md",
          "vector": "skvi"
        },
        {
          "applicability": "applicable",
          "reason": "SODV governs any future SSIAG release or derived documentation projection.",
          "reference": "knowledge/sodv/SPEC.md",
          "vector": "sodv"
        },
        {
          "applicability": "applicable",
          "reason": "SSIAG vector truth owns TOPS identity, enrollment, and isolation semantics.",
          "reference": "knowledge/ssiag/SPEC.md",
          "vector": "ssiag"
        }
      ],
      "distinctions": [
        {
          "distinction": "Host install and uninstall manage one binary; enrollment establishes one TOPS security identity and preserves its state independently.",
          "target_feature_id": "ssfv:symphony:qxctl.lifecycle-convergence"
        }
      ],
      "evidence": [
        "Lifecycle tests cover receipt-v2 and legacy installation prerequisites, exact user/system identity, reenrollment, preservation, live-socket and held-lock purge refusal, and unsafe-path refusal.",
        "Foundation-lifecycle and package tests cover bounded machine JSON, receipt-last package evidence, compiled version binding, exact CAS, replay, crash recovery, drift refusal, deferred-audit marking, and referenced-package uninstall refusal.",
        "Configuration and path tests cover strict bounded schemas, immutable TOPS IDs, mutable names, and isolated user/system namespaces."
      ],
      "feature_id": "ssfv:symphony:ssiag-foundation.tops-enrollment",
      "how": "Verifies either immutable receipt-v2 evidence for the running libexec entry point or the legacy fixed-path manifest, binds user scope to the enrolling effective identity or requires explicit system service identity, safely creates protected configuration and state, journals digest-linked lifecycle attempts outside purge roots, preserves service identity across reenrollment, and purges only through an explicit native operation after socket and lifecycle-lock proof.",
      "implementation_languages": [
        {
          "language": "Go",
          "role": "Implements the cgo-free enrollment command, strict configuration, protected per-TOPS paths, preservation, and explicit purge behavior."
        }
      ],
      "implementation_paths": [
        "modules/secure-identity-access-governance/cmd/symphony-ssiag/main.go",
        "modules/secure-identity-access-governance/internal/config/config.go",
        "modules/secure-identity-access-governance/internal/foundationlifecycle/codec.go",
        "modules/secure-identity-access-governance/internal/foundationlifecycle/engine.go",
        "modules/secure-identity-access-governance/internal/foundationlifecycle/engine_test.go",
        "modules/secure-identity-access-governance/internal/foundationlifecycle/store_unix.go",
        "modules/secure-identity-access-governance/internal/foundationlifecycle/types.go",
        "modules/secure-identity-access-governance/internal/lifecycle/lifecycle.go",
        "modules/secure-identity-access-governance/internal/lifecycle/lifecycle_test.go",
        "modules/secure-identity-access-governance/internal/packageinstall/package.go",
        "modules/secure-identity-access-governance/internal/packageinstall/package_test.go",
        "modules/secure-identity-access-governance/internal/paths/paths.go"
      ],
      "kind": "subfeature",
      "non_claims": [
        "Does not create a system principal, grant policy permission, start a service, enable a provider, or purge data by default.",
        "The qxctl v1 lifecycle intent does not expose purge, and the implemented typed deferred-audit binding hook is not a completed STAV reconciliation endpoint."
      ],
      "owner_contract": "modules/secure-identity-access-governance/SPEC.md",
      "parent_feature_id": "ssfv:symphony:ssiag-foundation",
      "record_version": 2,
      "relationships": [
        {
          "rationale": "Module enrollment establishes SSIAG's TOPS-local security namespace while qxctl lifecycle convergence coordinates exact installed component state.",
          "target_feature_id": "ssfv:symphony:qxctl.lifecycle-convergence",
          "type": "distinguished_from"
        }
      ],
      "source_scope": "modules/secure-identity-access-governance",
      "status": "experimental",
      "title": "Per-TOPS SSIAG enrollment and isolation",
      "what": "Establishes or removes one SSIAG TOPS enrollment with an immutable TOPS UUID, mutable display name, exact service UID and GID, strict configuration, isolated state, and a preserved-by-default unenrollment path.",
      "when": "After host binary installation and before serving, supervision, policy administration, or provider discovery for that TOPS.",
      "where": "In the selected user or system per-TOPS configuration, state, and runtime namespaces.",
      "who": "TOPS owners, target-host administrators, installation tooling, and maintainers.",
      "why": "Several TOPS instances may share a host, so each security authority needs an independently established identity and storage boundary that cannot be derived from a display name."
    }
  ],
  "source_scope": "modules/secure-identity-access-governance"
}
```
<!-- symphony:ssfv:feature-file:v1:end -->
