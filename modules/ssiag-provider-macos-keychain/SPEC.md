# SSIAG macOS Keychain Provider Specification

## Status

Metadata-only scaffold. The operational architecture is owner-ratified, but Apple Keychain access remains disabled until its exact platform gates pass.

## Executable Boundary

The adapter MUST remain a separately built Swift executable and MUST NOT be linked into the Go SSIAG foundation. It MUST open no TCP listener, run no background daemon, and make no network request.

## Scaffold IPC

`serve` reads one JSON object per newline from standard input. Each input line is limited to 65,536 bytes and contains exactly:

- `schema`: `symphony.ssiag.provider.request.v1`;
- `request_id`: a bounded safe identifier;
- `operation`: `hello`, `status`, or `capabilities`.

Unknown or missing fields, unknown schemas, oversized input, malformed JSON, and credential operations fail closed. Responses use `symphony.ssiag.provider.response.v1` and contain only constant provider metadata and the request ID. Native errors and input values are never echoed.

## Descriptor Truth

The scaffold descriptor MUST report:

- protocol and adapter version;
- platform `macos`;
- transport `stdio-jsonl`;
- capabilities `metadata` and `capability-discovery` only;
- status `declared_not_operational`;
- `operational_access_enabled: false`.

## Installation Contract

Installation is host-level and independent of TOPS enrollment. It atomically copies the running executable, records its SHA-256 digest and exact path, rejects symlink/non-regular targets, and requires explicit force to replace a changed binary. Uninstall validates the manifest and digest and removes only owned host-level files.

## Future Operational Contract

An operational version MUST use Apple Security/Keychain APIs only inside this process and MUST run per-user in a session-aware topology. System/headless scope MUST report the provider unavailable and MUST NOT fall back.

It MUST add no secret-valued CLI arguments or environment variables. It MUST authenticate the invoking foundation under the ratified path, ownership, and code-signing policy; verify protocol compatibility; constrain items by immutable TOPS ID; disable synchronization by default; honor the most restrictive usable accessibility and user-presence policy; bound messages and time; sanitize errors; and fail closed.

The JSON-lines control channel MUST NOT carry secret bytes. Non-exportable sign/assert/decrypt operations remain inside the adapter. Any explicitly policy-authorized export MUST use a request-bound, bounded, one-shot protected descriptor or equivalent local channel that closes after delivery and never reaches qxctl, OpenAPI, STAV, arguments, environment variables, logs, or examples.

Exact Keychain namespaces, item classes, operation names, access-control matrix, signing requirements, entitlements, notarization, provisioning, secret-channel framing, and memory/crash policy remain implementation gates.
