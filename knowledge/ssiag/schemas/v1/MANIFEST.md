# SSIAG Protocol Schemas v1

These exact Draft 2020-12 schemas are canonical SSIAG authorization, grant-planning, and protected local policy-administration truth.

- `authorization-request.schema.json` closes caller-declared operation, resource, audience, scope, correlation, freshness, and requested expiry.
- `authorization-decision.schema.json` closes the caller-neutral allow/deny result and safe policy/configuration evidence.
- `capability.schema.json` closes non-secret, non-transferable capability evidence. Possession is not bearer authority and `canonical_apply` is always false.
- `lifecycle-grant-plan.schema.json` closes deterministic proposal-only grant input for one exact TOPS/profile lifecycle boundary plus its domain-separated, separately permissioned per-TOPS profile-catalog read resource. `apply_enabled` and `canonical` are always false.
- `authorization-policy.schema.json` closes the exact deny-by-default local policy value.
- `policy-proposal-request.schema.json` and `policy-proposal.schema.json` close subject-free intent and the kernel-subject-bound digest proposal.
- `policy-apply-request.schema.json`, `policy-recovery-request.schema.json`, and `policy-result.schema.json` close compare-and-swap apply, explicit recovery, and metadata-only results.
- `policy-state.schema.json` and `policy-attempt.schema.json` close the protected generation state and crash-recovery journal.

The schemas never accept caller class as a decision input. Subject identity comes from the authenticated local channel; target-host ownership or an exact granted permission is the gate. Policy apply mutates only an operational per-TOPS overlay and reports `canonical: false`; it cannot apply canonical repository truth. A grant plan remains non-mutating input.
