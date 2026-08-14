# SSIAG Protocol Schemas v1

These exact Draft 2020-12 schemas are canonical SSIAG authorization, grant-planning, protected local policy-administration, and provider-protocol truth.

- `authorization-request.schema.json` closes caller-declared operation, resource, audience, scope, correlation, freshness, and requested expiry.
- `authorization-decision.schema.json` closes the caller-neutral allow/deny result and safe policy/configuration evidence.
- `capability.schema.json` closes non-secret, non-transferable capability evidence. Possession is not bearer authority and `canonical_apply` is always false.
- `lifecycle-grant-plan.schema.json` closes deterministic twenty-six-operation proposal-only grant input for one exact TOPS/profile lifecycle boundary plus its domain-separated, separately permissioned per-TOPS profile-catalog read resource. `apply_enabled` and `canonical` are always false.
- `authorization-policy.schema.json` closes the exact deny-by-default local policy value.
- `policy-proposal-request.schema.json` and `policy-proposal.schema.json` close subject-free intent and the kernel-subject-bound digest proposal.
- `policy-apply-request.schema.json`, `policy-recovery-request.schema.json`, and `policy-result.schema.json` close compare-and-swap apply, explicit recovery, and metadata-only results.
- `policy-state.schema.json` and `policy-attempt.schema.json` close the protected generation state and crash-recovery journal.
- `provider-handshake.schema.json`, `provider-control-request.schema.json`, and `provider-control-response.schema.json` close the metadata-only, one-request/one-response provider subprocess protocol and its embedded safe error.
- `provider-executable-trust.schema.json` is the externally administered per-TOPS exact-version binding. One `provider-trust/<provider_name>.json` file, where `provider_name` is a safe token, lives under the protected SSIAG configuration tree; absence means `unbound`. It binds both adapter and foundation path, installation/executable digest, ownership, signing evidence, and exact provider protocol. Future qxctl binding mutation is a separate gate.
- `provider-trust-verification-request.schema.json` and `provider-trust-result.schema.json` close the safe qxctl-to-foundation request and exact read-only result shape. Capabilities and checks are explicit, sorted, unique arrays bounded to 128 entries.
- `provider-one-shot-channel.schema.json` closes only a synthetic inherited-file-descriptor descriptor for Phase 9 conformance. Its invalid descriptor `-1`, zero-byte limit, `synthetic: true`, and `operational: false` assertions prohibit opening it or treating it as secret delivery.

Provider control v1 is an exact protocol, not a version range. Each adapter process consumes exactly one JSON value no larger than 65,536 bytes and emits exactly one JSON value no larger than 65,536 bytes, then exits. The default deadline is five seconds and a caller may not request more than thirty seconds. Future incompatible majors install and bind side by side; neither the foundation nor qxctl selects the newest version automatically.

The control request binds request, correlation, TOPS, provider, adapter, operation, deadline, and exact foundation evidence. The adapter independently observes its parent executable and installed receipt/signature and compares that observation with the request; caller-supplied claims alone never prove trust. A successful control response proves mutual trust only when `handshake.foundation_trust.verified` is true and every observed path/digest/signing field matches the protected per-TOPS executable-trust declaration and request.

There is no persistent cancellation message or cancellation schema in v1. The one-request-per-process transport makes the Go invocation context authoritative: cancellation or deadline closes the child pipes, terminates the child, and waits for cleanup. A wire cancellation promise would be unobservable after a hung child stops reading and would incorrectly imply a persistent session.

Every `*_digest` in these provider schemas uses the same rule as the qxctl trust-result implementation: parse the closed object, omit its own digest member, serialize compact UTF-8 JSON with Go `encoding/json` semantics and recursive lexical object-key order, hash those bytes with SHA-256, and encode `sha256:` followed by 64 lowercase hexadecimal characters. Array order is significant. Capability and check arrays must therefore already be sorted and unique before hashing.

The schemas never accept caller class as a decision input. Subject identity comes from the authenticated local channel; target-host ownership or an exact granted permission is the gate. Policy apply mutates only an operational per-TOPS overlay and reports `canonical: false`; it cannot apply canonical repository truth. A grant plan remains non-mutating input.
