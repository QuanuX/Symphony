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

The shared C++ foundation, the coordinator's user-scope reconciliation, authenticated-session, report-only lifecycle-planning, and durable report-only lifecycle-journal slices, and the independently installable SKVI, SCLV, SACV, SODV, and SSFV engines are implemented at `0.1.0-dev`. The coordinator provides separate dual-slot durability and evidence-based recovery for worktree reconciliation, authority epochs, and per-TOPS/profile lifecycle boot streams. Its lifecycle planner emits deterministic forward/inverse dependency-ready-set plans with exact target/receptor identities and disabled apply. Its boot operation independently validates SSIAG evidence and stable inventory, persists protected noncanonical plan/checkpoint identity, treats timestamp-only rescans as no-ops, and advances or repairs only exact digest-linked state. qxctl authenticates local SSIAG, maintains protected desired profiles, observes fixed receipt layouts, revalidates one exact bound coordinator, and administers report, boot, status, and recovery with strict result validation.

SSIAG derives subjects from kernel peer credentials, evaluates explicit exact grants with a deny default, and returns short-lived non-transferable evidence only after a committed STAV append. The existing vector engines retain their bounded check/proposal/projection scopes, and SSFV remains an explicitly partial catalog. Lifecycle action execution, applied-state persistence, host boot-hook installation, installation/uninstall, activation, coordinator-to-vector invocation, repository/system/TOPS engine-binding profiles, safeguard administration, programmatic canonical apply, live Maestro docking, external package-manager publication, additional SSFV feature records, and SACV-governed HTTP API documents remain unimplemented or separately gated.

The twenty-eight exact common v1 schemas under `knowledge/schemas/v1/` govern process requests/responses, descriptors, receipts, user-default bindings, reconciliation and authenticated-session commands/state/results, explicit session-transition results, proposals, provider evidence, protected lifecycle profiles, the desired/observed/plan/applied/boot command/result/journal family, and temporal encodings. The exact common receipt v2 schema provides immutable content-addressed package ownership and compatibility evidence. Lifecycle boot-journal persistence and bounded recovery are implemented; action execution, applied-state persistence, apply, and Maestro docking remain unavailable. Three SSIAG authorization schemas and each vector's own operation schemas remain owned by their applicable Contract Quad. Installed coordinator and vector-engine packages remain `installed_undocked`; qxctl bindings and protected journals do not alter those receipts.

## Non-Authorization Statement

This manifest does not authorize an engine to rewrite canonical files, manufacture ratification, classify callers, hold credentials, edit STAV ledgers, publish documentation or releases, expose network listeners, enter hot/warm execution, or implement semantics not assigned by its vector Contract Quad.
