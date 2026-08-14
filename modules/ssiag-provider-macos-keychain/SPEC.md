# SSIAG macOS Keychain Provider Specification

## Status

Metadata-only scaffold. The operational architecture is owner-ratified, but Apple Keychain access remains disabled until its exact platform gates pass.

## Executable Boundary

The adapter MUST remain a separately built Swift executable and MUST NOT be linked into the Go SSIAG foundation. It MUST open no TCP listener, run no background daemon, and make no network request.

## Scaffold IPC

`serve` is a one-invocation process. It reads exactly one strict `symphony.ssiag.provider-control-request.v1` JSON value from standard input, limited to 65,536 bytes, and emits exactly one strict `symphony.ssiag.provider-control-response.v1` JSON value to standard output, also limited to 65,536 bytes. It then closes and exits. The only v1 operations are `capabilities`, `handshake`, and `status`, owned in source as `engop:symphony:ssiag.provider.metadata-capabilities`, `engop:symphony:ssiag.provider.metadata-handshake`, and `engop:symphony:ssiag.provider.metadata-status`. The default timeout is five seconds and the maximum requested timeout is thirty seconds. Unknown or missing fields, unknown protocols, unowned operation names, extra values/output, oversized or malformed JSON, expired deadlines, and credential operations fail closed.

Every exchange binds the exact request ID, correlation ID, TOPS ID, provider name, adapter identifier, operation, and deadline. The request includes the protected foundation path, installation/executable digests, and signing identity. The adapter independently observes its parent process path and validates its installed receipt, ownership, permissions, and executable digest. Phase 9 accepts signing identity only as `not_applicable`; any named identity fails closed until an independently verifiable signing policy is ratified. A caller assertion alone never proves trust. Even a `succeeded` response is not mutual executable trust unless `handshake.foundation_trust.verified` is true and every protected field matches exactly. Errors contain only stable reason codes, safe categories, retryability, and explicit false assertions for native detail and secret content.

There is no cancellation frame or cancellation schema. Because one exchange is one process, the Go invocation context enforces cancellation and deadline by closing pipes, terminating the adapter, and waiting for cleanup; a hung adapter cannot be trusted to read another message.

## Descriptor Truth

The scaffold descriptor MUST report:

- protocol and adapter version;
- platform `macos`;
- transport `stdio-one-shot-json`;
- capabilities `metadata` and `capability-discovery` only;
- `exportable: false` and `interactive: true` as truthful provider metadata, without enabling an operation;
- one request and response per process, 65,536-byte bounds, five-second default and thirty-second maximum timeout, and 128-entry capability/check bounds;
- one of the canonical statuses `declared`, `ready`, `degraded`, `locked`, `unavailable`, or `disabled` (the metadata-only adapter reports `disabled` until trust is usable);
- `operational_access_enabled: false`, `provider_operations_enabled: false`, and `secret_channel_enabled: false`.

## Installation Contract

Installation is host-level and independent of TOPS enrollment. It installs the running executable into an immutable versioned prefix, commits a strict `symphony.knowledge.install-receipt.v2` last, records exact path/size/SHA-256 evidence, rejects symlink and non-regular targets, and never creates an active alias or chooses a newest version. Multiple exact versions may coexist. Reinstalling identical bytes is idempotent; different, changed, or unreceipted bytes at the same version fail closed. The legacy `--force` flag cannot replace or remove receipt-owned bytes. Uninstall validates the receipt and every present owned byte, removes the binary before its receipt, and treats that retained receipt as recovery evidence if interruption occurs between removals. A completed uninstall is idempotent. Pre-receipt-v2 custom manifests remain legacy evidence and never satisfy provider trust.

Use by one TOPS additionally requires one externally administered `provider-executable-trust.v1` declaration at `provider-trust/<provider_name>.json` in that TOPS's protected SSIAG configuration tree. `provider_name` must be a safe token and must equal the object's identity. The declaration binds the exact adapter and foundation paths, versions, installation/executable digests, ownership, signing evidence, scope, and provider protocol. Absence reports `unbound`. Multiple installed versions may coexist, but no component may scan for or select the newest as authority. Future qxctl mutation of this binding is a separate gate.

## Future Operational Contract

An operational version MUST use Apple Security/Keychain APIs only inside this process and MUST run per-user in a session-aware topology. System/headless scope MUST report the provider unavailable and MUST NOT fall back.

It MUST add no secret-valued CLI arguments or environment variables. It MUST authenticate the invoking foundation under the ratified path, ownership, and code-signing policy; verify protocol compatibility; constrain items by immutable TOPS ID; disable synchronization by default; honor the most restrictive usable accessibility and user-presence policy; bound messages and time; sanitize errors; and fail closed.

The JSON control channel MUST NOT carry secret bytes. Non-exportable sign/assert/decrypt operations remain inside the adapter. The Phase 9 inherited-file-descriptor object is synthetic (descriptor `-1`, zero-byte limit, `synthetic: true`, `operational: false`) and MUST NOT be opened or used for delivery. Any future explicitly policy-authorized export requires a separately ratified operational request-bound, bounded, one-shot protected channel that closes after delivery and never reaches qxctl, OpenAPI, STAV, arguments, environment variables, logs, or examples.

Exact Keychain namespaces, item classes, operation names, access-control matrix, signing requirements, entitlements, notarization, provisioning, operational secret-channel framing, and memory/crash policy remain implementation gates.
