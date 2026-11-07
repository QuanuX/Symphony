# SRRAPIV maintenance and coverage contract

Owner: SRRAPIV. The implemented native catalog operations are pure validation, exact selection and impact assessment. Package mutation, vendor qualification, migrations and provider operation recovery remain separate runtime/installation contracts. This document does not claim that a supplied inventory or evidence reference was independently verified.

## Artifact and binding lifecycle

Source observation, candidate construction, installation, strategy rebinding, runtime activation and retirement are distinct actions. A new vendor release or knowledge observation does not change an existing binding. Existing strategy selections retain their exact archive, adapter, dependency, schema, connection, trust and interpretation identities.

Changes are assessed across SDK signatures and fields, callbacks and errors, operation semantics, native ABI/dependencies, platform, connection/trust profiles, authority references, feed and metadata interpretations, control/event schemas, state writers/readers and qualification scope. `binding_compare` reports every changed input path and added/removed declared capabilities without substituting a newer artifact or applying the change.

Bindings retain old SDK choices while the provider permits them. Local pinning cannot preserve a retired endpoint, expired certificate, withdrawn entitlement or server-side compatibility. An unchanged older client can become operationally unavailable. Current provider observations must not be inferred from a historical compatibility assertion.

Each candidate receives its own immutable identity, including backports or dependency/trust changes under an unchanged SDK release number. Secret value rotation at a stable authorized reference is separately governed and does not require changing the SDK. Changing the reference, scope or selected resource is an explicit binding change.

## Reader direction and rollback

State and journal readers are independent compatibility dimensions. The target runtime's declared reader schemas must cover the source writer schemas for a direct migration. The reverse is checked separately for rollback. A newer runtime accepting old state does not establish that the older runtime accepts new state.

Retain old decoder/build identities while retained data, unresolved attempts or active bindings reference them. A transformation produces a new derived artifact with provenance; it does not alter the original observation. A positive schema declaration still requires actual reader qualification before activation. Software rollback never rolls back exchange fills or automatically cancels orders. Uncertain consequential requests require operation-specific reconciliation, not replay based on absent completion records.

## Coverage assessment

`coverage_assess` accepts exactly:

```json
{
  "schema": "symphony.srrapiv.coverage.v1",
  "sdk_artifact_id": "sha256:<64-lowercase-hex-digits>",
  "universe": ["method:example", "callback:example", "field:example"],
  "mappings": []
}
```

The universe is a nonempty unique set of explicit public contract elements. It must include deprecated overloads, callbacks and their fields, configuration, utility and lifecycle contracts applicable to the selected SDK. The function assesses the supplied universe; header/AST extraction and independent review are required to establish that this universe actually covers the SDK. Its denominator cannot be inferred from a short list of familiar method names.

Each mapping has exactly `element_id`, `disposition`, `handler_id`, `operation_id`, `qxctl_route`, `input_schema`, `result_schema`, `event_schemas`, and `evidence_ids`. An optional `equivalence_evidence_id` is required for `disposition:equivalent`. Dispositions are `implemented`, `equivalent`, `unresolved`, or `provider_unavailable`.

For `implemented` or `equivalent`, handler, operation, command route and input/result schemas must be nonempty, and at least one evidence reference is required. Event schemas may be empty only when the mapped element has no event route; the pure assessor cannot establish that semantic fact. Equivalent adaptations require a separate equivalence evidence reference. For unresolved/unavailable elements the corresponding unimplemented handler/route/schema fields may be null. Unknown fields, duplicate mappings and elements outside the supplied universe fail validation.

Missing mappings remain `unmapped` and count against completeness. Provider-unavailable and unresolved records also remain in the denominator. The result contains `total_elements`, `mapped_elements`, all unresolved records and `complete_for_supplied_universe`. It always reports `sdk_universe_independently_verified:false`, `implementation_evidence_verified:false` and `live_qualified:false`. A syntactically complete mapping is not an operational feature claim, registered dispatch proof, or vendor conformance result.

The same input-size and identity bounds from `BINDINGS.md` apply, with up to 16,384 supplied universe/mapping records per request. A coverage report returns the exact supplied-universe digest so evidence cannot silently change its denominator.

## Qualification and maintenance practice

Focused tests accompany changed fields, semantics, readers and affected contracts. Exact-host installation, actual native SDK behavior, runtime fault scenarios, qxctl dispatch parity, provider Test acceptance and production qualification remain separate evidence classes. Supplied claims cannot authorize live use.

Candidate comparisons must not duplicate consequential provider requests for behavioral testing. Concurrent SDK versions require explicit session-capacity and shared-account consideration: separate processes do not isolate provider quotas, positions, PnL or account-wide cancellation effects. Read-only queries and subscriptions still have provider-specific concurrency, entitlement and pacing limits.

Retirement plans inspect active bindings, installed/running instances, pending or outcome-unknown attempts and retained state requiring old readers. Deprecation, local uninstall, artifact deletion, historical-reader removal and credential erasure remain separate decisions. A newer package is not sufficient reason to remove an older selection.

Vendor-maintenance evidence should cover supported client/server relationships, platform/ABI policy, historical package access, notices and retirement horizons, trust/dependency rotation, session limits, replay/identity guarantees, distribution rights and the scope of repeated conformance after changes. Unanswered questions stay unknown; the catalog does not manufacture support policy.
