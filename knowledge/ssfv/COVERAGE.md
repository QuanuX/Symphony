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
| `modules/sclv-engine` | registered | `ssfv:symphony:sclv-engine` | `modules/sclv-engine/FEATURES.md` |
| `modules/secure-identity-access-governance` | registered | `ssfv:symphony:ssiag-foundation` | `modules/secure-identity-access-governance/FEATURES.md` |
| `modules/skvi-engine` | registered | `ssfv:symphony:skvi-engine` | `modules/skvi-engine/FEATURES.md` |
| `modules/sodv-engine` | registered | `ssfv:symphony:sodv-engine` | `modules/sodv-engine/FEATURES.md` |
| `modules/ssfv-engine` | registered | `ssfv:symphony:ssfv-engine` | `modules/ssfv-engine/FEATURES.md` |
| `modules/ssiag-provider-macos-keychain` | registered | `ssfv:symphony:ssiag.macos-keychain-metadata` | `modules/ssiag-provider-macos-keychain/FEATURES.md` |
| `modules/stav-append-authority` | registered | `ssfv:symphony:stav-append-authority` | `modules/stav-append-authority/FEATURES.md` |
| `tools/qxctl` | registered | `ssfv:symphony:qxctl` | `tools/qxctl/FEATURES.md` |
| `tools/symphony-validator` | registered | `ssfv:symphony:symphony-validator` | `tools/symphony-validator/FEATURES.md` |

## Ratified Nested Review Progress

The F1 nested review records fourteen implemented `subfeature` boundaries inside three already registered owner scopes:

| Owner scope | Ratified nested records | Review disposition |
|---|---:|---|
| `tools/qxctl` | 8 | Exact engine bindings, authenticated sessions, lifecycle convergence, Linux report-only host receptor, SSIAG administration, STAV administration, Maestro administration, and governed validation are registered. |
| `modules/knowledge-session-coordinator` | 5 | Reconciliation, authority epochs, semantic maintenance, lifecycle planning, and lifecycle apply coordination are registered. |
| `modules/maestro` | 1 | Complete derived receptor inventory is registered separately from durable receptor presence. |

This review advances completion condition 2 for those exact boundaries only. It does not imply that every other nested application boundary has been adjudicated or that an unlisted boundary has an explicit non-feature disposition.

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

The current inventory satisfies top-level owner-scope routing and the exact F1 nested-review progress recorded above. Other nested feature, subfeature, microfeature, and non-feature adjudication remains incomplete. Therefore the canonical and engine-reported state remains `partial`.

## Non-Authorization Statement

This contract authorizes the explicit inventory and completion test. It does not declare nested review complete, create semantic truth from filesystem discovery, register proposal-only modules, authorize canonical apply, or permit tooling to decide feature-worthiness.
