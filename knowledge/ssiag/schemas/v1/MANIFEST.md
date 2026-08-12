# SSIAG Protocol Schemas v1

These exact Draft 2020-12 schemas are canonical SSIAG authorization and grant-planning protocol truth.

- `authorization-request.schema.json` closes caller-declared operation, resource, audience, scope, correlation, freshness, and requested expiry.
- `authorization-decision.schema.json` closes the caller-neutral allow/deny result and safe policy/configuration evidence.
- `capability.schema.json` closes non-secret, non-transferable capability evidence. Possession is not bearer authority and `canonical_apply` is always false.
- `lifecycle-grant-plan.schema.json` closes deterministic proposal-only grant input for one exact TOPS/profile lifecycle boundary plus its domain-separated, separately permissioned per-TOPS profile-catalog read resource. `apply_enabled` and `canonical` are always false.

The schemas never accept caller class as a decision input. Canonical subject identity comes from the authenticated local channel; host ownership or an exact granted permission is the gate. A grant plan is not a decision and does not mutate policy.
