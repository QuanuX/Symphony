# Symphony Semantic Feature Vector Coverage

## Status

Architect-ratified SSFV owner-scope coverage inventory. Coverage remains `partial` while nested subfeature and microfeature adjudication is incomplete.

## Purpose

Define the current repository source universe, explicit exclusions, evidence boundary, freshness rule, and the conditions that must be satisfied before SSFV may report `complete`.

## Source Universe

The v1 owner-scope universe is the repository root plus every tracked immediate child of `libraries/`, `modules/`, and `tools/` that has both an owning contract and implemented or experimental application behavior. An owner scope may contain several semantic records, but one stable feature identity has exactly one canonical owner file.

The inventory is intentionally explicit. Directory discovery, file count, language, code volume, and naming do not establish feature-worthiness. A new or removed owner scope changes this contract through ordinary owner-ratified review; tooling must never silently expand the universe.

## Canonical Owner-Scope Inventory

| Source scope | Disposition | Canonical feature identity | Evidence |
|---|---|---|---|
| `.` | registered | `ssfv:symphony:platform` | `FEATURES.md` |
| `libraries/knowledge-vector-engine-cpp` | registered | `ssfv:symphony:knowledge-vector-engine-foundation` | `libraries/knowledge-vector-engine-cpp/FEATURES.md` |
| `libraries/stav-protocol-go` | registered | `ssfv:symphony:stav-protocol-kernel` | `libraries/stav-protocol-go/FEATURES.md` |
| `modules/bus-troll` | excluded | none | Proposal-only Contract Quad seed; no executable implementation exists. |
| `modules/hotpath-runtime` | excluded | none | Proposal-only Contract Quad seed; no executable implementation exists. |
| `modules/knowledge-session-coordinator` | registered | `ssfv:symphony:knowledge-session-coordinator` | `modules/knowledge-session-coordinator/FEATURES.md` |
| `modules/maestro` | registered | `ssfv:symphony:maestro-presence-authority` | `modules/maestro/FEATURES.md` |
| `modules/node-troll` | excluded | none | Proposal-only Contract Quad seed; no executable implementation exists. |
| `modules/sacv-engine` | registered | `ssfv:symphony:sacv-engine` | `modules/sacv-engine/FEATURES.md` |
| `modules/sav-engine` | registered | `ssfv:symphony:sav-engine` | `modules/sav-engine/FEATURES.md` |
| `modules/sclv-engine` | registered | `ssfv:symphony:sclv-engine` | `modules/sclv-engine/FEATURES.md` |
| `modules/secure-identity-access-governance` | registered | `ssfv:symphony:ssiag-foundation` | `modules/secure-identity-access-governance/FEATURES.md` |
| `modules/sev-engine` | registered | `ssfv:symphony:sev-engine` | `modules/sev-engine/FEATURES.md` |
| `modules/skvi-engine` | registered | `ssfv:symphony:skvi-engine` | `modules/skvi-engine/FEATURES.md` |
| `modules/sodv-engine` | registered | `ssfv:symphony:sodv-engine` | `modules/sodv-engine/FEATURES.md` |
| `modules/ssfv-engine` | registered | `ssfv:symphony:ssfv-engine` | `modules/ssfv-engine/FEATURES.md` |
| `modules/ssiag-provider-macos-keychain` | registered | `ssfv:symphony:ssiag.macos-keychain-metadata` | `modules/ssiag-provider-macos-keychain/FEATURES.md` |
| `modules/stav-append-authority` | registered | `ssfv:symphony:stav-append-authority` | `modules/stav-append-authority/FEATURES.md` |
| `tools/qxctl` | registered | `ssfv:symphony:qxctl` | `tools/qxctl/FEATURES.md` |
| `tools/symphony-validator` | registered | `ssfv:symphony:symphony-validator` | `tools/symphony-validator/FEATURES.md` |

## Ratified Nested Review Progress

