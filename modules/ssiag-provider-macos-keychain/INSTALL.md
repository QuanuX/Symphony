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

## User Installation

```bash
.build/release/symphony-ssiag-provider-macos-keychain install --scope user
```

This installs:

- binary: `~/.local/bin/symphony-ssiag-provider-macos-keychain`
- manifest: `${XDG_STATE_HOME:-~/.local/state}/symphony/ssiag/providers/macos-keychain/install.json`

## System Installation

Run the same executable under an owner-approved privilege boundary:

```bash
.build/release/symphony-ssiag-provider-macos-keychain install --scope system
```

This installs the binary under `/usr/local/bin` and its manifest under `/var/lib/symphony/ssiag/providers/macos-keychain/`.

## Verification

```bash
symphony-ssiag-provider-macos-keychain version
symphony-ssiag-provider-macos-keychain status
printf '%s\n' '{"schema":"symphony.ssiag.provider.request.v1","request_id":"verify-1","operation":"hello"}' | symphony-ssiag-provider-macos-keychain serve
```

Verification MUST report `operational_access_enabled: false`. The metadata protocol, fail-closed lifecycle, and disabled-operation behavior are implemented and tested. Operational Keychain access remains gated until its exact namespace, operation, code-signing, protected-delivery, lifecycle, and security contracts are implemented and verified.

## Upgrade

Build the reviewed replacement and run `install --scope <scope> --force`. The lifecycle refuses a changed installed digest without explicit `--force`.

## Uninstall

```bash
symphony-ssiag-provider-macos-keychain uninstall --scope user
```

Use `--scope system` for a system installation. A changed installed binary requires `--force`. Uninstall removes only the recorded adapter binary and manifest. It does not delete Keychain items or any TOPS state.
