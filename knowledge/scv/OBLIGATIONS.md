# SCV Obligation Inventory and Follow-up Contract v1

## Ownership and scope

Exact `0.8.0-dev` adds native `composition_obligations` and `composition_followup`, bringing each supplied C++ engine to 28 operations. The existing composition and reassessment protocols retain their meanings. This companion owns `obligation.schema.json` and the qxctl relationship contract `scv-obligation-link.schema.json`.

The first operation makes outstanding work precisely addressable. The second evaluates the same native check in a caller-supplied later composition and retains the supplied provenance reference. It does not treat a document, observation report, implementation declaration, signature or digest as a self-proving resolution. SCV owns the evidence comparison; the user owns requirements, choices, policy, time and execution.

## Precise inventory

Input is `{composition}`, containing a complete replayable composition exploration. Result `symphony.scv.composition-obligations.v1` retains `{protocol,domain,input,composition_digest,obligations,limitations,digest}`. Inventory entries are `{obligation_id,kind,target,prior_status,claim_ids,description}`, sorted by obligation ID.

Each target has exactly ten nullable fields: `scenario_id`, `candidate_id`, `connection_id`, `check_id`, `requirement_id`, `slot_id`, `recipe_id`, `interface_ref`, `pack_evaluation_digest`, and `fixture_id`. Its kind determines the applicable fields; the others are null. The obligation ID is the ordinary SCV seal digest of `{composition_digest,kind,target}`. It binds the complete composition identity and precise subject. An ID cannot be moved to another composition simply because a recipe or check has the same name.

| Kind | Inventoried subject | Prior status |
| --- | --- | --- |
| `check` | Each non-satisfied native candidate check, including optional findings, identified by scenario/candidate/connection/check. | Its exact native check status. |
| `binding` | Each unbound or ambiguous common requirement, identified by scenario/candidate/requirement. | `missing_binding` or `ambiguous_binding`. |
| `interface` | Each unmatched declared interface, identified by scenario/candidate/slot/recipe/interface. | `missing`. |
| `implementation` | Every selected recipe’s implementation, identified by scenario/candidate/slot/recipe. | Its caller-declared status. |
| `fixture` | Every failed or unrun global provider-pack fixture, identified by exact pack-evaluation digest and fixture ID. | Its fixture status. |

The inventory is reconstructed from the replayed native checks and exact subjects. It does not match anonymous legacy obligation summaries by kind/detail, nor assume that two identical summaries identify the same prerequisite. Claim IDs retain the applicable check, binding or fixture references; they do not add missing proof. Inventory completeness concerns the evaluated finite composition only, not unevaluated candidates or the entire vendor domain.

## Subsequent-evidence comparison

Follow-up input is exactly `{before,after,submissions}`. Both composition results are replayed. Requirements, full slots/recipes, provider selections, guarantee permissions, counterfactuals, bounds and composed evidence policy must remain deeply equal. Candidate identity alone is insufficient to establish that the caller’s criterion stayed fixed. A changed architecture or policy belongs in an ordinary new composition/reassessment, rather than satisfying an old criterion by editing it.

Evidence and the explicitly selected query times may differ. Existing reassessment axes report both. The engine does not substitute the current time, infer chronology from the words before/after, or claim that a difference was caused by the submitted reference. A refreshed capture cannot silently become evidence at a time before its observation. Alternatively, a caller can explicitly re-evaluate the old evidence at a shared later time; that creates a new composition and new obligation identities.

There are one to 32 submissions, each exactly `{submission_id,obligation_id,provenance}`. Submission IDs and obligation IDs are independently unique within the request. Each obligation must exist in the precise before inventory. Provenance is `{kind,producer,reference,content_digest,recorded_at,description}`. Kind is `source`, `observation`, `adapter` or `caller_decision`; producer/reference are caller-supplied text; content digest is null or a SHA-256 identifier; recorded time is canonical UTC seconds. Text identities are at most 512 UTF-8 bytes and descriptions at most 2,048 bytes, without ASCII controls.

Provenance is retained as `reference_only`. The engine neither obtains the external content nor authenticates its producer, validates its signature, verifies its declared digest or executes it. Actual semantic support comes from the replayed after composition and its retained evidence. A native claim about a reported observation retains that report’s source meaning; it does not imply that Symphony performed the observation.

