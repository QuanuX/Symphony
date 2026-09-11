# Symphony Knowledge Vector Manifest

## Canonical Target

`knowledge/`

## Identity

The Symphony Knowledge Vector (SKV) is the umbrella contract surface for declarative platform knowledge and the common mechanics used by independently installed vector engines.

## Declared Contract Truth Role

The SKV umbrella owns:

- cross-vector source-truth and projection doctrine;
- the common vector-engine process, descriptor, proposal, reconciliation/session/SSFV-maintenance journal, explicit session-transition result, provider-evidence, install-receipt, engine-binding-registry, desired-state/first-boot lifecycle, shared-root ownership/reclamation/compatibility-fence, and docking/inventory identifier family;
- the separation among authenticated sessions, worktree reconciliation contexts, proposals, permission-backed ratification, and later apply;
- qxctl cross-vector administration grammar and the topology in `knowledge/LIFECYCLE.md`;
- the exact foundational SSIAG/STAV enrollment and native-supervision envelope in `knowledge/FOUNDATIONAL-LIFECYCLE.md`;
- lowest-authoritative-layer invariant ownership and regression routing in `knowledge/INVARIANTS.md` and `knowledge/INVARIANT-OWNERSHIP.json`;
- common temporal semantics and representation profiles in `knowledge/TIME.md`;
- common deterministic validation evidence, protected warning-policy, and baseline semantics in `knowledge/VALIDATION.md`;
- owner-routed platform nomenclature in `knowledge/SLANG.md` and universal identity-family delegation in `knowledge/NAMESPACES.md`;
- common installability, dependency, path-safety, and thermal-isolation requirements.

Each vector Contract Quad owns its domain semantics, canonical paths, operations, machine-managed boundaries, and projection eligibility. The umbrella cannot invent a vector-specific fact for tooling convenience.

`knowledge/SLANG.md` and `knowledge/NAMESPACES.md` are canonical companions governed by this Contract Quad, not additional Quad members. SLANG routes terms to semantic owners without duplicating their rules. NAMESPACES registers and delegates identity families without replacing family-owned grammar, allocation, lifecycle, or compatibility.

## Canonical Surfaces

- `knowledge/ACCORD-AUDIT.md`
- `knowledge/ARCHITECTURE.md`
- `knowledge/FEATURE-ADMINISTRATION-PROFILE.json`
- `knowledge/FEATURE-ADMINISTRATION-PROFILE.md`
- `knowledge/FEATURE-ADMINISTRATION.md`
- `knowledge/FOUNDATIONAL-LIFECYCLE.md`
- `knowledge/INTENT.md`
- `knowledge/INVARIANT-OWNERSHIP.json`
- `knowledge/INVARIANTS.md`
- `knowledge/LIFECYCLE.md`
- `knowledge/MANIFEST.md`
- `knowledge/NAMESPACES.md`
- `knowledge/SKILL.md`
- `knowledge/SLANG.md`
- `knowledge/SPEC.md`
- `knowledge/TIME.md`
- `knowledge/VALIDATION.md`

## Subordinate Manifests

