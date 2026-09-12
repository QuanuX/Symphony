# SCV Agent Operating Workflow Contract v1

## Scope and Ownership

Duncan authorized this increment after the SCV qxctl coverage audit and reaffirmed “Agentic-first, humans welcome.” This companion defines the additive `0.4.0-dev` development interface. It does not claim a published release or complete provider knowledge. Earlier exact `.1`, `.2` and `.3` installations retain their own receipts and interfaces.

SCV's C++ engines own interpretation-profile validation, native knowledge and graph meaning, connection checks and reassessment. qxctl owns command presentation, exact receipt resolution, bounded transport, immutable local evidence retention and the sequencing of those explicit operations. These local evidence workflows confer no selected source/graph head, canonical-write, cloud-provider, deployment or credential authority. Existing SSIAG/STAV circuits continue to govern their existing protected operations. Human and permitted agent callers use the same existing permission contracts.

Caller-owned source selections, profile mappings, additional knowledge, requirements, policies and times remain explicit inputs. A worked example or a template does not become a platform requirement or evidence of compatibility.

## Profile Preparation

The new pure C++ operation `profile_prepare` accepts `{profile}`. The profile contains exactly the existing `symphony.scv.interpretation-profile.v1` fields except `digest`. The engine computes the seal and applies the same strict profile/rule validation used by `provider_interpret`, returning the existing sealed profile shape. Preparation does not inspect a capture, certify a statement, infer a provider fact or grant authority. Exact author/version provenance remains caller supplied.

`qxctl scv profile prepare` administers this operation. The `.4` descriptor exposes 21 operations; earlier exact descriptors continue to expose 13, 17 and 20 operations. Consumer checks must reject altered or mismatched results. Existing interpretation/evaluation/reassessment protocols remain unchanged.

## Discoverable Inputs and Output

`qxctl scv schema list|show|template` exposes the selected `.4` installation's receipt-owned schema catalog and exact schema documents. Protocol IDs identify named schema definitions where the input contains no top-level protocol field. Relative references must resolve inside the returned package bundle. Discovery requires no source checkout, network service or backend query language.

Catalog and schema digests identify bytes, not semantic correctness. Templates contain explicit authoring holes and accompanying required-input guidance. Placeholders cannot be passed off as valid provider claims or silently replaced with inferred user policy. Process, byte, value, depth and operation-specific bounds remain separate and discoverable; schema structure alone does not implement cross-field validation.

Human help derives from the actual executable command tree. Machine discovery continues through the digest-bound qxctl command registry. Registered command IDs, engine-operation IDs and feature IDs remain separate.

For SCV invocations requesting JSON, failures must preserve a structured error envelope with stable diagnostic classification and nonzero exit status. Engine rejection codes survive transport without arbitrary raw error strings leaking payloads or secrets. Existing success protocols and specialized exit semantics remain intact. Human presentation and JSON describe the same outcome.

## Immutable Artifact Retention

`qxctl scv artifact import|show|list` retains supported native profiles, knowledge, provider interpretations, connection evaluations and reassessments. A retained record binds native content, validating operation/input and exact installed-engine identity. Import must validate or replay the native artifact through its semantic owner before describing it as validated. A self-digest alone is insufficient. Provider-authored records preserve their native domain when later selected by a parent-domain composition.

Native content keeps its own digest. A separate record digest binds that content to the validating input, operation and installation; caller references select this exact provenance record. Identical native content validated by different installations can coexist. Import accepts `{operation,input,result}`; null result invokes and retains the validated owner output, while a supplied result additionally requires exact replay equality. Names, versions and domain labels are inspectable metadata, not implicit latest aliases. Duplicate publication checks existing bytes. Lists use bounded digest-ordered pages and explicit continuation; they report envelope integrity without claiming current owner replay. Show and semantic use replay retained results through their exact owner. No automatic deletion or private-to-public publication is implied.

## Retained Runs and Recovery

