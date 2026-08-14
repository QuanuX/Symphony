# SSIAG macOS Keychain Provider Manifest

## Module Identity

- **name**: `ssiag-provider-macos-keychain`
- **path**: `modules/ssiag-provider-macos-keychain/`
- **language**: Swift 6
- **platform**: macOS 13 or later
- **binary**: `symphony-ssiag-provider-macos-keychain`
- **protocol**: exact `symphony.ssiag.provider.v1`
- **status**: metadata-only scaffold under an owner-ratified operational architecture

## Owned Capabilities

- independent user or system installation lifecycle;
- one bounded standard-input/output JSON request and one response per process;
- provider version, status, and capability metadata;
- source-owned stable engine-operation identities for the exact `capabilities`, `handshake`, and `status` metadata operations;
- independent parent-foundation path, ownership, receipt, and executable-digest comparison, with undeclared signing identity failing closed;
- rejection of unknown fields and all credential operations.

## Prohibited Claims

The scaffold MUST NOT claim Keychain readiness, credential access, signing, decryption, assertion, export, rotation, or lease delivery. Its inherited-file-descriptor descriptor is synthetic and MUST NOT be opened. It MUST NOT become an implicit fallback or receive secret values through arguments, environment variables, logs, qxctl, or the control channel.

Future operational behavior MUST remain per-user and session-aware, authenticate the invoking SSIAG executable from independently observed parent and installation evidence, and add signing verification only after its exact policy is ratified. It must prefer non-exportable operations, keep secret bytes out of the JSON control envelope, and report unavailable in system/headless scope without fallback. A caller-provided path, digest, or signing label never proves foundation identity by itself.

## Dependency Boundary

The module uses Swift and Apple system frameworks only. It is optional and excluded from non-macOS builds. It does not introduce Swift, Objective-C, cgo, or Apple linking into `modules/secure-identity-access-governance/`.

## Contract Files

- `INTENT.md`
- `MANIFEST.md`
- `INSTALL.md`
- `SKILL.md`
- `SPEC.md`

## Independent Lifecycle

Each exact installed binary and its immutable receipt-v2 record are owned by this module. Uninstall validates every receipt-owned byte and does not remove other adapter versions, legacy v1 evidence, SSIAG foundation binaries, TOPS configuration, STAV data, or Keychain items.