- `knowledge/sacv/MANIFEST.md`
- `knowledge/sacv/schemas/v1/MANIFEST.md`
- `knowledge/sav/MANIFEST.md`
- `knowledge/sav/schemas/v1/MANIFEST.md`
- `knowledge/schemas/v1/MANIFEST.md`
- `knowledge/schemas/v2/MANIFEST.md`
- `knowledge/sclv/MANIFEST.md`
- `knowledge/sclv/schemas/v3/MANIFEST.md`
- `knowledge/scv/MANIFEST.md`
- `knowledge/scv/fixtures/v1/MANIFEST.md`
- `knowledge/scv/scev/MANIFEST.md`
- `knowledge/scv/scev/cf/MANIFEST.md`
- `knowledge/scv/schemas/v1/MANIFEST.md`
- `knowledge/scv/schv/MANIFEST.md`
- `knowledge/scv/schv/aws/MANIFEST.md`
- `knowledge/scv/schv/azure/MANIFEST.md`
- `knowledge/scv/schv/do/MANIFEST.md`
- `knowledge/scv/schv/gcp/MANIFEST.md`
- `knowledge/sev/MANIFEST.md`
- `knowledge/sev/schemas/v1/MANIFEST.md`
- `knowledge/shv/MANIFEST.md`
- `knowledge/siv/MANIFEST.md`
- `knowledge/siv/smcv/MANIFEST.md`
- `knowledge/skvi/MANIFEST.md`
- `knowledge/skvi/schemas/v1/MANIFEST.md`
- `knowledge/snv/MANIFEST.md`
- `knowledge/snv/sciv/MANIFEST.md`
- `knowledge/snv/scnv/MANIFEST.md`
- `knowledge/snv/sniv/MANIFEST.md`
- `knowledge/snv/snrv/MANIFEST.md`
- `knowledge/sodv/MANIFEST.md`
- `knowledge/sodv/schemas/v1/MANIFEST.md`
- `knowledge/sov/MANIFEST.md`
- `knowledge/sqv/MANIFEST.md`
- `knowledge/sqv/soov/MANIFEST.md`
- `knowledge/ssfv/MANIFEST.md`
- `knowledge/ssfv/schemas/v1/MANIFEST.md`
- `knowledge/ssfv/schemas/v2/MANIFEST.md`
- `knowledge/ssiag/MANIFEST.md`
- `knowledge/ssiag/schemas/v1/MANIFEST.md`
- `knowledge/stav/MANIFEST.md`
- `knowledge/stav/fixtures/v1/MANIFEST.md`
- `knowledge/stav/schemas/v1/MANIFEST.md`
- `libraries/knowledge-vector-engine-cpp/MANIFEST.md`
- `libraries/stav-protocol-go/MANIFEST.md`
- `modules/accordare-stav-producer/MANIFEST.md`
- `modules/hotpath-runtime/MANIFEST.md`
- `modules/knowledge-session-coordinator/MANIFEST.md`
- `modules/maestro/MANIFEST.md`
- `modules/sacv-engine/MANIFEST.md`
- `modules/sav-engine/MANIFEST.md`
- `modules/scev-cf-engine/MANIFEST.md`
- `modules/scev-engine/MANIFEST.md`
- `modules/schv-aws-engine/MANIFEST.md`
- `modules/schv-azure-engine/MANIFEST.md`
- `modules/schv-do-engine/MANIFEST.md`
- `modules/schv-engine/MANIFEST.md`
- `modules/schv-gcp-engine/MANIFEST.md`
- `modules/sclv-engine/MANIFEST.md`
- `modules/scv-engine/MANIFEST.md`
- `modules/secure-identity-access-governance/MANIFEST.md`
- `modules/sev-engine/MANIFEST.md`
- `modules/skvi-engine/MANIFEST.md`
- `modules/sodv-engine/MANIFEST.md`
- `modules/ssfv-engine/MANIFEST.md`
- `modules/ssiag-provider-macos-keychain/MANIFEST.md`
- `modules/stav-append-authority/MANIFEST.md`
- `tools/qxctl/MANIFEST.md`
- `tools/symphony-validator/MANIFEST.md`

These exact declarations are the machine-discoverable canonical-surface closure. A file, directory, implementation marker, or indexed implementation surface does not become part of that closure by existing. Adding or removing an owner requires an explicit update to this subordinate-manifest list; discovery never scans for owners.

## Cleared Implementation Namespace

| Role | Candidate path | Executable |
|---|---|---|
| authority-free shared C++ mechanics | `libraries/knowledge-vector-engine-cpp/` | none |
| authenticated-session and worktree-reconciliation coordinator | `modules/knowledge-session-coordinator/` | `symphony-knowledge-session` |
| SKVI engine | `modules/skvi-engine/` | `symphony-skvi` |
| SCLV engine | `modules/sclv-engine/` | `symphony-sclv` |
| SACV engine | `modules/sacv-engine/` | `symphony-sacv` |
| SODV engine | `modules/sodv-engine/` | `symphony-sodv` |
| SSFV engine | `modules/ssfv-engine/` | `symphony-ssfv` |
| SAV engine | `modules/sav-engine/` | `symphony-sav` |
| SEV engine | `modules/sev-engine/` | `symphony-sev` |
| Maestro receptor presence authority | `modules/maestro/` | `symphony-maestro` |
| SCV source-knowledge engine | `modules/scv-engine/` | `symphony-scv` |
| SCHV source-knowledge engine | `modules/schv-engine/` | `symphony-schv` |
| SCEV source-knowledge engine | `modules/scev-engine/` | `symphony-scev` |
| SCHV-AWS source-knowledge engine | `modules/schv-aws-engine/` | `symphony-schv-aws` |
| SCHV-AZURE source-knowledge engine | `modules/schv-azure-engine/` | `symphony-schv-azure` |
| SCHV-DO source-knowledge engine | `modules/schv-do-engine/` | `symphony-schv-do` |
| SCHV-GCP source-knowledge engine | `modules/schv-gcp-engine/` | `symphony-schv-gcp` |
| SCEV-CF source-knowledge engine | `modules/scev-cf-engine/` | `symphony-scev-cf` |

