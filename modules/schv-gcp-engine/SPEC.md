# Symphony Cloud Hyperscalers Vector — Google Cloud Engine Specification

## Status and Ownership

Architect-ratified source-knowledge development increment, version `0.1.0-dev`; not a published release or complete provider implementation. Domain meaning belongs to `knowledge/scv/schv/gcp/SPEC.md` and `knowledge/scv/SOURCE-KNOWLEDGE.md`. Common process semantics belong to `knowledge/SPEC.md`. All eight packages use the same explicit operation implementation with an exact installed domain identity.

## Process and Operation Registry

Reads one bounded `symphony.knowledge.engine-process.v1` request from stdin and writes one digest-bound response to stdout. Direct diagnostics are `--help`, `--version` and `--descriptor`. The exact descriptor is v2. Payload objects are closed; unsupported operations, malformed fields, mismatched identities or digests, stale source state, invalid time and exceeded bounds fail visibly. There is no listener, watcher or implicit background work.

Each operation's stable ID is `engop:symphony:schv-gcp.` followed by its wire name with underscores replaced by dots. The descriptor is derived from the same registry as dispatch. The qxctl leaf uses `--domain schv-gcp` and selects an exact receipt-backed prefix and version.

| Wire operation | qxctl scv leaf | Result protocol | Interaction |
|---|---|---|---|
| `inspect` | `inspect` | `symphony.knowledge.engine-descriptor.v2` | inspect |
| `provider_onboard` | `provider-onboard` | `symphony.scv.provider.v1` | discover |
| `source_plan` | `source-plan` | `symphony.scv.source-plan.v1` | propose |
| `source_apply` | `source apply / source recover` | `symphony.scv.source-transition.v1` | apply/recover |
| `source_status` | `source-check / source status` | `symphony.scv.source-status.v1` | inspect |
| `capture_import` | `capture-import / acquire` | `symphony.scv.capture.v1` | invoke |
| `capture_compare` | `capture-compare` | `symphony.scv.capture-diff.v1` | validate |
| `knowledge_interpret` | `interpret` | `symphony.scv.knowledge.v1` | invoke |
| `graph_build` | `graph` | `symphony.scv.graph.v1` | invoke |
| `graph_query` | `query` | `symphony.scv.query-result.v1` | query |
| `graph_diff` | `diff` | `symphony.scv.diff-result.v1` | validate |
| `graph_explain` | `explain` | `symphony.scv.explain-result.v1` | query |
| `graph_evaluate` | `evaluate` | `symphony.scv.evaluate-result.v1` | validate |

## Source and Capture Payloads

Exact desired-source, source, plan, transition, status, provider and capture objects are governed by `knowledge/scv/schemas/v1/source.schema.json`, `capture.schema.json` and the named definitions in `source-operation.schema.json`. `inspect` accepts an empty object. `source_apply` only computes the owner-validated transition; it is pure and cannot claim a protected storage commit. qxctl storage administration has separate authorization, journal and expected-state responsibilities.

`provider_onboard` accepts an explicit provider ID, family ID, display name and selected desired-source records. It emits a candidate; it never claims automatic understanding of every vendor service. SCV accepts declared domains, family engines constrain the family ID, and a provider engine constrains both family and provider. This is an installed tool boundary, not a rule constraining user compositions.

Capture input retains the exact source revision and supplied retrieval metadata. Body hashing binds the UTF-8 bytes and size. Import is not a live fetch. The optional qxctl HTTPS acquisition adapter passes its actual bounded retrieval evidence to the same import operation; it cannot silently reinterpret a failed or partial retrieval as complete.

## Knowledge Payloads

`knowledge_interpret` accepts `{captures, claims, interpreter_version, selection_policy}` under `interpretation-input.schema.json`. Evidence quotations must match retained capture bodies. This validates attribution and bounded structure, not independent truth of an interpretation. Claims retain typed values, qualification, statement kind, support alternatives, searched-scope dependencies and optional validity bounds. Unsupported or partial native extraction stays visible.

`graph_build` accepts `{knowledge:[Knowledge]}` whose domain and policy must be compatible with the invoked engine. `graph_query` and `graph_evaluate` accept `{graph, query_time}` plus optional `claim_ids`, `subject` and `predicate`. `graph_explain` accepts `{graph, query_time, claim_id}`. `graph_diff` accepts `{before, after, query_time}`. Times use strict UTC whole seconds. Every operation revalidates digest-bound input and emits deterministic semantic evidence under the selected policy.

Explanation retains transitive dependencies and separate support paths. Difference retains before/after identities and evaluates both at one explicit time. Absence, conflict, hypothesis, stale observation, unsupported adapter and proven constraint remain distinct evidence. Results never certify empirical truth, rank a provider, select a deployment or authorize an action.

## Storage and Reproduction

The C++ process is a deterministic reducer/projector. It does not select durable source, corpus or graph heads, keep a provider account database, or retain private observations outside explicitly supplied inputs. Same sealed evidence and accepted explicit interpretations produce the same semantic projection; fresh AI generation is not required. The storage/consumer layer must enforce selection, retention, authorization, replay and expiry where those operations exist.

## Bounds and Evidence

Shared process limits remain unchanged. Source declarations have 1–16 locators; provider onboarding admits 1–32 selected sources. UTF-8 capture bodies are at most 65,536 bytes. Knowledge is bounded to 16 captures, 128 claims, eight support sets per claim, 16 premises per set, 512 native nodes and 1,024 native edges. Aggregate byte, string, value-count, depth and deadline limits apply independently. Overflow does not silently change selected facts.

Source/knowledge owner regressions, qxctl consumer rejection and exact installed-process checks establish only their exercised scope. The invariant registry records their traceability; a named test or source contract alone is not proof of execution. Complete online corpus acquisition, binary/OCR ingestion, general graph connectors, arbitrary multistep synthesis, provider accounts/credentials and cloud deployment remain beyond this increment. qxctl supplies a separate protected graph-head selection adapter; this C++ process validates the exact graph without mutating that head.

## Non-Authorization

No valid result grants permission, authorizes canonical source changes, publishes documentation, invokes source instructions, activates an engine or mutates a cloud resource. Private local state and official repository truth remain distinct. The C++ process does not weaken existing SSIAG local kernel-peer and STAV audit contracts.
