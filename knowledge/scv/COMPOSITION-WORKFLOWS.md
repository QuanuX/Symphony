# SCV Maintained Composition Workflow Contract v1

## Scope and ownership

The exact `0.7.0-dev` interface adds recoverable qxctl coordination of provider-package evaluation, finite composition and optional reassessment. The native C++ owner retains its 26 operations and unchanged source, interpretation, pack and composition protocols. qxctl supplies explicit sequencing, immutable artifact retention and checkpoint recovery. This companion owns `scv-composition-workflow.schema.json`; it adds no native semantic operation or authority to acquire sources, execute an obligation, select a graph head or change a provider.

The installed declaration admits the `composition_workflow` adapter surface only for exact `.7`. Earlier `.1`–`.6` installations, declarations, receipts, command defaults and workflow records remain unchanged. Caller-authored sources, requirements, recipe/provider selections, interpretation policies and query times remain explicit.

## Operations and inputs

`qxctl scv composition workflow run|status|recover` uses an explicit `--workflow-root` and `--tops-id`. New run/recovery additionally selects domain, prefix and exact `.7` composition owner. Status inspects local state without requiring an executable owner. Earlier `scv workflow run|status|recover` remains its original corpus/profile/connection contract.

Run input is exactly `{operation_id,pack_evaluations,pack_evaluation_refs,interpretation_refs,knowledge_refs,composition,prior_composition_ref}`. Every reference names an immutable artifact **record digest**, which binds the native artifact, invocation and original installed owner; it is not a mutable alias or merely a native content digest.

`pack_evaluations` contains zero to sixteen `{evaluation_id,pack_ref,evaluation}` entries. `pack_ref` selects a retained `provider_pack`; `evaluation` contains exactly `{captures,bindings,selection_policy,fixtures}` from the native package-evaluation contract. Its pack is loaded from that record, and the stage executes through the pack's original exact installation. Distinct evaluation IDs identify stages, not new providers or independent corroboration. New evaluations and `pack_evaluation_refs` together may select at most sixteen packages. The native owner independently enforces duplicate evidence, policy and aggregate limits.

`interpretation_refs` and `knowledge_refs` each select at most sixteen exact retained records of their corresponding kind. New package stages, retained package evaluations, interpretations and ordinary knowledge together are bounded to sixteen evidence artifacts in this workflow adapter. `composition` contains exactly `{query_time,requirements,slots,allowed_guarantee_changes,counterfactuals,bounds}`. These caller fields are passed unchanged into native exploration; evidence arrays are assembled from the retained references and completed package stages. `prior_composition_ref` is null or one exact prior composition record for the optional before/after native reassessment.

Status and recover inputs are `{operation_id}`. A different requirement, capture, policy, time, source, root, TOPS or installation requires a new run identity. Recovery never changes these selections to the current time or newest evidence.

## Durable stages and exact replay

Run intent binds `{operation_id,root,tops_id,installation,request}` before a stage advances. Records use `symphony.qxctl.scv-composition-workflow-run.v1`, a separate `composition-runs-v1` journal namespace and the existing private no-follow storage and atomic publication primitives. Existing corpus/workflow records are not rewritten or silently migrated.

Stages are `pack:<evaluation_id>`, `explore` and optionally `reassess`. Each checkpoint retains `{input_ref,result_ref}` using the existing sealed payload and artifact-record formats. The owner input is durable before invocation; the validated immutable result is durable before checkpoint advancement. Interrupted pure work can be recomputed. A completed run can be replayed without reacquisition, changed policy or a new result selection.

Every completed stage must match the exact input mechanically reconstructed from the pinned request and references. Run/recovery additionally verify the installation and replay the result through its original native owner. A same-operation-ID/different-intent collision, changed installation, missing or altered object, wrong artifact kind or mismatched checkpoint fails explicitly. An interruption cannot produce a falsely complete run.

## Results and evidence limits

Result `symphony.qxctl.scv-composition-workflow-result.v1` contains `{protocol,run,status,validation,pack_evaluation_refs,composition_ref,reassessment_ref}`. Status is `prepared`, `evaluating_packs`, `explored` or `complete`; these describe workflow progress. `pack_evaluation_refs` maps newly requested evaluation IDs to completed artifact record digests. Input references remain visible in the retained request. Unavailable composition/reassessment references are null.

`validation=sealed_checkpoints_only` means observational inspection of sealed records and checkpoint correspondence, with no native semantic replay claim. Successful run/recovery uses `owner_replayed`. Completion means the requested stages completed; it does not mean all fixtures passed, every requirement is satisfied, a proposed implementation exists or the arrangement works in a deployed environment. Native obligations, source gaps, partiality, conflict and implementation status remain in the retained results.

Evidence obligations are inspectable work descriptions. Supplying or retaining a result does not authorize probes, follow links as commands, declare the obligation resolved, alter acceptance policy or execute a recipe. A later evidence change produces a new exact workflow and optional reassessment; original evidence remains available.

## Bounds and acceptance

The request is at most one MiB, records at most four MiB and the run has at most eighteen stages. The independent native byte/value/depth/capture/claim/search limits still apply after evidence assembly. Independent array maxima are not a promise that all maximum-sized bodies fit together. This workflow does not extend search beyond the existing finite candidate bound, implement a scheduler, prune history or add a graph database backend.

Acceptance covers installed schema discovery, source-independent CLI use, original-owner pack replay, successful exploration/reassessment, required/optional evidence distinctions, failed fixtures remaining visible, wrong-kind references, exact-intent collisions, interruption/recovery, checkpoint corruption, installation drift and observational status. Earlier exact packages and workflow behavior remain part of regression evidence. Source captures and independent semantic fixtures retain their provenance; a successful fixture does not certify general provider expertise.