These independently installable modules remain in the Symphony monorepo. Source co-location grants no runtime authority or deployment coupling.

Repository-scoped immutable release tags use the owning path followed by the semantic version, for example `modules/skvi-engine/v0.1.0`. The cleared tag prefixes are the paths listed above. No Homebrew, Debian/RPM, OCI, Conan, or other external package coordinate is authorized; each registry identity requires a fresh SODV namespace and publication check.

## Language and Process Boundary

Vector engines, the coordinator, and Symphony-authored shared engine mechanics are C++. qxctl remains Go with Cobra/Viper and invokes engines out of process. Protocol v1 is bounded JSON request/response over protected standard input/output. It is not HTTP or OpenAPI, carries no secrets, introduces no C ABI, and uses no cgo. STSC is a common contract implemented by the shared C++ mechanics and equivalent standard-library behavior in Go; it is not an independently installed engine.

SSIAG and STAV remain Go under their existing canonical exceptions. A platform-required adapter may use another language only as a separately installed process behind its ratified IPC contract.

## Installability

Every engine, the coordinator, and the validator are independently buildable, installable, upgradeable, rollbackable, and uninstallable. qxctl is the administrator-facing lifecycle and validation surface. Installation succeeds without Maestro as `installed_undocked`; a compatible engine version may dock later through an administrator-selected receptor. Multiple compatible versions may coexist without silently changing the active binding.

Symphony is Linux-first. Native Windows engines are not planned. Windows operation uses WSL's Linux execution path or qxctl administration of a remote Symphony node. Existing macOS support is not revoked, but Linux is the engine deployment priority.

## Current Delivery State

The shared C++ foundation, the coordinator's user-scope reconciliation, authenticated-session, persistent SSFV-maintenance, report-only lifecycle-planning/journal, and separately authorized apply-capable lifecycle slices, and the independently installable SKVI, SCLV, SACV, SODV, and SSFV engines are implemented at `0.1.0-dev`. The coordinator provides separate dual-slot durability and evidence-based recovery for worktree reconciliation, authority epochs, SSFV semantic baselines, report-only lifecycle streams, and apply-capable lifecycle streams. Its planner emits deterministic forward/inverse dependency-ready-set plans with exact target/receptor identities. Its apply operations bind one exact report source, serialize attempts before host mutation, verify complete after-observation, and select content-addressed applied evidence through the v2 journal/head. qxctl authenticates local SSIAG, maintains protected desired/runtime profiles, observes fixed receipt layouts, revalidates exact bound coordinator/SSFV installations, administers SSFV session maintenance, and validates optional complete Maestro inventory evidence.

