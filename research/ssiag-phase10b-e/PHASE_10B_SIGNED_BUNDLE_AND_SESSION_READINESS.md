# Phase 10B — Signed Bundle and Session Readiness Research

> **Historical research:** Phase 10B later advanced through PR #149. Current
> canonical SSIAG contracts and implementation take precedence; this document
> is retained only as noncanonical design provenance.

## Status and Outcome

Phase 10B is decision-neutral research. It may report safe observations, but
it cannot authenticate a subject, authorize an operation, decide operational
eligibility, mutate a provider binding, or access a Keychain item. All three
provider operational flags remain false.

The research prototype is internal to the Swift support library. It is not
wired into `serve`, qxctl, SSIAG HTTP, the provider v1 schemas, or installation
lifecycle. Its explicit result fields state:

- `metadata_only: true`;
- `authorization_decision_made: false`;
- `operational_eligibility_decided: false`;
- `operational_access_enabled: false`;
- `provider_operations_enabled: false`;
- `secret_channel_enabled: false`.

## Source-Backed Platform Findings

Apple distinguishes the legacy file-based Keychain from the data-protection
Keychain. Apple recommends the data-protection implementation for new work,
but it is available only in a user login context. Its access groups are built
from code-signing entitlements. A command-line tool using restricted access
group entitlements needs an app-like bundle containing an authorized
provisioning profile. These are platform constraints, not Symphony policy.
[TN3137](https://developer.apple.com/documentation/technotes/tn3137-on-mac-keychains)

Apple's distribution guidance requires restricted entitlement claims to be
authorized by a provisioning profile; a nonbundled executable using such a
claim must be placed in an app-like structure. Developer ID distribution also
requires a valid signature, Hardened Runtime, secure timestamp, and
notarization workflow. The shipping code must not carry
`com.apple.security.get-task-allow`.
[Distribution signing](https://developer.apple.com/documentation/xcode/creating-distribution-signed-code-for-the-mac/),
[notarization](https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution)

The current package installs and stages one bare Mach-O. Copying only that file
does not preserve `Contents/Info.plist`, `Contents/embedded.provisionprofile`,
or the bundle code-signature resource envelope. Therefore the existing private
single-file staging routine must not be reused for data-protection Keychain
access.

## Candidate Bundle Contract — Requires Ratification

The names below are structural placeholders, not selected identities:

```text
<version-root>/
└── <RATIFY_PROVIDER_BUNDLE_NAME>.app/
    └── Contents/
        ├── Info.plist
        ├── MacOS/
        │   └── symphony-ssiag-provider-macos-keychain
        ├── embedded.provisionprofile
        └── _CodeSignature/
            └── CodeResources
```

Required properties:

1. `CFBundleIdentifier` is `RATIFY_PROVIDER_BUNDLE_ID`; it is never inferred
   from a path or module name.
2. The designated requirement binds an Architect-approved signing policy,
   signing identifier, Apple Team ID, and distribution class. No QuanuX,
   enterprise, or local Team ID is hardcoded by this research.
3. Restricted entitlements are the minimal exact set. The access group remains
   `RATIFY_KEYCHAIN_ACCESS_GROUP`; no default-group inference is accepted.
4. The distribution profile authorizes those exact entitlements.
5. Hardened Runtime and secure timestamp are required for production.
6. Ad-hoc, unsigned, locally altered, expired-profile, wrong-team,
   wrong-identifier, or `get-task-allow` builds are metadata-only.
7. Receipt v2 owns every regular bundle file, its size, digest, and required
   mode. Bundle symlinks, plug-ins, mutable resources, and unreceipted files are
   prohibited in the first profile.
8. Multiple exact bundle versions coexist. One protected per-TOPS binding
   selects one exact receipt and executable. No alias or newest-version scan
   gains authority.

Apple defines a designated requirement as the criteria used to recognize the
same signed code across versions. The eventual requirement must be evaluated,
not compared as display text.
[Applying Code Requirements](https://developer.apple.com/documentation/security/applying-code-requirements),
[TN3127](https://developer.apple.com/documentation/technotes/tn3127-inside-code-signing-requirements)

## Candidate Bundle-Preserving Launch Contract

The Go foundation remains responsible for exact receipt and byte trust. The
Swift adapter remains responsible for native code-signing and session
observation. Neither side accepts caller assertions as proof.

1. Load the exact protected binding and immutable receipt v2.
2. Reject a missing file, unknown file, symlink, nonregular file, unsafe
   ancestor, mode/owner mismatch, size mismatch, digest mismatch, or unsupported
   bundle layout.
3. Create one private `0700` staging directory for the invocation.
4. Reconstruct the complete receipt-owned bundle using descriptor-relative,
   no-follow opens. Create every destination exclusively, preserve only the
   ratified modes, and fsync every file and directory boundary.
5. Re-read and digest every staged file. The staged manifest must exactly equal
   the protected receipt before launch.
6. Launch only the staged `Contents/MacOS` entry point. Keep the request bound
   to the original package receipt and staged-manifest digest.
7. The child validates its dynamic code object and the staged static bundle
   with Security Code Signing Services. It returns safe identity values and
   digests only—never certificates, provisioning payloads, entitlement values,
   raw requirements, or native error text.
8. The child independently validates the running SSIAG parent under the
   ratified parent designated requirement plus existing receipt/path/digest
   checks.
9. A successful self-report is necessary but not independently sufficient:
   Go must already trust the exact staged bytes, and the operating system must
   validate the signature. All observed values must match the protected policy.
10. Cancellation kills and reaps the process group. Cleanup removes only the
    invocation staging directory after the child exits.

The design intentionally preserves the current check-to-exec defense while
retaining the app-like profile/signature context.

## Candidate Preflight Observation

The preflight may safely observe:

- app-like bundle layout present or absent;
- protected embedded profile present or absent;
- static signature state `valid`, `invalid`, or `unavailable`;
- dynamic self-signature state `valid`, `invalid`, or `unavailable`;
- bounded signing identifier and Team ID;
- SHA-256 digests of the designated requirement and entitlements dictionary;
- whether `SessionGetInfo(callerSecuritySession, ...)` returned a session;
- root, graphics, TTY, and remote attribute bits;
- stable, allowlisted reason codes.

It must not emit the raw security-session ID. Apple states that session IDs are
login-session-scoped and can be queried with `SessionGetInfo`; the attributes
are observations rather than authorization evidence.
[SessionGetInfo](https://developer.apple.com/documentation/security/sessiongetinfo(_:_:_:))

Preflight cannot claim that a remote or graphical session is permitted, that
user interaction will succeed, or that a Keychain is unlocked. Those decisions
belong to the later access-control and operation contracts.

## Test Gate

Automated research tests must prove:

- all three operational flags are explicitly false;
- no authorization or eligibility decision is made;
- a nonbundle executable cannot become bundle evidence;
- a fabricated bundle/profile cannot become signing authority;
- safe output contains no native error, certificate, profile, requirement, or
  entitlement payload;
- Swift support source contains no Keychain item or cryptographic-key operation
  call;
- provider v1 tests remain unchanged and pass.

Release-gated tests requiring real signing infrastructure remain outstanding:

- correct and wrong Team ID/signing identifier;
- correct, absent, expired, and entitlement-mismatched profiles;
- valid Developer ID, ad-hoc, unsigned, tampered, and `get-task-allow` builds;
- byte-identical bundle staging and tamper injection at each copy boundary;
- graphical user, TTY user, SSH/remote user, fast-user-switch, logout, root,
  and supervisor contexts;
- signed parent replacement and adapter replacement;
- notarization and Gatekeeper evidence for the distributed artifact.

## Architect Ratification Slate

- **10B-A:** data-protection Keychain is the intended first implementation;
- **10B-B:** production distribution form is an app-like Developer ID-signed
  provider bundle;
- **10B-C:** organization-specific Team/signing policy is protected
  configuration rather than a Symphony hardcode;
- **10B-D:** exact bundle identifier and access group;
- **10B-E:** exact adapter and foundation designated requirements;
- **10B-F:** entitlement set, profile ownership, notarization, and update
  continuity policy;
- **10B-G:** full-bundle private staging versus a separately proven
  descriptor-based direct-exec alternative;
- **10B-H:** whether remote TTY sessions are noninteractive-only or always
  unavailable for the first profile;
- **10B-I:** signed development-build policy; recommendation is metadata-only
  unless a distinct, explicit development profile is ratified.

Until all applicable items are ratified and tested, the provider remains
disabled and Phase 10C cannot perform an item operation.
