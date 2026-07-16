# SSIAG macOS Keychain Provider Manifest

## Module Identity

- **name**: `ssiag-provider-macos-keychain`
- **path**: `modules/ssiag-provider-macos-keychain/`
- **language**: Swift 6
- **platform**: macOS 13 or later
- **binary**: `symphony-ssiag-provider-macos-keychain`
- **protocol**: `symphony.ssiag.provider.v1`
- **status**: metadata-only scaffold under an owner-ratified operational architecture

## Owned Capabilities

- independent user or system installation lifecycle;
- bounded standard-input/output JSON-lines transport;
- provider version, status, and capability metadata;
- rejection of unknown fields and all credential operations.

## Prohibited Claims

The scaffold MUST NOT claim Keychain readiness, credential access, signing, decryption, assertion, export, rotation, or lease delivery. It MUST NOT become an implicit fallback or receive secret values through arguments, environment variables, logs, or qxctl.

Future operational behavior MUST remain per-user and session-aware, authenticate the invoking SSIAG executable, prefer non-exportable operations, keep secret bytes out of the JSON control envelope, and report unavailable in system/headless scope without fallback.

## Dependency Boundary

The module uses Swift and Apple system frameworks only. It is optional and excluded from non-macOS builds. It does not introduce Swift, Objective-C, cgo, or Apple linking into `modules/secure-identity-access-governance/`.

## Contract Files

- `INTENT.md`
- `MANIFEST.md`
- `INSTALL.md`
- `SKILL.md`
- `SPEC.md`

## Independent Lifecycle

The installed binary and its manifest are owned by this module. Uninstall validates the recorded binary digest and does not remove SSIAG foundation binaries, TOPS configuration, STAV data, or Keychain items.