The F1 through F3 review, feature-administration assurance slice, root-summary assurance slice, invariant-assurance slice, provider-trust assurance slice, provider-binding lifecycle slice, and Accordare durability review record sixty-eight implemented `subfeature` boundaries inside fifteen already registered owner scopes:

| Owner scope | Ratified nested records | Review disposition |
|---|---:|---|
| `tools/qxctl` | 10 | Exact engine bindings, authenticated sessions, lifecycle convergence, Linux report-only host receptor, SSIAG administration, STAV administration, Maestro administration, governed validation, the stable command registry, and invariant assurance are registered. |
| `modules/knowledge-session-coordinator` | 6 | Reconciliation, authority epochs, semantic maintenance, lifecycle planning, lifecycle apply coordination, and protected Named Version durability are registered. |
| `modules/maestro` | 1 | Complete derived receptor inventory is registered separately from durable receptor presence. |
| `libraries/stav-protocol-go` | 2 | Canonical bytes, digests, and bounded local frames are separated from exact content and identifier validation; durable checksummed ledger framing is explicitly owned by the append authority. |
| `modules/secure-identity-access-governance` | 9 | TOPS enrollment, kernel peer trust, authorization capabilities, policy administration, provider metadata, provider trust assurance, exact provider-installation and protected binding lifecycle, safe audit production, and native supervision are registered. |
| `modules/stav-append-authority` | 5 | TOPS enrollment, serialized append, ledger durability, authorized query, and native supervision are registered. |
| `libraries/knowledge-vector-engine-cpp` | 3 | Bounded process protocol, content-addressed evidence snapshots, and temporal representation conformance are registered without turning the static foundation into a runtime engine. |
| `modules/skvi-engine` | 3 | Structural index assurance, content-addressed change proposals, and disposable structural projection are registered. |
| `modules/sclv-engine` | 5 | Append-only ledger assurance, provider-neutral evidence normalization, evidence-bound append proposals, forward-only closure recovery, and disposable provider-neutral history are registered. |
| `modules/sacv-engine` | 4 | API-contract conformance, OpenAPI compatibility evidence, contract-registration proposals, and rebuildable contract inventory are registered. |
| `modules/sav-engine` | 4 | CURRENT/accord evaluation, immutable Named Versions, Extension Capsule admission, and two-way Installation Blueprint planning are registered. |
| `modules/sodv-engine` | 5 | Release-ledger validation, observed-publication verification, forward release-record proposals, interrupted-publication reconciliation, and rebuildable release-transaction projection are registered. |
| `modules/ssfv-engine` | 5 | Catalog-integrity snapshots, semantic-freshness comparison, catalog-change proposals, portable semantic-graph projection, and engine-first administration assurance are registered. |
| `modules/sev-engine` | 4 | Dynamic evolution, SCSEV command-surface assessment, private novelty/watch policy, and shared lifecycle-session binding are registered. |
| `tools/symphony-validator` | 2 | Deterministic root-summary projection plus invariant-ownership and implemented-module-admission assurance are registered separately from the complete repository checker. |

The macOS Keychain metadata adapter remains one narrow registered subfeature with no child record. Its corrected record states that the Go SSIAG foundation invokes only the mutually verified metadata handshake, qxctl never invokes the Swift executable directly, and operational Keychain access, credential operations, and secret delivery remain disabled.

These reviews advance completion condition 2 for those exact boundaries only. They do not imply that every other nested application boundary has been adjudicated, that an unlisted boundary has an explicit non-feature disposition, that all legacy invariants are registered, or that every package installed on a host has been inventoried.

## F2 Explicit Non-Feature Dispositions

The F2 security and audit review keeps these boundaries as implementation evidence rather than independent semantic records:

