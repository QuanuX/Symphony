# SSIAG Provider Readiness Contract

## Purpose

This contract separates three facts that must never collapse into one another:

1. **structural validation** — the installed and privately staged artifact has a valid platform structure and code-signature envelope;
2. **policy match** — the platform has evaluated the exact protected installation policy against that artifact;
3. **operational eligibility** — SSIAG has separately admitted a declared provider operation in the current host and session context.

Passing one layer never implies the next. A receipt, successful process exit, code signature, signer label, Team ID, requirement digest, provisioning-profile file, session flag, or qxctl result is not authority by itself.

## Phase 10B Boundary

Phase 10B implements structural and native signing-policy observation for the macOS Keychain provider. Operational eligibility remains explicitly unevaluated and disabled. `operational_access_enabled`, `provider_operations_enabled`, and `secret_channel_enabled` are false in every adapter, SSIAG, and qxctl result. No Keychain item or key operation is present.

The first future operational target is Apple's data-protection Keychain in an actual user-login context. A file-based system Keychain, if required later, is a separately identified provider and never a fallback. Graphical login, local TTY, remote TTY/SSH, root, supervisor, fast-user-switching, lock, and logout are observations of host capability—not proxies for caller class. Host ownership or an effective SSIAG grant remains the administrative gate.

## Production Bundle Profile

A production adapter is distributed as one complete app-like bundle that is:

- Developer ID signed;
- hardened-runtime enabled;
- securely timestamped;
- notarized before release;
- installed from immutable receipt-v2 evidence;
- launched only after the complete receipt-owned bundle is reconstructed in a private directory.

The exact signing identifier, Team ID, private application-identifier access group, entitlements, provisioning requirements, and native designated requirement are protected build and installation evidence. They are not universal Symphony constants, qxctl flags, environment values, or caller assertions. The first item namespace uses the adapter's private application-identifier access group. Shared access groups require a separately ratified use case.

The minimum bundle contains `Contents/Info.plist` and `Contents/MacOS/symphony-ssiag-provider-macos-keychain`. Receipt v2 may additionally own only:

- `Contents/Resources/ssiag-signing-policy.json`;
- `Contents/embedded.provisionprofile` when the selected entitlements require a profile;
- `Contents/_CodeSignature/CodeResources`.

Unknown files, symlinks, unsafe modes, changed bytes, missing required files, duplicate paths, unsorted ownership records, and receipt/executable disagreement fail closed. The receipt is committed last on install and removed last on uninstall. An interrupted uninstall may self-heal only from the protected receipt and the absence of previously verified receipt-owned files; changed or unreceipted bytes are never removed automatically.

Legacy bare-executable packages remain readable and executable by new foundations for provider-v1 compatibility. They cannot satisfy the production bundle profile. Older foundations safely reject the new multi-file package rather than partially interpreting it. Multiple exact versions may remain installed and are selected only through the existing explicit provider-binding lifecycle.

## Native Signing Policy

`Contents/Resources/ssiag-signing-policy.json`, when present, is an immutable receipt-owned `symphony.ssiag.macos-signing-policy.v1` object. Its `adapter_requirement` is compiled with Apple's `SecRequirementCreateWithString` and evaluated with `SecStaticCodeCheckValidity` against the complete bundle. A digest of the compiled requirement may leave the adapter; the requirement text may not.

Structural validation uses strict complete-bundle code validation with all-architecture and nested-code checks and no caller-supplied requirement. It is reported separately from policy match. The adapter does not request or emit the broad signing-information dictionary, certificates, profile payloads, raw entitlements, requirement text, native errors, or a security-session identifier. A provisioning profile is observed only as a bounded safe regular-file shape; its existence is not described as validation, protection, or policy success.

Development, unsigned, and ad-hoc artifacts remain metadata-only unless a separately named nonproduction profile is ratified and installed. Absence of a signing policy is `not_configured`, not success. Invalid policy is distinct from signature invalidity. Notarization is release evidence and is not inferred from a local structural signature check.

## Process and Administrative Protocol

Provider control v1 is unchanged. Readiness is a separate one-shot adapter operation and schema family so existing provider-v1 request and response bytes remain compatible. The Go foundation first validates the exact declaration and receipt, privately stages every receipt-owned file while rechecking size and digest, and invokes the staged entry point with the fixed `readiness` operation. Output is one strict JSON value bounded by the existing provider-control limit.

The safe request/result surfaces are:

- `provider-readiness-observation-request.v1` — request/correlation identity plus `host_owner` or `granted_permission` administration basis;
- `provider-readiness-observation.v1` — native structural, policy, and security-session observations with all operational flags false;
- `provider-readiness-result.v1` — SSIAG-bound TOPS, provider, exact installation, safe observation, timestamp, and result digest.

SSIAG exposes `POST /v1/provider-readiness/<provider-name>/observations`. qxctl owns `qxcmd:symphony:ssiag.provider.readiness`, calls that route over the authenticated Unix endpoint, validates the entire closed result, and prints only safe metadata. The backend operations are `engop:symphony:ssiag.provider.readiness.observe` and `engop:symphony:ssiag.macos-keychain-provider.readiness.observe`.

qxctl never accepts an executable path, signing policy, requirement text, Team ID, bundle identifier, access group, entitlement, profile, certificate, provider payload, or secret for this command. It does not select an installation, infer a newest version, enable a provider, or convert readiness into authority.

## Failure and Upgrade Semantics

Unknown adapter readiness commands, missing policy, unsupported legacy adapters, invalid or changed bundles, unavailable security sessions, timeouts, and malformed results produce bounded unavailable/not-ready evidence. There is no fallback to a weaker provider or package shape.

Forward upgrade installs a new exact bundle beside the old version, verifies it, and uses the provider-binding plan/apply circuit to select it. Reverse migration binds an older still-installed exact version through the same circuit. If a new foundation is rolled back before a multi-file bundle is unbound, the old foundation rejects that package and the administrator rebinds a compatible exact version; it never guesses or rewrites the receipt.

Phase 10C remains a separate gate. No readiness result authorizes item creation, lookup, update, rotation, retirement, deletion, reconciliation, signing, decryption, assertion, export, or secret delivery.
