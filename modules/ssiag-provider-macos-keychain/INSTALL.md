# SSIAG macOS Keychain Provider Installation

## Prerequisites

- macOS 13 or later;
- Swift 6 toolchain;
- no SSIAG foundation rebuild is required.

## Build and Test

```bash
cd modules/ssiag-provider-macos-keychain
swift test
swift build -c release
```

The bare SwiftPM executable is a development/compatibility artifact and remains metadata-only. A production build must use `scripts/build-production-bundle.sh` with Architect-approved bundle identity, native requirement, entitlements, signing identity, and optional provisioning/notarization configuration. The script creates the complete app-like bundle before signing; no file may be added after signing or notarization.

## User Installation

```bash
.build/release/symphony-ssiag-provider-macos-keychain install --scope user
```

When invoked from the bare development executable, this installs the legacy-compatible metadata-only path:

- binary: `~/.local/libexec/symphony/ssiag-macos-keychain-provider/<version>/symphony-ssiag-provider-macos-keychain`
- receipt: `~/.local/share/symphony/receipts/ssiag-macos-keychain-provider/<version>/install-receipt.json`

Both paths are immutable and receipt-v2 owned. There is no active alias and no newest-version discovery. An administrator may choose another absolute, non-root installation prefix with `--prefix`; the default is only a convenience.

When invoked from `SymphonySSIAGMacOSKeychainProvider.app/Contents/MacOS/symphony-ssiag-provider-macos-keychain`, the same command installs the complete bundle at:

- bundle: `~/.local/libexec/symphony/ssiag-macos-keychain-provider/<version>/SymphonySSIAGMacOSKeychainProvider.app`
- entry point: `.../Contents/MacOS/symphony-ssiag-provider-macos-keychain`
- receipt: `~/.local/share/symphony/receipts/ssiag-macos-keychain-provider/<version>/install-receipt.json`

The receipt owns every admitted bundle file. Unknown or changed bytes prevent reinstall and uninstall. An incomplete uninstall heals only by removing remaining unchanged receipt-owned files and finally the retained receipt.

## System Installation

Run the same executable under an owner-approved privilege boundary:

```bash
.build/release/symphony-ssiag-provider-macos-keychain install --scope system
```

This uses the same versioned receipt-v2 topology under `/usr/local`. A different absolute, non-root prefix may be supplied explicitly with `--prefix`.

## Verification

Installation alone does not enable or select the provider. Before verification, an administrator acting through the target host's permission boundary must add an exact provider declaration to that TOPS `config.json`:

```json
"providers": [
  {
    "name": "macos-keychain",
    "kind": "macos-keychain",
    "enabled": true,
    "capabilities": ["capability-discovery", "metadata"],
    "exportable": false,
    "interactive": true
  }
]
```

The provider name is an administrator-selected safe token. The same exact value must name the trust file, the trust object's `provider_name`, and the qxctl argument. `interactive: true` is required because the macOS adapter truthfully declares that future Keychain interaction is possible even though Phase 9 enables no provider operation. `exportable` must remain false.

The administrator must also provision one strict `symphony.ssiag.provider-executable-trust.v1` object at:

- user: `${XDG_CONFIG_HOME:-$HOME/.config}/symphony/<tops-id>/ssiag/provider-trust/<provider-name>.json` with mode `0600`;
- system: `/etc/symphony/<tops-id>/ssiag/provider-trust/<provider-name>.json`, root-owned and not group/other writable.

The object must validate against `knowledge/ssiag/schemas/v1/provider-executable-trust.schema.json`. It binds the exact TOPS ID and scope, provider name/kind, adapter ID `adapter:symphony:ssiag.macos-keychain-provider.v1`, adapter version, provider protocol, adapter executable path, receipt/executable digests, owner UID/GID and mode, and the exact receipt-v2-installed SSIAG foundation path, receipt/executable digests, owner UID/GID, and signing policy. All three operational flags remain false. `declaration_digest` is the SHA-256 digest of the compact Go `encoding/json` representation after omitting `declaration_digest`. Phase 9 accepts both signing identities only as `not_applicable`; a named signing label is not proof and fails closed.

qxctl intentionally does not mutate this binding in Phase 9. Until an external administrator creates the exact protected declaration, `provider show` reports `unbound` and `provider verify` cannot report mutual trust. The SSIAG foundation itself must run from its immutable receipt-v2 package; a development or historical fixed-path foundation has `foundation_installation_digest: not_applicable` and the adapter rejects it.

After those prerequisites, restart the selected SSIAG instance and verify only through the foundation:

```bash
qxctl ssiag provider show macos-keychain --scope user --tops-id "$TOPS_ID" --json
qxctl ssiag provider verify macos-keychain --scope user --tops-id "$TOPS_ID" \
  --authority-basis host_owner --json
qxctl ssiag provider readiness macos-keychain --scope user --tops-id "$TOPS_ID" \
  --authority-basis host_owner --json
```

Direct `status` and `capabilities` adapter CLI bypasses are intentionally absent. `serve` accepts only one fully bound, digest-bearing control request from its invoking SSIAG parent and emits at most one response before exit.

Verification and readiness MUST report all operational flags false. Readiness separately reports structural validity, native receipt-owned policy match, and security-session capability; operational eligibility remains disabled. Operational Keychain access remains gated until Phase 10C's item, access-control, audit, recovery, and negative-security contracts are implemented and verified.

## Upgrade

Build the reviewed new version and install it beside the old version. Receipt-owned versions are immutable: `--force` is accepted only for legacy invocation compatibility and never authorizes replacement. Change the protected per-TOPS binding explicitly after verification. Retain or uninstall the old version according to administrator policy; no component silently changes the binding.

## Uninstall

```bash
ADAPTER_VERSION='0.1.0-draft' # use the exact version returned by install
"$HOME/.local/libexec/symphony/ssiag-macos-keychain-provider/$ADAPTER_VERSION/symphony-ssiag-provider-macos-keychain" \
  uninstall --scope user
```

For a production bundle, invoke the exact versioned entry point under `SymphonySSIAGMacOSKeychainProvider.app/Contents/MacOS/` with the same arguments.

For system scope, invoke the exact versioned executable below `/usr/local/libexec`, pass `--scope system`, and repeat any explicit `--prefix`. There is no unversioned alias. Uninstall validates the receipt and every present owned byte and refuses changed or unreceipted paths even when `--force` is present. It removes only that exact version's receipt-owned adapter binary and then its receipt. If interruption occurs after binary removal, the retained receipt is recovery evidence and the same command finishes cleanup on retry; replay after completed removal is a no-op. It does not delete legacy v1 evidence, other installed versions, Keychain items, TOPS bindings, or any TOPS state.

The pre-receipt-v2 custom binary and `install.json` locations are legacy evidence only. Phase 9 never selects or trusts them for a provider handshake.

## Cross-Language Integration Fixture

`Tests/Integration/prepare-real-adapter.sh ABSOLUTE_PREFIX [user|system]` builds and installs the actual Swift adapter and emits its package-result JSON. A Go-foundation integration test may consume that exact binary and receipt; it must still build and receipt the real Go SSIAG parent, provision the configuration and trust declaration above, start the instance, and require `qxctl ssiag provider verify` to report both mutual-trust booleans true. A shell stand-in or fabricated response is not equivalent evidence.