- host binary copy or removal and install manifests;
- command flag wiring that only exposes a recorded capability;
- build-tag credential and filesystem shims implementing one recorded Darwin or Linux behavior;
- safe path, digest, JSON, error, copy, and atomic-file helpers without an independently observable purpose;
- persistent socket locks and stale-socket recovery as durability qualities of native supervision;
- the STAV client package as shared transport evidence rather than a second authority;
- protocol constants and model structs as evidence for canonical-wire and semantic-validation behavior;
- tests, fixtures, and Swift lifecycle or protocol helpers as evidence or internals rather than separate application capabilities;
- operational Keychain access, secret delivery, credential or lease operations, safeguards, audit-deferred recovery, STAV signed checkpoints or non-repudiation, remote export, automatic rotation or retention, and general repair because those behaviors remain disabled, deferred, or unimplemented.

## F3 Explicit Non-Feature Dispositions

The F3 shared-foundation and knowledge-engine review keeps these boundaries as implementation evidence or facets of the registered records rather than independent semantic records:

- foundation SHA-256 rounds, JSON aliases, stable-error types, limit constants, file-descriptor RAII, string sanitization, vendored dependencies, and individual conformance cases;
- engine `inspect`, descriptors, main-process wrappers, diagnostic flags, error emission, exit-status mapping, and operation dispatch;
- Markdown parsers, field detectors, trimming, scalar/list rendering, digest helpers, counters, individual finding codes, and per-rule validation branches;
- CMake targets, receipt generation, install/uninstall mechanics, licenses, tests, and fixtures as packaging or evidence rather than new semantic purposes;
- individual SKVI add/replace/remove choices, SCLV recovery outcomes, SODV observation states, or freshness modes when those choices are bounded variants of one registered capability;
- local-Git and air-gap adapters as separate records because their feature-worthy outcome is the shared provider-neutral evidence-normalization boundary;
- schemas, templates, legacy-normalization branches, compatibility-model helpers, and projection encoders as contract or implementation evidence for the registered engine behaviors;
- speculative JSONL, search-index, graph-database, endpoint, publication, provider-network, persistent-projection, canonical-apply, or semantic-decision behavior that is not implemented.

## Exclusion Rules

The following do not independently enter the owner-scope universe:

- canonical `knowledge/` contract directories whose application behavior is owned by a separately registered engine or tool;
- build output, generated output, vendored dependencies, test fixtures, scratch state, caches, and external projections;
- a directory, language, source file, symbol, schema, or documentation page that does not pass the five-part feature-worthiness gate;
- proposal-only module seeds without executable implementation;
- nested directories already represented by an owner record until their behavior is separately reviewed as a feature, subfeature, or microfeature.

Exclusion is not a claim that a scope can never become feature-worthy. It records the current reviewed disposition.

## Freshness and Change Rule

Each registered inventory row must route through `REGISTRY.md`, have an SKVI entry, retain its owner contract, and pass structural SSFV validation. When an owner scope is added, removed, renamed, implemented, retired, or materially changes purpose, the inventory and affected semantic records must be reviewed in the same change. Generated discovery may report a candidate or mismatch but cannot amend this inventory.

## Completion Rule

`complete` is permitted only when all of the following are owner-ratified and mechanically valid:

1. every current owner scope in the explicit universe has a registered record or an explicit evidence-backed exclusion;
2. every nested application boundary has been reviewed and recorded as a feature, subfeature, microfeature, or explicit non-feature disposition;
3. every record has current implementation, contract, relationship, and cross-vector evidence;
4. no unregistered `FEATURES.md`, stale route, missing owner contract, missing SKVI entry, broken relationship, or unresolved semantic-freshness finding remains;
5. one reviewed change records the exact inventory revision and evidence establishing completeness.

The current inventory satisfies top-level owner-scope routing and the exact F1 through F3 nested-review progress recorded above. Other nested feature, subfeature, microfeature, and non-feature adjudication remains incomplete. Therefore the canonical and engine-reported state remains `partial`.

## Non-Authorization Statement

This contract authorizes the explicit inventory and completion test. It does not declare nested review complete, create semantic truth from filesystem discovery, register proposal-only modules, authorize canonical apply, or permit tooling to decide feature-worthiness.