SSIAG derives subjects from kernel peer credentials, evaluates explicit exact grants with a deny default, returns short-lived non-transferable evidence only after a committed STAV append, verifies one exact receipt-bound macOS provider through a mutually authenticated metadata-only handshake, and now inventories exact compatible provider installations plus administers one protected active/previous binding per TOPS. Binding plans and mutation use expected-state compare-and-swap, candidate re-verification, a distinct safe STAV event before commit, and bounded forward or reverse recovery; the installed-foundation offline recovery route reuses the same SSIAG owner and requires the live socket to be absent. Operational Keychain access and secret delivery remain disabled. The existing vector engines retain their bounded check/proposal/projection scopes, and SSFV remains an explicitly partial 100-record catalog with top-level owner-scope coverage, seventy-two ratified nested features, reviewed non-feature dispositions, and incomplete remaining nested review. Accordare now provides durable pre-mutation intent, typed terminal outcomes, separate intent/candidate recovery, installed producer-to-STAV acceptance, and native per-TOPS supervision. The current invariant-assurance slice reports sixteen incremental invariant records and the 218-leaf stable qxctl command registry; it neither transfers invariant ownership nor proves complete legacy or installed-host coverage. Generic staged receipt-v2 installation/uninstall, root-local multi-profile ownership/reclamation, protected selection/activation, and Maestro docking-presence changes are implemented only within explicit lifecycle administration/apply. Protected validator warning profiles, baselines, and subject-aware lifecycle state are implemented separately and never change detection; deterministic root-summary projection/checking remains read-only. Receipt-v1 mutation, downloads, arbitrary entry-point execution, general live service activation, Maestro engine invocation/supervision, login/session hook installation, native Windows host integration, hidden watchers, coordinator-to-vector invocation, repository/system/TOPS engine-binding profiles, broader safeguard administration, programmatic canonical apply, external package-manager publication, unratified SSFV feature records, and SACV-governed HTTP API documents remain unimplemented or separately gated.

The sixty-six exact common v1 schemas under `knowledge/schemas/v1/` govern process requests/responses, descriptors, receipts, bindings, reconciliation/session/SSFV-maintenance/generic-lifecycle/foundational-lifecycle/shared-root/Maestro operations, feature-administration profiles and coverage, invariant ownership and query results, qxctl command registries, temporal encodings, and deterministic validation results/policies/baselines/warning state. Six common v2 schemas provide side-by-side stable engine-operation descriptors, bounded extensible engine-binding selection, immutable content-addressed package ownership, apply-journal/head truth, and the separately versioned generic engine-adapter invariant registry. Report/apply/SSFV-maintenance journal persistence, bounded recovery, reviewed generic local actions, shared-root claim/reclamation safety and old-client fencing, applied-state commitment, durable Maestro presence and derived inventory, exact validator packaging, qxctl warning controls, and module-owned SSIAG/STAV foundational lifecycle transactions are implemented. Canonical knowledge apply, arbitrary live-service activation, and Maestro engine execution remain unavailable. SSIAG authorization schemas and each vector's own operation schemas remain owned by their applicable Contract Quad.

## SCV Source-Knowledge Increment

The eight domain engines listed above add bounded source planning, pure expected-state transition validation, UTF-8 capture import, explicit claim interpretation and reproducible graph queries. qxctl supplies a separately governed protected source-store adapter. A computed transition is not a durable or authorized commit. The additive `0.2.0-dev` corpus profile in `knowledge/scv/CORPUS.md` supplies immutable bounded evidence retention and explicit snapshot selection for queries without a selected corpus head. The additive `0.3.0-dev` profile in `knowledge/scv/INTERPRETATION.md` adds retained authored mappings and scoped connection evaluation/reassessment. Exact supported operations and remaining broad acquisition, semantic interpretation and operational gaps are declared in the owner and module contracts.

The `0.4.0-dev` agent operating profile in `knowledge/scv/AGENT-WORKFLOWS.md` adds native profile preparation, exact packaged schema discovery, structured SCV CLI failures, separate artifact provenance records and recoverable runs. It retains the existing owner, authorization and caller-composition boundaries.

The `0.5.0-dev` provider coverage profile in `knowledge/scv/COVERAGE.md` adds native inventory, selected corpus and replayed interpretation accounting through qxctl while preserving source uncertainty and caller authority.

## Non-Authorization Statement

This manifest does not authorize an engine to rewrite canonical files, manufacture ratification, classify callers, hold credentials, edit STAV ledgers, publish documentation or releases, expose network listeners, enter hot/warm execution, or implement semantics not assigned by its vector Contract Quad.

The SCV `.6` increment additionally admits the interface declaration and portable provider/composition companions recorded in `knowledge/scv/MANIFEST.md` and its schema manifest. Each of eight exact C++ packages exposes 26 operations; qxctl has 244 registered leaves, including 46 SCV leaves. This bounded increment extends the existing 21 invariant records and 11 adapter identities and retains caller authority over requirements, evidence policy, providers and permitted recipes.
