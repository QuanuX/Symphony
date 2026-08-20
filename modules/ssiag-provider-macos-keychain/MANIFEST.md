# SSIAG macOS Keychain Provider Manifest

## Module Identity

- **name**: `ssiag-provider-macos-keychain`
- **path**: `modules/ssiag-provider-macos-keychain/`
- **language**: Swift 6
- **platform**: macOS 13 or later
- **binary**: `symphony-ssiag-provider-macos-keychain`
- **protocol**: exact `symphony.ssiag.provider.v1`
- **status**: metadata-only provider and signed-bundle readiness foundation; Keychain operations disabled

## Owned Capabilities

- independent user or system installation lifecycle;
- one bounded standard-input/output JSON request and one response per process;
- provider version, status, and capability metadata;
- source-owned stable engine-operation identities for the exact `capabilities`, `handshake`, and `status` metadata operations;
- independent parent-foundation path, ownership, receipt, and executable-digest comparison, with undeclared signing identity failing closed;
- exact legacy-binary or complete app-bundle receipt-v2 installation, idempotent removal, and interruption recovery;
- separate native structural-signature, receipt-owned `SecRequirement`, and security-session readiness observation;
- source-owned operation `engop:symphony:ssiag.macos-keychain-provider.readiness.observe` with every operational flag false;
- rejection of unknown fields and all credential operations.

## Prohibited Claims

The scaffold MUST NOT claim Keychain readiness, credential access, signing, decryption, assertion, export, rotation, or lease delivery. Its inherited-file-descriptor descriptor is synthetic and MUST NOT be opened. It MUST NOT become an implicit fallback or receive secret values through arguments, environment variables, logs, qxctl, or the control channel.

Future operational behavior MUST target the per-user data-protection Keychain, remain session-aware, and authenticate the invoking SSIAG executable from independently observed parent and installation evidence. The implemented readiness check never substitutes for operational authorization. It must prefer non-exportable operations, keep secret bytes out of the JSON control envelope, and report actual unavailable capability without fallback. A caller-provided path, digest, requirement, or signing label never proves identity by itself.

## Dependency Boundary

The module uses Swift and Apple system frameworks only. It is optional and excluded from non-macOS builds. It does not introduce Swift, Objective-C, cgo, or Apple linking into `modules/secure-identity-access-governance/`.

## Contract Files

- `INTENT.md`
- `MANIFEST.md`
- `INSTALL.md`
- `SKILL.md`
- `SPEC.md`

## Independent Lifecycle

Each exact legacy binary or complete app-like bundle and its immutable receipt-v2 record are owned by this module. Uninstall validates every present receipt-owned byte, rejects unknown entries, uses the retained receipt to finish interrupted removal, and does not remove other adapter versions, legacy v1 evidence, SSIAG foundation binaries, TOPS configuration, STAV data, or Keychain items.
