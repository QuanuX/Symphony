# Symphony Knowledge Vector Manifest

## Canonical Target

`knowledge/`

## Identity

The Symphony Knowledge Vector (SKV) is the umbrella contract surface for declarative platform knowledge and the common mechanics used by independently installed vector engines.

## Declared Contract Truth Role

The SKV umbrella owns:

- cross-vector source-truth and projection doctrine;
- the common vector-engine process, descriptor, proposal, reconciliation/session journal, explicit session-transition result, provider-evidence, install-receipt, engine-binding-registry, desired-state/first-boot lifecycle, and docking identifier family;
- the separation among authenticated sessions, worktree reconciliation contexts, proposals, permission-backed ratification, and later apply;
- qxctl cross-vector administration grammar and the topology in `knowledge/LIFECYCLE.md`;
- common temporal semantics and representation profiles in `knowledge/TIME.md`;
- common installability, dependency, path-safety, and thermal-isolation requirements.

Each vector Contract Quad owns its domain semantics, canonical paths, operations, machine-managed boundaries, and projection eligibility. The umbrella cannot invent a vector-specific fact for tooling convenience.

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

These independently installable modules remain in the Symphony monorepo. Source co-location grants no runtime authority or deployment coupling.

Repository-scoped immutable release tags use the owning path followed by the semantic version, for example `modules/skvi-engine/v0.1.0`. The seven cleared tag prefixes are the paths listed above. No Homebrew, Debian/RPM, OCI, Conan, or other external package coordinate is authorized; each registry identity requires a fresh SODV namespace and publication check.

## Language and Process Boundary

Vector engines, the coordinator, and Symphony-authored shared engine mechanics are C++. qxctl remains Go with Cobra/Viper and invokes engines out of process. Protocol v1 is bounded JSON request/response over protected standard input/output. It is not HTTP or OpenAPI, carries no secrets, introduces no C ABI, and uses no cgo. STSC is a common contract implemented by the shared C++ mechanics and equivalent standard-library behavior in Go; it is not an independently installed engine.

SSIAG and STAV remain Go under their existing canonical exceptions. A platform-required adapter may use another language only as a separately installed process behind its ratified IPC contract.

## Installability

Every engine and the coordinator is independently buildable, installable, upgradeable, rollbackable, and uninstallable. qxctl is the eventual administrator-facing lifecycle surface. Installation succeeds without Maestro as `installed_undocked`; a compatible version may dock later through an administrator-selected receptor. Multiple compatible versions may coexist without silently changing the active binding.

Symphony is Linux-first. Native Windows engines are not planned. Windows operation uses WSL's Linux execution path or qxctl administration of a remote Symphony node. Existing macOS support is not revoked, but Linux is the engine deployment priority.

## Current Delivery State

The shared C++ foundation, the coordinator's user-scope reconciliation, authenticated-session, and report-only lifecycle-planning slices, and the independently installable SKVI, SCLV, SACV, SODV, and SSFV engines are implemented at `0.1.0-dev`. The coordinator provides compatibility negotiation and content-addressed begin/status/checkpoint/close/recover over separate durable per-worktree reconciliation journals and per-TOPS/subject/repository authority-epoch journals. Its report-only lifecycle operation validates complete caller-supplied desired/observed evidence and emits a deterministic forward/inverse dependency-ready-set plan with exact target/receptor identities, isolated cycles/blockers, and disabled apply. qxctl authenticates the local SSIAG endpoint, requests exact caller-neutral authorization decisions, maintains protected per-TOPS desired profiles with compare-and-swap and durable replacement, observes only fixed receipt layouts under configured roots, preserves unsupported packages as unknown evidence, and invokes one exact bound coordinator for a fresh dynamic report. Collection time changes the observation document digest but is excluded from the stable inventory key, so a timestamp-only refresh cannot restart a transaction or renumber semantic actions. SSIAG derives the subject from kernel peer credentials, evaluates explicit exact grants with a deny default, and returns short-lived non-transferable capability evidence only after a committed STAV append. This evidence authorizes only its exact protected noncanonical operation; it is neither a bearer credential nor canonical apply authority. SKVI implements bounded structural inspect/check, caller-declared proposal, and disposable projection operations. SCLV implements provider-neutral v1/v2/v3 ledger checking, v3 proposals, non-mutating recovery reconciliation, disposable projection, and local-Git/air-gapped evidence normalization. SACV implements bounded OpenAPI 3.2.0 JSON checks, deterministic compatibility diffs, caller-declared registry proposals, and disposable registry-conformance inventories; YAML fails closed until its parser gate. SODV implements local append-only v1/v2 release-ledger checks, caller-supplied external-state verification, provider-neutral v2 release-record proposals, non-mutating interrupted-session recovery, and disposable release inventories without network or publication authority. SSFV implements structural checks, content-addressed semantic snapshots and freshness modes, baseline-versus-live diffs, caller-declared multi-file proposals, and disposable semantic graphs without feature-worthiness or mutation authority. Its first partial bootstrap records exactly the platform capability, shared engine foundation, and coordinator foundation; it does not establish repository-wide catalog completeness. qxctl invokes each implemented engine version only after validating its inactive undocked receipt and owned files. Durable lifecycle boot journals, action execution, installation/uninstall, activation, coordinator-to-vector invocation, repository/system/TOPS engine-binding profiles, safeguard administration, programmatic canonical apply, live Maestro docking, external package-manager publication, additional SSFV feature records, and SACV-governed HTTP API documents remain unimplemented or separately gated.

The twenty-six exact common v1 schemas under `knowledge/schemas/v1/` govern process requests, process responses, descriptors, install receipts, user-default bindings, reconciliation and authenticated-session commands/state/results, explicit session-transition results, immutable proposals, normalized provider evidence, protected lifecycle profile input/state, the desired/observed/plan-command/plan/applied/boot-journal lifecycle family, and reusable temporal encodings. The exact common receipt v2 schema under `knowledge/schemas/v2/` provides immutable content-addressed package ownership, entry-point, capability, receptor, and platform evidence without mutable activation or docking state. Its dynamic plan contract uses a dependency-ready-set scheduler with forward/inverse action relationships while preserving ordered safety phases. qxctl implements profile persistence, configured-root observation, and report invocation; the coordinator implements deterministic planning over the supplied evidence. Lifecycle boot-journal persistence, recovery, apply, and Maestro docking remain unimplemented. Three SSIAG-specific authorization v1 schemas govern requests, decisions, and non-transferable capabilities. Four SKVI-specific v1 schemas, five SCLV-specific v3 schemas, six SACV-specific v1 schemas, eight SODV-specific operational schemas, and eighteen SSFV-specific v1/v2 schemas govern their vector payloads and results. Installed coordinator, SKVI, SCLV, SACV, SODV, and SSFV packages report `installed_undocked`, create no active alias, and declare no default receptor until a receptor contract is separately selected. A qxctl binding, reconciliation context, authenticated-session journal, or lifecycle schema does not alter those receipts.

## Non-Authorization Statement

This manifest does not authorize an engine to rewrite canonical files, manufacture ratification, classify callers, hold credentials, edit STAV ledgers, publish documentation or releases, expose network listeners, enter hot/warm execution, or implement semantics not assigned by its vector Contract Quad.
