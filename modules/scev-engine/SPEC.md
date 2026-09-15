# Symphony Cloud Edge Vector Engine Specification

## Status and Ownership

Architect-ratified source-knowledge development increment, current source package `0.10.0-dev`; not a published release or complete provider implementation. Domain meaning belongs to `knowledge/scv/scev/SPEC.md` and `knowledge/scv/SOURCE-KNOWLEDGE.md`. Common process semantics belong to `knowledge/SPEC.md`. All eight packages use the same explicit operation implementation with an exact installed domain identity.

## Process and Operation Registry

Reads one bounded `symphony.knowledge.engine-process.v1` request from stdin and writes one digest-bound response to stdout. Direct diagnostics are `--help`, `--version` and `--descriptor`. The exact descriptor is v2. Payload objects are closed; unsupported operations, malformed fields, mismatched identities or digests, stale source state, invalid time and exceeded bounds fail visibly. There is no listener, watcher or implicit background work.

Each operation's stable ID is `engop:symphony:scev.` followed by its wire name with underscores replaced by dots. The descriptor is derived from the same registry as dispatch. The qxctl leaf uses `--domain scev` and selects an exact receipt-backed prefix and version.

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
| `capture_index` | `corpus acquire / import / recover / inspect / export / diff` | `symphony.scv.capture-index.v1` | invoke |
| `corpus_build` | `corpus acquire / import / recover / inspect / export / diff` | `symphony.scv.corpus.v1` | invoke |
| `corpus_query` | `corpus export` | `symphony.scv.corpus-query.v1` | query |
| `corpus_diff` | `corpus diff` | `symphony.scv.corpus-diff.v1` | validate |
| `provider_interpret` | `provider interpret` | `symphony.scv.provider-interpretation.v1` | invoke |
| `connection_evaluate` | `connection evaluate` | `symphony.scv.connection-evaluation.v1` | validate |
| `connection_reassess` | `connection reassess` | `symphony.scv.connection-reassessment.v1` | validate |
| `profile_prepare` | `profile prepare` | `symphony.scv.interpretation-profile.v1` | propose |
| `provider_coverage` | `provider coverage` | `symphony.scv.provider-coverage.v1` | query |
| `provider_pack_prepare` | `provider pack prepare` | `symphony.scv.provider-pack.v1` | propose |
| `provider_pack_evaluate` | `provider pack evaluate` | `symphony.scv.provider-pack-evaluation.v1` | query |
| `composition_explore` | `composition explore` | `symphony.scv.composition-exploration.v1` | query |
| `composition_reassess` | `composition reassess` | `symphony.scv.composition-reassessment.v1` | validate |
| `composition_obligations` | `composition obligations inspect` | `symphony.scv.composition-obligations.v1` | query |
| `composition_followup` | `composition obligations followup` | `symphony.scv.composition-followup.v1` | validate |
| `bundle_inspect` | `composition bundle inspect` | `symphony.scv.bundle-inspection.v1` | inspect |
| `composition_bundle_evaluate` | `composition bundle evaluate` | `symphony.scv.composition-bundle-evaluation.v1` | validate |

`knowledge/scv/CORPUS.md@v1` owns the four additive operations and their closed schemas. The table records composed qxctl routes; the C++ operations remain independently callable and never persist a corpus head. The `0.3.0-dev` descriptor has exactly twenty operations, including the three operations owned by `knowledge/scv/INTERPRETATION.md@v1`. Existing `0.1.0-dev` and `0.2.0-dev` installations retain their exact thirteen- and seventeen-operation descriptors, immutable receipts/documents and original payloads. The qxctl consumer checks the selected version's finite operation set; it cannot substitute a newer package. Original command defaults remain `0.1.0-dev`; corpus commands default to `0.2.0-dev` and explicitly permit `0.3.0-dev`; provider interpretation and connection commands default exactly to `0.3.0-dev`.

## Source and Capture Payloads

Exact desired-source, source, plan, transition, status, provider and capture objects are governed by `knowledge/scv/schemas/v1/source.schema.json`, `capture.schema.json` and the named definitions in `source-operation.schema.json`. `inspect` accepts an empty object. `source_apply` only computes the owner-validated transition; it is pure and cannot claim a protected storage commit. qxctl storage administration has separate authorization, journal and expected-state responsibilities.

`provider_onboard` accepts an explicit provider ID, family ID, display name and selected desired-source records. It emits a candidate; it never claims automatic understanding of every vendor service. SCV accepts declared domains, family engines constrain the family ID, and a provider engine constrains both family and provider. This is an installed tool boundary, not a rule constraining user compositions.

