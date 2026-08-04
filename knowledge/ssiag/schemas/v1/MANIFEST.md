# SSIAG Protocol Schemas v1

These exact Draft 2020-12 schemas are canonical SSIAG authorization protocol truth.

- `authorization-request.schema.json` closes caller-declared operation, resource, audience, scope, correlation, freshness, and requested expiry.
- `authorization-decision.schema.json` closes the caller-neutral allow/deny result and safe policy/configuration evidence.
- `capability.schema.json` closes non-secret, non-transferable capability evidence. Possession is not bearer authority and `canonical_apply` is always false.

The schemas never accept caller class as a decision input. Canonical subject identity comes from the authenticated local channel; host ownership or an exact granted permission is the gate.
