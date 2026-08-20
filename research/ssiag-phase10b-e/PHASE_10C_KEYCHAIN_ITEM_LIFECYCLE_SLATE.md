# Phase 10C — Keychain Item Lifecycle Design Slate

> **Noncanonical research:** This slate is archived design provenance, not an
> authorization or implementation claim. Current canonical SSIAG contracts
> take precedence, and Phase 10C remains unratified unless recorded there.

## Status

Design only. This document contains no selected reverse-domain namespace,
access group, Team ID, provider operation, schema, or enabled behavior. It does
not authorize a Keychain call. Phase 10C begins only after the applicable 10B
bundle, signing, session, and launch gates are ratified and proven.

## Recommended Platform Posture

Use the modern `SecItem` API with the data-protection Keychain, explicitly
selecting that implementation and explicitly disabling synchronization for the
first profile. Apple recommends the data-protection implementation for new
work; it is user-context-only and uses access groups plus optional
`SecAccessControl`.
[TN3137](https://developer.apple.com/documentation/technotes/tn3137-on-mac-keychains)

This recommendation is not yet ratified. The legacy file-based Keychain is an
alternative only if the Architect accepts its different ACL model and reduced
semantic consistency. It must never become an automatic fallback.

## Canonical Identity Before Platform Mapping

Ratify one provider-neutral identity tuple before selecting Apple attribute
strings:

```text
TOPS ID
provider binding ID
credential/reference ID
item purpose
item class
generation
schema version
```

Rules:

- TOPS, binding, reference, and generation identifiers are opaque stable IDs;
- display names, usernames, labels, module paths, and current adapter version
  never define item identity;
- one tuple maps to one exact Apple query;
- every query names the exact class and all ratified identity attributes;
- wildcard, partial, multi-result, search-list, or newest-generation discovery
  is prohibited;
- the reverse-domain prefix remains `RATIFY_ITEM_NAMESPACE` until approved;
- the access group remains `RATIFY_KEYCHAIN_ACCESS_GROUP` until approved;
- adapter upgrades preserve the item identity and schema deliberately; they do
  not rewrite or migrate an item merely because a newer binary is bound.

Apple explains that item attributes distinguish records and that exact
attributes are required to find them again.
[Adding a password to the keychain](https://developer.apple.com/documentation/security/adding-a-password-to-the-keychain),
[Keychain items](https://developer.apple.com/documentation/security/keychain-items)

## Item-Class Slate

| Candidate class | Purpose | First phase | Default disposition |
|---|---|---|---|
| cryptographic private key | non-exportable create/use | 10D | preferred where supported |
| generic password | explicitly exportable small secret | 10E | deferred |
| internet password | server-bound credential | later | not admitted initially |
| certificate | public certificate material | later | metadata/public only |
| identity | certificate plus private key | later | separate review |

No universal item wrapper is recommended. Different Apple classes have
different identity, access-control, synchronization, migration, and result
semantics.

## Operation Families Requiring Separate IDs

1. `metadata-exact`: query one exact reference without returning secret data;
2. `create-generation`: create one absent generation idempotently;
3. `prepare-rotation`: create a new generation while retaining the active one;
4. `activate-generation`: CAS the SSIAG reference from old to new after proof;
5. `retire-generation`: revoke use without immediate destructive deletion;
6. `delete-generation`: explicit destructive removal under a separate grant;
7. `reconcile-operation`: resolve an interrupted or ambiguous attempt;
8. `capability-probe`: test platform capability without enumerating items.

Create, update, rotate, activate, retire, and delete must never collapse into a
generic mutation. Metadata lookup must suppress authentication UI unless the
operation explicitly permits interaction. Apple provides
`kSecUseAuthenticationUI` to choose whether authentication prompting is
allowed.
[Authentication UI](https://developer.apple.com/documentation/security/ksecuseauthenticationui)

## Access-Control Matrix to Ratify

Apple advises choosing the most restrictive accessibility that meets the use
case and supports `SecAccessControl` constraints such as user presence,
current-biometry set, passcode, and private-key use.
[Restricting accessibility](https://developer.apple.com/documentation/security/restricting-keychain-item-accessibility),
[access-control flags](https://developer.apple.com/documentation/security/secaccesscontrolcreateflags)

For each operation, ratify all columns:

| Field | Required decision |
|---|---|
| accessibility | exact `ThisDeviceOnly`/unlock/passcode policy |
| interaction | required, permitted, prohibited, or not applicable |
| user presence | none, user presence, current biometry, or explicit combination |
| authentication context | one-use or bounded reuse duration; recommend one-use initially |
| session | graphical user, noninteractive user, remote TTY, root/system |
| synchronization | false initially; query must never use “any” |
| migration | device restore, same-device backup, and new-host behavior |
| destructive behavior | retain, retire, delete, or provider-native recovery |

System/root and unsupported session contexts report unavailable without
changing Keychain implementation, user, or provider.

## Crash-Safe Mutation Circuit

The Keychain call may commit atomically while the adapter crashes before its
response. Therefore “no response” cannot mean “no mutation.” The Go SSIAG
foundation should own a safe, no-secret operation journal outside the one-shot
adapter:

```text
authorized
  -> prepared
  -> audit_intent_committed
  -> provider_invoked
  -> provider_observed
  -> audit_result_committed
  -> SSIAG_reference_committed
  -> complete
```

The journal contains only IDs, operation kind, generation, expected-state and
configuration digests, timestamps, request/correlation/idempotency IDs, safe
result category, and STAV receipts. It contains no Keychain attributes that
reveal a secret, no native error text, and no secret digest.

Recovery rules:

- one exact idempotency ID identifies one intended item tuple and generation;
- retry first performs an exact metadata-only reconciliation query;
- absent means create may be retried;
- exact matching safe metadata means creation is treated as committed;
- conflicting class/generation/policy metadata fails closed for explicit
  administrator recovery;
- ambiguous secret import is never blindly replayed;
- rotation keeps old and new generations until reference activation and audit
  complete;
- delete is never automatic recovery for a failed create;
- concurrent writers serialize per TOPS/provider/reference;
- uninstall never removes Keychain items.

## STAV Ordering Gap

The current closed SSIAG vocabulary can record provider and rotation outcomes,
but the exact requested/completed ordering for a mutation must be ratified
before implementation. The recommended requirement is:

1. durable safe intent accepted before mutation;
2. Keychain mutation or exact reconciliation;
3. durable safe result accepted before SSIAG exposes the new reference;
4. explicit reconciliation-required state if result audit is unavailable after
   a provider-side commit.

No event may carry item values, assertions, raw queries, access-control blobs,
native errors, signatures, public-key payloads, or secret-derived digests.

## Safe Error Taxonomy

Map native status only inside the adapter to stable categories such as:

- `item_absent`;
- `item_conflict`;
- `locked`;
- `interaction_required`;
- `interaction_cancelled`;
- `authentication_failed`;
- `missing_entitlement`;
- `session_unavailable`;
- `policy_mismatch`;
- `provider_unavailable`;
- `internal_failure`.

Never return the raw integer or `SecCopyErrorMessageString` output. Apple lists
the relevant result families, including duplicate, not-found, cancelled,
interaction-not-allowed, authorization, and entitlement failures.
[Security result codes](https://developer.apple.com/documentation/security/security-framework-result-codes)

## Negative Test Gate

- cross-TOPS and cross-reference collision attempts;
- duplicate item and multiple-result rejection;
- synchronization accidentally true or query “any”;
- default access-group inference;
- locked session, denied/cancelled prompt, interaction prohibited;
- wrong item class or access-control policy;
- crash before and after every journal, audit, provider, and reference boundary;
- replay with same and changed idempotency evidence;
- concurrent create/rotate/delete;
- adapter upgrade, downgrade, rollback, and incompatible item schema;
- fast-user-switch, logout, remote TTY, root/system;
- uninstall with retained items;
- secret-marker scans across args, env, JSON, qxctl, STAV, journal, logs, and
  crash artifacts.

## Architect Ratification Slate

- **10C-A:** data-protection versus file-based Keychain;
- **10C-B:** reverse-domain namespace and exact attribute mapping;
- **10C-C:** access group and update continuity;
- **10C-D:** admitted item classes;
- **10C-E:** exact operation IDs and authorization tuples;
- **10C-F:** accessibility, user-presence, interaction, and session matrix;
- **10C-G:** generation/rotation/retirement/deletion policy;
- **10C-H:** journal owner, format, retention, and recovery authority;
- **10C-I:** STAV intent/result ordering and audit-deferred recovery;
- **10C-J:** item-schema compatibility and explicit migration protocol.