Capture input retains the exact source revision and supplied retrieval metadata. Body hashing binds the UTF-8 bytes and size. Import is not a live fetch. The optional qxctl HTTPS acquisition adapter passes its actual bounded retrieval evidence to the same import operation; it cannot silently reinterpret a failed or partial retrieval as complete.

## Knowledge Payloads

`knowledge_interpret` accepts `{captures, claims, interpreter_version, selection_policy}` under `interpretation-input.schema.json`. Evidence quotations must match retained capture bodies. This validates attribution and bounded structure, not independent truth of an interpretation. Claims retain typed values, qualification, statement kind, support alternatives, searched-scope dependencies and optional validity bounds. Unsupported or partial native extraction stays visible.

`graph_build` accepts `{knowledge:[Knowledge]}` whose domain and policy must be compatible with the invoked engine. `graph_query` and `graph_evaluate` accept `{graph, query_time}` plus optional `claim_ids`, `subject` and `predicate`. `graph_explain` accepts `{graph, query_time, claim_id}`. `graph_diff` accepts `{before, after, query_time}`. Times use strict UTC whole seconds. Every operation revalidates digest-bound input and emits deterministic semantic evidence under the selected policy.

Explanation retains transitive dependencies and separate support paths. Difference retains before/after identities and evaluates both at one explicit time. Absence, conflict, hypothesis, stale observation, unsupported adapter and proven constraint remain distinct evidence. Results never certify empirical truth, rank a provider, select a deployment or authorize an action.

## Corpus Payloads

`capture_index` validates and projects an exact Capture v1. `corpus_build` constructs a sorted immutable snapshot from the full explicit member set and optional predecessor. Failed/partial refresh retains old complete evidence separately without changing its source revision or age. `corpus_query` applies an explicit latest-attempt or last-complete selection with query time and freshness; `corpus_diff` reports evidence and membership changes without inferring retirement. Corpus/member lineage checks are limited to the supplied immediate predecessor. At most 128 member indexes fit subject to the existing process bounds; materializing captured bytes is separately bounded to sixteen captures and the original request limits.

## Provider Interpretation and Connections

`provider_interpret` accepts exact captures, sealed authored profiles, explicit one-profile/one-capture bindings and the existing selection policy. Literal or delimited extraction requires unique selected context; unresolved or ambiguous extraction emits a finding without a claim. The wrapper retains profiles, bindings and ordinary knowledge v1 for exact replay. Structural attribution is not semantic or empirical proof.

`connection_evaluate` replays retained wrappers, composes explicit additional knowledge and evaluates caller-selected required/optional checks at a specified time. Claims must match exact subjects, scopes, types and units. Missing, stale or conflicted evidence remains unresolved; assumptions and recommendations remain conditional. `connection_reassess` replays both retained results and separates changed captures, profiles, knowledge, policy, requirements and time. Results do not test a deployed route, select a provider or mutate a graph head. `knowledge/scv/INTERPRETATION.md` and its closed schemas define the exact payloads and finite bounds.

## Storage and Reproduction

The C++ process is a deterministic reducer/projector. It does not select durable source, corpus or graph heads, keep a provider account database, or retain private observations outside explicitly supplied inputs. Same sealed evidence and accepted explicit interpretations produce the same semantic projection; fresh AI generation is not required. The storage/consumer layer must enforce selection, retention, authorization, replay and expiry where those operations exist.

## Bounds and Evidence

Shared process limits remain unchanged. Source declarations have 1–16 locators; provider onboarding admits 1–32 selected sources. UTF-8 capture bodies are at most 65,536 bytes. Knowledge is bounded to 16 captures, 128 claims, eight support sets per claim, 16 premises per set, 512 native nodes and 1,024 native edges. Aggregate byte, string, value-count, depth and deadline limits apply independently. Overflow does not silently change selected facts.

Source/knowledge owner regressions, qxctl consumer rejection and exact installed-process checks establish only their exercised scope. The invariant registry records their traceability; a named test or source contract alone is not proof of execution. Complete online corpus acquisition, binary/OCR ingestion, general graph connectors, arbitrary multistep synthesis, provider accounts/credentials and cloud deployment remain beyond this increment. qxctl supplies a separate protected graph-head selection adapter; this C++ process validates the exact graph without mutating that head.

## Non-Authorization

No valid result grants permission, authorizes canonical source changes, publishes documentation, invokes source instructions, activates an engine or mutates a cloud resource. Private local state and official repository truth remain distinct. The C++ process does not weaken existing SSIAG local kernel-peer and STAV audit contracts.

## Agent Operating Increment

