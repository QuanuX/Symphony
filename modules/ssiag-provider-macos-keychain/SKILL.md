# SSIAG macOS Keychain Provider Skill

## Safe Workflow

1. Read `knowledge/ssiag/SPEC.md` and this module's `SPEC.md`.
2. Run `swift test` before building.
3. Confirm capability output remains metadata-only, sorted, unique, and bounded to 128 entries unless an operational gate has been ratified.
4. For Phase 10B readiness, require the exact receipt-owned app bundle and policy, inspect structural, policy, and session layers independently through `qxctl ssiag provider readiness`, and confirm every operational flag remains false. Never treat local signature validity, a requirement digest, or session presence as authorization.
5. Install explicitly by scope.
6. Configure the provider as enabled with capabilities `capability-discovery` and `metadata`, `exportable: false`, and `interactive: true`.
7. Bind the exact installed adapter and foundation versions with one protected per-TOPS `provider-executable-trust.v1` declaration; absence is `unbound`, and automatic newest-version selection is prohibited.
8. Confirm one process consumes one request and emits one response, each at most 65,536 bytes, with five-second default and thirty-second maximum deadline.
9. Require adapter-side independent parent/receipt observation and configured native code-requirement evaluation. Never treat request assertions or a successful outcome alone as mutual trust or authorization.
10. Verify version, protocol, binary permissions, complete receipt ownership, bounded IPC, and fail-closed behavior.

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