## Outcomes and limits

Result `symphony.scv.composition-followup.v1` retains `{protocol,domain,input,before_digest,after_digest,change_axes,candidate_changes,entries,limitations,digest}`. Candidate changes are the ordinary native reassessment records, including unchanged and unrelated candidates. Entries are sorted by submission ID and retain `{submission_id,obligation_id,kind,target,prior_status,current_status,outcome,claim_ids,provenance_validation,causation}`.

For a check, the engine finds the exact same scenario/candidate/connection/check in the after result and verifies its specification. `current_status` is that native status. Outcome is `criterion_satisfied` only for `satisfied`; conditional, unresolved and contradicted remain `criterion_not_satisfied`, with the exact status visible. This describes the criterion under the selected after evidence and time. Every entry reports `causation=not_established`; an irrelevant submission cannot receive causal credit for an improvement elsewhere in the selected evidence.

Bindings, interfaces, implementation and fixtures remain `requires_separate_verification`, with null current status. This release makes those subjects addressable and preserves their submitted references, but does not claim their verification lifecycle is implemented. Fixture replacement needs its own precise expected-meaning comparison. A declared available implementation still requires runtime verification. No submission globally closes an obligation or certifies a deployment.

The unchanged one-MiB request/four-MiB response limits and native evidence/check/search bounds still apply. Inventory is not silently truncated to fit; oversized results fail explicitly. No general scheduler, crawler, database connector, obligation executor, ranking or automatic requirement relaxation is introduced.

## qxctl and immutable relationship retention

All native functionality is available through `qxctl scv composition obligations inspect|followup`, defaulting to exact `.8`, and the existing artifact import/show/list surfaces. Installed schema discovery supplies both native inputs/results and local relationship inputs/results.

`qxctl scv composition obligations retain` accepts `{before_ref,after_ref,submissions}` in an explicit `--workflow-root` and `--tops-id`. Each reference is an immutable composition **record digest**, binding its content, original input and exact installed owner. Both original owners are replayed before the selected `.8` follow-up owner is invoked. The new native result is retained as an ordinary `composition_followup` artifact record.

The adapter then stores an immutable relationship `{protocol,root,tops_id,before_ref,after_ref,followup_ref,submissions_digest}` with protocol `symphony.qxctl.scv-obligation-link.v1`, inside the existing sealed workflow payload envelope. Its wrapper digest is `link_ref`. Submission digest binds the canonical object `{submissions}`. The two recorded source references and the new result reference remain distinct from native content digests.

`qxctl scv composition obligations show` accepts `{link_ref}`. It checks root/TOPS binding, replays both original composition owners and the recorded follow-up owner, and verifies that the follow-up input matches the referenced native compositions and exact submissions digest. A resealed wrapper with substituted references fails correspondence checks. It returns the same sealed `symphony.qxctl.scv-obligation-link-result.v1` shape as retain: `{protocol,link_ref,link,followup_ref,validation,digest}`, with `validation=owner_replayed`.

Retained artifact records keep the existing four-MiB file limit. Their exact record envelope may contain up to 65,792 JSON values, with at most 256 metadata values; input and native artifact are independently limited to 32,768 values each, with one-MiB input and four-MiB artifact byte limits. The total record byte bound still applies to their combination. This separate envelope allowance accounts for two already bounded children; ordinary inputs, results, payloads and runs retain their existing JSON value limits. Duplicate-key, depth, UTF-8 and integer checks remain active. Oversized native requests still fail; references do not authorize clipping their dependency closure.

Publication writes the native result before its link. Retrying the exact input after an interruption reuses immutable identical objects; it does not select a head or modify the prior link. A result retained before an interruption may exist without a published link and remains ordinary inspectable evidence. No external reference is fetched or executed during either operation.

## Acceptance

Independent owner and consumer checks cover exact target identity, duplicate prerequisite summaries, satisfied/conditional/unresolved/contradicted evidence, fixed caller criteria and policy, explicit time changes, opaque provenance, unsupported verification categories, bounds, forged results and substituted references. Installed-process checks demonstrate the selected runtime and its earlier exact owner compatibility. Source examples preserve real capture provenance and independent authored expectations. Neither schemas nor matching digests replace those semantic and runtime checks.
