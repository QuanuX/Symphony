# SSIAG macOS Keychain Provider Skill

## Safe Workflow

1. Read `knowledge/ssiag/SPEC.md` and this module's `SPEC.md`.
2. Run `swift test` before building.
3. Confirm capability output remains metadata-only unless an operational gate has been ratified.
4. Install explicitly by scope.
5. Configure the SSIAG foundation to invoke the exact installed binary only after provider compatibility support exists.
6. Verify version, protocol, binary permissions, manifest digest, bounded IPC, and fail-closed behavior.

## Agent Restrictions

Agents may inspect metadata, build, test, and propose configuration. They must not request Keychain values, type secret values into IPC fixtures, use `security` CLI as a hidden fallback, weaken prompts or access controls, add silent provider fallback, or claim operational Keychain access from the scaffold.

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