`knowledge/scv/AGENT-WORKFLOWS.md` owns additive `.4` profile preparation, receipt-owned schema discovery and qxctl evidence workflow coordination. The C++ engine seals and validates caller-authored profile drafts without inventing claims. Its `.4` descriptor contains 21 operations. Metadata corpus queries and retained runs use existing owner operations with their original bounds. Existing `.1`/`.2`/`.3` installations and success payloads remain separately valid at their exact scope.

## Maintained Provider Coverage

The additive `0.5.0-dev` package has 22 operations. `knowledge/scv/COVERAGE.md` owns native accounting of declared sources, exact corpus selection and independently replayed interpretations. Missing, unlisted, unselected, partial and stale evidence remains explicit. The operation does not rank providers or establish runtime compatibility. Earlier exact `.4` and prior installations remain preserved.

## Portable Provider Authoring and Composition Exploration

The additive `0.6.0-dev` package exposes 26 operations. `knowledge/scv/PROVIDER-PACKS.md` owns portable caller-authored provider packs, native sealing, detached conformance fixtures and bounded structured extraction. A pack links an explicit provider declaration, authored profiles, source references and selected fixture expectations; a successful fixture comparison describes those cases alone. Qualified knowledge remains reusable through the existing evidence interfaces. This is an authoring surface for independently chosen providers, without requiring a new compiled provider enum in SCV; a supplied leaf still enforces its advertised family/provider identity.

`knowledge/scv/COMPOSITION.md` owns finite exploration of caller-selected recipes against caller-selected requirements. Native operations preserve evidence, prerequisites, interface declarations, guarantee changes and authored resolution pointers; missing evidence remains unresolved. The engine does not enumerate an open-ended design space, rank providers, choose a user's requirements, provision infrastructure or turn a declaration into observed compatibility. Reassessment preserves exact before/after inputs and changed axes.

`knowledge/scv/OWNER-INTERFACE.json` is the versioned owner declaration for operation metadata, release admission, artifact kinds and installation inventories. Its checked-in generated projections drive native dispatch metadata and Go interface admission; domain validation and adversarial consumer tests remain independent. The package includes eight owner companions, the schema catalog and a receipt-owned copy of the declaration, inspectable through `qxctl scv interface show`. Normal builds and engine invocation do not require the C++ authoring generator. Earlier exact `.1`–`.5` receipts and CLI defaults remain unchanged; new pack, composition and interface commands default to exact `.6`.

## Maintained Composition Coordination

The exact `0.7.0-dev` release retains 26 native operations and adds the installed `knowledge/scv/COMPOSITION-WORKFLOWS.md` companion and its workflow schema. qxctl coordinates original-owner package evaluations, finite exploration and optional reassessment with pinned intent, immutable artifact records and interruption recovery. Nine owner companions and 24 schemas expose 89 protocol entries. The existing native meanings remain unchanged; a completed run preserves source gaps, failed fixtures and implementation obligations. Earlier exact `.1`–`.6` installations and command defaults remain available.

## Precise Obligation Follow-up

Exact `0.8.0-dev` exposes 28 native operations, including replayed obligation inventory and subsequent-evidence comparison under `knowledge/scv/OBLIGATIONS.md`. qxctl exposes direct operations and immutable original-owner relationship retention/show. Native check state remains separate from supplied reference provenance, causal attribution and runtime verification. Ten owner companions and 26 schemas expose 97 catalog protocols. Earlier exact packages, protocols and command defaults remain preserved.

## Exact Evidence Bundles

Exact `0.9.0-dev` exposes 30 native operations. `knowledge/scv/BUNDLES.md` owns complete bounded evidence-reference transport, transport-only inspection and evaluation through the unchanged composition owner. qxctl exposes pack, inspect, evaluate and expanded-result routes, plus existing exact-owner artifact retention/show. Repeated evidence is stored once inside a complete bundle; reconstruction counts all logical occurrences before allocation. Caller requirements, provider selections and semantics remain unchanged. Eleven owner companions and 27 schemas expose 103 catalog protocols. Earlier exact packages and command defaults remain preserved.

## Explicit Bundle Workflows

Exact `0.10.0-dev` retains thirty native operations and adds qxctl bundle workflow run/status/recover and immutable bundled obligation retain/show. `knowledge/scv/BUNDLE-WORKFLOWS.md` owns the versioned coordination and logical-reference contracts. Separate v2 journals pin explicit transport, caller input and exact installation before work; original records preserve both native and transport identities. Status reports sealed checkpoint validation, while run/recovery and relationship inspection replay original owners. Twenty-nine schemas expose112 catalog protocols with twelve owner companions. Prior routes, defaults, records and packages remain preserved.
