# SSIAG macOS Keychain Provider Specification

## Status

Metadata-only provider and Phase 10B signed-bundle readiness foundation. Apple Keychain access remains disabled until the separate Phase 10C platform gates pass.

## Executable Boundary

The adapter MUST remain a separately built Swift executable and MUST NOT be linked into the Go SSIAG foundation. It MUST open no TCP listener, run no background daemon, and make no network request.

## Scaffold IPC

`serve` is a one-invocation process. It reads exactly one strict `symphony.ssiag.provider-control-request.v1` JSON value from standard input, limited to 65,536 bytes, and emits exactly one strict `symphony.ssiag.provider-control-response.v1` JSON value to standard output, also limited to 65,536 bytes. It then closes and exits. The only v1 operations are `capabilities`, `handshake`, and `status`, owned in source as `engop:symphony:ssiag.provider.metadata-capabilities`, `engop:symphony:ssiag.provider.metadata-handshake`, and `engop:symphony:ssiag.provider.metadata-status`. The default timeout is five seconds and the maximum requested timeout is thirty seconds. Unknown or missing fields, unknown protocols, unowned operation names, extra values/output, oversized or malformed JSON, expired deadlines, and credential operations fail closed.

Every exchange binds the exact request ID, correlation ID, TOPS ID, provider name, adapter identifier, operation, and deadline. The request includes the protected foundation path, installation/executable digests, and signing identity. The adapter independently observes its parent process path and validates its installed receipt, ownership, permissions, and executable digest. Phase 9 accepts signing identity only as `not_applicable`; any named identity fails closed until an independently verifiable signing policy is ratified. A caller assertion alone never proves trust. Even a `succeeded` response is not mutual executable trust unless `handshake.foundation_trust.verified` is true and every protected field matches exactly. Errors contain only stable reason codes, safe categories, retryability, and explicit false assertions for native detail and secret content.

Provider control v1 remains byte- and behavior-compatible. `readiness` is a separate fixed process operation that takes no payload and emits exactly one strict `symphony.ssiag.provider-readiness-observation.v1` value. It uses Apple Security only for complete structural signature validation, native evaluation of the receipt-owned `ssiag-signing-policy.json` requirement, and security-session capability observation. Structural validation, policy match, and operational eligibility are distinct. Eligibility is unevaluated and disabled in Phase 10B. The output excludes requirement text, certificates, profile content, entitlements, native errors, security-session identity, provider payloads, and secrets.

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

Installation is host-level and independent of TOPS enrollment. A legacy bare executable remains supported for provider-v1 compatibility but can never satisfy production readiness. Production installs the complete `SymphonySSIAGMacOSKeychainProvider.app` bundle into an immutable versioned prefix. Receipt v2 owns a sorted exact set containing required `Contents/Info.plist` and `Contents/MacOS/symphony-ssiag-provider-macos-keychain`, plus only the optional signing policy, embedded provisioning profile, and code-resource envelope named by the canonical readiness contract. The receipt is committed last and records every file's exact path, kind, size, and SHA-256 digest. Symlinks, unknown files, unsafe modes, duplicates, changed bytes, and partial ownership fail closed.

There is no active alias or newest-version selection. Multiple exact versions may coexist. Reinstalling identical bytes is idempotent; different, changed, or unreceipted bytes at the same version fail closed. The legacy `--force` flag cannot replace or remove receipt-owned bytes. Uninstall validates all present owned bytes, rejects unknown bundle entries, removes files before its receipt, and treats the retained receipt as recovery evidence if interruption occurs between removals. Only absent previously verified files self-heal; changed bytes never do. A completed uninstall is idempotent. Pre-receipt-v2 custom manifests remain legacy evidence and never satisfy provider trust.

Use by one TOPS additionally requires one externally administered `provider-executable-trust.v1` declaration at `provider-trust/<provider_name>.json` in that TOPS's protected SSIAG configuration tree. `provider_name` must be a safe token and must equal the object's identity. The declaration binds the exact adapter and foundation paths, versions, installation/executable digests, ownership, signing evidence, scope, and provider protocol. Absence reports `unbound`. Multiple installed versions may coexist, but no component may scan for or select the newest as authority. Future qxctl mutation of this binding is a separate gate.

## Phase 10B Production Readiness Profile

A production artifact is an app-like bundle signed with Developer ID, hardened runtime, secure timestamp, and notarization. Exact signing identifier, Team ID, bundle identifier, private application-identifier access group, entitlements, provisioning requirement, and native requirement are build and protected installation evidence; none is a Symphony hardcode or caller input. A provisioning profile is embedded only when selected entitlements require it. `get-task-allow` and unrelated runtime exceptions are prohibited by the production requirement.

The Go foundation validates the entire receipt-owned package, copies every owned file into one private directory while rechecking size and digest, preserves bundle layout, and executes only the staged entry point. Native signature policy is compiled and evaluated with `SecRequirement`; a digest is evidence, not a substitute. Unsigned, ad-hoc, bare, or policy-absent artifacts remain metadata-only unless a separately named nonproduction profile is ratified.

## Future Operational Contract

An operational version MUST use Apple Security/Keychain APIs only inside this process and MUST target the data-protection Keychain from an actual user-login context. It remains session-aware. A file-based system Keychain is a separate future provider, not fallback behavior.

It MUST add no secret-valued CLI arguments or environment variables. It MUST authenticate the invoking foundation under the ratified path, ownership, and code-signing policy; verify protocol compatibility; constrain items by immutable TOPS ID; disable synchronization by default; honor the most restrictive usable accessibility and user-presence policy; bound messages and time; sanitize errors; and fail closed.

The JSON control channel MUST NOT carry secret bytes. Non-exportable sign/assert/decrypt operations remain inside the adapter. The Phase 9 inherited-file-descriptor object is synthetic (descriptor `-1`, zero-byte limit, `synthetic: true`, `operational: false`) and MUST NOT be opened or used for delivery. Any future explicitly policy-authorized export requires a separately ratified operational request-bound, bounded, one-shot protected channel that closes after delivery and never reaches qxctl, OpenAPI, STAV, arguments, environment variables, logs, or examples.

The first item namespace will use the exact private application-identifier access group of the selected production bundle. Exact item classes, operation names, accessibility and access-control matrix, operational secret-channel framing, and memory/crash policy remain Phase 10C or later gates.
