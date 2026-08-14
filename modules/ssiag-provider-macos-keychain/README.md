# SSIAG macOS Keychain Provider

Independent Swift adapter scaffold for the future macOS Apple Keychain boundary of Symphony Secure Identity and Access Governance.

Phase 9 is intentionally metadata-only: the adapter accepts one bounded, digest-bound control request from an independently inspected SSIAG parent, emits at most one response, and exits. It installs beside other exact versions through an immutable install-receipt-v2 package and is selected only by an explicit protected TOPS binding. No active alias, newest-version discovery, Security/Keychain operation, credential operation, secret channel, or operational Keychain access is enabled.

Start with `INTENT.md`, follow `INSTALL.md`, and administer live verification through `qxctl ssiag provider verify <name>` rather than invoking a metadata bypass.
