# Phase 10D — Non-Exportable Key Use Design Slate

> **Noncanonical research:** This slate is archived design provenance, not an
> authorization or implementation claim. Current canonical SSIAG contracts
> take precedence, and Phase 10D remains unratified unless recorded there.

## Status

Design only. This phase does not generate, look up, or use a cryptographic key.
It assumes Phase 10B signing/session trust and the applicable Phase 10C item
identity, access-control, journal, and audit contracts have passed.

## Why Non-Exportable Use Comes First

Apple's Secure Enclave keeps the private key material outside application
memory and performs operations through an opaque key reference. Apple supports
generation of a permanent P-256 private key with
`SecKeyCreateRandomKey`, `kSecAttrTokenIDSecureEnclave`, and an access-control
object containing `privateKeyUsage`.
[Protecting keys with the Secure Enclave](https://developer.apple.com/documentation/security/protecting-keys-with-the-secure-enclave)

That architecture aligns with SSIAG's preference for provider-internal
operations. It avoids a provider-to-foundation secret channel for the private
key. It does not eliminate the need to protect request payloads, signatures,
public keys, authorization, prompts, audit metadata, or availability.

Secure Enclave support is a distinct capability, never an assumed property of
every macOS host. Apple limits these items to generated 256-bit elliptic-curve
private keys; existing keys cannot be imported into the Secure Enclave.
[Secure Enclave token](https://developer.apple.com/documentation/security/ksecattrtokenidsecureenclave)

## Capability Slate

Advertise capabilities separately and only after a real platform probe:

| Candidate capability | Meaning | Fallback |
|---|---|---|
| hardware P-256 generate | provider generates a permanent private key in Secure Enclave | none |
| hardware P-256 sign digest | provider signs one bounded digest using that key | none |
| public-key export | provider returns the public key only | explicit safe result |
| software Keychain key generate/use | nonhardware, still non-exportable by policy | separate capability and review |
| decrypt/key-agreement | provider-internal use | later algorithm-specific review |

Unsupported hardware returns capability absent/unavailable. It must not select
a software key, exportable key, different algorithm, or different provider
unless the protected configuration explicitly requested that separate
capability.

## Candidate Operation Separation

Each operation requires its own stable provider-operation ID and SSIAG
authorization tuple:

1. generate one exact absent key generation;
2. fetch safe public-key metadata;
3. return the bounded public key;
4. sign one exact domain-bound digest;
5. prepare rotated generation;
6. activate rotated generation;
7. retire old generation;
8. explicitly delete one retired generation;
9. reconcile an interrupted generation.

“Use key,” “crypto,” and “execute provider” are too broad for permission
targets. Generate permission must not imply sign, export-public, rotate, or
delete permission.

## Key Creation Candidate

The first candidate profile is:

- exact opaque application tag derived from the ratified Phase 10C identity;
- P-256 only;
- permanent private key;
- explicit data-protection Keychain and explicit non-synchronizing policy;
- `ThisDeviceOnly` accessibility chosen by the Phase 10C matrix;
- `privateKeyUsage` plus the ratified user-presence constraint;
- exact access group;
- private key non-exportable;
- no automatic deletion when creation response is lost;
- public key returned only through an explicitly safe bounded result.

The actual accessibility and presence flags remain a ratification decision.
Apple recommends the most restrictive setting compatible with the operation
and permits combining `privateKeyUsage` with presence constraints.
[Access-control flags](https://developer.apple.com/documentation/security/secaccesscontrolcreateflags)

## Signing Candidate

Do not send arbitrary documents through the metadata control envelope. The
smallest candidate signs exactly one fixed-size digest under an exact algorithm
and explicit purpose/domain ID. Required bindings include:

- request, correlation, TOPS, provider, reference, generation, and capability;
- algorithm and digest length;
- purpose/domain ID;
- fresh SSIAG authorization and policy/configuration digests;
- deadline and no automatic retry;
- one-use local authentication context unless another reuse window is
  explicitly ratified.

The output signature and public key are not secret, but they are provider
payloads. They must be bounded, schema-validated, excluded from STAV and normal
logs, and returned only to the authenticated operation consumer. qxctl may
administer capability and report safe status but should not become a general
signing-data pipe.

Apple provides `SecKeyCreateSignature` for an exact key, algorithm, and input.
The implementation must first verify that the algorithm is supported and must
not switch algorithms silently.
[SecKeyCreateSignature](https://developer.apple.com/documentation/security/seckeycreatesignature(_:_:_:_:))

## Interaction and Replay Rules

- The caller declares whether interaction is permitted; provider policy may
  require it but cannot weaken it.
- An operation requiring presence fails if the session cannot display or
  perform the ratified authentication.
- User cancellation is final for that request and is not retried.
- Timeout invalidates the request even if a late prompt completes.
- No prompt is issued during metadata lookup or reconciliation unless the
  operation contract explicitly requires it.
- An authentication context is bound to one request initially. Apple permits a
  previously authenticated `LAContext` to be reused, so reuse must be disabled
  or bounded deliberately rather than left implicit.
[Authentication context](https://developer.apple.com/documentation/security/ksecuseauthenticationcontext)
- Per-subject, per-provider, and per-item concurrency limits prevent prompt
  storms. One key mutation is serialized; signing may use a separately
  ratified bounded concurrency limit.

## Crash and Audit Semantics

Generation is mutating and uses the complete Phase 10C journal/audit circuit.
A crash after key generation reconciles by exact tag/generation metadata and
never creates another generation implicitly.

Signing is normally nonmutating at the item level but is security-relevant:

- require the committed safe STAV precondition chosen by the Architect;
- never record digest-to-sign, signature, public key bytes, algorithm payload,
  native prompt details, or Keychain attributes;
- record only safe references, operation identity, outcome category, and
  required configuration digests;
- do not retry an ambiguous completed signature automatically;
- if no safe result reaches the consumer, return an explicit indeterminate
  outcome rather than claiming failure or success without evidence.

## Threat and Test Gate

- capability absent on unsupported hardware;
- no hardware-to-software fallback;
- duplicate application tag and cross-TOPS query;
- generated private key cannot be externally represented;
- wrong algorithm, digest length, purpose, audience, reference, or generation;
- denied/cancelled/expired interaction and reused authentication context;
- concurrent generation, rotation, signing, and deletion;
- crash after hardware generation and after signature creation;
- changed access control, enrolled-biometry change, device restore, and host
  migration;
- adapter/foundation version swap and signing-policy mismatch;
- stdout/stderr/args/env/qxctl/STAV/journal/crash artifact marker scans;
- public result bounds and malformed signature rejection by the consumer.

## Architect Ratification Slate

- **10D-A:** Secure Enclave P-256 generation/signing is the first operational
  capability;
- **10D-B:** whether a separately named software-Keychain capability is in the
  same release or deferred;
- **10D-C:** exact key attributes, accessibility, presence, and access group;
- **10D-D:** exact algorithm and raw-data versus fixed-digest input posture;
- **10D-E:** purpose/domain binding and consumer result schema;
- **10D-F:** signature/public-key delivery surface and qxctl exclusion;
- **10D-G:** prompt, authentication-context reuse, timeout, and concurrency
  rules;
- **10D-H:** STAV precondition/result ordering for nonmutating key use;
- **10D-I:** ambiguous-result and retry semantics;
- **10D-J:** device migration, backup, rotation, and destruction policy.
