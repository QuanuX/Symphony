# SSIAG macOS Keychain Provider Skill

## Safe Workflow

1. Read `knowledge/ssiag/SPEC.md` and this module's `SPEC.md`.
2. Run `swift test` before building.
3. Confirm capability output remains metadata-only, sorted, unique, and bounded to 128 entries unless an operational gate has been ratified.
4. Install explicitly by scope.
5. Configure the provider as enabled with capabilities `capability-discovery` and `metadata`, `exportable: false`, and `interactive: true`.
6. Bind the exact installed adapter and foundation versions with one protected per-TOPS `provider-executable-trust.v1` declaration; absence is `unbound`, and automatic newest-version selection is prohibited.
7. Confirm one process consumes one request and emits one response, each at most 65,536 bytes, with five-second default and thirty-second maximum deadline.
8. Require adapter-side independent parent/receipt observation; signing identity remains `not_applicable` until a separate verifier is ratified. Never treat request assertions or a successful outcome alone as mutual trust.
9. Verify version, protocol, binary permissions, receipt digest, bounded IPC, and fail-closed behavior.

## Current Capability Boundary

Any caller may inspect metadata, build, test, and propose configuration within its effective target-host permission. Caller type does not affect capability. Operational Keychain access is disabled for every caller, so no supported operation may request Keychain values, place secret values in IPC fixtures, open the synthetic one-shot descriptor, use `security` CLI as a hidden fallback, weaken prompts or access controls, add silent provider fallback, or claim operational access from the scaffold.

## Ratified Architecture

The provider will be per-user and session-aware, mutually authenticate the SSIAG executable, keep control metadata separate from secret bytes, prefer non-exportable operations, and fail unavailable in system/headless scope without fallback.

## Remaining Operational Gates

Before importing Apple's Security framework or enabling a credential operation, record and verify:

- exact Keychain operations and item classes;
- access-control and user-presence requirements;
- code-signing, entitlements, notarization, and distribution;
- exact SSIAG and adapter signing/path/ownership requirements;
- one-shot secret-channel framing, size, lifetime, memory, and crash-dump handling;
- timeouts, cancellation, interaction, replay, concurrency, and error sanitization;
- STAV safe-event mappings and explicit exclusions.