`qxctl scv workflow run|status|recover` composes retained corpus selection, provider interpretation, connection evaluation and optional reassessment against an exact earlier evaluation. The request explicitly selects the corpus snapshot and member/evidence policy, profile-to-member bindings, extra retained evidence, caller connection requirements and query time. The selected root/TOPS namespace and exact installation belong to the run identity.

The adapter records immutable intent before advancing stages. Stable operation IDs cannot be reused for different inputs or installations. Exact captures and profiles are selected before interpretation. Each stage retains its invocation and validated output before advancing; a crash may leave reusable unpublished evidence but cannot claim an uncompleted stage succeeded. Recovery replays and verifies recorded checkpoints against the original intent and exact installation. A pure stage may be recomputed after interruption. Completed reruns return the retained result without reacquisition or new policy selection.

Status is observational: it reports retained progress and failures without advancing stages or fabricating success. Recovery never silently substitutes the current time, a newer schema/profile/engine, a different corpus head, a changed file or newer network content. A refresh is a new explicit corpus operation and a new run; prior evidence remains available for reassessment. Acquisition/import and their interruption recovery remain the existing maintained-corpus workflow.

The retained run is optional composition assistance. Direct owner operations remain available. External modules and callers may prepare alternate valid compositions under their own contracts.

### Exact Workflow Request

Run input is `{operation_id,corpus,profile_bindings,interpretation_refs,knowledge_refs,selection_policy,query_time,connections,prior_evaluation_ref}`. References select retained record digests, not native content digests. Status and recover accept `{operation_id}`. All commands use an explicit owned `--workflow-root` and `--tops-id`; execution selects an exact `--prefix`, `--domain` and `--version`. The initial corpus-backed run additionally supplies a clean absolute `--corpus-root`, which recovery retains.

`corpus` is null or `{domain,snapshot_digest,member_ids,selection,max_age_seconds}`. A non-null corpus selects one to sixteen explicit members. Each `{profile_ref,member_id}` binding selects a prepared profile for an exact member; every selected member is covered, and one profile reference is not ambiguously bound twice. Multiple distinct profiles may address the same selected capture within the existing native interpretation limits. `selection_policy` is an explicit owner policy for the newly produced corpus interpretation. Separately retained interpretations and knowledge preserve their own embedded policies. A reference-only run supplies null `corpus`, an empty `profile_bindings` array and null `selection_policy`; supplying an unused policy is rejected.

Each interpretation/knowledge reference array contains at most sixteen distinct record references. The owner still enforces its aggregate input and connection limits. `prior_evaluation_ref` is null or one exact retained evaluation to reassess against the new result. The selected query time and caller connections are passed unchanged to the C++ owner. Neither the CLI nor recovery reinterprets the caller's evidence policy as a universal provider preference.

Artifact list input is `{after_digest,limit}` with an explicit null initial cursor and a page size from one to 128. Listings report sealed-envelope integrity, include an explicit continuation cursor, and do not claim live native replay. The bounded inventory scan permits at most 65,536 directory entries. Pagination observes the current immutable inventory rather than a frozen listing snapshot; concurrent additions before a cursor may require a new scan. Show accepts `{record_digest}` and requires original-owner replay. Record and run files are at most four MiB; native invocation/stage input remains at most one MiB and still obeys all existing JSON/process bounds.

## Storage and Boundedness

Local evidence storage follows the existing corpus adapter's no-follow, owned private-directory, bounded regular-file, digest verification, durable write and atomic publication principles. Concurrent runs use independent operation identity and serialization; collisions do not select a winner by last write. This adapter follows the existing qxctl/corpus implementation targets, Darwin and Linux; it adds no support for other hosts or weaker fallback storage semantics. New state layouts are explicitly versioned.

Existing C++ process limits, capture and interpretation bounds remain in force. The workflow does not widen an evaluation to 128 captures merely because the corpus has 128 members. `qxctl scv corpus query` exposes bounded metadata selection without materializing captures; `corpus export` retains its sixteen-capture limit. Exact executable interface and schemas define additional adapter bounds.

