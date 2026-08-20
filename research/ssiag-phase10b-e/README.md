# SSIAG Phase 10B–10E Research Archive

This directory preserves noncanonical research originally developed on
`agent/ssiag-phase10b-readiness-research`. It is historical design evidence,
not ratified protocol truth, and does not amend `knowledge/ssiag/` or make an
operational claim. Canonical contracts and current implementation always take
precedence over this archive.

Phase 10B subsequently advanced through canonical review and implementation in
PR #149. Its current behavior is defined by `knowledge/ssiag/` and
`modules/ssiag-provider-macos-keychain/`, not by this research snapshot. Phases
10C–10E remain design material unless later ratified and represented in their
own canonical contracts and implementation evidence.

The research is intentionally sequenced:

1. Phase 10B observes signed-bundle and login-session prerequisites without
   accessing Keychain items or enabling a provider operation.
2. Phase 10C designs exact item lifecycle and crash recovery.
3. Phase 10D designs non-exportable key use before secret export.
4. Phase 10E designs import/export delivery last.

No literal bundle identifier, Apple Team ID, access group, reverse-domain item
namespace, or signing requirement is selected here. Tokens beginning with
`RATIFY_` are placeholders and must fail validation if they ever reach a
runtime configuration.

All four documents preserve these invariants:

- the Go SSIAG foundation remains cgo-free;
- the Swift adapter remains out of process;
- provider v1 remains metadata-only and byte/behavior compatible;
- operational, provider-operation, and secret-channel flags remain false;
- system/headless access has no implicit user-Keychain or weaker-provider
  fallback;
- qxctl carries administrative metadata only, never secrets or provider
  payloads;
- installation, compatibility, and readiness never select the newest version;
- no proposal becomes authority without Architect ratification and canonical
  SKV contracts.

## Primary Apple Sources

- [TN3137: On Mac keychains](https://developer.apple.com/documentation/technotes/tn3137-on-mac-keychains)
- [Code Signing Services](https://developer.apple.com/documentation/security/code-signing-services)
- [Creating distribution-signed code for macOS](https://developer.apple.com/documentation/xcode/creating-distribution-signed-code-for-the-mac/)
- [Notarizing macOS software before distribution](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution)
- [Sessions](https://developer.apple.com/documentation/security/sessions)
- [Keychain items](https://developer.apple.com/documentation/security/keychain-items)
- [Restricting keychain item accessibility](https://developer.apple.com/documentation/security/restricting-keychain-item-accessibility)
- [Protecting keys with the Secure Enclave](https://developer.apple.com/documentation/security/protecting-keys-with-the-secure-enclave)
