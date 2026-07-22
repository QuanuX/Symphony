# Symphony Knowledge Vector Manifest

## Canonical Target

`knowledge/`

## Identity

The Symphony Knowledge Vector (SKV) is the umbrella contract surface for declarative platform knowledge and the common mechanics used by independently installed vector engines.

## Declared Contract Truth Role

The SKV umbrella owns:

- cross-vector source-truth and projection doctrine;
- the common vector-engine process, descriptor, proposal, session-journal, provider-evidence, install-receipt, and docking identifier family;
- the separation among authenticated sessions, worktree reconciliation contexts, proposals, permission-backed ratification, and later apply;
- qxctl cross-vector administration grammar;
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
| SSFV engine, after its separate contract gate | `modules/ssfv-engine/` | `symphony-ssfv` |

These independently installable modules remain in the Symphony monorepo. Source co-location grants no runtime authority or deployment coupling.

Repository-scoped immutable release tags use the owning path followed by the semantic version, for example `modules/skvi-engine/v0.1.0`. The seven cleared tag prefixes are the paths listed above. No Homebrew, Debian/RPM, OCI, Conan, or other external package coordinate is authorized; each registry identity requires a fresh SODV namespace and publication check.

## Language and Process Boundary

Vector engines, the coordinator, and Symphony-authored shared engine mechanics are C++. qxctl remains Go with Cobra/Viper and invokes engines out of process. Protocol v1 is bounded JSON request/response over protected standard input/output. It is not HTTP or OpenAPI, carries no secrets, introduces no C ABI, and uses no cgo.

SSIAG and STAV remain Go under their existing canonical exceptions. A platform-required adapter may use another language only as a separately installed process behind its ratified IPC contract.

## Installability

Every engine and the coordinator is independently buildable, installable, upgradeable, rollbackable, and uninstallable. qxctl is the eventual administrator-facing lifecycle surface. Installation succeeds without Maestro as `installed_undocked`; a compatible version may dock later through an administrator-selected receptor. Multiple compatible versions may coexist without silently changing the active binding.

Symphony is Linux-first. Native Windows engines are not planned. Windows operation uses WSL's Linux execution path or qxctl administration of a remote Symphony node. Existing macOS support is not revoked, but Linux is the engine deployment priority.

## Current Delivery State

The shared C++ foundation, the coordinator's first read-only development slice, and the independently installable SKVI, SCLV, SACV, and SODV engines are implemented at `0.1.0-dev`. SKVI implements bounded structural inspect/check, caller-declared proposal, and disposable projection operations. SCLV implements provider-neutral v1/v2/v3 ledger checking, v3 proposals, non-mutating recovery reconciliation, disposable projection, and local-Git/air-gapped evidence normalization. SACV implements bounded OpenAPI 3.2.0 JSON checks, deterministic compatibility diffs, caller-declared registry proposals, and disposable registry-conformance inventories; YAML fails closed until its parser gate. SODV implements local append-only v1/v2 release-ledger checks, caller-supplied publication observation verification, provider-neutral v2 record proposals, non-mutating interrupted-session recovery, and disposable release inventories without network or publication authority. qxctl invokes each exact version only after validating its inactive undocked receipt and owned files. Session mutation, authentication binding, mutable coordinator journals, locks, observers, coordinator-to-vector invocation, lifecycle administration, programmatic canonical apply, live Maestro docking, external package-manager publication, SSFV engines, and all API endpoints remain unimplemented or gated.

The six exact common v1 schemas under `knowledge/schemas/v1/` govern process requests, process responses, descriptors, install receipts, immutable proposals, and normalized provider evidence. Four SKVI-specific v1 schemas, five SCLV-specific v3 schemas, six SACV-specific v1 schemas, and eight SODV-specific operational schemas govern their vector payloads and results. Installed coordinator, SKVI, SCLV, SACV, and SODV packages report `installed_undocked`, create no active alias, and declare no default receptor until a receptor contract is separately selected.

## Non-Authorization Statement

This manifest does not authorize an engine to rewrite canonical files, manufacture ratification, classify callers, hold credentials, edit STAV ledgers, publish documentation or releases, expose network listeners, enter hot/warm execution, or implement semantics not assigned by its vector Contract Quad.