## Acceptance and Limits

Verification must exercise prepared-profile rejection, schemas/templates without a repository, native owner replay, machine-readable failures, duplicate/colliding runs, missing or corrupted artifacts, interrupted stages, exact-engine drift, and retained before/after reassessment. Existing retained Cloudflare/GCP documents can test workflow behavior without representing a new live provider verification. A labeled synthetic source change remains separate from actual vendor changes.

A satisfied documentation check does not establish deployed reachability, account permissions, application identity, payload compatibility or a working strategy. Such evidence remains independently supplied and scoped. Neither profile preparation nor workflow completion resolves missing facts by assumption.

## Additive Provider Coverage Profile

The `.5` interface in `knowledge/scv/COVERAGE.md` adds native provider coverage and permits provider candidate and coverage-result artifact retention. Earlier installed `.4` schema/catalog bytes and records remain preserved. Existing preparation, workflow, corpus and discovery defaults remain unchanged; explicit `.5` selection uses its exact receipt.

## Exact .6 Agent Surface

The additive `0.6.0-dev` package installs eight owner companions and the receipt-owned declaration `share/symphony/contracts/<module>/<version>/OWNER-INTERFACE.json`. `qxctl scv interface show --prefix <prefix> --version 0.6.0-dev --domain <domain> --json` verifies and reads that declaration without a source checkout. `schema list`, `schema show` and `schema template` expose its exact schema catalog; templates retain caller authoring holes. Existing command defaults continue selecting their earlier exact releases.

New native routes are `scv provider pack prepare`, `scv provider pack evaluate`, `scv composition explore` and `scv composition reassess`. All accept exact installed domain/prefix/version and bounded JSON inputs through qxctl. `artifact import`, `artifact show` and `artifact list` retain and replay the corresponding `provider_pack`, `provider_pack_evaluation`, `composition` and `composition_reassessment` artifacts under their original exact installation. Package preparation computes detached fixture input digests from caller-supplied fixture bodies; callers need no external hashing helper.

These routes expose all four native operations directly. The existing recoverable `workflow run` remains the earlier corpus/profile/connection workflow; it does not silently acquire a new composition stage. Callers explicitly select package fixtures, requirements, provider IDs, recipe alternatives, permitted guarantee changes and candidate bounds. Failed or unrun conformance evidence and required follow-up work remain visible in exploration. Reading an obligation never executes its reference. Exact semantics belong to `knowledge/scv/OWNER-INTERFACE.md`, `knowledge/scv/PROVIDER-PACKS.md` and `knowledge/scv/COMPOSITION.md`.

## Maintained Composition Coordination

The exact `0.7.0-dev` release retains 26 native operations and adds the installed `knowledge/scv/COMPOSITION-WORKFLOWS.md` companion and its workflow schema. qxctl coordinates original-owner package evaluations, finite exploration and optional reassessment with pinned intent, immutable artifact records and interruption recovery. Nine owner companions and 24 schemas expose 89 protocol entries. The existing native meanings remain unchanged; a completed run preserves source gaps, failed fixtures and implementation obligations. Earlier exact `.1`–`.6` installations and command defaults remain available.

## Precise Obligation Follow-up

Exact `0.8.0-dev` exposes 28 native operations, including replayed obligation inventory and subsequent-evidence comparison under `knowledge/scv/OBLIGATIONS.md`. qxctl exposes direct operations and immutable original-owner relationship retention/show. Native check state remains separate from supplied reference provenance, causal attribution and runtime verification. Ten owner companions and 26 schemas expose 97 catalog protocols. Earlier exact packages, protocols and command defaults remain preserved.

The `.8` obligation delivery corrects retained-record envelope value accounting: only the exact artifact record receives the bounded two-child allowance defined in [OBLIGATIONS.md](OBLIGATIONS.md). Its input and native result retain their independent native bounds, and all record bytes remain within four MiB. Payloads, runs and direct native invocations do not acquire that envelope allowance.
