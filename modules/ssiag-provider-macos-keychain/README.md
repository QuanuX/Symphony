# SSIAG macOS Keychain Provider

Independent Swift adapter scaffold for the future macOS Apple Keychain boundary of Symphony Secure Identity and Access Governance.

Phase 9 remains intentionally metadata-only: the adapter accepts one bounded, digest-bound control request from an independently inspected SSIAG parent, emits at most one response, and exits. Phase 10B adds a separate no-payload readiness operation, complete app-like bundle receipt ownership, native structural-signature and configured code-requirement evaluation, and bounded security-session evidence. It installs beside other exact versions and is selected only by an explicit protected TOPS binding. No active alias, newest-version discovery, Keychain item/key operation, credential operation, secret channel, or operational Keychain access is enabled.

Start with `INTENT.md`, follow `INSTALL.md`, and administer live verification through `qxctl ssiag provider verify <name>` rather than invoking a metadata bypass.
